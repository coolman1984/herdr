## Summary

Prevent agent messages from falling through to shell execution; bind every write to the intended pane incarnation (D1).
Fix raw-input ordering, per-page visibility, and launch/reservation admission before adding more controls (D2–D5).
Replace font shrinking with an explicitly reflowing prose view; the current fixed-row renderer cannot meet the owner's width requirement (D7).
Separate agent lifecycle from terminal text heuristics, and repair snapshot/patch coordinates and scroll anchors (D8–D11).
Stop rebuilding navigation and split dividers on ordinary screen diffs; this is a direct source of unnecessary work in the real SSE path (P1).
Bound background queues and replace serial, competing polling with a fair scheduler and cached health sampling (P2–P4).
Reorganize card identity and controls, preserve drawer edits, and reconcile asynchronous jobs after reconnect (D14–D17).
Evidence below distinguishes executed probes, source proofs, and unverified live behavior. Historical PERF.md numbers are not current measurements.
Unqualified Go filenames refer to `app/internal/deck/`; browser filenames refer to `app/web/`.

## Must fix (bugs)

### D1 — High — Agent messages can execute at a shell prompt

**Symptom:** Screenshot `2026-10-28 17 40 25.png` shows an exited agent, `unknown`, and prose producing a PowerShell `ParserError`.

**Cause / evidence:** `app/internal/deck/server.go:741` reads a fresh screen but at `:755` explicitly changes `agent prompt` to `pane run` when both agent and name are absent. `input.go:36` exempts shells from the occupied-input guard. `app/web/api.js:87` sends only the pane key and text, so the server cannot know which agent the user intended. `app.js:396` keeps the composer available. Executed `TestAstraShellPrompt` against a fake fresh shell snapshot returned `err=<nil> writes=[pane|run|w1:p1|please review]`, failing the zero-writes assertion. This directly explains the unsafe behavior in the screenshot; a fresh snapshot alone does not protect intent. A pane ID can also survive an agent exit/replacement, or be reused after a server restart.

**Fix:** Introduce explicit `agent_prompt`, `shell_command`, and raw-terminal intents in the API. Default the card composer to agent prompts, reject missing/replaced agents with 409, preserve the draft, and display “Agent exited / shell remains.” Require an explicit switch to a clearly labeled shell composer before `pane run`. Send expected agent/process identity and server/session generation; validate through one dispatch service used by composer, broadcast, queued tasks and hub routes. A final atomic compare-and-send needs Herdr support to close the snapshot-to-write race; until then describe that residual limitation and fail closed when identity is uncertain.

**Verify:** Fake an agent exiting between page render and POST: zero shell writes and draft preserved. Repeat with reused pane ID, renamed/replaced agent, queued/broadcast send and explicit shell mode. Exercise exit between validation and dispatch against an isolated server when upstream identity checks exist.

### D2 — High — Raw typing can change targets, reorder Enter, and reject its own continuation

**Symptom:** Characters buffered for A can land in B; Enter may arrive before text; typing a second chunk can stop with `input_not_empty`.

**Cause / evidence:** `app/web/panels.js:434` has one global `rawBuf/rawTimer`; `rawMode()` changes `S.raw` without draining the old target. `flushRaw():452` resolves the target only when the timer fires. `rawKey():461` calls `flushRaw()` and `api.keys()` without awaiting the first request. `server.go:766` routes text through the occupied-composer guard, which cannot distinguish the owner's previous raw chunk from conflicting input. Executed `TestAstraRawSecondChunk`, with Codex screen `› first` and text ` second`, returned `error=input_not_empty writes=[]`. Serialization by a server mutex does not establish browser request order.

**Fix:** Give each raw interaction an immutable target/identity and ordered async queue. End or drain A before activating B; await text completion before keys. Add a deliberate raw-input endpoint/transaction distinct from “paste into an empty agent composer,” retaining safe-mode and identity checks. Handle composition events, graphemes, cancellation and paste without synthesizing duplicate key input.

**Verify:** Buffer text, switch panes within 60 ms, then press Enter with delayed/reordered HTTP responses. Assert exact bytes and keys per pane, multiple consecutive chunks, IME text, leaving raw mode, and safe-mode changes during a queued write.

### D3 — High — One page can stop polling for every page; offline state can become permanent

**Symptom:** Hiding a tab or selecting an empty view freezes another page; a failed snapshot can prevent automatic Herdr recovery.

**Cause / evidence:** `extensions.go:204` replaces global `a.visible` and sets `a.active = len(ids)>0`; legacy `server.go:163` has the same global design. `server.go:353` gates **all** refreshes on that flag. `app.js:190` sends an empty visible list for a hidden page. Executed `TestAstraEmptyVisibilityStopsMetadata`: POST one visible key, then an empty list; output `active=false visible=map[]`. After a snapshot error, `server.go:382` publishes zero panes; `app.js:119` sends visibility again, potentially setting active false permanently. No remaining metadata poll discovers recovery or newly created panes.

**Fix:** Maintain visibility/activity per SSE client, expire disconnected clients, and poll the union of their interests. Keep a low-frequency metadata/recovery heartbeat whenever a page is connected, even with zero visible panes. Distinguish offline/unavailable data from a verified empty session; show stale panes with writes disabled until refreshed.

**Verify:** Two pages with disjoint filters; hide/close either; hide all cards; start from an empty session; stop/restart only the isolated Herdr server. Discovery and the other page must continue without a manual refresh.

### D4 — High — Launch limits/account checks are bypassable and queued launches oversubscribe

**Symptom:** Equivalent “start agents” controls enforce different policies; a queued wave can exceed the intended capacity.

**Cause / evidence:** `parity.go:236` workspace-open and `:312` layout-build call scripts/jobs without the `launchGuard()` used by `server.go:635`. `tools.go:142` team creation likewise constructs Claude/Codex launches directly. `guard.js:62` drains its entire queue synchronously while `capacity()` counts only already-observed agents. `guards.go:303` also counts cached state, with no pending-launch reservation. `jobs.go:47` limits script jobs, not the number/accounts of agents those jobs create.

