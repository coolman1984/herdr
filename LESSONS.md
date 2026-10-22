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
    new id, so find spaces by label, not by a remembered id. A headless server with no
    window attached can fail splits with `ghostty error -1`; attach a window
    (`herdr --session <name>`) before testing layouts.
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
