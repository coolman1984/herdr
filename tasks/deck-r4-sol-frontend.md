# Task: Fix every frontend item of the Deck round-4 report

## 1. Role and goal
You are a senior frontend performance engineer (vanilla ES modules, no frameworks) working for the
lead (Claude). Implement the web-side fixes of `tasks/deck-r4-astra-report.md` in `app/web/`, following
`tasks/deck-r4-contracts.md` exactly, and make `mock.js` (`?mock=1`) speak the same contracts.
Why: the owner finds the Deck rough and slow. Astra (another agent) is implementing the Go backend side
of the same contracts at the same time; your code must work against that backend without changes.

## 2. Context
- Folder `D:\WORK\Software Development\GitHub\HERDR`. Read `CLAUDE.md`, `AGENTS.md`, the report, the
  contracts, `app/API.md` (current routes), HISTORY Phase 50.
- Code: `app/web/app.js` (state, SSE, cards, render loop), `screen.js` (virtual screen with wrapped
  display rows - the lead rewrote it in Phase 50; keep its design), `ansi.js`, `panels.js` (drawers, raw
  input), `guard.js` (queue, reservations, launches, uncertain delivery), `parity.js` (hub, jobs),
  `api.js` (the only file that knows routes), `ui.js`, `style.css`, `mock.js`.
- Already fixed by the lead (do not redo): D1 (prompt `expect`), D7, D9, D11, D14, D21, D22, D23, D25.

## 3. Exact scope
- Change ONLY `app/web/*`. New JS tests go in a new folder `app/tests/web/` (Node, `*.test.mjs`, run with
  `node --test app/tests/web/`; stub only the DOM/browser pieces you need, import the real modules).
- Do NOT touch `app/internal/*` (Astra owns it), `app/API.md`, `app/build.ps1`, other `app/tests/*`,
  docs, `app/HerdrDeck.exe`. No git commands that change anything. Do not start agents. No browser
  automation (your sandbox cannot open Edge or the network anyway; the lead runs the browser checks).
- If you build, ONLY: `pwsh -NoProfile -File app\build.ps1 -Output "$env:TEMP\sol-r4\HerdrDeck.exe"` (it
  waits on a shared lock; never run go.exe directly). The Go tests may fail while Astra is mid-change:
  that is not your problem; report it and continue.

## 4. Work list (all of it)
Contracts, page side: C1 (store `client_id` from `hello`, send it with every `/api/visible`, hidden
page sends `keys: []`, stale panes dimmed + writes disabled + banner), C2 (apply `settings`), C3 (per-pane
ordered raw queue on `/api/pane/raw`; flush and await before switching panes; IME `isComposing` safe),
C4 (reserve before dispatch, overlap option, release on failure, "reservations unavailable"), C5 (start
queue drained one at a time; attach tasks by returned `pane_id`), C6 (uncertain only per contract), C7
(`input_unknown` like `input_not_empty`), C8 (prefs load/write-through; restart banner), C9 (`null` =
"unknown" in health/disk views).
Report items: D12 (launch records by pane_id, both new panes and shell->agent transitions), D13 (use
`/api/scope/baseline` BEFORE dispatch and `/api/scope/check` after; unknown when baseline failed), D15
(card menus in a body-level popover positioned from the trigger, Esc/outside click, focus return, viewport
collision; every key hidden at narrow widths is in that menu), D16 (drawer generation + AbortController;
live refresh patches fields in place and never erases unsaved input), D17 (upsert job results by id;
fetch `/api/jobs` on start, reconnect and hub open; never downgrade a finished job), P1 (topology dirty
only when workspace/tab/layout VALUES change; rail and dividers untouched on ordinary screen updates),
P4 (cap queued patches per pane, e.g. 64 or 256 KB -> drop and resync once; create card views lazily;
delete every per-pane map/record when a pane disappears; cap local jobs/history), P5 (no layout reads
inside ResizeObserver callbacks except the entry sizes; batch reads before writes in the rAF passes).
Do not attempt I2/I4.

## 5. Steps
1. For each item write a Node test that fails on the current code where logic allows it (DOM-heavy items
   like D15: test the positioning/close logic as functions); implement; pass.
2. Update `mock.js` so `?mock=1` exercises hello/settings/raw/prefs/start pane_id/stale.
3. `node --check` every changed file; `node --test app/tests/web/` passes.
4. Old-code check: run your tests against `git show HEAD:app/web/<file>` copies in a temp folder (add a
   `package.json` with `{"type":"module"}` there) and report how many fail.
5. Write `tasks/deck-r4-sol-answer.md` and reply.

## 6. Definition of done
- [ ] C1-C9 page side and D12, D13, D15, D16, D17, P1, P4, P5 implemented.
- [ ] `node --test app/tests/web/` passes; every changed file passes `node --check`.
- [ ] New tests fail on the old code (count reported).
- [ ] `git status --short` lists only files in your scope (plus other agents' files - leave them).

## 7. Answer format
`tasks/deck-r4-sol-answer.md`: per item - ID, what changed (file:function), test name, old-code result;
what the lead must check in the browser (exact steps); doubts.
Reply in the pane exactly:
DONE | BLOCKED | PARTIAL
Answer: tasks/deck-r4-sol-answer.md
Checks: node-test=<pass/fail counts> new-tests-failing-on-old=<n>/<m>
Not done / doubts: <list or "none">
