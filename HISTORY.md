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
Desktop + Start-menu `Herdr.lnk` -> the exe (were `wt.exe -d "<former GitHub folder>" --title Herdr herdr.exe`; icon, working folder kept); backups in
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

## Phase 33 - `ask` startup-dialog guard; first real-task model trial (2026-10-24)

**Symptom:** a task prompt sent to an agent at a startup dialog was lost and its Enter answered
the dialog for the owner (LESSONS #53).
**Fix:** `Get-AgentDialog` + a guard in `ask`: before any prompt (and before `-Fresh`) it reads
the visible screen; a trust / browser / model dialog (phrase in the last 20 lines AND a selection
footer in the last 6) or a sign-in message (last 6 lines) -> `BLOCKED ... the owner answers it in
the pane`, run saved, exit 3, nothing sent. A failed read never blocks.
**How it was built (owner's idea: tests must improve the app):** the first real-task trial -
Claude Sonnet 5.5 high and Claude Haiku 5.5 xhigh (account 1984, own worktrees) implemented the
same spec against the lead's hidden tests. Both 14/14 and honestly PARTIAL with the same proven
root cause (a round-2 fixture consumed by the new read - the lead's spec conflict); Haiku
($0.45, 9 min) also found that Codex's "Trust this folder?" was not matched; Sonnet ($0.78, 5 min)
was faster. Haiku's version merged after two lead corrections: fixture fix, footer rule.
**Proof:** hidden tests 17/17 (reference 17/17, unchanged code 2/14 earlier, 2 mutants caught);
`tests/test-dialog-guard.ps1` 26/26 (23 fail on the old code; Haiku's own mutation: 4 fail
without the footer rule); round2 18/18; batch A 48/48. LIVE: `ask haiku "..."` on a real Claude
pane -> first version BLOCKED (false positive), final version DONE.
**Lesson:** #59.

## Phase 34 - Herdr app source merged into `main` (2026-10-24)

**Symptom:** the GitHub repo held two unrelated histories: `main` / `windows-toolkit` (this
toolkit) and `master` / `herdr-app` / `feature/custom-command-menus` (the upstream Herdr source,
1868 commits, plus the owner's custom-menus commit `b14199c`). The owner wanted everything in `main`.
**Fix:** subtree merge of `herdr-app` into `main` under `herdr-app/` (history kept; no toolkit file
touched). Not under `herdr/`: that path is the owner's local upstream clone (ignored), and a tracked
`herdr/` would collide with it on `git pull`.
**Proof:** `git merge -s ours --allow-unrelated-histories` + `git read-tree --prefix=herdr-app/`;
3107 files added, `git diff HEAD~1 --stat -- . ':!herdr-app'` shows only this entry and FILES.md.
`check.ps1` not run (no pwsh in the cloud session).

## Phase 35 - CLI update day and Codex Sol 6.1 medium beside the lead (2026-10-24)

**Symptom:** the owner asked to update every CLI and run Codex GPT-6.1-Sol medium beside the lead.
`npm install -g` failed halfway with ENOSPC (C: had 0.7 GB free; npm cache was 3.2 GB). Then
`herdr-agents start codex` failed twice with `agent_pane_busy ... not an available shell` although
the new pane showed a normal pwsh prompt.
**Fix:** `npm cache clean --force` (C: 0.7 -> 3.8 GB), reran the installs. Updated: codex 0.162.0 ->
0.164.0, cline 3.0.60 -> 3.0.73, opencode 1.18.35 -> 1.19.4, gemini 0.57.0 -> 0.65.0, kilo 7.5.5 ->
7.8.17, qwen 0.22.2 -> 0.26.0, grok 1.0.50 -> 1.0.70, agy 1.1.13 -> 1.4.7, cursor-agent -> 2026.10.22.
Claude Code 2.1.310 already current. NOT updated: openclaw (needs Node >=24.22.1, has 24.14.0; in one
npm batch it aborts every other package), claude-mem 13 -> 15 and pnpm 11 -> 12 (major versions),
pm2 (runs as a service). Codex started by hand: `herdr pane run w1:p3 "codex -m gpt-6.1-sol -a never
-s workspace-write -c model_reasoning_effort=medium"`; Herdr then detected it (agent list: codex,
gpt-6.1-sol; screen: `GPT-6.1-Sol medium`).
**Lesson:** #60.

## Phase 36 - Diagnose setup checks on this PC (2026-10-24)

**Symptom:** `pwsh -NoProfile -File I:\herdr\check.ps1` exits 1 with 20 FAIL lines.
**Cause / evidence (local file times, UTC+03:00):**

| Failures | Observed cause | Since when / limits |
|---|---|---|
| 10 keys: prefix+f, shift+f, t, shift+v, shift+b, a, shift+s, i, shift+q, shift+c | The live config is 74 bytes: onboarding plus theme only; no key blocks | Created 2026-10-24 10:36:19, last written 10:55:54; absent at this check. No `config.toml*` backups beside it. The exact writer is unknown |
| 6 settings lines: toast table/delivery, agents table/rows, spaces table/rows | Those three tables are absent from the same live config | Same file and time evidence as the keys |
| 2 skills | Both installed directories and SKILL.md files are absent, rather than different text; `-Check` calls either case DIFFERS | Absent at this check; exact removal/non-install time unknown. Master files created here 2026-10-24 10:21:43; Phase 25/26 already documented unsynchronized skill copies on the earlier setup |
| 1 App Path | HKCU App Paths/herdr.exe key is absent, not merely a broken target; no launcher backup directory either | Absent at this check; exact removal/non-install time unknown. Phase 23's earlier registration is not present here |
| 1 file-list row | FILES.md listed the ignored upstream `herdr/` clone as present | Phase 25/26 already reported it missing on 2026-10-23; Phase 34 retained that assumption. Tracked source is `herdr-app/`, which exists |

The local `.git/logs/HEAD` was read directly (no git commands, per task scope): clone at
10:17:39, switch to main at 10:21:43, fast-forward at 10:35:32 on 2026-10-24.
The installed Herdr executable was created here at 10:36:06; version output is 0.9.5,
the same version recorded in Phase 2. This supports an incomplete installation of this
personal setup after cloning to I:, not a proven config reset or format change.
Neither the responsible process nor whether this is different hardware is established.
The repository's `config.toml.backup` is also theme-only, not a recovery copy of our blocks.

**Fix:** corrected only the absent-clone row in FILES.md; added LESSONS #61. External
writes are forbidden by this task's workspace sandbox, so no config, skills, registry
or shortcuts were changed and no external backup was created. No agents or servers were
started or stopped. Existing root/docs edits were preserved; no git commands were run.

**Proof:** check.ps1: before 20 FAIL, after 19 FAIL (exit 1); docs line OK with 37 phases
and 62 lessons (including a concurrent agent's separate entry). `herdr.exe --session lab config check` -> `config: ok` (exit 0; read-only,
no server needed). Python's `tomllib` parsed the current config plus the two master files:
10 unique keys, notifications system, current catppuccin-latte theme preserved. All ten
unmodified master commands target nonexistent D: scripts. Replacing only the old folder
with I:\herdr in memory produces valid TOML with zero missing script targets. This is
syntax/path verification, not a live test of shortcuts or a completed installation.

**Owner commands for the remaining 19 failures (not executed):** run in PowerShell 7.
The key master needs the following path correction as a follow-up outside this task's
file-edit scope. It is backed up first; otherwise a matching config would still invoke
ten missing scripts. These commands assume the observed theme-only config is unchanged;
the guard stops if another process has already added keys or the relevant settings.

```powershell
$ErrorActionPreference = 'Stop'
$repairStamp = Get-Date -Format 'yyyyMMdd-HHmmss-fffffff'
$liveConfig = Join-Path $env:APPDATA 'herdr\config.toml'
$keyMaster = 'I:\herdr\config\keys.toml'
$liveText = [IO.File]::ReadAllText($liveConfig)
if ($liveText -match '(?m)^\s*\[(?:\[?keys|ui\.toast|ui\.sidebar\.(?:agents|spaces))') {
    throw 'Config changed: merge the blocks instead of appending duplicates.'
}
$configBackup = "$liveConfig.$repairStamp.bak"
$keyBackup = "$keyMaster.$repairStamp.bak"
[IO.File]::Copy($liveConfig, $configBackup, $false)
[IO.File]::Copy($keyMaster, $keyBackup, $false)
Write-Output "Backups: $configBackup ; $keyBackup"
$keyText = [IO.File]::ReadAllText($keyMaster).Replace(
    '<former toolkit folder>', 'I:\herdr')
[IO.File]::WriteAllText($keyMaster, $keyText)
$settingsText = [IO.File]::ReadAllText('I:\herdr\config\settings.toml')
[IO.File]::WriteAllText($liveConfig, $liveText + "`r`n" + $keyText + "`r`n" + $settingsText)
pwsh -NoProfile -File I:\herdr\sync-skills.ps1
if ($LASTEXITCODE -ne 0) { throw 'Skill sync failed.' }
pwsh -NoProfile -File I:\herdr\sync-skills.ps1 -Check
if ($LASTEXITCODE -ne 0) { throw 'Installed skills still differ.' }
```

The sync script backs up changed installed copies before overwriting; both copies are
currently missing. For App Path, SETUP step 7 requires an existing key before
use-launcher.ps1 can export it. Record its original absence before creating it:

```powershell
$ErrorActionPreference = 'Stop'
$appKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\App Paths\herdr.exe'
if (-not (Test-Path -LiteralPath $appKey)) {
    $absenceRecord = Join-Path $env:APPDATA ('herdr\apppath-absent-' + (Get-Date -Format 'yyyyMMdd-HHmmss-fffffff') + '.json')
    [IO.File]::WriteAllText($absenceRecord, '{"existed":false,"key":"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\App Paths\\herdr.exe"}')
    Write-Output "Original absent registration recorded: $absenceRecord"
    New-Item -Path $appKey -Force | Out-Null
    Set-ItemProperty -LiteralPath $appKey -Name '(default)' -Value 'I:\herdr\launcher\HerdrLauncher.exe'
}
pwsh -NoProfile -File I:\herdr\launcher\use-launcher.ps1
if ($LASTEXITCODE -ne 0) { throw 'Launcher registration failed.' }
& "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe" --session lab config check
if ($LASTEXITCODE -ne 0) { throw 'Config validation failed; restore the printed config backup.' }
pwsh -NoProfile -File I:\herdr\check.ps1
if ($LASTEXITCODE -ne 0) { throw 'Setup still has failures; read the FAIL lines.' }
```

use-launcher.ps1 prints its own registry/shortcut backup directory under
`%APPDATA%\herdr\backup\launcher-<time>\` and reads the targets back. Its `-Undo` returns
to its exported registration; restoring the originally absent key instead requires removing
that new key, using the absence record. No live reload command is included because this
task forbids modifying the owner's running session; applying the config to that session
is a separate owner action. The missing Desktop/Start-menu/HUD links and user PATH are
warnings, outside the permitted repairs; SETUP steps 6, 8 and 9 cover them.

**Not verified:** the owner commands have not run; external repairs and running shortcut
behaviour remain unverified. Commit history (`git log`) was not queried because the task
says no git; the local reflog and existing HISTORY entries provide the chronology above.
No regression tests were added for this documentation-only change.
**Lesson:** #61.

## Phase 37 - Hermes/OpenClaw removal blocked; scoped script and evidence (2026-10-24)

**Symptom:** the owner requested complete removal; Hermes folders, executable and a
Start-menu link remained during discovery.
**Cause:** this session permits writes only inside I:\herdr; external removal is outside
its sandbox. Task/service discovery and some filesystem reads also returned access denied.
**Fix:** saved tasks/remove-hermes-openclaw.ps1 with -WhatIf, exact deletion allowlist,
size output, process stops scoped by executable path, PATH recovery copy, admin recovery
commands and before/after checks. Ran discovery only; no removal, registry edits, git or pane
operations. Unknown discoveries are reported for review, not automatically deleted.
**Proof:** tasks/remove-hermes-openclaw-dryrun.txt records sizes and free space; openclaw
lookup/shims absent, hermes lookup present. Script parsing passed. check.ps1 exited 1 with
19 setup/configuration failures, saved in tasks/remove-hermes-openclaw-check.txt.
Removal and full drive-content verification remain BLOCKED. The script keeps deletion in
PowerShell with LiteralPath rather than handing paths to a second shell.
**Lesson:** #62.

## Phase 38 - Complete the setup repairs with owner-authorized disk access (2026-10-24)

**Symptom:** Phase 36 left 19 setup FAILs because external writes were sandbox-blocked.
**Cause:** the same incomplete local setup remains: theme-only config, both installed skill
directories absent, and App Path key absent. Rechecked before repair; no evidence of an
update reset. The absent upstream-clone FILES.md row was already corrected in Phase 36.
**Fix:** owner explicitly authorized full disk access for this task on 2026-10-24. Backed up
the live config, recorded original skill/registry absence, appended the unmodified master key
and settings blocks, ran sync-skills.ps1, created the required App Path key, then ran
launcher/use-launcher.ps1. Original config text (including catppuccin-latte) is preserved.
No change to ~/.claude/settings.json, master keys, approval/trust settings or herdr-app/;
no git, agents, browser, live reload or server start/stop commands.

**Backups / evidence:**
- `C:\Users\mflma\AppData\Roaming\herdr\backup\check-repair-20261024-110714-8581912\`:
  config.toml.bak, skills-before.json, apppath-before.json, check-before.txt,
  check-after.txt and key-targets.json. Both skill directories and the App Path key were
  absent, so their original absence is recorded instead of copying nonexistent files.
- `C:\Users\mflma\AppData\Roaming\herdr\backup\launcher-20261024-110727\`:
  use-launcher.ps1 registry export (created after initial registration). Its -Undo restores
  that registration, not the original absence; apppath-before.json records the original state.
- `tasks/fix-check-failures-backup-path.txt` points to the repair backup directory.

**Proof:** `pwsh -NoProfile -File I:\herdr\check.ps1`: before 19 FAIL / exit 1,
after 0 FAIL / exit 0. `herdr.exe --session lab config check`: config: ok / exit 0
(no server started). `sync-skills.ps1 -Check`: OK herdr-team, OK herdr-workspace / exit 0.
Launcher script readback: App Path = I:\herdr\launcher\HerdrLauncher.exe (file exists).
Original-config prefix comparison: True. Path inspection: 10 key commands, 10 missing
script targets under the former toolkit folder on D:.

**Not done / doubts:** owner specifically requested reporting the missing D: paths without
guessing replacements, so they were copied unchanged and remain broken. Passing check.ps1
does not prove shortcut operation. No live shortcut test or reload was performed. Four
warnings remain: Desktop and Start-menu Herdr links, Startup HUD link, user PATH. These
are outside the allowed repairs. No new code or regression tests were needed.
**Lesson:** #63.

## Phase 39 - Execute owner-authorized Hermes/OpenClaw cleanup (2026-10-24)

**Symptom:** Phase 37 only produced a dry run. The owner now authorized full disk access
and actual removal, with exact elevated commands for anything requiring an administrator.
**Cause:** the old script missed the discovered Start-menu link; its null environment
writes left named empty registry values in the real run. A root/home-only scan could not
establish complete removal. Disk access did not grant task/service deletion rights.
**Fix:** reviewed the script; added the shortcut, uv tool-directory and Startup discovery,
explicit task/service inventory counts, ScanOnly and FileTargetsOnly modes, and registry
value-name verification. Removed the nine listed folders and shortcut (11,110,902,502 bytes),
then three archived configuration/backup folders (54,460 bytes). Removed both user environment
values and verified their absence. The exact user PATH entry was already absent and was not
changed. No git commands, panes, owner session, Claude settings or ERP files were touched.
**Proof:** tasks/remove-hermes-openclaw-real.txt and -extra.txt record sizes, read-back,
free space, and access-denied results. C: free space 12,412,694,528 -> 22,647,898,112 bytes;
G: 305,942,528 -> 309,477,376 bytes (other activity can change these later). Focused read-only
verification found no command, environment, PATH or matching npm/pip/uv/scoop package names;
only two protected tasks and the stopped service remained. Their deletions were attempted
once, denied, and not retried; exact recovery is in tasks/remove-hermes-openclaw-admin.ps1.
A names-only recursive scan records matching names, unreadable folders and untraversed
links; Windows/other-app references are preserved. Final wide-scan results follow below.
**Lesson:** #64.

**Additional verified removal:** the wide scan discovered another Hermes source clone,
installer, archived Moltbot files, project cache/history, temporary logs/screenshots and
separate E: Moltbot/OpenClaw project copies. They were individually added to the reviewed
allowlist, measured, deleted and read back; -FileTargetsOnly prevented retries of the three
denied Windows deletions. Evidence: -cache.txt, -leftovers.txt, -e-leftovers.txt and
-e-projects.txt. The E: Moltbot copy alone contained 1,463,958,594 bytes / 49,218 files.
Read-only focused verification inspected 211 tasks and 337 services with zero scan errors;
only the two denied tasks and stopped service remained. No matching process was found.
All cleanup/admin/scan scripts parsed, and check.ps1 exited 0 (all checks passed; existing
shortcut/PATH warnings remain). Another agent concurrently completed Phase 38 setup repairs;
this task did not alter or revert that work. The interrupted initial wide-before snapshot
is incomplete; wide-after is the canonical recursive evidence, reconciled after deletions.
An automatic approval review rejected a combined housekeeping command including deletion
of that incomplete log, stating only blocked by policy; the log was left in place.

**Final source-copy removal:** E:\Github Projects\Open Claw (space in its name) contained
5,497,373,122 bytes / 218,142 files and was removed with read-back verification; evidence
is tasks/remove-hermes-openclaw-e-source.txt. All 42 reviewed targets are now absent
(37 were removed, five were already absent). The recursive scan can record DirectoryNotFound
for targets removed while it was running; those are resolved by the final explicit checks.
ERP project copies, research notes/cloud documents and other software's own matching files
remain untouched and must not be described as a completely empty keyword scan.

**Environment behavior proof:** an isolated process-only sentinel reproduced the old null call: namePresent=True, valueLength=0. Explicit Env: removal then gave namePresent=False. Evidence: tasks/remove-hermes-openclaw-environment-proof.txt. No temporary user registry value was created.

**Completed evidence:** all five recursive names-only drive traversals finished. They
inspected 4,760,324 entries in 575,657 directories; H: had zero unreadable locations.
The totals include 732 unreadable/disappeared directories (some disappeared because this
cleanup removed them) and 4,212 untraversed directory links. This is not proof that all
protected/linked contents are clear. The completed scan exposed further downloaded source,
skill archives and dedicated editor/project caches; those were measured and removed too.
Final removal total: 70 targets, 18,669,241,805 bytes (17.387 GiB); all 75 reviewed paths
absent. Five targets had already been absent. Final read-only inventory: 211 tasks, 337
services, zero scan errors; only Clawdbot Gateway, Hermes_Gateway and stopped clawdbot remain.
Command/package/environment/PATH checks clear, no matching process, PM2 list empty.
Report: tasks/remove-hermes-openclaw-result.txt; per-target proof: -target-verification.txt;
final focused checks: -verification.txt; full raw names: -wide-after.txt. The report
rechecks matched names after removal and groups the ERP log matches, while preserving
301 other-app/research/history/evidence name matches. The task is PARTIAL, not DONE;
no claim that nothing is left. The three administrator deletions were never retried.

## Phase 40 - Rebuild key-binding paths for this checkout (2026-10-24)

**Symptom:** all ten configured key commands targeted missing scripts in the former
D: toolkit folder, even though check.ps1 passed.
**Cause:** the master/live commands and several defaults/docs retained the former
checkout location. The saved pre-repair config confirms 10/10 missing targets.
**Fix:** changed master/live commands to I:\herdr, keeping headless conhost, arguments
and STA flags; changed folder-picker/layout/launcher defaults to their own toolkit
folder and the trial default to the sibling HERDR-trials folder. Rebuilt the launcher.
Updated skills and path-only docs/archived task references; historical old locations
are described as former locations. The current task contract was left unchanged.
Synced skills with installed-copy backups. No git, agents, browser, live reload,
pane changes, Claude settings, trust or approval changes.
**Backups:** live config:
C:\Users\mflma\AppData\Roaming\herdr\config.toml.20261024-121607-7139695.bak;
installed skills: SKILL.md.20261024-121609.bak beside both installed skill files.
**Proof:** target inspection printed every key/path: 10/10 Test-Path=True and parses=True;
master/live key table equality=True (ten blocks), and live-config comparison with its
backup confirms only folder replacement. Offline herdr.exe --session lab config check:
config: ok (no server started). Offline suites: batch A 48/48, batch B 18/18,
round2 18/18, HUD 11/11, dialog guard 26/26, headless 9/9 (130/130).
No new tests or git baseline were used for these path replacements; pre-repair config
provides the missing-target baseline. The first computed trial parameter default broke
the AST-only alias fixture (17/18); moving initialization after parameter binding restored
18/18 without changing tests. External trial-kit availability is not established by
choosing a portable default. Final setup and path-search results follow after verification.
**Not verified live:** no shortcut was pressed; the lead tests one key. The unchanged
current task text still quotes the former path as the problem/search instruction.
**Lesson:** #63.
**Final checks:** check.ps1 exit 0, 0 FAIL (four existing shortcut/PATH warnings).
Launcher ResolveFolder invoked offline from System32 returns I:\herdr; folder-picker
and layout default assignments evaluated with a moved toolkit folder return that folder.
The full toolkit path search prints only four lines in the unchanged task contract;
excluding that contract yields no matches. The live config search has no matches.
The expected sibling trial-kit folder is absent on this PC; no kit was created or run.
## Phase 41 - Sign-ins, Cursor profile, Antigravity on Argon High (2026-10-24)

**Symptom:** the owner moved to this PC and asked for the other tools to be signed in and set up:
Cursor with Composer high, Antigravity with Argon high.
**Fix:** signed in (owner did it in panes): Claude 2040 and 1984 (`CLAUDE_CONFIG_DIR` folders), Grok,
Cline Pass (`cline auth cline-pass`); Cursor (`cursor-agent status`: logged in) and Antigravity
(answered a test prompt) were already signed in. `agents.json`: `gemini` default model
`gemini-4-argon-medium` -> `gemini-4-argon-high`; new profile `cursor` (`cursor-agent --model
composer-2.5`; owner decision 2026-10-24, same as Phase 18: `--force` = run without approval prompts). Cursor has no
"Composer high": `cursor-agent models` lists only `composer-2.5` and `composer-2.5-fast`.
Read back: `herdr-agents tools` shows both; the cursor pane was detected by Herdr (`agent list`: cursor).
**Lesson:** #65.

## Phase 42 - Shortcuts and PATH restored on this PC (2026-10-24)

**Symptom:** `check.ps1` had four WARNs: Desktop and Start-menu `Herdr.lnk`, the Startup `Herdr HUD.lnk`,
and this folder not on the user PATH (SETUP steps 6, 8, 9 were never done on this PC).
**Fix:** created Desktop/Start-menu `Herdr.lnk` -> `launcher\HerdrLauncher.exe`, Startup `Herdr HUD.lnk`
and Desktop `Stop Herdr.lnk` (both `conhost.exe --headless pwsh ...`), added `I:\herdr` to the user PATH.
Backups (old shortcuts, raw PATH + kind) in `%APPDATA%\herdr\backup\shortcuts-20261024-122942`.
**Proof:** `check.ps1`: All checks passed, no WARN. Not tested live: clicking the shortcuts.
**Lesson:** first try failed on a parameter named `$args` (automatic variable, AGENTS.md rule 6).
## Phase 43 - Fix independent toolkit review findings (2026-10-24)

**Symptom:** setup checks missed broken script/shortcut targets and stale Claude account skills;
folder shortcuts bypassed profiles; HUD, trials and ownership handling could hide failures.
**Cause:** text/existence-only checks, pane IDs without server identity, ignored command results,
and silent read failures. Offline regressions reproduced the failures on HEAD copies.
**Fix:**
- Findings 1-4: check master/live key script targets and shortcut destinations; sync/check every
  existing Claude account whose skills exist or whose profile copy list requires them; use
  herdr-app for custom builds; folder shortcuts call herdr-agents start for flags and ownership.
- Findings 5-11: validate lowercase accounts and agent names before account creation; scope HUD
  records by normalized server identity; log background HUD errors; verify stop exit/result and
  explicit stopped status; show the confirmation owner; keep HUD -Once read-only; give missing
  trial kits a clear -TrialRoot error, check agent health while waiting, and clean up owned trial
  agents on failure; skip prompt-start polling during collect; reject unreadable ownership JSON.
  Validate agents.json before dispatch so the shared account reader cannot quietly call
  account 2030 'main' after a registry read failure; the shared reader stays outside this scope.
- Additional LOW 12: failed pane account-environment setup was ignored before agent start;
  abort with its command error. The new fixture proves it stops before continuing.
- Additional LOW 13: HUD discovery returned false, while -Once always exited zero; throw a
  clear discovery error so the one-pass command fails and the background loop logs it.
- Additional LOW 14: failed pane metadata writes were ignored; throw their command error,
  retain the last successful cache, and let the background error log record the failure.

**Proof:** all seven offline suites pass: batch A 48/48, batch B 18/18, dialog 26/26,
headless 9/9, Cline HUD 11/11, round2 18/18, review fixes 25/25 (155/155).
`tests/test-review-fixes.ps1 -SourceRoot <temporary HEAD copies>`: 0/25 pass, 25/25 fail;
current sources: 25/25 pass. The old copies were obtained only with the proof-required
read-only git show; no git mutations. The batch B model-only fixtures were updated for the
now-required shared profile start. Offline `herdr.exe --session lab status server --json`
returns status=not_running, confirming the stop read-back shape without starting a server.

**Not done / doubts:** check.ps1 reports 5 FAIL: four stale/missing account skill copies
(1984 and 2040, two skills each) and unavailable/broken App Path registration; user PATH
also warns. Account sync is explicitly reserved for the lead: run sync-skills.ps1, then
-Check and check.ps1. No external repair, live session, pane, build/install, shortcut click,
approval setting or sign-in was changed/tested. Shortcut destinations all exist; key targets
all exist. No claim that the setup check passes before the lead completes external repairs.
**Lesson:** #66, #67, #68.
**Cleanup limitation:** automatic approval review rejected removal of the proof-only temporary
baseline folder twice, including an exact LiteralPath attempt, stating only "blocked by policy".
Left tests/.review-old-32db5c5becec4590a36494f1668372b5 and tests/.review-baseline-path.txt for
lead cleanup; no further deletion method was attempted. Final review-fixes run: 25/25;
final check.ps1: 5 FAIL (the external issues listed above).

## Phase 43 - AutoSync to GitHub main (2026-10-24)

**Symptom:** the owner wants this folder kept in sync with `main` all the time, automatically.
**Fix:** new `herdr-autosync.ps1`: every 5 minutes (one instance, mutex) it stages changed files, skips
secret-like names/text and files > 5 MB, commits `autosync: N file(s) <time>`, `git pull --rebase
--autostash`, pushes. On a rebase conflict it aborts the rebase and logs CONFLICT; it never force-pushes,
resets or checks out; it does nothing on a branch other than `main` or during a merge/rebase.
Log: `%APPDATA%\herdr\autosync.log`. `-Once -DryRun` prints what it would commit; `-Status` shows the loop.
**Proof:** dry run listed 97 files; first real pass committed 97 files and pushed (`986d77bb`),
`git log origin/main..HEAD` = 0. Auto-start at sign-in is NOT installed: Claude Code's auto-mode
blocked creating the Startup shortcut and the hidden background loop (persistence) - the owner starts it.
**Lesson:** #66 (a function named `Git` calls itself: PowerShell names ignore case).

## Phase 44 - Shutdown handoff for the web/phone session (2026-10-24)

**Symptom:** the owner had to close the PC mid-project (Herdr Deck app) and continue from claude.ai on a phone.
**Fix:** both Deck builders were told to stop, finish their current edit and write status blocks
(`app/API.md`, `app/PERF.md`, `app/README.md`, `app/FEATURES.md`); the lead wrote
`tasks/HANDOFF-web-2026-10-24.md` (what was done, Deck state, ordered next steps, rules, PC commands),
ran `herdr-autosync.ps1 -Once` and checked `git log origin/main..HEAD` = 0.
**Lesson:** a handoff must name what cannot be done from the web (build, run, live tests) so the
web session writes code and docs and marks everything "not verified" until the PC is back.

## Phase 45 - Android phone workbench and radar (2026-10-24)

**Goal:** the owner's phone (Termux) as an all-day coding platform: herdr, the agent CLIs and
Hermes Agent in an Ubuntu proot workbench, pushing to GitHub from anywhere.
**Fix:** `mobile/termux/` - `setup.sh` (Termux side: proot-distro Ubuntu, `dev` launcher, Termux:Boot
server start), `workbench-setup.sh` (inside Ubuntu: Node 22, gh, herdr, Claude/Codex/Gemini/OpenCode/
Copilot, Hermes via its Linux installer, `dev-update` every 12 h, `ship` = commit + push with retries),
and `radar.sh` (self-updating scan, repair and redacted report to the private repo `<owner>/phone-radar`).
**Symptoms seen on the device and their causes:**
- proot `new_child: assertion failed` / signal 6 in herdr panes -> `PROOT_NO_SECCOMP=1` in `dev`, boot and radar (not yet confirmed on the device).
- `PREFIX: unbound variable` -> radar was pasted inside Ubuntu; radar now says "type exit" when `$PREFIX/bin/pkg` is missing.
- `proot-distro: .../python3.14: bad interpreter`, then `CANNOT LINK EXECUTABLE "curl"` -> radar installed one package on a
  stale Termux (partial upgrade). Recovered with `apt update && apt full-upgrade -y`; radar now does a full apt upgrade when
  proot-distro does not run, never a single-package install.
- `bash: /root/.workbench/setup.sh: No such file` -> radar wrote into `installed-rootfs/ubuntu` on disk, but the current
  proot-distro keeps the container elsewhere; files now go in through stdin of `proot-distro login`.
**Lesson:** on Termux, never install single packages on a stale system (always full-upgrade first), and never write into
a proot-distro container by its disk path. Hermes' own Termux APT package is marked broken upstream (2026-10), so Hermes
runs inside the Ubuntu workbench. Not verified on the device: herdr agent detection under proot, the Hermes reinstall.
**First device report (phone-radar, 2026-10-24 16:36 UTC):** herdr was never installed in the workbench (the
"herdr server running: yes" line was `pgrep -f` matching its own command; now `pgrep -x herdr`). Hermes answered
`--version` but could not update: `venv/bin/ty is owned by uid 10362, not the current uid 0` (files written from Termux
outside proot), then `ImportError: cannot import name 'YAML' from 'ruamel.yaml'` in its PM runtime. radar now runs
`chown -R 0:0 ~/.hermes/hermes-agent ~/.local/bin`, `hermes pm repair` and `hermes update`, and counts Hermes as
broken when its output says "repair failed". The Hermes chat provider (kilocode) returned HTTP 402 "Low Credit" with a
negative balance; the xAI fallback has no token. That is the owner's billing/provider choice, not something radar changes.
**Second device report (2026-10-24 17:00 UTC):** the workbench setup ran - herdr 0.9.5, Claude, Codex, Gemini,
OpenCode and Copilot installed, gh signed in, SSH key added. Hermes still crashed (`No module named 'ruamel'`), yet
radar printed "hermes: WORKING": `proot-distro login` (5.10.2) returns 0 whatever the inner command returns, so every
`if wb ...` check was always true. radar now asks yes/no questions through `wb_ok`, which prints a marker. `chown`
under proot's fake root did not change the uid Hermes sees, so a broken Hermes now gets a clean reinstall (data in
`~/.hermes` kept) with `hermes pm repair` only as a second step. The `dev` launcher was never written (only the
workbench half of setup ran); radar now runs `setup.sh --launchers`. The first `herdr` attach timed out after 15 s
while radar's own background `herdr server` was starting; radar no longer starts a server.
**Third device report (2026-10-24 17:48 UTC):** dev launcher and boot script installed; `herdr` still "did not become
ready within 15s". Cause: the client socket's host path is 114 bytes, over the 107-byte unix socket limit (LESSONS #70);
reproduced natively with herdr 0.9.5. Fix: `HERDR_SOCKET_PATH=/tmp/herdr.sock` in `/etc/profile.d` (radar installs it on
old workbenches), `--shared-tmp --no-kill-on-exit` in `dev` and the boot script, no duplicate `/sdcard` bind. radar now
probes a short and a long socket path in an isolated config dir and reports which one answers. Hermes still crashed
(`No module named 'ruamel'`) while `hermes --version` printed a version, so radar judged it healthy and skipped the clean
reinstall; radar now judges Hermes by `hermes status`.
**Fourth device report (2026-10-24 18:16 UTC):** herdr opens on the phone (owner screenshot: workspace "code",
welcome dialog). Correction: radar's socket probe answered on both a short and a 119-byte host path, so the
socket-length cause written in the previous entry was wrong; LESSONS #70 now says so. The fresh `/tmp/herdr.sock`
path still fixed it, most likely by avoiding a stale server on the default socket. Hermes: the clean reinstall
(v0.22.5) failed the same way (`cannot import name 'YAML' from 'ruamel.yaml' (unknown location)`), and radar again
printed "hermes: WORKING" because `command -v hermes && ...; ! grep` was true when hermes was missing. Fixed the check
(tested against stubs: missing, crashing, healthy). The reinstall now also moves the old Hermes uv cache aside
(it survived the first reinstall) and sets `UV_LINK_MODE=copy`; the scan reports foreign-owned files under
`~/.hermes` and the installed `ruamel/yaml/__init__.py`. Not yet confirmed on the device.
**Hermes root cause, found by Claude Code running inside herdr on the phone (2026-10-24 ~19:00 UTC):** after our full
Termux upgrade, Termux's `python3.14` (bionic) sat at the end of the workbench PATH. Hermes' installer prefers
Python 3.14 (Ubuntu 25.10 has 3.13), so `uv python find --system` picked Termux's interpreter; `pm/store.py`
`current_target()` then saw `sys.platform == "android"` and pinned the whole runtime to `linux-arm64-bionic`
inside a glibc Ubuntu, which ends in `cannot import name 'YAML' from 'ruamel.yaml'`. Fix: radar's workbench commands,
`workbench-setup.sh` and `dev-update` drop `/data/data/com.termux/*` from PATH; radar refreshes `dev-update` from main.
The uv-cache and ownership guesses in the earlier entries were not the cause.

## Phase 46 - Linux box port (2026-10-24)

**Goal:** run the owner's Herdr fork on a Linux PC with the same agent profiles and a
six-column team layout (claude, codex, grok, gemini, cline, opencode).
**Fix:** `linux/` - `herdr-agents` and `herdr-layout` (bash, `agents.json` + jq/node),
`install.sh` (symlinks into `~/.local/bin`, copies skills and `AGENTS.md` only when missing,
seeds `~/.config/herdr/config.toml` only when absent, `herdr integration install` for
installed CLIs; Gemini/Agy -> `antigravity-cli`). `HERDR_HOME` defaults to the repo root
(not `I:\herdr`). Verified: `bash -n` on all scripts, `install.sh --dry-run`, real
`install.sh` on the box, `herdr-agents tools`.
**Lesson:** #72.
**Review (2026-10-25):** `install.sh --dry-run` printed "copied ..." / "linked ..." for files it did not touch; those lines
now print only on a real run. Open points for the owner, unchanged: `install.sh` runs `herdr integration install` itself
(CLAUDE.md safety rule 4 says agents give that command to the owner instead), and it copies the repo's `AGENTS.md` to
`~/.codex/AGENTS.md`, which makes these folder rules Codex's global rules on that machine.

**Merge verification (2026-10-25, Windows):** merged PR #11 into current main,
preserving both documentation histories and assigning the incoming Linux lesson #72
because main already uses #70 and #71. Git Bash syntax checks pass for all three
scripts; installer and layout dry runs exit 0. No installation or agent launch ran.
Existing offline suites: batch A 48/48, batch B 18/18, dialog guard 26/26,
review fixes 25/25 and round2 18/18 pass. Headless is 8/9 (temporary path not found)
and Cline HUD is 5/11; these PowerShell sources/tests are unchanged by this PR.
The pre-merge main setup check has 28 failures: foreign I: key paths and master/live
mismatches (20), stale installed skill copies (6), and existing duplicate history/lesson
numbers (2). These machine setup and baseline issues were not repaired by this merge.
Linux runtime behaviour remains unverified on this Windows PC.

### Phase 46 follow-up - Phone workbench made lighter and faster (2026-10-25)

**Owner order:** "do everything yourself, develop the program a lot, make it light and fast on the phone".
**Fix:**
- `radar` measures the workbench once a week (200 commands with and without `PROOT_NO_SECCOMP`) and keeps the
  faster mode only when it wins by at least 30 %, because the crash-safe mode avoided a `new_child` assertion once.
  The choice lives in `~/.radar/proot.env`, read by `radar`, `dev` and the boot script; `radar safe` pins the safe
  mode, `radar fast` measures again. Tested against a stub proot (fast 433 ms vs safe 632 ms → fast).
- Quick scan by default: no socket probes (the socket-length idea was disproved), no disk-wide `find`, no
  `hermes doctor`/`dump`; those run with `radar deep`.
- herdr on the phone: radar appends `[advanced] scrollback_limit_bytes = 2000000` (was 10 MB per pane) and
  `[ui.sound] enabled = false` to `~/.config/herdr/config.toml` when those tables are absent; herdr 0.9.5 logged
  `pane_scrollback_limit_bytes=2000000` with that file here.
- `dev-update` runs at the lowest CPU/disk priority, the apt upgrade at most weekly, every 24 h by default (was 12).
- `linux/herdr-agents --help` prints usage instead of failing when herdr is missing.
- The owner's open points from the Phase 45 review stay as they are (owner-run installer, owner's choice).
**Not verified on the device:** the measured mode, the herdr config, the lighter updates.

## Phase 47 - C: cleanup by Sol; Deck round 3 restarted with DeepSeek Flash (2026-10-26)

**Symptom:** C: had 1.57 GB free of ~123 GB. The Deck round-3 work of 2026-10-25 never ran: the
frontend start failed with `EmptyResponse` and the backend was never started.
**Cleanup (Codex GPT-6.1-Sol medium, spec `tasks/sol-free-c-drive.md`):** it used its own
`c-drive-safe-cleanup` and `windows-disk-cleanup-guard` skills only (Inspect -> Preview -> Clean),
and kept the Recycle Bin step to C:. C: 1,681,948,672 -> 2,716,782,592 bytes free (Temp 957 MB, npm
cache 139 MB, crash dumps 51 MB); 30 locked/denied paths skipped. Proof: lead read
`~\.cache\disk-cleanup-guard\cleanup-log.txt` (DELETED/CLEANED lines match) and `Get-PSDrive C`.
Report with a read-only survey of the biggest folders (AppData 21 GiB, Downloads 6.4 GiB, .codex
3.3 GiB, .rustup 2.6 GiB - owner decides): `tasks/sol-free-c-drive-result.md`. Pane closed.
**Traps seen:** `ask codex -Fresh` hit Codex's new "where should the new conversation run" dialog
and reported DONE early (#73); a Cline self-update hid the `cline` command for two minutes (#74).
**Deck round 3:** owner chose ONE worker, DeepSeek V4.1 Flash (`ds-flash`, headless), with lead
review and Sonnet 5.5 high fixes. Phase 1 spec `tasks/deck-r3-ds-backend.md` (R19-R24, account
guard, named-session start, /fonts cache, `input_not_empty` guard with real ANSI fixtures in
`app/tests/fixtures/input/`, lesson #75). Baseline before it: `app\build.ps1` exit 0, exe 8,600,576 bytes.
**Phase 1 result (DeepSeek, answer `tasks/deck-r3-ds-backend-answer.md`):** DONE, items 4.0-4.11, 21
new tests; old code does not compile against them (30 missing symbols). It also fixed `build.ps1`,
which had a hard-coded Go path from another PC and exited 1 (#77), and raised the `/api/git` budget from
5 s to 15 s (#76).
**Lead review:** the lead's own `build.ps1` run failed in 2 of 4 full test runs:
`TestEveryCLIRequestHasACancelDeadline` -> `contain child process: Access is denied`. The bug was already
there (`herdr.go` unchanged): under load the 100 ms deadline killed the child between `cmd.Start()` and
`AssignProcessToJobObject`, and the job error hid the timeout (#78). Fix: `runCommand` reports the
timeout when `ctx.Err()` is set; 4 of 4 full runs pass. Second fix: in `input.go`, a basic foreground
colour (30-37, 91-97) now ends a grey run, so text typed in white after grey text is still seen; new
assertion in `TestTypedTextIgnoresDimTextAndKeepsSpaces` fails on the delegate's version. Final
`build.ps1`: exit 0, 8,657,920 bytes. Lab not run: every R3 route is unit-tested only.
**Lesson:** #73, #74, #75, #76, #77, #78.

## Phase 48 - Deck round 3 phase 2: lead finishes the code, Codex Astra tests, Sonnet watches (2026-10-28)

**Owner's choice:** the lead finishes the Deck itself; Codex `gpt-6-astra` medium verifies it live
(spec `tasks/deck-r3-codex-terra.md`, tester only, no product code); a Claude Sonnet 5.5 high helper on
account 1984 watches Codex and scores it (`tasks/deck-r3-sonnet-watch.md`, log
`tasks/deck-r3-sonnet-watch-log.md`). Build notes for the tester: `tasks/deck-r3-lead-notes.md`.
**Lead's changes (observed, harness `app/tests/deck-ui.ps1` = isolated APPDATA + headless Deck + one
headless Edge over CDP):**
- Sideways scroll at every width (66 px, even at 2560): the right-hand tooltip was centred past the edge;
  it now opens leftwards; counts compact below 1700 px. scrollWidth == innerWidth at 900/1100/1440/2560,
  dark and light; help icon visible; no toast over a visible card header (one bottom-right stack).
- `?theme=` was ignored (only the stored choice was read): now applied and remembered.
- Hard-coded `I:\herdr` defaults (the other PC) -> `State.root` from the backend (test
  `TestStateReportsTheToolkitRoot`).
- Size: HTML/JS/CSS now gzip-compressed with ETag revalidation (`TestStaticHandlerGzipsPageCodeAndRevalidates`):
  209,386 B raw -> 70,013 B transferred; budget redefined as < 150 KB transferred. Dead code removed.
- Cold first paint 3.4 s: Edge proxy detection (#79); the Deck window now uses `--no-proxy-server`
  (`TestDeckWindowSkipsTheSystemProxy`): 3,384 -> 876 ms in the harness.
- Stress 8 panes, 20 s: 46 -> 60 fps, long tasks 15 -> 0, render p95 17.9 -> 10 ms (#83). Target 8 ms
  not reached (scrollTop forced layout remains ~2.5 ms/frame).
- R3 wiring: `input_not_empty` toast with the server's sentence, never retried; dispatch id on the
  delivery-unknown card; reservation conflicts listed; client overlap = backend folder rule; Usage panel
  reads the real `herdr-agents usage` table.
**Codex repair:** the 2026-10-27 interrupted npm update had broken `codex`; npm cannot fetch the binary
through the proxy; a complete 0.157.1 set from `~\.codex\packages` was installed (#81, #82). Not updated
to 0.165.0.
**Lesson:** #79-#84.

## Phase 49 - Codex updated to 0.165.0: GPT-6.1-Sol works again (2026-10-29)

**Symptom:** `codex exec -m gpt-6.1-sol -c model_reasoning_effort=medium` on 0.157.1 failed with HTTP 400
`The 'gpt-6.1-sol' model is not supported when using Codex with a ChatGPT account.` (#85).
**Cause:** the old CLI version; 0.165.0 (npm latest) accepts the model on the same account.
**Fix:** backup of the 0.157.1 package in the lead's session scratchpad; `npm install -g
@openai/codex@0.165.0` (installed the Windows binary package this time); its files were hash-compared
with a ranged, integrity-checked download of the same tarball: 0 differences (#86).
**Proof:** `codex --version` -> `codex-cli 0.165.0`; headless `codex exec ... gpt-6.1-sol medium` -> `OK`,
exit 0; `herdr-agents start codex -Model gpt-6.1-sol -Effort medium` -> STARTED pane wV:p3, `ask` ->
`SOL-LIVE-OK`, no error line, footer `GPT-6.1-Sol medium`.
Same day, owner's request: every scheduled task of `Computer Use\opening-nerp-tcode` (22 GMES_* /
MF_Ops_* tasks; 5 were active, MF_Ops_RemoteControl was running) was stopped and disabled "for now".
**Lesson:** #85, #86.

## Phase 50 - Deck round 4: Astra review, first fix batch (2026-10-29)

**Owner's request:** the Deck needs big improvements; Codex `gpt-6-astra` high reviews everything (code,
web best practice, GitHub projects) and writes only what must be done; then fix. Spec
`tasks/deck-r4-astra-review.md`, report `tasks/deck-r4-astra-report.md` (D1-D25, P1-P5, I1-I4, order of
work; five backend gaps reproduced with temporary overlay tests, JS logic with Node probes).
**Fixed by the lead (owner-visible problems first):**
- D1 **Symptom:** an exited agent's card sent the owner's message into the bare PowerShell left behind
  (`ParserError`, screenshot 2026-10-28 17:40). **Cause:** `server.go` prompt action switched to
  `pane run` whenever the fresh snapshot showed no agent. **Fix:** `/api/pane/prompt` takes `expect`;
  agent text to a shell -> 409 `agent_exited`, nothing written; the page sends `shell` only from a card
  that never had an agent and shows "Agent exited · messages are not sent to the shell" (API.md R25).
  **Proof:** `TestPromptToExitedAgentNeverRunsInTheShell` passes; on HEAD it fails with
  `writes=[pane|run|w1:p1|make text reflow [see screenshot]]`.
- D7 **Symptom:** card screens scrolled sideways (wide pane) or left half the card empty (narrow pane).
  **Cause:** `screen.js` only scaled the font (9 px .. 1.25x) to the pane's fixed column count; rows were
  `white-space: pre` with `overflow: auto`. **Fix:** text reflows: each line is cut into display rows of
  the card's width (`ansi.js parseRows`, wide CJK/emoji = 2 cells, rule lines of box characters cut, not
  wrapped); virtual scrolling counts display rows; the font never shrinks (grows at most 1.25x for a
  narrow pane); `.screen` has no sideways scroll; font metrics are re-measured after the web font loads.
  A narrow Herdr pane still gives narrow text: the agent breaks its own lines at its PTY width and Herdr
  cannot set a pane's columns (Astra, `pane resize --help`).
  **Proof:** headless harness, mock page at 900 px: a 400-char line became 13 rows, widest row 382 px in a
  387 px screen; every screen `scrollWidth <= clientWidth` at 900 and 1440 px.
- D9 a full snapshot now always sets the screen version (a resync with unchanged text repairs it).
- D11 reading older output: dropped top lines move the scroll position up by their display rows.
- D14 names cut to `clau...`: an empty `.grow` spacer in the card header took half the free width.
  Removed; below 560 px card width the status shows as its dot and the context ring hides.
  **Proof:** 4/4 mock card names uncut at 900 and 1440 px (before: `claude-2040` 76 of 92 px).
- D21 a clipped long line no longer leaks its colour into later lines; D22 Save-snippet textarea gets the
  draft (`ui.js` sets `.value`); D23 `/foo/g` search finds every line; D25 unknown sign-in shows "unknown".
  **Proof (JS):** Node probe importing the real `screen.js`/`ansi.js`/`search.worker.js`: 5/5 pass; on HEAD
  4/5 fail (wrap rows 2, top 400 not 200, next row red, 2 of 4 hits).
**Checks:** `go vet` 0; `go test ./...` ok; `build.ps1` exit 0, 8,673,280 bytes.
**Not verified:** the exited-agent placeholder in a browser (mock cannot exit an agent); live lab run.
**Open (report order of work):** D2-D6, D8, D10, D12, D13, D15-D20, D24, P1-P5, I1-I4.
**Lesson:** #88.

## Phase 51 - Deck round 4 phase 2: Astra (backend) and Sol (frontend) fix the whole report (2026-10-29)

**Owner's request:** "give batches to Astra and Sol and fix all". The lead split by file (Astra
`gpt-6-astra` high: `app/internal/deck/*.go` + API.md; Sol `gpt-6.1-sol` medium: `app/web/*` + Node
tests) and wrote the shared contracts first (`tasks/deck-r4-contracts.md` C1-C10); specs
`tasks/deck-r4-astra-backend.md`, `tasks/deck-r4-sol-frontend.md`; answers `tasks/deck-r4-*-answer.md`.
**Fixed:** C1 per-page visibility (SSE `hello` client id, union reads, metadata heartbeat, stale panes when
Herdr is offline), C2 `settings` event, C3 `/api/pane/raw` ordered per-pane typing, C4 reserve before
dispatch, C5 one launch admission for every route + `pane_id` in start replies, C6 typed timeouts, C7
`input_unknown`, C8 `/api/prefs`, C9 drive of every launch folder + null unknowns, C10
`/api/debug/counters`; D6, D8, D10, D12, D13, D15 (menus in a body popover), D16, D17, P1-P5. API.md R26-R35.
**Lead's own changes:** the Go lock (`app/tests/go-lock.ps1`, `go-locked.ps1`, `build.ps1 -Output`; two
agents never build at once), `agent_exited` checked before the screen (an old dialog on screen hid the
real reason), "G: unknown" wording, `verify-lab.ps1` updated for R25/C7 and extended with the R4 checks,
agent stub prints Codex's `›` input box.
**Proof:**
- `go vet` 0; `go test ./...` ok; Astra's 35 new tests fail on HEAD (35/35); `node --test app/tests/web/`
  30/30, Sol's tests fail on HEAD 30/30 (mostly missing functions, Sol says so).
- `build.ps1` exit 0, 8,771,072 bytes.
- Isolated lab (`verify-lab.ps1`): 17/17 PASS incl. hello first, raw ordered, prefs, counters, exited
  agent -> 409 `agent_exited` with nothing run, settings broadcast to a second page, offline keeps 17
  stale panes, recovery after restart.
- Real page on the lab backend (headless harness): 16 cards; Herdr stopped -> 16/16 composers disabled
  + banner "Herdr offline - panes are stale; writes disabled"; restarted -> recovered; no JS errors.
- Mock page: no JS errors, no sideways scroll, no cut names, the More menu portals to body, fits the
  viewport and closes on Esc.
**Not verified:** IME composition and two real pages typing raw at once in a visible browser; one-hour
soak / handle slope (counters now exist for it); race detector. I2 (Herdr lifecycle events) and I4
(xterm.js comparison) not attempted.
**Lesson:** #89, #90, #91.

## Phase 52 - Deck round 5: cards fill the window, every inner area resizable (2026-10-29)

**Symptom:** at 1900x1030 two cards were ~440 px tall with the lower half of the window empty and the
status bar floating in the middle (owner's screenshots 12:29 and 12:33); nothing could be resized in
grid view; the cards showed a horizontal scrollbar and cut lines.
**Cause (proven in `tasks/deck-r5-shots/before-*.png`):** `body` was a CSS grid `auto auto auto 1fr auto`
for header, banner, rawbar, broadcast bar, `.body`, status bar. The three bars are `hidden`
(`display:none`), so auto-placement moved `.body` into an `auto` row and the empty `1fr` row ended up
under the status bar. Card rows were then sized from `100vh` formulas (a `min-height:1000px` media query
halved them for an assumed second row). The horizontal scrollbar is from the owner's running exe:
`app\HerdrDeck.exe` is dated 2026-10-24 and predates the round-4 reflow (`.screen` has
`overflow-x: hidden` and wraps long lines since round 4) - it needs a rebuild.
**Fix (`app/web`):** `body` is a flex column and `.body` takes the rest; `.main` is a flex column and the
grid fills it; rows share the height (`minmax(0, Nfr)`), `--grid-min` makes many rows scroll instead of
collapsing. New `split.js`: dividers between card columns and rows (grid view), on the rail edge and on
the drawer edge; drag (window listeners, one write per frame), arrow keys (Shift = 4x), Home or
double-click resets; minimum card 360x240 px, rail 140 px..50%, drawer 280 px..window-360; sizes
saved in LS (`railW`, `drawerW`, `gridSizes` per column count and per row count). Screen refits on a
width change at most every 100 ms (`ScreenView.fitSoon`). Herdr layout view and its own dividers are
unchanged; it now also fills the height. `mock.js`: `&prefs=` and `&drag=` for headless screenshots.
**Not possible from Deck:** sizing the real Herdr pane/PTY to a card (owner's addition): Herdr's
public CLI/API only moves a split line (`pane resize --direction --amount`); the tab size comes from the
attached Herdr client terminal (`ClientMessage::Resize`, client protocol) (#95). Deck wraps the text
to the card instead.
**Proof:** `build.ps1` exit 0 (go fmt/vet/test ok); `node --test` web tests 35/35 (5 new in
`app/tests/web/split.test.mjs`; on HEAD that file fails: `split.js` missing); `verify-ui.ps1` lab +
mock: HEAD 37 PASS / 9 FAIL; this branch 38/8, then 39/7 with the scoped selector. Every remaining
failure also fails on HEAD (spaces join, usage panel, tools drawer, raw typing, broadcast, typed-input
guard, divider resize; git paths and prompt history fail on some runs). "Divider mouse resize": on HEAD
no divider existed (the layout grid had no height); now the divider exists and a real mouse drag calls
`api.resize(key, "right", 9)` (mock probe), but the lab pane geometry does not change - Herdr side, not
verified further. Its selector `.divider.v` also matched the new rail handle, so it is scoped to
`#grid .divider.v`. Screenshots:
`tasks/deck-r5-shots/before-*` and `after-*` (1900x1030 and 1280x720, 2 and 4 cards, rail on/off,
after drags, drawer drag, Splits view, double-click reset).
**Lesson:** #94, #95, #96.

## Phase 53 - Deck round 5 phase 2: one organised navigation (2026-10-29)

**Owner's request:** "the navigation system is bad and buggy and not arranged; it needs to be
professional and organized". Audit with every element, every bug and before/after evidence:
`tasks/deck-r5-nav-audit.md`.
**Symptom / cause (measured by a probe in the headless harness, mock page):** at 1280x720 the top bar
was 98 px too wide, so Commands, Help and Safe were off screen (no breakpoint between 1279 and 1500 px
removed enough); at <= 1100 px CSS hid + Shell, Folder and Team; the state chips were spans styled as
buttons; panel buttons had no open state; closing a panel left focus on `<body>`; the ? key opened a
keys-only panel, the ? button another one; the rail kept the tab row active while an agent was zoomed;
the Agents hub used a shield icon and the Layout builder the "columns" glyph.
**Fix (`app/web`):** a 48 px bar in four groups - Create (Agent, Shell, Folder, Team) | View (1 2 3
Splits, Layout builder) | Status (state chips as buttons -> Inbox / Agent board, Inbox, Usage) ... Tools
(Search, Commands, More, Safe). Labels fold into icons in steps (1700/1560/1440/1240/960 px) instead of
disappearing. A static More menu (`#more-menu`, `ui.bindMenu`: arrows, Home/End, Esc returns focus) holds
Agent board, Agents hub, Tools, Spaces side by side, Settings, Keys and guide, Stop everything (ids kept).
`[data-drawer]` buttons are `aria-pressed` while their panel is open; `closeDrawer` returns focus to the
opener (a menu item -> the More button) when focus was in the panel; one help panel for button, ?,
Ctrl+B ?, Ctrl+B i; the zoomed agent row is the one active rail row; rail toggle has `aria-pressed`.
**Proof:** probe before -> after at 1900x1030, 1280x720, 1100x720: clipped [Commands, Help, Safe] -> [],
top-bar overflow 98 px -> 0, hidden actions [Shell, Folder, Team] -> [], chips SPAN -> BUTTON, panel
pressed null -> true, focus after Esc BODY -> btn-usage, help titles differ -> same, rail agent active
false -> true, menu item click closes the menu and opens its panel, focus back on More; the Splits
divider still calls `api.resize` (before and after). `node --test` 38/38 (3 new in `nav.test.mjs`, all 3
fail on the phase-1 commit). `verify-ui.ps1` 38 PASS / 8 FAIL; all 8 also fail on HEAD 764efb69 (divider resize, spaces join, usage panel, tools drawer, prompt history, raw typing, broadcast, typed-input guard); the 16 layout checks (900-2560 px, both themes) pass. Shots `tasks/deck-r5-shots/nav-before-*`, `nav-after-*`.
**Lesson:** #97.
## Phase 54 - Cline agents get panes again (2026-10-29)

**Symptom:** the owner asked for Cline MiniMax M3.1 (high) to take over a coding task "with a pane".
**Cause:** owner decision; the 2026-10-23 rule kept Cline headless only (#47).
**Fix:** CLAUDE.md section 0 now lets Cline agents have panes beside the lead, like Codex/Antigravity/
Claude. `herdr-headless.ps1` stays for one-shot runs. Agents are started with `herdr pane run`
because `herdr-agents start/team` still fails with `agent_pane_busy` (#60, #89).
**Lesson:** #89.

## Phase 55 - Deck round 6: calm restyle, light + dark theme that follows Windows (2026-10-29)

**Owner's request:** "the user interface is ... not professional and ugly; make it simple. I want this
design and style and font, light and dark theme" (reference screenshot 2026-10-29 17 26 39: charcoal
cards, hairline borders, round coloured icons on a thin timeline, kebab menus).
**Symptom (round-5 shots):** lime-green (#D4FF3A) buttons, accent borders and underlines on every
pressed control, loud amber "NEEDS YOU" block, mono-font chips, two rows of key and icon buttons on
every card, a custom display font (Alexandria).
**Fix (`app/web`):** new tokens on `:root` only (dark #141416 page / #1F1F22 cards / #2C2C30 lines,
light #F5F5F7 / white / #E4E4E8), 12 px card radius, one calm blue (#3B82F6 / #2563EB) for Agent and
Send only; system font (Segoe UI Variable), mono only in the terminal; ANSI palettes tuned per theme.
Top-bar buttons are ghost buttons, the view switch is a segmented track. Cards: round agent icons
(the only colour in a header), quiet chips, status = dot + text, "Needs you" a soft amber pill; split
buttons and the broadcast box show on hover/focus; the `...` card menu moved into the header; the key
row (Esc, Ctrl+C, ...) and copy/rename/close show only while the card has focus (all also in `...`).
Rail = a timeline: every status dot in one column on a thin line. Theme: `system` (default, follows
`prefers-color-scheme` live) / light / dark / high contrast; a top-bar button cycles System -> Light ->
Dark (saved with `LS`); Settings and Ctrl+K have "Theme: system".
**Bug fixed on the way:** a card's `...` menu closed whenever ANY element scrolled (window scroll
listener in capture mode), so a streaming pane closed it at once; now only a scroll that moves the
trigger closes it (`ui.js openMenu`).
**Proof:** `build.ps1 -Output app\HerdrDeck.r6.exe` exit 0 (go test ok); `node --test` 38/38.
Shots `tasks/deck-r6-shots/r6-{dark,light}-{2,4}cards-{1900x1030,1280x720}.png`, `*-more-menu-*`,
`*-card-menu-*` (mock page; menus shot over DevTools because `--screenshot` resizes the window and a
resize closes menus). `verify-ui.ps1` not run for this phase.
**Lesson:** #101.

## Phase 56 - Deck round 6 phase 2: Herdr panes take the card width (2026-10-29)

**Symptom (owner's screenshot 2026-10-29 17 29 44):** the CLI text filled only the left third of a
wide card; the rest was empty.
**Cause (proven in the lab):** with no Herdr terminal attached, Herdr keeps every tab at 120x40, so two
panes are 60 columns each (a PowerShell pane printed `57 columns x 38 rows`). Only an attached client
sets the size (#95).
**Fix:** Deck joins Herdr as a size-only client. `app/internal/deck/sizer.go` (+ `sizer_windows.go`
pipe dial) speaks the stable endpoint hello on `herdr-client.sock`, sends `ClientShellResize` only when
the size changes, reads and drops everything Herdr streams back, reconnects at most every 5 s and
detaches on exit; `POST /api/size {cols,rows}` (not in read-only mode). Page: `web/herdrsize.js`
measures the visible cards' screens in cells at the base font (first card row = columns, first card
column = rows, + Herdr's 3x2 pane frame), debounced 300 ms, sent only on change; a zoomed card sends
nothing. Measuring at the base font means the fit-to-width font never changes the answer: no loop.
**Grid view choice:** Deck does not rearrange the owner's Herdr splits. It sizes the tab area from the
first row of cards, which matches exactly when the cards show one tab (rail tab filter or Splits
view). In "All panes" with tabs of different shapes, a tab whose pane count differs from the card
columns gets a wider or narrower pane, and Deck's own reflow wraps it as before.
**Second Herdr window (from Herdr's code, not run live):** each tab follows one client; the first
viewer controls it until another client claims it (typing or focusing there). Deck never answers a
Herdr size change, it only re-sends when its own card area changes, so the two cannot fight in a loop;
the owner's terminal window may resize the panes until Deck's window is next resized.
**Proof:** lab only (isolated APPDATA, `herdr --session lab`), real Claude Code in pane 1 (left at its
folder-trust question, not answered), PowerShell printing its size in pane 2: before 57x38; after the
Deck page loaded at 1900x1030: tab area 212x37, panes 106 wide, PowerShell `103 columns x 35 rows` =
the card's 103 cells; at 1280x720: `62 columns x 20 rows`. Shots
`tasks/deck-r6-shots/r6-phase2-claude-fills-card-1900x1030.png` (dark) and
`r6-phase2-claude-fills-card-light-1280x720.png`. `go test` ok (2 new TestSizer*: fake Herdr checks the
hello JSON, the exact resize bytes, no frame for an unchanged size, a refusal and the retry window);
`node --test` 42/42 (4 new in `herdrsize.test.mjs`); both new test files cannot run on the phase-1
commit (`sizer.go`, `herdrsize.js` missing).
**Lesson:** #95 (rewritten), #102.

## Phase 57 - Goal engine: ledger, Supervisor, Watchdog, merges, `herdr-goal` (2026-10-30)

**Symptom:** the owner's loop was still "type continue": no part kept a goal going, sensed a stuck or
blocked agent, recovered after a crash, or proved DONE. The M1 work of the earlier delegate
(`I:\herdr\.wt\goal-m1`, branch `goal-engine`) was never pushed; this cloud session could not see it, so
the engine was built here from the spec and the M1/M2 fix lists (`tasks/goal-m0-m1.md`, `goal-m2.md`).
**Cause:** missing feature (spec M1/M2).
**Fix:** `app/internal/goal` + `app/cmd/herdr-goal` (Go standard library; docs/GOAL-ENGINE.md):
- Ledger: one atomically written `goal.json` + append-only `events.jsonl`; an inbox (`inbox.jsonl`) is
  the only way the lead/owner change a running goal; intents written before every outward action and
  reconciled after a crash (no lost or doubled start); a runner lock with heartbeat (a second runner
  exits, a dead one is replaced).
- Messages with ids: a brief file per agent, an ack file as receipt, a result file checked against the
  message id. Idle is not done; "done" without a commit is rejected.
- Watchdog (`prompts.json`, one line per known prompt): secret/sign-in never answered and re-checked
  before every send; trust answered only inside the project (the "Yes" option, also when "No" is
  highlighted); Cline pop-up and queued text; menus to a decider command or the CLI default (recorded);
  capacity -> "continue" then another model; account limit -> next account, next profile, then
  `waiting-limit` and resume at the reset (never the owner); dead -> restart in the same worktree with the
  work committed; launch check, stopped-early and stall (hash without timer/spinner/digits) with capped
  nudges; lead handoff when its context is low.
- Scheduler: dependencies, file reservations, `max_agents` 2 and free RAM/disk gates, writer by the
  goal's record, reviewer always another vendor, last attempt to another vendor, failed tasks to the lead.
- Git: worktree per task, `--no-ff` merges as checkpoints into `goal/<id>`, project tests after each merge
  (a failing merge is undone), conflicts back to the writer, `revert` of one task keeps later merges,
  DONE only after the Supervisor's own acceptance run and an independent review pass per task.
- `herdr-goal keep`: restarts a killed runner by itself (keep.sh: 3 kills -9, 3 restarts, DONE).
- Backends: Herdr panes (`pane split` + `pane run`, pane ids kept in `panes.json`) and detached
  processes (`herdr-goal relay`). Skill `skills/herdr-architect` (not synced: `sync-skills.ps1` is the
  owner's PC). Example profiles `docs/goal-profiles.example.json`.
**Bugs found by the e2e runs and fixed:** prompt answered 3 times from a log screen (#103); timeout check
after `cancel()` (#106); goal acceptance run after partial merges looped (#105); `__pycache__` blocked
merges (#104); merge failures had no attempt cap; lead nudged on every event; relay refused under a
delegate role; an agent whose CLI cannot start was restarted every tick for ever (found in self-review:
120 starts in 60 ticks before, now an attempt failure and a capped lead, test `TestAgentsThatCannotStartAreCapped`).
**Proof (Linux cloud container, scripted agent stand-ins, no Herdr):** `go test ./internal/goal` 33 tests
pass; `app/tests/goal-mutations.py` 24/24 rule mutations caught; `node --test` 42/42; `GOOS=windows go
vet ./...` clean. `app/tests/goal-e2e/run.sh`: 5 tasks, 2 agent kinds, runner and lead killed -9,
second runner refused, trust/crash/idle/limit/capacity/review-reject/merge-conflict all recovered:
DONE in 51 s, acceptance re-run green by the script, 0 agents left. Soak: see below.
**Not proven:** real CLIs and real Herdr panes (#108), Windows run, `check.ps1`.
**Lesson:** #103-#108.

## Independent audit preparation (2026-10-30)

Separate branch from cd846676: 18 source-inspection findings, six unexecuted behavioral regression specifications, five pinned upstream skill candidates with licenses and adoption policy, and a Linux/Windows CI evidence workflow. No engine production behavior changed. Go is absent locally and download failed. New tests must run before any passing claim. Candidates are not activated. Real-agent integration remains unverified.
