# Deck round 5: fill the window, every pane resizable (Opus 5.5 medium)

Read I:\herdr\CLAUDE.md and I:\herdr\AGENTS.md (delegate rules) first.

## Owner's words
"fix this stupid look and make the app flexible and can be resize all the inside windows"
Screenshot: C:\Users\mflma\Pictures\Screenshots\2026-10-29 12 29 26.png (open it with Read).

## What is wrong in the screenshot
- Window is ~1900x1030. Two agent cards sit side by side but are only ~440 px tall; the lower
  half of the window is empty black space. The terminal text inside each card scrolls in a tiny box
  and also shows a horizontal scrollbar.
- Nothing inside the window can be resized by the user in grid view.

## Goal
1. The main area always fills the window height and width (any window size, any column count,
   rail open or closed, 1..N cards). Cards stretch to fill; no dead space. Rows share the height
   evenly (2 cards + 2 cols = one full-height row; 4 cards + 2 cols = two half-height rows).
2. Every inner area is user-resizable by dragging a divider:
   - between card columns and between card rows in grid view,
   - the left rail width,
   - the right-side drawer/panel (the `auto` third column of `.body`) width,
   - the composer (message box) height inside a card is fine as is (auto-grow); the terminal
     screen inside the card takes all remaining card height.
   Sizes persist per view in localStorage (reuse the existing `LS` helper in web/app.js);
   double-click a divider resets it. Minimum sizes so nothing collapses to zero.
   Keyboard: dividers focusable, arrow keys move them (accessibility).
3. The terminal view refits (columns/rows) after any resize, throttled - reuse the existing
   fit path (`c.view.fit()` / paneCols) and do not break the Herdr layout view (`.grid.herdr-layout`,
   `Q.placeLayout`, which already has `.divider.v/.h`; reuse that divider style/code if it fits).
4. Avoid the horizontal scrollbar in a card when the card is wide enough.

## Scope
Only `I:\herdr\app\web\*` (style.css, app.js, and whatever else in web/ you need) plus
app\tests if there are web tests, plus the docs entries below. No Go backend change unless
truly needed (say why). Work on a git branch `deck-r5-resize` in I:\herdr (create it; commit there;
never push, never merge to main - the lead merges).

## Check it
- Build per I:\herdr\app\build.ps1 / README.md and run the existing tests (`go test ./...` in app,
  plus any web tests). All must pass.
- Look at it for real: the frontend has a mock mode (web/mock.js) - serve and screenshot it at
  1900x1030 and 1280x720, with 2 and 4 cards, rail open and closed, before and after a drag.
  No browser automation tools (AGENTS.md rule 7): use a headless screenshot (e.g. Edge/Chrome
  `--headless --screenshot`) or the built exe; save shots to I:\herdr\tasks\deck-r5-shots\.
- Do NOT stop or restart the owner's running HerdrDeck.exe or Herdr server.

## Docs (CLAUDE.md section 0/1)
HISTORY.md entry (Symptom / Cause / Fix / Lesson), LESSONS.md for any durable fact, then
`pwsh -File check.ps1` (it already failed before for lesson 66 duplicate / phase numbering -
say if that is still the only failure).

## Answer format
Short: what was wrong (real cause, proven), what changed (files), screenshot paths,
test output lines, doubts.

## Phase 2 (owner, 2026-10-29): professional navigation
Owner's words: "make overall improvements as the navigation system is bad and buggy and not
arranged; it needs to be professional and organized."
Do this AFTER the fill/resize/reflow work is committed, on the same branch, separate commits.
1. Audit first: list every navigation element (top bar: + Agent, + Shell, Folder, the icon buttons,
   tab buttons 1/2/3 + Splits, idle/done chips, search, icons, Usage, Inbox, Commands, help, Safe;
   left rail; drawers; keyboard shortcuts) and every bug you can reproduce (wrong active state,
   buttons that do nothing, two controls doing the same thing, unclear icons without labels or
   tooltips, focus/keyboard traps). Write the list to I:\herdr\tasks\deck-r5-nav-audit.md.
2. Redesign into a clear, grouped layout like a professional desktop tool (e.g. VS Code / Windows
   Terminal): left = navigation (spaces/tabs, in the rail, one place only), top = a slim title bar
   with grouped actions (Create | View/layout | Status | Tools), overflow into one menu, every icon
   button has a tooltip and an accessible name, consistent sizes and spacing, one clear active
   state. Keep every existing feature reachable (nothing removed silently; list any move).
3. Fix every bug from the audit, with a test where the code has tests.
4. Before/after screenshots at 1900x1030 and 1280x720 in tasks\deck-r5-shots\.
