# Deck round 6: simple, calm restyle with light + dark theme (Opus 5.5 medium)

Read I:\herdr\CLAUDE.md and I:\herdr\AGENTS.md (delegate rules) first.

## Owner's words
"now the user interface is very pretty bad and not logic and not professional and ugly; make it
simple. I want this design and style and font, light and dark theme"
Reference: C:\Users\mflma\Pictures\Screenshots\2026-10-29 17 26 39.png (open it with Read).
Current look (after round 5): I:\herdr\tasks\deck-r5-shots\nav-after-1900x1030.png and
after-4cards-dragged-1900x1030.png.

## The reference style, in words (match this)
- Calm neutral surfaces: page near-black charcoal (#141416-ish), cards one step lighter
  (#1f1f22-ish) with a hairline border (#2c2c30-ish) and soft rounded corners (~10-12 px).
  No neon. The current lime-green (#c6f432-like) accent buttons and loud chips must go.
- Typography: one clean sans (Inter, or the system UI font: "Segoe UI Variable", system-ui);
  title text ~14 px medium weight, secondary text ~12 px muted grey; numbers tabular. Generous
  line height and padding. Monospace only inside the terminal screen.
- Colour only in small round status/agent icons (filled circles with a white glyph, like the
  reference's blue moon / green check / red sun) - use them for agent kind and status. Everything
  else greyscale.
- Quiet secondary actions: a simple "..." (kebab) menu per card instead of rows of buttons. The
  per-card key row (Esc, Ctrl+C, Enter, arrows, Tab, y, n) and icon row move into that menu or
  appear only on hover/focus; the message box stays.
- Left rail like the reference timeline: a vertical list with round coloured status dots joined
  by a thin line, small muted meta text.
- Simple and logical: fewer visible controls, consistent spacing (8 px grid), one primary action
  colour (a calm blue, e.g. #3b82f6) used sparingly.

## Themes
- Light and dark, both designed (light: white/very light grey cards on #f5f5f7, same layout).
- Default = follow Windows (`prefers-color-scheme`); a toggle (System / Light / Dark) in the
  top bar or More menu, remembered with the existing `LS` helper. All colours as CSS variables
  on :root; dark/light override the variables only. The terminal screen colours follow the theme
  too (ANSI palette readable on both).

## Keep
Every feature and keyboard shortcut from round 5 (resizing, dividers, More menu, nav fixes,
reflow). Change the look, not the behaviour. Don't break web tests or Go tests.

## Scope and git
Only `I:\herdr\app\web\*`, app\tests\web, docs entries. Work in a git worktree on a new branch
`deck-r6-restyle` (e.g. `git worktree add I:\herdr\.wt\deck-r6 -b deck-r6-restyle` - inside
I:\herdr\.wt, NOT at the drive root; add `.wt/` to .git/info/exclude if needed). Commit there;
never push, never merge. Do not stop or replace the owner's running HerdrDeck.exe; build to your
own path (`build.ps1 -Output <worktree>\app\HerdrDeck.r6.exe`).

## Check it
- go test ./... and the web tests pass.
- Screenshots (headless Edge/Chrome against mock mode, like round 5; no browser automation
  tools) in dark AND light at 1900x1030 and 1280x720, with 2 and 4 cards, plus the More menu, into
  I:\herdr\tasks\deck-r6-shots\. Compare side by side with the reference and say honestly what
  still differs.
- HISTORY.md entry + LESSONS if anything durable; `pwsh -File check.ps1` (2 known old failures:
  lesson 66, Phase 43).

## Answer format
Short: what changed (files), screenshot paths, test lines, what still differs from the reference,
doubts.

## Phase 2 (owner, 2026-10-29 17:29): the terminal must use the whole card width
Screenshot: C:\Users\mflma\Pictures\Screenshots\2026-10-29 17 29 44.png - the right card shows the
CLI text in only its left third; the rest is empty.
Proven cause (lead): `herdr pane layout` shows the tab area is 120x40 (the default, because no Herdr
terminal client is attached - the owner uses only Deck), so each of 2 panes is 60 columns. Lesson 95:
the size comes only from an attached client's `ClientMessage::Resize` (herdr-app/src/client);
`pane.resize` only moves the split.
Fix: make Deck act as a Herdr client for sizing only. Study the client protocol in
I:\herdr\herdr-app\src\client (read only; never edit herdr-app). From the browser, report the main
area's size in terminal cells (from the card grid / Herdr layout view and the font metrics) to the Go
backend, throttled; the backend attaches as a client (or reuses one connection) and sends Resize so the
tab area = the Deck area, then the split ratios give each pane its card width. If the owner also opens
a real Herdr terminal window, say what happens (whose size wins) and make it safe (no fight/loop).
Grid view (cards not matching Herdr splits): pick the most useful behaviour, explain it, e.g. switch
the active tab's layout to equal splits matching the card columns via existing split/resize APIs.
Test in the isolated lab session only (CLAUDE.md section 3), never the owner's live server.
Prove with a screenshot: a Claude pane's text reaching the card's right edge. Separate commit;
update lesson 95 (it said this could not be done from Deck).
Do Phase 1 (restyle) first unless Phase 2 is quick; both on the same branch.
