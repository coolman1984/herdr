# Task: fix the ready signal of `ask -Fresh` (live test failed)

You are a senior PowerShell 7 engineer working for the lead (Claude). Read `AGENTS.md` first.
Folder: your current folder (worktree, branch `round2`). Context: `tasks/astra-round2.md`
section 4A describes `ask -Fresh`; `herdr-agents.ps1` has `Start-FreshConversation`.

## What the lead measured live (2026-10-23 16:58, Codex 0.163, your own pane)
- `agent_session.value` was `01a168f5-...` before `/new` and STILL `01a168f5-...` 33 s after:
  Codex creates the new conversation id only with the first message. So "wait for a new session
  id" (the current code, the lead's earlier correction) never succeeds -> `-Fresh` always fails
  after 20 s. The lead's design was wrong; this replaces it.
- The visible screen of a FRESH conversation: the banner line `>_ OpenAI Codex (v...)`, the logo,
  then the ready line `› Ask Codex to do anything` - and NO transcript lines. An OLD
  conversation always has transcript lines starting with `• ` (e.g. `• Ran ...`, `• Edited ...`,
  `• DONE`, `Worked for ...`), even when its ready line is shown.

## Change (only `herdr-agents.ps1` `Start-FreshConversation`, and `tests/test-round2-agents.ps1`)
- Ready = the menu text is NOT on screen AND the screen contains `>_ OpenAI Codex` AND
  `Ask Codex to do anything` AND no line matching `^\s*• ` (multiline regex) AND no line
  containing `Worked for`. Keep: answering the worktree menu once with `Enter`, the 20 s timeout
  and its message. Remove the session-id logic and the "no initial session id" branch.
- Tests: replace A5/A6 with `A5 old conversation screen with bullets does not count`,
  `A6 fresh banner screen without bullets returns`, `A7 menu answered then fresh screen returns`;
  keep A1-A4 and B1-B11 passing. Use real screen texts like the ones above in the fixtures.
- Do not touch other files. No git commits. Do not run `ask -Fresh` live (the lead will).

## Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Tests: <names> - suite <n/n>
Doubts: <list or none>
