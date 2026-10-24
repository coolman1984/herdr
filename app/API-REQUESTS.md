# API requests from the FRONTEND (Herdr Deck)

Written by the frontend agent. One request per block: endpoint, JSON in/out, why.
The frontend calls everything through `web/api.js` (route table at its top), so the backend may
choose other paths - write the real ones in `app/API.md` and the frontend adapts.
Status per request: OPEN until `API.md` answers it.

## R1 - Token for static assets (OPEN)
The page loads ES modules (`app.js` imports `./api.js`, `./ansi.js` ...). Relative module URLs lose
the `?t=` query. Please either set a cookie on `GET /?t=<token>` (`HttpOnly; SameSite=Strict;
Path=/`) and accept it for assets/API/SSE, or accept asset requests that carry the cookie only.
The frontend also sends `?t=<token>` on every `/api/...` and SSE URL, and header `X-Deck-Token`.
Why: no inline scripts (CSP), no bundling.

## R2 - State (OPEN)
`GET /api/state` -> the `State` from `model.go` plus:
```json
{"session":"lab","herdr":{"running":true,"error":""},
 "panes":[{"key":"lab|w1:p1","pane_id":"w1:p1","workspace_id":"w1","tab_id":"w1:t1","cwd":"I:\\herdr",
   "label":"","name":"codex-1","agent":"codex","title":"","agent_status":"working",
   "tokens":{"model":"gpt-6.1-sol","context":"8%","total":"23k"},"blocked":false,
   "screen":"...ansi text...","revision":12,"error":""}],
 "workspaces":[{"workspace_id":"w1","label":"herdr","agent_status":"idle","focused":true}],
 "tabs":[{"tab_id":"w1:t1","workspace_id":"w1","label":"1","agent_status":"idle","focused":true}],
 "profiles":[{"name":"codex","kind":"codex","model":"gpt-6.1-sol","effort":"medium","accounts":["2030"]}],
 "error":""}
```
`blocked` = backend's `blockedScreen()` result (the frontend shows it; never answers).
Please keep `workspace_id` on tabs as the parent workspace (today `parseSnapshot` overwrites
`Tabs[i].Workspace` with `Tabs[i].ID`, which is the tab id key - check the JSON names).

## R3 - Event stream (OPEN)
`GET /api/events?t=...` (SSE). Events the frontend understands:
- `state` - full State (on connect and after structure changes);
- `panes` - `{"panes":[<changed pane objects, any subset of fields + key>],"removed":["lab|w1:p3"]}`;
- `herdr` - `{"running":false,"error":"..."}`;
- `error` - `{"message":"..."}` (shown as a toast);
- `result` - `{"id":"...","ok":true,"message":"..."}` for slow actions (optional).

## R4 - Visible panes (OPEN)
`POST /api/visible {"keys":["lab|w1:p1"]}` -> `{"ok":true}`. The frontend calls it whenever the
grid filter/zoom changes and when the window hides (`keys:[]`), so the backend polls visible panes at
~700 ms and the rest at ~2 s (spec section 4). When hidden, the frontend also closes the SSE stream.

## R5 - Pane actions (OPEN)
All `POST`, JSON in, `{"ok":true,"message":"..."}` or HTTP 4xx/5xx `{"error":"..."}` out:
- `/api/pane/prompt {"key","text"}` - agent prompt (agents) or `pane run` (shells);
- `/api/pane/keys {"key","keys":["Escape"]}` - names: `Escape`,`C-c`,`Enter`,`Up`,`Down`,`Tab`,`y`,`n`;
- `/api/pane/text {"key","text"}` - `pane send-text` exactly, no Enter (Paste to pane);
- `/api/pane/rename {"key","label"}`; `/api/pane/close {"key"}`;
- `/api/pane/split {"key","cwd"}` and `/api/shell {"cwd"}` (+ Shell);
- `/api/agent/start {"profile","model","effort","account","key","cwd"}`;
- `/api/agent/stop {"name"}`; `/api/agent/focus {"name"}`; `/api/tab/focus {"tab_id"}`;
- `/api/herdr/start {}`; `/api/herdr/reload-config {}`.

## R6 - P1 data (OPEN, in this order)
1. Copy/paste: covered by R5 `/api/pane/text`.
2. Inbox: frontend-only (uses `agent_status` + `blocked`).
3. Tasks: `GET /api/tasks` -> `{"tasks":[{"file":"tasks/x.md","title":"...","mtime":"..."}]}`,
   `GET /api/tasks/read?file=tasks/x.md` -> `{"file","text"}` (allow-list `tasks/*.md` only).
4. Broadcast: frontend-only (many `/api/pane/prompt`).
5. Usage: `GET /api/usage` -> `{"text":"<herdr-agents usage output>","accounts":[{"name":"2040",
   "five_hour":41,"weekly":12}],"codex":{...}}` (text alone is fine; frontend shows it).
6. Health: `GET /api/health` -> `{"ram_used_pct":61,"disks":[{"drive":"C:","free_gb":12.3}],"panes":4}`;
   `POST /api/panes/close-finished {"keys":[...]}` (frontend lists them and confirms first).
