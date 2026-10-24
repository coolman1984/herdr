# Task: point the ten Herdr key bindings at scripts that exist in I:\herdr

## 1. Role and goal
You are a senior PowerShell engineer working for the lead (Claude). The owner approved this task and
full disk access for it (2026-10-24).
Goal: every key binding in `config/keys.toml` (and the same blocks in `%APPDATA%\herdr\config.toml`)
runs a script that really exists, from `I:\herdr`, and works.
Why: `check.ps1` passes, but all ten bindings still reference `D:\WORK\Software Development\GitHub\HERDR\...`
which does not exist on this PC (LESSONS #63).

## 2. Context
- Folder `I:\herdr`; read `CLAUDE.md`, `AGENTS.md`, LESSONS #39 (every command is
  `conhost.exe --headless pwsh -NoProfile ...`, nothing may flash a console) and #63 first.
- Scripts live in `I:\herdr`: `herdr-open-folder.ps1`, `herdr-layout.ps1`, `herdr-help.ps1`,
  `herdr-stop.ps1`, `herdr-spaces.ps1`, `herdr-columns.ps1`, `herdr-hud.ps1`. Master copy of the
  bindings: `config/keys.toml`. Skills docs also mention the old D: path: check `skills/`, `docs/`,
  `launcher/`, `Herdr.cmd`, `install*.ps1`, `sync-skills.ps1`, `herdr-agents*` for `D:\WORK`.

## 3. Scope
- Change: `config/keys.toml`, `config/settings.toml` if it has D: paths, `%APPDATA%\herdr\config.toml`
  (back it up to `%APPDATA%\herdr\config.toml.<time>.bak` first), scripts/docs only where they hard-code
  the D: path. Prefer a path computed from the script's own folder over a new hard-coded `I:` path.
- Do NOT touch: `~/.claude/settings.json`, `herdr-app\`, approval/trust settings, the running Herdr
  session's panes (do not close or stop anything). No git. Test only in an isolated session
  (`herdr --session lab server`, LESSONS #20) or offline; never in the live session.

## 4. Steps
1. List every `D:\WORK` reference in the folder and in the live config.
2. Fix them; keep the invisible-window form `conhost.exe --headless pwsh -NoProfile -File ...`.
3. Confirm each of the ten bindings' target file exists and parses (`pwsh` parser check) and that
   the key table in the live config equals `config/keys.toml`.
4. Run `pwsh -File check.ps1` (0 FAIL) and the relevant offline tests under `tests/`.
5. Add a HISTORY entry and update LESSONS #63.

## 5. Definition of done
- [ ] No `D:\WORK` left in tracked files or the live config (print the grep).
- [ ] Each binding: key, target path, `Test-Path` result printed.
- [ ] `check.ps1` 0 FAIL; offline tests still pass.
- [ ] Not verified live: say so; the lead tests one key.

## 6. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Checks: <each check and result>
Not done / doubts: <list or none>
