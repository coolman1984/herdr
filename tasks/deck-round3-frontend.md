# Herdr Deck round 3 - FRONTEND (Claude Sonnet 5.5 high)

Goal (owner): a fast, beautiful Windows interface for Herdr so the owner does not have to work only in
PowerShell and the CLI. The Deck exists (Go backend + vanilla web front end in one `HerdrDeck.exe`);
you finish and polish the front end. Opus 5.5 built it yesterday on the owner's other PC; you continue.

Read first: `CLAUDE.md`, `AGENTS.md`, `tasks/herdr-deck-app.md`, `tasks/herdr-deck-split.md`,
`tasks/herdr-deck-parity.md`, `tasks/herdr-deck-performance.md`, `tasks/deck-design-brief.md`, then
`app/FEATURES.md` (your file - its STATUS, Known flaws and NEXT STEPS at the end are your starting
point), `app/API.md` (backend contract, its STOP HANDOFF first), `app/API-REQUESTS.md`.

## This PC (M-LABIB03)
- Repo `D:\WORK\Software Development\GitHub\HERDR` (old docs say `I:\herdr`, the other PC).
- Build: `$env:PATH = "D:\tools\go\go1.26.10\go\bin;$env:PATH"; pwsh -File app\build.ps1` (passes now).
- RAM and disk are short: one headless Edge at a time, throwaway profile in your temp folder, delete
  screenshots you do not keep; close Edge and any Deck/lab process you start.

## You own (write only these)
`app/web/**`, `app/FEATURES.md`, `app/API-REQUESTS.md`. The backend (Codex, the pane next to you)
owns everything else in `app/` plus HISTORY/LESSONS/FILES. Need a backend change: one block in
`app/API-REQUESTS.md` (endpoint, JSON in/out, why); re-read `app/API.md` before each work step. Put
your HISTORY/LESSONS lines under "For HISTORY/LESSONS" in FEATURES.md.

## Work, in this order
1. Layout flaws: no sideways scroll at 900 px; help icon visible at 1440 px; loud toasts never cover a
   card header; light theme applies via every path (palette, Ctrl+B s, `?theme=light`, stored choice).
   Prove with headless-Edge screenshots at 900 / 1100 / 1440 / 2560 px, dark and light, mock and lab.
2. Size: product JS+CSS+HTML under 150 KB raw (measured 206-257 KB). Remove duplication
   (panels.js vs parity.js), dead code, unused CSS. No feature lost - list what you merged.
3. Speed and feel (the owner's main wish: FAST): first paint, input-to-echo latency, 60 fps under the
   8-pane stress mock (`?mock=1&stress=8&perf=1`), 5-minute stress run. PERF.md is backend-owned: put
   your numbers in FEATURES.md and ask the backend (API-REQUESTS.md) to copy them into PERF.md.
4. Wire the backend's new routes as they land (R19-R24, `input_not_empty` 409 -> a clear toast "<agent>
   has text in its input box - not sent", never retry by itself) and turn mock-only BUILD NOW rows live.
5. Live-verify in the LAB session (never the owner's): drag resize/swap, columns, spaces join/separate,
   layout builder, stop-all typed confirm, copy screen, notifications. Each row in FEATURES.md gets
   WORKS (with evidence) / UI READY (waits for R..) / NOT DONE.

## Rules
- Vanilla JS/CSS, ES modules, no framework/CDN/npm, no timers when hidden, never auto-answer a blocked
  pane, never show secrets. Design tokens from `deck-design-brief.md` (Ink Studio dark default, Paper light).
- No git writes. No other agents, no subagents. AGENTS.md rules apply: prove each claim with its output.
- Work through the list without stopping between items; stop only when BLOCKED or done. Do not start
  review loops of your own work beyond one self-check.

## Answer
DONE | PARTIAL | BLOCKED
Changed: files (+ bytes before/after)
Checks: screenshots taken (paths), perf numbers, live-verified rows
Not done / doubts: list