7. Team presets: frontend-only (sequence of `/api/agent/start`).
8. Sign-in: `GET /api/signin` -> `{"tools":[{"tool":"claude","account":"2040","signed_in":true,
   "detail":"..."}]}`; `POST /api/signin/open {"tool","account"}` opens a login pane (the owner signs in).

## R7 - P2 data (OPEN, nice)
`GET /api/git?key=` (status + diff --stat of the pane cwd), `GET /api/runs` + `/api/runs/read?file=`,
`GET /api/docs?name=HISTORY|LESSONS|AGENT-ROSTER|TEAM-PLAYBOOK`, `POST /api/lessons/append {"text"}`,
`POST /api/check` -> `{"lines":["OK ...","WARN ..."]}`.

---
# Parity addendum requests (tasks/herdr-deck-parity.md) - 2026-10-24, priority order

Seen in API.md v2: R1-R7 accepted, thanks. Frontend now: keeps SSE open while hidden and sends
`/api/visible {keys:[]}`; listens to `state`, `panes`, `diff`, `herdr`; uses `/api/read-only`.

## R8 - Layout + HUD data in State (OPEN, P0 parity A1-A3)
Add to State: `layouts` exactly as `api snapshot` returns them (per tab: `area`, `panes[{pane_id,rect}]`,
`splits`, `zoomed`, `focused_pane_id`), `focused_pane_id`, and per pane `revision`, `focused`.
Keep every Herdr metadata token the HUD writes (`name, path, model, ctx, in, out, cost, limit, account`)
and pane `title`; workspaces: keep `branch`, `git_status` tokens if Herdr has them.
Why: the Deck draws the real split tree (rects) and the HUD values exactly as today.

## R9 - Herdr pane/tab/workspace verbs (OPEN, A1 + A15)
POST, `{ok,message}` out, all lockedWrite:
- `/api/pane/split {"key","direction":"right"|"down","cwd"}` (direction new; default right);
- `/api/pane/resize {"key","direction":"left|right|up|down","amount":N}` (`herdr pane resize`);
- `/api/pane/swap {"key","with":"<key>"}` (`herdr pane swap`); `/api/pane/move {"key","tab_id"}`;
- `/api/pane/zoom {"key"}` (Herdr's own zoom, separate from the Deck's local zoom);
- `/api/pane/focus {"key"}`; `/api/tab/create {"workspace_id","cwd"}`, `/api/tab/rename {"tab_id","label"}`,
  `/api/tab/close {"tab_id"}` (UI confirms); `/api/workspace/rename {"workspace_id","label"}`,
  `/api/workspace/close {"workspace_id"}` (UI confirms), `/api/workspace/focus {"workspace_id"}`.
- `/api/pane/keys`: please also allow `space`, `shift+tab`, `ctrl+<a-z>`, `f1`-`f12` (raw typing mode).

## R10 - Folder picker (OPEN, A5-A7)
- `GET /api/folders/recent` -> `{"folders":["I:\herdr", ...],"last":"I:\herdr"}` (from
  `%APPDATA%\herdr\open-folder-last.txt` + folders of current panes).
