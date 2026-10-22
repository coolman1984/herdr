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

## Phase 12 - Layout builder: a folder and an agent per column (2026-10-22)

**Symptom:** the owner wanted to choose each column's project folder when opening
side-by-side panes, and to put agents that are already running into those columns.
**Cause:** `ctrl+b shift+v` (Phase 11) gives every column the focused pane's folder
and empty shells only.
**Fix:** `herdr-layout.ps1` (`ctrl+b shift+b`): a window with 1-6 columns, "Open in"
(new tab here / new space), and per column a folder (Browse...), what runs (PowerShell,
`New: <profile>` from agents.json with optional model, or `Running: <agent>`) and the
model. Equal widths with ratios 1/N..1/2 (LESSONS #17); new agents via
`herdr-agents start <profile> -Pane <id>` (new `-Pane` option, same safety args);
running agents moved next to a placeholder that is then closed (LESSONS #18), and
the herdr-agents pane record follows the new id. Last choices remembered in
`%APPDATA%\herdr\layout-last.json`. `-PlanFile` builds the same without a window
(for agents and tests; errors print instead of message boxes).
**Proof:** source space with Codex `codexsrc` in `docs`; plan = shell in HERDR, new
codex in `skills`, running `codexsrc` -> new space: panes x=0/85/170, widths 85/85/84,
folders HERDR / skills / docs; `codex` started (gpt-6.1-sol), `codexsrc` moved and
still alive under its name, record updated to the new pane; the source space closed
itself. Window opened and was screenshot-checked (4 rows, defaults = focused pane's
folder). Test agents stopped, test space closed.
**Lesson:** #18.

## Phase 13 - Research: features from other agent managers (2026-10-22)

**Symptom:** the owner asked for web research on more useful features, a plan, and the build.
**Research:** agent-deck (status board, waiting/blocked alerts, worktree per session, cost,
conductors), claude-squad / ccmanager / Parallel Code / Vibe Kanban (worktree per agent,
diff review before merge), agentmaxxing / Code Agent Orchestra articles (parallel second
opinions, review is the bottleneck, keep a record of tasks). Herdr 0.9.5 is the latest
release; it already has Windows notifications (off by default) and a `worktree` command.
**Plan and build:**
1. Windows notifications: `[ui.toast] delivery = "system"` (new `config/settings.toml`,
   checked by `check.ps1`) - LESSONS #22.
2. `herdr-agents start <agent> -Worktree <branch>`: own branch + folder
   (`~\.herdr\worktrees\<repo>\<branch>`) in a new space; `stop` keeps the worktree and
   prints how to remove it after merging. Plain git, not `herdr worktree` (LESSONS #19).
3. `herdr-agents diff <name>`: read-only review - status, commits since the base branch,
   files changed.
4. `herdr-agents fanout "<task>" -Agents a,b`: the same task to several agents in
   parallel (thread jobs), one combined report.
5. Every `ask` / `fanout` saved as Markdown in `%APPDATA%\herdr\agent-runs\<date>\`;
   `herdr-agents history [<n>]` lists or shows them.
6. `herdr-agents watch` - live board, blocked first, coloured; `ctrl+b a` opens it in a pane
   below (`-OpenPane`).
7. Layout presets: "Preset" list + "Save preset..." in the layout builder
   (`%APPDATA%\herdr\layouts\<name>.json`), `herdr-layout.ps1 -Preset <name>` without a window.
Not taken: cost tracking, Telegram/Slack conductors, Docker sandboxes, auto-merge (outward-
facing or destructive; would need the owner's decision).
**Incident:** the first probe, `herdr worktree create` on the owner's LIVE session, crashed
the Herdr server (0xc0000409) - the owner's window closed and the agents in it stopped. The
saved layout (`session.json`) comes back when Herdr is opened. Every later test ran in an
isolated session (LESSONS #20); CLAUDE.md now requires that.
**Bugs found by the tests:** a local `$lines` overwrote the `-Lines` parameter (PowerShell
names ignore case) -> "invalid value for --lines"; renamed. One agent of a fan-out returned
`agent_prompt_stalled` although it had answered -> now waits and shows the answer marked
UNCONFIRMED (LESSONS #21).
**Proof (isolated session):** worktree agent started on `feat-x` in a new space; Codex then
showed its trust question and `watch -Once` listed it BLOCKED in red (LESSONS #23; not
answered - owner's decision). Fan-out to two Codex agents: both DONE with "FAN2-OK 72",
report saved; `history` listed both runs, `history 1` printed the report. `diff` on a
worktree agent: 1 commit since main, 2 files changed, uncommitted `a.txt` shown. Preset of
3 folders -> 3 equal columns (31/32/31) in HERDR / docs / skills; unknown preset -> error
listing the saved ones. Window screenshot: Preset list and Save preset... button shown.
`herdr config check` -> ok after adding the key and the setting (config backed up first).
Test session stopped, test worktrees, preset and session folder removed.
**Lesson:** #19, #20, #21, #22, #23.

## Phase 14 - Spaces side by side (2026-10-22)

**Symptom:** the owner wanted the spaces in the sidebar (`Digital- Factory`, `MR.Park`,
`opening-nerp-tcode`) shown side by side at the same time.
**Cause:** Herdr shows one space at a time; the layout builder only moves agents, one
pane each, into a new layout.
**Fix:** `herdr-spaces.ps1` (`ctrl+b shift+s`): a window lists every space (all ticked);
**Side by side** opens a new space `side by side` with N equal placeholder columns, moves
each space's first active-tab pane into its column and closes the placeholder
(LESSONS #18), then stacks that space's other panes inside its own column (down-ratios
1/2, 2/3, 3/4 ... = equal rows). Emptied spaces close. `%APPDATA%\herdr\side-by-side.json`
records the groups; **Separate again** moves each group back with
`pane move --new-workspace --label <old name>` (LESSONS #24). herdr-agents' pane record
follows every move. `-Spaces "a,b"` / `-Separate` do the same without a window.
A first version used the first space as the host and split its rightmost pane; a space
with 2 panes then got its second pane stacked across two columns - replaced by the
placeholder method.
**Also fixed:** the UNCONFIRMED fallback of Phase 13 returned while Codex was still working
(Herdr said idle, `agent wait` returned at once) - it now polls `agent get` up to 30 s for
`working` before waiting (LESSONS #21, updated).
**Proof (isolated session, window attached):** spaces Digital- Factory (HERDR), MR.Park (2
panes), opening-nerp-tcode -> `side by side`: columns x=0/40/80, width 40 each, MR.Park's two
panes at y=0 and y=20 inside column 2, folders right; the three spaces closed. Separate ->
the three spaces back under their names with 1 / 2 / 1 panes and the right folders,
`side by side` closed, record removed. With Codex `sbs` running in a space: joined and
separated, still alive under its name, pane record updated each time, `stop sbs` closed the
right pane. Window screenshot checked. Key added to `config/keys.toml` and the live config
(backed up), `config check` ok, reload `applied`. Test session removed.
**Lesson:** #21 (updated), #24.

## Phase 15 - Each agent keeps its name; pane HUD with path, model and usage (2026-10-22)

**Symptom:** after "side by side" every agent in the sidebar was called `side by side`; the
owner asked what the red `main` and the arrows mean, and wanted every pane to show its
project path, model and consumption the way Claude's and Codex's own status lines do.
**Cause:** the agent panel's default first row is the SPACE name (LESSONS #25); the branch
colour is gruvbox's; panes had no title. Herdr already supports metadata titles on pane
borders and custom sidebar tokens (LESSONS #25, #26).
**Fix:**
- `herdr-spaces.ps1` names every moved pane after the space it came from (`pane rename`),
  unless the pane already has its own name.
- `herdr-hud.ps1` (background, one copy, started at sign-in by a Startup shortcut and by
  `Herdr.cmd`): every 5 s sets each pane's border title to
  `name · path · model · ctx N% (used/window) · out N` and the tokens `$name $model $ctx $path
  $out` (TTL 2 min). Usage comes from the CLIs' session logs (LESSONS #28), read
  incrementally; Claude's window from the new agents.json `contextWindow` (1M, as its own
  status line shows).
- `config/settings.toml`: agent rows `state + $name + space / agent + $model / $ctx`; space
  rows with the branch in a calm colour. Applied to the live config (backed up), `config
  check` ok, reload `applied`.
**Bugs found by the tests:** (1) the log reader returned its two results as one array -> no
usage shown; fixed. (2) names were read from `pane layout`, which has no labels -> every
title said `side by side`; read from `pane list` (LESSONS #27).
**Note:** one `herdr-hud.ps1 -Once` dry run went to the owner's live session before the
isolated test (it set border titles that expire after 2 minutes; nothing else).
**Proof (isolated session with a test config):** 3 spaces + a Codex agent -> side by side ->
titles `Digital- Factory · D:\...\GitHub\HERDR · gpt-6.1-sol medium · ctx 5% (22.7k/475k) ·
out 8`, `MR.Park · ...`, `opening-nerp-tcode · ...`; labels and tokens confirmed by
`pane get`; after Separate the names stayed. Claude reader on this session's log:
`claude-opus-5-5 · ctx 30% (303.8k/1M) · out 245.1k`, 0.9 s first pass, 0.005 s after.
Live: HUD running, the owner's Claude pane titled `HERDR · D:\...\opening-nerp-tcode ·
claude-opus-5-5 · ctx 33% (327k/1M)`. A screenshot of the test window was not possible
(Windows kept the owner's window in front).
**Lesson:** #25, #26, #27, #28.

## Phase 16 - Shortcuts window by a Codex subagent, Claude's cost, team routine (2026-10-22)

**Symptom:** the owner wanted (1) Claude's dollar cost on the pane border, (2) a key that
pops up every shortcut, (3) a guide section explaining how to use every feature, (4) Codex
SOL 6.1 medium as a visible subagent writing code while Claude reads and controls it, and
then (5) this as the standing routine for any team of subagents, with Claude as the lead
senior engineer who plans everything and gives agents complete specs.
**Fix:**
1. Cost: the owner approved changing `~\.claude\statusline.js` (backed up): new
   `shareCostWithHerdr` writes `%APPDATA%\herdr\claude-cost\<session>.json`; the status line
   output is unchanged. `herdr-hud.ps1` adds `$cost` for Claude panes (LESSONS #30).
2. `herdr-help.ps1` (`ctrl+b i`): Shortcuts tab (filter + list from `config/keys.toml`, Herdr's
   keys, commands) and Guide tab (`docs/GUIDE.md` rendered), Open in browser, `-List`.
   **Written by Codex** in a pane next to Claude from the spec `tasks/help-window.md`.
3. `docs/GUIDE.md` section 13: step by step for every feature (18 parts).
4. `herdr-agents start -Effort <level>` (agents.json `effortArgs`, Codex) - LESSONS #29.
5. `herdr-agents team -PlanFile` (lead keeps 40% left, agents in equal named columns, each
   started and given its task) and `collect -Agents` (waits for all, one report).
6. Routine in the `herdr-team` skill section 1b (lead engineer: research, plan, design,
   specs from `tasks/TEMPLATE.md`, monitor screens, correct, review, collect, short report)
   and in Claude's memory.
**How Codex was led:** started at `-Effort medium` (process argv confirmed
`model_reasoning_effort=medium`; footer showed `default` until its first turn, then
`GPT-6.1-Sol medium`). Claude read its screen during the work: it found and fixed its own
ListViewGroup bug and reported DONE (37 rows, 0 parse errors). Claude's review found one
defect - "Open guide in browser" depended on a Chrome-launcher Python script from Codex's
own skills (LESSONS #31) - and sent it back: fixed in 40 s (`Start-Process` on the file,
error boxes owned by the window). Claude then fixed the descriptions (they repeated the
key) itself.
**Proof:** `-List` 38 rows (8 keys of ours + 27 Herdr + 3 commands), descriptions clean;
window screenshot: tabs, filter, 38 rows, buttons. Status line test: same output, cost file
written. Live border: `HERDR · ... · claude-opus-5-5 · $10.18 · ctx ...`; Codex pane
`gpt-6.1-sol medium · ctx 14% (67.9k/475k)`. Isolated session: `team` with 2 Codex agents ->
lead 48 / agents 36 + 36 columns, panes named `ta` / `tb`, both answered (`TA-OK`, effort
`low` shown); `collect` -> `ta=DONE tb=DONE`, report saved; an agent without `cwd` starts in
the lead's folder. Key added to keys.toml + live config (backed up), `config check` ok,
reload `applied`. Test session removed.
**Lesson:** #29, #30, #31.

## Phase 17 - Team pages for more than 3 agents; agent roster of this PC (2026-10-22)

**Symptom:** (1) the owner asked that with more than 3 subagents the extra agents open on a
new page, 4 columns per page, and so on; (2) the owner asked for a scan of every agent on the
PC and research on what each is good at, its cost and thinking levels, so tasks go to the
right agent.
**Fix:**
1. `herdr-agents team`: page 1 = the lead + up to 3 agents; then 4 agents per page in new tabs
   of the lead's space ("team 2", "team 3" ...), or new spaces with `-Overflow space`; up to
   16 agents.
2. `docs/AGENT-ROSTER.md`: what is installed, whether it answers now, models, thinking
   levels, strengths / weaknesses, cost, routing rules, edge cases, how to re-scan (web
   research cited in the file). Skill section 1b chooses agents from it.
3. `agents.json`: new profiles `gemini` (Gemini 4 Argon medium) and `gemini-flash` (Gemini
   3.8 Flash) through Antigravity, both in read-only `--mode plan` (LESSONS #32);
   `effortArgs` for Cline (`--thinking`) and Antigravity (`--effort`).
**Scan result:** installed = Claude Code 2.1.308, Codex 0.163.0, Antigravity 1.4.0, Cline
3.0.73, OpenCode 1.18.11, Grok 0.2.106 (+ Windsurf editor, LM Studio without models, GitHub
CLI not signed in). Working with one tiny prompt: Claude, Codex, Antigravity (`AGY-OK`, 19 s).
Not working: OpenCode (OpenAI 401, Zen error page), Cline (re-authentication; DeepSeek endpoint
returned HTML = network block), Grok (not signed in) - LESSONS #33.
**Note:** Codex `coder`, asked by the owner in its own pane, committed the pending work as
`cd9779c` and is waiting for the owner to choose where to push; Claude did not answer for
the owner.
**Proof (isolated session):** 5 agents -> tab 1 `lead | a1 | a2 | a3`, tab `team 2`
`a4 | a5`; 4 agents with `-Overflow space` -> space `lead2 team 2` with `b4`. Profile `gemini`
started as "Gemini 4 Argon · medium" and stopped at Antigravity's trust question (owner's
answer). Test session removed.
**Lesson:** #32, #33.
