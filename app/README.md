# Herdr Deck

## STOP HANDOFF — owner shutdown, 2026-10-24

**PARTIAL; safe to continue from these files.** Work stopped immediately for shutdown.
Read [API.md](API.md) first: its STOP HANDOFF matrix labels **every endpoint and guard**
DONE / PARTIAL / NOT STARTED, gives evidence, exact broken frontend contracts and ordered next steps.
Read [PERF.md](PERF.md) for measured budgets; [API-REQUESTS.md](API-REQUESTS.md) is Opus's request list.
These handoff blocks supersede older design notes below them.

- **DONE:** app/build.ps1 exits 0 (Go format, vet, tests and GUI build); direct go test exits 0;
  30 behavioral tests pass, 1 stopped-lab test skipped, direct socket test passed earlier.
  Deliberately broken parser: **2 behavioral tests fail** (source never edited).
- **DONE:** real isolated lab shell prompt/paste/Enter, offline foreground-agent transport,
  blocked state and no auto-answer, HUD model/ctx, rename/down-split/close, safe mode 403,
  second-instance reopen and stopped/started recovery. Evidence: tests/evidence/lab-results.txt,
  state.json and sse-sample.txt. Actual paid AI-agent startup/sign-in remains NOT VERIFIED.
- **DONE:** seven existing PowerShell suites **155/155**. **PARTIAL:** check.ps1 exits 1:
  missing/broken external App Path; concurrent docs contain duplicate Phase 43 and lesson 66.
  Those other-agent additions were preserved. External repairs were not attempted.
- **PARTIAL:** parity/toolkit endpoints and the ten BUILD NOW safeguards exist in varying
  states; API.md lists each. R19–R24 still need wire-format completion. Research routes have
  no verified adapter. Main-account normalization and named-session launcher need review.
- **PARTIAL:** performance. Initial 10-minute soak failed RAM/traffic; improved interrupted
  160-second run reached max **15.312 MiB**, CPU peak **3.787% of one core**, **1.81 KB/s total**,
  zero goroutine growth. Handle growth +20 needs follow-up. No final 10-minute/hour or 60-fps proof.
- **PARTIAL:** frontend is **257,316 bytes**, above 150 KB; Opus owns these files. Screenshots,
  visible Edge launch/focus/close, font cache headers and full parity UI still need verification.

Final exe: **8,573,440 bytes / 8.18 MiB**, GUI subsystem=2.
SHA256: `2E3D42A9BC5E5DFA0E5FB1BCD33ECE8B1637EB280E6D21825D9BDA3A48E4A491`.
Its embedded frontend reflects the latest build; later frontend edits require rebuilding.

**Cleanup DONE:** workspace-only lab reports not_running; own headless Deck PID 12796 was stopped;
the interrupted soak runner exited. No owner pane/session was stopped. The soak finally block
reported server_not_running because the owner-requested shutdown stopped the lab first.
No git commands/commits were made by this backend task.

**NEXT STEPS:** (1) reconcile API.md's R19–R24 mismatch list and critical account/session guards;
(2) Opus trims frontend and checks its buttons; (3) build, lab verification, mutation and full
600-second soak; (4) independent real-window/performance review; (5) lead resolves external
App Path and duplicate document numbering, then reruns check.ps1. Do not repair owner settings
or restart the live session as a shortcut.

---

One window on the same Herdr session. Build with `pwsh -File app/build.ps1`, then double-click
`app/HerdrDeck.exe`. No install, package downloads or console. Close Deck to leave Herdr running.

## Verified command discovery (2026-10-24)

Stable executable: `%LOCALAPPDATA%/Programs/Herdr/bin/herdr.exe`. `HERDR_BIN` can override
it with an executable test stub. `--session NAME` selects that session for every CLI call;
otherwise the inherited `HERDR_SOCKET_PATH` is preserved. Every pane key includes the socket/session.

Commands inspected with their own `--help`: `pane read/send-text/send-keys/split/close/run/rename`,
`agent prompt/get/focus`, `workspace create`, `tab focus`. `api schema --json` reports protocol 22.
Lab read-back: `status server --json` reports running, `api snapshot` returns
`result.snapshot.{panes,agents,workspaces,tabs}`, `agent list` returns `result.agents`.
Live reads failed with PermissionDenied: the owner's server is elevated; Deck does not change it.

Exact calls used by the backend (arguments are separate process arguments, never shell interpolation):

- `api snapshot`, `agent list` for navigation, lifecycle and display tokens.
- `pane read ID --source recent --lines N --format ansi` (viewport + 2000 rows maximum).
- `agent prompt NAME TEXT` for submission-only agent prompts; `pane run ID TEXT` for shell submissions.
  The CLI rejects `--timeout` unless `--wait` is also set; Deck enforces its own process timeout.
- `pane send-keys ID KEY`, `pane rename ID LABEL`, `pane close ID`.
- `pane split ID --direction right --cwd FOLDER --no-focus`; `workspace create --cwd FOLDER --label Deck --no-focus` when empty.
- `agent focus NAME`, `tab focus ID`, `server reload-config` only on explicit user actions.
- Hidden PowerShell bridge to `herdr-agents.ps1 start PROFILE -Pane ID ...`, `stop NAME`, `usage`, `accounts`.
  Bridge input is JSON on stdin. Busy-pane fallback uses `pane run` with an encoded PowerShell launch
  built from the selected profile; existing named account folders only. No trust/sign-in answers.
- Start Herdr invokes the sibling `launcher/HerdrLauncher.exe`, with a clear error if absent.

## Lab isolation

The ordinary lab start could not create data under the real roaming profile (sandbox access denied).
Tests set APPDATA to `app/tests/runtime` before starting `herdr --session lab server`, so its socket
is `app/tests/runtime/herdr/sessions/lab/herdr.sock`. This is a separate, workspace-only lab server.
The owner's socket, settings and panes are never changed. Test output is under `app/tests`.

## Window and security

Loopback-only random port and per-run token, checked on every resource, SSE and API request.
Mutations require JSON and same-origin requests; no CORS. Assets are embedded. No arbitrary file
or command API. Pane text is untrusted and rendered as text with safe ANSI color spans. Deck does
not read credential files. Its profile and single-instance record live under `%APPDATA%/herdr/deck-*`.
Tokens in the window URL are local access credentials; do not share the URL or instance record.

Edge app mode uses `--app=URL --user-data-dir=.../herdr/deck-profile --window-size=1400,900`.
`--headless --address-file PATH` starts only Deck's HTTP server for isolated checks; address files
contain a temporary access token and should be removed after tests. No production shortcut is created.
