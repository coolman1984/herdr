# Deck round 4 backend answer

DONE — C1–C10, D6 multiline, D8, D10, P2, P3, and D12/D17 backend work implemented.
API.md R26–R35 documents the contracts. All **35 new top-level regression tests pass on
current code and fail behaviorally on HEAD**. Subtests are not counted separately.

## Per-item evidence

Paths below are under `app/internal/deck/`. Every named `TestR4...` test returned **FAIL on
old code**, and **PASS on current code**.

| ID | What changed (file:function) | Regression tests; old-code result |
|---|---|---|
| C1 / D3 / P4 | `server.go:events, poll, refresh, publish`; `extensions.go:extensions`; `r4.go:setVisibleLocked, clientStateLocked`. Hello/settings/state ordering; independent client sets and union reads; filtered screen events; disconnect cleanup; metadata polling with empty interest; stale panes retained offline. | `TestR4VisibilityIndependent`, `TestR4ClientScreensAndDisconnect`, `TestR4HiddenClientMetadataHeartbeat`, `TestR4OfflineRetainsStalePanes` — FAIL, all four. |
| C2 / D18 | `server.go:ServeHTTP`; `r4.go:queueLocked`. Read-only changes broadcast settings; overflow snapshots carry current settings. | `TestR4SettingsBroadcast` — FAIL; initial settings also exercised by stream tests. |
| C3 / D2 | `r4.go:r4Endpoint, rawTurn, normalizeKey`. Raw input validates the whole batch, sends text then keys, uses per-pane FIFO tickets, allows blocked/occupied panes, enforces read-only. | `TestR4RawOrderedAndReadOnly` — FAIL (also asserts invalid batches send nothing). |
| C4 / D5 | `guards.go:guardEndpoint, Reservation`. Same owner/task is idempotent; different owners retain separate records; explicit overlap persists shared ownership; unreadable ledgers fail closed with 503 and retain GET load_error. | `TestR4ReservationOwnershipAndOverlap`, `TestR4UnreadableReservationLedger` — FAIL, both. |
| C5 / D4 | `admission.go:admit, scriptLaunchPlan, reconcilePendingLocked, admission.finish`; `server.go:action`; `parity.go:script`; `jobs.go:startJob`; `tools.go:toolEndpoint`. One admission service expands plans, applies account/resource rules per agent, atomically reserves pending slots, releases failures, expires at 120 s and reconciles snapshot confirmations. Columns/shell-only plans need zero agent slots. | `TestR4LaunchPendingAndIdentity`, `TestR4EveryLaunchRouteUsesAdmission`, `TestR4LaunchFailureReleasesPending`, `TestR4TeamReservesExpandedPlan`, `TestR4LayoutChecksEveryAccount`, `TestR4SnapshotConfirmsPendingLaunch`, `TestR4PendingLaunchExpiresAt120Seconds` — FAIL, all seven. Expiry uses Go's virtual clock, including refusal at 119 s and admission at 120 s. |
| C5 / D12 | `admission.go:startedIdentities`; `extensions.go:extensions`. Start replies add name/pane_id only from a complete STARTED line; absent otherwise. | `TestR4LaunchPendingAndIdentity` — FAIL (missing identity on old code). |
| C6 / D24 | `herdr.go:runCommand`; `guards.go:deliver, recovery`; `server.go:failHTTP`; `r4.go:commandError, readFailure, notSent`. Child-start provenance distinguishes not_sent from uncertain delivery; read timeouts have no dispatch id and no delivery-unknown recovery classification. | `TestR4ReadTimeoutIsNotDelivery`, `TestR4CancelledWriteNeverStarts`, `TestR4MissingExecutableNotSent` — FAIL, all three. Existing uncertain/rejected delivery tests still pass. |
| C7 / D6 | `input.go:inputRegion, inputNotEmpty, claudeRegion, codexRegion, inputStylePrefix`. Unrecognized Claude/Codex composers return input_unknown. Full multiline regions, tall Claude boxes, blank Codex continuation rows and inherited SGR state are parsed. Other tools keep their permissive unknown-input behavior. | `TestR4UnknownInputAndMultiline`, `TestR4CodexTallDraftWithBlankLines`, `TestR4InheritedInputStyle` — FAIL, all three; existing ANSI fixture tests pass. |
| C8 / D20 | `r4.go:r4Endpoint`. GET/POST prefs; whole-object atomic replacement under APPDATA/herdr/deck-prefs.json; 64 KiB body limit/413; restart persistence; explicit read/write failures. | `TestR4PrefsPersistAndLimit` — FAIL. |
| C9 / D19 | `platform_windows.go:systemHealthVolumes`; `admission.go:launchFolder`; `guards.go:launchGuard`; `extensions.go:cachedHealth`. D: plus pane/launch/worktree volumes; explicit null unknown disk/memory values; launch-folder and worktree-volume floors; disk_unknown for unknown nonzero floors. | `TestR4HealthIncludesDAndNullUnknown`, `TestR4HealthIncludesPaneVolume`, `TestR4UnknownLaunchVolumeRefused`, `TestR4WorktreeVolumeFloor` — FAIL, all four. |
| C10 / I1 backend | `r4.go:r4Endpoint, countSSE, finishPoll`; `socket_windows.go:Socket.waiting`. Always-on, token-protected counters with every specified field; event bytes/count, poll errors/durations, live clients/jobs, queued RPCs and resyncs. | `TestR4Counters` — FAIL. |
| D8 | `model.go:parseSnapshot`; `server.go:refresh`. Store source lifecycle status separately; recompute text-derived blocking from the newest successful screen instead of promoting cached blocking into permanent lifecycle state. | `TestR4BlockedClearsOnFreshScreen` — FAIL. |
| D10 | `model.go:parseScreen`; `diff.go:validPatches`; `server.go:publish`. Last 2000 retained rows match the client; validate evolving patch coordinates/version sequence before publishing, with full-screen fallback. | `TestR4RetainedRowsAndPatchBounds` — FAIL; existing patch-reconstruction and compact-scroll tests pass. |
| P2 | `server.go:refresh`; `socket_windows.go:acquire, request`; `extensions.go:cachedHealth`. Four-read round robin, per-read budgets, 2–32 s failure backoff, preserved cached rows on malformed reads; no agent-list RPC; background reads defer to active/queued requests; cancellable socket admission and in-flight I/O; one cached health sampler with bounded memory probes. | `TestR4PollFairnessAndBackoff`, `TestR4HealthSingleSample`, `TestR4SocketCancelledBeforeAdmission`, `TestR4FailedScreenRetainsLastGoodRows` — FAIL, all four. Socket test also holds admission and verifies a queued request times out without touching a pipe. |
| P3 | `diff.go:rowChangesRows, rowDiffRows`; `server.go:publish`. Split once, compute prefix/suffix once, avoid false scrolls for repeated-row tail edits, cap shift-comparison work and validate resulting patches. | `TestR4DiffTailUsesSinglePatch` — FAIL; existing reconstruction/scroll tests pass. |
| D17 backend | `r4.go:queueLocked`; `jobs.go:startJob`. Completed results remain in the jobs ledger. If overflow would discard a result, close only the affected SSE stream so reconnect fetches authoritative jobs. Ordinary screen overflow still resyncs in place. | `TestR4ResultOverflowReconnectsForLedger` — FAIL. Frontend reconciliation remains Sol's scope. |

