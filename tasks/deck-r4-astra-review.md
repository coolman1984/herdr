# Task: Full review of Herdr Deck - every bug, every performance problem, best practices, fix plan

## 1. Role and goal
You are a senior web-performance and terminal-UI engineer (Go + vanilla JS) reviewing for the lead (Claude).
Goal: one report, `tasks/deck-r4-astra-report.md`, that lists ONLY what must be done in Herdr Deck:
each bug and problem with its proven cause and the concrete fix (file, function, approach), plus the
big improvements worth making, backed by best practice from the web and by how other open-source
projects on GitHub solve the same problem.
Why: the owner finds the Deck rough and slow. The lead will implement the fixes from your report, so
every item must be precise enough to implement without guessing, and every claim must be proven.

## 2. Context you need
- Folder: `D:\WORK\Software Development\GitHub\HERDR`. Read first: `CLAUDE.md`, `AGENTS.md`, then
  `app/README.md`, `app/FEATURES.md`, `app/API.md`, `app/PERF.md`, HISTORY.md Phases 44-49 and
  LESSONS.md #73-#86.
- What the Deck is: `app/` builds `HerdrDeck.exe` (Go backend `app/internal/deck/*.go`, embedded web
  UI `app/web/*.js|css|html`). It reads Herdr panes through Herdr's NDJSON socket API, streams screen
  diffs to the page over SSE, and shows every agent pane as a card (screen, state, tokens, message box,
  key buttons). Build: `pwsh -File app/build.ps1`. Unit tests: `go test ./...` in `app/`.
- Earlier rounds: `tasks/deck-r3-codex-bugs.md` (bug list B1-B20 with triage),
  `tasks/deck-r3-codex-terra-answer.md`, `tasks/deck-r3-lead-notes.md`. Re-check those bugs against
  the CURRENT code: say which are fixed, which remain.
- The working tree has uncommitted round-3 changes. Review the working tree as it is now.
- Owner's screenshots (open them with your image viewer tool):
  - `C:\Users\m.labib\Documents\PicPick\2026-10-28 17 31 51.png` - 3 cards. Problems the owner marked:
    (a) the left card's screen scrolls SIDEWAYS (horizontal scrollbar) instead of re-flowing the text to
    the card width - the owner wants text to reflow, never a sideways scroll;
    (b) the middle card's text uses only a narrow column, a large empty area on the right
    (red box) - the screen does not fill the card;
    (c) cramped headers: names cut to `clau...`, `wat...`, `cod...`.
  - `C:\Users\m.labib\Documents\PicPick\2026-10-28 17 40 25.png` - the agent in a card had exited,
    state shows `unknown`, and the owner's message was typed into the bare PowerShell prompt
    (`ParserError: Missing ] ...`). Sending agent text to a shell must not happen silently.
