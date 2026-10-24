# Task: find the cause of the 20 `check.ps1` failures and fix what is safe

## 1. Role and goal
You are a senior PowerShell engineer working for the lead (Claude).
Goal: explain why `pwsh -File I:\herdr\check.ps1` reports 20 FAIL lines and fix the ones that can be
fixed safely, with proof.
Why: the owner asked to look into them (2026-10-24). Herdr's config and skills seem reset.

## 2. Context you need
- Folder: `I:\herdr`; read `CLAUDE.md`, `AGENTS.md`, `HISTORY.md`, `LESSONS.md` first.
- The FAILs: 10 key bindings from `config/keys.toml` and 6 settings from `config/settings.toml`
  missing in `%APPDATA%\herdr\config.toml`; skills `herdr-team` and `herdr-workspace` differ from
  `~/.claude/skills` (`sync-skills.ps1`); App Path `herdr.exe` missing or broken; `docs/FILES.md`
  lists `herdr/` which does not exist. Also WARN: HUD startup shortcut missing, folder not on PATH.
- Relevant HISTORY phases: 23 (launcher/App Path), 34 (herdr-app merge), earlier config phases.

## 3. Exact scope
- Diagnose first: for each FAIL say what is wrong and since when (check backups
  `%APPDATA%\herdr\config.toml.*.bak`, file dates, `git log`).
- You may change: `docs/FILES.md` (the `herdr/` row: the upstream clone is not on this PC), and
  with a backup first (rule 6): `%APPDATA%\herdr\config.toml` (re-add our key/setting blocks the way
  the install docs say), run `sync-skills.ps1`, `launcher\use-launcher.ps1` for the App Path.
- Do NOT touch: `~/.claude/settings.json`, `herdr-app\`, approval/trust settings, other panes.
  No git. Do not stop the Herdr server. Test only in an isolated session (`--session lab`).
- If your sandbox blocks a write outside `I:\herdr`, do not bypass it: give the exact commands.

## 4. Steps
1. Run `check.ps1`, capture the FAIL list.
2. Find the cause of each group (config reset? new PC? Herdr update 0.9.5 changed config format?).
3. Fix what is allowed, back up first, say where the backup is.
4. Re-run `check.ps1`; report the FAIL count before and after.
5. Add a HISTORY entry (next phase number) and a LESSONS entry for the real cause.

## 5. Definition of done
- [ ] Every FAIL has a stated, evidenced cause.
- [ ] `check.ps1` FAIL count lower, or each remaining FAIL has exact owner commands.
- [ ] Docs updated; `check.ps1` docs line still OK.

## 6. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Checks: <before/after FAIL counts, causes>
Not done / doubts: <owner commands, unverified items>
