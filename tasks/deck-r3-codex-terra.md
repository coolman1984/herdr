# Task: Herdr Deck round 3, phase 2 - live verification and evidence (Codex gpt-6-astra, medium)

## 1. Role and goal
You are a senior test engineer working for the lead (Claude Opus). The lead is changing the Deck's product
code (`app/web/**`, `app/internal/**`) AT THE SAME TIME; you do not change product code. A Claude Sonnet
agent watches your pane and reports to the lead.
Goal: prove, in an isolated lab Herdr session, which Deck features really work, record the evidence, and
report every bug precisely so the lead can fix it. Every row of `app/FEATURES.md` and of the status table
at the top of `app/API.md` ends as WORKS (with evidence), PARTIAL (exact reason) or NOT DONE (exact reason).
Why: the round-3 backend is unit-tested only; most UI rows were checked only in mock mode.

## 2. Context you need
- Repo `D:\WORK\Software Development\GitHub\HERDR`. Read first: `CLAUDE.md`, `AGENTS.md` (delegate rules),
  `LESSONS.md` #20, #24, #39, #56, #75-#78, then `app/FEATURES.md` (STATUS / Known flaws / NEXT STEPS),
  `app/API.md` (status table + `## Round 3 (2026-10-26)`), `tasks/deck-r3-ds-backend-answer.md`.
- Build: `pwsh -NoProfile -File app\build.ps1` (Go is found by the script). Only build when the lead says
  in `tasks/deck-r3-lead-notes.md` that a new build is ready, or once at the start. NEVER run a build while
  another Go build runs (RAM). If the build fails because of the lead's in-progress edits, wait 2 minutes
  and retry; do not fix product code.
- **UI harness: `app/tests/deck-ui.ps1`** (you may extend it). Dot-source it:
  `. app\tests\deck-ui.ps1; try { Start-DeckUi -Lab; Set-DeckSize 1440 900; Open-DeckPage '?theme=dark'; Invoke-DeckJs '...'; Save-DeckShot 'name' } finally { Stop-DeckUi }`
  It sets APPDATA to `app\tests\runtime` (isolated lab session `lab`), runs `HerdrDeck.exe --headless`,
  ONE headless Edge with a throwaway profile over CDP, keeps an SSE client alive, and stops only what it
  started. `Invoke-Lab @('pane','list')` gives lab Herdr JSON. Lab pane patterns: `app/tests/verify-lab.ps1`
  (workspace create, split, `report-agent`, `report-metadata`, the offline `agentstub`).
  Mouse/keys over CDP: `Send-Cdp 'Input.dispatchMouseEvent' @{...}`, `Input.dispatchKeyEvent`.

## 3. Exact scope
- You may change: `app/tests/*.ps1` (new test scripts welcome, e.g. `app/tests/verify-ui.ps1`),
  `app/tests/shots/*.png` (<= 25 kept), `app/FEATURES.md`, the STATUS column/evidence of the table in
  `app/API.md`, `app/PERF.md` (FRONTEND section).
- Create: `tasks/deck-r3-codex-bugs.md` (one block per bug, newest last: id B1.., route/feature, exact
  steps, expected, observed with output/screenshot path, severity) and `tasks/deck-r3-codex-terra-answer.md`.
- Do NOT touch: `app/web/**`, `app/internal/**`, `app/main.go`, `app/build.ps1`, `HISTORY.md`,
  `LESSONS.md`, `docs/**`, `agents.json`, `herdr-app/`, `skills/`, `.ps1` outside `app/tests/`,
  `app/tests/fixtures/**`.
- Do not: git writes; start real AI agents (use shells and the offline `agentstub`); touch the owner's
  live Herdr session (always the harness / `--session lab` with APPDATA = `app\tests\runtime`); read
  auth/credential files; install anything; open visible windows; use any browser except via the harness.
- PowerShell 7 trap: `Start-Process -ArgumentList` does not quote paths with spaces; use the harness's
  `Start-UiProcess`. Every wait loop needs a timeout. Temp files under `app\tests\tmp-*`, deleted at the end.

## 4. Work items, in this order
1. **Lab verification script** `app/tests/verify-ui.ps1`: repeatable, prints PASS/FAIL per check, cleans up.
   Cover every FEATURES row and every API.md PARTIAL row that runs without real AI agents/accounts:
   prompt/paste/keys, rename/close, split/columns, drag swap + divider resize, tabs/workspaces
   rename/close/focus, spaces join/separate, layout build/save, folder picker, stop-all typed confirm (it
   stops the LAB server only), copy screen (read `navigator.clipboard` back), close finished, health/usage
   panels, dispatch ledger `/api/dispatches`, reservations `/api/reservations`, git `root`/`paths`, and the
   `input_not_empty` guard live: an agentstub pane whose input box holds typed text (Claude-style rule +
   U+276F + U+00A0, see `app/tests/fixtures/input/`) must show the toast "<agent> has text in its input
   box - not sent" and cause ZERO writes (read the pane back).
2. **Layout check** at 900/1100/1440/2560, dark+light, mock and lab: `scrollWidth <= innerWidth`, help
   icon inside the window, no toast over a visible card header. Screenshots of 1440 dark/light lab.
3. **Speed numbers** (mock stress `?mock=1&stress=8&perf=1`, `window.__deckPerf.report()`; cold and warm
   first load): run them after each lead build note; put the table in PERF.md FRONTEND.
4. **Every bug** -> a block in `tasks/deck-r3-codex-bugs.md` at once (the lead reads it while you work).
5. **Docs**: FEATURES.md rows + Measurements + Known flaws; API.md status rows with evidence.

## 5. Steps
1. Read section 2. Build once (paste exe size). 2. Items 1-5 in order; re-run `verify-ui.ps1` whenever
`tasks/deck-r3-lead-notes.md` announces a new build. 3. Final run, cleanup, answer file.
Do not loop: if the same check fails twice for the same reason, file it as a bug and move on.

## 6. Definition of done
- [ ] `pwsh -NoProfile -File app\tests\verify-ui.ps1` runs end to end; paste its PASS/FAIL summary.
- [ ] Layout table (8 rows) pasted; perf table pasted.
- [ ] Every FEATURES.md row and API.md table row has WORKS / PARTIAL (reason) / NOT DONE (reason).
- [ ] No lab Herdr, Deck or Edge process of yours running; no `app\tests\tmp-*` left.
- [ ] `git status --short`: your changes only in the section-3 files.

## 7. Answer format
Write `tasks/deck-r3-codex-terra-answer.md` and reply with exactly the same text:
```
DONE | PARTIAL | BLOCKED
Changed: <files>
Items: 1..5 each -> done / partial (why)
Checks: verify-ui summary; layout table; perf table; rows WORKS/PARTIAL/NOT DONE counts
Bugs filed: <ids + one line each>
For HISTORY/LESSONS (the lead writes them): <durable facts with proof>
Not done / doubts: <list>
```
