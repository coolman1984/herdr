# Herdr Deck - ADDENDUM: performance, speed and memory (owner, 2026-10-24)

The owner: "fully optimize the performance and speed - super fast, a full 60 frames all the time, no
lag, full memory management." Binding for BACKEND (Codex) and FRONTEND (Opus), together with
`herdr-deck-app.md`, `herdr-deck-split.md`, `herdr-deck-parity.md`. Measure, don't guess: every budget
below needs a number in `app/PERF.md` (BACKEND owns the file; FRONTEND appends its section).

## Budgets (4 live panes, 2 of them streaming output, on this PC)
| What | Budget |
|---|---|
| Frame rate while typing, scrolling, streaming | steady 60 fps; no frame over 16.7 ms main-thread work; 0 long tasks (>50 ms) |
| Keystroke -> character visible in the prompt | < 16 ms; the page never waits for the network to echo input |
| Click/palette action -> visible response | < 100 ms (optimistic UI, then confirm) |
| First paint of the shell | < 150 ms from window open; interactive < 400 ms |
| Backend CPU, idle (nothing changing) | < 1 % of one core; no process spawn when nothing changed |
| Backend CPU, 4 panes streaming | < 8 % of one core |
| Backend RAM | < 40 MB steady; no growth over a 1-hour soak (RSS slope ~ 0) |
| Page RAM (the Edge app-mode renderer + the page) | < 150 MB with 4 panes; no growth over a 1-hour soak |
| Server -> page traffic | diffs only (changed lines), < 20 KB/s per streaming pane |
| exe size / front end size | < 12 MB / < 150 KB |
| Window hidden or minimised | 0 timers, 0 fetches, SSE kept alive but at most 1 heartbeat per 15 s |

## BACKEND (Codex) - speed and memory
- Replace polling-by-spawn with the cheapest correct source: read the Herdr API through ONE long-lived
  connection to its socket if the schema allows it (study `herdr api schema`; fall back to the CLI only
  where there is no socket call); batch calls (one snapshot instead of N pane reads); never spawn a
  process per pane per tick. Adaptive polling: fast (150-300 ms) for the focused pane only while its
  screen changes, slow (1-2 s) for visible panes, 5-10 s for hidden ones, stop when no client.
- Diff engine: keep the last screen per pane (ring buffer), send only changed rows/ranges with a
  version counter; coalesce bursts to at most one SSE message per pane per frame (16 ms); drop stale
  frames instead of queueing (latest wins); gzip big first snapshots only.
- Memory management: bounded everything - per-pane scrollback cap (2000 lines), bounded SSE client
  queues (drop-oldest), bounded caches with TTL/LRU, `sync.Pool` for byte buffers, no goroutine leaks
  (context cancel on client disconnect; test with `runtime.NumGoroutine`), set `GOGC`/`debug.SetGCPercent`
  and `debug.SetMemoryLimit(48<<20)`, free pane state when a pane is closed, `FreeOSMemory` after
  large exports. Atomic file writes; no unbounded logs (rotate at 1 MB).
- Prove it: `go test -bench` for the diff engine and parser (allocs/op reported), a soak test script
  (`app/tests/soak.ps1`: 4 lab panes streaming for 10 min minimum, samples RSS/handles/goroutines/CPU
  each 5 s into `app/tests/soak.csv`, prints slope); pprof endpoint behind a debug flag, off by default.

## FRONTEND (Opus) - 60 fps and no lag
- One `requestAnimationFrame` render loop; all DOM writes batched, reads separated (no layout thrash);
  update only changed text rows; use `textContent`/DocumentFragment, `content-visibility: auto`,
  `contain: layout paint style`, `will-change` only where measured; transforms/opacity for animation.
- Terminal view = virtualised rows (render only visible rows + a small overscan); fixed row height;
  scrolling is native, anchored to the bottom unless the user scrolled up; ANSI parsed once per changed
  row and cached; very long lines clipped with a "show more" affordance; no per-character elements.
- Input never blocks: typing is local-first, sends are fire-and-forget with a pending chip; debounce
  searches (web worker for the global screen search so typing stays at 60 fps); no sync XHR, no
  `alert`, no heavy JSON on the main thread (parse diffs incrementally, big payloads in a worker).
- Memory: cap each card buffer to the Herdr viewport + 2000 lines (drop the oldest); revoke object URLs;
  remove listeners on card removal (AbortController); no detached DOM; `WeakRef`/`FinalizationRegistry`
  only if measured; pause everything on `visibilitychange`; cap toast/history lists; avoid closures that
  capture whole screens.
- Measure in the real Edge app window: Performance panel / `PerformanceObserver` (long tasks, frame
  times, `performance.measureUserAgentSpecificMemory` or heap snapshots via CDP `--remote-debugging-port`
  on the LAB profile only), a built-in `?perf=1` overlay (fps, p95 frame time, long tasks, JS heap, DOM
  node count, SSE msgs/s) that is ALSO the evidence in `app/PERF.md`; synthetic stress: mock mode with
  8 panes x 20 lines/s for 5 minutes must hold 60 fps.
- Ship nothing heavy: no frameworks, no web fonts, SVG icons as one sprite, CSS under 40 KB.

## Done means (report the numbers)
Every budget row measured with its number in `app/PERF.md`; the 10-minute soak shows flat memory and
0 goroutine growth; mock stress holds 60 fps (p95 frame < 16.7 ms); anything not met is listed with the
measured value and the cause. Then the lead runs an independent performance audit.
