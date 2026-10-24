# HANDOFF for the web / phone session (written 2026-10-24, owner closing the PC)

You are continuing the owner's personal Herdr setup from claude.ai on a phone: you only have this
GitHub repo (`coolman1984/herdr`, branch `main`), NOT the PC. Nothing here can be run from the web;
write code and docs, the owner syncs it back on the PC (pull) and runs it. Read in this order:
`CLAUDE.md` -> `AGENTS.md` -> this file -> `HISTORY.md` (last phases 33-44) -> `LESSONS.md` (#56-66).
Answer style (owner rule): plain words first, one-line TLDR, short bullets, end with the decision
the owner must take. Owner's first rule: every new experience goes into the docs by yourself.

## 1. What this repo is
The owner's Herdr toolkit on Windows 11 (PC name FAWZY, folder `I:\herdr`): Claude Code leads the
other agent CLIs (Codex, Antigravity/agy, Cline, Grok, Cursor, OpenCode) in Herdr panes via
`herdr-agents.ps1` + `agents.json`. `herdr-app/` is the upstream Herdr source (never edit it for a
quick fix; needs Zig + MSVC, not installed). Rules: no secrets handled by agents, owner answers
blocked agents / sign-ins / approvals, close finished panes, no browser tools, PowerShell 7 only,
nothing flashes a console (`conhost.exe --headless pwsh ...`).

## 2. What was done today (HISTORY phases 35-44)
- 35: all agent CLIs updated (codex 0.164.0, cline 3.0.73, opencode 1.19.4, gemini 0.65, kilo 7.8.17,
  qwen 0.26, grok 1.0.70, agy 1.4.7, cursor-agent 2026.10.22). Held back: openclaw (needs newer Node),
  claude-mem 13->15 and pnpm 11->12 (major), pm2 (service). C: was full (npm cache cleaned).
- 36-39: check.ps1 failures diagnosed and fixed (config blocks, skills, App Path); Hermes Agent and
  OpenClaw/Clawdbot removed from the PC (17 GB). Admin leftovers were removed via UAC.
- 40: ten Ctrl+B key bindings rebuilt to run scripts from `I:\herdr` (they pointed to a missing D: folder).
  The running Herdr must `herdr server reload-config` after config changes (no auto-reload).
- 41: signed in Claude 2040 + 1984, Grok, Cline Pass; Cursor + agy already in. `agents.json`: gemini
  default `gemini-4-argon-high`; new `cursor` profile (`composer-2.5`, `--force`; Cursor has NO "Composer high").
- 42: Desktop/Start `Herdr.lnk`, Startup `Herdr HUD.lnk`, `Stop Herdr.lnk`, PATH restored.
- Review: Sonnet 5.5 high reviewed the toolkit (4 medium + ~10 low findings); Codex fixed them
  (155/155 tests, 25 new tests fail on old code). check.ps1 now tests key targets and account skills.
- 43: `herdr-autosync.ps1` (commit + pull --rebase + push every 5 min; secret/size guards). Not installed
  as a service: Claude Code's auto-mode blocked persistence. Start by hand on the PC (section 6).
- 44: this handoff + the Herdr Deck app state below.

## 3. The Herdr Deck app (the big open project) - `app/` + `tasks/herdr-deck-*.md`
Goal (owner): a fast, light, beautiful web-tech control app in ONE double-click `HerdrDeck.exe` that
shows and controls exactly what the Herdr terminal shows (same live session through the Herdr
CLI/socket): all current Ctrl+B features + all `herdr-agents` commands, better and more useful, 60 fps,
memory-managed, award-level design based on the owner's `I:\Animation` studio (Ink Studio dark: bg
#0C0D10, ink #F2F0EA, accent lime #D4FF3A, Alexandria + Space Mono, 18px radius; Paper light theme).
Specs (all binding, read them): `tasks/herdr-deck-app.md` (main), `-split.md` (backend Codex /
frontend Opus ownership + the 20-feature list P0-P2), `-parity.md` (everything the setup has today +
"better" + beauty bar), `-performance.md` (60 fps, numeric budgets, memory rules), `deck-feature-research.md`
(26 researched extra features; build the 10 marked BUILD NOW, avoid the 5 anti-features),
`deck-design-brief.md` (design tokens/components from I:\Animation).
Stack: Go 1.26 stdlib only (backend, `app/internal/deck`, `app/main.go`, `app/build.ps1`), vanilla
JS/CSS front end in `app/web` embedded with go:embed, window = Edge `--app` mode. Backend contract:
`app/API.md`; frontend asks in `app/API-REQUESTS.md`; status per feature in `app/FEATURES.md`;
numbers in `app/PERF.md`.
CURRENT STATE: see the "Final status" block at the end of this file (filled from both builders'
last reports) and `app/FEATURES.md`, `app/API.md`, `app/PERF.md`.
Known flaws from the lead's mock screenshots (headless Edge on `app/web/index.html?mock=1` served by
`python -m http.server`): light theme did not apply; the blocked-agent toast overlaps a card header;
the help icon is cut off at 1440 px; front end ~190 KB vs the 150 KB budget.