- Key fact for (a)/(b): a terminal screen has a fixed column count (the pane's PTY width in Herdr).
  Find out how the card width relates to the pane's columns today, and what the right fix is
  (e.g. resize the Herdr pane to the card's columns, measure cell width, re-request the screen;
  or another approach used by xterm.js / ttyd / Wetty / VS Code terminal / tmux web clients).
  Check what Herdr's API and CLI offer for pane size (`herdr.exe --help`, `app/internal/deck/herdr.go`).

## 3. Exact scope
- Create ONLY: `tasks/deck-r4-astra-report.md`. Scratch files go in `%TEMP%\astra-r4\` and are
  deleted at the end.
- Do NOT change any product code, test, doc or config file. Do NOT run git commands that change
  anything (status/diff/log are fine). Do not install anything. Do not start other agents.
- Never touch the owner's live Herdr session: no `herdr` command without
  `$env:HERDR_SOCKET_PATH = "$env:APPDATA\herdr\sessions\astra\herdr.sock"` pointing at your own
  isolated session (`herdr.exe --session astra server`, stop it at the end with
  `herdr.exe --session astra server stop`). Absolute path:
  `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe`.
- Browser: the ONLY browser use allowed is the project's own headless harness
  `app/tests/deck-ui.ps1` (one headless Edge over CDP, isolated APPDATA), one run at a time, closed
  when done. No other browser automation, no opening web pages.
- RAM is short (about 4-5 GB free): before `build.ps1` or `go test`, check no other `go build` /
  `build.ps1` process runs; never run two builds at once. Build into `%TEMP%\astra-r4\`, not over
  `app/HerdrDeck.exe`, if build.ps1 allows an output path; otherwise report that you skipped the build.
- Research: use your web search tool for best practices and for GitHub projects (xterm.js, ttyd,
  Wetty, gotty, sshx, VS Code terminal, Warp, Zellij web, tmux web clients, Tabby, Wave Terminal,
  any agent-dashboard projects). Cite the URL for each borrowed idea. If web search is not available
  to you, say so at the top of the report and continue with code review only.

## 4. What to review (all of it)
1. The screenshot problems (a), (b), (c) and the exited-agent message (screenshot 2): root cause in
   code (file:line), fix.
2. Rendering and performance in `app/web/*` (screen.js, app.js, panels.js, ansi.js, style.css):
   layout thrash, forced reflows, DOM size, per-frame work, scroll handling, ANSI parsing cost, CSS
   containment (`contain`, `content-visibility`), virtualised rows, canvas/WebGL rendering vs DOM,
   requestAnimationFrame batching, Web Worker use, memory growth, idle CPU with the page hidden.
3. Backend in `app/internal/deck/*`: polling cadence, diff cost, SSE traffic, goroutine/handle leaks
   (PERF.md says handle growth is undiagnosed), lock contention, error paths that fail silently.
4. Correctness: state detection (working/idle/blocked/unknown), sending input to the wrong target,
   races between card actions, reconnect after the Deck or Herdr restarts, multiple pages open.
5. UX: header crowding, buttons, keyboard use, readability, what a dashboard for agent panes should
   show (compare with the GitHub projects you find).
6. The earlier bug list B1-B20: status of each.

Prove each finding: a command and its output, a test you ran, a measurement, or an exact code
reference with the reasoning. Mark anything you could not run as "not verified".

## 5. Steps
1. Read the context files and the code.
2. Run `go vet ./...` and `go test ./...` in `app/` (after the RAM check) and record the results.
3. If feasible within the limits above, run the Deck against your isolated Herdr session with the
   harness and reproduce (a), (b), (c) and the exited-agent case; capture numbers (fps, long tasks,
   render ms, scrollWidth vs clientWidth, RSS).
4. Research best practices and GitHub solutions; map each to a concrete Deck change.
5. Write the report, clean up `%TEMP%\astra-r4\`, stop your isolated Herdr session.

## 6. Definition of done
- [ ] `tasks/deck-r4-astra-report.md` exists and follows section 7.
- [ ] Every item has: severity, symptom, proven cause (file:line + evidence), fix (files/functions,
      approach), how to verify the fix.
- [ ] Screenshot problems (a), (b), (c) and the exited-agent message are each covered.
- [ ] `git status --short` shows no change except the new report (compare with the start).
- [ ] Isolated Herdr session stopped; no headless Edge or HerdrDeck process of yours left running.

## 7. Answer format
The report file, in this order:
1. `## Summary` - 5-10 lines: the biggest problems and the biggest wins.
2. `## Must fix (bugs)` - numbered D1, D2 ..., ordered by severity (critical/high/medium/low).
   Each: Symptom / Cause (file:line, evidence) / Fix / Verify.
3. `## Performance` - numbered P1 ...; same fields plus the measured or expected gain.
4. `## Improvements` - numbered I1 ...; the change, why, source URL(s) of the practice or repo.
5. `## Earlier bugs B1-B20` - one line each: fixed / still open (-> D#) / not reproducible.
6. `## Order of work` - the fix order you recommend, grouped into batches that touch separate files.
7. `## Doubts` - what you could not verify.
No praise, no history, nothing that is already fine.

Your reply in the pane, exactly:
DONE | BLOCKED | PARTIAL
Report: tasks/deck-r4-astra-report.md
Counts: D=<n> P=<n> I=<n>
Checks: <each item of section 6 and its result>
Not done / doubts: <list, or "none">
