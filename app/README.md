# Herdr Deck

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
