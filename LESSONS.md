# Lessons

Durable facts about Herdr, the agent CLIs and Windows, learned on this PC.
Numbered so HISTORY.md and the skills can point at them. Add, never renumber.

1. **Old shells do not see new PATH entries.** A program started before the installer
   (Explorer, Claude Code's shell, Herdr's server and its panes) keeps its old PATH.
   Use `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe`, not bare `herdr`, in scripts.
2. **Explorer's address bar and Win+R use App Paths.** `HKCU\...\App Paths\<name>.exe`
   is read live; PATH changes need a sign-out.
3. **Claude Code's auto mode refuses to change Claude Code itself** (`~/.claude/settings.json`,
   Herdr integrations), and may then refuse further herdr calls in that turn. Give the
   owner the exact `!` command instead.
4. **`[[keys.command]]` runs through `cmd.exe /d /c` on Windows**, detached, with no
   console. Use TOML literal strings (`'...'`) for paths with spaces; a dialog needs
   `pwsh -STA` and a top-most owner window or it opens behind the terminal.
5. **Default prefix keys that are taken:** `prefix+o` (open notification),
   `prefix+shift+t` (rename tab), `prefix+shift+w` (rename workspace). Check
   `src/config/model.rs` defaults before binding.
6. **Herdr's CLI answers in JSON:** `workspace create` -> `.result.root_pane.pane_id`;
   `pane split` -> `.result.pane.pane_id`; errors are JSON on stderr with exit 1,
   syntax errors exit 2.
7. **`agent read` pads with empty rows.** The answer can sit above dozens of blank
   lines; strip them before showing or searching.
8. **Herdr's menus are hard-coded** (global menu, space/tab/pane right-click). Adding
   items needs a source change; there is no plugin hook for menus in 0.9.5.
9. **Building Herdr on Windows needs MSVC Build Tools (`link.exe`) and Zig 0.16.0**
   (for the vendored `ghostty-vt`). `rustup` alone is not enough.
10. **The server<->client protocol is bincode.** Adding a field to a wire struct breaks
    compatibility with official builds; keep custom features client-side.
11. **Herdr's Windows binary is unsigned** and Windows support is beta. Corporate
    endpoint protection may block or flag it; do not work around a block.
12. **Cline auto-approves every tool by default** (`--auto-approve` default true) and
    shows a "Cline Desktop" promotion on first start (any key closes it). Start it
    with `--tui --auto-approve false` when another agent drives it.
13. **Model flags differ:** codex `-m`, claude `--model`, opencode `-m provider/model`,
    grok `-m`, agy `--model`, cline `-P <provider> -m <model>` (provider key via
    `cline auth <provider>`, typed by the owner).
14. **Herdr detects these kinds:** pi, claude, codex, gemini, cursor, devin, agy, cline,
    omp, mastracode, opencode, copilot, kimi, kiro, droid, amp, grok, hermes, kilo,
    qodercli, qwen, letta, maki, muse (`herdr agent` prints the list).
15. **`herdr server reload-config` applies `config.toml` live**; unknown keys produce a
    "has unknown keys" notice but the rest still applies.
16. **Herdr's own agent skill** (`herdr --skill`, copy in `herdr-skill.md`) is the
    authority for CLI syntax; it says agent commands require `HERDR_ENV=1`.
17. **`pane split --ratio R` gives the ORIGINAL pane the share R** (0.25 on a 254-wide
    pane -> 64 | 190). Equal N columns: split with 1/N, then the new pane with
    1/(N-1), ... 1/2. Repeated plain `ctrl+b v` halves each time (1/2, 1/4, 1/8 ...).
    `herdr pane current` with no argument returns the UI-focused pane.
18. **`pane move <id> --tab T --split right --target-pane P` moves a running agent
    without stopping it**; it keeps its agent name but gets a new pane id
    (`.result.move_result.pane.pane_id`). Moving the last pane out of a space closes
    that space. To put a moved pane exactly in a placeholder's place: move it next
    to the placeholder, then `pane close` the placeholder - the sibling takes its rect.
19. **`herdr worktree create` crashed the server once** (2026-10-22, Windows error
    0xc0000409, 3 s after a successful create, no log line): every pane and agent in the
    session died. Not reproduced in 3 tries in an isolated session. It also opens TWO
    spaces when the repo has none (repo + worktree). Make worktrees with plain
    `git worktree add` and open them with `workspace create --cwd`.
20. **Isolated test session:** `herdr --session <name> server` runs a separate headless
    server with its own socket (`%APPDATA%\herdr\sessions\<name>\herdr.sock`);
    `HERDR_SOCKET_PATH` points every herdr CLI call (and our scripts) at it;
    `herdr --session <name>` attaches a window; `herdr --session <name> server stop` ends it.
