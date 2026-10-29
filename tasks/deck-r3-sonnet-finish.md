# Task: Herdr Deck round 3, phase 2 - finish the Deck (frontend + live lab verification)

## 1. Role and goal
You are a senior full-stack engineer (vanilla JS/CSS + Go) working for the lead (Claude Opus).
You are the ONLY worker on the Deck now; the lead reviews and fixes after you.
Goal: finish Herdr Deck (`app/`): a fast, good-looking Windows control panel for Herdr, so the owner
(who is not technical) does not have to use PowerShell. Every row of `app/FEATURES.md` and of the
status table at the top of `app/API.md` ends as WORKS (with evidence) or NOT DONE (with the reason).
Why: the backend for round 3 has just landed and is unit-tested only; the frontend still has open
layout/size flaws and many rows that were only checked in mock mode.

## 2. Context you need
- Repo `D:\WORK\Software Development\GitHub\HERDR`. Read first: `CLAUDE.md`, `AGENTS.md` (delegate
  rules), `LESSONS.md` #20, #24, #39, #56, #75-#78, then `app/FEATURES.md` (STATUS, Known flaws, NEXT
  STEPS at the end are your starting point), `app/API.md` (status table + `## Round 3 (2026-10-26)`),
  `app/API-REQUESTS.md`, `tasks/deck-design-brief.md` (design tokens: Ink Studio dark default, Paper light),
  `tasks/deck-r3-ds-backend-answer.md` (what the backend just added).
