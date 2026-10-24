# Herdr Deck API — backend contract

Backend owns this file. Frontend owns `API-REQUESTS.md`. Version 2, 2026-10-24.

## Frontend requests accepted — R1–R7

R1: implemented token cookie bootstrap. `GET /?t=TOKEN` sets `deck-token` HttpOnly,
SameSite=Strict, Path=/; all assets/API/SSE accept it. `X-Deck-Token` also accepted.
R2: State adds `herdr:{running,error}`, `profiles`, `blocked`, `revision`. Tabs retain
`workspace_id` as parent workspace. `tokens` keep Herdr's actual names (usually `ctx`, `out`).
R3: SSE sends `state` on connect/structure change, `panes` on changed cards, and `herdr`
on running/error changes. `diff` is retained for compatibility but frontend should consume
`panes` and `state`. Each panes event includes full changed pane objects and removed keys.
R4: `POST /api/visible {keys:[SESSION_PANE_KEYS]}` supported. **Keep SSE open while hidden**:
send keys:[] to pause polling; the open connection keeps Deck alive without any JS timers.
Closing SSE means the window closed and starts the required ~3-second exit countdown.
R5: all listed dedicated routes accepted, using the R5 JSON field names. The close/stop
routes represent the owner's confirmed button action; UI must show confirmation before POST.
Legacy `/api/action` still requires confirm:true. Key aliases Escape/C-c/Enter/Up/Down/Tab
are normalized. Agent start/shell splits the key target if supplied, otherwise creates a new
workspace. Focus can target a name or tab. `message` and `output` both carry action results.
R6: tasks, usage (`text` plus `output`), health, close-finished, signin/open queued next;
support status is recorded below as implemented. Inbox, broadcasts and presets remain frontend.
R7: tasks/docs/runs readers use allow-lists; check returns exit result even when checks fail.
Git mutations are forbidden; read-only git view is permitted by the split feature contract.
Lesson append conflicts with the task's file-write safety boundary for runtime data and is
deferred; it will return an explicit unavailable error, not pretend to save.

R17 fixed: metadata-only polls retain the blocked classification from the full cached screen;
the regression test covers a live trust footer while raw Herdr status changes to unknown.
R18: disk values are numeric GiB; UI rounding is appropriate. Jobs endpoint is available.

Implementation status: P0 R1–R5 implemented; P1 R6 tasks/paste/usage/health/close-finished
implemented; sign-in status is honest unknown for unsupported tools. R7 readers/check/lesson
append implemented (the later parity addendum explicitly authorizes append-only lessons).
Runtime script actions use the existing scripts' conventional data locations, as required
by the parity addendum; Deck-owned temp plans/settings stay under deck-*.

## Performance protocol update (binding addendum)

Polling reads now use the public Herdr NDJSON named-pipe API directly, with no CLI processes.
Verified source behavior: protocol 22 accepts ONE RPC and closes that connection. Thus one
persistent request/reply connection is impossible in this version; the client makes short,
serialized direct pipe calls. `events.subscribe` is persistent but cannot carry further RPCs
and its output matcher emits on match transitions, not every output change. No upstream edits.

Upcoming `panes` diff objects add `screen_version` and `screen_patch`:
`{key,...metadata,screen_patch:{from_version:3,version:4,start:10,delete:2,lines:["new row"]}}`.
Apply `lines.splice(start,delete,...lines)` only if the local version equals from_version;
on mismatch reload `/api/state`. New panes/full state retain `screen` plus screen_version.
Patch objects omit `screen`. Bounded queues resync with a full `state` if a client falls behind.
Heartbeat is now at most once per 15 seconds. Visible polling is 1.4 s, focused/streaming 280 ms,
background panes 7 s; nothing is polled when hidden. This supersedes the original 700 ms rule.

## Parity requests R8–R16 accepted

