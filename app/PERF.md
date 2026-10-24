# Herdr Deck performance evidence

Backend owns this file; frontend appends its measured section. No unmeasured budget is a pass.

## Backend implementation

- Native public Herdr NDJSON API for reads; no per-tick/per-pane child processes.
- Protocol 22 closes each ordinary RPC after one response. A persistent multipurpose connection
  failed in the lab (`The pipe is being closed`); six direct one-response connections then passed
  the same six reads with zero CLI processes. Subscription output-match events are match-edge
  events, not a live terminal stream; they cannot replace screen reads.
- Adaptive 280 ms focused-changing / 1.4 s visible / 7 s background reads; metadata 2 s.
- Versioned row-splice diffs. Two-slot client queues resync to latest full state on overflow;
  max four clients. Screens cap at viewport+2000 rows and 512 KiB; jobs cap at 4 active/24 recent,
  256 KiB retained result each. Process stdout/stderr cap at 4/1 MiB.
- Hidden pages retain SSE, pause all fetch/poll work, heartbeat at most every 15 s.
- Go GC 75%, soft memory limit 48 MiB. All child trees are contained in kill-on-close Windows jobs.
- Token-protected pprof is available only with explicit `--debug` (off by default).

## Measurements (in progress)

| Budget | Observed | Result |
|---|---|---|
| Exe <12 MB | Initial build 7,580,672 bytes (7.23 MiB); final pending | Initial pass |
| Backend idle <1% one core | not verified yet | pending |
| Backend streaming <8% one core | not verified yet | pending |
| Backend RSS <40 MB | not verified yet | pending |
| Ten-minute RSS/handles/goroutines soak | not verified yet | pending |
| One-hour RSS slope ~0 | not verified | pending |
| Traffic <20 KB/s/streaming pane | not verified yet | pending |
| Diff/parser allocations and latency | benchmarks pending | pending |
| Frontend <150 KB | frontend measurement pending | pending |
| 60 fps, frame <16.7 ms, zero >50 ms tasks | frontend measurement pending | not verified |
| Input <16 ms / click <100 ms | frontend measurement pending | not verified |
| Paint <150 ms / interactive <400 ms | frontend measurement pending | not verified |
| Page RSS <150 MB, one-hour slope ~0 | frontend measurement pending | not verified |

## Frontend measurements — append below

