#!/usr/bin/env bash
# The keeper restarts a killed runner by itself: 5-task goal, `herdr-goal keep`, runner killed -9 three times.
#   app/tests/goal-e2e/keep.sh WORKDIR
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"; app="$(cd "$here/../.." && pwd)"
work="$(mkdir -p "$1" && cd "$1" && pwd)"; bin="$work/bin"; proj="$work/project"; mkdir -p "$bin"
(cd "$app" && go build -o "$bin/herdr-goal" ./cmd/herdr-goal && go build -o "$bin/goalagent" ./tests/goalagent) || exit 1
export PATH="$bin:$PATH"
rm -rf "$proj"; mkdir -p "$proj"; cd "$proj"
git init -q -b main; git config user.name o; git config user.email o@x
printf '# wordstat\n\nTODO\n' > README.md; git add -A; git commit -qm init
herdr-goal new --research "$here/research.json" --project "$proj" --id keep --profiles "$here/profiles.json" --plan "$here/plan.json" --goal "wordstat" \
  --test "[ ! -d tests ] || python3 -m unittest discover -s tests -q" \
  --accept "python3 -m unittest discover -s tests -q" --accept "grep -q '## Usage' README.md && grep -q '## Install' README.md" >/dev/null || exit 1
L="$proj/.herdr-goal/keep"
herdr-goal keep "$L" --every 2s --interval 300ms >> "$work/keep.log" 2>&1 & kp=$!
for i in 1 2 3; do
  sleep 8
  pid=$(python3 -c "import json;print(json.load(open('$L/runner.lock'))['pid'])" 2>/dev/null) && kill -9 "$pid" 2>/dev/null && echo "killed runner $pid" >> "$work/keep.log"
done
wait $kp; code=$?
cat "$work/keep.log"; echo "keep exit $code"
for d in "$L"/proc/*/; do [ -f "$d/exit" ] || echo "ALIVE $d"; done
exit $code
