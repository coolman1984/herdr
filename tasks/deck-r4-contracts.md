# Deck round 4 - backend/frontend contracts (decided by the lead; both sides follow exactly)

Astra builds the backend side (`app/internal/deck/*.go`, documents each in `app/API.md` as R26-R35).
Sol builds the frontend side (`app/web/*`, including `mock.js` so `?mock=1` follows the same contract).
Report IDs (D#, P#) refer to `tasks/deck-r4-astra-report.md`. Field names are exact.

## C1 - per-page visibility (D3, P4)
- `GET /api/events` (SSE) sends `event: hello` with `{"client_id":"c-<n>"}` as its FIRST event, then
  `settings` (C2), then `state` as today.
- `POST /api/visible {"client_id":"c-3","keys":["session|w1:p1",...]}`: the keys that page shows. The
  backend keeps one set per SSE client, drops it when that client's stream closes, and reads screens for
  the UNION of all sets. A request without `client_id` uses the client id `legacy` (old pages keep working).
- Metadata/recovery (snapshot) polling runs every 2 s while at least one SSE client is connected, even
  when every visible set is empty. Hidden page: the page sends `keys: []` (not a disconnect).
- Screen events are sent only to clients whose set contains that key; other clients get no screen
  payload for it (metadata/status still go to all).
- Herdr offline: the state keeps the last known panes, `herdr.running=false`, `error` set, and every pane
  gets `"stale":true`. The page shows the panes dimmed with writes disabled and a banner until a fresh
  state arrives.

## C2 - settings event (D18)
- SSE `event: settings` `{"read_only":true|false}`: sent after `hello`, and to every client right after
  each `POST /api/read-only` change. The page applies it (Safe toggle, disabled write controls, clears
  queued raw input).

## C3 - raw typing (D2)
- `POST /api/pane/raw {"key":"session|w1:p1","text":"abc","keys":["enter"]}`: writes `text` (send-text)
  then each key (send-keys) in order, inside one request; requests for the same pane are serialized in
  arrival order. No input-box guard (it is the owner typing), allowed on blocked panes, refused in
  read-only (403). `keys` uses the same allowed key names as `/api/pane/keys`. Either field may be empty,
  not both. Response `{"ok":true}`.
- The page keeps ONE ordered queue per pane: each chunk is sent after the previous request resolved;
  switching raw mode to another pane first flushes and awaits the old pane's queue.

## C4 - reservations before work (D5)
- The page reserves BEFORE it dispatches: `POST /api/reservations {"task","owner","paths"}`; on 409 it
  shows the conflicts and sends nothing unless the owner picks "overlap", which re-posts with
  `"overlap":true` (stored as a shared reservation). If the dispatch then fails (not delivery-unknown),
  the page releases the new id with `POST /api/reservations/release {"id"}`.
- Backend: same owner + same task -> 200 with the existing id (idempotent). A different owner never
  replaces or drops an existing reservation. Ledger unreadable -> `GET` keeps `load_error` and `POST`
  returns 503 `{"error":"ledger_unavailable","message":"..."}`; the page shows "reservations
  unavailable", never "no conflicts".

## C5 - launches (D4, D12)
- Every route that starts agents (`/api/agent/start`, `/api/workspace/open`, `/api/layouts/build`,
  `/api/columns`, team routes in tools.go) passes ONE admission function in Go: plan expanded first,
  account rules and floors checked per agent, pending slots reserved atomically and released on
  failure/timeout (120 s) or confirmed by the next snapshot. Refusal -> 409
  `{"error":"capacity","message":"..."}` (or the existing account/floor error codes).
- `/api/agent/start` 200 response adds `"pane_id"` and `"name"` when the helper output contains
  `STARTED name=<n> ... pane=<id>` (absent otherwise, never guessed).
- The page drains its start queue one item at a time, awaiting each response; it attaches a task to
  the returned `pane_id` (no name-prefix guessing; no `pane_id` -> the task card says "attach manually").

## C6 - delivery unknown only for started writes (D24)
- Backend: 504 `{"error":"delivery unknown","dispatch_id":...}` ONLY when a prompt write was started
  and its acknowledgement is unknown. Read timeouts -> 504 `{"error":"read_timeout","message":...}`
  with NO dispatch_id; a write that never started -> 502 `{"error":"not_sent","message":...}`.
- Page: uncertain delivery iff (status 504 and `dispatch_id` present) or a network failure after the
  prompt POST was sent. Everything else is a plain failure (draft restored).

## C7 - input box state (D6)
- For `claude` and `codex` panes, a prompt/paste when the input region is NOT recognized -> 409
  `{"error":"input_unknown","message":"<name>: cannot see its input box - not sent"}`; recognized empty
  -> sent; occupied -> `input_not_empty` as today. Other agent kinds keep today's behaviour.
- Page: `input_unknown` handled like `input_not_empty` (warn toast, draft kept, no retry).

## C8 - preferences survive a Deck restart (D20)
- `GET /api/prefs` -> `{"prefs":{...}}`; `POST /api/prefs {"prefs":{...}}` replaces the whole object
  (JSON <= 64 KiB, else 413), saved to `%APPDATA%\herdr\deck-prefs.json` (atomic write).
- Page: loads prefs at start (backend values win over localStorage), writes through debounced 500 ms.
  When the event stream has been down > 10 s AND `GET /api/state` fails, show a persistent banner
  "Deck stopped or restarted - reopen it from the Herdr launcher" and disable writes.

## C9 - disks (D19)
- `/api/health` disk rows also include the volume of every launch/worktree folder in use (and D:).
  Unknown values are `null`, never 0. Launch floors use the volume of the launch folder; unknown capacity
  with a non-zero floor -> 409 `{"error":"disk_unknown",...}`. Page renders `null` as "unknown".

## C10 - counters (I1)
- `GET /api/debug/counters` (token like every route, always on, cheap) ->
  `{"goroutines","handles","rss_mb","sse_clients","sse_events","sse_bytes","polls","poll_errors",
  "poll_ms_last","poll_ms_max","rpc_waiting","resyncs","jobs_live"}`. Page: nothing required.