21. **`agent prompt --wait` can return `agent_prompt_stalled` although the prompt was
    delivered and answered** (seen on Codex's first prompt, two prompts in parallel):
    Herdr saw no `working` within 5 s. Its state can lag: right after, `agent get` may
    still say `idle` while the agent works, and `agent wait` then returns at once. Poll
    `agent get` until it shows `working`, then `agent wait`, then `agent read`.
22. **`[ui.toast] delivery = "system"` = Windows notifications** for agents that finish or
    need attention in a space/tab you are not looking at (the active tab is suppressed);
    the default is off. Sounds for background agents are on by default (`[ui.sound]`).
23. **Codex asks "Trust this folder?" once per repository**; for a git worktree it asks
    about the main repo root. A worktree agent of a new repo therefore starts BLOCKED
    until the owner answers in its pane.
24. **`pane move <id> --new-workspace --label <text>` gives a pane its own space again**
    (agents keep running). Workspace ids are never reused: a space rebuilt this way has a
    new id, so find spaces by label, not by a remembered id. Test sessions work headless: once,
    right after the server started, a split failed with `ghostty error -1` - wait for
    `status: running` and retry. Never open a window on a test session from inside a Herdr
    pane: Herdr refuses nesting ("nested herdr is disabled") and the owner sees an error window.
25. **The sidebar is configurable** (`[ui.sidebar.agents] rows`, `[ui.sidebar.spaces] rows`):
    built-in tokens `state_icon`, `workspace`, `tab`, `pane`, `agent`, `branch`, `git_status`
    ..., plus `$name` custom tokens set by `pane report-metadata --token name=value`; a missing
    token is left out. The agent panel's default first row is the SPACE name, so every agent
    in one space shows the same name. Space rows: `branch` = the folder's git branch (the
    theme colours it; gruvbox = red, not an error), `git_status` = up N commits not pushed,
    down N commits on the remote not pulled (hidden when both are 0).
26. **A pane's top border shows its metadata title** (`pane report-metadata --title`), else
    its own name (`pane rename` / ctrl+b shift+p). The sidebar `pane` token uses the same
    order, so put names in a `$token`, not in `pane`. Metadata with `--ttl-ms` disappears
    by itself when nobody refreshes it.
27. **A pane's own name is `label` in `pane list` / `pane get`** (absent when unset);
    `pane layout` does NOT return it.
28. **Agent usage is in the CLIs' session logs**, found by Herdr's `agent_session.value`:
    Codex `~\.codex\sessions\**\rollout-*-<id>.jsonl` (`turn_context` model + effort,
    `token_count` last input = context, `model_context_window`, total output); Claude
    `~\.claude\projects\*\<id>.jsonl` (usage per reply, written on several lines with one
    message id - count once). Read usage fields only.
29. **Codex effort: `-c model_reasoning_effort=<level>`** (levels in `~\.codex\config.toml`
    `enabled-reasoning-efforts`). Its footer says `default` until the first turn, then the
    real level (`GPT-6.1-Sol medium`); `herdr pane process-info` shows the real argv.
30. **Claude's dollar cost exists only inside Claude Code**: the status line command gets it
    on stdin (`cost.total_cost_usd`, `session_id`); the session log has tokens, no cost.
31. **Codex brings its own environment rules into our code** (it used a Chrome-launcher
    script from its skills for "open in browser"). Name the exact API in the spec and
    review for outside dependencies.
32. **Antigravity (`agy`) starts in `accept-edits` mode** (edits files without asking) and asks
    "Do you trust the contents of this project?" once per folder. `--mode plan` keeps it
    read-only. `agy -p "<prompt>" --model <id>` is a quick headless health check.
33. **"Installed" is not "usable"**: on 2026-10-22 OpenCode's OpenAI sign-in had expired (401)
    and its Zen models returned an error page, Cline's provider needed re-authentication and
    its DeepSeek endpoint returned an HTML page (network block), Grok was not signed in. Check
    with one tiny prompt before planning work on an agent (docs/AGENT-ROSTER.md section 6).
34. **Approval and trust are separate.** Auto modes (Claude `--permission-mode auto`, Codex
    `-a never`, Cline `--auto-approve true`, Antigravity `--mode auto`, Grok `--always-approve`)
    stop tool prompts, but Codex and Antigravity still ask once per repository/folder whether
    to trust it (Codex: `[projects.'<path>'] trust_level = "trusted"` in `~\.codex\config.toml`).
    Claude Code's classifier will not let Claude itself turn other agents into auto-approvers
    or edit trust settings - the owner runs that change.
35. **Antigravity `--mode auto` still asks before shell commands**; only
    `--dangerously-skip-permissions` runs everything without asking. Its permission prompt
    can chain several commands with `;` in one request.
36. **Closing Herdr's window only detaches**: the server is its own process (no living parent)
    and keeps every pane and agent running. Only `herdr server stop`, sign-out or a restart
    end them; after a restart Herdr restores the layout and resumes Claude/Codex sessions.
37. **Cline Pass model ids need the `cline-pass/` prefix** (`-P cline-pass -m cline-pass/glm-5.3`);
    a bare id fails with "Expected format: modelType/model". A retired model answers 404 with
    its replacement's name. Each headless `cline` run spends ~60 s before the first token.
38. **Separate accounts per CLI** (verified 2026-10-22: each reported "not logged in" in an
    empty folder while the normal sign-in stayed): Claude `CLAUDE_CONFIG_DIR`, Codex
    `CODEX_HOME`, Grok `GROK_HOME`, OpenCode `XDG_DATA_HOME` + `XDG_CONFIG_HOME`, Cline
    `--config <dir> --data-dir <dir>\data`. Antigravity: `ANTIGRAVITY_GEMINI_DIR` did NOT
    separate its sign-in - no account support. Herdr gives a pane its own variables with
    `pane split --env K=V` / `workspace create --env`.
