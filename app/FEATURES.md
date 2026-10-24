# Herdr Deck - features (frontend owner, 2026-10-24)

Written by the FRONTEND agent (Claude Opus 5.5, account 2040). Each feature: one-line behaviour,
a small UI sketch, then **Evidence** (filled after testing). Priority order P0, P1, P2.
All network calls go through one file, `web/api.js`, so the backend contract (`API.md`) is wired in
one place. `/?mock=1` loads `web/mock.js` (fake state + fake event stream) for offline work.

## Parity checklist (tasks/herdr-deck-parity.md section A) - binding, status kept current

Every item: a button (top bar, card or panel) + a Ctrl+K palette entry + the same Ctrl+B chord as
the terminal (press Ctrl+B, release, then the key; a chord hint shows while waiting; `Ctrl+B ?`
lists all). Status: WORKS (verified, see Test log) / UI READY (built, waits for the endpoint named) /
NOT DONE. Endpoint names refer to `API-REQUESTS.md`.

| # | Today's feature | Deck version | Chord | Status |
|---|---|---|---|---|
| A1 | Workspaces/tabs rail, pane tree, split right/down, focus, zoom, resize, swap/move, rename, close | rail; "Herdr layout" view draws the real split tree from `layouts` rects; drag a divider = resize, drag a card onto another = swap; card buttons for split/zoom/rename/close | `Ctrl+B "` / `%` split, `z` zoom, `x` close, `,` rename, arrows focus | see Test log |
| A2 | HUD info on pane borders | card header: name, path, model chip, ctx ring (SVG) + `ctx N% (used/window)`, out, in, cost, 5h limit, account - the HUD tokens as written | - | see Test log |
| A3 | Sidebar rows (state icon, name, workspace, model, ctx; branch + git status for spaces) | rail rows show the same tokens | - | see Test log |
| A4 | Herdr toasts / system notifications | toasts + Windows notifications (Notification API) for blocked / finished | - | see Test log |
| A5 | Ctrl+B F folder picker -> new workspace | folder dialog: recent folders, browse, type a path | `Ctrl+B f` | R10 |
| A6 | Ctrl+B Shift+F folder + Claude | same dialog, "Open + Claude" | `Ctrl+B F` | R10 |
| A7 | Ctrl+B T team (Claude left + Codex right) | same dialog, "Open + team" | `Ctrl+B t` | R10 |
| A8 | Ctrl+B Shift+V N equal columns | dialog: count + direction | `Ctrl+B V` | R11 |
| A9 | Ctrl+B Shift+B layout builder | visual builder: N columns, each folder + shell/agent/move a running agent; presets save/load | `Ctrl+B B` | R11 |
| A10 | Ctrl+B A live agent board | Board panel from live state, blocked first (no extra polling) | `Ctrl+B a` | see Test log |
| A11 | Ctrl+B Shift+S spaces side by side / separate | Spaces dialog: tick spaces, Join / Separate again | `Ctrl+B S` | R12 |
| A12 | Ctrl+B I shortcuts + guide | Help panel: every Deck chord + Herdr keys + GUIDE.md, one search box | `Ctrl+B i` | see Test log |
| A13 | Ctrl+B Shift+Q stop everything | typed-word confirm ("stop") | `Ctrl+B Q` | R13 |
| A14 | Ctrl+B Shift+C Claude on least-used account | button + chord | `Ctrl+B C` | R13 |
| A15 | Herdr keys: split, zoom, detach, reload config, `?` | split/zoom per card, reload config in palette; detach = "the Deck window is the client; close it, Herdr keeps running" | `Ctrl+B r` reload, `Ctrl+B ?` | see Test log |
| A16 | herdr-agents: tools, start, ask, read, status, stop, diff, fanout, history, watch, team, collect, login, accounts, usage, handoff | Agents hub panel with one tab per command | `Ctrl+B g` | R14 |
| A17 | headless, trial, check.ps1, sync-skills, tasks/, skills, docs hub | Tools panel | `Ctrl+B o` | R15 / R7 |

## Layout (whole window)