## Checks and observed output

All Go commands ran through the shared lock. No direct go.exe invocation, browser, live
Herdr operation, agent launch, or mutating Git command was used.

- `pwsh -NoProfile -File app/tests/go-locked.ps1 vet ./...`
  - `vet_exit=0`, no diagnostics.
- `pwsh -NoProfile -File app/tests/go-locked.ps1 test ./... -count=1`
  - `? herdrdeck [no test files]`
  - `ok herdrdeck/internal/deck 29.460s`
  - `? herdrdeck/tests/agentstub [no test files]`
  - `test_exit=0`.
- `pwsh -NoProfile -File app/build.ps1 -Output "$env:TEMP\astra-r4\HerdrDeck.exe"`
  - Its own format/vet/test/build passed; package tests: `ok herdrdeck/internal/deck 26.551s`.
  - `Built: C:\Users\m.labib\AppData\Local\Temp\astra-r4\HerdrDeck.exe`
  - `Size: 8771584 bytes (8.37 MiB)` — below 12 MiB (and 12,000,000 bytes).
  - `SHA256: 254B127AA4994F871583C62A75EC3B2B86E54AB2B298231B04B23681C9302502`
  - `build_exit=0`.
- Old-code comparison:
  - Baseline HEAD: `0dbcad5f5f435a9a9efc1ca49a27200726154267`.
  - Temp copy: `%TEMP%\astra-r4\old-app`; production Go files copied from read-only
    `git show HEAD:<file>`, new production files omitted, current tests retained.
  - `pwsh -NoProfile -File app/tests/go-locked.ps1 -C "$env:TEMP\astra-r4\old-app" test ./internal/deck -run '^TestR4' -json -count=1`
  - `Old-code exit=1`; **35/35 top-level tests FAIL, zero unexpected passes**.
  - Evidence: `%TEMP%\astra-r4\old-tests.jsonl`; baseline hash in `baseline-head.txt` beside it.
  - An initial comparison found a new Stale-field reference in an updated existing test;
    that direct field assertion was removed (the JSON regression still checks stale:true).
    The reported 35 failures are from the subsequent compiling, executing test run.
