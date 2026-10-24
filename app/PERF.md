# Herdr Deck performance evidence

## STOP HANDOFF — measured status at owner shutdown, 2026-10-24

**PARTIAL overall.** Build/vet/test pass now; 30 backend tests pass, 1 lab-only test skipped
after shutdown (it passed separately). Exe **8,573,440 bytes (8.18 MiB)**. No 60-fps claim.
Every endpoint and guard's DONE/PARTIAL/NOT STARTED state is in API.md's authoritative
STOP HANDOFF table. Do not treat implemented-but-unmeasured endpoints as performance passes.

| Budget / work | Status | Evidence |
|---|---|---|
| Exe <12 MB | DONE | Final build 8,573,440 bytes, build.ps1 exit 0 |
| Frontend <150 KB | PARTIAL / FAIL | Latest web files total 257,316 bytes; Opus owns reduction |
| Direct API reads without process spawns | DONE | Six lab reads, no CLI process; protocol 22 closes each ordinary RPC after one response |
| Diff engine + bounded queues | DONE | Splice, scrolling two-patch reconstruction, latest-state resync and disconnect tests |
| Diff 2000 rows | DONE benchmark | 39,535 ns/op, 65,536 B/op, 2 allocs/op (initial single-splice benchmark) |
| Snapshot parser | DONE benchmark | 2,341 ns/op, 1,288 B/op, 24 allocs/op; tests/bench.txt |
| First complete 600-second soak | DONE measurement, FAIL RAM/traffic | 120 samples; RSS 50.352→53.117 MiB, slope +0.2788 MiB/min; CPU mean 0.570%, peak 2.482%; goroutines 7→7; handles +44; SSE 52,342.33 B/s total |
| Improved build, interrupted soak | PARTIAL | 32 samples through 160.06 s; RSS 13.664→15.215 MiB, max 15.312; CPU mean 1.660%, peak 3.787% of one core; goroutines 7→7; handles 149→169; SSE 1,809.52 B/s TOTAL for two streaming panes |
| Backend streaming CPU <8% | PARTIAL pass so far | Improved 160 s mean/peak below budget; full 600 s not completed |
| Backend RSS <40 MB | PARTIAL pass so far | Improved max 15.312 MiB; shorter run is not a ten-minute or one-hour proof |
| Idle CPU <1% | NOT STARTED separately | Measurements above include streaming; do not label them idle |
| Traffic <20 KB/s/streaming pane | PARTIAL pass so far | Improved combined 1.81 KB/s, below even one pane's budget; interrupted sample |
| Ten-minute final soak / flat handles | PARTIAL | Initial completed soak failed; improved run interrupted by explicit owner stop; handle growth still needs diagnosis/plateau check |
| One-hour RAM slope | NOT STARTED | Not verified |
| Page RSS <150 MB / one-hour stability | NOT STARTED by backend | Frontend evidence must be completed separately |
| Steady 60 fps, p95 <16.7 ms, zero long tasks | NOT STARTED by backend | No real-window or five-minute mock stress proof supplied here |
| Input <16 ms, click <100 ms | NOT STARTED by backend | Not verified |
| Paint <150 ms, interactive <400 ms | NOT STARTED by backend | Not verified |
| Hidden/minimized zero JS timers/fetches | PARTIAL | Backend visibility stops reads, 15-second SSE heartbeat; frontend runtime proof still required |
| Gzip big first snapshot / sync.Pool | NOT STARTED | No compressed snapshot path or pool implemented |
| Debug pprof flag | PARTIAL | Implemented off-by-default and authenticated; not used for diagnosis yet |

Changes between soaks: 24 MiB soft Go memory limit, GC 50%, and compact top-drop/tail
patches for scrolling screens. The previous 48 MiB/75% configuration missed the RSS budget.
Raw evidence: tests/soak-initial.csv and soak-initial-summary.json (full first run),
tests/soak.csv (interrupted improved run), tests/bench.txt, tests/evidence/lab-results.txt.
The stale tests/soak-summary.json still describes the INITIAL completed run; do not mistake
it for the interrupted improved run. Lab is stopped and the test Deck process is stopped.

**NEXT:** fix API R19–R24 mismatches and frontend size first; rebuild; rerun a complete
600-second final soak, separately sample idle CPU, inspect handle growth with pprof/Windows
process counters, then do the one-hour soak and real Edge/front-end budgets. Keep any missed
budget visible. Do not start agents, browsers or tests automatically from this handoff.

---

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

