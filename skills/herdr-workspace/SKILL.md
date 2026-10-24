---
name: herdr-workspace
description: "Operate, fix, change or document the owner's personal Herdr setup in I:\\herdr - Herdr install/update, the one-click launcher, the 'herdr' Explorer command, the folder-picker keys (ctrl+b f / shift+f / t), the herdr-agents delegation tool and agents.json, the Claude skills, Cline/other agent CLIs and their models, and the folder's docs (HISTORY, LESSONS). Use when the user asks to change, repair, extend, check or explain this Herdr setup, add a new agent CLI or model, or when Herdr misbehaves."
---

# Herdr workspace: maintain the owner's setup

Folder: `I:\herdr`. Herdr itself is the official
build; everything else there is ours.

## Before any change

1. Read `CLAUDE.md` in the folder completely (rules), then the relevant phase of
   `HISTORY.md` and `LESSONS.md` (numbered lessons).
2. Run `pwsh -NoProfile -File check.ps1` to see the current state.

## Doing the change

- Herdr CLI: always `& "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe" ...` (LESSONS #1).
- Key bindings: edit `config/keys.toml` AND `%APPDATA%\herdr\config.toml` (back it up
  first), then `herdr server reload-config` - read `diagnostics`, it must be empty.
- New agent CLI: add a profile to `agents.json` (kind from LESSONS #14, model flag
  from its `--help`, safety args so tools need approval), `herdr-agents tools`, then
  test `start` / `ask` / `stop` in a test workspace and close it.
- Skills: edit `skills/<name>/SKILL.md`, then `pwsh -NoProfile -File sync-skills.ps1`.
- Tests: use the isolated session from CLAUDE.md section 3 (LESSONS #20, #24).
  Call the absolute executable path and start the lab server headless:
  ```powershell
  $herdr = "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe"
  $oldSocket = $env:HERDR_SOCKET_PATH
  $serverCommand = '"& ''{0}'' --session lab server"' -f $herdr.Replace("'", "''")
  Start-Process "$env:SystemRoot\System32\conhost.exe" -WindowStyle Hidden -ArgumentList @(
      '--headless', 'pwsh', '-NoProfile', '-Command', $serverCommand)
  try {
      $env:HERDR_SOCKET_PATH = "$env:APPDATA\herdr\sessions\lab\herdr.sock"
      # Wait for status: running; create a test workspace with --no-focus.
      # Read its pane id from JSON; set HERDR_ENV=1 and HERDR_PANE_ID in the test process.
      # Run tests here without attaching a terminal window.
  } finally {
      & $herdr --session lab server stop
      $env:HERDR_SOCKET_PATH = $oldSocket
  }
  ```
  Use a test workspace in the live session only for what the lab cannot show.
  Never split or close the owner's own panes.

## After the change

1. HISTORY.md entry (Symptom / Cause / Fix / Lesson, observed behaviour, proof).
2. New durable fact -> LESSONS.md (next number, never renumber).
3. Docs that describe the changed thing (README, docs/SETUP.md, docs/DELEGATION.md,
   docs/FILES.md).
4. `pwsh -NoProfile -File check.ps1` passes.
5. `git add -A; git commit` in the folder (local repo, explain why in the message).

## Never

Handle API keys or passwords; approve an agent's tool request for the owner; run
`herdr integration install ...` or edit `~/.claude/settings.json` yourself (give the
owner the `!` command); stop the owner's server; edit files under `herdr/` (upstream clone)
for a quick fix.
