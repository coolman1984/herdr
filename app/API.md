# Herdr Deck API — backend contract

## STOP HANDOFF — 2026-10-24, owner shutdown request (authoritative status)

Work stopped at the owner's request. No git commands or live-owner-session writes were performed.
**Current build:** build.ps1 PASS (format/vet/test/build, exit 0); direct go test PASS,
30 behavioral tests passed, 1 lab-only test skipped because lab is now stopped. The direct socket
test passed separately earlier. Exe: **8,573,440 bytes / 8.18 MiB**, SHA256
`2E3D42A9BC5E5DFA0E5FB1BCD33ECE8B1637EB280E6D21825D9BDA3A48E4A491`.
Evidence: tests/evidence/final-go-tests.txt, tests/evidence/lab-results.txt, tests/mutation-result.txt.
The following status table supersedes optimistic wording and older shapes later in this file.
DONE means backend behavior verified; it does not certify every frontend button.

| Endpoint(s) | Status | Evidence / limits |
|---|---|---|
| GET /api/state, /api/events | DONE | curl state, SSE sample, lab shell/fixture/HUD/blocked state; row-diff/scroll/overflow tests |
| GET /api/profiles | PARTIAL | Sanitized profiles plus account directory names implemented; final account selection not live-tested |
| GET /api/accounts, /api/usage | PARTIAL | Existing herdr-agents bridge; JSON/stdin bridge tested; actual account usage not live-tested |
| POST /api/visible, /api/visibility | DONE | Used by lab/soak; hidden polling pauses; keep SSE open |
| POST /api/read-only | DONE | Unit and real lab mutation returns 403 |
| POST /api/action | PARTIAL | Legacy action route implemented; core prompt/keys/rename/close tested, not every action |
| POST /api/reopen | DONE | Second instance reached existing headless instance, exit 0; visible Edge focus not verified |
| /api/pane/prompt, /keys, /text, /rename, /split, /close | DONE | Real lab computed output, offline foreground codex.exe fixture, blocked rejection 409, rename/down-split/close JSON read-back |
| /api/pane/resize, /swap, /move, /zoom, /focus | PARTIAL | Validated handlers implemented; live geometry/drag/focus not verified |
| /api/tab/create, /rename, /close, /focus; /api/workspace/rename, /close, /focus | PARTIAL | IDs validated against fresh snapshot; CLI help inspected; live UI not verified |
| POST /api/shell | PARTIAL | Handler implemented, split path verified; empty-session workspace path not separately exercised |
| POST /api/agent/start, /stop, /focus | PARTIAL | Existing helper plus busy fallback; no paid AI agents started for verification; main-account normalization and next-account guard need review |
| POST /api/herdr/start, /reload-config | PARTIAL | Implemented; not invoked; named-session Start currently uses default launcher and MUST be fixed before lab use |
| GET /api/tasks, /api/tasks/read, /api/docs, /api/runs, /api/runs/read | DONE | Allow-list/traversal/docs tests; task list real lab. Actual run-history content not inspected |
| GET /api/docs?name=KEYS | PARTIAL | Calls existing help -List; not live-tested. GUIDE and master skill documents supported |
| GET /api/health | PARTIAL | RAM/disks/process RSS/handles/goroutines measured; pane process memory implemented. R21 ram_free_mb alias still missing (actual field ram_available_mb) |
| POST /api/panes/close-finished | PARTIAL | Validates idle/done before any close; no live multi-close test |
| GET /api/signin, POST /api/signin/open | PARTIAL | Three read-only probes; unsupported tools honestly unknown; per-account probes and login panes not tested |
| GET /api/git | PARTIAL | Read-only status/stat handler; R24 paths/root fields NOT STARTED |
| POST /api/check | PARTIAL | Fixed script handler. Direct setup check has 3 external/concurrent-document failures; see README |
| POST /api/lessons/append | DONE | Temp-directory behavior test confirms append-only numbered content; real lesson not appended through UI |
| GET /api/jobs | PARTIAL | Bounded job results implemented; asynchronous live toolkit jobs not verified |
| GET /api/folders/recent, /list | PARTIAL | Directory names only, local paths, bounded listing; not UI-tested |
| POST /api/workspace/open, /api/columns | PARTIAL | Reuses original folder/columns scripts; not live-tested |
| GET /api/layouts; POST /api/layouts/build, /save | PARTIAL | Script-compatible plans/presets and atomic save; not live-tested |
| GET /api/spaces; POST /api/spaces/join, /separate | PARTIAL | Reuses spaces script; not live-tested |
| POST /api/stop-all | PARTIAL | Requires literal stop; invokes original verified-stop function; endpoint itself not exercised |
| GET /api/agents/tools, /status, /read, /diff, /history, /watch, /accounts, /usage, /collect | PARTIAL | Existing-script adapters; collect is an async job, requires names query; not all commands exercised |
| POST /api/agents/ask, /fanout, /team, /handoff | PARTIAL | Existing helper, bounded async jobs/temp plans; no actual agent dispatch or handoff tested |
| POST /api/headless, /api/trial, /api/sync-skills | PARTIAL | Fixed-script adapters, bounded jobs. Not run: they launch agents/change external account copies |
| /debug/pprof/* | PARTIAL | Token-protected, off unless --debug; not profiled |
| /assets/* cache, WOFF2 MIME | PARTIAL | ETag/immutable/font/woff2 implemented for /assets; frontend currently uses /fonts, which still lacks immutable caching |

### Every safeguard — exact current status

| Guard | Status | Evidence / unfinished work |
|---|---|---|
| Auth/localhost/origin/cookie, hidden children, deadlines, bounded screen/queues | DONE | Unit tests, process stub has no console, PE GUI subsystem=2, disconnect/goroutine and queue-resync tests |
| Blocked prompt guard + R17 status flicker | DONE | Real waiting fixture rejects 409 without answering; metadata-only poll regression passes |
| Task readiness | DONE | /api/tasks/validate phrase warnings tested; frontend owns launch integration; wording presence is not proof |
| File ownership reservations / R19 | PARTIAL | Overlap normalization tested; current API requires repo, uses owner-based release and absolute paths. Task/id/release route/409 conflict shape and persisted reload NOT STARTED |
| Bounded waves / R21 | PARTIAL | Global capacity/RAM/disk floors and explicit queued release implemented. Frontend floor payload and ram_free_mb alias NOT STARTED |
| Helper accounts / R20 | PARTIAL | Non-secret lead field + empty/same/unknown rejection implemented; error currently 502, not requested 409. Default account normalization and profile next need audit; no credential reads |
| Research route / R22 | PARTIAL | Agent-start research rejected as unavailable. GET /api/headless/routes NOT STARTED; normal Cline coding headless still exists; never advertise it as verified research |
| Proof checklist + citation audit | DONE | /api/proof/audit tests: mentions/URLs never become verified; no pages opened. Frontend also implements its own view |
| Scope drift / R24 | PARTIAL | Separate baseline/check handlers bounded and read-only; no live git baseline tested; requested git paths/root fields missing |
| Uncertain delivery / R23 | PARTIAL | Latest 100 /api/deliveries records, rejected vs unknown, no replay tested. Requested dispatch_id/text_sha/key route and HTTP 504 NOT STARTED |
| Failure recovery cards | DONE | Error kind/next_step/cause fields + frontend classification; unknown causes explicitly unknown |

### NEXT STEPS (phone-friendly order)

1. Read this block, README shutdown handoff, PERF.md, then API-REQUESTS.md R19–R24.
2. Fix the six frontend/backend mismatches above before adding features. Fix /herdr/start session
   targeting, main-account alias normalization and next-account helper protection first.
3. Match reservation JSON/release route, lead-account 409s, ram_free_mb, research routes (empty
   verified list), dispatch IDs/504/ledger, and git paths/root. Keep all refusal tests.
4. Have Opus reduce the frontend: last measurement **257,316 bytes**, over the 150 KB budget.
   Do not overwrite frontend-owned files. Add /fonts immutable cache/MIME parity.
5. Rebuild; restart only workspace APPDATA lab; run verify-lab.ps1, mutation.ps1, then soak.ps1
   for a COMPLETE 600 seconds. The improved soak was interrupted at 160 seconds by shutdown.
6. Verify parity script controls in lab, native Edge opening/close/focus, font headers and screenshots;
   frontend must finish its 60-fps/paint/input/page-memory evidence. No such claim is verified now.
7. Resolve external App Path and concurrent HISTORY/LESSONS numbering with the lead, then rerun
   check.ps1. Seven existing PowerShell suites already passed 155/155; no git work was done.

Cleanup verified: workspace lab status=not_running; own headless Deck PID 12796 stopped;
soak runner exited (cleanup reported server_not_running because shutdown stopped lab first).
No owner panes/session were stopped. Remaining test files are bounded evidence, not running work.

---

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
