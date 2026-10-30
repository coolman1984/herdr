# Goal engine: how it works and how to use it

Spec: [SPEC-GOAL-ENGINE.md](SPEC-GOAL-ENGINE.md). Code: `app/internal/goal` (Go standard library only),
command `app/cmd/herdr-goal`. Architect skill: `skills/herdr-architect`. History: HISTORY Phase 57.

## Use

```
cd app && go build -o herdr-goal ./cmd/herdr-goal        # Windows: herdr-goal.exe
herdr-goal new --project I:\HERDR-trials\x --id x1 --goal "..." \
   --accept "go test ./..." --test "go test ./..." --profiles goal-profiles.json [--plan plan.json]
herdr-goal run I:\HERDR-trials\x\.herdr-goal\x1 --backend herdr     # agents in Herdr panes
herdr-goal status <ledger>      herdr-goal events <ledger>
herdr-goal keep <ledger> [--every 15s] [run flags]   # restarts a dead runner until the goal ends
herdr-goal pause|resume|stop <ledger>
herdr-goal env [--project DIR]                       # capability report
```

- `--accept`: the goal's acceptance, the gate to DONE, run by the Supervisor itself.
- `--test`: the project's full tests, run on the goal branch after every merge; a merge that breaks
  them is undone at once.
- `--profiles`: agent profiles (command with `{prompt}`/`{model}`/`{account}`, kind = vendor, roles
  write/review/lead, accounts in rotation order, `fallback`), the `lead` profile, a `decider` command
  for routine menus, and `limits`. Example: `app/tests/goal-e2e/profiles.json`.
