# Goal engine M0 + M1 (Opus 5.5 medium)

Read I:\herdr\CLAUDE.md, AGENTS.md (delegate rules) and docs\SPEC-GOAL-ENGINE.md first. The spec is the
contract.

Work in worktree `I:\herdr\.wt\goal-m1`, branch `goal-engine` (create it from main). Commit there.
Never push, never merge. Do not touch the owner's live Herdr server or running Deck; test in an isolated
session (CLAUDE.md section 3). Do not change trust or approval settings.

## M0: settle the risks. Write tasks/goal-m0-report.md (facts with commands and output)
1. Does a folder trusted once in Claude Code / Codex cover a git worktree inside it? Inspect how trust is
   stored without printing any secret or auth file (never read credentials). Say exactly which paths
   stop an unattended run, and propose the option that needs the fewest owner answers.
2. Why does `herdr-agents start/team` give `agent_pane_busy`, and what is the reliable way to start an
   agent in a new pane from code? Reproduce in the lab session.
3. How can code tell, with no screen scraping if possible, that an agent is idle, working, blocked or
   dead (`agent_status` in `pane list`, hooks)? Note gaps. Lesson: a narrow pane cut off the
   "esc to interrupt" text and fooled a screen-reading watcher.
4. How to read an account's 5-hour usage from code (`claude-limits.ps1`); `usage` showed "no data" on
   2026-10-29 - find out why.

## M1: `app/internal/goal` (Go standard library only)
- Ledger per spec: goal.json, tasks.json, events.jsonl, plan.md in `<project>\.herdr-goal\<goal-id>\`.
  Atomic writes; a kill mid-write loses nothing; loading gives the same state.
- Supervisor per spec, section "Design" item 3: pure decision step
  `Step(state, observations, now) -> actions`, with Herdr and the clock behind small interfaces.
  Every rule tested with fakes: idle, 95% limit handoff, dead restart, no progress, caps, needs-owner,
  done requires the Supervisor's own acceptance run.
- No UI and no Deck server routes yet. Nothing starts a real agent in tests.

## Done means
`go test ./...` and `node --test` pass in `app`; tests fail when the rule is removed (show one mutation per
rule); HISTORY entry, LESSONS if durable; `pwsh -File check.ps1` shows only the 2 known failures
(lesson 66, Phase 43).

## Answer format
Short: M0 answers, M1 files, test lines, what the spec got wrong, doubts.

## Added by the owner (2026-10-30): Agent Watchdog - top priority inside M1
Owner: "Opus 5.5 is replacing the human all the time, no need for a human until the goal is met."
Read the new section "Agent Watchdog" in I:\herdr\docs\SPEC-GOAL-ENGINE.md (main branch, committed; the
copy in your worktree is older) and build it in `app/internal/goal`:
- Launch check, state sensing, answers, retry caps, escalation and event records, as the spec says.
- `prompts.json` with the real prompts this project has met: Claude folder trust (default highlighted is
  "No, exit", so move to Yes, then confirm, and read the screen back), Codex trust, Cline pop-up (esc),
  Cline queued message (empty Enter), Codex "at capacity", Cline "model not found", Claude rate limit.
  Sources: LESSONS.md #98-100, #47, #91 and the `ask` dialog guard `Get-AgentDialog` in herdr-agents.ps1,
  with its fixture tests/fixture-claude-answer-mentions-trust.txt (an answer that MENTIONS trust is not a
  prompt: this false positive must have a test).
- Never answers secrets or sign-ins; trust only for folders inside the goal's project. Tests for both.
- Each rule tested with fake screens; one mutation per rule. Real lab proof for the Claude trust prompt:
  start an agent in a fresh folder in the isolated session and show the Watchdog gets it working.
