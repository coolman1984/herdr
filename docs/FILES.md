# Files

## In this folder

| Path | Purpose |
|---|---|
| `README.md` | Overview and quick start |
| `CLAUDE.md`, `AGENTS.md` | Rules for AI agents working here |
| `HISTORY.md`, `LESSONS.md` | What changed and why; numbered lessons |
| `docs/` | GUIDE (complete user guide), SETUP, DELEGATION, this file |
| `Herdr.cmd` | Launcher: Windows Terminal + Herdr in the current folder (App Path target) |
| `herdr-open-folder.ps1` | Folder picker -> new workspace (+ Claude / + team) |
| `herdr-layout.ps1` | Layout builder window (ctrl+b shift+b); `-PlanFile` builds without a window |
| `herdr-columns.ps1` | Split the focused pane into N equal columns or rows (ctrl+b shift+v = 4) |
| `herdr-agents.ps1`, `herdr-agents.cmd` | Delegation tool |
| `agents.json` | Agent profiles for the delegation tool |
| `config/keys.toml` | Master copy of the key bindings added to Herdr's config |
| `config/settings.toml` | Master copy of other settings added there (Windows notifications) |
| `skills/herdr-team/`, `skills/herdr-workspace/` | Master copies of the Claude skills |
| `sync-skills.ps1` | Copies `skills/` to `~/.claude/skills/` (`-Check` reports drift only) |
| `check.ps1` | Verifies the whole setup |
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
| HKCU `...\App Paths\herdr.exe` | -> `Herdr.cmd` |
| `%APPDATA%\herdr\config.toml` | key bindings (`config/keys.toml`), settings (`config/settings.toml`); backups `config.toml.<time>.bak` |
| `%APPDATA%\herdr\agent-runs\` | every `ask` / `fanout` task and answer (Markdown, per day) |
| `%APPDATA%\herdr\layouts\` | layout presets saved in the layout builder |
| `~\.herdr\worktrees\<repo>\<branch>` | worktrees made by `herdr-agents start -Worktree` (removed by the owner after merging) |
| `%APPDATA%\herdr\open-folder-last.txt` | last folder picked |
| `%APPDATA%\herdr\herdr-agents-panes.json` | panes started by `herdr-agents` |
| `%APPDATA%\herdr\layout-last.json` | last layout chosen in the layout builder |
| Desktop + Start menu `Herdr.lnk` | one-click start |
| `~/.claude/skills/herdr-team`, `herdr-workspace` | Claude skills (from `skills/`) |
| npm global `cline@3.0.73` | Cline CLI |
| `~/.claude/settings.json` | only if the owner installs the Claude integration (one SessionStart hook) |