39. **No console pop-ups:** Herdr runs `[[keys.command]]` through `cmd.exe /d /c` without a
    console, so a `pwsh` started there opens its own console window, and `-WindowStyle Hidden`
    still flashes it. A program started hidden (VBS `Run ..., 0`) also gets its FIRST window
    hidden - dialogs stayed invisible, even after showing a throwaway window first. What works:
    `conhost.exe --headless pwsh ...` - no console at all, and dialogs show normally.
40. **Windows command-line quoting:** in `"..."`, a backslash run followed by `"` is halved
    and `\"` is an escaped quote, so `"D:\"` swallows the closing quote and breaks the line.
    Double the trailing backslashes (`"D:\\"`). Test a built command line by splitting it with
    `CommandLineToArgvW` (shell32), the rule programs use - not by reading it.
41. **Herdr's state for Cline is unreliable:** it showed `idle` while Cline (GLM-5.3 Flash) was
    still thinking. For Cline agents read the screen (`read <name> -Visible`) before believing
    `idle` / `done`.
42. **Claude's limits are in its status line input:** `rate_limits.five_hour` and `.seven_day`
    (`used_percentage`, `resets_at` unix seconds) - account-wide, per account; our
    `statusline.js` saves them with `config_dir` ("" = the main sign-in). An account has data
    only after a Claude on it drew its status line; after `resets_at` the old % no longer holds.
    Codex's `rate_limits` in its logs were null on this plan.
43. **A pane gets the Herdr SERVER's environment**, not the caller's, plus `--env`. A profile
    without an account (e.g. alias `2030`) inherits `CLAUDE_CONFIG_DIR` if the server was
    started from an account's shell. Account folders hold COPIES of skills and settings made
    when the account was created: `sync-skills.ps1` updates only `~\.claude\skills`.
44. **`agent prompt --wait` can return `done` within seconds while the agent has just
    started working** (seen with Codex 0.163, 2026-10-23); confirm with `agent get` before trusting it.
45. **The company web gateway (Skyhigh) can reject one long Codex conversation** with `403 Forbidden`
    and its HTML notification page (`url: https://chatgpt.com/backend-api/codex/responses`) while
    chatgpt.com and the API still answer (2026-10-23, conversation at ~200k tokens). Codex retries 5
    times, then stops. A fresh conversation (`/new`) with the task in a spec file worked at once.
    Keep Codex conversations short: one task per conversation, context in a file. Do not try to get
    around the gateway itself.
46. **Every Herdr pane has `HERDR_SOCKET_PATH`** = the default socket
    (`%APPDATA%\herdr\herdr.sock`), while `[[keys.command]]` commands and older tools have none.
    Both mean the same (default) server: compare server identities after mapping the default
    socket path to "default", or the same server looks like two.
47. **Cline runs headless and reports exactly when it is done:** `cline -P cline-pass -m <model>
    --thinking high --auto-approve true -c <folder> --json -t <seconds> "<prompt>"` works the task
    and exits (exit 0); its last JSON line is `{"type":"run_result","finishReason":"completed",
    "usage":{"inputTokens","outputTokens","cacheReadTokens",...}}` (verified 2026-10-23, Cline
    3.0.73, DeepSeek V4.1 Flash, 59 s for a small task). No pane, no idle/done guessing (#41).
    Trap: for cline `-p` is PLAN mode, not "prompt".
48. **Codex creates a new conversation id only with the first message after `/new`:** Herdr's
    `agent_session.value` stayed the same 33 s after `/new` (Codex 0.163). A fresh conversation is
    visible on screen instead: the `>_ OpenAI Codex` banner and NO transcript lines (`• ...`,
    `Worked for ...`); an old one always shows them. Measured timing (live): worktree menu after ~2 s,
    fresh screen ~21 s after Enter, each `agent read` ~3.5 s - wait up to 60 s.
49. **Cline's session `pid` is the shared `cline-hub-daemon`**, not the pane's process: matching a
    pane to its Cline session by pid rarely works; match by folder and newest live session
    (found by DeepSeek, 2026-10-23). Session `status` values seen: idle, completed, failed, exited, done.
50. **`Start-Process -ArgumentList @(...)` does not quote elements**: a path with spaces splits into
    several arguments. Pass one pre-quoted string, or run the script directly as a background job.
51. **Codex has a monthly credit limit on this plan** (`/status`: 3,795 of 5,875 used on
    2026-10-23, resets on the 1st; one round-2 task = 61 credits). Route volume to Cline Pass.
52. **Models that write code well can still review badly:** Gemini 4 Argon High scored 41/41 on the
    trial kit (9.4 min) but, like GLM-5.3 Flash, stated a wrong cause for a review bug without
    running it; DeepSeek V4.1 Flash ran it and was right. Require proof for review claims.
53. **A new Claude account shows startup dialogs that swallow the first prompt:** on account 2040,
    Claude Code 2.1.309 opened with "Yes, use my browser / Enter to confirm - Esc to keep browser
    tools off". A prompt sent then was lost, and its Enter likely CONFIRMED the dialog (owner's
    choice, not ours). Before the first prompt, read the screen and treat any dialog (trust,
    browser, sign-in, model pick) as BLOCKED for the owner (2026-10-23).
