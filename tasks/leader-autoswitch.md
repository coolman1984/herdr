# Task: automatic leader switch - a background watcher, no key press

Follow-up to `tasks/leader-switch.md` and `tasks/leader-switch-per-project.md`. Since your last run the
lead committed (read `git log -5` and HISTORY Phases 61-62 first): `start` falls back to `pane run` +
`agent rename` on `agent_pane_busy`; `Get-ClaudeSignIn` / `Get-FreeLeaders` skip chain items signed in
as the current leader's email. Same rules, style, test and report format as the first spec. No git
commands; the lead commits. Do not start real agents or touch the owner's live Herdr session.

## Why
The owner approved (2026-10-31): when the lead hits its limit, the switch must happen by itself.
Today the owner must notice and press `Ctrl+B` `Shift+C`.

## Design (decided by the lead)
1. **Command** `herdr-agents autoswitch [-IntervalSec 60] [-Once]` (add to `ValidateSet`, header comment).
   One instance per Windows user: named mutex `Local\herdr-autoswitch` (copy `herdr-hud.ps1`'s pattern);
   a second start prints `autoswitch already running` and exits 0.
2. **Each tick** (every `-IntervalSec`, default 60; `-Once` = one tick, for tests and checks):
   for every project leader file in `%APPDATA%\herdr\leaders\*.json`:
   - Skip when its pane is not a live agent on this server (`Get-LiveAgents`, same name + pane).
   - Skip unless the agent status is not `working` (a limit message shows on an idle/done/blocked agent;
     this avoids matching text scrolling past while it works).
   - Read `agent read <name> --source recent`; `Get-LimitFromScreen` (unchanged). No limit -> next project.
   - Limit seen -> run the switch for that project in-process (reuse `Switch-Leader`; set its inputs
     the way the key does: focused pane = the old leader's pane, `-Cwd` = the project). Never shell out.
3. **Notify** with `herdr notification show "<title>" --body "<text>" --sound request` (Herdr turns it into
   a Windows notification when the owner looks elsewhere):
   - success: title `Lead switched`, body `<old> hit its limit (until <time>) - now <new> in <project>`.
   - BLOCKED (exit 3, e.g. a sign-in screen): title `Lead switch needs you`, body the new leader's name
     and `answer its screen`.
   - failure (e.g. every leader limited): title `Lead switch failed`, body the error line.
4. **No loops, no spam.** Keep `%APPDATA%\herdr\autoswitch-state.json`: per project the last outcome and
   `retry_after`. After a failure or BLOCKED, do not try that project again until `retry_after`
   (failure: the earliest free time from the chain, else +30 min; BLOCKED: +10 min) and notify once
   per outcome change. After success the leader file names the new leader, so the old screen is not
   read again. Atomic writes (reuse `Save-LeaderJson`).
5. **Log** one line per action to `%APPDATA%\herdr\autoswitch.log` (time, project, outcome); keep it
   under 200 KB (drop the oldest half when over, as other logs here do - check `herdr-autosync.ps1`).
6. **Lifetime.** `start -Lead` and a successful `switch` make sure the watcher runs: if the mutex is
   free, start `conhost.exe --headless pwsh -NoProfile -File "<this script>" autoswitch` detached
   (CLAUDE.md: nothing may flash a console). The watcher exits by itself after 30 minutes with no live
   leader in any project (the PC is short on RAM). Errors inside one tick are logged and the loop goes
   on; an error that repeats 5 ticks in a row stops the watcher with a `Lead watcher stopped` notification.
   No broad silent catch: every caught error is logged with its message.
7. **Status:** `herdr-agents leaders` prints one extra line: `Watcher: running` or
   `Watcher: not running (starts with start -Lead or switch)`.

## Tests (extend `tests/test-leader-switch.ps1`, offline, fakes as the existing fixtures)
- A tick with an idle leader showing a limit calls the switch once and notifies `Lead switched`.
- A `working` leader with limit text: no switch. A leader with no limit text: no switch.
- A dead leader pane: skipped. Two projects: only the limited one switches.
- Failure: notified once; the next tick before `retry_after` does nothing; after it, it retries.
- BLOCKED: `Lead switch needs you` once, retry after 10 min.
- The second instance exits 0 with `already running` (mutex held by the test).
- `start -Lead` starts the watcher only when the mutex is free (fake the process start; assert the
  command line uses `conhost.exe --headless`).
- Log trimming keeps the newest lines under the limit.
Report how many new tests fail on HEAD. Run all tests in `tests/` and `pwsh -File check.ps1`.
Update HISTORY (Phase 63), docs/GUIDE.md 13.20, docs/FILES.md (the new state and log files),
README.md (one line), skills/herdr-team/SKILL.md (one line). LESSONS only for a durable new fact.
