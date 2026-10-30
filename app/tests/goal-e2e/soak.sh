#!/usr/bin/env bash
# Goal engine soak: goals one after another for MINUTES, with chaos: the runner is killed -9 every
# 40-90 s and restarted by a keeper loop (like a scheduled task re-running `herdr-goal run`), and a random
# agent is killed -9 every 20-50 s. The last goal plants a secret request: it must end needs-owner.
#   app/tests/goal-e2e/soak.sh WORKDIR MINUTES
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"; app="$(cd "$here/../.." && pwd)"
work="$(mkdir -p "$1" && cd "$1" && pwd)"; minutes="${2:-30}"
bin="$work/bin"; mkdir -p "$bin"
(cd "$app" && go build -o "$bin/herdr-goal" ./cmd/herdr-goal && go build -o "$bin/goalagent" ./tests/goalagent) || exit 1
export PATH="$bin:$PATH"
end=$(( $(date +%s) + minutes * 60 )); round=0
summary="$work/summary.txt"; : > "$summary"
while :; do
  round=$((round + 1)); secret=""
  [ $(date +%s) -ge $end ] && secret=secret
  proj="$work/p$round"; rm -rf "$proj"; mkdir -p "$proj"
  (cd "$proj" && git init -q -b main && git config user.name o && git config user.email o@x && echo seed > NOTES.md && git add -A && git commit -qm init)
  python3 "$here/genplan.py" 8 "$round" $secret > "$work/plan$round.json"
  herdr-goal new --project "$proj" --id soak$round --profiles "$here/profiles.json" --plan "$work/plan$round.json" \
    --goal "Soak round $round: eight notes with tests." --test "[ ! -d tests ] || python3 -m unittest discover -s tests -q" \
    --accept "python3 -m unittest discover -s tests -q" --accept "for i in 1 2 3 4 5 6 7 8; do test -f notes/T\$i.txt || exit 1; done" > /dev/null || exit 1
  L="$proj/.herdr-goal/soak$round"; log="$work/runner$round.log"
  t0=$(date +%s); kills=0; akills=0; next_rk=$((t0 + 40 + RANDOM % 50)); next_ak=$((t0 + 20 + RANDOM % 30))
  while :; do
    state=$(python3 -c "import json;print(json.load(open('$L/goal.json'))['state'])")
    case "$state" in done|failed|stopped|needs-owner) break;; esac
    now=$(date +%s)
    [ $((now - t0)) -gt 900 ] && { echo "round $round TIMEOUT" >> "$summary"; break; }
    pid=$(python3 -c "import json;print(json.load(open('$L/runner.lock'))['pid'])" 2>/dev/null || echo 0)
    if [ "$pid" = 0 ] || ! kill -0 "$pid" 2>/dev/null; then   # keeper: start a runner when none is alive
      herdr-goal run "$L" --interval 300ms >> "$log" 2>&1 &
      sleep 1
    fi
    if [ $now -ge $next_rk ]; then
      pid=$(python3 -c "import json;print(json.load(open('$L/runner.lock'))['pid'])" 2>/dev/null || echo 0)
      [ "$pid" != 0 ] && kill -9 "$pid" 2>/dev/null && kills=$((kills + 1)) && echo "== chaos: kill -9 runner $pid" >> "$log"
      next_rk=$((now + 40 + RANDOM % 50)); sleep $((3 + RANDOM % 10))
    fi
    if [ $now -ge $next_ak ]; then
      victim=$(for d in "$L"/proc/*/; do [ -f "$d/exit" ] || basename "$d"; done 2>/dev/null | shuf -n1)
      if [ -n "$victim" ]; then kill -9 -- -"$(cat "$L/proc/$victim/pid")" 2>/dev/null && akills=$((akills + 1)) && echo "== chaos: kill -9 agent $victim" >> "$log"; fi
      next_ak=$((now + 20 + RANDOM % 30))
    fi
    sleep 1
  done
  sleep 2
  state=$(python3 -c "import json;g=json.load(open('$L/goal.json'));print(g['state'],'|',g['reason'])")
  ok="-"
  [ -d "$L/wt/_main" ] && (cd "$L/wt/_main" && python3 -m unittest discover -s tests -q >/dev/null 2>&1) && ok=tests-green
  alive=$(for d in "$L"/proc/*/; do [ -f "$d/exit" ] || echo x; done 2>/dev/null | wc -l)
  echo "round $round: $state; $(( $(date +%s) - t0 ))s; runner kills $kills, agent kills $akills; own check $ok; agents left $alive" | tee -a "$summary"
  for d in "$L"/proc/*/; do [ -f "$d/exit" ] || kill -9 -- -"$(cat "$d/pid")" 2>/dev/null; done
  [ -n "$secret" ] && break
done
echo "== soak finished $(date +%T)" | tee -a "$summary"
