# Task: `ask` must not type a task into a startup dialog

## 1. Role and goal
You are a senior PowerShell 7 engineer working for the lead (Claude). Read `AGENTS.md` first.
Goal: before `herdr-agents ask` sends a task, it reads the agent's visible screen; if the agent
shows a dialog that only the owner may answer (folder trust, browser tools, sign-in, model pick),
`ask` sends NOTHING and reports BLOCKED with the screen.
Why (real incident, LESSONS #53): a new Claude account opened with "Yes, use my browser / Enter to
confirm". A task prompt sent then was lost and its Enter likely confirmed the dialog - a choice
that belongs to the owner (CLAUDE.md safety rule 2).

## 2. Context
- Folder: your current folder (a git worktree on your own branch). Read `CLAUDE.md`, `AGENTS.md`,
  `LESSONS.md` (#21, #41, #44, #53), `herdr-agents.ps1` (`Complete-Ask`, the `ask` branch,
  `Invoke-Herdr`, `Format-Screen`), `tests/test-batch-a.ps1` (test harness style).
- `herdr agent read <name> --source visible` prints the visible screen as plain text.
- Real dialog texts seen on this PC:
  - Antigravity: `Do you trust the contents of this project?` / `> Yes, I trust this folder` / `No, exit`
  - Claude Code (new account): `Yes, use my browser` / `Enter to confirm · Esc to keep browser tools off`
- Real idle screens: Claude ends with `❯ Try "write a test for <filepath>"` and a status line;
  Codex ends with `› Ask Codex to do anything`; both may show old conversation text above.

## 3. Exact scope
- Change only `herdr-agents.ps1` (new function + the `ask` branch) and add `tests/test-dialog-guard.ps1`.
- Do not touch anything else. No git commit. Do not run live herdr commands that change
  anything; no agents.

## 4. Design (decided by the lead - follow it exactly; hidden tests check it)
- New function `Get-AgentDialog([string]$Screen)` returns `$null` or
  `[pscustomobject]@{ Name = <'trust'|'browser'|'sign-in'|'model'>; Line = <the matching line, trimmed> }`.
  - Look ONLY at the last 20 non-empty lines of `$Screen` (older text is conversation history).
  - Case-insensitive patterns, first match wins in this order:
    - `trust`:   `do you trust`
    - `browser`: `use my browser` or `browser tools`
    - `sign-in`: `not logged in`, `not signed in`, `please (log|sign) in`, `run /login`
    - `model`:   `(select|choose|pick) a model`
  - Empty or `$null` screen -> `$null`.
- In the `ask` branch, after the task is known and BEFORE `agent prompt` (and before `-Fresh`):
  `$screen = Invoke-Herdr @("agent", "read", $Target, "--source", "visible")`.
  - If the read fails: continue exactly as today (the guard must never stop a normal ask).
  - If `Get-AgentDialog $screen.Text` finds a dialog: write
    `BLOCKED: <Target> shows a <Name> dialog ("<Line>") - the owner answers it in the pane.`,
    then `--- screen of <Target> ---` and `Format-Screen $screen.Text`; save the run with
    `Save-Run $Target $Task "BLOCKED" <that text>`; print `--- saved: <file> ---`; `exit 3`.
    No `agent prompt` call may happen.
- Update the header comment of `ask` (one short line).

### Tests: `tests/test-dialog-guard.ps1`
Same harness style as `tests/test-batch-a.ps1`; your own cases, including real screens from
section 2, history text above the last 20 lines, and a failed read.

## 5. Definition of done
- [ ] Your tests pass; `tests/test-batch-a.ps1` 48/48 and `tests/test-round2-agents.ps1` 18/18 still pass.
- [ ] Your tests fail on the previous `herdr-agents.ps1` (`git show HEAD:herdr-agents.ps1`).
- [ ] `herdr-agents.ps1` parses.

## 6. Answer format (also write it to `tasks/dialog-guard-answer.md`)
DONE | PARTIAL | BLOCKED
Changed: <files>
Tests: <names> - suite <n/n>; on HEAD: <failures>
Doubts: <list or none>
