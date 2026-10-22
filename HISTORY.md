# History

Every change to this setup, newest last. Shape: **Symptom / Cause / Fix / Lesson**.
Lesson numbers (#n) point into [LESSONS.md](LESSONS.md).

---

## Phase 1 - Evaluate Herdr (2026-10-22)

**Question:** can the owner use herdrdev/herdr personally, next to the G-MES project?
**Finding:** Herdr is a Rust terminal workspace for coding agents (Apache-2.0,
~45k stars, young: v0.x, ~390 open issues). Windows support is native (ConPTY) but
labelled beta; the binary is **not signed**.
**Decision:** personal use only - never inside the G-MES repository, never on its server.
**Lesson:** #11.

## Phase 2 - Install, reviewed first (2026-10-22)

**Fix:** downloaded `install.cmd` and `install.ps1` into this folder and read them
completely before running. They write only to the user profile
(`%USERPROFILE%\.herdr\packages\standalone\releases\...`, junctions
`...\current` and `%LOCALAPPDATA%\Programs\Herdr\bin`) and to the user PATH (HKCU);
they verify the download's SHA-256 from the herdr.dev manifest. Stable `latest.json`
already carried Windows (v0.9.5). Ran `install.cmd -Channel stable` -> Herdr 0.9.5.
`Get-AuthenticodeSignature` -> NotSigned.
**Lesson:** #11.

## Phase 3 - Claude/Codex integrations blocked (2026-10-22)

**Symptom:** `herdr integration install claude` was refused by Claude Code's auto-mode
classifier ("Self-Modification"), and after that even a read-only `herdr --default-config`.
**Cause:** the Claude integration edits `~/.claude/settings.json` (adds one
`SessionStart` hook, verified by reading `src/integration/claude_settings.rs`);
changing Claude's own configuration needs the owner.
**Fix:** none by the agent; the owner runs
`! & "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe" integration install claude`.
**Lesson:** #3.

## Phase 4 - One-click start (2026-10-22)

**Fix:** `Herdr.cmd` launcher + Desktop and Start-menu shortcuts that start Windows
Terminal directly (`wt.exe -d <dir> --title Herdr <herdr.exe>`), so no console flashes.
They use the stable alias `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe`, so
`herdr update` does not break them. Verified: shortcut started a herdr process.

## Phase 5 - `herdr` in File Explorer's address bar (2026-10-22)

**Symptom:** the owner wanted to type `herdr` in Explorer's address bar.
**Cause:** Explorer keeps the PATH it started with; the new PATH entry from the
installer is invisible to it until sign-out.
**Fix:** registered `HKCU\Software\Microsoft\Windows\CurrentVersion\App Paths\herdr.exe`
-> `Herdr.cmd`, which is read live. `Herdr.cmd` now opens in the current folder
(`%CD%`), falling back to the GitHub folder when started from System32 (Win+R).
Verified with `Start-Process herdr -WorkingDirectory <folder>`.
**Lesson:** #2.

## Phase 6 - Open a folder from inside Herdr (2026-10-22)

**Symptom:** changing the project folder inside Herdr had no easy way.
**Cause:** Herdr has no folder picker; "new" follows the focused pane's folder.
**Fix:** `herdr-open-folder.ps1` (Windows FolderBrowserDialog on a hidden top-most
owner, remembers the last folder in `%APPDATA%\herdr\open-folder-last.txt`, then
`herdr workspace create --cwd --label --focus`), bound with `[[keys.command]]`
type `shell` to `prefix+f` and `prefix+shift+f` (+ Claude via `herdr pane run`).
`prefix+o` was taken (open notification). Config reloaded with
`herdr server reload-config` -> `status: applied`, no diagnostics.
**Lessons:** #4, #5.

## Phase 7 - Claude sees and commands Codex (2026-10-22)

**Fix:** global skill `herdr-team`; team mode `prefix+t` (Claude left, Codex right,
Codex named `codex-<workspace>` via `herdr agent start --kind codex`).
**Proof:** `herdr agent prompt codex-w5 "...TEAM-LINK-OK..." --wait` -> Codex answered
`TEAM-LINK-OK`; its footer showed `GPT-6.1-Sol medium` (the model in `~/.codex/config.toml`).

## Phase 8 - Menus inside Herdr (parked) (2026-10-22)

**Symptom:** owner wanted "open folder" in Herdr's menus, not only keys.
**Cause:** Herdr's global and right-click menus are hard-coded in
`src/client/shell/global_menu.rs` and `context_menu.rs`.
**Fix written, not built:** custom commands accept `menu = "label"`; the client reads
the label from its local config and appends it to the global, space, tab and pane
menus (no change to the server<->client wire format, which is bincode). Test
`custom_commands_with_menu_labels_run_from_global_and_context_menus` added.
**Build failed:** no MSVC `link.exe`; the `ghostty-vt` crate also needs Zig 0.16.
**Decision (owner):** do not install the build tools; keep the keys. Code kept on
branch `feature/custom-command-menus` in `herdr/`; `install-custom-build.ps1` would
install such a build (keeps `herdr-official.exe`, `-Restore` puts it back).
**Lessons:** #8, #9, #10.

## Phase 9 - Cline + delegation for every CLI (2026-10-22)

**Fix:**
- Installed Cline CLI 3.0.73 (`npm install -g cline@3.0.73`, package from
  github.com/cline/cline). Herdr already detects Cline (`src/detect/manifests/cline.toml`).
- `agents.json`: profiles (kind, command, model flag, default model, safety args)
  for claude, codex, cline, deepseek (Cline with `-P deepseek`), opencode, grok, agy.
- `herdr-agents.ps1` (+ `herdr-agents.cmd`, folder added to user PATH): tools /
  start / ask / read / status / stop. `stop` closes only panes it started
  (`%APPDATA%\herdr\herdr-agents-panes.json`).
- Cline starts with `--tui --auto-approve false`.
- `herdr-team` skill rewritten around the tool.
**Proof:** in test workspace w8: `start codex` -> STARTED (gpt-6.1-sol); `ask codex
"...DELEGATION-OK + file count..."` -> Codex ran a read-only count and answered
`DELEGATION-OK 10`; `start cline` -> STARTED, detected; `stop` both -> panes closed.
**Symptom found:** the answer was invisible in `ask` output.
**Cause:** `agent read` returns the whole viewport, padded with empty rows.
**Fix:** `Format-Screen` drops blank padding.
**Lessons:** #6, #7, #12, #13.

## Phase 10 - Documentation system (2026-10-22)

**Fix:** README, CLAUDE.md (+ AGENTS.md), HISTORY.md, LESSONS.md, docs/, master
copies of the skills in `skills/` + `sync-skills.ps1`, `check.ps1` (fails when docs,
skills, config keys, agents.json or the installed tools disagree), local git repo.
**Symptom met:** owner's `! herdr integration install claude` -> "herdr is not recognized".
**Cause:** the Claude Code session's shell was started before the installer changed PATH.
**Lesson:** #1.
**State after:** `herdr integration status` -> claude current (v11), codex current (v10),
installed by the owner. `check.ps1` -> all checks passed.
**Proof the check can fail:** with a reference to a lesson number that does not exist, Cline set to
`--auto-approve true` and an undocumented file, `check.ps1` reported 4 FAIL, exit 1;
restored -> all checks passed.

## Phase 11 - Side-by-side panes: 2 and 4 equal columns (2026-10-22)

**Symptom:** the owner wanted 2 panes side by side, and 4 PowerShell panes side by
side in one space.
**Cause:** 2 is built in (`ctrl+b v`), but repeating it halves the focused pane each
time (1/2, 1/4, 1/8), so 4 equal columns needed the right pane each time and a ratio.
**Fix:** `herdr-columns.ps1` (`-Count 2..8`, `-Direction right|down`) splits the
focused pane (`herdr pane current`) with ratios 1/N, 1/(N-1) ... 1/2, every new
pane in the same folder, focus unchanged. Bound to `prefix+shift+v` (4 columns) in
`config/keys.toml` and the live config (backed up first); reload -> `applied`, no
diagnostics. The owner had meanwhile switched the theme to `gruvbox` - kept.
**Proof:** test workspace, 254 columns wide -> panes at x=0/64/127/191, widths
64/63/64/63, all four in the same folder; workspace closed.
**Lesson:** #17.
