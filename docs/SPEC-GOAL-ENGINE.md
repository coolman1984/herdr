# Spec: Herdr as one product - give a goal, the Architect finishes it

Status: draft for owner review, 2026-10-30. M1 + the M2 engine are built (`app/internal/goal`,
`herdr-goal`; how it works and what is proven: [GOAL-ENGINE.md](GOAL-ENGINE.md), HISTORY Phase 57).

Changes from this draft made while building (2026-10-30):
- One state file: `goal.json` holds goal, tasks, agents, messages and intents, so one atomic write keeps
  them consistent; separate `goal.json` + `tasks.json` could be torn apart by a kill between two writes.
- Two check lists: `accept` (the DONE gate) and `tests` (the project's full tests after every merge).
  Running the goal's acceptance after each partial merge rejects every correct partial merge (LESSONS #105).
- No cost limit: no source reports cost per agent; limits are time, tasks, attempts, agents and RAM/disk.
- A goal folder that is not a git repository with one commit is refused with a message (no `git init`
  in someone's folder by the engine).
- The lead changes the plan only through `herdr-goal` commands (an inbox the runner applies once each).

## Objective

The owner gives one goal and a project folder. An Architect (Claude Opus 5.5) understands the goal,
writes the big plan, splits it into tasks, writes each agent's spec, starts agents, reviews their work,
sends fixes, gives the next task, and continues until the goal is proven done. It can run for hours
with no human prompts. Herdr, Deck and the scripts stop being loose parts and become one installed
program.

Owner's words (2026-10-30, Arabic, summary): "a real, respectable program, files not sticking out; the
human leaves the loop completely; full automation for any project."

## What already exists (reuse, do not rewrite)

| Need | Already have |
|---|---|
| Panes, agents, screens, state | Herdr server + `herdr` CLI |
| Window, cards, resize, themes | Deck (`app/`, Go + embedded web, Edge `--app` window) |
| Start, ask, read, stop, team, usage, handoff | `herdr-agents.ps1`, `agents.json` |
| How to lead a team | `skills/herdr-team`, `docs/TEAM-PLAYBOOK.md`, `tasks/TEMPLATE.md` |
| Which agent for which job | `docs/AGENT-ROSTER.md` |
| Blocked-agent guards, jobs, Inbox | `app/internal/deck` (`guards.go`, `jobs.go`) |

Missing: the loop that keeps going without a human. Today that loop is the owner typing "continue" or
"check". That is the whole gap, plus packaging.

## Design: three parts

1. **Architect** = a real Claude Code session in a pane, with a new `herdr-architect` skill. All the
   thinking lives here; no planning logic in Go. It writes specs and starts agents with the existing
   tools.
2. **Goal ledger** = files in `<project>\.herdr-goal\<goal-id>\`: `goal.json` (goal, limits, state),
   `plan.md`, `tasks.json` (each task: spec, agent, branch, state, proof), `events.jsonl` (append-only).
   Everything needed to resume is on disk, so a new Architect can continue after a crash or a limit.
3. **Supervisor** = a small Go engine inside Deck (`app/internal/goal`). It does not think. It only
   keeps the loop alive:
   - Architect idle and goal not done: send "read the ledger, take the next step".
   - Architect at 95% of its 5-hour limit: make it hand off, continue on the next account.
   - Architect dead: restart it from the ledger.
   - No progress (no new ledger event, commit or finished task) for N minutes: stop and report.
   - Limits reached (time, tasks, cost): stop cleanly and report.
   - All tasks done and the final gate passes: state DONE, notify the owner.

Goal states: `planning -> running -> verifying -> done`; also `paused`, `needs-owner`, `failed`.

**Done needs proof.** A goal ends only when its acceptance commands run green, run by the Supervisor
itself and not just claimed by the Architect, and a reviewer on a different model has passed the work.

## Agent Watchdog: Opus replaces the human at every agent

Owner's decision (2026-10-30): "Opus 5.5 is replacing the human all the time, no need for a human until
the goal is met." Every agent launched is watched from its first second to its end. The Watchdog is a
Go part of the Supervisor, and it runs for every agent the Architect starts:

1. **Launch check.** After each start: confirm the CLI came up, see any startup prompt, confirm the agent
   really began its task (status working, or a new line of output). Not begun in time: retry, then
   report to the Architect. "Sent" never counts as "started".
2. **Sense.** Each agent has one live state: `starting`, `working`, `done`, `stopped-early`, `choice`,
   `trust`, `blocked`, `error`, `rate-limited`, `stalled`, `dead`. Sources, in order: Herdr
   `agent_status`, then the screen read back, then the agent's answer file and output growth.
3. **Answer.**
   - `trust` for a folder inside the goal's project: answer "Yes, I trust this folder", then confirm the
     agent starts. Owner-authorized. Outside the project: `needs-owner`.
   - `choice` (a menu, a question, a plan to approve): the Architect picks and the Watchdog sends it.
   - `stopped-early` or `stalled` with queued text: send Enter, then a "continue" nudge, then escalate.
   - `error` (model at capacity, model not found, tool crash): retry, then a different model per the roster.
   - `rate-limited`: wait, or restart on another account.
   - `dead`: restart from its spec and branch, keeping its work.
4. **Escalate.** Each problem has a retry cap. After it, the Architect gets the screen and decides:
   re-spec, change agent, or split the task.
5. **Record.** Every state change and answer goes to `events.jsonl` with the screen that caused it.

Known prompts live in one data file (`prompts.json`: pattern, kind, answer), so a new CLI prompt is one new
line. An unknown prompt goes to the Architect, which answers it and records the pattern.

## Human out of the loop: what still stops the run

The run never asks about normal work, and Opus answers trust and choice prompts. It stops with state
`needs-owner` only for:
- sign-in, API keys or passwords (rule: never handle secrets)
- a trust prompt for a folder outside the goal's project
- publishing outward: push, release, deploy, or a live service
- deleting anything outside the goal's folder
- two failed attempts by two different models at the same task, when the Architect has no other path

Everything else the Architect decides itself and records in `events.jsonl`. Stopping keeps all work; the
owner answers once and the run continues.

## Product shape

- One per-user installer, no admin rights. Installs Deck, the pinned Herdr binary, the scripts, agent
  profiles and skills in one folder. Start-menu entry, uninstall entry.
- The Deck window is the only thing the owner opens. Setup page: checks each CLI, shows what is missing.
- App code lives in the install folder, user data in `%APPDATA%`, project data in the project.
- Source tree: `app/` (product), `scripts/` (PowerShell), `docs/`, `tests/`, `tasks/` (history).
  `herdr-app/` stays upstream.
- Later option, not in this spec: real native window (WebView2) in place of Edge `--app`.

## Commands

- Build: `pwsh -File app\build.ps1`
- Go tests: `go test ./...` in `app`
- Web tests: `node --test` in `app`
- Setup check: `pwsh -File check.ps1`
- Installer: `pwsh -File app\package.ps1` (new)

## Testing

- Supervisor: unit tests with a fake Herdr and fake clock for every rule above (idle, limit, dead,
  no progress, caps, needs-owner, done).
- Ledger: survives a kill mid-write; resume gives the same state.
- Lab: full run in an isolated Herdr session (CLAUDE.md section 3), never the owner's live server.
- Acceptance: see Success criteria.

## Boundaries

- Always: work on branches; agent work reviewed by a different model; every step in `events.jsonl`;
  HISTORY and LESSONS updated.
- Ask first: push, release, installer changes on the live PC, moving scripts (paths reach PATH,
  shortcuts, registry, skills).
- Never: handle secrets; answer sign-ins; widen sandboxes on its own; touch the owner's live server in
  tests; edit `herdr-app/`.

## Success criteria

1. In a fresh test project, the owner enters one goal and touches nothing else. It finishes with
   acceptance tests green, on a real goal that needs at least 5 tasks and 2 agent kinds.
2. Killing the Architect mid-run: the Supervisor restarts it and it continues with no lost task.
3. Hitting an account limit: it moves to the next account and continues.
4. A run lasting over 2 hours completes with no human input.
5. A planted secret request stops at `needs-owner` and is never answered.
7. Lab agents hitting a trust prompt, a menu, a stop, a capacity error and a kill each recover with no human.
6. A clean Windows user account: installer to working Deck with no admin rights.

## Milestones

| # | Result | Owner sees |
|---|---|---|
| M0 | Risk checks: trust prompt in worktrees, unattended pane starts, `agent_pane_busy` | short report |
| M1 | `internal/goal`: ledger + Supervisor + tests | nothing yet |
| M2 | `herdr-architect` skill; first real unattended run in lab | a small goal done alone |
| M3 | Deck "Goals" screen: new goal, live plan, tasks, log, pause/stop | the new main screen |
| M4 | Setup page + installer | one install file |
| M5 | Source tidy-up into `scripts/` with path migration | clean repo |
| M6 | Acceptance runs, criteria 1-6 | proof report |

## Known risks

- A prompt the Watchdog misreads could get a wrong answer or block a run; answers must act only on known
  patterns and confirm the result on screen.
- `herdr-agents start/team` fails with `agent_pane_busy`; the Supervisor needs a reliable start path.
- Deck's size feature uses Herdr internals, so the Herdr version must be pinned in the installer.
- Long runs cost a lot of Claude usage, hence hard limits per goal.
- Disk: I: 21.5 GB free, C: 20.6 GB free; agent builds are large. Runs must clean up their build folders.

## Open questions (defaults used until the owner says otherwise)

1. Default limits per goal? Default: 8 hours, 40 tasks, stop at account limits only.
2. May a run commit and merge to the project's main branch locally? Default: yes; never push.
3. Trust: answered by the Watchdog for folders inside the goal's project (owner decided 2026-10-30).
4. Install folder on this PC? Default: `I:\Programs\Herdr` (keeps C: free); elsewhere the per-user
   programs folder.