- `GET /api/folders/list?path=I:\` -> `{"path","parent","dirs":["a","b"]}` (directory names only,
  no file contents; drives list when path empty).
- `POST /api/workspace/open {"cwd","agent":""|"claude"|"team"}` -> same as `herdr-open-folder.ps1`
  without its dialog (new workspace there; `claude` starts Claude; `team` = Claude left + Codex right).

## R11 - Columns + layout builder (OPEN, A8-A9)
- `POST /api/columns {"key","count":N,"direction":"right"|"down"}` = `herdr-columns.ps1 -Count N`.
- `GET /api/layouts` -> `{"presets":[{"name","plan":{...}}],"last":{...}}` from `%APPDATA%\herdr\layouts\`
  + `layout-last.json`; `POST /api/layouts/build {"plan":{...}}` writes a temp plan and runs
  `herdr-layout.ps1 -PlanFile`; `POST /api/layouts/save {"name","plan"}`. Plan shape = whatever
  herdr-layout.ps1 reads; please document it in API.md (frontend builds that JSON).

## R12 - Spaces side by side (OPEN, A11)
`POST /api/spaces/join {"workspace_ids":[...]}` and `POST /api/spaces/separate {}` = `herdr-spaces.ps1`
non-interactive; `GET /api/spaces` -> current `side-by-side.json` (or `{}`).

## R13 - Stop everything + next-account Claude (OPEN, A13-A14)
- `POST /api/stop-all {"confirm":"stop"}` = `herdr-stop.ps1` without its dialog; frontend shows a typed
  confirmation first and sends the typed word; backend rejects anything else.
- `POST /api/agent/start {"profile":"next", ...}` = `herdr-agents start next` (allow "next").

## R14 - herdr-agents commands (OPEN, A16)
Read-only GET, `{output}` text (ANSI stripped) is enough; JSON where the tool has it:
`/api/agents/tools`, `/api/agents/status`, `/api/agents/read?name=`, `/api/agents/diff?name=`,
`/api/agents/history`, `/api/agents/collect`.
POST (lockedWrite): `/api/agents/ask {"name","text","fresh":bool}` (returns at once with a job id;
answer later as SSE `result {id, ok, message}`), `/api/agents/fanout {"names":[...],"text"}`,
`/api/agents/team {"preset","cwd"}`, `/api/agents/handoff {"text"}` (writes the handoff file from
`tasks/HANDOFF-TEMPLATE.md`; say where), `/api/agent/start` + `"worktree":true`.

## R15 - Other tools (OPEN, A17)
`POST /api/headless {"profile","text"}` -> job id; SSE `result {id, ok, message:"DONE|FAILED ...", output, tokens}`.
`POST /api/trial {...}` (herdr-trial.ps1; document its args), `POST /api/sync-skills {}`.
`GET /api/docs?name=GUIDE` (docs/GUIDE.md) and `?name=KEYS` (`herdr-help.ps1 -List` text) for the
searchable help (A12). Skills: `GET /api/docs?name=herdr-team|herdr-workspace` (skills/*/SKILL.md).

## R16 - Health per pane memory (OPEN, B resource-aware)
`/api/health` add `panes_mem:[{"key","mb"}]` (process-info working set per pane) so the card warns
when a pane eats memory.

## R17 - BUG: blocked status flaps in SSE (OPEN, P0) - 2026-10-24 18:15
Lab session, shell pane w1:p1 showing a static trust prompt ("Do you trust this folder?" /
"enter to confirm  esc to cancel", waiting in Read-Host). `/api/state` says `blocked:true,
agent_status:"blocked"`, but the card state seen through the event stream goes
blocked (101 ms) -> unknown (1612) -> blocked (3021) -> unknown (3827) -> blocked (4633) while the
screen does not change. Likely the `panes` diff objects carry `blocked:false` / `agent_status:"unknown"`
when the poll that produced them did not re-evaluate the screen (patch-only polls?). Please send
`blocked` and `agent_status` only when they really changed, computed from the full current screen.
Frontend mitigation meanwhile: one toast/notification/inbox entry per pane per 30 s.

## R18 - small (OPEN)
- `/api/health` disks: `free_gb` arrives with 13 decimals (`18.0252685546875`); the frontend rounds,
  fine either way.
- `GET /api/jobs` noted; the frontend reads it when the Agents hub opens.

---
# BUILD NOW features (tasks/deck-feature-research.md rows 1-10) - 2026-10-24, priority order
Frontend does rows 1, 6, 9, 10 alone (task text / screen text / existing errors). It uses existing
`/api/git`, `/api/health`, `/api/tasks/read`, `/api/runs/read`. Needed from the backend:

## R19 - File ownership reservations (row 2)
`GET /api/reservations` -> `{"reservations":[{"id","task","owner","paths":["app/web/app.js"],"created"}]}`;
`POST /api/reservations {"task","owner","paths":[...]}` -> `{ok,id}` or 409 `{error, conflicts:[{path,task,owner}]}`;
`POST /api/reservations/release {"id"}`. Paths normalised (lower-case, forward slashes, relative to the
toolkit root). Stored under `%APPDATA%\herdr\deck-reservations.json`, bounded (<= 200), atomic writes.
Dispatch check only - never blocks file access.

## R20 - Helper account guard (row 4)
State adds `"lead":{"tool":"claude","account":"2040"|"" ,"known":true|false}` = the non-secret account
NAME of the Claude session that started the Deck (from its CLAUDE_CONFIG_DIR folder name under
`.herdr-accounts\claude\<name>`; empty = default folder; known:false when it cannot tell).
`/api/agent/start` for kind claude: reject with 409 when `account` is empty or equals the lead's
account, or when the lead is unknown ("account guard: pick another account"). Never read credentials.

## R21 - Bounded launch waves (row 3)
Frontend holds launches itself (max active agents + RAM/disk floor from `/api/health`), so only:
`/api/health` keep `ram_used_pct`, `disks[{drive,free_gb}]`, and add `ram_free_mb`. Optional server-side
guard: `/api/agent/start` with `"floor":{"ram_free_mb":N,"disk_free_gb":{"C:":N}}` -> 409 when below.

## R22 - Research route guard (row 5)
`GET /api/headless/routes` -> `{"routes":[{"profile":"cline-ds","verified":true,"search":"server",
"note":"..."}]}` - only profiles whose headless run uses the CLI's server-side search (no browser).
`/api/headless` rejects a profile not listed as verified. Frontend shows "unavailable" when empty.

## R23 - Uncertain delivery guard (row 8)
`/api/pane/prompt` returns `{ok, dispatch_id}`. When the Herdr call times out after it may have
delivered, return 504 `{error:"delivery unknown", dispatch_id}` (not 502), and never retry by itself.
`GET /api/dispatches?key=` -> last 20 `{dispatch_id, key, text_sha, state:"sent|unknown|failed", at}`.

## R24 - Scope drift baseline (row 7)
`/api/git?key=` already gives `status`. Add `"paths":[{"path","code":" M"}]` (parsed porcelain, repo-
relative, forward slashes) and `"root"` (repo top level) so the frontend compares with the task's
declared files without parsing text.
