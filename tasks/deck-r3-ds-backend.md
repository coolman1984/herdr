# Task: Herdr Deck round 3, phase 1 - backend contract fixes and the input-box guard (Go)

## 1. Role and goal
You are a senior Go engineer working for the lead (Claude Opus). You run headless; nobody can answer
questions, so this spec decides everything. Where it is silent, choose the smallest change that
matches the surrounding code.
Goal: make the Deck backend answer the frontend's open requests R19-R24 exactly as the frontend
already calls them, fix the account guard, the named-session Start and the font cache, and add the
new `input_not_empty` guard. Every change has a Go test; `app\build.ps1` passes.
Why: the frontend (`app/web/*.js`) already sends these shapes; today they fail or return the wrong
status. The input guard prevents a real incident: the lead's prompt was appended to text the owner
was typing in an agent's input box.

## 2. Context you need (nothing else is needed)
- Repo root: `D:\WORK\Software Development\GitHub\HERDR`. Read `AGENTS.md` first (delegate rules).
  Then `app/API.md` (the status table at the top) and `app/API-REQUESTS.md` blocks R19-R24.
- Go 1.26.10 portable, NOT on PATH: `D:\tools\go\go1.26.10\go\bin\go.exe`. Stdlib only, no downloads
  (`GOPROXY=off`). Build + vet + test + exe: `pwsh -NoProfile -File app\build.ps1` (it keeps caches in
  `app\tests\go-cache`). Fast loop while working:
  `$env:GOCACHE="$PWD\app\tests\go-cache"; $env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; Push-Location app; & D:\tools\go\go1.26.10\go\bin\go.exe test ./internal/deck/ -count=1; Pop-Location`
- Package `app/internal/deck` (all files `package deck`):
  - `server.go`: `App` struct, `ServeHTTP`, `failHTTP`, `respond`, `readBody` (strict JSON, unknown
    fields rejected), `refresh`/`publish` (polling), `Action` + `(a *App) action(...)` (prompt/paste/keys/
    start/shell/launch...).
  - `extensions.go`: `paneRequest`, `paneID(key)`, `lockedWrite(w, fn)` (maps errors to HTTP: prefix
    `blocked:` -> 409, else 502), the alias routes `/api/pane/prompt` etc., `/api/health`, `/api/git`.
  - `guards.go`: `Reservation`, `Delivery`, `LeadIdentity`, `leadIdentity`, `GuardState`,
    `normalizeOwnership`, `overlaps`, `deliver`, `launchGuard`, `guardEndpoint` (reservations, launch
    queue, scope baseline).
  - `tools.go`: `toolEndpoint` (`/api/agents/ask`, `/fanout`, `/team`, `/handoff`, `/api/headless`...).
  - `herdr.go`: `Herdr` interface (`Call`, `Helper`), `CLI{Bin,Session,Root}`, `runCommand`,
    `hiddenCommand`, `Profile`, `readProfiles` (`DefaultAccount` = `accountNames[tool]`, e.g. claude -> "2030").
  - `model.go`: `Pane`, `State`, `parseSnapshot`, `blockedScreen`. `ansiPattern` is in `herdr.go`.
  - `run.go`: `run()` builds `CLI` and calls `newApp(...)`; static files handler (`/assets/` cache block).
  - Tests: `server_test.go` has `fakeHerdr` (records writes in `f.writes`), `testApp(f)`, `request(a,
    method, path, body, token, origin)`. Copy that style. Token `test-token`, origin `http://127.0.0.1:9876`.
- Style to copy: short functions, `fmt.Errorf` messages in plain English, no comments except where a
  rule is not obvious, gofmt.

## 3. Exact scope
- Change: `app/build.ps1`, `app/internal/deck/*.go` (including new files `input.go`, `input_test.go`,
  `r3_test.go`), `app/API.md` (status table + one new section), `app/tests/fixtures/input/*` (read
  only - the lead made them; do not edit them).
- Create: `tasks/deck-r3-ds-backend-answer.md` (your answer, section 7).
- Do NOT touch: `app/web/**`, `app/FEATURES.md`, `app/API-REQUESTS.md`, `HISTORY.md`, `LESSONS.md`,
  `docs/**`, any `.ps1` outside `app/`, `agents.json`, `herdr-app/`.