R8 preserves layouts, focused_pane_id, focused, revision and every token verbatim.
R9 endpoints/JSON exactly as requested; raw keys accept space, shift+tab, ctrl+a..z, f1..f12.
R10 folder browsing returns directory names only. Workspace open calls herdr-open-folder.ps1.
R11 columns calls herdr-columns.ps1 with the explicit pane. Layout build calls herdr-layout.ps1
with a Deck temp plan. Plan shape:
`{Where:0,Name:"work",WorkspaceId:"w1",Columns:[{Folder:"I:\\herdr",Type:"shell|new|existing",Value:"codex or pane ID",Model:"",Account:""}]}`.
Where=0 new tab in WorkspaceId; Where=1 new workspace. Presets preserve the script's saved
shape `{count,where,name,columns:[{folder,runs,model,account}]}`; existing agents save as shells.
R12 join/separate calls herdr-spaces.ps1; current record is read-only JSON.
R13 typed confirm must equal `stop`; backend invokes the unchanged Stop-HerdrServer function
from herdr-stop.ps1, including its stopped-state verification. It never auto-confirms a dialog.
R14 all helper verbs reuse herdr-agents.ps1; long actions return `{ok:true,id,job_id}` immediately.
SSE `result` has `{id,ok,status,message,output}`; `GET /api/jobs` retains up to 24 recent results.
At most four jobs run together. Ask supports fresh:true. `worktree:true` generates a unique
deck-* branch name; a string supplies a branch name. `profile:next` delegates selection to the tool.
Team presets: `claude-codex` and `claude-codex-gemini`; cwd absolute. Handoff uses the existing
template through the tool; supplied text is attached after template creation and reported path.
R15 headless `{profile,text,model?,effort?,account?,cwd?,timeout_sec?}` calls herdr-headless.ps1.
Trial `{profile,model?,effort?,run?,trial_root?,timeout_min?}` calls herdr-trial.ps1. No trials or
agents are started by verification. Sync is an explicit user action and reuses sync-skills.ps1.
Docs names GUIDE, KEYS, herdr-team, herdr-workspace accepted. KEYS calls herdr-help.ps1 -List.
R16 health adds per-pane process memory where readable, plus goroutines/RSS/handles for the soak.

Parity R8–R16 endpoints are implemented. Live script controls that would launch agents,
sign in, sync accounts or stop the owner's session are NOT run by verification.

## BUILD NOW safeguards (backend)

- `GET /api/tasks/validate?file=tasks/x.md` → `{ready,warnings,method,proof:"not verified"}`.
  This checks explicit scope/answer/check/stop wording, never invents instructions.
- `GET /api/reservations`; `POST /api/reservations {owner,repo,paths:["app/**"],release:false}`.
  Case-normalized absolute paths; overlapping file/subtree writers are refused. Release only
  that owner's record with `{owner,release:true}`. Shared dispatch bookkeeping, no filesystem lock.
- `GET|POST /api/launch/policy {max_active:4,min_ram_mb:512,min_disk_gb:0.5}`. No work is killed.
  Direct agent starts check the chosen capacity and free RAM/project-drive disk floors.
- `GET|POST /api/launch/queue` (POST uses the same fields as agent start). Returns a held launch ID.
  `POST /api/launch/release {id}` releases an explicitly chosen wave member when limits permit.
  Failed starts remain failed and are never automatically replayed. Queue is bounded at 32.
- Start adds `helper:true,lead_account:"2030"`: Claude helpers require an explicit OTHER known
  account (1984/2040); missing identity holds the launch. `purpose:"research",route:...` refuses
  browser/pane research; no verified server-search-only adapter is currently available in Deck.
- `POST /api/proof/audit {text,official_domains:["openai.com"]}` returns five proof slots and
  extracted citations. Mentioned evidence remains not verified. No URLs are opened.
- `POST /api/scope/baseline {id,repo,allowed:["app/**"]}`, then `GET /api/scope/check?id=...`.
  Read-only changed-path fingerprints distinguish pre-existing edits; unexpected changes
  produce a warning, never blame, rollback, commit, merge or deletion.
- `GET /api/deliveries` retains the latest 100 prompt dispatches. Each guarded prompt creates
  a dispatch ID; timeouts/errors after submission become `delivery unknown`. No automatic replay.
- Errors add `kind,next_step,cause` for recovery cards. Unknown causes remain unknown.

These endpoints are implemented and being verified; frontend integration is owned by Opus.
The safe-mode toggle blocks all launch/reservation/file writes. Proof audit and scope checks
are read-only. Retention: 4 jobs active, 24 recent, 32 scope baselines, 64 reservations,
128 paths per reservation, 512 changed files per scope comparison.