54. **`usage` data can be a day old:** it showed account 2040 at 51% weekly (data age 1450 min)
    while Claude itself said 90%. Check the data age column; an old row means "unknown", not "free".
55. **A one-word Cline prompt fails headless:** .NET `ArgumentList` passes a word without spaces
    unquoted, and cline answers `Prompt text must be passed as a single quoted argument` (exit 1).
    Real prompts (with spaces) are fine (found by DeepSeek, 2026-10-23).
56. **Antigravity research drives the owner's Chrome:** `agy -p <research brief> --mode plan` used
    its browser tools in the owner's running Chrome - stopping it dropped Chrome from 31 processes /
    3.5 GB to 23 / 2.0 GB (2026-10-24). `agy --help` has no switch to turn the browser off. The PC
    is short on RAM: do research with `codex --search exec ...` (OpenAI's server-side web search,
    no local browser); use Antigravity only for coding/review, or when the owner accepts the RAM. Its route into Chrome is the
    "Antigravity Browser Extension" in Chrome's Default profile (the owner can switch it off in
    chrome://extensions). It also keeps its own Edge profiles under `~\.gemini\antigravity-cli\brain\<id>\scratch`
    (521 MB from one 2026-10-17 session).
57. **Claude Code reads AGENTS.md only when no CLAUDE.md exists** (working folder or above); with
    both, it loads CLAUDE.md only - so Claude helpers never saw our delegate rules. Fix: import it
    from CLAUDE.md with `@AGENTS.md` (Claude Code docs, "AGENTS.md"; needs v2.1.277+).
58. **Claude Haiku 5.5 at `xhigh` can end a turn with no visible answer** (the whole answer stays in
    its thinking) - Anthropic's Haiku prompting guide. Give Haiku helpers an answer FILE and read it.
    Anthropic's defaults: Haiku and Sonnet start at `medium`, Opus at `medium`; `max` showed
    timeouts / scope violations for Sonnet. Codex SOL 6.1 medium found this headless with
    `codex --search exec` in 19 min, every claim with an official URL (lead spot-checked two).
59. **A screen-text guard must look for the dialog's FOOTER, not its words:** the first dialog guard
    blocked a normal Claude pane because the agent's own answer on screen said "Trust this folder?".
    Real choice dialogs end with a selection footer (Antigravity `↑/↓ Navigate · enter Confirm`,
    Claude `Enter to confirm · Esc to ...`); a reply that only mentions the words has none. Mocked
    tests passed both designs; only the live `ask` on a real pane showed the false positive (third
    time on 2026-10-23/24 that a live check beat green mocks - always do it).
60. **Update CLIs one npm package at a time or in a safe batch, and check C: first:** one package
    that refuses to install (openclaw needs a newer Node) aborts the whole `npm install -g` batch, and
    a full disk (ENOSPC, npm cache 3.2 GB) leaves half-installed packages. Check `Get-PSDrive C`,
    run `npm cache clean --force`, leave out the blocked package. Separate: on 2026-10-24
    `herdr-agents start` got `agent_pane_busy: not an available shell` on fresh panes (status
    `unknown`, revision 1) although pwsh showed its prompt; cause not found. Workaround that worked:
    `herdr pane run <pane> "codex -m ... -c model_reasoning_effort=medium"` - Herdr still detects the agent.
61. **Cloning the toolkit does not install the personal Windows setup.** On 2026-10-24,
    this PC had a newly created, theme-only Herdr config, no installed Herdr skills, no
    App Path registration and no launcher backups. The repository's local reflog records
    a clone that morning; this supports incomplete setup here, not a proven update reset.
    `sync-skills.ps1 -Check` says DIFFERS for missing skills too. The master key commands
    still point to the old, absent D: folder: copying them unchanged can pass check.ps1
    while leaving shortcuts broken. Check command targets as well as matching text.
62. **Removal from a workspace sandbox needs an honest handoff.** On 2026-10-24,
    the removal dry run found Hermes folders and its executable still present, while
    openclaw shims and the named Codex skill were absent. Task/service enumeration,
    H: root and Hermes pending_messages could not all be read; uv tool inventory exited 2.
    User environment and inherited process environment differed. Record each separately;
    neither a clear command lookup nor a bounded folder scan proves complete removal.
63. **A clean setup check does not prove key commands can run.** On 2026-10-24,
    restoring unmodified config blocks, skills and App Path reduced check.ps1 from
    19 FAIL to 0, while all ten key commands still referenced absent D: scripts.
    The check compares master/live command text; it does not test target existence.
    Report path checks separately. When registration was originally absent, record
    that before creating it: use-launcher.ps1 exports the key only after it exists,
    so its -Undo alone cannot restore the original absence.
    The key-binding repair on 2026-10-24 pointed all ten commands at I:\herdr:
    each target exists and parses, and the live key table equals the master.
    Keep conhost.exe --headless and the STA flag for dialogs. Script and launcher
    defaults should use their own toolkit folder so moving the checkout does not
    break them again; TOML key commands need the installed absolute script path.
    Syntax, path and offline checks do not prove live key operation: the lead must
    still press one key. Historical path references were refreshed or described as
    former locations; the task contract itself remains unchanged.
