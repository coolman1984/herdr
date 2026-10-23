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
4. **Key bindings** - back up `%APPDATA%\herdr\config.toml`, then add all ten `[[keys.command]]` blocks from
   [`../config/keys.toml`](../config/keys.toml) to `%APPDATA%\herdr\config.toml`, then
   `herdr server reload-config`.
5. **Skills** - `pwsh -File sync-skills.ps1` copies `skills/*` to `~/.claude/skills/`.
6. **One-click start** - run `pwsh -NoProfile -File launcher/build.ps1` to build the
   windowless `launcher\HerdrLauncher.exe`. On a fresh PC create Desktop and Start-menu
   `Herdr.lnk` shortcuts targeting the full path of this exe (no arguments; working
   folder `D:\WORK\Software Development\GitHub`). If they already exist, leave them
   for step 7 to back up and update. `Herdr.cmd` is the fallback; it flashes a console.
7. **`herdr` in Explorer** - on a fresh PC create the registry key
   `HKCU\Software\Microsoft\Windows\CurrentVersion\App Paths\herdr.exe` and set its
   default value to the full path of `launcher\HerdrLauncher.exe`. Leave an existing
   registration intact, then run `pwsh -NoProfile -File launcher/use-launcher.ps1`.
   It needs the key to exist and updates existing `Herdr.lnk` shortcuts; it does not
   create missing ones. It backs up the registry and shortcuts under
   `%APPDATA%\herdr\backup\launcher-<time>\`; `-Undo` restores them. Read its printed
   targets back to check the result.
8. **HUD and Stop shortcuts** - back up any existing links before replacing them.
   Create `Herdr HUD.lnk` in the Startup folder (Win+R `shell:startup`), targeting
   `%SystemRoot%\System32\conhost.exe`, arguments
   `--headless pwsh -NoProfile -File "<this folder>\herdr-hud.ps1"`.
   Create Desktop `Stop Herdr.lnk` with the same target, arguments
   `--headless pwsh -NoProfile -STA -File "<this folder>\herdr-stop.ps1"`.
   The launcher also starts the HUD; its mutex keeps one copy. Use PowerShell 7 and
   the headless host for all key commands and these shortcuts (LESSONS #39).
9. **`herdr-agents` command** - back up the raw user PATH and its registry kind,
   then add this folder to the user PATH.
10. **Cline Pass** - the owner runs `cline auth cline-pass`: `cline` =
    `cline-pass/glm-5.3`, `cline-flash` = `cline-pass/glm-5.3-flash`, `deepseek` =
    `cline-pass/deepseek-v4-pro`, `kimi` = `cline-pass/kimi-k3`. Direct providers are
    optional: sign in yourself (e.g. `cline auth deepseek`), then use
    `herdr-agents start cline -Provider deepseek -Model <direct-model-id>`.
11. `pwsh -File check.ps1` - everything should be OK.

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