- Go 1.26.10 portable, not on PATH: `D:\tools\go\go1.26.10\go\bin\go.exe`. Full build + vet + tests +
  exe: `pwsh -NoProfile -File app\build.ps1` (it finds Go itself). Fast loop:
  `$env:GOCACHE="$PWD\app\tests\go-cache"; $env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; Push-Location app; cmd /c "D:\tools\go\go1.26.10\go\bin\go.exe test ./internal/deck/ -count=1 > %TEMP%\deck-test.txt 2>&1"; Pop-Location`
  then read the file (PowerShell hides go errors otherwise - LESSONS #77). NEVER two Go builds at once.
- Frontend: `app/web/*` (vanilla ES modules, embedded into `HerdrDeck.exe`). Mock mode `/?mock=1`
  (`&stress=8&perf=1` for the stress mock). Backend: `app/internal/deck/*.go`.
- Test stub agent: `app/tests/agentstub` (a fake agent CLI) - use it for agent-like panes in the lab.

## 3. Exact scope
- You may change: everything under `app/` (`web/**`, `internal/deck/**`, `API.md`, `FEATURES.md`,
  `API-REQUESTS.md`, `PERF.md`, `build.ps1`, tests). Keep every Go change covered by a Go test.
- Create: `tasks/deck-r3-sonnet-finish-answer.md` (section 7).
- Do NOT touch: `HISTORY.md`, `LESSONS.md`, `docs/**`, `agents.json`, `herdr-app/`, `skills/`, any
  `.ps1` outside `app/`, `app/tests/fixtures/**` (read only).
- Do not: git writes (commit/add/stash/checkout/reset); start real AI agents (Claude, Codex, Cline ...)
  - use shells and `agentstub`; touch the owner's live Herdr session; read auth/credential files;
  install anything; open visible windows.
- Browser exception to AGENTS.md rule 7 (the lead allows it for this task only): ONE headless Edge at a
  time over CDP from a script, throwaway profile under `app\tests\tmp-edge\`, closed after each batch.
  RAM is ~4 GB free: never more than one Edge, one Deck and one lab Herdr at once.
- C: has ~2.7 GB free: keep every temp file under `app\tests\tmp-*` (D:) and delete it at the end;
  keep at most the final screenshots (`app\tests\shots\`, <= 25 PNGs).

## 4. Design (decided by the lead - follow it)
### 4.1 Lab session (never the owner's)
Start: `& "$env:LOCALAPPDATA\Programs\Herdr\bin\herdr.exe" --session lab server` hidden
(`conhost.exe --headless pwsh -NoProfile -Command ...` or `Start-Process -WindowStyle Hidden` on
herdr.exe itself); point tools at it with
`$env:HERDR_SOCKET_PATH = "$env:APPDATA\herdr\sessions\lab\herdr.sock"`; run the Deck as
`app\HerdrDeck.exe --headless --session lab` (read `app/main.go`/`run.go` for its flags and port);
end with `herdr --session lab server stop` and stop the Deck process you started. Stop only what you
started (by PID).
### 4.2 Work items, in this order
1. **Layout flaws** (FEATURES Known flaws 2-5): no sideways scroll at 900 px; help icon visible at 1440 px;
   loud toasts never cover a card header (move them below the top bar / bottom-right); light theme
   applies via every path (palette, Ctrl+B s, `?theme=light`, stored choice). Prove with headless-Edge
   screenshots at 900 / 1100 / 1440 / 2560 px, dark and light (mock), plus 1440 dark+light in the lab.
   Also check: `document.documentElement.scrollWidth <= innerWidth` at each width (print the numbers).
2. **Size**: product JS+CSS+HTML in `app/web` (fonts excluded) under 150 KB raw (now ~196 KB without
   the font). Remove duplication (panels.js vs parity.js), dead code, unused CSS. No feature lost -
   list what you merged/removed with before/after bytes per file.
3. **Wire round 3 routes** (`app/API.md` Round 3 section) into the UI: reservations (R19), account
   guard 409 with `lead_account` (R20, profile `next` now picks an account server-side), floor fields
   (R21), research routes empty + refusal (R22), `dispatch_id` + 504 "delivery unknown" card - show
   the id, NEVER auto-retry; a Resend button only on the owner's click (R23), git `root` + `paths` (R24),
   and `input_not_empty` 409 -> toast "<agent> has text in its input box - not sent" (use the
   server's `message` field), never retry. Turn mock-only BUILD NOW rows live.
4. **Speed**: first paint, input-to-echo latency in the lab, 60 fps under `?mock=1&stress=8&perf=1`,
   plus one 5-minute stress run (frames > 16.7 ms, long tasks, heap start/end). Write the numbers in
   FEATURES.md Measurements and the FRONTEND section of `app/PERF.md`.
5. **Live-verify in the lab** every FEATURES row and every API.md PARTIAL row that can be exercised
   without real AI agents or the owner's accounts: drag resize/swap, columns, spaces join/separate,
   layout build/save, stop-all typed confirm, copy screen (read the clipboard back in Edge), close
   finished, folders, tabs/workspaces rename/close/focus, shell, prompt/paste with an agentstub pane
   whose screen has text in its input box (expect the 409 toast and zero writes), dispatch ledger.
   Rows that need a real agent/account/sign-in/notification stay PARTIAL with that exact reason.
6. **Bugs you find**: fix them (frontend or Go, with a Go test for Go), note each in the answer.
### 4.3 Status words
FEATURES.md and API.md rows: `WORKS` (evidence: what you did and what you read back), `PARTIAL
(reason)`, `NOT DONE (reason)`. Never WORKS from mock alone.

## 5. Steps
1. Read section 2. Run `pwsh -NoProfile -File app\build.ps1` once (baseline must pass).
2. Items 4.2.1 -> 4.2.6 in order, without stopping between them. After each item: build, quick check.
3. Final: `pwsh -NoProfile -File app\build.ps1`; stop lab/Deck/Edge you started; delete temp folders;
   write the answer file.
Do not start review loops of your own work beyond one self-check. If you re-edit the same thing twice
without progress, stop that item, mark it NOT DONE with the reason, and continue with the next.

## 6. Definition of done
- [ ] `pwsh -NoProfile -File app\build.ps1` exits 0 (paste exe size).
- [ ] `app/web` product bytes (fonts excluded) < 150,000 (paste the measured number and the command).
- [ ] scrollWidth <= innerWidth at 900/1100/1440/2560, dark and light (paste the numbers).
- [ ] Every FEATURES.md row and API.md table row has WORKS / PARTIAL (reason) / NOT DONE (reason).
- [ ] No lab Herdr, Deck or Edge process of yours still running; no `app\tests\tmp-*` left.
- [ ] `git status --short` shows changes only in `app/**` and your answer file.

## 7. Answer format
Write `tasks/deck-r3-sonnet-finish-answer.md` and reply with exactly the same text:
```
DONE | PARTIAL | BLOCKED
Changed: <files + bytes before/after for app/web>
Items: 4.2.1..4.2.6 each -> done / partial (why)
Checks: build exit + exe size; web bytes; scrollWidth table; screenshots kept (paths); perf numbers;
        rows now WORKS (count) / PARTIAL / NOT DONE
Bugs found and fixed: <list with proof>
For HISTORY/LESSONS (the lead writes them): <durable facts with proof>
Not done / doubts: <list>
```
