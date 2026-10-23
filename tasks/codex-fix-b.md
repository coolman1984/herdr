# Task: fix batch B - other scripts, launcher, installer, skills and docs (+ regression tests)

## 1. Role and goal
You are a senior PowerShell 7 / Windows engineer working for the lead (Claude). Fix the
approved findings below with the smallest correct change, and prove every code fix with a
regression test that fails on the old code. Why: these are the remaining defects of your audit;
the lead reviews every line, and a fix without a test or a change outside the list is rejected.

## 2. Context you need
- Folder: your current folder, git worktree on branch `fix/bug-audit` (batch A is committed:
  `git log -1`). Read first: `CLAUDE.md`, `AGENTS.md`, `LESSONS.md` (#17, #39, #40, #45),
  `tasks/audit-findings.md` (your evidence and proposals), and `tests/test-batch-a.ps1`
  (copy its loader and `Check` style).
- This spec overrides the audit's proposals where they differ.

## 3. Exact scope
- Change: `install.ps1` (F04, F30), `install.cmd` (F22), `Herdr.cmd` (F21),
  `herdr-open-folder.ps1` (F17), `herdr-spaces.ps1` (F18, F19), `herdr-stop.ps1` (F20),
  `launcher/HerdrLauncher.cs` + rebuilt `launcher/HerdrLauncher.exe` (F23),
  `skills/herdr-workspace/SKILL.md` (F05), `skills/herdr-team/SKILL.md` (F26, M2 text),
  `docs/CLINE-PASS.md` (F25), `docs/DELEGATION.md`, `docs/GUIDE.md`, `docs/SETUP.md` (F09, F26,
  F27, F28, M2 text), `docs/FILES.md` (tests row), `HISTORY.md` (new Phase 26 only).
- Create: `tests/test-batch-b.ps1`.
- Do NOT touch anything else; do NOT run `sync-skills.ps1`, `install*`, `use-launcher.ps1`,
  `herdr-stop.ps1`, `herdr-spaces.ps1` live, or any live-mutating herdr command. No git
  commit/push/reset/stash. No secrets, no other agents, no browser.

## 4. Design (decided by the lead - follow it)
- **F04 install.ps1 `Update-PathRegistryEntry`:** before `SetValue`, write a backup JSON
  `{"existed":bool,"value":"<raw, unexpanded>","kind":"<RegistryValueKind>","time":"<s>"}` to
  `%APPDATA%\herdr\backup\user-path-<yyyyMMdd-HHmmss>.json` (unique name; never overwrite). If the
  backup cannot be written, throw and do NOT change the registry. Print `PATH backup: <file>`.
  Read the raw value with `DoNotExpandEnvironmentNames` (as the code already does).
- **F30 `Remove-OldReleases`:** the current release always survives and uses one of the `Keep`
  slots; keep only `Keep - 1` newest non-current releases. `Keep < 1`: unchanged behaviour.
- **F22 install.cmd:** run `pwsh.exe` when `where pwsh.exe` finds it; otherwise fall back to
  `powershell.exe` and print one line: `PowerShell 7 not found - using Windows PowerShell.`
  Keep exit-code forwarding. (Fallback stays: a fresh PC may not have pwsh yet.)
- **F21 Herdr.cmd:** before the `wt.exe -d "%START_DIR%"` line, if START_DIR ends with `\`,
  append one more `\` (LESSONS #40). Do not touch interior backslashes or the `cd /d` path.
  Test by building the exact command line for `D:\`, `D:\a b\`, `D:\a b` and splitting it with
  `CommandLineToArgvW` (P/Invoke shell32) - expect the folder as one argument, unchanged.
- **F23 HerdrLauncher.cs `ResolveFolder`:** return `Path.GetFullPath(argFolder)` for an existing
  folder; keep the System32 fallback and trailing-root handling. Rebuild with
  `launcher/build.ps1` (writes only `launcher/HerdrLauncher.exe`). Test by reflection on the
  rebuilt exe (no `Main`): `ResolveFolder('docs')` from the repo root -> absolute path ending `\docs`.
- **F17 herdr-open-folder.ps1 team:** read the `codex` profile from `agents.json` and pass
  `-- <modelFlag> <model>` to `agent start` (only these two; no other profile args - approval
  settings of this shortcut are not changed here). Missing profile/model -> start as today.
- **F18 herdr-spaces.ps1 `Join-Spaces`:** equal rows: when splitting the bottom pane for pane
  k of n (k = 1..n-1), the ORIGINAL pane keeps `1/(n-k+1)` of the remaining height
  (LESSONS #17). Test: simulate the recurrence for n = 2, 3, 5 -> all heights equal (±1e-9).
- **F19 `Split-Spaces`:** if `pane list` fails -> show the error, keep the record, exit. After
  moves, rewrite the record with only the groups not fully separated (with new pane ids); delete
  it only when every group is separated.
- **F20 herdr-stop.ps1 `Get-Running`:** return `[pscustomobject]@{ Ok = <bool>; Agents = <array> }`
  (`, @(...)` safe), so 0 agents on a running server still offers the stop dialog, and a failed
  call says the server is not reachable. Callers updated.
- **F05 skills/herdr-workspace/SKILL.md:** replace the "Live tests" procedure with the isolated
  session procedure from CLAUDE.md section 3 (absolute herdr path, `--session lab server`,
  `HERDR_SOCKET_PATH`, headless, `server stop` at the end; live session only for what lab cannot show).
- **M2 text:** batch A changed `start -Worktree` to open beside the caller (new space only
  outside Herdr). Fix every doc/skill line that still says "new space" for `-Worktree`.
- **F09/F25/F26/F27/F28 docs:** as in the audit's proposals: GUIDE "cost today" -> "session cost
  (updated today)" with its meaning; CLINE-PASS stale re-auth text marked as history; Cline
  profiles = Cline Pass (`cline` glm-5.3, `cline-flash` glm-5.3-flash, `deepseek`
  deepseek-v4-pro, `kimi` kimi-k3); GUIDE section 5 dollar-cost sentence; SETUP uses the current
  launcher (`launcher/build.ps1` + `use-launcher.ps1`), all key blocks, HUD/Stop shortcuts, with
  `Herdr.cmd` named as the fallback. Keep the docs' plain style; do not rewrite unrelated text.

### Tests: `tests/test-batch-b.ps1`
Same style as batch A (`Check`, `PASS n/m`, exit 1 on failure, AST loader, mocks; never touch
the real registry, real files outside `$env:TEMP`, or Herdr). One or more tests per code item:
F04 (backup written before SetValue; backup failure -> no SetValue), F30, F21, F22 (parse the
batch text for the pwsh/fallback branches), F23, F17, F18, F19, F20. Run the suite against the
`HEAD` copies too (temp folder) and report how many fail there.

### HISTORY.md
`## Phase 26 - Audit batch B: launcher, installer, spaces, stop, docs (2026-10-23)` in the
Symptom / Cause / Fix / Proof / Lesson style, one bullet per item id.

## 5. Steps
1. Read section 2. 2. Write the tests, see them fail. 3. Fix item by item. 4. Section 6. 5. Answer.

## 6. Definition of done
- [ ] `pwsh -NoProfile -File tests/test-batch-b.ps1` -> `PASS n/n`; batch A suite still 46/46.
- [ ] HEAD baseline of batch B suite: every code item's tests fail.
- [ ] All `.ps1` parse; `launcher/build.ps1` exit 0.
- [ ] `pwsh -NoProfile -File check.ps1` -> only the 3 known worktree FAILs.
- [ ] `git status --porcelain` shows only section 3 files.

## 7. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Per item: <id> - <one line> - test <name or "docs">
Checks: <each check of section 6 and its result>
Not done / doubts: <list, or "none">
