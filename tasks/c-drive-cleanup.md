# Task: clean drive C with the c-drive-safe-cleanup skill (as usual)

## 1. Role and goal
You are the owner's Windows maintenance engineer. C: has about 1.1 GB free of 114.6 GB.
Goal: free space on C: with your skill `c-drive-safe-cleanup`, exactly as its SKILL.md says.

## 2. Context
- Skill: `%USERPROFILE%\.codex\skills\c-drive-safe-cleanup\SKILL.md` - read it first and follow it.
- Script: `%USERPROFILE%\.codex\skills\c-drive-safe-cleanup\scripts\run-c-drive-cleanup.ps1`.
- The owner asked for the cleanup ("clean the drive C as usual"), including `-Mode Clean`.

## 3. Exact scope
- Run ONLY: the skill script with `-Mode Inspect`, `-Mode Preview`, `-Mode Clean`,
  `-Mode VsCodePreview`, then `-Mode Inspect` again. Plus read-only commands to read the
  audit log in `%USERPROFILE%\.cache\disk-cleanup-guard\`.
- `-Mode VsCodeClean` only if VsCodePreview reports VS Code fully closed; never close VS Code.
- Never delete anything by hand, never use `-Force`, never edit the skill or its scripts.
- NEVER touch (even if large): `%LOCALAPPDATA%\GMES_Automation\`, `%LOCALAPPDATA%\GMES\`,
  `%LOCALAPPDATA%\Google\Chrome\CDP Profile`, any Chrome/Edge user data, Desktop, Documents,
  Downloads, Videos, Windows, Program Files, ProgramData. If Preview lists any of these: STOP.

## 4. Steps
1. Inspect -> note free space. 2. Preview -> check every target is an approved cache/temp.
3. Clean. 4. VsCodePreview (+ VsCodeClean only if allowed above). 5. Inspect again.

## 5. Definition of done
- [ ] Free space before and after reported, and the GB freed.
- [ ] Skips / locked files listed.
- [ ] Audit log path given.

## 6. Answer format
DONE | BLOCKED | PARTIAL
C: free before -> after (GB), freed (GB)
Cleaned: <target: size> ...
Skipped / locked: <list or "none">
Audit log: <path>
Not done / doubts: <list or "none">