**Fix:** Centralize launch admission in Go for every route, expand plans before admitting them, validate each account and resource requirement, and reserve pending agent slots atomically. Convert reservations to live identities on confirmed launch; release on failure/timeout. Drain the browser queue sequentially from server admission results. Reconcile the browser's local limits with the actual backend policy instead of displaying a separate promise.

**Verify:** Submit concurrent start/team/layout/workspace requests from two pages with one available slot. Assert the total admitted agents, account rules and failure cleanup. Test a team larger than available capacity and script completion before metadata discovery.

### D5 — High — File ownership is reserved after dispatch and can be silently reassigned

**Symptom:** Work can start before a conflicting reservation is rejected; launching the same task for a second owner can replace the first reservation.

**Cause / evidence:** `guard.js:171–184` calls `reserve()` only after `api.start()` or `sendGuarded()`, and catches conflicts as a toast. `guards.go:544` drops an existing reservation solely because its task matches, irrespective of owner. `guards.go:418` returns `load_error`, while `guard.js:147` consumes only `reservations`, treating an unreadable ledger like an empty one.

**Fix:** Make reserve/preflight/dispatch a backend workflow: validate ownership and persistence first; dispatch only after success; roll back a newly acquired reservation if nothing was sent. Make repeat ownership idempotent by reservation ID and owner, and require an explicit transfer/overlap operation for a different owner. Surface ledger load errors as unavailable, never as no conflicts. The UI's “accept overlap” must map to a defined backend operation rather than a post-dispatch failed reserve.

**Verify:** Two concurrent owners, same task/different task overlapping paths, corrupt/unreadable ledger, failed dispatch, and retry after an ambiguous delivery. Assert no unauthorized work starts and no existing reservation disappears.

### D6 — High — An unrecognized or multiline input box is treated as empty

**Symptom:** The occupied-input protection can permit a prompt/paste while the owner has an existing draft, especially with unsupported agent UIs or multiline input.

**Cause / evidence:** `input.go:19–32` returns the same empty string for “recognized empty input” and “prompt region not found”; `inputNotEmpty():42` allows both. All non-Codex tools go through Claude's parser. `claudeRegion():75` requires a lower rule within twelve lines, so a taller region cannot be recognized. `codexRegion():101` returns only the glyph's line, ignoring continuation rows; a blank first line plus text on subsequent lines is treated as empty. `typedText()` also assumes dim/grey text is always a suggestion. These are concrete parser paths, not a live capture of every provider's current UI.

**Fix:** Return a typed result with `recognized`, `empty/occupied/unknown`, region bounds and parser capability/version. Refuse automated composer sends when input state is unknown, while preserving the deliberate raw-input workflow in D2. Parse the complete supported multiline region and carry its initial ANSI state; do not use generic grey color alone as proof of placeholder text. Prefer I2's structured input state when available. Make any explicit owner override visible and target-bound.

**Verify:** Unknown provider/layout, no prompt marker, a Claude box taller than twelve lines, Codex blank-first-line multiline draft, wrapped long draft, dim user text and genuine dim placeholders. Unknown/occupied inputs produce zero automatic writes; recognized empty input still sends. Actual current provider renderings remain a required fixture-capture task.

### D7 — Medium — The width implementation cannot provide reflow (screenshots a/b)

**Symptom:** Wide terminal rows scroll sideways; narrow source output leaves unused space. Font sizes also vary between cards.

**Cause / evidence:** `app.js:103` obtains source columns from Herdr layout rectangles. `screen.js:70` scales font size to those columns, clamped between 9 px and 1.25 times the chosen base; it never resizes the PTY or reflows rows. `style.css:225` allows horizontal overflow; `:231` makes each row absolute, fixed-height and `white-space:pre`. `screen.js:67` guesses missing columns with UTF-16 string length, not terminal cell width. For example, **illustrative arithmetic**, 160 columns at 9 px and a 0.6 cell ratio need 864 px before padding, exceeding a 600 px card. Narrow output is still narrow when scaling hits its maximum or the TUI intentionally writes only part of the terminal. Screenshot dimensions are visual evidence, not current DOM measurements.

**Fix:** Implement a stable-font **prose view** in `screen.js`, `style.css` and the screen-read contract. Request `recent_unwrapped` where supported, preserve hard newlines, wrap long words/paths, and virtualize by measured visual height (prefix heights plus an anchor), not `rowIndex*lineHeight`. Require `scrollWidth <= clientWidth` without clipping content. Keep exact terminal representation as an explicit zoom/detail mode. Do not heuristically merge every newline: code, tables and real paragraphs would be corrupted.

The installed CLI's `pane resize --help` returned `--direction` and `--amount`, **not columns/rows**. Its `api schema --json` lists `recent_unwrapped`. Local upstream evidence: `herdr-app/src/api/schema/panes.rs:256` describes split resizing; `herdr-app/src/app/api_helpers.rs:108` supports unwrapped reads; `herdr-app/src/terminal/history_read.rs:188` joins only soft-wrapped rows. Hard breaks inserted by an agent TUI cannot be recovered as paragraphs from this snapshot. For genuinely responsive terminal layout, add an upstream absolute viewport/PTY-size capability and a single size owner/lease, measure cell dimensions, issue the resize, await acknowledgement and request a new versioned screen. Never silently resize shared Herdr splits independently from multiple cards/pages.

