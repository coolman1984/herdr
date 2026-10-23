# Task: two corrections to batch A (lead review)

You are a senior PowerShell 7 engineer working for the lead (Claude). Batch A
(`tasks/codex-fix-a.md`) is implemented in this folder and its suite passes 43/43. The lead's
review found two defects. Fix only these, with tests. Same scope and safety rules as
`tasks/codex-fix-a.md` section 3 (read it first).

## R1 - NOT READY agent in a supplied pane is not recorded (regression)
`herdr-layout.ps1` (line 319) and `team` create the pane themselves, then call
`herdr-agents.ps1 start <agent> -Pane P`. If the agent comes up NOT READY (sign-in or trust
question: `$started.Ok` is false), `start` now saves no record, so `stop` later refuses it.
Fix in `start`: when `-Pane` was supplied and the start failed, still save the record IF
`agent list` shows an agent named `$Name` in pane `$pane` (Herdr keeps a not-ready agent's
identity). Otherwise save nothing (F03 stays as it is).
Tests: `R1 not-ready agent in supplied pane is recorded`, `R1 failed start without identity saves nothing`.

## R2 - key-binding cwd overrides -Pane
`herdr-agents.ps1` line ~277 sets `$Cwd` from `HERDR_ACTIVE_PANE_CWD` before the `-Pane` cwd
check, so `start x -Pane P` run from a key can fail with a mismatch the user never typed.
Apply `HERDR_ACTIVE_PANE_CWD` only when `-Pane` is not given.
Test: `R2 active-pane cwd ignored with -Pane`.

## Also
- Add one bullet `- R1/R2 (lead review): ...` to Phase 25 in `HISTORY.md`.
- Add the tests to `tests/test-batch-a.ps1`; run the full suite, and the HEAD-baseline run.

## Answer format
DONE | PARTIAL | BLOCKED
Changed: <files>
Tests: <the three names> - suite <n/n>; HEAD baseline <failures/total>
Doubts: <list or none>