## 4. NEXT STEPS (in order) - what a web session can do now
1. Read `app/FEATURES.md` + `app/API.md` + `app/PERF.md`; fix the doc gaps; keep them true.
2. Frontend (`app/web/**`): fix the four flaws above; finish P0/P1 features against `app/web/mock.js`
   (`?mock=1`); build the BUILD NOW features' UI; keep < 150 KB, no framework, no CDN.
3. Backend (Go, `app/internal/deck`): finish endpoints listed NOT STARTED/PARTIAL in `app/API.md`;
   the guards (readiness, ownership, waves, account, route, scope drift, delivery, recovery, citation).
   You cannot run `go test` from the web: write the code, say "not verified" in the docs.
4. When the owner is back at the PC: `git pull`, `pwsh -File app\build.ps1`, run the exe against an
   isolated lab session (`herdr --session lab server`), then a Sonnet 5.5 review + a Codex performance
   audit (soak test `app/tests/soak.ps1`, `?perf=1`), then a Desktop shortcut.
5. Other open items: the three paste ideas (Windows Terminal ctrl+v is bound; Herdr takes plain
   right-click; use Ctrl+Shift+V / Shift+Insert / Shift+right-click) - the Deck is meant to give real
   copy/paste; `herdr-agents start` fails with `agent_pane_busy` (workaround: `herdr pane run <pane>
   "<command>"`, LESSONS #60/#65); account skills must be synced after skill edits
   (`pwsh -File sync-skills.ps1`).

## 5. Facts a web session must not forget
- Models: Codex = GPT-6.1-Sol (`-m gpt-6.1-sol -a never -s workspace-write -c model_reasoning_effort=medium`),
  Claude helpers run on the owner's OTHER accounts (`CLAUDE_CONFIG_DIR=%USERPROFILE%\.herdr-accounts\claude\1984|2040`),
  Sonnet 5.5 = `claude-sonnet-5-5`, Opus 5.5 = `claude-opus-5-5`, Antigravity Argon High = `gemini-4-argon-high`.
- The lead never uses its own Claude subagents; research is headless (Codex `--search exec`, Gemini).
  Codex `-s danger-full-access` only when the owner grants it for one task.
- Never read or print `.credentials.json` / `auth.json`; sign-ins are typed by the owner in the pane
  (Grok/Cline use a device code - no paste needed; Claude needs a paste or sign in from a normal PowerShell
  with `CLAUDE_CONFIG_DIR` set).
- Pending owner-only items: none blocking. Start autosync by hand (section 6).

## 6. When back on the PC (owner commands)
- Pull what the web session did: `git -C I:\herdr pull --rebase`.
- Start the 5-minute sync loop: `! conhost.exe --headless pwsh -NoProfile -File I:\herdr\herdr-autosync.ps1`
  (status: `pwsh -File I:\herdr\herdr-autosync.ps1 -Status`; log `%APPDATA%\herdr\autosync.log`).
- Reload Herdr's config after config edits: `herdr server reload-config`.
- Build the Deck: `pwsh -NoProfile -File I:\herdr\app\build.ps1` -> `app\HerdrDeck.exe`.

## Final status of the two Deck builders (filled at shutdown)

**Backend (Codex GPT-6.1-Sol medium) - PARTIAL, safe to continue.** `go build`, `go vet`, `go test`
and `app\build.ps1` pass (30 backend tests pass, 1 lab-only test skipped); `check.ps1` had 1 failing
step at stop time (see `app/README.md` "STOP HANDOFF"). Authoritative status per endpoint: the
"STOP HANDOFF" blocks at the top of `app/API.md`, `app/PERF.md`, `app/README.md`. Herdr's public API on
Windows is newline-delimited JSON over a named pipe (LESSONS #69). Needs: six new endpoints for the
ten BUILD NOW guards; a bug where the "blocked" status flickers every ~1.5 s while the screen is
unchanged (the frontend hides it with one popup per pane per 30 s).

**Frontend (Claude Opus 5.5 medium, account 2040) - PARTIAL.** Work in `app/web/**` and
`app/FEATURES.md` (per-feature evidence, test log, next steps at the end). Measured: slowest frame
10.1 ms (no frame over 16.7 ms), first paint 152-164 ms, hidden window = 0 timers/redraws/requests.
NOT met: front end is 206 KB vs the 150 KB budget (62 KB compressed + 30 KB font). Layout flaws: at
900 px the page scrolls sideways by 66 px; help icon cut off at 1440 px; blocked popup overlaps a
card header; check the light theme (it applied in real-session screenshots, not in the mock one).
Not done: the 5-minute stress run and its part of `app/PERF.md`. Not tested live: starting agents from
the UI, drag-resize/swap, spaces side by side, stop-everything, layout-builder writes, copy screen,
Windows notifications; the 10 guard features work only on mock data until the backend endpoints exist.

**Lead (Claude Sonnet 5.5):** everything above is committed and pushed by `herdr-autosync.ps1 -Once`
at shutdown. All lab Herdr sessions were stopped. Panes were closed at shutdown.
