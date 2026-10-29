# Task: watch Codex (gpt-6-astra) while it verifies Herdr Deck - Claude Sonnet 5.5 high

## 1. Role and goal
You are the watcher for the lead (Claude Opus, the pane on the left). Codex runs in the Herdr pane named
`codex` with the spec `tasks/deck-r3-codex-terra.md` (live verification of the Deck; it must NOT change
product code). Goal: keep a short, honest log of what Codex does, catch scope breaks, loops and stalls
early, and judge its work quality so the lead can score it.

## 2. How to watch (read-only)
- Tool: `pwsh -NoProfile -File "D:\WORK\Software Development\GitHub\HERDR\herdr-agents.ps1" read codex -Visible`
  (screen now) and `... status`. Every 4-5 minutes, until Codex shows its final answer or the lead stops you.
- Scope check every round: `git -C "D:\WORK\Software Development\GitHub\HERDR" status --short` and
  `git diff --stat`. Codex may only change `app/tests/*.ps1`, `app/tests/shots/*`, `app/FEATURES.md`,
  `app/API.md`, `app/PERF.md`, `tasks/deck-r3-codex-*.md`. The LEAD changes `app/web/**`,
  `app/internal/**`, `app/tests/deck-ui.ps1` and `tasks/deck-r3-lead-notes.md` at the same time - those are
  not Codex's unless the screen shows Codex editing them.
- Also read `tasks/deck-r3-codex-bugs.md` when it changes: is each bug reproducible as written (steps,
  expected, observed, evidence)?

## 3. Rules
- Never type into Codex's pane, never answer its questions or approvals, never stop it, never start
  agents, never edit files except your log. If Codex is BLOCKED (approval/question/sign-in), write ALERT
  in the log at once.
- Never read auth/credential files. No browser.

## 4. Output
Append to `tasks/deck-r3-sonnet-watch-log.md`, one block per round:
`## HH:MM` - what Codex is doing (1-3 lines), files changed since last round, `OK` or `ALERT: <reason>`
(scope break, product code edited, loop: same step 3 rounds, stall: screen unchanged 3 rounds, blocked,
claims without evidence, process left running).
When Codex finishes (or after 3 hours), append `## Verdict`: score 0-10 for each of: following the spec,
scope discipline, evidence quality (claims proven by output), bug report quality, test script quality
(`app/tests/verify-ui.ps1` repeatable, cleans up), speed/efficiency; an overall score with 3-6 lines of
reasons; then reply DONE with the overall score.
