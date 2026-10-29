# Deck round 5 - navigation audit (2026-10-29)

Branch `deck-r5-resize` (worktree `I:\herdr-r5`). Measured with the headless harness
(`app/tests/deck-ui.ps1`, mock page) by a probe script at 1900x1030, 1280x720 and 1100x720, on the
Phase 1 build ("before") and the Phase 2 build ("after"). Shots: `tasks/deck-r5-shots/nav-before-*`,
`nav-after-*`.

## 1. Every navigation element (before)

| Where | Element | What it did | Note |
|---|---|---|---|
| Top | Sidebar icon | hide/show rail (Ctrl+B b) | no pressed state |
| Top | Herdr Deck + session chip | label | |
| Top, Create | + Agent / + Shell / Folder | start agent / shell / folder picker | hidden at <= 1100 px |
| Top, Create | columns icon | **Layout builder** | same glyph as "3 columns"; sat in Create |
| Top, Create | people icon | Team presets | hidden at <= 1100 px, no label |
| Top, View | 1 / 2 / 3 / Splits | grid columns / Herdr splits | ok |
| Top | blocked / working / idle / done chips | nothing (spans styled like buttons) | |
| Top, Panels | search icon | search drawer | |
| Top, Panels | lines icon | Agent board | |
| Top, Panels | **shield** icon | Agents hub (herdr-agents) | shield reads as "safe mode" |
| Top, Panels | wrench icon | Tools drawer | |
| Top, Panels | Usage, Inbox | drawers | no pressed state |
| Top, Panels | Commands Ctrl K | palette | label hidden <= 1500 px, left only "Ctrl K" |
| Top, Panels | ? icon | help drawer "Shortcuts and guide" | |
| Top, Panels | Safe checkbox | read-only mode | |
| Rail | All panes / spaces / tabs / agent rows | filter; agent row = filter + zoom | the only place for spaces/tabs (good) |
| Rail | Health | RAM / disks / close finished | |
| Drawer | one right panel, close button, Esc | | |
| Keys | Ctrl+B chords (57), Deck keys (1/2/3, L, J/K, Z, /, X, I, ?, Alt+arrows, Ctrl+K, Ctrl+Shift+F/V/C) | | |
| Palette | every command | | |

## 2. Bugs reproduced (before -> after)

| # | Bug | Evidence before | After |
|---|---|---|---|
| N1 | At 1280x720 the top bar is 98 px too wide: **Commands, Help and Safe are pushed off screen** | probe `clipped: [btn-palette, btn-help, Safe]`, `topScroll: 98`; shot `before-4cards-small-1280x720.png` | `clipped: []`, `topScroll: 0` at 1900, 1280, 1100 |
| N2 | At <= 1100 px **+ Shell, Folder and Team disappear** (CSS `display:none`), no mouse path left | probe `hiddenActions: [btn-shell, btn-folder, btn-team]` | `hiddenActions: []`; labels fold into icons instead |
| N3 | **Status chips look like buttons but do nothing** (spans, not focusable) | `chips: [{tag: SPAN, focusable: false} x3]` | real buttons: blocked -> Inbox, others -> Agent board |
| N4 | **No active state for open panels**: Usage/Inbox/Search look the same open or closed | `usagePressed: null` | `aria-pressed="true"` while its panel is open (same style as 1/2/3) |
| N5 | **Focus lost after closing a panel with Esc** (keyboard users land on `<body>`) | `focusAfterClose: "BODY (lost)"` | back on the opener (`btn-usage`); from a More item -> the More button |
| N6 | **Two different help panels**: the ? button opened "Shortcuts and guide", the ? key "Keys" | `helpTitle` != `questionTitle` | one panel (keys + guide) for button, ?, Ctrl+B ?, Ctrl+B i |
| N7 | **Wrong active row in the rail**: clicking an agent row zooms it, but the tab row stays highlighted and the agent row does not | `railAgentActive: false`, active = `main` | the zoomed agent row is the one active row (`aria-current`) |
| N8 | Agents hub used a shield icon (reads as Safe mode); Layout builder used the "3 columns" glyph and sat under Create | code read (index.html) | Agents hub is a labelled item in More; Layout builder has a grid-plus icon in View |
| N9 | The rail toggle has no pressed state | code read | `aria-pressed` follows the rail |
| N10 | Help panel lists no way to find the new dividers / chips / More menu | code read | DECK_KEYS rows added |

Not bugs (checked): every icon button has an accessible name and a tooltip (`unnamed: []`,
`noTip: []` before and after); spaces/tabs live only in the rail (`topTabs: 0`); the Splits divider
drag still calls `api.resize` (`["lab|w1:p1","right",9]` before and after).

## 3. New layout (after)

`[Sidebar] Herdr Deck (session) | Create: +Agent, Shell, Folder, Team | View: 1 2 3 Splits, Layout builder |
Status: state chips, Inbox (n), Usage (%) ......... Tools: Search, Commands Ctrl K, More..., Safe`

- Slim 48 px bar (was 56), every control 32 px high, groups split by one rule line.
- Narrow windows fold labels into icons in steps (1700 / 1560 / 1440 / 1240 / 960 px); nothing is hidden.
- More menu (static, keyboard: arrows, Home/End, Esc returns focus): Agent board, Agents hub,
  Tools, Spaces side by side, Settings (look), Keys and guide, Stop everything.

## 4. Moves (nothing removed)

| Feature | Before | After |
|---|---|---|
| Agent board | top icon | More menu (`#btn-board`), Ctrl+B a, status chips |
| Agents hub | top shield icon | More menu (`#btn-agents`), Ctrl+B m |
| Tools | top wrench icon | More menu (`#btn-tools`), Ctrl+B . |
| Keys and guide | top ? icon | More menu (`#btn-help`), ?, Ctrl+B ?, Ctrl+B i |
| Layout builder | Create group | View group, new icon |
| Spaces side by side, Settings, Stop everything | chord / palette only | also in More menu |
| "Keys" (keys-only) panel | ? key | merged into "Shortcuts and guide" |

## 5. Tests

- `app/tests/web/nav.test.mjs` (3): menu open/arrows/Esc focus/toggle; panel pressed state and focus
  return (incl. menu item -> More button, no focus steal); one help panel. All 3 fail on the Phase 1
  commit, pass now. Web total 38/38.
- `verify-ui.ps1` layout check now measures the rightmost control (`.top .safe-t`) instead of `#btn-help`.
