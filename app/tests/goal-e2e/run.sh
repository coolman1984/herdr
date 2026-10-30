#!/usr/bin/env bash
# Goal engine end-to-end run with scripted agents (no model, no network). Linux/macOS.
#   app/tests/goal-e2e/run.sh WORKDIR
# Builds herdr-goal + goalagent, makes a fresh git project, creates a 5-task goal, runs it, kills the
# runner (SIGKILL) and the lead agent once mid-run, restarts the runner, then checks the result itself.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
app="$(cd "$here/../.." && pwd)"
work="$(mkdir -p "$1" && cd "$1" && pwd)"
bin="$work/bin"; proj="$work/project"
mkdir -p "$bin"
(cd "$app" && go build -o "$bin/herdr-goal" ./cmd/herdr-goal && go build -o "$bin/goalagent" ./tests/goalagent)
export PATH="$bin:$PATH"
rm -rf "$proj"; mkdir -p "$proj"; cd "$proj"
git init -q -b main; git config user.name owner; git config user.email owner@x
printf '# wordstat\n\nTODO\n' > README.md; git add -A; git commit -qm init
herdr-goal new --project "$proj" --id wordstat --profiles "$here/profiles.json" --plan "$here/plan.json" \
  --goal "A tiny word-count tool: package, CLI, tests, README with usage and install." \
  --test "[ ! -d tests ] || python3 -m unittest discover -s tests -q" \
  --accept "python3 -m unittest discover -s tests -q" --accept "grep -q '## Usage' README.md && grep -q '## Install' README.md"
L="$proj/.herdr-goal/wordstat"
herdr-goal run "$L" --interval 300ms >> "$work/runner.log" 2>&1 & pid=$!; echo "runner $pid"
sleep "${KILL_RUNNER_AFTER:-12}"
echo "== kill -9 runner $pid at $(date +%T)" | tee -a "$work/runner.log"
kill -9 "$pid" || true
sleep 2
lead=$(ls "$L/proc" | grep '^lead' | head -1 || true)
if [ -n "$lead" ]; then
  echo "== kill -9 lead agent $lead at $(date +%T)" | tee -a "$work/runner.log"
  kill -9 -"$(cat "$L/proc/$lead/pid")" 2>/dev/null || kill -9 "$(cat "$L/proc/$lead/pid")" || true
fi
sleep 3
echo "== a second runner while one is alive must be refused:" | tee -a "$work/runner.log"
herdr-goal run "$L" --interval 300ms >> "$work/runner.log" 2>&1 & pid=$!; sleep 1
if herdr-goal run "$L" --once >> "$work/runner.log" 2>&1; then echo "SECOND RUNNER WAS ALLOWED" | tee -a "$work/runner.log"; fi
code=0; wait "$pid" || code=$?
echo "== runner exit $code at $(date +%T)" | tee -a "$work/runner.log"
herdr-goal status "$L" > "$work/status.txt" || true
herdr-goal events "$L" > "$work/events.txt" || true
echo "== independent acceptance by the test script on goal/wordstat:"
git -C "$proj" worktree list
cd "$L/wt/_main" && python3 -m unittest discover -s tests -q && grep -q '## Usage' README.md && grep -q '## Install' README.md && echo ACCEPTANCE-GREEN
git -C "$proj" log --oneline --graph goal/wordstat | head -40
leftover=$(for d in "$L"/proc/*/; do [ -f "$d/exit" ] || echo "$d"; done)
echo "== agents still alive: ${leftover:-none}"
exit $code
