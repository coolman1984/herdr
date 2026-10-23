# Files

## In this folder

| Path | Purpose |
|---|---|
| `README.md` | Overview and quick start |
| `CLAUDE.md`, `AGENTS.md` | Rules for AI agents working here |
| `HISTORY.md`, `LESSONS.md` | What changed and why; numbered lessons |
| `docs/` | GUIDE (complete user guide), AGENT-ROSTER (which agent for which task), SETUP, DELEGATION, this file |
| `launcher/HerdrLauncher.exe` | One-click start without a console flash (GUI exe): Windows Terminal + Herdr in the current folder + the HUD. App Path and `Herdr.lnk` target (Phase 23). Written by GLM-5.3 Flash |
| `launcher/HerdrLauncher.cs`, `launcher/build.ps1` | Its source (C# 5) and build (.NET Framework `csc.exe`, no SDK); `HerdrLauncher.exe --dry-run [folder]` writes what it would start to `%TEMP%\herdr-launcher-dryrun.txt` |
| `launcher/use-launcher.ps1` | Points App Path + `Herdr.lnk` at the exe, with backups; `-Undo` restores the newest backup |
| `Herdr.cmd` | Fallback launcher (the same, as a batch file - flashes a console for a moment) |
| `herdr-open-folder.ps1` | Folder picker -> new workspace (+ Claude / + team) |
| `herdr-layout.ps1` | Layout builder window (ctrl+b shift+b); `-PlanFile` builds without a window |
| `herdr-help.ps1` | Shortcuts + guide window (ctrl+b i); `-List` prints the shortcuts. Written by Codex (Phase 16) |
| `tasks/` | Task specs for subagents: `TEMPLATE.md` (the spec every agent gets) and past tasks; `HANDOFF-TEMPLATE.md` (lead -> lead handoff, filled by `herdr-agents handoff`) and `handoff-*.md` |
| `claude-limits.ps1` | Claude 5-hour / weekly limits per account (dot-sourced by `herdr-agents` and the HUD; Phase 24) |
| `herdr-stop.ps1` | Stop Herdr for real, after asking (ctrl+b shift+q, Desktop "Stop Herdr"); `-Status` prints what runs |
| `herdr-hud.ps1` | Pane HUD: name, path, model, context, tokens on every pane border + sidebar (runs in the background) |
| `herdr-spaces.ps1` | Spaces side by side and separate again (ctrl+b shift+s) |
| `herdr-columns.ps1` | Split the focused pane into N equal columns or rows (ctrl+b shift+v = 4) |
| `herdr-agents.ps1`, `herdr-agents.cmd` | Delegation tool |
| `agents.json` | Agent profiles for the delegation tool |
| `config/keys.toml` | Master copy of the key bindings added to Herdr's config |
| `config/settings.toml` | Master copy of other settings added there (Windows notifications) |
| `skills/herdr-team/`, `skills/herdr-workspace/` | Master copies of the Claude skills |
| `sync-skills.ps1` | Copies `skills/` to `~/.claude/skills/` (`-Check` reports drift only) |
| `check.ps1` | Verifies the whole setup |
| `tests/test-batch-a.ps1` | Offline regression tests for delegation, Claude limits and setup checks; `-SourceRoot` tests baseline script copies |
| `install.cmd`, `install.ps1` | Official installers as reviewed (Phase 2) |
| `install-custom-build.ps1` | Installs a self-built Herdr (parked, Phase 8) |
| `herdr-skill.md` | Herdr's own agent skill (`herdr --skill`), for reference |
| `config.toml.backup` | Herdr config before our key bindings |
| `herdr/` | Upstream source clone (own git; branch `feature/custom-command-menus`) - not tracked here |

## Outside this folder (changed by this setup)

| Location | Change |
|---|---|
| `%USERPROFILE%\.herdr\`, `%LOCALAPPDATA%\Programs\Herdr\bin` | Herdr install (installer) |
| HKCU `Environment\Path` | Herdr release dir (installer); this folder (`herdr-agents`) |
| HKCU `...\App Paths\herdr.exe` | -> `launcher\HerdrLauncher.exe` (was `Herdr.cmd`; backup `%APPDATA%\herdr\backup\launcher-<time>\`) |
| `%APPDATA%\herdr\config.toml` | key bindings (`config/keys.toml`), settings (`config/settings.toml`); backups `config.toml.<time>.bak` |
| `%APPDATA%\herdr\agent-runs\` | every `ask` / `fanout` task and answer (Markdown, per day) |
| `%APPDATA%\herdr\layouts\` | layout presets saved in the layout builder |
| `%APPDATA%\herdr\side-by-side.json` | which panes "Separate again" sends back to which space (removed after separating) |
| `~\.herdr\worktrees\<repo>\<branch>` | worktrees made by `herdr-agents start -Worktree` (removed by the owner after merging) |
| `%APPDATA%\herdr\open-folder-last.txt` | last folder picked |
| `%APPDATA%\herdr\herdr-agents-panes.json` | panes started by `herdr-agents` |
| `%APPDATA%\herdr\layout-last.json` | last layout chosen in the layout builder |
| Desktop + Start menu `Herdr.lnk` | one-click start -> `launcher\HerdrLauncher.exe` (backup as above) |
| Desktop `Stop Herdr.lnk` | runs `herdr-stop.ps1` (asks first) |
| `~/.claude/statusline.js` | owner-approved addition `shareCostWithHerdr`: writes `%APPDATA%\herdr\claude-cost\<session>.json` (backup `statusline.js.<time>.bak`) |
| `%USERPROFILE%\.herdr-accounts\<tool>\<name>` | separate sign-in folders made by `herdr-agents -Account` / `login` |
| `D:\WORK\Software Development\GitHub\HERDR-trials\` | test projects for trying agents (Phase 22 GLM-5.3 Flash trial, hidden tests) |
| `%APPDATA%\herdr\claude-cost\` | Claude's cost and rate limits per session (`config_dir` = account), read by the HUD and `herdr-agents usage` |
| Startup folder `Herdr HUD.lnk` | starts `herdr-hud.ps1` hidden at sign-in (one copy; the launcher starts it too) |
| `~/.claude/skills/herdr-team`, `herdr-workspace` | Claude skills (from `skills/`) |
| npm global `cline@3.0.73` | Cline CLI |
| `~/.claude/settings.json` | only if the owner installs the Claude integration (one SessionStart hook) |
