# Goal engine report (2026-10-30, lead: Claude Code cloud session)

Branch `ccr-efd59209-8jk5t6`. Code `app/internal/goal`, `app/cmd/herdr-goal`; doc docs/GOAL-ENGINE.md;
HISTORY Phase 57; LESSONS #103-#108.

## Starting point

- The delegate's M1 worktree (`I:\herdr\.wt\goal-m1`, branch `goal-engine`) and the Codex review file
  `tasks/goal-m1-review.md` were never pushed: no branch on origin has them. The engine was built
  fresh from the spec and the fix lists in `tasks/goal-m0-m1.md` and `tasks/goal-m2.md` (Part A items
  1-6 and the lead's directions for review findings #1, #2, #5, #9, #11, #12 are all covered: see the
  map below). If the delegate's branch still exists on the PC, compare before merging either.
- The container has no agent CLIs, no `pwsh`, and Herdr cannot be built (Zig blocked, LESSONS #108).
  So: real git, real processes, real files; scripted agent stand-ins (`app/tests/goalagent`).

## Fix list map

| Item | Where |
|---|---|
| A1 rate-limit loop, no usage data, never the owner | `accountLimited`: per profile/account "limited until" from the screen (else 1 h), next account, next profile, `waiting-limit` + resume. Test `TestAccountLimitsRotateThenWaitThenResume` |
| A2 stall never fires | `ProgressHash` drops spinner/timer lines, digits, bottom 3 lines. `TestProgressHashIgnoresTimerAndSpinner`, `TestStalledAgentIsNudged` |
| A3 no answer file | every delegate gets `answers/<agent>.json` in its brief, checked against the message id |
| A4 secret prompt with status not blocked | status-free `secret-question` / `secret-field` lines; `TestSecretQuestionButNotPasswordTalk` |
| A5 acceptance blocks the tick | checks, merges and acceptance are background jobs picked up on later ticks |
| A6 non-git folder | refused with a message (`TestCreateRefusesNonGitFolderAndCostlessProfiles`) |
| Review #1/#2 nothing typed into a secret screen | `perform` re-reads the screen before every send/keys. `TestNothingIsTypedIntoASecretScreen` |
| Review #5/#9 lost or doubled actions | intents + `Recover`; `TestRecoverNeitherLosesNorDoublesAStart`; e2e runner kill -9 |
| Review #11 routine menus | `decider` command (validated "PICK n"), else the CLI default, recorded as an assumption |
| Review #12 cost cap | no cost limit at all (no source); spec updated |
| RAM rule | `max_agents` 2, no start under `min_free_ram_mb` 1536 / free disk. `TestMaxAgentsAndFreeMemoryGate` |

## Evidence

- `go test ./internal/goal`: 33 tests pass. `python3 app/tests/goal-mutations.py`: 24/24 rules, each
  broken alone, make their test fail. `node --test`: 42/42. `GOOS=windows go vet ./...`: clean.
- e2e (`app/tests/goal-e2e/run.sh`), goal "wordstat", 5 tasks, writers alpha + beta (two agent kinds),
  reviewers always the other kind, lead agent present. Timeline (UTC):

```
15:37:36 runner start; T1 trust prompt ("No, exit" highlighted) -> keys [1]
15:37:41 T4 account alpha/a1 limited ("resets in 0h 1m") -> restarted on a2
15:37:48 runner killed -9; 15:37:50 lead agent killed -9
15:37:53 new runner; a second runner is refused ("already running")
15:37:53 T3 stopped without result -> Enter; 15:38:00 -> "continue" -> result
15:37:55 lead dead -> restarted
15:38:01 T2 writer crashed -> half-done work committed -> restarted in the same worktree
15:38:03 T3 merge conflict in README.md -> back to writer -> resolved -> merged 15:38:12
15:38:15 T5 "model at capacity" -> "continue"; 15:38:18 reviewer rejects -> writer fixes
15:38:27 Supervisor's own acceptance green -> DONE (51 s, 175 events, 0 human inputs, 0 agents left)
```

- Soak (`app/tests/goal-e2e/soak.sh`, 35 min): see the table appended below.

## Doubts

See docs/GOAL-ENGINE.md "Not proven yet". The independent review of this engine by another vendor
(Codex) was not possible from this session: the owner's rule forbids the lead's own subagents and no
other agent CLI is reachable here. Next step on the PC: `tasks/goal-m1-review.md.spec` against this branch,
then a lab run with `--backend herdr` and real Claude + Codex.