- Do not: run git commands that write (commit, add, stash, checkout, reset); `git show HEAD:<file>`
  and `git diff` are allowed. Do not start Herdr servers, HerdrDeck.exe, agents or a browser. Do not
  install anything. Do not read `auth.json`, credentials or tokens.
- C: has ~2 GB free: keep every temp folder under `app\tests\` and delete it when done.

## 4. Design (decided by the lead - follow it)

### 4.0 HTTP errors with a status and extra JSON fields
Add to `server.go`:
```go
// statusError carries an HTTP status and extra JSON fields to the client.
type statusError struct {
	Status int
	Fields map[string]any // merged into the error body; "error" is set from Msg
	Msg    string
}
func (e *statusError) Error() string { return e.Msg }
```
Change `failHTTP(w, status, err)`: if `errors.As(err, &se)` then use `se.Status` and add every
`se.Fields` key to the body (after `error`, `kind`, `next_step`, `cause`; a Fields key may override
`error`). Change `lockedWrite` and the `/api/action` branch: a `*statusError` keeps its own status;
otherwise the old rule (`blocked:` -> 409, else 502). Test: a handler returning a statusError{409,...}
gives 409 and the extra field.

### 4.1 build.ps1 - find Go
Replace the hard-coded path. Order: `$env:DECK_GO` (if set and the file exists), then
`C:\Users\mflma\tools\go\go1.26.1\go\bin\go.exe`, then `D:\tools\go\go1.26.10\go\bin\go.exe`, then
`Get-Command go.exe`. None found -> `throw 'Go not found: set DECK_GO to go.exe (tried: <list>)'`.
Print `Go: <path>` once. Keep everything else in build.ps1 as it is.

### 4.2 R19 - reservations in the frontend's shape (guards.go)
The frontend calls `POST /api/reservations {"task","owner","paths"}`, `POST /api/reservations/release
{"id"}`, `GET /api/reservations`. Replace the old owner/repo/release design:
```go
type Reservation struct {
	ID      string    `json:"id"`      // "res-<unixnano>"
	Task    string    `json:"task"`    // e.g. "tasks/x.md", 1-200 bytes
	Owner   string    `json:"owner"`   // agent or profile name, 1-120 bytes
	Paths   []string  `json:"paths"`   // normalized, see below
	Created time.Time `json:"created"`
}
func normalizeReservationPath(root, p string) (string, error)
```
- Normalize: trim spaces; `\` -> `/`; strip a leading `./`; if absolute, make it relative to `root`
  (`filepath.Rel`); reject (error) when it leaves root (`..`), is empty, contains `*` or `?` other than
  one trailing `/**`, or contains NUL. Strip the trailing `/**` and any trailing `/`. Lower-case. Result
  e.g. `app/web/app.js`, `app/web`.
- Overlap: `a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")`.
- POST: 1-128 paths, task/owner required; a conflict with ANY existing reservation of another task
  -> `&statusError{Status:409, Msg:"reservation conflict", Fields: {"conflicts": []map[string]string{{"path","task","owner"}}}}`
  listing every overlapping pair (path = the requested normalized path). A reservation with the same
  task replaces the old one of that task. Success -> `{"ok":true,"id":..., "reservations":[...]}`.
- Release: `{"id"}` -> removes it; unknown id -> 404 statusError `reservation not found`. Response
  `{"ok":true}`.
- Limit 200 reservations (else 409 `reservation limit reached`).
- Persist the full list after every change with the existing `atomicWrite` to
  `filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-reservations.json")`. Load it once in
  `initGuardsLocked` (missing file = empty; unreadable/invalid file -> keep empty list AND store the
  error text in a new `GuardState.LoadError string`, returned as `"load_error"` by GET; never silently).
- GET -> `{"reservations":[...], "load_error":""}`.
- Keep `normalizeOwnership`/`overlaps` unchanged: the scope-baseline routes still use them.
- Tests must set `APPDATA` to `t.TempDir()` (`t.Setenv`) so no real file is touched. Cover: create,
  conflict 409 with conflicts list, same-task replace, release, unknown release 404, traversal rejected,
  `app\WEB\App.js` and `app/web/app.js` overlap, reload from file in a new App.

### 4.3 R20 - account guard (guards.go)
Add `func normalizeAccount(p Profile, account string) string`: `""`, `"main"`, `"default"` (any case)
-> `p.DefaultAccount`; otherwise lower-cased trimmed input.
In `launchGuard`, for `profile.Kind == "claude"`: compare `normalizeAccount(profile, p.Account)` with
`a.lead.Account`. Refuse when the lead is unknown or they are equal, with
`&statusError{Status:409, Msg:"account guard: pick another account (the lead uses <lead or unknown>)", Fields:{"lead_account": a.lead.Account}}`.
The helper (`p.Helper`) rules stay, but also return 409 statusErrors. Never read credential files.
When the start goes ahead and the normalized account equals `profile.DefaultAccount`, call the
helper WITHOUT an `Account` param (the main sign-in has no account folder).

**Profile `next`** (Claude on the least-used account): today it is passed straight to
`herdr-agents start next`, which may pick the lead's own account. New rule, in `action` before the
`next` branch: call `a.herdr.Helper(ctx, {"Command":"usage"})`, parse it with
`func pickNextAccount(usage, lead string) (string, error)`. Real output format (verbatim):
```
account  5h     5h resets    weekly  week resets  session cost (updated today) data age
2030*    16%    Mon 11:50    11%     Fri 13:00    $6.72                        0 min
1984     0%     Sun 17:00    4%      Fri 13:00    $0.00                        1188 min
2040     0%     Fri 21:00    90%     Tue 07:00    $0.00                        3798 min
(* = the main sign-in; LIMIT = at or above 95% of the 5-hour or weekly limit)
NEXT: 1984
```
Rows: `^(\S+?)\*?\s+(\d+)%\s+.+?\s(\d+)%\s`. Skip the lead's account and any row with 5h or weekly
>= 95. Pick the lowest 5h, then lowest weekly, then name order. None left -> 409 statusError `no
Claude account available besides the lead`. Then start profile `claude` with that account (main
alias rule above). Lead unknown -> 409. Tests use the text above: lead 2030 -> 1984 (5h 0 ties
2040, weekly 4 < 90); lead 1984 -> 2040 (5h 0 < 16); lead 2040 -> 1984; all rows at 95% -> error.

### 4.4 R21 - RAM field and floors
`/api/health`: add `ram_free_mb` (same number as `ram_available_mb`); keep every old field.
`paneRequest` gets `Floor *struct{ RAMFreeMB float64 `json:"ram_free_mb"`; DiskFreeGB map[string]float64 `json:"disk_free_gb"` } `json:"floor"``.
In `launchGuard`, when Floor is set, compare with `systemHealth()` (RAM) and the `disks` list (drive
like `C:`); below -> 409 statusError `held: below the requested floor` with Fields
`{"ram_free_mb":<now>,"disk_free_gb":{"C:":<now>}}`. The existing `held:` errors also become 409.

### 4.5 R22 - research routes
`GET /api/headless/routes` -> `{"routes":[], "note":"No server-search-only research route has been verified for the Deck; research stays unavailable."}`.
Keep it a fixed empty list (never claim research works). In `toolEndpoint` `/api/headless`: add an
optional `"purpose"` field to its request struct; `purpose == "research"` -> 409 statusError `research
route unavailable: no verified server-search route`. Without purpose, behaviour is unchanged.

### 4.6 R23 - dispatch IDs and uncertain delivery
`Delivery` becomes:
```go
type Delivery struct {
	ID      string    `json:"id"`          // same value as DispatchID (old field, keep)
	Dispatch string   `json:"dispatch_id"`
	Key     string    `json:"key"`
	Pane    string    `json:"pane"`
	TextSHA string    `json:"text_sha"`    // hex sha256 of the prompt text
	State   string    `json:"state"`       // "sent" | "unknown" | "failed"
	Error   string    `json:"error"`
	At      time.Time `json:"at"`
}
```
Change `deliver` to `deliver(ctx, key, pane, text string, args ...string) (string, string, error)`
returning (output, dispatchID, err). Mapping: success -> `sent`; error containing `exit status 2`,
`agent_not_ready`, `agent_blocked`, `invalid_request` -> `failed` and a 502 statusError with
Fields `{"dispatch_id":id}`; any other error (timeout, transport) -> `unknown` and
`&statusError{Status:504, Msg:"delivery unknown", Fields:{"dispatch_id":id,"detail":err.Error()}}`.
Never retry. Keep the 100-record cap. The key for a pane = `a.state.Session + "|" + paneID`.
`action` "prompt" passes them; the success text stays `Dispatch <id> accepted ...`.
The alias route `/api/pane/prompt` success JSON adds `"dispatch_id"`: return it from `action` without
parsing text - add an unexported field `lastDispatch string` to the result path, e.g. make `action`
store the id in a local variable through a small wrapper `a.prompt(...)`; your choice, but no regexp
on output text.
New `GET /api/dispatches?key=<key>` -> `{"dispatches":[...]}` newest first, at most 20, filtered by key
when given. `/api/deliveries` keeps working (same records).
Tests with a fake Herdr whose `agent prompt` returns (a) ok, (b) `context deadline exceeded` -> 504 +
dispatch_id, (c) `agent_not_ready` -> 502 + state failed; and a check that the fake saw exactly ONE
prompt call (no retry).

### 4.7 R24 - git paths and root (extensions.go `/api/git`)
Add `root` = `git -C <cwd> rev-parse --show-toplevel` with forward slashes, and `paths` =
`[]{"path","code"}` parsed from `git --no-optional-locks -C <cwd> status --porcelain=v1 -z
--untracked-files=all` (code = first 2 bytes; path = repo-relative, forward slashes; for R/C entries
use the NEW path and skip the next NUL field). Put the parser in `func parsePorcelainZ(raw []byte)
[]map[string]string` and unit-test it with a hand-made byte string including a rename and a path with
spaces. Keep `status`, `stat`, `text`.

### 4.8 /fonts cache (run.go)
The page loads `fonts/Alexandria-latin.woff2`. Extend the existing `/assets/` block so it also handles
paths starting with `/fonts/`: same ETag, `Cache-Control: private, max-age=31536000, immutable`,
`Content-Type: font/woff2` for `.woff2`, 304 on matching `If-None-Match`. Move that handler into a
function `func staticHandler(files fs.FS) http.Handler` so a test can call it with `fstest.MapFS`.

### 4.9 /api/herdr/start on a named session (server.go action "launch")
Today it always starts `launcher\HerdrLauncher.exe`, which opens the DEFAULT session even when the
Deck runs with `--session lab`. Add fields to `App`: `sessionName, herdrBin string` and
`startDetached func(name string, args ...string) error` (default: `hiddenCommand(context.Background(),
...)` + `Start()` + `go cmd.Wait()`). In `run.go` set `app.sessionName = *session` and `app.herdrBin = bin`.
Rule: `sessionName != ""` -> `startDetached(herdrBin, "--session", sessionName, "server")` and return
`"Herdr server for session <name> started"`; else the old launcher path, also through `startDetached`.
Test with a recording `startDetached`: session "lab" -> args exactly `--session lab server`.

### 4.10 NEW: input_not_empty guard (new file input.go)
```go
// inputText returns the text the owner has typed into an agent's input box, or "" when the box is
// empty or shows only dim suggestion text. tool is the pane's agent kind ("claude", "codex", ...).
func inputText(screen, tool string) string
```
Facts from REAL Herdr ANSI output (see `app/tests/fixtures/input/claude-empty.real.ansi`):
- Herdr renders colours as truecolor SGR: grey prompt/suggestion = `ESC[38;2;153;153;153m`, rules =
  `ESC[38;2;136;136;136m`, reset = `ESC[0m`. It does NOT use SGR 2/90 for Claude's grey.
- Claude Code's input box: a rule line (only `─` U+2500 after stripping ANSI and spaces, length >= 20),
  then the prompt line starting with `❯` (U+276F) followed by a NO-BREAK SPACE U+00A0 (not 0x20!),
  possibly more input lines, then another rule line. Below it: status lines.
- Codex: the prompt line starts with `›` (U+203A) + space; its empty-box placeholder is dim (SGR 2).
Algorithm:
1. Split into lines; look only at the last 40 lines.
2. Claude (`tool == "claude"` or the box is found): find the LAST `❯` line that has a rule line
   directly above it (ignore blank lines) and a rule line below it within 12 lines. The input region =
   the `❯` line after the glyph and its following space/NBSP, plus all lines up to the lower rule.
3. Codex (`tool == "codex"`): the LAST line in the last 10 lines that starts (after trimming spaces)
   with `›`; region = the rest of that line after `› `.
4. In the region, walk the ANSI SGR state: a visible rune counts as TYPED unless it is space/NBSP
   or the current style is dim. Dim = SGR 2, or 90, or `38;5;n` with n == 8 or 232 <= n <= 250, or
   `38;2;r;g;b` with max-min <= 12 and max <= 170 (a mid grey). `0`/`39`/`22` reset colour/dim.
5. Return the typed text (ANSI stripped, trimmed, at most 200 runes). Other tools / no box -> "".
Expected results for the fixtures (write one table-driven test that reads them from disk with
`os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "input", name))`):
| fixture | tool | want |
|---|---|---|
| claude-empty.real.ansi | claude | "" |
| claude-typed.synthetic.ansi | claude | "please also fix the tests" |
| claude-typed-white.synthetic.ansi | claude | "stop and wait" |
| claude-suggestion.synthetic.ansi | claude | "" |
| claude-multiline.synthetic.ansi | claude | contains "second line typed by the owner" |
| claude-history-prompt.synthetic.ansi | claude | "" (the old `❯` line has no rule above it) |
| codex-empty.synthetic.ansi | codex | "" |
| codex-typed.synthetic.ansi | codex | "fix the failing test first" |
| shell-prompt.synthetic.ansi | "" | "" |
Wire it in (read the pane with `pane read <id> --source visible --lines 60 --format ansi`; reuse that
screen for `blockedScreen`, which strips ANSI itself):
- `action` "prompt" and "paste" for panes with an agent (`pane.Agent != "" || pane.Name != ""`).
- `toolEndpoint` `/api/agents/ask` (find the pane by name in the snapshot) and `/api/agents/fanout`
  (check EVERY named agent first; if any has text, send nothing).
Refusal: `&statusError{Status:409, Msg:"input_not_empty", Fields:{"error":"input_not_empty",
"agent":<name or pane id>,"pane_id":id,"typed":<first 60 runes>,"message":"<agent> has text in its input box - not sent"}}`.
No write may happen before the check (test with `fakeHerdr.writes`: zero writes on refusal).
`fakeHerdr` returns `f.screen` for every `pane read`; set it to a fixture's text.

### 4.11 API.md
Update ONLY the rows of the status table you changed (R19-R24 guards rows, `/api/herdr/start`, fonts,
health), with the test name as evidence, and add a section `## Round 3 (2026-10-26)` documenting each
new/changed shape exactly as implemented (routes, JSON in/out, statuses). DONE only with a passing
test; anything not live-tested says "unit-tested; lab not run".

## 5. Steps
1. Read section 2 files. Run `pwsh -NoProfile -File app\build.ps1` once to see it pass (baseline).
2. 4.0, then 4.1 ... 4.10 in order. After each item: add its tests, run the fast loop, fix.
3. Old-code check (AGENTS.md rule 4): make `app\tests\old-check\` = a copy of `app\` WITHOUT
   `tests\go-cache`, `tests\go-tmp`, `HerdrDeck.exe`, `web\fonts`; in it, replace every changed `.go`
   file with `git show HEAD:app/<path>` (do not copy new non-test .go files); copy your new/changed
   `_test.go` files in; run `go vet`-free `go test ./internal/deck/ -count=1` there. Report how many
   tests fail or that the package does not compile (name the missing symbols). Then delete
   `app\tests\old-check\`.
4. 4.11 API.md. Full `pwsh -NoProfile -File app\build.ps1`. Delete temp folders you made.
5. Write the answer file.

## 6. Definition of done (all must be true)
- [ ] `pwsh -NoProfile -File app\build.ps1` exits 0 and prints `Go: D:\tools\go\go1.26.10\go\bin\go.exe`.
- [ ] `go test ./internal/deck/ -count=1 -v` lists every new test as PASS (paste the names + count).
- [ ] The fixture table in 4.10 passes row by row.
- [ ] Old-code check result reported (fail count or compile errors).
- [ ] `git status --short` shows changes only in the section 3 files (+ your answer file).
- [ ] No `app\tests\old-check` or other temp folder left.

## 7. Answer format
Write `tasks/deck-r3-ds-backend-answer.md` and reply with exactly the same text:
```
DONE | BLOCKED | PARTIAL
Changed: <files>
Items: 4.0..4.11 each -> done / partial (why)
Checks: build exit code + exe size; go test count; new test names; old-code check result
For HISTORY/LESSONS (the lead writes them): <durable facts you learned, with proof>
Not done / doubts: <list, or "none">
```
Stop only when done or truly blocked; do not ask questions, decide by this spec.