**Practice:** [xterm.js FitAddon](https://raw.githubusercontent.com/xtermjs/xterm.js/master/addons/addon-fit/src/FitAddon.ts) derives rows/columns from measured cells and available space; [ttyd](https://raw.githubusercontent.com/tsl0922/ttyd/main/html/src/components/terminal/xterm/index.ts) transmits terminal resize dimensions to its backend. [xterm.js reflow](https://raw.githubusercontent.com/xtermjs/xterm.js/master/src/common/buffer/BufferReflow.ts) preserves wrapped-line metadata and wide-cell boundaries. These are architectural examples, not an existing Deck `--cols` API.

**Verify:** At 320/480/640 px card widths, 80/160-column source terminals, long paths, ANSI, CJK, emoji, soft wraps and hard-break code: no sideways scrolling in prose mode, readable fixed font, correct selection/copy/search mapping and anchored scroll after resizing. Test the installed server's unwrapped output before relying on it; its live behavior was not verified here.

### D8 — Medium — Blocked status survives fresh nonblocked output

**Symptom:** A resolved trust/login prompt can remain “blocked” until a later metadata cycle; text heuristics can also flag quoted instructions.

**Cause / evidence:** `server.go:431` derives status from the cached screen before reading. At `:465` a new screen only sets blocked on a positive match; a negative match does not restore the underlying metadata status. On a nonmetadata cycle the underlying status is already the derived `blocked`. Executed `TestAstraBlockedRecovery`: idle metadata with trust footer, then fresh `ready` screen in a nonmetadata cycle; output `state=blocked blocked=true screen="ready"`. `model.go:139` scans text markers, not structured lifecycle events.

**Fix:** Keep reported agent status separate from screen-derived attention reason and timestamp. Recompute derived state from the latest successful screen, clear it on negative evidence, and show stale/unknown explicitly after read failure. Restrict heuristics to supported prompt regions; display inferred confidence/reason. Do not use a historical screen as definitive proof of current agent state.

**Verify:** Trust prompt → ready without a metadata refresh; quoted `/login` guidance; idle/working transitions; screen-read failure and recovery. Assert both label and send policy.

### D9 — Medium — Identical full snapshots fail to repair the screen version

**Symptom:** A reconnect/resync repeatedly rejects otherwise valid patches until the text changes.

**Cause / evidence:** `app.js:369` assigns `c.view.sv` only inside `p.screen !== c.view.lastText`. Same text with a new version leaves the old version installed. Executed Node probe using the actual ScreenView and the caller's condition: `serverVersion=9 clientVersion=1 nextPatchAccepted=false`.

**Fix:** Treat every full snapshot as authoritative for version/generation even when its text is unchanged. Clear queued patches, update version unconditionally, then skip only redundant parsing/painting. Reject stale responses to an outstanding resync by generation/request sequence.

**Verify:** Full snapshot of identical text at version 9 followed by patch 9→10 succeeds; repeated reconnects, reordered resync responses and session-generation changes do not loop or overwrite newer data.

### D10 — Medium — Client row trimming changes the server patch coordinate system

**Symptom:** A patch can silently replace the wrong row after a long full snapshot.

**Cause / evidence:** `screen.js:91` trims full screens to 2,000 rows; `:97` applies server indices directly to that shortened array. Executed probe: load 2,040 labeled rows, then replace server row 100; expected client index 60, actual index 100, replacing original row 140. `server.go:451` requests viewport rows plus 2,000; mock long-screen paths also exceed the client cap. The local upstream currently clamps recent history requests to 1,000 (`herdr-app/src/app/api_helpers.rs:116`), so reachability with this installed live server is **not verified**; the mock/protocol defect is reproduced.

**Fix:** Define one canonical retained row window in Go before diffing, with explicit base/generation and a maximum the client shares. Alternatively carry absolute row-base offsets through all patches. Validate start/delete bounds and version; request resync on invalid input instead of JavaScript splice coercion. Include ANSI starting state for the retained window if snapshots do not independently encode it.

**Verify:** Differentially apply snapshots/patch sequences around 1,999/2,000/2,040/2,500 rows and compare every retained row with the server. Include top deletion plus tail append, invalid bounds and missed versions.

### D11 — Medium — Reading older output jumps when top rows are dropped

**Symptom:** With follow disabled, the same scrollTop begins showing later content as history advances.

**Cause / evidence:** `screen.js:111` increases `base` and removes rows without adjusting `top`; the full-screen shift path at `:132` has the same omission. Executed probe: top=400, line height=20, drop ten rows; visible content moves from row 20 to row 30, while preserving row 20 requires top=200.

**Fix:** Capture the logical first visible row plus intra-row pixel offset; restore it after drops, edits and font/wrap changes. If the anchor was evicted, show that history was truncated. Add a persistent “Resume live” control and unread count instead of forcing follow.

**Verify:** Scroll up during streaming, top trims, font changes and D7 reflow; original content stays anchored. Resume-live alone moves to the tail. Exercise repeated identical rows where content matching is ambiguous.

### D12 — Medium — Task-to-agent association uses a global guess

**Symptom:** Two rapid launches can lose a task or associate it with another agent; a shell discovered before its agent never receives the association.

**Cause / evidence:** `guard.js:189` has one `pendingLaunch`; `onNewPane():195` matches a profile/name prefix. `app.js:83` calls it only when a previously absent pane is already an agent. A later agent transition on an existing shell does not qualify. Starting another task overwrites the pending record.

**Fix:** Return the created pane ID, launch ID and generation from the start operation/job. Track records by launch ID, reconcile both added panes and identity transitions, and expire with a visible failure. Remove name-prefix matching. Keep task associations server-side if they must survive page reloads.

**Verify:** Two same-profile launches finishing out of order, preexisting matching names, shell-first discovery, reload and launch failure. Each task attaches exactly once to its returned identity.

### D13 — Medium — Scope checks miss edits to already-dirty files

**Symptom:** Proof can say “No changes outside the declared files” after an out-of-scope edit to a file that was dirty before dispatch.

**Cause / evidence:** `guard.js:192` captures the baseline **after** sending and stores only paths; `:307–312` subtracts that set from current paths. A path in both sets disappears regardless of content change. Failed/missing baseline can still produce reassuring output. The backend already has content fingerprints in `guards.go:615` and `:651`, but this UI does not use them.

**Fix:** Capture `/api/scope/baseline` before dispatch and use `/api/scope/check` afterward. Display unknown when baseline acquisition fails; retain attribution caution when another writer may be responsible. Compare content/deletion fingerprints, not only path membership.

**Verify:** Modify an already-dirty out-of-scope file after baseline; delete it; rename it; leave preexisting dirt unchanged; fail baseline reading. Only verified unchanged content may be excluded.

### D14 — Medium — Card names remain width-constrained (screenshot c)

**Symptom:** Agent identity competes with status, metrics and controls; screenshot names are truncated to `clau...`, `wat...`, `cod...`.

**Cause / evidence:** `style.css:192–211` still uses a single-line title inside overflowing `.tt`, alongside fixed-size context/status elements; `app.js:305` adds identity controls to the same header. Current CSS gives the title more room and hides some chips at container breakpoints, so the exact screenshot truncation at its original width is **not reproduced on current code**. The source still cannot guarantee a readable identity on narrow cards.

**Fix:** Put full agent name and lifecycle in a dedicated first row, allow a second line, and move model/account/context details to a secondary row or accessible details popover. Use card container breakpoints, not just window width. Keep the complete identity available to keyboard and assistive technology.

**Verify:** Three cards at the screenshot viewport and 320 px cards, long names/accounts, large fonts and 200% zoom. Require a readable agent identifier without hover and no overlap with actions.

### D15 — Medium — The overflow menu is clipped by its toolbar

**Symptom:** The menu intended to expose secondary actions extends outside a clipped ancestor; narrow cards also hide key buttons.

**Cause / evidence:** `style.css:256` sets `.keys { overflow:hidden }`; `.more` is inside the key toolbar in `app.js`, and `.menu` at `style.css:269` is positioned above it. `z-index:30` cannot escape ancestor clipping, including the card's paint containment at `:176`. Container rules hide later `.key` buttons rather than moving them to an accessible overflow list. This is a CSS/DOM proof; current pixel bounds were not measured.

**Fix:** Render menus in a top-layer popover or body portal positioned from the trigger, with Escape/outside-click dismissal, focus return and viewport collision handling. Put all hidden key actions into that menu. Preserve card containment for rendering.

**Verify:** Open every menu at narrow card widths, bottom/right edges and zoomed layout; hit-test every item and keyboard through it. All hidden key actions remain reachable.

### D16 — Medium — Drawer refreshes erase edits and late responses populate another panel

**Symptom:** Queue-limit fields lose in-progress edits/focus; a slow tasks/usage response can append into a newly opened drawer.

**Cause / evidence:** `panels.js:59` clears the shared `#drawer-body` and passes that same node to async renderers. No generation or cancellation guards their continuation. `refreshOpen():70` repeats destructive rendering for live panels at 500 ms; `guard.js:71` rebuilds editable queue inputs from saved limits each time.

**Fix:** Give each open-panel instance a generation and AbortController; apply results only to its owned content node while current. Patch live fields in place, retain unsaved form state, and refresh data independently from the form's DOM. Cancel pending paint timers on close/change.

**Verify:** Delay tasks fetch, switch to usage, then finish tasks; no mixed content. Type unsaved limits during SSE traffic for several seconds; value, caret and focus remain unchanged.

### D17 — Medium — Job completion is lost across response races and reconnects

**Symptom:** The hub shows a completed job as running, or forgets its output after reload.

**Cause / evidence:** `jobs.go:55` can finish and emit `result` before the HTTP response is handled. `parity.js:429` updates only an existing local entry, created later at `:476`/`:498`. The page does not reconcile from the available jobs API on reconnect. `server.go:549` can replace queued SSE messages with a state snapshot that contains no job results; job events are not a durable ledger in the browser.

**Fix:** Upsert unknown result events by ID, merge pending responses without downgrading completed records, and fetch authoritative jobs on startup/reconnect/hub open. Include job revision/status in reconciliation and bound retained output/history.

**Verify:** Immediate completion before POST resolution; disconnect at completion; SSE overflow; reload; two pages. All report the same terminal status/output without duplicate execution.

### D18 — Medium — Safe-mode changes do not propagate to other open pages

**Symptom:** One page shows writes enabled while another has enabled global safe mode, or keeps disabled controls after it is turned off.

**Cause / evidence:** `server.go:189–195` mutates `a.readOnly` and `a.state.ReadOnly` but emits no event. `publish():533` overwrites the next state with the same value, and its normal diff has no read-only field. `app.js:111` changes UI mode only from full states. Server-side rejection is not sufficient UI synchronization.

**Fix:** Publish a dedicated versioned settings event or authoritative state immediately after toggling, outside the state mutex. Subscribe every page and clear/cancel pending raw/queued writes consistently. Keep backend enforcement independent of UI state.

**Verify:** Toggle safe mode in either of two pages while streaming, idle, disconnected/reconnecting and while a raw queue has pending input. Both indicators and controls converge without a metadata/topology change.

### D19 — Medium — Disk-floor policy omits this project's drive

**Symptom:** A launch in `D:\WORK\...` can bypass the configured free-disk floor; unavailable disk data can render as NaN.

**Cause / evidence:** `platform_windows.go:131` samples only C:, G:, I:. `guards.go:324–335` checks the launch drive only if a matching row exists; absent D: is not rejected. `guard.js:56` independently checks only C:. `panels.js:259` coerces disk values for display without distinguishing unavailable values.

**Fix:** Query the volume containing the resolved launch/worktree path, plus explicitly configured monitored volumes. Require known capacity when a nonzero floor is requested. Return structured unavailable/error states and show them without numeric coercion. Use the backend's policy/result in the queue UI.

**Verify:** D: launch, mount-point paths, missing/removable volume, denied query, zero floor and insufficient space. Mock volume APIs; do not fill a real disk.

### D20 — Medium — Restarting Deck changes its origin and invalidates reconnect/settings

**Symptom:** An old page cannot automatically reconnect to a restarted Deck; origin-scoped preferences may appear reset.

**Cause / evidence:** `run.go:111–122` generates a new token and binds `127.0.0.1:0` each run. `api.js` captures the page's token/origin for fetch/EventSource. Reconnecting to the old port cannot discover a different server port/token. `app.js`/panels use localStorage, which is scoped to that changing origin. Reconnecting to a temporary dropped SSE connection is a different case.

**Fix:** Choose and document a restart contract: a stable per-session loopback origin with authenticated restart handshake and persisted preferences, or explicitly detect permanent disconnection and instruct reopening Deck while storing preferences outside the random origin. Preserve token security; do not place a replacement token in an unauthenticated discovery endpoint. On restart use a new server generation and revalidate all pending writes.

**Verify:** Restart Deck at a new port, restart on a stable configured port, expire a token, and restart Herdr independently. Distinguish each state, preserve preferences by design, and never replay uncertain writes.

### D21 — Medium — Display clipping corrupts subsequent ANSI color state

**Symptom:** A long clipped line can color otherwise normal later rows.

**Cause / evidence:** `ansi.js:65–70` truncates raw text at 6,000 characters before processing trailing SGR resets. Executed probe: red SGR + 6,100 `x` characters + reset, then `normal`; output is `<span class="f1">normal</span>` instead of `normal`.

**Fix:** Separate visible-character clipping from state parsing: scan all bounded input for SGR state while emitting only the allowed display prefix. If an upstream byte limit prevents a complete parse, explicitly reset/mark state unknown at the boundary instead of leaking the previous color. Clip on grapheme/cell boundaries rather than UTF-16 halves.

**Verify:** Trailing resets beyond the display limit, long OSC/SGR sequences, Unicode at the boundary, and a following plain row; colors must match an unclipped parse while emitted DOM stays bounded.

### D22 — Low — Saving a prompt as a snippet opens an empty text field

**Symptom:** The save-snippet dialog does not initialize its textarea from the current draft.

**Cause / evidence:** `panels.js:413` passes a string `value` to `el('textarea', ...)`. `ui.js:9` assigns properties only for nonstrings, so `:10` sets a `value` attribute instead. A textarea's current value is its property, not a `value` content attribute ([platform contract](https://developer.mozilla.org/en-US/docs/Web/API/HTMLTextAreaElement/value)). Source proof; live dialog not run.

**Fix:** Assign `.value` for textarea/input/select form values explicitly in the helper, without treating arbitrary attributes as properties. Preserve deliberate boolean/ARIA handling.

**Verify:** Open Save snippet with a multiline draft and special characters, save it, clear the composer and reinsert. Require exact round trip and no draft loss on cancel.

### D23 — Low — Global/sticky regex searches skip matching lines

**Symptom:** `/foo/g` misses alternating identical matching rows.

**Cause / evidence:** `search.worker.js:14–16` reuses `RegExp.test()` across lines without resetting `lastIndex`. Actual worker run in Node VM: four `foo` lines, expected four hits, returned two.

**Fix:** Reset `lastIndex` before each independent line test, or reject/remove stateful flags with a documented search contract. Add worker request deadlines/replacement for pathological user regexes so search can recover from an unresponsive worker.

**Verify:** `/foo/`, `/foo/g`, sticky expressions, empty matches, multiline text and a deliberately expensive expression. Results stay complete and a later search remains usable.

### D24 — Low — Read-only timeouts are described as uncertain delivery

**Symptom:** A failed Git/read request can tell the owner that a prompt may have been delivered.

**Cause / evidence:** `guards.go:231–245` classifies generic timeout/cancellation strings as `delivery_unknown`, without operation type or evidence that dispatch began. Executed `TestAstraReadTimeoutClassification` with `git.exe timed out or cancelled: context deadline exceeded` returned `kind=delivery_unknown`. The reason any particular earlier Git subprocess timed out is not established.

**Fix:** Use typed operation errors: read timeout, mutation not started, mutation rejected, and mutation started but acknowledgement unknown. Only the last category should produce an uncertain-delivery warning/dispatch ID; do not infer delivery from text matching.

**Verify:** Feed identical timeout causes through Git, snapshot, preflight and actual prompt-write paths; only a started write can be delivery-unknown.

### D25 — Low — Unknown sign-in state is presented as signed out

**Symptom:** Unsupported/unavailable account detection displays “NOT signed in” and blocked styling.

**Cause / evidence:** `panels.js:316–318` uses a boolean truthiness ternary for `signed_in`, collapsing null/unknown with false. This can contradict the backend's explanatory detail.

**Fix:** Render explicit signed-in/signed-out/unknown states and preserve the provider's reason. Offer sign-in actions without claiming an unverified diagnosis.

**Verify:** True, false, null, missing field and lookup error fixtures show distinct truthful states.

## Performance

### P1 — High — Real screen diffs rebuild navigation and split controls

**Symptom:** Streaming updates do structural work; divider nodes can be removed during a drag, and rail focus can be lost.

**Cause / evidence:** `diff.go:54` embeds a full Pane in each update, including workspace/tab fields. `app.js:84` interprets their **presence**, not a value change, as topology change. `flush():58` calls `renderRail()`/`renderMain()`; `renderRail():218` replaces navigation, and `renderMain():286–289` resets layout and removes/recreates dividers. `mock.js` patch-only traffic does not model all of these real update fields, so mock-only FPS cannot validate the live path.

**Fix:** Compare topology before merging; separate data/status, navigation and layout dirty flags. Patch keyed rail entries and preserve divider nodes/pointer capture unless topology actually changes. Send only changed metadata or a separately versioned topology event. Keep rAF coalescing, but make each frame perform only necessary work.

**Gain:** Expected elimination of navigation/divider reconstruction on every ordinary screen update; no measured FPS multiplier is claimed. **Verify:** Replay actual-shaped PaneUpdate JSON at controlled rates, count mutations and render calls, drag continuously under traffic and record p50/p95/p99 frame work/long tasks. [Browser guidance](https://web.dev/articles/avoid-large-complex-layouts-and-layout-thrashing) supports reducing DOM/layout work and batching reads before writes.

### P2 — High — Serial polling and health queries compete for one socket lock

**Symptom:** One slow pane can delay all others and user actions; slow health requests can overlap indefinitely.

**Cause / evidence:** `server.go:366–468` shares one seven-second refresh context across serial snapshot, redundant agent-list and every due screen read. Failed reads do not advance `lastRead`, so they become due again at the 280 ms scheduler tick. “Fast” selection is Herdr focus, not necessarily the card the owner is using. `socket_windows.go:36` waits on a mutex before constructing a per-call timeout; waiting cannot be cancelled. It sets an I/O deadline but does not promptly react to parent cancellation without an earlier deadline. `extensions.go:245–310` health synchronously queries every pane through that same bridge; `panels.js:284` starts another health request every ten seconds without an in-flight guard.

**Fix:** Add a bounded, cancellation-aware scheduler with per-request budgets, fair round-robin pane reads, action priority and failure backoff. Keep metadata/recovery independent of screen interest. Use snapshot agents unless a separately scheduled diagnostic comparison is needed. Cache system health separately from expensive per-pane process memory; sample the latter lazily/infrequently with a total budget. Schedule the next browser health request after completion, abort on hide/close and share cached samples across pages. Preserve protocol-22 one-response connections; do not “fix” this by incorrectly reusing closed connections or spawning unlimited goroutines.

**Gain:** Expected bounded action latency and removal of redundant/overlapping work; current end-to-end latency not measured. **Verify:** Slow first pane, unreachable pipe, 1/10/50 panes and two pages; assert later panes still update, queue lengths remain bounded, cancelled requests leave promptly and action latency has an explicit upper bound.

### P3 — Medium — Row-diff detection rescans and reallocates full snapshots

**Symptom:** Small edits still allocate/split whole retained screens; repetitive output can trigger avoidable shift searches and two-patch traffic.

**Cause / evidence:** `diff.go:30` splits both strings, scans up to 256 candidate shifts, then fallback `rowDiff():13` splits both again. Worst-case comparison work is proportional to candidates times retained rows; that is a source-derived bound, not measured user latency. Repeated equal rows can also satisfy the shift heuristic even when only the last row changed, producing a top-drop plus tail-update and aggravating D11. `server.go:490` also deep-compares full panes before diffing.

**Fix:** Pass pre-split retained row slices into one diff function; prefer cheap common prefix/suffix checks for small edits, then a bounded shift detector using validated row hashes/anchors for genuine scroll. Select the smaller patch representation, with a full-snapshot threshold. Keep sequence/version validation independent of optimization.

**Gain / measurement:** On this PC, GOMAXPROCS=2, 200 ms microbenchmarks: repeated-row tail change **153,825 ns/op, 65,648 B/op, 3 allocs/op**; 20-row scroll of 2,000 labeled rows **130,204 ns/op, 65,648 B/op, 3 allocs/op**. Existing `BenchmarkRowDiff2000Lines` returned 168,773 ns/op, 65,536 B/op, 2 allocs/op. These are different workloads, not an old/new speed comparison. The measured diff times alone do **not** explain a slow dashboard; prioritize P1/P2. Expect reduced allocations and bounded pathological work, with no claimed browser FPS gain. **Verify:** Add near-matching prefixes, ANSI-heavy lines and complete replacement; apply every output patch and compare full arrays. Track bytes/op, allocs/op, ns/op and serialized SSE bytes.

### P4 — Medium — Hidden pages retain incoming work and old-pane state

**Symptom:** A background page can accumulate patch work while another page keeps streaming; many pane lifecycles retain history/guard records beyond their usefulness.

**Cause / evidence:** `app.js:82` appends patches until the render loop drains them; rAF can pause when hidden while EventSource remains active. There is no patch count/byte budget. `renderMain():275` creates card views for all panes before filtering; `removePane():88` does not clear every per-pane history/guard map. `guard.js:190`/`:205` retain launch/uncertain records; `parity.js:428` retains local jobs without a cap. Fixed per-screen row limits do not bound these structures. The hidden-page timing follows the [Page Visibility API](https://developer.mozilla.org/en-US/docs/Web/API/Page_Visibility_API), not a local hidden-browser measurement.

**Fix:** With D3's per-client visibility, stop sending unwanted screen payloads to hidden clients or discard/coalesce them and fetch one authoritative snapshot on return. Impose count/byte limits on queued patches. Use actual viewport intersection, not only filter membership, to prioritize screens. Instantiate views lazily; clean pane-scoped maps/workers/observers when identities disappear, and bound intentional retained history/jobs by age and size. Apply `content-visibility:auto` only to appropriate offscreen card shells, with intrinsic sizing and virtualizer-aware visibility; it does not stop JS parsing by itself ([browser guidance](https://web.dev/articles/content-visibility)).

**Gain:** Expected bounded memory and near-zero hidden-page parsing/painting; no measured memory saving is claimed. **Verify:** One active/one hidden page for 30 minutes, thousands of pane open/close events, filter changes and reconnect. Record heap/retained DOM/queue bytes and ensure return-to-page renders a correct fresh snapshot without replaying a backlog.

### P5 — Medium — Font measurement and frame phases still mix layout reads and writes

**Symptom:** Font or card-size changes can force extra style/layout work and use stale font metrics.

**Cause / evidence:** `screen.js:25` appends a span, immediately reads its bounding rect, and caches one global ratio. It does not invalidate after the actual font loads. `fit():70` runs from ResizeObserver and can change font/row size again. In `paintSize():168`, each view may read computed style then write spacer height before the next view reads. Thus the comment promising a single read/write phase does not cover first paint/font changes. Existing row virtualization and containment must not be replaced by unbounded wrapped DOM.

**Fix:** Resolve/calculate font metrics once per actual font/size after `document.fonts.ready`, cache by font identity, then batch all dirty-view style reads before spacer/font writes and finally scroll/row writes. Avoid synchronous measurements inside per-card resize callbacks; accumulate work into the shared scheduler. With D7, use ResizeObserver measurements plus cached row heights and anchor correction rather than measuring every row every frame.

**Gain:** Expected fewer forced layouts and stable fit calculations during resize; steady-state gain is unmeasured. **Verify:** Delayed font load, theme/font switching, repeated divider drag and 20 cards. Use layout-event traces and observer-loop warnings, comparing identical workloads. [Read/write batching guidance](https://web.dev/articles/avoid-large-complex-layouts-and-layout-thrashing).

## Improvements

### I1 — High — Make performance and handle diagnosis reproducible

**Change / cause:** Parameterize `app/build.ps1` for output/cache/no-format verification and `app/tests/deck-ui.ps1:30` for session, executable and scratch root. The current scripts cannot be run unchanged within this review's astra-only/output-only scope. Add real-shaped SSE fixtures, two-page/offline cases and a single-process measurement run. Expose inexpensive counters for queued RPCs, poll duration/errors, bytes/events, rendered rows, queued patch bytes, live jobs and dropped/resync events.

**Why / source:** PERF.md's unexplained handle growth needs attribution, not an invented root cause. Sample warmed-up process handles/RSS/goroutines over repeated connect/disconnect, polling, actions and job completion; correlate the growing resource category with allocation stacks/ownership. Use the already guarded debug/pprof path and [Go diagnostics guidance](https://go.dev/doc/diagnostics). **Verify:** Repeated runs produce comparable CSV/JSON and finish with zero owned child processes; a sustained slope leads to an identified owner and regression test before calling a leak fixed.

### I2 — High — Obtain structured lifecycle and prompt boundaries upstream

**Change / cause:** Extend the Herdr-to-Deck contract with process incarnation, agent start/exit, command/input state, event timestamps and source/confidence. `input.go` and `model.go` currently infer critical behavior from terminal text; unsupported prompt formats cannot safely mean “empty input.” Use explicit ready/busy/attention/exited/offline presentation and display evidence age.

**Why / source:** [VS Code shell integration](https://code.visualstudio.com/docs/terminal/shell-integration) uses escape-sequence markers for prompt/command boundaries and completion/exit status. Adopt the structured-event principle with appropriate trust checks; these markers alone do not prove an agent's identity or authorize writes. **Verify:** Recorded fixtures from supported agents and shell lifecycles, unknown protocol versions, spoofed output, exit/restart and missed events; uncertainty never becomes silent shell dispatch.

### I3 — Medium — Separate the scan view from detailed terminal/artifact reading

**Change / cause:** In `app.js`/`panels.js`, make cards prioritize identity, lifecycle/attention reason, last activity age, delivery state and readable recent output. Use a focused full-width view for long terminal output and a separate artifact/read-answer view for report/code reading; retain per-pane draft and scroll position. Keep context/tokens secondary when unavailable or stale. This addresses D7/D14 without shrinking every card's text.

**Why / sources:** [Wave Terminal](https://github.com/wavetermdev/waveterm) separates terminal blocks from file/editor/preview tools; [sshx](https://github.com/ekzhang/sshx) provides a spatial overview of collaborative terminals. Borrow independent overview/detail navigation, not an unbounded canvas by default. **Verify:** At three-card screenshot dimensions, keyboard-only users can identify the agent, attention reason and pending/failed send, open detail and return without losing draft/position.

### I4 — Medium — Benchmark a terminal renderer only for a true terminal mode

**Change / cause:** Build an isolated comparison before replacing `screen.js`. Compare the repaired virtual DOM prose renderer with a single focused xterm.js terminal for real PTY streaming; do not mount a WebGL terminal in every card or feed snapshot ANSI repeatedly as though it were an incremental PTY stream. Move ANSI parsing to a Worker only if profiling after P1–P5 shows it dominates; retain versioned results and bound pending work.

**Why / sources:** [Wetty](https://github.com/butlerx/wetty) uses xterm.js for browser terminal functionality. [ttyd's terminal implementation](https://raw.githubusercontent.com/tsl0922/ttyd/main/html/src/components/terminal/xterm/index.ts) integrates fit/render addons and flow control tied to write completion. Borrow backpressure and an explicit resize/stream contract; Canvas/WebGL does not repair snapshots, targeting, or lifecycle. **Verify:** Compare input latency, p95 frame time, long tasks, CPU/RSS/GPU resources, Unicode, selection/copy, accessibility and context loss on the same constrained PC. Adopt only a measured win with a DOM/readable fallback.

## Earlier bugs B1-B20

These IDs in `tasks/deck-r3-codex-bugs.md` are initial verifier failures with later triage, not twenty established product causes. “Fixed” below means the identified **verifier defect** is corrected in current source; it does not certify a browser rerun.

- B1 — Fixed (verifier): current `verify-ui.ps1` derives the actual session/pane key instead of hardcoding `lab|...`; current UI runtime not verified.
- B2 — Not reproducible independently: null composer cascaded from B1; the separate real shell-dispatch bug remains → D1.
- B3 — Not reproducible independently: null button cascaded from B1; separate raw input failures → D2.
- B4 — Not reproducible independently: null card cascaded from B1; verifier now enables focus emulation (`verify-ui.ps1:108`); clipboard runtime not verified.
- B5 — Not reproducible independently: null rename button cascaded from B1; runtime not verified.
- B6 — Not reproducible independently: null zoom control cascaded from B1; width/reflow remains → D7.
- B7 — Not reproducible independently: null split control cascaded from B1; runtime not verified.
- B8 — Fixed (verifier): positional `pane layout <id>` changed to `pane layout --pane <id>`; drag behavior under streaming still needs P1 verification.
- B9 — Not reproducible as originally attributed: verifier now selects a workspace/tab for split layout; current divider lifecycle risk → P1; runtime not verified.
- B10 — Fixed (verifier): uses the actual session key; the earlier cross-session rejection was not a product defect.
- B11 — Fixed (verifier): async wrapper for top-level awaited dynamic import (`verify-ui.ps1:14`); folder actions runtime not verified.
- B12 — Fixed (verifier): same awaited-import correction; columns dialog runtime not verified.
- B13 — Not reproducible: original `Failed to fetch` does not prove a layout-save/build cause; current launch guard omission → D4.
- B14 — Not reproducible: original Herdr `ghostty error -1` lacks a proven Deck cause; live workspace creation not verified.
- B15 — Fixed (verifier): actual session key replaces the invalid key; generic read-timeout messaging remains → D24.
- B16 — Not reproducible: original board-title timeout has no proven product cause; independently proven drawer race → D16.
- B17 — Fixed (verifier): `Screen()` uses supported `--format text` (`verify-ui.ps1:56`); two-page safe-mode presentation remains → D18.
- B18 — Fixed (verifier): same `plain`→`text` correction; raw continuation needs D2 and unknown prompt formats need I2.
- B19 — Not reproducible independently: null card cascaded from B1; independently proven blocked-state recovery → D8.
- B20 — Fixed (verifier): awaited-import wrapper corrected; live close-finished behavior not verified.

## Order of work

1. **Dispatch and admission:** D1–D6, D18–D19, D24; primarily `server.go`, `extensions.go`, `guards.go`, `input.go`, `tools.go`, `parity.go`, `platform_windows.go`, plus API intent changes. Agree on identity, client visibility and admission contracts first. Centralize shared policy instead of patching each button. Serialize edits to these shared backend files.
2. **Screen correctness:** D7–D11, D21 and P3/P5; `screen.js`, `ansi.js`, `diff.go`, screen-read contract and screen CSS. Define retained-row/version/height semantics before implementing wrapped virtualization. Keep upstream PTY-size support a distinct capability with explicit ownership.
3. **Render scheduling and sampling:** P1/P2/P4 and D20; `app.js`, backend poll/socket/health paths and `run.go`. Finish backend changes from batch 1 before this batch touches the same files. Use actual-shaped SSE fixtures and independent metadata recovery.
4. **Task/job UI correctness:** D12/D13/D16/D17/D22/D23/D25; `guard.js`, `panels.js`, `parity.js`, `ui.js`, `search.worker.js`. Replace client guesses with the server contracts from batch 1. Do not concurrently rewrite the shared drawer code for unrelated panels.
5. **Layout and usability:** D14/D15 and I3; header/toolbar markup and `style.css`, after screen CSS stabilizes. Verify narrow cards, focus, zoom, draft preservation and all overflow actions.
6. **Evidence and architectural choices:** I1 starts with the first batch and gates every batch; I2 follows the agreed upstream contract. Evaluate I4 only after corrected DOM/scheduler measurements. Add regression tests for the behavior that fails on the pre-fix code, then repeat the single-process isolated workload; do not certify performance from mock FPS alone.

## Doubts

- **Required Go checks passed:** From `app/`, `D:\tools\go\go1.26.10\go\bin\go.exe vet ./...` returned exit 0 with no diagnostics; `go test ./... -count=1` returned exit 0: `ok herdrdeck/internal/deck 23.247s`; the root and tests/agentstub packages reported no test files. Before starting, a read-only process/RAM check found 4,803 MiB available and zero concurrent Go/build.ps1 processes. Scratch GOCACHE/GOTMPDIR/TEMP/TMP were used, with network module lookup disabled and GOMAXPROCS=2. No product build was run.
- **Five additional backend behaviors reproduced:** A scratch Go overlay added tests without adding a repository test file. `go test -overlay <scratch>/overlay.json ./internal/deck -run '^TestAstra' -v -count=1` returned exit 1 with **five intended failing assertions**: ShellPrompt (D1), EmptyVisibilityStopsMetadata (D3), BlockedRecovery (D8), RawSecondChunk (D2), ReadTimeoutClassification (D24). Outputs appear with their findings. These failures demonstrate gaps beyond the passing existing suite. All use fakes/in-memory handlers; none sends an owner-session command. No fix was implemented, so there is no pre-fix/post-fix test comparison to claim.
- **Microbenchmarks passed:** After a second precheck found 5,232 MiB available and zero competing build processes, probes and then benchmarks ran sequentially. `go test -overlay <scratch>/overlay.json ./internal/deck -run '^$' -bench 'Benchmark(Astra|RowDiff|ParseSnapshot)' -benchmem -benchtime=200ms` returned exit 0. P3 records row results; ParseSnapshot returned 7,613 ns/op, 1,288 B/op, 24 allocs/op. CPU: i7-1165G7, Windows amd64. Warm-cache, short-run CPU measurements are not live UI/RSS/handle measurements.
- **Working-tree comparison is qualified by concurrent external Git activity:** The saved starting `git status --short` contained the dirty round-3 tree. During this review those changes were committed/merged externally: log entries `61af9ab6` (09:12:23), `e2f52aaf` (09:13:44), `0dbcad5f` (09:14:55, 2026-10-29 +03:00). This review ran no mutating Git command and did not alter product files. The end status has the new report plus the two preexisting untracked entries `tasks/deck-r4-astra-review.md` and `tasks/sol-free-c-drive-audit-20261026/`. Set comparison shows **only this report newly added**; the old dirty entries disappeared as part of that external commit. Therefore literal byte-for-byte status preservation cannot be certified, and the external changes were left alone.
- **Cleanup verified:** The owned astra Herdr PID 2960 was absent at the final check; zero Go/compiler/linker/test processes remained. No Edge or HerdrDeck process was launched by this review. After checking the resolved absolute scratch path and rejecting a reparse-point root, `%TEMP%\astra-r4\` was recursively removed; `Test-Path` returned `False`. The isolated server is no longer running even though its stop RPC could not connect. Only the report remains from this review.
- **Browser/build not verified:** No browser was launched. `build.ps1` has no output parameter, changes Go files with `go fmt`, uses workspace caches and overwrites `app/HerdrDeck.exe`; it was skipped as requested. The allowed harness hardcodes `lab` and workspace output/runtime directories and uses that executable. It was not run or modified. Therefore current FPS, long-task counts, render milliseconds, actual `scrollWidth/clientWidth`, page RSS, pixel-level header/menu layout and screenshot reproduction are not measured.
- **Screenshot evidence:** Both supplied images were opened with the image viewer: `C:\Users\m.labib\Documents\PicPick\2026-10-28 17 31 51.png` (D7/D14) and `2026-10-28 17 40 25.png` (D1). They establish the owner's observed symptoms; they do not establish the exact current executable/source revision or replace a current UI run.
- **Isolated live test not verified:** A hidden server was attempted only with scratch APPDATA, session `astra`, and its explicit socket path. The snapshot failed with Windows named-pipe `Access is denied (os error 5)`. Cleanup attempted `--session astra server stop`; it had the same connection failure, and the owned server process was verified exited. No agent workload or owner-session operation was performed. CLI help/schema worked without connecting and established capabilities, not successful live resize/read behavior.
- **Executed JavaScript evidence:** `node %TEMP%\astra-r4\probe.mjs <repository>` imported actual `screen.js`/`ansi.js`, stubbed only ResizeObserver, exercised non-DOM ScreenView methods, and ran actual `search.worker.js` in a Node VM. Output: patch coordinate expected 60/actual 100; identical snapshot server 9/client 1/rejected next patch; anchor expected top 200/actual 400; regex expected 4 hits/actual 2; ANSI next row unexpectedly red. These are logic probes, not browser layout measurements.
- **No proven handle-leak cause:** Current source and historical PERF.md do not establish the growing handle category or owner. No current sustained live-process sample was available. Do I1 before claiming a socket, PowerShell or process-memory handle leak. Go race-detector execution and real multi-page/agent lifecycle tests were not run.
- **Scope of conclusions:** Exact code references name the working tree reviewed on 2026-10-29, which was initially uncommitted. Source-derived failure paths are called out as such; lack of a live reproduction is not presented as a measured failure. D10 is reproduced at a protocol/mock size whose installed-server reachability is unverified; D14 does not pretend the old screenshot was captured from today's CSS. No review can prove absence of every additional bug.
