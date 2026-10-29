# Task: free space on drive C: safely, with proof

## 1. Role and goal
You are a careful Windows maintenance engineer working for the lead (Claude).
Goal: reclaim as much space on C: as is SAFE, using only the approved cleanup skills, and
report before/after free space with audit logs. Then list (read-only) the other big space
users on C: so the owner can decide about them.
Why: C: has only ~1.69 GB free of ~123 GB (2026-10-26). npm installs fail on a full disk
(LESSONS #60) and agents need temp space. A wrong deletion of user data is far worse than
freeing less space.

## 2. Context you need (nothing else is needed)
- Folder: `D:\WORK\Software Development\GitHub\HERDR`; read `AGENTS.md` (delegate rules).
- User profile on this PC: `C:\Users\m.labib`.
- Your own skills already hold the proven cleanup workflow - use them, do not rewrite them:
  - `C:\Users\m.labib\.codex\skills\c-drive-safe-cleanup\SKILL.md` (primary: Inspect -> Preview -> Clean -> VsCodePreview -> Inspect)
  - `C:\Users\m.labib\.codex\skills\windows-disk-cleanup-guard\SKILL.md` (extra approved paths: uv cache, VS Code CachedExtensionVSIXs, Temp; use its `-DryRun` first, then `-NoInteractive`)
- Earlier run (HISTORY Phase 21): C: 1.15 -> 3.66 GB via Temp, npm cache, crash dumps, pip, Recycle Bin.

## 3. Exact scope
- Allowed deletions: ONLY the approved targets of those two skills (temp, crash dumps,
  npm / pip / uv / codex-runtime caches, app updater caches, CachedExtensionVSIXs, Recycle Bin).
  Old VS Code updater folders only if VS Code is fully closed - never close it yourself.
- Allowed, after the above: `npm cache clean --force` is covered by npm-cache; nothing else.
- Do NOT delete or move: `Videos`, `Downloads`, `Desktop`, `Documents`, any browser user data
  (Chrome/Edge/other), `.claude`, `.codex`, `.herdr`, `%APPDATA%\herdr`, Windows, Program Files,
  ProgramData, `C:\Windows\Installer`, WinSxS, Windows Update files, pagefile/hiberfile, app
  install folders, anything under `D:\`. Do not run DISM, cleanmgr, `powercfg`, or anything elevated.
- Do not: run git, install anything, change settings, start other agents, use a browser,
  stop processes.
- Files locked in use: skip them, count them, do not retry in a loop.

## 4. Design (decided by the lead - follow it)
1. `Get-PSDrive C` -> record free bytes BEFORE.
2. c-drive-safe-cleanup: `-Mode Inspect`, `-Mode Preview` (check every target is in the
   approved list), `-Mode Clean`, `-Mode VsCodePreview` (clean only if VS Code is closed).
3. windows-disk-cleanup-guard: `scan-cleanup-targets.ps1`, `clean-cleanup-targets.ps1 -DryRun`,
   then `clean-cleanup-targets.ps1 -NoInteractive` for approved targets the first skill did not cover.
4. `Get-PSDrive C` -> free bytes AFTER.
5. READ-ONLY survey: the 25 largest folders under `C:\Users\m.labib` (depth 3, incl. AppData\Local
   and AppData\Roaming) and the sizes of `C:\Windows\SoftwareDistribution`, `C:\hiberfil.sys`,
   `C:\pagefile.sys`, `C:\$Recycle.Bin`. Use `-Force -ErrorAction SilentlyContinue` only for
   reading sizes and report the count of unreadable folders. Do not delete anything found here.
6. Write the full report to `D:\WORK\Software Development\GitHub\HERDR\tasks\sol-free-c-drive-result.md`.

## 5. Steps
Do 4.1 - 4.6 in order. If a script errors, report the exact error; do not improvise a new
deletion method.

## 6. Definition of done (all must be true)
- [ ] Before and after free bytes from `Get-PSDrive C` are in the result file.
- [ ] Every deleted path is in an approved list; audit-log paths are named.
- [ ] Skipped/locked counts are reported.
- [ ] Read-only survey table (path, size GB, suggestion: "owner decides" / "app-managed" / "keep").

## 7. Answer format
Reply with exactly:
DONE | BLOCKED | PARTIAL
C: free before -> after: <bytes / GB>
Freed by target: <target: size, ...>
Skipped/locked: <count and paths>
Audit logs: <paths>
Biggest remaining (read-only, top 10): <path size>
Result file: tasks\sol-free-c-drive-result.md
Not done / doubts: <list, or "none">
