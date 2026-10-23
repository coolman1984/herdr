# Task: find every real bug and broken feature in the Herdr toolkit (audit only, no fixes yet)

## 1. Role and goal
You are a senior PowerShell 7 / Windows engineer and code auditor working for the lead (Claude).
Goal: produce one findings file that lists every REAL defect in this toolkit - code that
crashes, gives a wrong result, silently does nothing, contradicts its own docs/help text, or
breaks an owner rule - each with proof and a proposed minimal fix.
Why: the owner sees features that do not work well. The lead will review your list, approve
items, and then give you a second task to fix the approved ones. A false or vague finding
costs the lead review time, so precision beats volume.

## 2. Context you need (nothing else is needed)
- Folder: your current folder is a git worktree of `D:\WORK\Software Development\GitHub\HERDR`
  on branch `fix/bug-audit`. Audit THE FILES IN YOUR CURRENT FOLDER.
- Read first, completely: `CLAUDE.md`, `AGENTS.md`, `LESSONS.md` (numbered durable facts -
  many "bugs" are deliberate workarounds explained there), `docs/FILES.md`; skim `HISTORY.md`
  (what each phase meant to achieve - a feature that does not do what its phase claims is a bug).
- The pieces:
  - `herdr-agents.ps1` (674 lines) - the delegation tool: start/ask/read/stop/team/collect/
    fanout/diff/usage/handoff/login/accounts... Profiles in `agents.json`.
  - `herdr-hud.ps1` - background HUD writing pane metadata; `claude-limits.ps1` (dot-sourced).
  - `herdr-layout.ps1`, `herdr-columns.ps1`, `herdr-spaces.ps1` - layouts/panes.
  - `herdr-open-folder.ps1`, `herdr-help.ps1`, `herdr-stop.ps1` - key-bound commands
    (`config/keys.toml`), run via `cmd.exe /d /c` + `conhost.exe --headless` (LESSONS #4, #39).
  - `install.ps1`, `install-custom-build.ps1`, `sync-skills.ps1`, `check.ps1`, `Herdr.cmd`,
    `install.cmd`, `herdr-agents.cmd`, `launcher/HerdrLauncher.cs` + `build.ps1` + `use-launcher.ps1`.
  - `skills/*/SKILL.md` and `docs/*.md` - documented commands must exist and behave as written.
- Herdr CLI syntax authority: `herdr-skill.md`. JSON shapes: LESSONS #6, #17, #18, #27.
- Known PowerShell traps to check for: single-item array unrolling from functions and
  `ConvertFrom-Json`; `$null` comparisons on the wrong side; `-eq` on arrays filtering instead
  of comparing; case-insensitive keys; `$ErrorActionPreference = "Stop"` vs native exit codes
  (`$LASTEXITCODE` not checked); `Start-Process` argument quoting with spaces (the repo path
  has spaces: `Software Development`); `Join-Path` with absolute second part; `-LiteralPath` vs
  `-Path` with `[` `]`; strings built for `cmd.exe`; `$PSScriptRoot` empty when dot-sourced
  wrongly; timeouts measured wrongly; regexes that miss; parameter sets that accept nonsense;
  error messages that point to the wrong fix.

## 3. Exact scope
- Create ONLY: `tasks/audit-findings.md` in your current folder.
- Do NOT change any other file. Do not run git commit/push/checkout/reset/stash.
- Do NOT touch the owner's live Herdr session: never run `herdr-agents.ps1 start|ask|stop|
  team|fanout|collect|login`, `herdr-layout.ps1`, `herdr-spaces.ps1`, `herdr-stop.ps1`,
  `herdr-hud.ps1`, `install*.ps1`, `use-launcher.ps1`, `sync-skills.ps1`, or any `herdr`
  command that creates, closes, moves, sends or stops anything - unless `HERDR_SOCKET_PATH`
  points at an isolated session you started yourself (below).
- Allowed: reading any file in this folder; the PowerShell parser
  (`[System.Management.Automation.Language.Parser]::ParseFile`); `pwsh -File check.ps1`
  (read-only); read-only herdr calls (`herdr --help`, `herdr agent`, `herdr pane list`,
  `herdr workspace list`); running small pure functions in a scratch `pwsh -NoProfile` by
  dot-sourcing or extracting them, with fake inputs; `herdr-agents.ps1 tools|status|usage|
  history|watch -Once|diff` (read-only commands).
- Optional, only if your sandbox allows it: an isolated test session
  (`herdr --session audit server` started in the background; `$env:HERDR_SOCKET_PATH =
  "$env:APPDATA\herdr\sessions\audit\herdr.sock"`; end with `herdr --session audit server
  stop`; LESSONS #20, #24). Never attach a window to it. If the sandbox blocks it, skip it
  and say so - do not try to bypass the sandbox.
- Never read secrets: no `auth.json`, `.credentials*`, tokens, `~\.codex\auth*`, keys.
- Do not start other agents. Do not use Chrome/browser tools or your own skills' scripts
  (LESSONS #31).
- Herdr path: `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe` (never bare `herdr`, LESSONS #1).

## 4. Design (decided by the lead - follow it)
- Method, per file: read it top to bottom; for every command/feature, trace one normal input
  and two edge inputs (empty / missing / one item / path with spaces or trailing `\` /
  agent not running / Herdr returns an error). Compare with what the help text, SKILL.md and
  docs say it does.
- What counts as a finding: a reproducible wrong behaviour. NOT a finding: style, naming,
  "could be cleaner", missing features nobody documented, anything LESSONS.md explains as
  deliberate, the approval-mode choices in CLAUDE.md 2.3 (owner decisions).
- Severity: `S1` = crash / data loss / touches the wrong pane/session/file / breaks a safety
  rule in CLAUDE.md; `S2` = a documented feature gives a wrong result or silently fails;
  `S3` = wrong message, docs/help disagree with code, minor edge case.
- Confidence: `PROVEN` (you ran it and saw it - give the command and the output) or
  `READ` (from reading the code - quote the exact lines and explain the input that breaks it).
  Prefer PROVEN where it is allowed by section 3.
- File format of `tasks/audit-findings.md`, exactly:

```
# Audit findings - <date>

Summary: <N> findings (S1: a, S2: b, S3: c). Files read: <list>. Isolated session used: yes/no (why).

## F01 [S1] [PROVEN] <file>:<line> - <one-line defect>
- Input / situation: <what the user or caller does>
- Expected: <what docs/help/phase say>
- Actual: <what happens; output quoted>
- Evidence: <command + output, or the exact code lines quoted>
- Proposed fix: <minimal change, in words or a short diff; name the function>
- Risk of the fix: <what else it could affect>

## F02 ...
```
- Order by severity, then file. Number F01, F02 ... without gaps.
- At the end add `## Checked and fine` - one line per file/feature you traced and found
  correct, so the lead knows what was covered.

## 5. Steps
1. Read the docs listed in section 2.
2. Run `pwsh -NoProfile -File check.ps1` and note every FAIL/WARN (a FAIL is a finding unless
   it is caused by your being in a worktree - say which).
3. Audit the files in this order: `herdr-agents.ps1`, `claude-limits.ps1`, `herdr-hud.ps1`,
   `herdr-layout.ps1`, `herdr-columns.ps1`, `herdr-spaces.ps1`, `herdr-open-folder.ps1`,
   `herdr-help.ps1`, `herdr-stop.ps1`, `config/keys.toml`, `agents.json`, `sync-skills.ps1`,
   `check.ps1`, `install.ps1`, `install-custom-build.ps1`, the `.cmd` files, `launcher/*`,
   then `skills/*/SKILL.md` and `docs/*.md` against the code.
4. For each suspected defect try to prove it within section 3's limits.
5. Write `tasks/audit-findings.md`. Re-read it: every finding has a file:line, evidence and a fix.

## 6. Definition of done (all must be true)
- [ ] `tasks/audit-findings.md` exists in the format of section 4, every file of step 3 appears
      either in a finding or under "Checked and fine".
- [ ] `git status --porcelain` in your folder shows ONLY `?? tasks/audit-findings.md`.
- [ ] No isolated session left running (if you started one).

## 7. Answer format
Reply with exactly:
DONE | BLOCKED | PARTIAL
Changed: tasks/audit-findings.md
Findings: S1 <a>, S2 <b>, S3 <c>
Checks: <each check from section 6 and its result>
Not done / doubts: <list, or "none">
