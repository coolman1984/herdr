# Setup, update, restore, troubleshooting

## Install on a PC (or rebuild this one)

1. **Herdr** - read the installer first, then run it (user scope, no admin):
   ```
   curl.exe -fsSLo install.cmd https://herdr.dev/install.cmd
   install.cmd -Channel stable
   ```
   Check: `& "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe" --version`.
2. **Agent CLIs** (only those you use): Claude Code, Codex (`npm i -g @openai/codex`),
   Cline (`npm i -g cline`), OpenCode, Grok, Antigravity. Sign in to each in its
   own terminal - never through an agent.
3. **Integrations** (session restore after a restart) - the owner types:
   ```
   ! & "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe" integration install claude
   ! & "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe" integration install codex
   ```
4. **Key bindings** - add the three `[[keys.command]]` blocks from
   [`../config/keys.toml`](../config/keys.toml) to `%APPDATA%\herdr\config.toml`, then
   `herdr server reload-config`.
5. **Skills** - `pwsh -File sync-skills.ps1` copies `skills/*` to `~/.claude/skills/`.
6. **One-click start** - Desktop/Start-menu shortcut `Herdr.lnk`:
   target `%LOCALAPPDATA%\Microsoft\WindowsApps\wt.exe`, arguments
   `-d "D:\WORK\Software Development\GitHub" --title Herdr "%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe"`.
7. **`herdr` in Explorer** - registry value
   `HKCU\Software\Microsoft\Windows\CurrentVersion\App Paths\herdr.exe` (default) =
   full path of `Herdr.cmd`.
8. **`herdr-agents` command** - add this folder to the user PATH.
9. **DeepSeek / other providers in Cline** - `cline auth deepseek` (the owner types the key).
10. `pwsh -File check.ps1` - everything should be OK.

## Update

- `herdr update` (stable channel). Shortcuts, App Path and scripts keep working
  (they use the stable alias). A running server keeps the old binary until
  `herdr server stop` (closes panes) and a new start.
- Cline: `npm i -g cline@<version>`; read its changelog for approval-default changes.
- After any update: `pwsh -File check.ps1`, and add a HISTORY.md entry if behaviour changed.

## Restore / undo

| Undo | How |
|---|---|
| Key bindings | restore `config.toml.backup` (this folder) to `%APPDATA%\herdr\config.toml` |
| Claude integration | `herdr integration uninstall claude` |
| `herdr` in Explorer | delete the `App Paths\herdr.exe` key |
| Skills | delete `~/.claude/skills/herdr-team` and `herdr-workspace` |
| Cline | `npm uninstall -g cline` |
| Herdr | `herdr server stop`, delete `%USERPROFILE%\.herdr` and `%LOCALAPPDATA%\Programs\Herdr`, remove its PATH entry (no official uninstaller) |

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `herdr` is not recognized | old shell (LESSONS #1): new terminal, or use the full path |
| Folder dialog does not appear | it may be behind the terminal; Alt+Tab. Check `pwsh` exists |
| A key does nothing | `herdr server reload-config` and read its diagnostics; key taken (LESSONS #5) |
| `ask` prints BLOCKED | the agent wants approval: answer it yourself in its pane |
| `start` prints NOT READY | the agent needs sign-in / model choice: do it in its pane |
| "config has unknown keys" | a key unknown to this Herdr version (LESSONS #15) |
| Security software blocks herdr.exe | unsigned binary (LESSONS #11): ask IT, do not bypass |