```
+--------------------------------------------------------------------------------+
| Herdr Deck  [lab] | + Agent  + Shell  Team v | Usage 41% | Inbox 2 | Ctrl+K  ?  |  top bar
+-----------+--------------------------------------------------------------------+
| WORKSPACES| +-------------------------------+ +-------------------------------+ |
| * herdr   | | claude-2040  I:\herdr  opus   | | codex-1  I:\herdr  gpt-6.1    | |
|   1 main  | | ctx 23% 230k  [working]       | | ctx 8%   [idle]               | |
|   2 tests | |-------------------------------| |-------------------------------| |
| o lab     | | live screen (monospace, ANSI) | | live screen                   | |
|           | |                               | |                               | |
| HEALTH    | |-------------------------------| |-------------------------------| |
| RAM 61%   | | [prompt box.............] Send| | [prompt box.............] Send| |
| I: 120 GB | | Esc ^C Enter Up Dn Tab y n .. | | Esc ^C Enter Up Dn Tab y n .. | |
+-----------+-------------------------------------------------------------------+ |
| status line: connected - lab - 4 panes - last update 0.4 s          [toasts] ->  |
+--------------------------------------------------------------------------------+
```

## P0 (must)

### P0.1 Live pane grid + focus/zoom
Cards for every pane of the selected tab (or all tabs), 1/2/3 columns (keys `1` `2` `3`); click a
card title or press Enter on it to zoom it to full width, Esc returns. Only changed cards re-render
(per-card hash), batched in `requestAnimationFrame`; screen keeps the scroll position unless the user
is at the bottom (then it follows). Long screens are capped at 2000 lines and virtualised.
```
[cols: 1 2 3]   card title click -> zoom   +-------------- zoomed card --------------+
```
**Evidence:** see "Test log" below.

### P0.2 Status dots
Dot colour per agent: working = blue, idle = grey, blocked = amber (pulsing), done = green,
error = red, unknown = hollow. Same dot in rail, card title and inbox.
```
(o) working   (o) idle   (!) BLOCKED   (v) done   (x) error   ( ) unknown
```

### P0.3 Prompt box per pane
Textarea; Enter sends (agent prompt for agents, run for shells), Shift+Enter newline, normal browser
paste. Sent text stays in per-pane history (Up on an empty box recalls). Send button shows "sending..."
then a toast with the backend result or the error.
```
| > Read CLAUDE.md, then...                                 [Send] [Paste to pane] |
```

### P0.4 Key buttons
Esc, Ctrl+C, Enter, Up, Down, Tab, y, n -> `pane send-keys`. Each is a real button. Ctrl+C asks
nothing (it is harmless); keys are never sent automatically.
```
[Esc] [Ctrl+C] [Enter] [Up] [Down] [Tab] [y] [n]   [Copy screen] [Rename] [Close]
```

### P0.5 Close / rename
Rename: inline input in the title (Enter saves, Esc cancels). Close: confirm dialog naming the pane
and agent ("Close pane w1:p2 (codex-1)? This stops the agent."), default button = Cancel.
```
+-- Close pane? ------------------------------+
| w1:p2  codex-1  I:\herdr                    |
| This stops the agent in it.                 |
|                       [Cancel]  [Close pane]|
+---------------------------------------------+
```

### P0.6 Blocked highlight + toast
Blocked card: amber 3 px border, amber title bar, "BLOCKED - needs you" badge, the last screen lines
visible. One toast per new blocked event (`aria-live=assertive`). The Deck never presses a key for it.
```
+== BLOCKED - needs you ======================== codex-1 ==+
| Do you trust this folder?  > Yes  No                     |
```

### P0.7 Workspaces / tabs rail
Left rail lists workspaces, under each its tabs, each with the worst status dot of its panes
(blocked > working > idle). Click = filter the grid; "All" shows every pane. Keyboard: Alt+Up/Down.
```
ALL PANES (4)
v herdr  (o)
    1 main   (!)
    2 tests  (o)
> lab      ( )
```

### P0.8 "+ Agent"
Dialog: profile (from backend profiles = agents.json names), model (default from profile, editable,
with datalist), effort (low/medium/high/xhigh/max), account (for claude/codex: none/2030/2040/1984),
target (new pane right of the focused pane / new tab), folder. Start -> backend `herdr-agents start`;
the result or error is a toast.
```
+-- Start agent ------------------------------+
| Profile [codex v]  Model [gpt-6.1-sol    ]  |
| Effort  [medium v] Account [none v]         |
| Folder  [I:\herdr              ]            |
|                       [Cancel]  [Start]     |
+---------------------------------------------+
```

