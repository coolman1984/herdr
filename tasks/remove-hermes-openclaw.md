# Task: completely remove Hermes Agent and OpenClaw/Clawdbot leftovers from this PC

## 1. Role and goal
You are a careful Windows/PowerShell engineer working for the lead (Claude).
Goal: nothing of Hermes Agent, OpenClaw or Clawdbot stays on this PC, and the owner can see proof.
Why: the owner asked for a COMPLETE delete (2026-10-24). They know the data goes; do not ask again.

## 2. Context you need
- Folder: `I:\herdr`; read `CLAUDE.md` and `AGENTS.md` first (delegate rules).
- The lead already did: unregistered scheduled tasks `BirdCLI-Twitter-To-Obsidian`, `...-DryRun`,
  `OpenClaw Gateway`, `OpenClawNightly`; deleted the `OpenClaw Gateway.cmd` startup file; started
  `npm uninstall -g openclaw`; backup of nothing else is needed.
- ACCESS DENIED for the lead (needs an administrator): scheduled tasks `Clawdbot Gateway`,
  `Hermes_Gateway`, and Windows service `clawdbot`. Try them; if denied, do not retry, list the
  exact commands for the owner under "Not done".

## 3. Exact scope - delete these (look at each target first, print its size, then delete)
- `C:\Users\mflma\.clawdbot`, `C:\Users\mflma\.hermes`, `C:\Users\mflma\Hermes-Mastery-Database`,
  `C:\Users\mflma\hermes-youtube-stack`, `C:\Users\mflma\AppData\Roaming\Hermes`,
  `C:\Users\mflma\AppData\Local\com.nousresearch.hermes.setup`, `C:\Users\mflma\AppData\Local\hermes`
  (~10 GB, 233k files: use `cmd /c rd /s /q`, not Remove-Item), `G:\Hermes Secrets`,
  `G:\hermes-agent-self-evolution`.
- Leftover `openclaw`, `openclaw.cmd`, `openclaw.ps1` in `C:\Users\mflma\scoop\persist\nodejs-lts\bin`
  and `...\node_modules\openclaw` if the npm uninstall left them.
- User env vars `HERMES_HOME`, `HERMES_GIT_BASH_PATH`; the `...\hermes\bin` entry in the user PATH
  (save the old PATH value to `I:\herdr\tasks\path-backup-2026-10-24.txt` first, then change only
  that one entry).
- `C:\Users\mflma\.codex\skills\hermes-agent-operations` if it still exists.
- Do NOT touch: anything under `G:\Real Accounting ERP System` (only the service `clawdbot` that
  points into it is ours), `~\.claude.json`, `~\.claude\settings.json`, `herdr-app\` (upstream
  source that merely mentions Hermes), other Herdr panes. No git. Do not close any pane.

## 4. Design
- Stop any process whose path is inside a target first (`Get-Process | ? Path -like ...`).
- One target at a time; print `OK` / `FAIL <reason>` per target; never a broad silent catch.
- If your sandbox blocks a delete outside `I:\herdr`, do NOT try to bypass it: report BLOCKED with
  the exact `pwsh` script (saved as `I:\herdr\tasks\remove-hermes-openclaw.ps1`, dry-run switch
  `-WhatIf`) for the lead to run.

## 5. Steps
1. Re-scan for any other trace: folders, tasks, services, Run keys, env vars, PATH, npm/pip/uv/scoop
   packages, shims, desktop/Start-menu shortcuts, on every drive root (C:, G:, H:, I:, E:) and the
   home folder. Names: `hermes`, `openclaw`, `clawdbot`, `moltbot`, `nousresearch`.
2. Delete as listed in section 3.
3. Re-scan and prove nothing is left.

## 6. Definition of done
- [ ] Re-scan prints nothing for those names (except the allowed `herdr-app\` and `.claude.json`).
- [ ] `Get-Command openclaw, hermes` finds nothing; env vars and PATH entry gone.
- [ ] Free space on C: and G: before and after is printed.

## 7. Answer format
DONE | BLOCKED | PARTIAL
Changed: <what was removed, with sizes>
Checks: <each check in section 6 and its result>
Not done / doubts: <owner admin commands, anything left>