Static assets `/assets/*` have ETag and private immutable one-year cache headers; WOFF2
has `Content-Type: font/woff2`. They still require the cookie/token. HTML/API are no-store.
All paths below require `?t=TOKEN` (the token from the window URL), including assets and SSE.
For JS/CSS imports, append this token to resource URLs. No CORS. JSON POSTs only; a present
Origin must equal the Deck origin. Errors: HTTP 400/403/404/409/502 with `{"error":"message"}`.
The `session` value is a normalized socket identity, not just a pane ID. Never guess pane IDs.

## P0 (being implemented now)

`GET /api/state` returns:

```json
{"session":".../lab/herdr.sock","panes":[{"key":"...|w1:p1","pane_id":"w1:p1","workspace_id":"w1","tab_id":"w1:t1","cwd":"I:\\herdr","label":"Shell","name":"coder","agent":"codex","agent_status":"idle","title":"","tokens":{"model":"gpt-6.1-sol","ctx":"42%","out":"120"},"screen":"terminal text, optionally ANSI","error":""}],"workspaces":[{"workspace_id":"w1","label":"Work","agent_status":"idle","focused":true}],"tabs":[{"workspace_id":"w1","parent_workspace":"w1","tab_id":"w1:t1","label":"1","agent_status":"idle","focused":true}],"error":""}
```

`GET /api/events` is SSE. Initial `event: state` has the full state above. Then
`event: diff` contains `{"panes":[CHANGED_PANES],"removed":[PANE_KEYS],"error":""}`;
`workspaces` and `tabs` are present only when changed. Apply changed panes by `key`.
Heartbeat comments keep the connection open. Reconnecting always receives a complete state.
No prompt/approval is ever sent by polling. Status `blocked` and screen dialog guards are
display-only signals. Unknown model/context remains unknown, never guessed.

`POST /api/visibility` input `{"visible":["w1:p1","w1:p2"],"active":true}`.
Call when the selected tab, zoom, or document visibility changes. Active visible screens
poll at ~700 ms; other metadata at ~2 s. Hidden pages pause polling. Closing SSE releases
the client; Deck exits ~3 seconds after its last client disconnects (startup grace 15 seconds).

`POST /api/action` input always has `action`, with these additional fields:

| action | Fields | Effect |
|---|---|---|
| prompt | pane_id, text | agent prompt for an agent; pane run for a shell |
| keys | pane_id, key | one of esc, ctrl+c, enter, up, down, tab, y, n, left, right, backspace |
| paste | pane_id, text | literal pane send-text, never submits Enter |
| close | pane_id, confirm:true | closes this pane; UI must confirm |
| rename | pane_id, text | rename pane |
| focus | pane_id | focus agent or its tab; card zoom is local UI state |
| shell | pane_id (optional), cwd (optional) | split explicit pane, or new workspace when empty |
| start | pane_id (optional), profile, model, effort, account, cwd (all optional except profile) | herdr-agents start in a new shell; busy fallback via pane run |
| stop | pane_id, confirm:true | herdr-agents stop NAME; error if not owned by that tool |
| reload | none | server reload-config |
| launch | none | start sibling HerdrLauncher.exe |

Output: `{"ok":true,"output":"Herdr result or helper output"}`. A shell/start result may
include the resulting pane in its JSON output. Action IDs are validated against a fresh snapshot.
Prompts to blocked panes return 409; explicit key/paste buttons remain the owner's decision.
There is deliberately no server-stop endpoint.

`GET /api/profiles` → `{"profiles":[{"name":"codex","kind":"codex","model":"gpt-6.1-sol","effort":""}]}`.
`GET /api/accounts` → `{"output":"herdr-agents accounts output"}`.
`GET /api/usage` → `{"output":"herdr-agents usage output"}` (unknown/stale values retained as such).

## P1/P2 extensions

Frontend: append requested endpoint blocks to `API-REQUESTS.md` with priority and JSON shapes.
Backend will document each accepted endpoint here before implementing it. Until documented as
implemented, treat an extension as unavailable and keep the UI's visible not-done state.

## Safety constraints

Never send automatic approval, sign-in, trust or queued prompts to blocked panes. The read-only
toggle is enforced by the backend via `POST /api/read-only {"enabled":true}`; state-changing
requests then return 403. This endpoint itself can always turn the mode off by explicit click.
No arbitrary executable, arbitrary path reader, credential file reader, or shell endpoint.
Task/doc readers will use fixed allow-lists. Any stored settings/export must stay under
`%APPDATA%/herdr/deck-*`. No git mutations. All child processes hidden and time-limited.
