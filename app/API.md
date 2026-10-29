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
| POST /api/size `{cols,rows}` | DONE | Deck card area in cells -> Herdr tab size via the client pipe (`sizer.go`); answers `{cols,rows,attached,error}`; skipped in read-only mode. Evidence: TestSizer*, lab proof HISTORY 56 |
| POST /api/action | PARTIAL | Legacy action route implemented; core prompt/keys/rename/close tested, not every action |
| POST /api/reopen | DONE | Second instance reached existing headless instance, exit 0; visible Edge focus not verified |
| /api/pane/prompt, /keys, /text, /rename, /split, /close | DONE | Real lab computed output, offline foreground codex.exe fixture, blocked rejection 409, rename/down-split/close JSON read-back |
| /api/pane/resize, /swap, /move, /zoom, /focus | PARTIAL | Validated handlers implemented; live geometry/drag/focus not verified |
| /api/tab/create, /rename, /close, /focus; /api/workspace/rename, /close, /focus | PARTIAL | IDs validated against fresh snapshot; CLI help inspected; live UI not verified |
| POST /api/shell | PARTIAL | Handler implemented, split path verified; empty-session workspace path not separately exercised |
| POST /api/agent/start, /stop, /focus | PARTIAL | Existing helper plus busy fallback; no paid AI agents started for verification; main-account normalization and next-account guard need review |
| POST /api/herdr/start, /reload-config | DONE | Named-session Start now runs `herdr --session <name> server` and names the session in the reply; the default session keeps `HerdrLauncher.exe`. Evidence: TestHerdrStartTargetsTheNamedSession (unit-tested; lab not run). /reload-config is unchanged |
| GET /api/tasks, /api/tasks/read, /api/docs, /api/runs, /api/runs/read | DONE | Allow-list/traversal/docs tests; task list real lab. Actual run-history content not inspected |
| GET /api/docs?name=KEYS | PARTIAL | Calls existing help -List; not live-tested. GUIDE and master skill documents supported |
| GET /api/health | DONE | RAM/disks/process RSS/handles/goroutines measured; pane process memory implemented; `ram_free_mb` added next to `ram_available_mb`, all older fields kept. Evidence: TestHealthAddsRamFreeAlias (unit-tested; lab not run) |
| POST /api/panes/close-finished | PARTIAL | Validates idle/done before any close; no live multi-close test |
| GET /api/signin, POST /api/signin/open | PARTIAL | Three read-only probes; unsupported tools honestly unknown; per-account probes and login panes not tested |
| GET /api/git | DONE | Read-only status/stat plus R24 `root` and `paths` from `status --porcelain=v1 -z`; the handler now allows 15 s for its four git calls. Evidence: TestGitEndpointReportsRootAndPorcelainPaths, TestParsePorcelainZReadsRenamesAndSpaces (unit-tested; lab not run) |
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
| /assets/* and /fonts/* cache, WOFF2 MIME | DONE | One `staticHandler` serves both prefixes with ETag, `private, max-age=31536000, immutable` and `Content-Type: font/woff2`; the page itself stays no-store. Evidence: TestStaticHandlerCachesFontsAndAssets (unit-tested; lab not run) |

### Every safeguard — exact current status

| Guard | Status | Evidence / unfinished work |
|---|---|---|
| Auth/localhost/origin/cookie, hidden children, deadlines, bounded screen/queues | DONE | Unit tests, process stub has no console, PE GUI subsystem=2, disconnect/goroutine and queue-resync tests |
| Blocked prompt guard + R17 status flicker | DONE | Real waiting fixture rejects 409 without answering; metadata-only poll regression passes |
| Typed-input guard (`input_not_empty`) | DONE | A prompt/paste/ask/fanout that would land in an agent's input box while the owner has text there is refused with 409 `input_not_empty` (`agent`, `pane_id`, `typed`, `message`) and no write happens first; dim suggestion text and history prompts are not "typed". Evidence: TestInputTextMatchesEveryHerdrFixture, TestPromptAndPasteRefuseTextInTheInputBox, TestEmptyInputBoxStillAcceptsAPromptAndPaste, TestAskRefusesTypedInputBeforeAnyWrite, TestFanoutChecksEveryNamedAgentFirst (unit-tested; lab not run) |
| Task readiness | DONE | /api/tasks/validate phrase warnings tested; frontend owns launch integration; wording presence is not proof |
| File ownership reservations / R19 | DONE | Frontend shape implemented: `GET /api/reservations` -> `{reservations,load_error}`, `POST {task,owner,paths}` -> `{ok,id,reservations}`, `POST /api/reservations/release {id}` -> `{ok}` or 404. Paths normalized to toolkit-relative forward-slash lower case, one trailing `/**` allowed; conflicts -> 409 with a `conflicts` list, 200-record limit, atomic file at `%APPDATA%\herdr\deck-reservations.json` reloaded on start (invalid file -> `load_error`). Evidence: TestReservationLifecycleConflictsAndReload, TestNormalizeReservationPathNormalizesAndRefuses (unit-tested; lab not run) |
| Bounded waves / R21 | DONE | `ram_free_mb` alias plus an optional start floor `{"floor":{"ram_free_mb":N,"disk_free_gb":{"C:":N}}}` -> 409 `held: below the requested floor`; the capacity/RAM/disk policy refusals are now 409 status errors. Evidence: TestLaunchFloorRefusesBelowTheRequestedFloor, TestHealthAddsRamFreeAlias (unit-tested; lab not run) |
| Helper accounts / R20 | DONE | `lead {tool,account,known}` in state; a Claude start is refused with 409 + `lead_account` unless the lead is known and the account is a different one; `""`/`main`/`default` resolve to `default_account` and start without an Account param; profile `next` starts claude on the least-used non-lead account below 95% (usage table parsed, 409 when none is left or the lead is unknown). No credential file is read. Evidence: TestClaudeAccountGuardRefusesTheLeadAndTheMainAlias, TestAccountAliasesMapOntoTheDefaultAccount, TestPickNextAccountSkipsTheLeadAndExhaustedAccounts (unit-tested; lab not run) |
| Research route / R22 | DONE | `GET /api/headless/routes` -> `{"routes":[],"note":"...unavailable..."}` (fixed empty list, never claims research works); `POST /api/headless {"purpose":"research"}` -> 409. Normal Cline coding headless is unchanged and is still not advertised as verified research. Evidence: TestHeadlessRoutesStayEmptyAndResearchIsRefused (unit-tested; lab not run) |
| Proof checklist + citation audit | DONE | /api/proof/audit tests: mentions/URLs never become verified; no pages opened. Frontend also implements its own view |
| Scope drift / R24 | DONE | Separate baseline/check handlers bounded and read-only; `/api/git` adds `root` and `paths:[{path,code}]` so the frontend compares with a task's declared files without parsing text. Evidence: TestGitEndpointReportsRootAndPorcelainPaths, TestParsePorcelainZReadsRenamesAndSpaces (unit-tested; lab not run) |
| Uncertain delivery / R23 | DONE | The prompt reply carries `dispatch_id`; `GET /api/dispatches?key=` returns the 20 newest `{dispatch_id,key,pane,text_sha,state,error,at}` records; a timeout after possible delivery is 504 `delivery unknown` with `dispatch_id`, a rejected submission is 502, and nothing is ever retried automatically. `/api/deliveries` still returns the same records. Evidence: TestPromptDispatchIDAndDispatchLedger, TestUncertainDeliveryIs504AndRejectedDeliveryIs502, TestUncertainDeliveryIsRecordedWithoutReplay (unit-tested; lab not run) |
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
- `GET /api/reservations` -> `{reservations,load_error}`; `POST /api/reservations {task,owner,paths:["app/web/app.js"]}`
  -> `{ok,id,reservations}` or 409 `reservation conflict` with a `conflicts` list; `POST /api/reservations/release {id}`.
  Paths are toolkit-relative, lower-case, forward slashes, one trailing `/**` allowed; a reservation of the
  same task replaces that task's earlier record. Shared dispatch bookkeeping, no filesystem lock.
  Round 3 below documents the exact shapes and statuses.
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
- `GET /api/deliveries` retains the latest 100 prompt dispatches; `GET /api/dispatches?key=` returns the
  20 newest with `dispatch_id,key,pane,text_sha,state`. Each guarded prompt creates a dispatch ID; a
  timeout after possible delivery is 504 `delivery unknown`, a rejected submission is 502. No automatic replay.
- Errors add `kind,next_step,cause` for recovery cards. Unknown causes remain unknown.

These endpoints are implemented and being verified; frontend integration is owned by Opus.
The safe-mode toggle blocks all launch/reservation/file writes. Proof audit and scope checks
are read-only. Retention: 4 jobs active, 24 recent, 32 scope baselines, 64 reservations,
128 paths per reservation, 512 changed files per scope comparison.

Static assets `/assets/*` and the page fonts `/fonts/*` have ETag and private immutable one-year cache
headers; WOFF2 has `Content-Type: font/woff2`. They still require the cookie/token. HTML/API are no-store.
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

## Round 3 (2026-10-26)

The frontend's R19-R24 plus the input-box guard, implemented exactly as the frontend already calls it.
Everything below is unit-tested; **the lab was not run**, so no row claims live verification. Build:
`app\build.ps1` exit 0, `HerdrDeck.exe` 8,656,384 bytes (8.26 MiB), SHA256
`70C0243EDC39F1970B8D5F2D6F8C7D1E2B6263F23A44FFAD7FC6C561EBF4CFDF`.

### HTTP errors with a status and extra fields
A handler may return `*statusError{Status, Msg, Fields}`. `failHTTP` then uses `Status` and merges every
`Fields` key into the error body after `error`, `kind`, `next_step`, `cause`; a `Fields["error"]` value
overrides `error`. Without a `*statusError`, `lockedWrite` and `/api/action` keep the old rule
(`blocked:` -> 409, everything else -> 502). Test: TestStatusErrorCarriesItsOwnStatusAndFields.

### Input-box guard (`input_not_empty`)
`inputText(screen, tool) string` reads the agent's input box out of real Herdr ANSI output:
- Claude: the last line containing the prompt glyph `U+276F` that has a rule line above it (a line of
  only `U+2500`, at least 20 runes, ANSI stripped) and another rule line within 12 lines below it. The
  region is that line after the glyph and its following space or NBSP (`U+00A0`), plus every line up to
  the lower rule. A history prompt has no rule above it and is not a box.
- Codex: the last line in the last ten lines that starts (after spaces) with `U+203A`; the region is the
  rest of that line.
Inside the region a visible rune counts as typed unless it is a space/NBSP or the current SGR style is
dim. Dim = SGR 2, SGR 90, `38;5;n` with n == 8 or 232 <= n <= 250, or `38;2;r;g;b` with max-min <= 12 and
max <= 170; `0`/`22`/`39` reset it. The result is ANSI-stripped, trimmed and at most 200 runes.

Wired in for `action` "prompt" and "paste" on panes with an agent, for `/api/agents/ask` and for
`/api/agents/fanout` (every named agent is checked first, so one typed box stops the whole fanout). The
pane is read with `pane read <id> --source visible --lines 60 --format ansi`, and no write happens before
the check. Refusal:

    HTTP 409 {"error":"input_not_empty","kind":"agent_or_request_error","next_step":"...","cause":"...",
              "agent":"<name or pane id>","pane_id":"w1:p1","typed":"<first 60 runes>",
              "message":"<agent> has text in its input box - not sent"}

Tests: TestInputTextMatchesEveryHerdrFixture (all nine fixtures under `app/tests/fixtures/input`),
TestTypedTextIgnoresDimTextAndKeepsSpaces, TestPromptAndPasteRefuseTextInTheInputBox,
TestEmptyInputBoxStillAcceptsAPromptAndPaste, TestAskRefusesTypedInputBeforeAnyWrite,
TestFanoutChecksEveryNamedAgentFirst.

### R19 - reservations
- `GET /api/reservations` -> `{"reservations":[{"id":"res-<unixnano>","task":"tasks/a.md","owner":"codex-1","paths":["app/web/app.js"],"created":"..."}],"load_error":""}`.
- `POST /api/reservations {"task","owner","paths":[...]}` -> 200 `{"ok":true,"id":...,"reservations":[...]}`.
  - 409 `{"error":"reservation conflict","conflicts":[{"path":"app/web/app.js","task":"tasks/a.md","owner":"codex-1"}]}`
    when another task already owns an overlapping path (the requested, normalized path is listed).
  - 400 for a missing/too long task (1-200 bytes) or owner (1-120 bytes), an empty path list or more than
    128 paths, a NUL, a `*`/`?` other than one trailing `/**`, or a path that leaves the toolkit root.
  - 409 `reservation limit reached` above 200 records. The same task replaces its own earlier record.
- `POST /api/reservations/release {"id"}` -> `{"ok":true}`, or 404 `reservation not found`.

Normalization: trimmed, `\` -> `/`, leading `./` dropped, absolute paths made relative to the toolkit
root, one trailing `/**` and any trailing `/` stripped, lower-cased (`app\WEB\App.js` ->
`app/web/app.js`). Overlap: equal, or one path is the other plus `/`. The list is written atomically to
`%APPDATA%\herdr\deck-reservations.json` and loaded once per process; a missing file means an empty list
and an unreadable or invalid file keeps the list empty and reports the reason in `load_error` (never
silently). This is dispatch bookkeeping only and never blocks file access.
Tests: TestReservationLifecycleConflictsAndReload, TestNormalizeReservationPathNormalizesAndRefuses.

### R20 - account guard
State carries `lead:{"tool":"claude","account":"2030"|"","known":true|false}` (the non-secret account
folder name of the Claude session that started the Deck).
- A Claude start (`/api/agent/start`, `action` "start") is refused with 409
  `{"error":"account guard: pick another account (the lead uses 2030)","lead_account":"2030"}` when the
  lead is unknown, the account is empty, or the account equals the lead's.
- `""`, `main` and `default` (any case) mean the profile's `default_account`. When a start goes ahead with
  that account, the helper is called **without** an `Account` parameter (the main sign-in has no account
  folder); any other account is passed through lower-cased.
- Helper launches (`helper:true`) keep their rules (explicit account and lead account, different, one of
  the owner's other accounts 1984/2040) and now also answer 409.
- Profile `next` (Claude on the least-used account): `herdr-agents usage` is parsed with
  `^(\S+?)\*?\s+(\d+)%\s+.+?\s(\d+)%\s`; the lead's row and every row at or above 95% (5-hour or weekly)
  are skipped; lowest 5-hour, then lowest weekly, then name order wins. The launch then starts profile
  `claude` with that account. 409 `no Claude account available besides the lead` when nothing is left,
  and 409 when the lead account is unknown.
No credential file is ever read.
Tests: TestClaudeAccountGuardRefusesTheLeadAndTheMainAlias, TestAccountAliasesMapOntoTheDefaultAccount,
TestPickNextAccountSkipsTheLeadAndExhaustedAccounts.

### R21 - RAM field and start floors
`GET /api/health` keeps every field and adds `ram_free_mb` with the same number as `ram_available_mb`.
`POST /api/agent/start` accepts an optional
`{"floor":{"ram_free_mb":N,"disk_free_gb":{"C:":N}}}`; when free RAM or a requested drive is below the
floor the start is refused with 409
`{"error":"held: below the requested floor","ram_free_mb":<now>,"disk_free_gb":{"C:":<now>}}`. A drive
the health report does not list counts as below the floor (never guessed). The capacity, RAM and disk
policy refusals (`held: ...`) are 409 status errors too; a health failure stays 502.
Tests: TestLaunchFloorRefusesBelowTheRequestedFloor, TestHealthAddsRamFreeAlias.

### R22 - research routes
`GET /api/headless/routes` ->
`{"routes":[],"note":"No server-search-only research route has been verified for the Deck; research stays unavailable."}`
(a fixed empty list; the Deck never claims research works). `POST /api/headless` accepts an optional
`"purpose"`; `"research"` -> 409 `research route unavailable: no verified server-search route`. Without
`purpose` the existing Cline coding behaviour is unchanged.
Test: TestHeadlessRoutesStayEmptyAndResearchIsRefused.

### R23 - dispatch IDs and uncertain delivery
- `POST /api/pane/prompt` (and the prompt action) -> `{"ok":true,"message":...,"output":...,"dispatch_id":"dispatch-<unixnano>"}`.
- `GET /api/dispatches?key=<key>` -> `{"dispatches":[{...}]}`, newest first, at most 20, filtered by `key`
  when given; `key` is `session|pane_id`. Each record is
  `{"id","dispatch_id","key","pane","text_sha","state","error","at"}` with `state` `sent` | `unknown` |
  `failed` and `text_sha` the hex SHA-256 of the prompt text.
- A transport failure or timeout after a possible delivery -> 504
  `{"error":"delivery unknown","dispatch_id":...,"detail":...}`. A submission rejected with
  `exit status 2`, `agent_not_ready`, `agent_blocked` or `invalid_request` -> 502 with `dispatch_id`.
- Nothing is ever retried automatically. `/api/deliveries` still returns the same records (latest 100).
Tests: TestPromptDispatchIDAndDispatchLedger, TestUncertainDeliveryIs504AndRejectedDeliveryIs502,
TestUncertainDeliveryIsRecordedWithoutReplay.

### R25 - prompt intent: agent text never runs in a shell
`POST /api/pane/prompt` takes `"expect": "agent" | "shell"` (missing = `agent`). When the fresh snapshot
shows no agent in the pane (it exited; only its shell is left) and `expect` is not `shell` -> 409
`{"error":"agent_exited","pane_id":...,"message":"The agent in <pane> has exited; only a shell is left - not sent"}`,
nothing written. `expect: "shell"` runs the text with `pane run` (a card that never had an agent).
The page sends `shell` only from a card that never showed an agent; a card whose agent exited keeps
sending `agent` and its box says "Agent exited · messages are not sent to the shell".
Test: TestPromptToExitedAgentNeverRunsInTheShell.

### R24 - git root and paths
`GET /api/git?key=` keeps `status`, `stat` and `text` and adds:
- `root`: `git -C <cwd> rev-parse --show-toplevel` with forward slashes.
- `paths`: `[{"path":"app/web/app.js","code":" M"}]` parsed from
  `git --no-optional-locks -C <cwd> status --porcelain=v1 -z --untracked-files=all`; `code` is the first
  two bytes, `path` is repo-relative with forward slashes. For a rename or copy entry the first NUL field
  is the new name and the following field (the original name) is skipped.
The handler now allows 15 s for its four git calls (starting `git.exe` takes about 2 s on this PC, which
made the old 5 s budget time out).
Tests: TestGitEndpointReportsRootAndPorcelainPaths, TestParsePorcelainZReadsRenamesAndSpaces.

### /fonts cache
The `/assets/*` cache block was extracted into `staticHandler(files fs.FS) http.Handler` and now covers
`/fonts/*` as well: ETag over the file bytes, `Cache-Control: private, max-age=31536000, immutable`,
`Content-Type: font/woff2` for `.woff2`, 304 on a matching `If-None-Match`, 404 for a missing file. The
page itself stays no-store. Test: TestStaticHandlerCachesFontsAndAssets.

### /api/herdr/start on a named session
With `--session <name>` the Deck now starts `herdr --session <name> server` through the same hidden,
detached process start the launcher uses, and answers
`Herdr server for session <name> started`. Without `--session` it starts `launcher\HerdrLauncher.exe`
exactly as before. Test: TestHerdrStartTargetsTheNamedSession.

### build.ps1 finds Go
Order: `$env:DECK_GO` (when set and present), `C:\Users\mflma\tools\go\go1.26.1\go\bin\go.exe`,
`D:\tools\go\go1.26.10\go\bin\go.exe`, then `go.exe` on PATH. Nothing found fails with
`Go not found: set DECK_GO to go.exe (tried: ...)`. The script prints `Go: <path>` once, then formats,
vets, tests and builds as before.

## Round 4 — R26–R35 (supersedes conflicting earlier descriptions)

Backend unit tests use fake Herdr responses; live integration remains the lead's check.

### R26 — C1: per-client visibility, recovery and retained screens

`GET /api/events` sends `hello {"client_id":"c-<n>"}`, then
`settings {"read_only":false|true}`, then `state`, in that order.
`POST /api/visible {"client_id":"c-3","keys":["session|w1:p1"]}` replaces only that
client's interest. Missing `client_id` means `legacy`. Disconnect removes the client's
set. Screen reads use the union; each stream receives screen content/patches only for
its own keys. Metadata/status still reach all clients. Updating visibility supplies a
filtered state snapshot so re-shown panes have the right screen version.

While any SSE client exists, metadata/recovery remains due every 2 seconds, including
when all sets are empty. Offline snapshots retain panes with `stale:true`, set
`herdr.running:false` and report `error`. A successful snapshot clears stale state.
The current screen recomputes the text-derived blocked flag; negative screen evidence
clears it while authoritative blocked metadata remains respected (D8).

Screens retain the last **2000** rows before diffing (D10). Patch coordinates and versions
are checked against the evolving retained array; invalid bounds use a full screen update.
One diff pass uses pre-split rows, prefix/suffix fast paths and a bounded shift search;
repeated-row tail edits remain a single patch (P3).

Polling uses a round-robin cursor, at most four screen reads per pass, 350 ms per screen,
a 750 ms metadata budget, and exponential failed-read backoff (2–32 s). Failed reads
preserve cached screen text. Snapshot agents replace the redundant agent-list RPC.
Background socket requests defer to active/queued requests; queue waits and in-flight
I/O respond to cancellation. Health sampling is shared/cached for 10 s, with one sampler
and a 1.5 s total pane-memory budget, 350 ms per pane (P2).

Tests: `TestR4VisibilityIndependent`, `TestR4ClientScreensAndDisconnect`,
`TestR4HiddenClientMetadataHeartbeat`, `TestR4OfflineRetainsStalePanes`,
`TestR4BlockedClearsOnFreshScreen`, `TestR4RetainedRowsAndPatchBounds`,
`TestR4DiffTailUsesSinglePatch`, `TestR4PollFairnessAndBackoff`,
`TestR4HealthSingleSample`, `TestR4SocketCancelledBeforeAdmission`,
`TestR4FailedScreenRetainsLastGoodRows`.

### R27 — C2: settings events

Every read-only change broadcasts `event: settings` with `{"read_only":true|false}`
immediately. Overflow resynchronization includes current settings and filtered state.
Test: `TestR4SettingsBroadcast` (initial ordering also covered by R26 stream tests).

### R28 — C3: ordered raw typing

`POST /api/pane/raw {"key":"session|w1:p1","text":"abc","keys":["enter"]}` writes
text with send-text, then each key with send-keys, in order; returns `{"ok":true}`.
Either field may be empty, but not both. Key aliases/allow-list match `/api/pane/keys`;
the whole key list is validated before writing. Per-pane FIFO tickets serialize incoming
requests; the browser remains responsible for preserving order across network transit.
The input-box guard is bypassed for this deliberate owner input, including blocked panes.
Read-only mode returns 403. Test: `TestR4RawOrderedAndReadOnly`.

### R29 — C4: reservation ownership and unavailable ledgers

`POST /api/reservations` keeps `task`, `owner`, `paths` and accepts `overlap:true`.
Same owner plus same task returns the existing id without replacing its record. A different
owner never replaces/drops a record. Conflicts remain 409; explicit overlap appends a
record with `shared:true`. Unreadable/invalid ledgers keep GET `load_error`; mutation returns
503 `{"error":"ledger_unavailable","message":"..."}` without overwriting the ledger.
Reserve before dispatch; release the new id after a definite dispatch failure, retaining
it when delivery is unknown. Tests: `TestR4ReservationOwnershipAndOverlap`,
`TestR4UnreadableReservationLedger`.

### R30 — C5: one launch admission and confirmed identity

Direct/action starts, queue release, workspace-open, layouts, columns, team, headless and
trial script adapters pass one admission service. Plans expand before execution; shell-only
columns reserve zero agent slots. Each agent passes account/resource checks; live plus
pending agents cannot exceed policy. Pending slots are reserved atomically, released on
failure or after 120 s, and consumed by new agents observed in a successful snapshot.
Capacity refusal is 409 `{"error":"capacity","message":"..."}`; existing account/floor
refusals retain their codes.

`/api/agent/start` adds `pane_id` and `name` only when a helper line contains
`STARTED name=<n> ... pane=<id>`. Otherwise both fields are absent; no identity is guessed.
This is the backend D12 fix. D17 recovery uses existing `GET /api/jobs` records keyed by
the returned `job_id`/`id`; clients reconcile those records after reconnect/response races.
If SSE overflow would discard a result, only that stream closes to trigger reconciliation;
completed job records remain available. Test: `TestR4ResultOverflowReconnectsForLedger`.

Tests: `TestR4LaunchPendingAndIdentity`, `TestR4EveryLaunchRouteUsesAdmission`,
`TestR4LaunchFailureReleasesPending`, `TestR4TeamReservesExpandedPlan`,
`TestR4LayoutChecksEveryAccount`, `TestR4SnapshotConfirmsPendingLaunch`,
`TestR4PendingLaunchExpiresAt120Seconds` (virtual clock, no two-minute sleep).

### R31 — C6: typed delivery failures

A prompt whose write process started but has no known acknowledgement returns 504
`{"error":"delivery unknown","dispatch_id":"..."}`. Read timeouts return 504
`{"error":"read_timeout","message":"..."}` without a dispatch id. Cancelled-before-start
or process-start failures return 502 `{"error":"not_sent","message":"..."}` without one.
Explicit server rejection remains 502. There is no automatic replay.
Tests: `TestR4ReadTimeoutIsNotDelivery`, `TestR4CancelledWriteNeverStarts`,
`TestR4MissingExecutableNotSent`, existing `TestUncertainDeliveryIs504AndRejectedDeliveryIs502`.

### R32 — C7: recognized, empty and occupied input regions

For Claude/Codex, prompt/paste/ask/fanout with an unrecognized input box returns 409
`{"error":"input_unknown","message":"<name>: cannot see its input box - not sent"}`.
Recognized empty boxes permit writes; occupied boxes retain `input_not_empty`. Other
agent kinds retain their earlier behavior. Claude reads the entire bounded box, including
boxes taller than twelve lines. Codex reads continuation/blank rows through the footer.
SGR state preceding the marker is preserved; existing dim-placeholder rules remain.
Tests: `TestR4UnknownInputAndMultiline`, `TestR4CodexTallDraftWithBlankLines`,
`TestR4InheritedInputStyle`, existing input fixture/send-guard tests.

### R33 — C8: durable preferences

`GET /api/prefs` returns `{"prefs":{...}}`; missing files mean `{}`. POST
`{"prefs":{...}}` atomically replaces `%APPDATA%\herdr\deck-prefs.json` and returns
`{"ok":true}`. JSON bodies over 64 KiB return 413. Read/write errors are surfaced;
preferences are allowed while read-only mode is enabled. Test: `TestR4PrefsPersistAndLimit`.

### R34 — C9: launch/worktree volumes and unknown disk values

`/api/health` includes D:, pane folders and admitted launch/worktree volumes. Unknown
`free_gb` and pane-memory `mb` values are explicit `null`, never fabricated zeroes.
Launch policy checks the launch folder's volume; worktrees use the user-profile worktree
volume. Unknown space with a nonzero floor returns 409
`{"error":"disk_unknown","drive":"...","message":"..."}`.
Tests: `TestR4HealthIncludesDAndNullUnknown`, `TestR4HealthIncludesPaneVolume`,
`TestR4UnknownLaunchVolumeRefused`, `TestR4WorktreeVolumeFloor`.

### R35 — C10: inexpensive debug counters

Token-protected `GET /api/debug/counters` is always available and returns
`goroutines`, `handles`, `rss_mb`, `sse_clients`, `sse_events`, `sse_bytes`, `polls`,
`poll_errors`, `poll_ms_last`, `poll_ms_max`, `rpc_waiting`, `resyncs`, `jobs_live`.
SSE counters measure successful event writes; poll durations use milliseconds; unavailable
RSS is `null`. Test: `TestR4Counters`.

