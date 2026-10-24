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

## Phase 18 - Owner decision: agents run without approval prompts (2026-10-22)

**Symptom:** Antigravity (Gemini 4 Argon High), pushing the toolkit, stopped at a permission
prompt for every git command; the owner asked that all CLI agents work like Claude Code's
auto mode and trust new folders automatically.
**Cause:** CLAUDE.md rule 3 started every agent with approval ON. Claude Code's own safety
classifier refused to let Claude switch other agents to auto-approve ("Create Unsafe
Agents"), and Claude did not route that change through another agent.
**Fix:** the owner ran the change from Claude's prompt (`!` command): `agents.json`
args = Claude `--permission-mode auto`, Codex `-a never -s workspace-write` (no prompts, still
sandboxed to its folder - chosen over Codex's full bypass), Cline `--auto-approve true`,
Antigravity `--mode auto` (`agy`, `gemini`, `gemini-flash`), Grok `--always-approve`; backup
`agents.json.bak` (git-ignored, skipped by check.ps1). Claude then recorded the decision:
CLAUDE.md rule 3 (owner's decision, agents never widen it), `check.ps1` (OK line per agent;
still FAILs on flags that remove the sandbox or skip all permissions), DELEGATION, GUIDE,
AGENT-ROSTER, header of herdr-agents.ps1. The classifier also refused one wording edit in
GUIDE section 12; that line still says "approval ON".
**Folder trust:** no "trust every folder" switch could be confirmed for any of the CLIs; each
remembers its answer per folder/repository (LESSONS #34).
**Lesson:** #34.

## Phase 19 - Antigravity: skip all permissions (owner decision, 2026-10-22)

**Symptom:** with `--mode auto` Antigravity still asked before every shell command (Argon's
git commands during the push).
**Fix:** the owner ran the change from Claude's prompt: `agy`, `gemini`, `gemini-flash` args =
`--dangerously-skip-permissions`. Claude recorded it: CLAUDE.md rule 3, `check.ps1` accepts
that flag for kind `agy` only (it still FAILs it for any other agent), AGENT-ROSTER.
**Lesson:** #35.

## Phase 20 - Closing Herdr never stops work; a real Stop with confirmation (2026-10-22)

**Symptom:** the owner wanted running coding tasks to survive closing Herdr, and a separate,
explicit way to stop everything; and asked how to open the shortcuts window.
**Cause / finding:** already true by design - the Herdr server is a separate process (its
parent process is gone; the window's client is a child of WindowsTerminal.exe), so closing the
window only detaches. Not survivable: a Windows restart or sign-out; then Herdr restores the
layout and resumes Claude/Codex conversations (`resume_agents_on_restore`, default true).
**Fix:** `herdr-stop.ps1` - lists every agent, warns about working/blocked ones, "No" is the
default button, then `herdr server stop`; `-Status` prints what runs. Bound to
`ctrl+b shift+q` (keys.toml + live config, backed up; `config check` ok, reload applied) and a
Desktop shortcut "Stop Herdr". README, GUIDE (keys table, 13.15) updated. The shortcuts
window stays `ctrl+b i` (it lists the new key from keys.toml by itself).
**Proof:** `-Status` listed claude / argon (working) and coder (idle); the dialog screenshot
showed all three and the warning; it was closed unanswered and the server was still running.
**Lesson:** #36.

## Phase 21 - Cline Pass signed in; models researched by Argon, verified live (2026-10-22)

**Symptom:** the owner bought Cline Pass and asked to update Cline and sign in, then asked
Argon (Antigravity, Gemini 4 Argon High) to research Cline Pass and its models so Claude can
recommend daily coding models and how to control Cline.
**Findings:** Cline 3.0.73 is already npm's latest. The owner ran `cline auth cline-pass`
(device-code sign-in). The saved model `stealth/ox-alpha` is retired (404: it was GLM-5.3
Flash). Argon wrote `docs/CLINE-PASS.md` from `tasks/cline-pass-research.md` (12 models,
benchmarks, routing, control). Claude's review: many sources are bare home pages, so benchmark
numbers and reference prices are marked unverified; price $9.99/month and the model families
confirmed by independent pages; live tests: `cline-pass/glm-5.3`, `glm-5.3-flash`,
`deepseek-v4-pro`, `kimi-k3` answer (~60 s each, mostly start-up); bare `glm-5.3` is rejected
(`modelType/model` format required).
**Fix:** agents.json - `cline` = Cline Pass GLM-5.3 (daily), new `cline-flash` (GLM-5.3 Flash),
`deepseek` = Cline Pass DeepSeek V4 Pro (gets around the network block on DeepSeek's own API),
new `kimi` (Kimi K3, hard problems); AGENT-ROSTER updated. Also this phase: Sol (Codex) ran its
`c-drive-safe-cleanup` skill: C: 1.15 -> 3.66 GB free (Temp 1.25 GB, npm cache 1.19 GB, crash
dumps 109 MB, pip 27 MB, Recycle Bin); G-MES credential store, profiles and CDP Profile
verified present afterwards.
**Lesson:** #37.

## Phase 22 - GLM-5.3 Flash trial; separate accounts per pane; no console pop-ups (2026-10-22)

**Symptom:** (1) the owner asked to try GLM-5.3 Flash on real coding tasks (thinking switched to
xhigh mid-way); (2) asked for agent panes on separate subscriptions/emails of the same tool to
get around per-account limits; (3) a test window showed "nested herdr is disabled", and the
owner asked that no CMD window ever pops up.
**Trial:** `HERDR-trials\flash-trial` (Python module with 2 seeded bugs, 5 daily tasks) and 16
hidden tests kept outside its folder. `cline-flash` (Cline Pass GLM-5.3 Flash, `-Effort xhigh`)
did all 5 tasks: hidden 16/16 pass, its own 13 tests pass, clean diff, honest doubts. Restarted
once from a git reset when the owner asked for xhigh. Cline's "Cline Desktop" promo appeared
again at start (LESSONS #12).
**Accounts:** `agents.json` `account` per profile (env / args with `{dir}`, login command,
settings to copy - never sign-in files); `herdr-agents -Account`, `login`, `accounts`; team
plans (`account`) and a new **account** box per column in the layout builder; HUD finds Codex
and Claude logs in account folders and shows `account <name>`. Proof: each CLI reported "not
logged in" in an empty folder while the normal Codex stayed "Logged in using ChatGPT"; in an
isolated session `start codex -Account trial` opened Codex's own "Sign in with ChatGPT" screen
(LESSONS #38). Antigravity could not be separated (no support). Test account and folders removed.
**Pop-ups:** the error window was Claude's own test window - opening `herdr --session lab` from
inside a Herdr pane is refused as nesting; the earlier "attach a window" advice (LESSONS #24)
was wrong and is corrected, and CLAUDE.md now forbids test windows. Every Herdr key command,
`Herdr.cmd`'s HUD start, the Startup HUD shortcut and the Stop Herdr icon now run
`conhost.exe --headless pwsh ...`. Tried first: a VBS hidden launcher - no console, but the
dialogs stayed invisible (`visible=False`), even after showing a throwaway window first; dropped.
Proof: layout builder and Stop dialog `visible=True` with no ConsoleWindowClass window; keys
(9) switched in keys.toml + live config (backed up), `config check` ok, reload applied. Left:
`herdr` typed in Explorer still runs `Herdr.cmd`, which flashes a console for a moment.
**Lesson:** #24 (corrected), #38, #39.
## Phase 23 - HerdrLauncher.exe: starting Herdr never flashes a console (2026-10-22)

**Symptom:** typing `herdr` in Explorer (App Path -> `Herdr.cmd`) flashed a console window for
a moment - the one pop-up left after Phase 22.
**Cause:** a batch file always gets a console from Windows.
**Fix:** `launcher\HerdrLauncher.exe`, a GUI-subsystem exe (`/target:winexe`, C# 5, built by
`launcher\build.ps1` with the .NET Framework `csc.exe` - no SDK). Same folder rules as
`Herdr.cmd` (argument, else current folder, System32 -> the GitHub folder); starts the HUD with
`conhost --headless pwsh` and `wt.exe -d <folder> --title Herdr herdr.exe`; `--dry-run` writes
the commands to `%TEMP%\herdr-launcher-dryrun.txt`. Written by GLM-5.3 Flash (Cline Pass) from
`tasks\herdr-launcher.md`. The lead's hidden tests
(`HERDR-trials\launcher-tests\test-launcher.ps1`, split every command line with Windows' own
`CommandLineToArgvW`): 16/20 first - a folder ending in `\` (including `D:\`) produced
`"D:\"`, which Windows reads as an escaped quote, so wt got a broken command line. Flash fixed
it (`tasks\herdr-launcher-fix1.md`): `Quote()` doubles trailing backslashes -> 21/21. While
Flash was still thinking, Herdr reported Cline `idle`; its screen showed the truth (LESSONS #41).
**Live proof (owner told first):** `LAUNCHER_REAL=1` -> 23/23: no visible ConsoleWindowClass
window at any 40 ms sample for 6 s, a new Herdr client (`herdr.exe`, 16:45:22) attached in a new
Windows Terminal window, and the HUD's mutex kept it to one copy.
**Switch:** the first attempt to change the registry from Claude was refused by Claude Code's
permission classifier ("persistence"); the owner said "try again" and Claude ran the new
`launcher\use-launcher.ps1`: App Path `herdr.exe` -> the exe (old value `...\HERDR\Herdr.cmd`),
Desktop + Start-menu `Herdr.lnk` -> the exe (were `wt.exe -d "D:\WORK\Software
Development\GitHub" --title Herdr herdr.exe`; icon, working folder kept); backups in
`%APPDATA%\herdr\backup\launcher-20261022-165104\`; `-Undo` restores. `Herdr.cmd` stays as the
fallback. The Flash pane was already closed when the lead came to stop it.
**Lesson:** #40, #41.

## Phase 24 - Claude usage limits per account; switch at 95%; handoffs (2026-10-22)

**Symptom:** the owner wanted to see each Claude account's limits and, when the account in use
passes 95% of its 5-hour limit, to continue on the account with the least use. Claude 2030
(main sign-in) reached 99% at 16:41 and handed over to 2040 with `tasks\handoff-2040.md`; the
owner then asked for a better, more detailed handoff, and signed in a third account (1984).
**Found:** Claude Code's status line input carries `rate_limits.five_hour` / `seven_day`
(`used_percentage`, `resets_at` unix seconds); `~\.claude\statusline.js` (owner-approved,
Phase 22) already writes them with `config_dir` to `%APPDATA%\herdr\claude-cost\<session>.json`
(LESSONS #42). Codex's `token_count.rate_limits` were null on this plan - Claude only.
**Fix:**
- `claude-limits.ps1` (dot-sourced): newest file per account wins; a window whose reset time
  has passed counts 0%; `LIMIT` at >= 95% of the 5-hour OR weekly window; NEXT = lowest 5-hour
  use below the limit (accounts without data are never chosen).
- `herdr-agents usage` (table + `NEXT:`), `start next`, `start 2030` (alias, main sign-in,
  `agents.json` `accountNames.claude = "2030"`), alias `1984`; `handoff` writes
  `tasks\handoff-<time>.md` from the new `tasks\HANDOFF-TEMPLATE.md` with the limits, live
  agents, git state and newest HISTORY phase filled in, 10 sections to fill (owner's words,
  done-with-proof, in progress, decisions, refused actions, numbered next steps with checks,
  traps, rules, definition of done) and never overwrites.
- HUD: Claude panes show `5h N%` (token `$limit`) and `account <name>` from `config_dir` (also
  for Claude panes not started by herdr-agents); one Herdr notification per account and reset
  window at 95% (`"<account> at N% - switch: Ctrl+B Shift+C"`, sound `request`).
- `ctrl+b shift+c` (added by 2030 for `start 2040`, uncommitted until now) now runs
  `start next` (keys.toml + live config, backup `config.toml.bak-20261022-165405`; `config
  check` ok, `server reload-config` applied).
- Third account: `herdr-agents login claude -Account 2050`, owner signed in (`claude auth
  status`: mflma1984, Pro), folder renamed to `1984` at the owner's word.
- Skill `herdr-team` (commands + 1b: usage before a big task, handoff at 95%), copied to the
  account folders' own skill copies too; memory `reference_claude_2040_account.md` names all three.
**Proof:** `usage` live: 2030* 99% LIMIT, 2040 5%, 1984 no data, `NEXT: 2040`; HUD `-Once`:
`... $2.24 · 5h 6% · ctx 14% ... · account 2040`. Lead's tests
`HERDR-trials\limits-tests\test-limits.ps1` (fake APPDATA): 14/14 - newest file wins, reset ->
0%, weekly limit, trailing `\` in config_dir, no data, NEXT, one notification not two, `-Once`
silent; mutation (threshold 99) -> 5 FAIL, restored. Isolated session: `start next` ->
`claude-2040` with `CLAUDE_CONFIG_DIR=...\2040`; `start 2030` -> no account. Both stopped at
Claude's folder-trust question (never answered for the owner). `handoff` to a temp file filled
every fact and refused to overwrite.
**Found on the way:** the lab server, started from 2040's shell, passed 2040's
`CLAUDE_CONFIG_DIR` into the `start 2030` pane (LESSONS #43); the owner's server has no such
variable. A stray `pane run` typed into the lab Claudes' trust screens (they exited; lab only).
**Lesson:** #42, #43.

## Phase 25 - Audit batch A: delegation tool, limits, check (2026-10-23)

**Symptom:** ask reported DONE before Codex started its turn; ownership records could close
another server's pane or a replacement agent, and failed closes lost their retry record.
The audit also reproduced misleading usage selection and setup checks accepting bad config.
**Cause:** settled state and read errors were trusted, pane records lacked server/kind identity,
and checks compared rounded or incomplete data and strings outside their own blocks.
**Fix:**

- M1: confirm successful done/idle results for up to 10 seconds, wait on observed activity with the remaining timeout, and report a non-idle answer read as UNCONFIRMED.
- M2: worktrees split beside the caller with the requested direction; only callers outside a pane create a new space, and STARTED reports that accurately.
- M3: reject missing cwd folders before creating anything and resolve supplied relative paths.
- M4: record server/kind, track pane creation, follow live agents by name after moves, and refuse changed kinds while preserving legacy records.
- F01: scope ownership lookups and record replacement/removal to the normalized socket identity; absent legacy identities mean default.
- F02: reserve run files exclusively with numeric collision suffixes and list history by write time.
- F03: failed starts in supplied panes save no ownership; stop refuses a different agent occupying the recorded pane.
- F06: reject Codex danger-full-access sandbox options in separate-value and equals forms.
- F07: compare each key's command and type within its own block, tolerate comments/order/quoting, and reject duplicate keys in master or live config.
- F08: check every backtick path in the first FILES table cell, ignoring description-column examples.
- F09: label usage as session cost (updated today); the corresponding GUIDE definition is reserved for batch B.
- F10: unknown five-hour percentages display as ? and cannot become NEXT; expired numeric windows still count as known zero.
- F11: classify agent_blocked before failures, return BLOCKED/3, and show the visible screen.
- F12: require a successful answer read before DONE; other read errors return FAILED/1.
- F13: treat alias names as suffixable bases while rejecting explicit duplicate names.
- F14: read supplied panes' cwd and reject a conflicting explicit cwd before agent start.
- F15: retain the lead's complete drive-root cwd when building a team.
- F16: remove ownership only after close success or a successful pane listing confirms absence; preserve it on close/discovery failure.
- F24: retain double percentages for limit decisions and round only table output.
- F29: validate start and team names with case-sensitive lowercase matching before opening panes.
- R1/R2 (lead review): NOT READY starts in supplied panes lost their stop record because only successful starts were recorded; record them when agent list confirms both name and pane, preserving no-identity refusal. Key starts also inherited the focused pane's cwd before checking a supplied pane; apply that fallback only without -Pane. Regression tests reproduce both failures and reject mismatched identities; suite 46/46 passes, HEAD baseline fails 46/46, all 16 PowerShell scripts parse, and check.ps1 retains only the three known worktree failures.

**Proof:** `tests/test-batch-a.ps1` uses AST-loaded definitions and command branches with fake
Herdr, time, filesystem and child commands: 43/43 pass. The same suite against temporary
`git show HEAD:<file>` copies fails all 43 tests, covering all 20 item IDs. No live session
was mutated. All PowerShell scripts parse; `check.ps1` retains only the three audit-baseline
worktree failures (two skill newline/hash differences and the absent untracked `herdr/`
clone), plus the existing PATH warning. Pre-existing untracked audit/spec inputs are retained.
**Found on the way:** the company gateway returned 403 to the long fix conversation of Codex; a fresh conversation worked (LESSONS #45).
**Lesson:** #44, #45. Read the answer before claiming completion; confirm ownership before closing.

## Phase 26 - Audit batch B: launcher, installer, spaces, stop, docs (2026-10-23)

- **F04** - **Symptom:** PATH updates had no recovery copy. **Cause:** the raw registry value was overwritten directly. **Fix:** exclusively reserve a timestamped JSON backup with existed/value/kind/time before SetValue; report its path and abort on backup failure. **Proof:** raw-value, failed-write and same-second collision fixtures. **Lesson:** back up before mutation, retaining the original registry kind.
- **F30** - **Symptom:** rollback could retain four releases with Keep=3. **Cause:** current consumed its slot only when encountered. **Fix:** reserve its slot before counting other releases. **Proof:** oldest-current fixtures with Keep=3 and Keep=1. **Lesson:** protected items consume the retention budget.
- **F22** - **Symptom:** the installer batch always chose Windows PowerShell. **Cause:** a hard-coded host. **Fix:** prefer pwsh.exe when found; otherwise print the specified fallback message and use powershell.exe, preserving exit forwarding. **Proof:** batch branch/exit text fixture. **Lesson:** keep fresh-PC fallback explicit.
- **F21** - **Symptom:** a trailing folder backslash broke the fallback launcher's arguments. **Cause:** it escaped the closing native quote. **Fix:** append one backslash before the wt command, after the separate cd fallback. **Proof:** CommandLineToArgvW fixtures preserve D:\, D:\a b\ and D:\a b. **Lesson:** #40; validate parsed arguments.
- **F23** - **Symptom:** a relative folder could resolve as docs\docs. **Cause:** the launcher used it for both WorkingDirectory and wt -d. **Fix:** normalize accepted folder arguments with Path.GetFullPath; rebuild the exe. **Proof:** reflection invokes only ResolveFolder('docs') and receives the absolute repo docs path. **Lesson:** resolve a relative folder once.
- **F17** - **Symptom:** the team shortcut ignored the configured Codex model. **Cause:** agent start had no model arguments. **Fix:** read agents.json's codex modelFlag/model and pass only those after --; retain the existing start when absent. **Proof:** configured-model and missing-profile/model fixtures. **Lesson:** use the shared model profile without changing approval settings.
- **F18** - **Symptom:** joined spaces with three or more panes had unequal rows. **Cause:** the original pane's ratio grew with k. **Fix:** use 1/(n-k+1) on the bottom remainder. **Proof:** recurrence simulations for 2, 3 and 5 panes agree within 1e-9. **Lesson:** #17; the ratio belongs to the original pane.
- **F19** - **Symptom:** failed Separate lost its retry record. **Cause:** discovery failures looked like absent panes and the record was always deleted. **Fix:** abort on failed listing; retain only unresolved groups with current IDs and already-separated destinations, skipping completed moves on retry. **Proof:** failed discovery, partial move/retry, first-move and lookup failure fixtures. **Lesson:** retain recovery state until separation succeeds.
- **F20** - **Symptom:** Stop reported an empty running server as stopped. **Cause:** PowerShell unrolled the empty array to null. **Fix:** return Ok and an Agents array, update status and dialog guards, and report failed calls as not reachable. **Proof:** zero, failed, one and many-agent fixtures. **Lesson:** distinguish a successful empty result from failure.
- **F05** - **Symptom:** the workspace skill still directed tests into the live session. **Cause:** the procedure predated isolated testing. **Fix:** document the absolute executable, headless lab server, explicit socket and finally teardown, reserving live tests for what lab cannot show. **Proof:** docs reviewed against CLAUDE.md section 3. **Lesson:** #20, #24, #39; protect the owner's session.
- **M2** - **Symptom:** current docs still promised a new space for every -Worktree start. **Cause:** batch A changed placement. **Fix:** GUIDE, DELEGATION and the team skill now describe a pane beside the caller, with a new space only outside Herdr. **Proof:** scoped text search and review. **Lesson:** update usage instructions with behaviour.
- **F09** - **Symptom:** GUIDE called cumulative refreshed-session cost today's spending. **Cause:** the old label implied a daily delta. **Fix:** explain session cost (updated today), including earlier days in refreshed sessions. **Proof:** docs matches batch A's usage label and calculation. **Lesson:** name the quantity actually measured.
- **F25** - **Symptom:** CLINE-PASS still asserted current re-authentication failure. **Cause:** pre-sign-in text survived later verification. **Fix:** label that state as history and retain the four tested models, bare-ID rejection and eight untested IDs. **Proof:** docs compared with the lead's verification in that file. **Lesson:** distinguish historical failure from verified setup.
- **F26** - **Symptom:** profiles and sign-in recipes described direct DeepSeek and CLI defaults. **Cause:** docs predated Cline Pass profiles. **Fix:** document cline/glm-5.3, cline-flash/glm-5.3-flash, deepseek/deepseek-v4-pro and kimi/kimi-k3 with prefixed IDs and Cline Pass sign-in; retain explicit direct-provider overrides. **Proof:** scoped docs/skill review against agents.json. **Lesson:** profiles decide their provider and model.
- **F27** - **Symptom:** GUIDE section 5 denied the displayed dollar cost. **Cause:** it predated the status-line bridge. **Fix:** explain cumulative session cost and its availability after a cost file is written. **Proof:** docs matches the existing HUD reader and section 13.7. **Lesson:** document the data's availability condition.
- **F28** - **Symptom:** SETUP rebuilt the superseded launcher and omitted required shortcuts. **Cause:** steps predated the windowless exe. **Fix:** document build/use-launcher, fresh-PC prerequisites, all ten key blocks, headless Startup HUD and Desktop Stop links, backups, and Herdr.cmd as fallback. **Proof:** docs checked against launcher scripts and config. **Lesson:** rebuild instructions must include registration prerequisites.

**Proof:** `tests/test-batch-b.ps1` passes 18/18 offline. Against temporary HEAD copies,
17/18 fail, covering every one of the ten code items; the passing missing-model fallback
fixture verifies intentionally unchanged behaviour. Batch A remains 46/46. All 17
PowerShell files parse; launcher/build.ps1 exits 0. check.ps1 reports only the three known
worktree failures (two unsynchronized skills and absent herdr/), plus the existing PATH
warning. No live Herdr session, registry value, installed skill or shortcut was changed.
The lead's AGENT-ROSTER.md edit and the supplied task spec were left intact.
**Lead review:** renamed a `$profile` variable in herdr-open-folder.ps1 (it hid PowerShell's automatic `$PROFILE`), and made the F21 test check behaviour only (it failed the correct `D:\a b` case on the old script just because the new line was absent; now 16/18 fail on the old code - every broken case). Lead's model trial (GLM-5.3 Flash vs DeepSeek V4.1 Flash, xhigh vs high) added to docs/AGENT-ROSTER.md section 7.
**Lesson:** a regression test must test behaviour, not the presence of the new code line.

## Phase 27 - DeepSeek V4.1 Flash profile with default effort high (2026-10-23)

**Symptom:** the owner's trial winner (DeepSeek V4.1 Flash, thinking high; docs/AGENT-ROSTER.md
section 7) had no profile: every start needed `start deepseek -Model cline-pass/deepseek-v4.1-flash
-Effort high`.
**Cause:** profiles had no default reasoning effort; `effortArgs` applied only with `-Effort`.
**Fix:** new optional profile field `effort` = the default level when `-Effort` is not given
(`-Effort` still overrides; profiles without it unchanged). New profile `ds-flash`
(`cline-pass/deepseek-v4.1-flash`, `effort: "high"`, same args and account block as the other
Cline Pass profiles). Skill herdr-team and the roster's routing rule name it first for bulk
coding and reviews.
**Proof:** `tests/test-batch-a.ps1` test E1 (default high, `-Effort xhigh` overrides, no field ->
no flag): 47/47 pass; E1 fails on the previous code. In the trial, the same model and flags
(`--thinking high`, read back from the process arguments) scored 41/41.
**Lesson:** a model goes into a profile only after a measured trial.

## Phase 28 - Default server identity: pane socket path = "default" (2026-10-23)

**Symptom:** right after the merge, `herdr-agents stop ds-push` (started by the pre-merge tool a
minute earlier) said "'ds-push' was not started by this tool"; `status` no longer marked Codex
as started-by-tool. Every older record was unmanageable from inside a pane.
**Cause:** Phase 25 (F01) scoped records by `HERDR_SOCKET_PATH`, else "default". Herdr sets that
variable in every pane to the DEFAULT socket, so a pane saw "c:\...\herdr.sock" while legacy
records and key-started records said "default" - one server, two identities. The lead's spec
designed it; the batch A tests never set the variable to the default socket.
**Fix:** `ConvertTo-ServerId` maps empty, "default" and the default socket path
(`%APPDATA%\herdr\herdr.sock`, any case) to "default"; used for the caller and for records.
Other sockets (isolated sessions) stay separate.
**Proof:** new test S1 (legacy, key-started and pane-started records all stop from a pane on the
default socket; a new record saves "default"): 48/48 pass, S1 fails on the previous code. Live:
`stop ds-push` -> STOPPED; `status` marks Codex started-by-tool again.
**Lesson:** #46. Test with the environment a real pane has, not only an empty one.

## Phase 29 - Round 2: fresh Codex conversations, model trials, Cline in the HUD, team rules (2026-10-23)

**Symptom:** long Codex conversations were rejected by the company gateway (LESSONS #45) and needed
a hand-typed `/new`; model trials were run by hand; the HUD showed nothing for Cline agents; every
spec repeated the same delegate rules; the owner asked for an always-on documentation rule and
for panes only for Codex, Antigravity and Claude.
**Fix:**
- `herdr-agents ask <codex> -Fresh` (Codex GPT-6-Astra medium): `/new`, answers the worktree menu,
  waits for the fresh screen (banner, no transcript lines), then the task. The lead's first design
  (wait for a new session id) failed live: Codex makes the id only with the first message
  (LESSONS #48). Timeout 60 s after a live trace (menu ~2 s, fresh ~21 s).
- `herdr-trial.ps1` (Astra): copy the trial kit, start a profile, prompt, wait for a stable
  REPORT.md, grade, read Cline tokens, append `results.csv`. `-Profile` is an alias of
  `-AgentProfile` (no automatic `$PROFILE` shadowing).
- HUD (DeepSeek V4.1 Flash): Cline panes show the session model and `in <fresh> · out <out>`;
  the session is found by folder (Cline's session pid is the shared hub daemon, LESSONS #49);
  completed/failed/exited/done sessions skipped; a newer live session replaces a cached one.
- `agents.json`: Claude profile gets `effortArgs` (`--effort {0}`).
- `AGENTS.md`: 9 delegate rules every agent reads at startup (deliver verified work, stay in scope,
  prove claims, tests fail on old code, no silent failure, style, safety, no loops, answer format).
- `CLAUDE.md` section 0: the owner's first rule (every experience into the docs at once) and the
  pane rule (Cline headless).
- `docs/TEAM-PLAYBOOK.md`: strategies, control cards, team-size rules, verification ladder, loops,
  from web research + our measurements. Roster: Argon High and Astra medium results.
**Proof:** suites: round2 18/18, HUD 11/11, batch A 48/48, batch B 18/18; new tests fail on the
previous code (reported by the agents, A tests rechecked by the lead). Live: `ask astra -Fresh`
-> `FRESH-OK` (69 s); HUD functions read a real Cline session (DeepSeek, read-only).
**Lesson:** #47-#54. Mocked tests passed a wrong design twice; only the live check caught it.

## Phase 30 - Cline headless runner; the toolkit becomes GitHub main (2026-10-23)

**Symptom:** the owner wants only Codex, Antigravity and Claude agents in panes; Cline agents in
panes also had unreliable idle/done state (LESSONS #41). The owner also wanted all toolkit
updates on GitHub's `main`, which held the Herdr Rust application.
**Fix:**
- `herdr-headless.ps1` (written by DeepSeek V4.1 Flash, itself run headless - 21 min, 64 steps):
  `-Profile <cline profile> -Task|-TaskFile [-Cwd -Model -Effort -Account -TimeoutSec]` builds
  the cline arguments from agents.json (no `--tui`; `--json -c -t`), runs with no window, logs
  JSONL to `agent-runs`, reads `run_result` (finishReason, tokens) and the final answer; exit 0 =
  DONE, 1 = FAILED; kills the process tree 30 s after the timeout. The lead renamed a `$Profile`
  parameter (AGENTS.md rule 6). Skill herdr-team: Cline models are started this way.
- GitHub (owner's choice "Replace main, keep a backup"): the old `main` (b14199cb, the Herdr app
  with the custom-command-menus work) copied to a new branch `herdr-app` and verified; then the
  toolkit force-pushed to `main` with `--force-with-lease` on that exact commit. `master`
  untouched; `windows-toolkit` kept equal to `main`; local `main` tracks `origin/main`.
**Proof:** `tests/test-headless.ps1` 9/9 (0/9 on the previous code; H9 fails on the agent's own
first, buggy version). Live: `herdr-headless.ps1 -Profile ds-flash -Task ...` -> `DONE ... in 61 s
- in 15297 out 476`, `product.txt` = 42. GitHub: `herdr-app` = b14199cb, `main` = local main,
`master` = 5bd900d8 before and after.
**Lesson:** #55. An honest agent disclosed its own mistake (a case-insensitive name clash
started the real cline twice) and added a test that catches it - keep asking for that.

## Phase 31 - Owner rules: no own subagents, headless research, close panes, no browser (2026-10-24)

**Symptom:** the lead used an in-session Claude subagent (its own account's tokens); headless
Argon research drove the owner's Chrome (31 processes / 3.5 GB -> 23 / 2.0 GB after stopping it);
a finished trial pane (Gemini Flash) stayed open overnight; the PC is short on RAM and disk.
**Fix (rules, CLAUDE.md section 0 + AGENTS.md):**
- Claude helpers only on the owner's OTHER accounts (1984 / 2040), in panes beside the lead;
  never subagents of the lead's own session.
- Research runs headless (Codex `--search exec`); only coding, review and fix work gets a pane.
- Close every pane once its task is done and no more work is planned; remove merged worktrees and
  temp copies.
- AGENTS.md rule 7: no browser tools for delegates. Antigravity's route into Chrome is the
  "Antigravity Browser Extension" (owner's setting, chrome://extensions).
**Proof:** Chrome RAM before/after stopping Argon (above); `t-gflash` pane graded (41/41) and closed.
**Lesson:** #56.

## Phase 32 - Claude helpers get the delegate rules; Claude model control cards (2026-10-24)

**Symptom:** Codex research (headless, `codex --search exec`, 19 min) reported that Claude Code
reads AGENTS.md only when no CLAUDE.md exists - this folder has one, so Claude helpers never saw
the 9 delegate rules; and that Haiku 5.5 at xhigh can end a turn with no visible answer.
**Fix:** CLAUDE.md imports `@AGENTS.md` for delegates; TEAM-PLAYBOOK control cards for Opus,
Sonnet and Haiku 5.5 (effort defaults, failure modes, counter-measures) from Anthropic's guides.
**Proof:** the lead checked both claims on Anthropic's pages (Haiku prompting guide; Claude Code
memory docs, "AGENTS.md" table).
**Lesson:** #57, #58.
