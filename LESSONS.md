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