- `git diff --check` returned 0; line-ending warnings are Git's CRLF policy, not whitespace errors.

## Intentional existing-test updates

- `TestScreenReadAndCap`: 2040 -> 2000 rows (D10).
- `TestRefreshDisplaysBlockedAndRecoversFromOffline`: offline retains one pane (C1).
- `TestSSEDisconnectReleasesClientsAndGoroutines`: first event is hello (C1).
- `TestPromptPreservesQuotesNewlinesAndMetacharacters`, `TestPromptDispatchIDAndDispatchLedger`,
  `TestUncertainDeliveryIs504AndRejectedDeliveryIs502`: use recognized empty Codex input
  fixtures instead of arbitrary/empty screens (C7); write/ledger assertions retained.
- `testApp` explicitly marks its fake page's pane visible; production reads require visibility (C1).
- Reservation lifecycle assertion messages now describe idempotency, not replacement;
  the new ownership test verifies stable IDs and preservation of both owners.

## Scope and deviations

No contract deviations. Product changes are confined to `app/internal/deck/*.go` and
`app/API.md`; this answer file is the explicitly requested handoff exception. No changes
were needed in `app/tests/agentstub`. The lead's R25 fix is preserved. No web, build-script,
PowerShell test, or app/HerdrDeck.exe edits were made by this task. Concurrent frontend,
history/lessons, build-lock and other pre-existing changes were left in place. The build
formats internal Go files; files with no content diff were restored to their original CRLF
line endings after checking them with read-only git diff.

## Doubts

- Live Herdr/browser integration, actual current provider UI captures, and sustained process
  handle/RSS performance are not verified here; they remain the lead's designated checks.
- The live socket integration test remains skipped without DECK_TEST_SOCKET. Unit fakes and
  cancellation tests do not claim an accessible live Herdr pipe. No race-detector run was made.
- I2/I4 were intentionally not attempted. Snapshot-to-write identity races requiring upstream
  atomic compare-and-send remain outside this task.
- `git status` may warn about Sol's inaccessible `app/tests/web/.baseline-*` scratch directory;
  it is outside this task and was not changed or removed.