64. **Full disk permission is not Windows administrator permission.** On 2026-10-24,
    Hermes/OpenClaw folders could be deleted after the owner widened disk access, and
    task/service/H: discovery worked. Deleting Clawdbot Gateway, Hermes_Gateway and
    the stopped clawdbot service still returned access denied. Do not retry those
    mutations; give exact elevated commands. Verify environment removal by checking
    HKCU Environment value names: passing PowerShell null to SetEnvironmentVariable
    left named empty values in this run. Existing parent applications can also retain
    old process variables after the persisted user values are removed. Name matches
    inside Windows and other installed packages do not prove a leftover installation.
65. **Check the model list before promising an effort level:** `cursor-agent models` has no Composer
    "high" (only `composer-2.5` and `-fast`), while `agy models` really has `gemini-4-argon-high`.
    Run the CLI's own `models` command, then set the exact id. Also: `grok whoami` is not a command
    and hangs the shell by opening the full-screen app; use `grok login` (device code, no paste) and
    `cursor-agent status` / `claude auth status` to read sign-in state without printing secrets.
    `herdr-agents start` still hits `agent_pane_busy` (see #60); `herdr pane run <pane> "<cmd>"` works.
66. **Setup checks must validate destinations and each account copy.** Matching key text
    does not prove its -File exists; a .lnk can exist with a missing destination. Claude
    accounts keep independent skill copies, including an absent skills folder when the
    profile copy list requires it. Check mode must report the account path without syncing.
67. **Pane IDs need server identity in display readers too.** Offline fixtures with the same
    pane ID on default and lab reproduced incorrect HUD model/account selection. Normalize
    missing and default-socket identities consistently before selecting ownership records.
    Unreadable ownership JSON must abort, never become an empty successful read.
68. **A one-pass command must preserve failure and avoid writes.** HUD -Once used to publish
    metadata and return success after failed discovery. Print-only must skip metadata writes;
    command failures must reach its caller or the background diagnostic log. Collect waits
    on existing tasks, so it must skip the prompt-start grace period needed after a new ask.
    Herdr 0.9.5 read-back is `status server --json` with top-level status=not_running;
    this was verified against an inactive lab session without starting or stopping anything.
69. **Herdr's public API on Windows is newline-delimited JSON over a named pipe.**
    The pipe name is `\\.\pipe\` followed by the socket marker path. Protocol 22 closes
    ordinary RPC connections after one response; a persistent request/reply client fails
    with "The pipe is being closed" on its next request. Deck's lab test passed six direct
    reads through six short pipe connections, without CLI processes. The persistent
    event subscription is a different surface; output-match subscriptions fire on match
    transitions, not every screen update. `pane read` CLI emits terminal text directly,
    while the socket returns `result.read.text`. Preserve this distinction in adapters.
66. **A PowerShell function must never share its name with the program it wraps:** `function Git { & git ... }`
    resolves `git` to the function itself (names ignore case, functions beat programs) and loops forever
    with no error - the first autosync dry run hung for 2 minutes. Name wrappers `Invoke-Git`.

70. **Herdr inside Termux proot-distro: give it a fresh, short socket and `--shared-tmp`.** On the phone `herdr`
    kept failing with "server did not become ready within 15s" until `HERDR_SOCKET_PATH=/tmp/herdr.sock` was set;
    then it opened. The first guess (host socket path over the 107-byte limit) was wrong: radar's probe showed a
    119-byte host path answering under proot 5.1.107, so proot handles long socket paths. The likely cause is a
    stale or wedged server on the default socket, left by radar's own background `herdr server`; not proven.
    Natively, herdr 0.9.5 refuses a socket path over the limit ("local socket name length exceeds capacity of
    sun_path"). Also: `proot-distro login` (5.10) always exits 0, and kills every process at logout unless
    `--no-kill-on-exit` is given.

71. **Inside a Termux proot workbench, keep Termux's bin dir out of PATH for installers.** proot-distro exposes
    `/data/data/com.termux/files/usr`, and a PATH entry for it lets a glibc installer find a bionic binary
    (Termux's `python3.14`). Hermes then detected Android and built an unusable `linux-arm64-bionic` runtime.
    Filter `^/data/data/com.termux` out of PATH before installing or updating anything in the workbench.

72. **Linux port of this setup lives in `linux/`**, not the Windows `install.ps1` path.
    `HERDR_HOME` is the repo directory; state and config use XDG (`~/.config/herdr/`,
    `~/.herdr-accounts/<tool>/<name>`). Bash `local a="$1" b="$2"` evaluates every RHS in
    one `local` line before any name exists - split assignments. Herdr's Gemini CLI integration
    target is `antigravity-cli` (profile `gemini` runs `agy`). Cline has no `herdr integration`
    target yet.

73. **Codex v0.163 `/new` asks "Where should the new conversation run?"** (1 = current worktree,
    2 = new Git worktree), so `ask codex -Fresh` lands its task text in that dialog. On 2026-10-26
    the "1" of the spec heading `## 1. Role` picked option 1 by luck, the rest reached the composer,
    and `ask` printed `DONE` within seconds while Codex had only just started. After `-Fresh` on
    Codex, read the screen before trusting `DONE`; wait with `collect`. The lead's own send-keys to
    answer the dialog was refused by Claude's auto-mode classifier: the owner answers it.

74. **Cline updates itself after a run, and `cline` is missing while it does.** On 2026-10-26 a
    63-second headless run was followed by `npm update -g cline --tag latest` (3.0.73 -> 3.0.74,
    08:27-08:29). In that window npm had renamed the shims to `%APPDATA%\npm\.cline-XXXX` and
    `herdr-headless.ps1` failed with `cline is not installed`. It is not a cleanup or a broken install:
    look for a `node ... npm-cli.js update -g cline` process, wait for it to exit, then run again.
    It needs disk on C: (LESSONS #60). Unknown Cline Pass model ids fail fast with `model not found`;
    a model's own answer about its identity proves nothing (DeepSeek said "claude-sonnet-4-5").

75. **How agent input boxes look in Herdr's ANSI output** (`pane read --format ansi`, 2026-10-26):
    Herdr re-emits colours as truecolor SGR - Claude Code's grey prompt/suggestion text is
    `ESC[38;2;153;153;153m` and its rules `ESC[38;2;136;136;136m`, not SGR 2/90. The Claude prompt
    glyph `❯` (U+276F) is followed by a NO-BREAK SPACE (U+00A0), so `"❯ "` with 0x20 never matches.
    Real capture: `app/tests/fixtures/input/claude-empty.real.ansi`.
    A history prompt (`❯` line with no `─` rule above it) is not the input box. Codex's prompt is
    `›` (U+203A) + space with an SGR 2 placeholder.

76. **Every `git.exe` start costs about 2 s on this PC** (Defender; `git status --short` measured
    2.31 s and 1.96 s, 2026-10-26). Give a handler that runs N git calls a budget of at least 2.5 s x N
    (the Deck's `/api/git` went 5 s -> 15 s). `git status --porcelain=v1 -z` writes a rename as
    `R  <NEW>\0<OLD>\0`, and `rev-parse --show-toplevel` already returns forward slashes.

77. **Go builds on this PC:** Go is portable at `D:\tools\go\go1.26.10\go\bin\go.exe`, not on PATH
    (`app\build.ps1` looks in `$env:DECK_GO`, then known paths, then PATH). Never run two Go builds at
    once: with ~4 GB RAM free the compiler died with `runtime: cannot allocate memory`. PowerShell hides
    `go test`/`go vet` failures behind a `NativeCommandError`; run
    `cmd /c "go ... > out.txt 2>&1"` and read the file.

78. **A Windows job object cannot take a process that is already dead.** `exec.CommandContext` kills
    the child when the deadline passes; if that happens before `AssignProcessToJobObject`, the assign
    fails with `Access is denied` and hides the real cause (the timeout). Check `ctx.Err()` before
    reporting a containment error. A test with a 100 ms deadline shows it only under load (2 of 4
    full runs), never alone.

79. **Edge on this PC loses ~2.5 s to proxy detection before its first request, even to 127.0.0.1**
    (2026-10-28, headless Edge over CDP: `requestStart` 3,157 ms by default vs 653 ms with
    `--no-proxy-server`; first paint 3,384 -> 876 ms; `curl` got the same page in 11 ms). The Deck window
    therefore starts Edge with `--no-proxy-server` (its own profile, page talks only to 127.0.0.1).

80. **PowerShell 7 `Start-Process -ArgumentList` does not quote an argument that contains spaces** (paths
    under `D:\WORK\Software Development\...` split into two arguments, the program refused its flags and a
    wait loop without a timeout hung). Use `ProcessStartInfo.ArgumentList` (`Start-UiProcess` in
    `app/tests/deck-ui.ps1`) and give every wait loop a deadline.

81. **An interrupted `npm update -g @openai/codex` leaves Codex unusable:** on 2026-10-27 08:06 the shims
    were left renamed (`%APPDATA%\npm\.codex-XXXX`, `.codex.cmd-XXXX`), so `codex` was "not recognized";
    `npm install -g @openai/codex@latest` restored the shims but twice dropped the 118 MB optional
    `@openai/codex-win32-x64` package silently ("changed 2 packages", then `Missing optional dependency`)
    on this proxy (`npm pack` of it: `npm error network ... proxy`). The launcher also accepts the binary
    in `node_modules\@openai\codex\vendor\x86_64-pc-windows-msvc\bin\codex.exe`, but `codex.exe` alone is
    not enough: its tools need `codex-code-mode-host.exe` beside it (Codex answered BLOCKED "cannot find
    codex-code-mode-host.exe"). Working repair: copy a COMPLETE, same-version set - e.g.
    `~\.codex\packages\app-server-daemon\releases\<ver>-x86_64-pc-windows-msvc\` (bin\, codex-path\,
    codex-resources\) - to that `vendor\x86_64-pc-windows-msvc\` folder (0.157.1 on 2026-10-28).

82. **`curl -C -` can corrupt a resumed download:** when the registry answered a resume with the full
    file instead of a 206 range, curl appended it (150 MB file for a 118 MB tarball). Download in explicit
    `-r start-end` ranges, append a piece only when `Content-Range` starts at the current size, and check
    the npm `dist.integrity` SHA-512 at the end.

83. **Deck rendering: a forced layout costs what is dirty when it runs.** Setting `scrollTop` (follow
    mode) forces a synchronous layout; with the top bar rewritten earlier in the same frame it cost 2.3 s
    of 10 s in the 8-pane stress mock. Screens first, top bar after, and write DOM text only when it
    changed: 46 -> 60 fps, long tasks 15 -> 0, render p95 17.9 -> 10 ms.

84. **A Claude helper told to "watch every 5 minutes" stops after one round:** its turn ends and nothing
    wakes it (2026-10-28, the Sonnet watcher wrote round 1 and said "checking again at 17:35"). Give it
    `/loop 5m <one round ...; stop the loop when done>`; it schedules the rounds itself (cron job that
    lives while that session is open).

85. **Codex 0.157.1 cannot run GPT-6.1-Sol on this account:** after the #81 repair put 0.157.1 in place,
    `start codex -Model gpt-6.1-sol` started fine, but the first turn failed with HTTP 400
    `"The 'gpt-6.1-sol' model is not supported when using Codex with a ChatGPT account." (model_not_found)`
    (2026-10-28 17:54). `ask` still reported `DONE: ... is idle`, so the error line was easy to miss. Check
    `codex --version` (the roster lists 0.164.0) before delegating to Sol, and read the screen for
    `■ {"type":"error"` after every DONE. Fixed 2026-10-29 by 0.165.0 (#86): Sol medium answers.

86. **Updating Codex through the proxy (2026-10-29, 0.157.1 -> 0.165.0):** the registry's HEAD answer
    for the 174 MB `codex-<ver>-win32-x64.tgz` has no `Content-Length`; read the total from a `-r 0-0`
    request's `Content-Range` instead. 11 ranges of 16 MB then downloaded cleanly and matched
    `dist.integrity`. This time `npm install -g @openai/codex@0.165.0` itself also installed the optional
    package (`node_modules\@openai\codex\node_modules\@openai\codex-win32-x64`, 47 files, hashes equal to
    the verified tarball, 3 min), so the proxy failure of #81 is intermittent: run npm first, then check
    that folder exists and `codex --version` shows the new version; keep the ranged download as the
    fallback. Before the first turn the 0.165.0 footer reads `GPT-6.1-Sol default` although the process
    has `-c model_reasoning_effort=medium`; after the first turn it reads `GPT-6.1-Sol medium`.

87. **`ask -TaskFile` with a long spec can return `DONE: <name> is idle` before the agent starts**
    (2026-10-29, Codex 0.165.0 Astra, 170-line spec): the answer showed only the pasted prompt, no report
    file existed, yet seconds later `herdr agent list` showed the agent `working` on the task. A DONE whose
    "recent output" is just the prompt is not a result: check `agent list` / `read -Visible`, then wait
    with `collect -Agents <name>` instead of resending.

88. **`app/build.ps1` runs `go fmt`, which rewrites every Go file from CRLF to LF:** `git status` then
    lists files nobody edited (content diff empty; git warns "LF will be replaced by CRLF"). Check
    `git diff --quiet -- <file>` per file and restore only those with no content change, or the commit
    carries noise and delegates report "unexpected changes" (2026-10-29, 11 files after one build).

89. **A PowerShell script with a `param()` block is an "advanced script": a forwarded `-o` becomes
    ambiguous with `-OutVariable`/`-OutBuffer`** (2026-10-29, `go-locked.ps1 build -o x` failed). A
    pass-through wrapper takes `$args` and has no `param()` block. Also: inside a function that returns an
    object, `Write-Output` text becomes part of the return value (the lock came back as `[string, FileStream]`);
    use `Write-Host` for messages there.

90. **A child process started with `UseShellExecute=$false` and no redirection inherits the caller's
    output handles:** a Herdr server started that way (verify-lab's restart step leaves it running) kept
    the tool's output pipe open, and the calling command hung past 10 minutes after the script had ended.
    Stop every server you start (`herdr --session lab server stop`) before the command returns, or start it
    with redirected/closed handles.

91. **Herdr keeps an agent record with status `done` after the agent process exited** when the agent was
    reported by hand (`pane report-agent`) and HUD metadata with a TTL is still alive: `agent list` still
    showed `codex` / `done` while the pane's title was `pwsh.exe` (2026-10-29, lab). Tests of "agent gone"
    must wait until `agent list` no longer has the pane. In the owner's real case (2026-10-28) the exited
    agent showed as `unknown` with no agent, which the Deck now refuses with `agent_exited`.

92. **Codex `-s workspace-write` cannot write inside `.git`:** `git fetch` fails with
    `error: cannot open '.git/FETCH_HEAD': Permission denied` (2026-10-29, Sol 6.1 push task), so a
    sandboxed Codex cannot commit or push. Owner's choice for pushes: the delegate runs the read-only part
    (checks, stage list, secret scan, commit body) and the lead runs the git writes. Its `node --test` also
    fails in that sandbox with `spawn EPERM`; `node --test --test-isolation=none <file>` works there.

93. **`git commit -F - @'...'@` in PowerShell does not pipe the message:** the here-string becomes an extra
    argument (a pathspec), git exits 1 with "did not match any file(s)", and a chained script without
    exit-code checks went on to push the OLD commit (2026-10-29: harmless, `windows-toolkit` fast-forwarded
    to `main`). Write the message to a file and use `git commit -F <file>`, and check `$LASTEXITCODE`
    after every git write - `$ErrorActionPreference = 'Stop'` does not stop native commands.

94. **A CSS grid row template on `<body>` breaks when a listed child is `hidden`** (2026-10-29, Deck round
    5): `display:none` children are not grid items, so auto-placement shifts every later child up one
    row - Deck's `.body` landed in an `auto` row and the `1fr` row became dead space under the status bar.
    Use a flex column (`flex: 1 1 0; min-height: 0` on the area that must stretch) or explicit
    `grid-row`/`grid-template-areas` for page shells whose bars can hide.

95. **Deck cannot set a Herdr pane's terminal size** (2026-10-29, checked in `herdr-app/src/client`):
    a tab's size comes only from the attached Herdr client terminal (`ClientMessage::Resize` on the
    client protocol); the public `herdr pane resize --direction --amount` (API `pane.resize`) only moves a
    split line inside that size. So a Deck card can never make the CLI wrap to the card's width; Deck
    wraps the text itself (`screen.js` reflow). A real fix needs Herdr itself (upstream) or Deck acting
    as a Herdr client, which would change the size of the owner's own Herdr window too.

96. **`node --test app/tests/web/` runs only `contracts.test.mjs`** (2026-10-29): the folder has a
    `package.json` whose `main` is that file, so node runs the folder as one module. Name every file:
    `node --test app/tests/web/contracts.test.mjs app/tests/web/split.test.mjs`. Headless Edge
    `--screenshot` with `--virtual-time-budget` can also fire before async page state exists; make test
    hooks wait in the page for the element (Deck `mock.js &drag=` waits for `#grid .card`).

97. **The web test DOM (`app/tests/web/dom.mjs`) is minimal** (2026-10-29): events do not bubble (no
    parent chain), there is no global `removeEventListener`, `cancelAnimationFrame` or
    `document.getElementById`, and selectors are only `.class`, `#id`, `[attr]` or `[attr="v"]` and a tag
    name (no `:scope`, `:not()`, compound `.a.b`). Front-end code that the tests import uses
    `window.addEventListener`, `document.querySelector('#id')` and simple class selectors; anything that
    needs real bubbling is checked in the headless browser harness (`app/tests/deck-ui.ps1`).
98. **Codex "Selected model is at capacity" ends the turn, and `ask` still exits 0** (2026-10-29,
    GPT-6.1-Sol high, review of another repo): the run stopped mid-task with no report file, but the
    answer looked like a normal DONE. Always check the promised output file exists. Resuming in the
    same conversation (a short `ask` without `-Fresh`: "continue from where you stopped") worked and kept
    everything it had already read. Also: a Codex started without `-Cwd` runs in `I:\herdr`, so its
    `workspace-write` sandbox lets it READ other folders but write only under `I:\herdr`. Put report and
    scratch paths there, and create scratch folders before you send the task. Codex's scratch copies can
    end up with ACLs that block a later `Remove-Item` (Access denied).
99. **A rustup install can list components that have no files** (2026-10-29, `I:\dev\rustup`, GNU host,
    after an earlier interrupted install): `rustup component list --installed` showed cargo, rust-std and
    rust-mingw, but `cargo.exe` was missing ("not applicable to the toolchain") and builds failed with
    E0463 (no std). Check `lib\rustlib\manifest-<component>-*` files. Fix that worked:
    `rustup component remove <c>` then `rustup component add <c>`. Also: the auto-mode classifier refuses
    to let the lead download and run installers or set permanent user env vars; give the owner a script
    (`! pwsh -File I:/path/script.ps1` - forward slashes, the `!` shell eats backslashes) and keep env
    vars per shell (an `env.ps1` to dot-source). `herdr-agents team/start` still fails with
    `agent_pane_busy` (#60); `herdr pane run <pane> "<cli> ... \"<first prompt>\""` started Codex and a
    second-account Claude fine (the pane already had CLAUDE_CONFIG_DIR set). Codex model ids live in
    `~/.codex/models_cache.json` (2026-10-29: gpt-6.1-sol, gpt-6.1-terra, gpt-6-astra, gpt-6-luna).
100. **Cline Pass has no `minimax-m3.1`; it is `cline-pass/minimax-m3.1-flash`** (2026-10-29): the TUI
    starts with the wrong id and shows the model in its footer, and the error ("model not found") appears
    only on the first prompt. Prove a model id first with a 5-second headless run
    (`herdr-headless.ps1 -Profile cline -Model <id> -Task "Reply with exactly: OK"`). `cline-pass/minimax-m3`
    also works. Cline's TUI can open a "Get Cline Desktop - press Enter to open" pop-up over the first
    prompt: send `esc`, never `enter` (that opens a web page).
    **Cline TUI turns end often, and a message sent while it works is only QUEUED** (2026-10-29,
    MiniMax M3.1 Flash and DeepSeek V4.1 Flash on a big Rust task): Herdr then shows the agent `idle` with
    a "Queued messages ... Enter with empty input to steer first" box, and nothing happens until an empty
    `enter` is sent. Check the screen for "Queued messages" before judging an idle Cline as stalled. Both
    models also end turns after long plans; DeepSeek still made real commits when nudged step by step.

101. **Headless Edge `--screenshot` resizes the page before it captures** (2026-10-29, Deck round 6):
    anything that closes on `resize` (Deck menus) is gone in the shot even when the page opened it.
    Shoot open menus over DevTools instead: `Emulation.setDeviceMetricsOverride` once, navigate, run
    the click with `Runtime.evaluate`, then `Page.captureScreenshot`.
