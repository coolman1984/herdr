# Task: build the FRONTEND and feature design of "Herdr Deck" (you are the second agent)

## 1. Role and goal
You are a senior frontend/UX engineer (vanilla JS/CSS, Windows desktop-app feel) working for the lead
(Claude). Codex GPT-6.1-Sol is building the Go backend in parallel. You build everything the owner
sees: the web app inside `app/web/`, and you decide how the features look and work.
Goal: a nice, fast, light, keyboard-friendly dark UI that shows and controls the owner's live Herdr
session (agent panes side by side with live screens, status, prompts) - see `tasks/herdr-deck-app.md`
(main spec, section 4 UI) and `tasks/herdr-deck-split.md` (who owns which file, the full feature list
P0-P2). The owner's screenshot reference: Claude pane left, Codex pane right, each with title
(agent, folder, model, ctx %), live screen text, status and a prompt line.

## 2. Context
- Folder `I:\herdr`; read `CLAUDE.md`, `AGENTS.md`, then `tasks/herdr-deck-app.md` and
  `tasks/herdr-deck-split.md` completely before anything else.
- Backend contract: `app/API.md` (written by Codex within its first minutes; re-read it before every
  step). Until it exists, build against your own mock (`app/web/mock.js`, `/?mock=1`).
- Use the engineering skills in `~/.claude/skills/` (frontend-ui-engineering, performance-optimization,
  test-driven-development) where they fit; copy no framework.

## 3. Scope
- Write ONLY: `app/web/**`, `app/FEATURES.md`, `app/API-REQUESTS.md`. Never edit Go code,
  `app/API.md`, build files, other docs. No git. No secrets. No live-session writes: you may only read
  the live session; test writes in the isolated lab session (`herdr --session lab server`,
  `$env:HERDR_SOCKET_PATH = "$env:APPDATA\herdr\sessions\lab\herdr.sock"`, stop it when done).
- Do not open panes yourself, do not start other agents, do not close panes you did not open.

## 4. Design rules
- Dark calm palette with CSS variables; light and high-contrast themes; 16px gutters; works from
  900 px to 4K; one accent colour; status colours: working = blue, idle = grey, blocked = amber (loud),
  done = green, error = red. System font for UI, monospace for screens.
- Speed: ES modules, no build step, render only changed cards, `requestAnimationFrame` batching, pause
  everything when `document.hidden`, virtualise long screens (cap 2000 lines), < 120 KB total.
- Accessibility: real buttons/labels, focus rings, `aria-live` for toasts, full keyboard use, `?` overlay.
- Never answer a blocked agent automatically; destructive actions (close pane, close finished panes,
  server stop) need an explicit confirm; "server stop" needs the typed word.
- Copy/paste is a first-class feature (the owner could not paste a sign-in code in the terminal):
  prompt box uses normal browser paste; add "Paste to pane" (send text exactly as pasted, no execute
  unless the owner presses the Send button) and per-card "Copy screen".

## 5. Steps
1. Read the specs; write `app/FEATURES.md` first: the final feature list with a one-line behaviour
   and a UI sketch (ASCII) per feature, in priority order P0, P1, P2.
2. Build the shell + mock + P0 against the mock; wait for `app/API.md`; wire to the real API; ask
   missing endpoints in `app/API-REQUESTS.md` (the backend answers in `app/API.md`).
3. P1 in the listed order, then P2. After each feature: test it (mock, then lab session with the
   real backend once `app/HerdrDeck.exe` or `go run` exists) and write the evidence in FEATURES.md.
4. Measure: total size of `app/web`, first paint time, CPU while idle (4 panes), memory; write numbers.
5. Self-review once with a fresh eye (keyboard-only walk-through, 900 px width, hidden window).

## 6. Definition of done
- [ ] P0 all working against the real backend in the lab session; P1 each working or listed as blocked
      with the missing endpoint; P2 working or "not done".
- [ ] `app/FEATURES.md` has per-feature evidence and the measured numbers.
- [ ] `app/web` < 120 KB, no external requests, no console errors in a headless load of the page
      (`msedge --headless --dump-dom` or screenshot is allowed only against the lab session).
- [ ] "Not verified" is written wherever you could not run something.

## 7. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Checks: <each check and result, with numbers>
Not done / doubts: <list or none>
