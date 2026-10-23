# Task: the HUD shows Cline agents' model and token use

## 1. Role and goal
You are a senior PowerShell 7 engineer working for the lead (Claude).
Goal: in `herdr-hud.ps1`, every pane running a Cline agent shows its real model and its token
use (fresh input and output), like Codex and Claude panes already show theirs.
Why: the owner wants cheap Cline agents to do most of the work and must see what they spend.
Today the lead had to dig the numbers out of Cline's session files by hand.

## 2. Context you need
- Folder: your current folder, git worktree on branch `round2`. Read first: `CLAUDE.md`,
  `AGENTS.md`, `LESSONS.md` (#26, #28, #41), then `herdr-hud.ps1` completely.
- Another agent (Codex) works in this SAME folder on `herdr-agents.ps1`, `herdr-trial.ps1`,
  `tests/test-round2-agents.ps1`, `docs/DELEGATION.md`, `docs/FILES.md`, `skills/`. Never touch them.
- How the HUD works: `Update-Panes` (line ~166) loops over panes; for an agent pane it builds
  `$parts` (the title) and `$tokens` (ordered: `path`, `name`, `model`, `cost`, `limit`, `ctx`,
  `out`, `account`), then `pane report-metadata` with `--token key=value`. Codex/Claude usage comes
  from `Find-SessionFile` + `Get-CodexUsage` / `Get-ClaudeUsage`. Cline gets nothing today: the
  model falls back to `Get-StartedModels` / agents.json.
- Cline session file (verified by the lead today):
  `%USERPROFILE%\.cline\data\sessions\<id>\<id>.json`, also under
  `%USERPROFILE%\.herdr-accounts\cline\<account>\data\sessions\` for separate accounts. Fields:
  `pid` (int), `cwd`, `model` (e.g. `cline-pass/deepseek-v4.1-flash`), `status`, `started_at`,
  `metadata.usage.inputTokens`, `.outputTokens`, `.cacheReadTokens`. Fresh input =
  `inputTokens - cacheReadTokens`. Other files in that folder (`*.messages.json`,
  `*.compaction.json`) hold conversation text - NEVER read them.
- Pane processes: `herdr pane process-info --pane <id>` returns JSON
  `.result.process_info.foreground_processes[]` with `pid`, `argv`, `cwd`, and `.shell_pid`.
- Style to copy: `herdr-hud.ps1` itself.

## 3. Exact scope
- Change: `herdr-hud.ps1` (new functions + the Cline branch in `Update-Panes`), `docs/GUIDE.md`
  (only the HUD section: one paragraph about Cline values).
- Create: `tests/test-hud-cline.ps1`.
- Do NOT touch anything else (not HISTORY/LESSONS - the lead writes them). Do NOT run the HUD
  for real (it writes metadata into the owner's live panes); do not run git
  commit/push/reset/stash/checkout; no agents; no secrets; no browser.

## 4. Design (decided by the lead - follow it)
- `Find-ClineSession([int[]]$Pids, [string]$Cwd) -> [string] path or $null`:
  scan the session roots for `<id>\<id>.json` files modified in the last 7 days (cheap filter
  first). 1) a file whose `pid` is in `$Pids` wins (newest if several); 2) else the newest whose
  `cwd` equals `$Cwd` (case-insensitive, trailing `\` ignored) and `status` is not `exited`/`done`;
  3) else `$null`. Cache: `$script:clineSessionCache[paneId] = @{ File; Pids }` and reuse while
  the pane's pid set is unchanged and the file still exists.
- `Get-ClinePids([string]$PaneId) -> [int[]]`: from `pane process-info` (foreground pids +
  shell_pid); failure -> empty array (`, @()` safe).
- `Get-ClineUsage([string]$File) -> pscustomobject @{ model; fresh; out }` from the session json;
  missing usage -> fresh/out `$null`. Parse with `ConvertFrom-Json`; any error -> `$null`.
- In `Update-Panes`, when `$kind -eq "cline"`: get pids, find the session, read usage. Model =
  session `model` when present (else today's fallback). Add `$parts += "in <fresh> · out <out>"`
  and tokens `in` and `out` (both via `Format-Tokens`). No `ctx` for Cline (no window data).
- Cost: process-info and file reads only for Cline panes; nothing changes for other kinds.
- Errors never stop the HUD loop: wrap in try/catch like the Codex/Claude branch.

### Tests: `tests/test-hud-cline.ps1`
Plain pwsh, same style as `tests/test-batch-a.ps1` (AST loader for the HUD functions, `Check`,
`PASS n/m`, exit 1 on failure). Use real temp files under `$env:TEMP` for fake session folders
(point the session roots at them through a variable/parameter you add for testability); mock
`Invoke-Herdr`. At least: `C1 pid match wins over newer cwd match`, `C2 cwd fallback ignores case
and trailing backslash`, `C3 exited sessions are not used by cwd fallback`, `C4 usage fresh = input
minus cache read`, `C5 missing usage gives nulls, bad json gives null`, `C6 process-info failure
gives empty pids`, `C7 Update-Panes adds in/out tokens and session model for a cline pane only`,
`C8 cache reused while pids unchanged`. Run them against the `HEAD` copy of `herdr-hud.ps1`
(temp folder) and report how many fail there.

## 5. Steps
1. Read section 2. 2. Write the tests, see them fail. 3. Implement. 4. Section 6. 5. Answer.

## 6. Definition of done
- [ ] `pwsh -NoProfile -File tests/test-hud-cline.ps1` -> `PASS n/n`.
- [ ] `tests/test-batch-a.ps1` and `tests/test-batch-b.ps1` still pass.
- [ ] `herdr-hud.ps1` parses (Parser::ParseFile, 0 errors).
- [ ] `git status --porcelain` shows only your section 3 files (Codex's files may also appear - ignore them).

## 7. Answer format (also write it to `tasks/ds-hud-cline-answer.md`)
DONE | PARTIAL | BLOCKED
Changed: <files>
Per item: <function / change> - test <names>
Checks: <each check of section 6 and its result>
Not done / doubts: <list or none>
