# Goal engine: M1 fixes + M2 (first real unattended run)

Same rules and worktree as tasks/goal-m0-m1.md (`I:\herdr\.wt\goal-m1`, branch `goal-engine`). First run
`git merge main` so you have the current spec and CLAUDE.md. Commit on this branch; never push.

## Part A: the lead's review of M1 found these. Fix each, failing test first
1. **Rate limit loop, no cap** (`rateLimited` -> `restart("rate-limit")`): two limited accounts swap for ever.
   Also usage data is gone, so `NextAccount` is always empty, and after `rateWait` the Architect's limit
   goes to `needs-owner`: a human is pulled in by a normal event. Fix: no usage data needed. Rotate through
   the accounts in agents.json (`accountNames`) reactively: record each account's "limited until" in the
   ledger (the time printed on screen, else one hour); restart on the next free account, Architect included.
   All accounts limited: state `waiting-limit`, resume by itself at the earliest reset. Never `needs-owner`.
   Owner's decision stands: no human until the goal is met.
2. **Stall can never fire**: `OutputHash` covers the whole screen, and Claude's running timer changes it
   every second. Hash a normalized screen: drop spinner and timer footer lines, digits, and the bottom
   status block. Prove with a real captured "working" screen pair.
3. **An agent with no answer file can never be done**: it falls to `stopped-early` and gets Enter and
   "continue" after it finished. Every non-Architect agent must have an answer file (generate
   `<ledger>\answers\<agent>.md` when the spec has none) and its first prompt must say to write it last.
4. **A password prompt with Herdr status not `blocked`** is missed: `any-secret-request` needs `blocked`, so
   an idle agent asking for a password gets Enter and "continue". Add a status-free entry for a real
   question for a secret in the last lines (a line asking to enter/type/paste one, ending in `:` or `?`),
   with a test that a normal reply mentioning "password hashing" is NOT matched.
5. `perform` runs acceptance inside the tick, so the Watchdog is blind for up to `verifyTimeout`. Run it in
   the background and pick up the result on a later tick.
6. A goal folder that is not a git repo makes every tick fail at `Head`. Goal creation must refuse with a
   clear message or `git init` it; pick one and say which.
Also merge any finding in `tasks/goal-m1-review.md` (a Codex reviewer is writing it now) that you can
confirm; say which you reject and why.

## Part B: M2
Goal: one command starts a goal, and it finishes with nobody touching anything.
1. Runner entry point: a sub-command of the Deck binary (`HerdrDeck.exe goal new|run|status|stop`) or a small
   `app/cmd/herdr-goal`, your choice. `run` is the tick loop: survives its own restart from the ledger.
2. What the Architect uses to act: commands to add/update tasks, start an agent for a task (the Watchdog
   then owns it), mark a task, answer an escalation, record a decision. It must never need to write ledger
   JSON by hand. It may not stop the loop or mark the goal done; only the Supervisor's own acceptance run does.
3. Skill `skills/herdr-architect/SKILL.md` (master copy, not synced yet): understand the goal, plan,
   split, write a spec per task from tasks/TEMPLATE.md, choose agents with docs/AGENT-ROSTER.md, never let a
   writer review its own work, review each result itself, next task, repeat. Reuse docs/TEAM-PLAYBOOK.md;
   do not copy it. The Architect is told plainly: no human is coming, decide and record.
4. Real lab run, isolated Herdr session, real CLIs, project `I:\HERDR-trials\goal-m2` (new git repo): a small
   but real goal needing at least 3 tasks and 2 agent kinds, e.g. a tiny CLI tool with tests and a README,
   acceptance = its test command. Architect on account 2040. During the run kill one agent once to prove
   recovery. Do not touch the run otherwise. Keep the ledger as proof.
5. Clean up every pane and session you started.

## Done means
Part A tests fail without each fix; `go test ./...` and `node --test` pass; check.ps1 shows only the 2 known
failures; HISTORY entry; LESSONS for durable facts (real prompt wordings, especially a captured Claude limit
message if one appears); `tasks/goal-m2-report.md` with the run's timeline, events, tasks, how many times a
human was needed (must be 0), usage cost if known.

## Answer format
Short: Part A results, M2 run result, what broke, doubts. No decisions for the owner unless truly the owner's.

## Added by the lead: Codex review result (mandatory, before the lab run)
Read tasks/goal-m1-review.md in your worktree. All 12 findings are confirmed real by their proofs. Fix ALL 12,
P1 first, before Part B step 4. Its 15 failing tests are in
C:\Users\mflma\AppData\Local\Temp\goal-m1-review-309153d5bacc4671bdbda90dd1611a79\goal\review_test.go:
copy them into `app/internal/goal` and make them pass; do not weaken one (say so if a test itself is wrong).
Direction for three of them:
- #1, #2: nothing is ever sent to a pane showing a secret request, from any path, wrapped or not. Check the
  recipient's current screen right before every send, in `perform`.
- #5, #9: keep a pending-action record so a crash can neither lose nor double an action; restart must
  reconcile with live panes first.
- #11: the Architect's own routine menu must not reach the owner: run a one-shot `claude -p` decider on another
  account with the screen and the goal, send its pick, record it. Owner only for the spec's listed reasons.
- #12: `ShellProbes.Cost` has no real source: refuse a positive cost cap at goal creation, say why.
Memory is short on this PC (3.6 GB free of 15.8 at 2026-10-30): at most 2 agents at once in the lab run, one
Rust/Go build at a time, and close each pane as soon as its task ends. The engine needs this as a rule too:
a `MaxAgents` limit (default 2) and no new agent while free RAM is under 1.5 GB.
