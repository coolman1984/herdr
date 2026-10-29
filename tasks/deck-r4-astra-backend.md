# Task: Fix every backend item of the Deck round-4 report

## 1. Role and goal
You are a senior Go engineer working for the lead (Claude). You wrote the review
`tasks/deck-r4-astra-report.md`; now implement its backend fixes in `app/internal/deck/`.
Goal: every backend item below fixed, each with a test that fails on the old code, and the contracts in
`tasks/deck-r4-contracts.md` implemented exactly.
Why: the owner wants the Deck safe and fast. Sol is implementing the web side in parallel against the
same contracts, so field names and status codes must match the contract file letter for letter.

## 2. Context
- Folder `D:\WORK\Software Development\GitHub\HERDR`. Read `CLAUDE.md`, `AGENTS.md`, your report, the
  contracts, `app/API.md`, HISTORY Phase 50.
- Already fixed by the lead (do not redo): D1 (prompt `expect`, R25), D7, D9, D11, D14, D21, D22, D23, D25.
- Existing helpers to reuse: `statusError`, `lockedWrite`, `currentSnapshot`, `deliver`, `launchGuard`,
  the fake Herdr in `server_test.go` (`fakeHerdr`, `testApp`, `request`).

## 3. Exact scope
- Change ONLY: `app/internal/deck/*.go` (new files allowed), `app/API.md` (add R26-R35, one per
  contract/fix group, with its test names), `app/tests/agentstub/*`.
- Do NOT touch `app/web/*` (Sol owns it), `app/build.ps1`, `app/tests/*.ps1`, docs outside `app/API.md`,
  `app/HerdrDeck.exe`. No git commands that change anything. Do not start agents. No browser.
- Builds and tests ONLY through the shared lock (two Go builds at once ran this PC out of RAM):
  `pwsh -NoProfile -File app\tests\go-locked.ps1 test ./... -count=1` (any go args) and
  `pwsh -NoProfile -File app\build.ps1 -Output "$env:TEMP\astra-r4\HerdrDeck.exe"`. Never call go.exe directly.
- Your sandbox cannot reach Herdr pipes or the network; prove behaviour with unit tests and fakes. The
  lead runs the live/browser integration afterwards.

## 4. Work list (all of it)
Contracts: C1 (per-client visibility, union reads, metadata heartbeat, per-client screen events, stale
panes offline), C2 (settings event), C3 (`/api/pane/raw`), C4 (reservation rules, 503 ledger), C5 (one
admission for every launch route + `pane_id`/`name` in the start response), C6 (typed timeout errors),
C7 (`input_unknown` for claude/codex), C8 (`/api/prefs`), C9 (disk volumes, null unknowns), C10
(`/api/debug/counters`).
Report items without a contract: D8 (blocked state recomputed from the latest screen, cleared on
negative evidence), D10 (Go trims retained rows to the last 2000 = client MAX_LINES before diffing;
validate patch bounds), D6 multiline parsing (whole Codex/Claude input region, taller boxes), P2 (fair
round-robin screen reads with per-request budget and failure backoff, cancellable socket wait, no
redundant agent list, cached health with in-flight guard), P3 (one diff pass on pre-split rows, prefix/
suffix fast path, bounded shift detection), D12/D17 backend parts if any.
Do not attempt I2 (needs upstream Herdr) or I4.

## 5. Steps
1. For each item: write the failing test first (run it, note FAIL), implement, run it (PASS).
2. Keep the existing tests passing; update an existing test only where a contract changes the
   behaviour on purpose (e.g. offline now keeps stale panes) and say which.
3. Full `go-locked.ps1 vet ./...`, `go-locked.ps1 test ./... -count=1`, then `build.ps1 -Output ...`.
4. Old-code check: copy `git show HEAD:<file>` versions of the changed non-test files into a temp copy
   of `app/` with your new tests and report how many of your new tests fail there.
5. Write `tasks/deck-r4-astra-answer.md` (format below) and reply.

## 6. Definition of done
- [ ] Every contract C1-C10 implemented with field names exactly as written; API.md R26-R35 added.
- [ ] D8, D10, D6, P2, P3 fixed with tests.
- [ ] vet 0, all tests pass, build exit 0 (to your temp output), exe < 12 MB.
- [ ] New tests fail on the old code (count reported).
- [ ] `git status --short` lists only files in your scope (plus files other agents changed - leave them).

## 7. Answer format
`tasks/deck-r4-astra-answer.md`: per item - ID, what changed (file:function), test name, old-code result.
Then: checks with output lines, deviations from the contract (should be none; if unavoidable, why), doubts.
Reply in the pane exactly:
DONE | BLOCKED | PARTIAL
Answer: tasks/deck-r4-astra-answer.md
Checks: vet=<exit> test=<exit> build=<exit> new-tests-failing-on-old=<n>/<m>
Not done / doubts: <list or "none">
