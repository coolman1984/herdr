# Task: build "Herdr Deck" - a fast, light web-tech control app in ONE double-click .exe

## 1. Role and goal
You are a senior full-stack engineer (Go + vanilla web) working for the lead (Claude).
Goal: the owner double-clicks `I:\herdr\app\HerdrDeck.exe` and gets a nice, fast, light window that
shows and controls exactly what the Herdr terminal shows today - same live session, same agents -
as a web app: the screenshot the owner sent shows two agent panes side by side (Claude on the left,
Codex on the right, status, model, context %, the agent's screen text, a prompt line).
Why: the owner wants the CLI experience in a clean web UI, with one exe, no console, no install.
It talks to Herdr through the SAME connection the CLI uses (the `herdr` command / its socket API), so
the terminal Herdr and the Deck always show the same session.

## 2. Context you need
- Folder `I:\herdr`; read `CLAUDE.md`, `AGENTS.md`, `LESSONS.md` (esp. #1 stable herdr path, #20
  isolated test session, #39 no console flash), `docs/GUIDE.md`, `herdr-skill.md`, `agents.json`.
- Herdr CLI: `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe` (stable path, never bare `herdr`).
  Study `herdr api schema`, `herdr api snapshot`, `herdr pane list/read/send-text/send-keys/split/close/run`,
  `herdr agent list/get/read/prompt/send-keys/start/wait/explain`, `herdr workspace ...`, `herdr tab ...`.
  Always read ids from Herdr's JSON, never guess. `herdr-agents.ps1` (+ `agents.json`) is the delegation
  tool: call it for start/stop/ask/usage/accounts rather than re-implementing it.
- Toolchain on this PC: Go 1.26 (`C:\Users\mflma\tools\go\go1.26.1\go\bin\go.exe`), Node 24, Edge +
  WebView2 runtime installed, no Rust/Zig, no .NET SDK. Internet for Go modules may not work: use ONLY
  the Go standard library (no third-party modules), and vanilla JS/CSS (no npm, no CDN, no framework).

## 3. Exact scope
- Create: everything under `I:\herdr\app\` (`main.go`, `internal/...`, `web/` (index.html, app.js,
  style.css, embedded with `go:embed`), `build.ps1`, `README.md`, `tests/`).
- Change: `docs/FILES.md` (new rows), `HISTORY.md` (new phase), `LESSONS.md` (durable facts only).
- Do NOT touch: `herdr-app/`, `~/.claude/settings.json`, approval/trust settings, other agents'
  panes, the owner's live session except READ-ONLY calls (list/read/snapshot). No git. Never read or
  print secrets (`.credentials.json`, `auth.json`). Test writes (send-text, prompt, split, close)
  ONLY in an isolated session: `herdr --session lab server`, `$env:HERDR_SOCKET_PATH =
  "$env:APPDATA\herdr\sessions\lab\herdr.sock"`, end with `herdr --session lab server stop`.

## 4. Design (decided by the lead - follow it)
**Shape:** one Go program, `-ldflags "-H=windowsgui -s -w"` (no console), embedded web assets, builds
to ONE file `app\HerdrDeck.exe` (target < 12 MB). Start: pick a free port on `127.0.0.1` only, random
per-run token in the URL (`/?t=<token>`; every request must carry it; reject others; no CORS), start the
server, then open the window with Edge in app mode: `msedge.exe --app=http://127.0.0.1:<port>/?t=<token>
--user-data-dir="%APPDATA%\herdr\deck-profile" --window-size=1400,900` (find msedge via App Paths /
Program Files (x86)\Microsoft\Edge\Application). Single instance (named mutex): a second double-click
focuses/reopens the window of the first instance. When the window closes, the exe exits after ~3 s
without clients. If Edge is missing: show a clear Windows message box (no silent failure).
**Backend:** polls Herdr through the CLI (`herdr api snapshot`, `herdr agent list`, `herdr pane read`)
only for what is on screen, ~700 ms for visible panes, ~2 s for the rest; pushes DIFFS to the page over
Server-Sent Events (stdlib, no websockets); input comes back over small `POST /api/...` calls. Every
Herdr call has a timeout; every error reaches the UI as a visible message (no silent catch). Herdr is
found at the stable path; env `HERDR_SOCKET_PATH` / `--session` honoured so tests can use the lab session.
Must work when Herdr is not running (clear empty state with a "Start Herdr" button that runs the
launcher exe) and recover when it starts later.
**UI (dark, calm, fast, keyboard friendly):**
- Left rail: workspaces and tabs (status dot per agent: working / idle / blocked / unknown).
- Main: a grid of pane cards like the screenshot (1, 2 or 3 columns, drag-free: click a card to
  focus/zoom it). Each card: title (agent name, folder, model, ctx %, tokens, from Herdr's `tokens`),
  live screen text in a monospace view that keeps scroll position, ANSI colours kept if Herdr gives
  them (otherwise plain), and a prompt box: Enter sends via `herdr agent prompt`, Shift+Enter newline;
  buttons: Esc, Ctrl+C, Enter, Up/Down/Tab/y/n (`pane send-keys`), Close pane (confirm), Rename.
- BLOCKED panes (approval, question, sign-in, folder trust) get a loud highlight and a toast; the
  Deck NEVER answers them by itself - only the owner's click does (CLAUDE.md rule 2).
- Top bar: "+ Agent" (profiles from `agents.json` with model/effort dropdowns, account choice;
  starts via `herdr-agents.ps1 start ...`; if that fails with `agent_pane_busy`, use the documented
  fallback `herdr pane run`, LESSONS #60), "+ Shell", usage/limits per Claude account (from
  `herdr-agents usage`), a command palette (Ctrl+K) with: focus agent, ask agent, stop agent, open
  docs, reload Herdr config (`herdr server reload-config`). Never offers `herdr server stop`
  without a typed confirmation.
- Tiny footprint: no framework, no build step, under ~60 KB of JS+CSS, no timers when the window is
  hidden, no layout thrash (update only changed cards).
**Safety:** never expose secrets; never write files outside `%APPDATA%\herdr\deck-*`; no admin;
nothing flashes a console (every child process started hidden: `CREATE_NO_WINDOW`); localhost only.
**Known traps:** PowerShell/Go quoting when passing prompts (pass text via stdin or a temp file,
never string-built command lines); pane ids repeat between sessions - key everything by
(session, pane_id); Windows paths in JSON; long screens (cap at the Herdr viewport + 2000 lines).

## 5. Steps
1. Read the rules and the Herdr CLI/API; write down in `app/README.md` the exact calls you will use
   (verified by running them read-only on the live session or on the lab session).
2. Backend with a `Herdr` interface and a fake implementation for offline tests (`HERDR_BIN` override
   pointing at a test stub script/exe).
3. Frontend; run the exe against the LAB session with 2 real panes (e.g. two shells) and drive the
   whole flow: list, live update, prompt, keys, split/close, blocked highlight (simulate with a
   prompt that waits for input), Herdr stopped/started.
4. `app\build.ps1` (go vet, go test, build, print size + SHA256). Never commit binaries to git:
   add `app/HerdrDeck.exe` to `.gitignore`.
5. A Desktop/Start shortcut is NOT part of this task; the lead adds it.
6. Docs: FILES.md, HISTORY (next phase), LESSONS (durable facts: API quirks, Edge app-mode flags).

## 6. Definition of done (all must be true)
- [ ] `pwsh -File app\build.ps1` exits 0; `app\HerdrDeck.exe` exists, < 12 MB, no console window.
- [ ] `go vet ./... && go test ./...` pass; tests fail on a deliberately broken parser (mutation check,
      report it); AGENTS.md rule 4 applies (tests test behaviour).
- [ ] Against the lab session: the page lists panes with status/model/ctx, a prompt reaches an agent,
      keys work, close works, blocked state shows and is never auto-answered. Show the evidence
      (curl of `/api/state`, the SSE stream sample, a screenshot via Edge `--headless --screenshot`
      is allowed ONLY for this check and only against the lab session).
- [ ] Idle CPU of the exe < 2 % and RAM < 60 MB with 4 panes (measured, say how).
- [ ] `pwsh -File check.ps1` passes; all `tests/*.ps1` suites still pass.
- [ ] Anything not verified is written as "not verified".

## 7. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Checks: <each check from section 6 and its result, with numbers>
Not done / doubts: <list, or "none">