### P0.9 "+ Shell"
Opens a plain shell pane (split right of the focused pane, or a new workspace when Herdr has none).

### P0.10 Herdr-not-running empty state
When the backend says Herdr is not reachable: a calm centred panel, the error text, "Start Herdr"
button (runs the launcher) and auto-retry; the grid comes back by itself when Herdr starts.
```
            +-----------------------------------------+
            |   Herdr is not running                  |
            |   (error: socket not found ...)         |
            |   [Start Herdr]   retrying every 2 s    |
            +-----------------------------------------+
```

### P0.11 Command palette (Ctrl+K)
Fuzzy list: focus agent, ask agent (opens its prompt box), stop agent (confirm), open docs, reload
Herdr config, toggle theme, columns, open inbox... "Server stop" is only there with a typed "stop".
```
+-- > ask cod ---------------------------------+
| Ask codex-1                          w1:p2   |
| Stop codex-1                        confirm  |
+----------------------------------------------+
```

## P1 (high value, in this order)

### P1.1 Real copy/paste
Prompt box takes native Ctrl+V / right-click paste. "Paste to pane" dialog: a textarea where the owner
pastes (e.g. a sign-in code); "Send text" sends it exactly (`pane send-text`, no Enter); a separate
"Send + Enter" button. Per card "Copy screen" copies the plain text (ANSI removed); screen text is
selectable (select-to-copy works because the screen is real text).

### P1.2 Attention inbox
Top-bar "Inbox N" opens a list of blocked (first) and finished/idle agents with their last line; click
= zoom that card. Browser notification (Notification API, asked once on a click) + optional soft beep
(WebAudio, off by default). Never answers.
```
Inbox (2)
(!) codex-1   Do you trust this folder?      2 min   [Show]
(v) claude-1  Done. Changed: app/web/...     5 min   [Show]
```

### P1.3 Task launcher
Lists `tasks/*.md` (backend), preview pane, "Send to agent [v]" puts
"Read CLAUDE.md and AGENTS.md, then do the task in tasks/<file>" into that agent's prompt box and
sends on the owner's click; or "Start new agent for it".

### P1.4 Broadcast / fan-out
Checkbox per card ("select"); a broadcast bar sends one prompt to all selected; "Compare" view shows
the selected cards side by side (zoomed columns).

### P1.5 Usage and limits
Top-bar usage chip; panel with bars per Claude account (5-hour / weekly), Codex limits, tokens per
pane, from backend `usage` (herdr-agents usage).

### P1.6 Health widget
Rail bottom: RAM %, free disk C:/G:/I:, pane count, idle panes list, "Close finished panes" (confirm
listing every pane that will close).

### P1.7 Team presets
"Team v" menu: Claude + Codex; Claude + Codex + Gemini; Research (headless) -> sequence of + Agent
calls, each shown as its own toast.

### P1.8 Sign-in helper
Panel with one card per tool (claude accounts, codex, cline, grok, agy, cursor): signed-in state from
backend, "Open login pane" button; the owner types/pastes codes in the pane (Paste to pane helps).

## P2 (nice)

9. Changes view (git status / diff --stat per pane folder) - read-only panel.
10. Run history viewer (`%APPDATA%\herdr\agent-runs\*.md`) with search.
11. Docs hub (HISTORY/LESSONS/ROSTER/PLAYBOOK) with search + append-only "add lesson".
12. Check runner (`check.ps1` OK/WARN/FAIL lines).
13. Pane layouts: save/restore columns + pinned/hidden panes (localStorage, frontend only).
14. Transcript export: save screen as Markdown; "copy as prompt context".
15. Prompt library: saved snippets with {vars}, per-agent history, Up recall.
16. Raw terminal mode: typing straight into the pane with a visible "typing into X" bar.
17. Themes (dark/light/high contrast), font size, density; `?` overlay with all shortcuts.
18. Safe mode: read-only switch that disables every write button.
19. Activity timeline: when each agent started/finished/blocked today (frontend-observed).
20. Queued prompts: send when the agent becomes idle (owner queues it explicitly).

## Test log (evidence)

(filled in as features are tested)

## Measurements

(filled in at step 4)

## For HISTORY/LESSONS

(frontend lines for the BACKEND agent to merge)