- `--plan`: tasks given up front; without it the lead (Architect) plans with `herdr-goal task add`.
- `run` is safe to start again at any time: a live runner holds `runner.lock` (pid + heartbeat), a
  second one exits; a killed runner's lock is taken over and its ledger reconciled. `herdr-goal keep`
  starts a detached runner whenever none holds the goal (`app/tests/goal-e2e/keep.sh`: runner killed -9
  three times, restarted three times, goal DONE). On Windows a scheduled task that starts `keep` at logon
  would make the pair self-healing (not set up: owner's call, it changes the PC).

## Before planning (owner's policy, docs/COMPANY-ENVIRONMENT-POLICY.md)

- `herdr-goal env [--project DIR]`: the capability report (host, container, memory/CPU after cgroup limits,
  disk, tools with versions, PowerShell first, Python modules found without importing, Office ProgIDs,
  proxy/DNS, the per-device `device-policy.json` profile). `new` saves it as `capabilities.json` and lowers
  the agent count to what the machine can carry.
- Quotas: each profile has a `provider`; limits are read per real account (Claude status-line files,
  Codex session logs; Antigravity/Cline `unsupported`). No new work at >= 90% of any limit; a running agent
  is moved off; unknown/stale = one agent at a time on that account. Profiles on one subscription share it.
- Skills: `skills/catalog.json` (`HERDR_SKILLS_CATALOG`); tasks carry `--tags`; only approved skills that
  the machine can run reach the brief (max 3), recorded per task.
- Commands carry a token: the owner key `keys/owner` (0600) or the lead's own token; delegates have none.

## Parts

| Part | File | What it does |
|---|---|---|
| Ledger | `ledger.go` | `goal.json` = the whole state (goal, tasks, agents, accounts, messages, intents, escalations, locks, record), written atomically (temp + fsync + rename); `events.jsonl` append-only + fsync, a torn last line is skipped; `plan.md` readable view. |
| Inbox | `commands.go` | The lead, the owner and the CLI change a running goal only by appending to `inbox.jsonl`; the runner applies each line once, in order, and stores the offset in the same atomic write. Lead may not pause/stop/resume or answer the owner's escalations; delegates may only `ack`/`status`. |
| Intents | `engine.go` | Every outward action (start, send, keys, stop) is written as an intent before it runs and removed after. After a crash `Recover` reconciles: a start that happened is kept (no double agent), one that did not is done now; a send/keys is not repeated blindly (the screen decides again); an interrupted merge is aborted. |
| Messages | `tasks.go` | Each agent gets a numbered brief file (`briefs/<agent>.md`, message `m<n>`). It must create `acks/m<n>` first (receipt) and write `answers/<agent>.json` last (`{"msg","status","verdict","summary","notes"}`, checked against the message id). Idle is never done. |
| Watchdog | `watchdog.go`, `screen.go`, `prompts.json` | Per agent per tick: backend error (never read as dead) / result file / dead / screen prompts (secret, trust, popup, queued, choice, capacity, model error, rate limit, context low) / launch check / idle without result / stall. Each problem has a retry cap, then restart (work committed first), then another model, then the task attempt fails. |
| Scheduler | `tasks.go` | Dependencies, file reservations, `MaxAgents` (default 2), free RAM and disk gates before every start; writer chosen by the goal's record; reviewer always another vendor; task checks -> review -> merge -> project tests. Last writer attempt goes to another vendor. |
| Git | `git.go` | Integration worktree `wt/_main` on `goal/<id>`; one worktree per task on `goal/<id>-<task>`; `--no-ff` merges are the checkpoints; a failed merge or failed tests reset to the pre-merge head; `revert` undoes one task's merge with a new commit and keeps the later ones. `.herdr-goal/` is kept out of git through `.git/info/exclude`. |
| Backends | `backend.go` | `herdr`: one pane per agent split right of the lead (`pane split` + `pane run`, LESSONS #99), screens by `pane read --source visible`, pane ids in `panes.json` so a new runner finds them. `proc`: each agent under a detached `herdr-goal relay` (inbox -> stdin, output -> `screen.log`, heartbeat); the screen is what the agent printed since its last input, like a terminal redraw. |

Goal states: `planning -> running -> verifying -> done`; also `paused`, `waiting-limit` (all accounts
limited: resumes by itself at the earliest reset), `needs-owner` (only secrets/sign-ins, trust outside
the project, or failed tasks with no lead), `stopped` (no progress for `no_progress`, or the owner),
`failed` (time limit; acceptance failed with no lead).

## What stops at the owner, and nothing else

A secret or sign-in request (checked again right before every send, whatever asked for it: nothing is
typed into such a pane), a trust prompt for a folder outside the project, and tasks that failed with
no lead to re-plan. Account limits never reach the owner: next account, next profile, then wait.

## Proof

Hardening after the independent audit: `docs/audits/goal-engine-response-20261030.md` (status of all 18
findings), HISTORY Phases 58-62. CI (`.github/workflows/independent-goal-audit.yml`) runs every push on Linux
and Windows with `-race`.

### First proof (2026-10-30, Linux cloud container, see HISTORY Phase 57)

- `go test ./internal/goal` 35 tests; `python3 app/tests/goal-mutations.py`: 24/24 rule mutations make
  their test fail.
- `app/tests/goal-e2e/run.sh`: real 5-task goal (a Python word-count tool), 2 agent kinds, faults
  injected (trust prompt with "No" highlighted, crash, stop without result, account limit, model at
  capacity, review rejection, README merge conflict), runner killed -9 and lead agent killed -9
  mid-run, second runner refused: goal DONE, acceptance green when re-run by the script itself.
- `app/tests/goal-e2e/soak.sh`: goals back to back with chaos (runner kill -9 every 40-90 s, random
  agent kill -9 every 20-50 s), last goal plants a secret request. Results in HISTORY Phase 57.

## Not proven yet (doubts)

- The agents in these runs are scripted stand-ins (`app/tests/goalagent`), not Claude/Codex/Cline:
  the cloud container has no agent CLIs and no Herdr (building Herdr needs Zig 0.16, not installable
  there). The prompt wordings in `prompts.json` come from LESSONS and earlier fixtures, not from new
  live captures.
- The Herdr backend was tested only against a fake `herdr` script that records the arguments
  (`backend_test.go`). The argument shapes follow `herdr-agents.ps1`; a lab run on the owner's PC must
  confirm them (`--backend herdr`, isolated session, CLAUDE.md section 3).
- The Windows build compiles (`GOOS=windows go vet`) but was not run on Windows; `check.ps1` and the
  PowerShell suites were not run (no `pwsh` in the container).
- Claude's exact trust-menu keys: the engine sends the number of the "Yes" option and then reads the
  screen back; if a CLI needs a different key, the answer repeats at most `retry_cap` times and then
  escalates to the lead.
- Context-low handoff of the lead and the `decider` command are unit-level only.
