# Herdr Deck - ADDENDUM: full parity with today's setup, then better and more beautiful (owner, 2026-10-24)

The owner: "I need exactly the features we have here to be in the Deck also - better, more useful,
more beautiful." This addendum is part of `tasks/herdr-deck-app.md` and `tasks/herdr-deck-split.md`
and is as binding as they are. Ownership is unchanged: BACKEND = Codex (Go, API, scripts, build),
FRONTEND = Opus (app/web/**). Priority: PARITY first (everything below works in the Deck), then the
P1/P2 list, then polish.

## A. Parity checklist - every item must be reachable from the Deck (button AND Ctrl+K palette AND,
where it exists today, the same Ctrl+B-style key chord shown in a keys overlay)
Herdr terminal (what the owner uses now):
1. Workspaces + tabs rail, pane tree with splits (right/down), focus, zoom, resize, swap/move, rename,
   close; dragging a split divider resizes; dragging a card to another slot swaps panes.
2. Pane borders/sidebar info (our HUD, `herdr-hud.ps1`): name, folder, model, ctx % (used/window),
   tokens out, account - on every card header, exactly the values the HUD writes today.
3. Agent status (working/idle/blocked/unknown) with the sidebar rows from `config/settings.toml`
   (state icon, name, workspace, model, ctx; git branch + git status for spaces).
4. Toasts/notifications like Herdr's (`delivery = "system"` -> Windows notifications).
Our key bindings (`config/keys.toml`, `herdr-help.ps1 -List`) - each becomes a Deck action:
5. Ctrl+B F  folder picker -> new workspace there (`herdr-open-folder.ps1`): Deck version = a
   folder browser with recent folders (`%APPDATA%\herdr\open-folder-last.txt`) + type-a-path.
6. Ctrl+B Shift+F  same, then start Claude in it.
7. Ctrl+B T  pick folder -> Claude left + Codex (Sol 6.1, medium) right (use the agents.json args).
8. Ctrl+B Shift+V  split the focused pane into N equal columns/rows (`herdr-columns.ps1`).
9. Ctrl+B Shift+B  layout builder (`herdr-layout.ps1`): visual builder - N columns, each with its own
   folder and an empty shell / new agent / a running agent moved in; save/load presets
   (`%APPDATA%\herdr\layouts\`, `layout-last.json`); build via `-PlanFile`.
10. Ctrl+B A  live agent board, blocked first (`herdr-agents watch`/`status`).
11. Ctrl+B Shift+S  spaces side by side / "Separate again" (`herdr-spaces.ps1`,
    `side-by-side.json`).
12. Ctrl+B I  shortcuts + guide window (`herdr-help.ps1`, `docs/GUIDE.md`): searchable in the Deck.
13. Ctrl+B Shift+Q  stop everything - ask first, typed confirmation (`herdr-stop.ps1`).
14. Ctrl+B Shift+C  Claude on the account with the least 5-hour use (`herdr-agents usage`).
15. Herdr's own keys: split, zoom, detach, reload config (`herdr server reload-config`), `?` list.
Delegation tool (`herdr-agents.ps1`: tools, start, ask, read, status, stop, diff, fanout, history,
watch, team, collect, login, accounts, usage, handoff) - each command is a Deck screen/action:
16. tools (profiles from agents.json, installed?, model, effort), start (profile + model + effort +
    account + folder + worktree), ask (send task, wait, show answer; `-Fresh`), read, status, stop
    (only panes the tool started), diff (agent's changes), fanout (one task -> many agents), history
    (`agent-runs`), watch, team (presets), collect (gather answers), login/accounts (sign-in helper:
    the OWNER finishes the sign-in), usage (Claude 5h/weekly bars per account), handoff (lead->lead
    handoff file from `tasks/HANDOFF-TEMPLATE.md`).
Other tools: 17. `herdr-headless.ps1` (run a Cline profile headless: DONE/FAILED, answer, tokens),
    `herdr-trial.ps1` (model trial), `check.ps1` (setup check), `sync-skills.ps1`, `tasks/` specs,
    skills `herdr-team` / `herdr-workspace`, docs hub (HISTORY, LESSONS, AGENT-ROSTER, TEAM-PLAYBOOK).
Safety is identical to today: nothing answers a blocked agent; approval/trust stay the owner's;
secrets never shown; start/stop only what the Deck started unless the owner confirms.

## B. "Better" - the Deck must be clearly more useful than the terminal
- See everything at once: many live cards with zoom, a global search over all pane screens, jump to
  the pane that mentioned an error, pin/hide cards, per-card status history sparkline.
- Mouse + keyboard: drag to resize/reorder, double-click to zoom, Ctrl+K for everything, Ctrl+B chords
  kept for muscle memory (same keys as the terminal), shown in a cheat-sheet overlay.
- Smart help: after an agent finishes, one-click "send to reviewer agent", "run check.ps1", "show
  diff"; a task-pipeline view (writer -> reviewer -> final gate) following docs/TEAM-PLAYBOOK.md.
- Resource-aware: RAM/disk widget, "close finished panes", warning when a pane eats memory.
- Real copy/paste everywhere (sign-in codes!), screen export, prompt library, queued prompts.

## C. "More beautiful" - quality bar (the owner will judge the look first)
- A real design system in CSS variables: 4/8 px spacing scale, one accent, soft layered surfaces,
  subtle borders + shadows, 12-14 px radii, consistent inline-SVG icon set (no icon fonts, no CDN),
  Inter-like system font stack for UI, a good monospace (Cascadia Code / Consolas fallback) for screens.
- Motion: 120-200 ms eased transitions, a calm "working" pulse, status chips that morph, toasts that
  slide, skeleton loaders - all disabled under `prefers-reduced-motion`; no layout jank.
- Polished states: designed empty states (Herdr not running, no agents), error banners with a fix
  button, loading skeletons, tooltips with shortcut hints, focus rings, dark (default) + light +
  high-contrast, density toggle, font-size toggle.
- Card anatomy like a premium terminal dashboard: coloured agent glyph per tool (Claude, Codex,
  Gemini, Cline, Grok, Cursor), model + effort chips, ctx ring (SVG) instead of plain "19%", token
  counter, tidy prompt composer with attachments-style chips (task file, folder, snippet).
- Check it like a designer: screenshot the page in the lab session at 1100x700, 1440x900, 2560x1440,
  dark and light, and fix what looks off. Keep the whole front end < 150 KB.
