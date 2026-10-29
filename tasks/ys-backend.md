# Task: Yousef Slint - Rust BACKEND (data side)

> **Handover 3 (lead, 2026-10-29): you are Cline DeepSeek V4.1 Flash, the fourth agent on this task.**
> The agent before you (MiniMax) planned at length three times and its turn ended before it wrote code.
> DO NOT do that: work in small steps, one tool call at a time, short thoughts. Your FIRST action is
> `. I:\yousef-slint\env.ps1; cargo test 2>&1 | Select-Object -Last 40` in your checkout. Then fix errors,
> then finish the missing parts in this order: excel (+ spelling, car DB first, sample workbook import),
> store/core (CoreApi complete), the fix_NN tests, backup/restore, auth, config. Commit after each part passes.
> Where the contract lacks something (restore actor/scopes, backups, login), add it in `src\api_ext.rs`.
>
> **Handover 2:** GPT-6.1-Sol, then GPT-6.1-Terra worked here before (owner switched models). Their uncommitted files in your checkout
> are a STARTING POINT, not verified work: `src\time\`, `src\speed\`, `src\store\`, `src\backup\`,
> `src\excel\`, `src\gateway\`, `src\photos\`, `src\core.rs`, `src\lib.rs`, `tests\`, Cargo.toml changes.
> First run `cargo build` and `cargo test` to see where it stands, then review, keep what is right, finish
> or rewrite the rest. The lead reviews your work at the end and scores it, so prove every claim by running it.
>
> **Handover 1 (lead, 2026-10-29):** a first agent (GPT-6.1-Sol) started this task and was replaced.
> Its uncommitted files in your checkout are a STARTING POINT, not verified work: `src\time\`, `src\speed\`,
> `src\store\` - review them, keep what is right, finish or rewrite the rest. The Rust toolchain was broken
> during its run (missing std/cargo) and is FIXED now: `cargo build` passes on branch main. Your branch
> now has the lead's pinned crates (calamine 0.37, rust_xlsxwriter 0.102, rusqlite 0.41 bundled). Its extra
> crates (ring, zip 2.6, quick-xml 0.31) were removed: re-add with `cargo add` only if truly needed, and
> prefer the versions calamine already uses (zip 9.x, quick-xml 0.42) so one copy is built.
> Commit early and often on branch `backend`.

You are a senior Rust engineer working for the lead (Claude). A second agent builds the Slint UI at
the same time in another worktree. You never touch its files; it never touches yours.

## Why
The owner is replacing the Python/web office app of Trip Orders with a native Rust + Slint desktop
app. You port the data side, and you fix the 15 problems a review found, by design, each with a test.

## Where
- Your checkout: `I:\yousef-slint\.wt\backend` (git worktree, branch `backend`). Work and commit ONLY there.
- Before EVERY cargo command, in the same shell: `. I:\yousef-slint\env.ps1` (sets RUSTUP_HOME,
  CARGO_HOME=I:\yousef-slint\.cargo-home, TEMP=I:\yousef-slint\.tmp, CARGO_TARGET_DIR=<your checkout>\target,
  PATH to cargo + MinGW gcc). Crates are pre-downloaded; if a network fetch fails use `--offline`.
  If you need a crate that is not cached, STOP that part and list it under "needs" in your answer.
- Source of truth, READ-ONLY: `I:\Yousef-Transportation`. Read first: its `CLAUDE.md`, `LAUNCH_SCOPE.md`,
  and the review `I:\herdr\tasks\yt-review-report.md` (15 findings with file:line).
- The full owner brief: `C:\Users\mflma\Downloads\yousef-slint-prompt.md` (read sections 2.2-3).

## Contract (do not change)
`src\api.rs` is the lead's contract: types + trait `CoreApi`. Implement it as `core::Core` in `src\core.rs`
(replace the stub; keep `pub fn open_core(home: &Path) -> Result<Arc<dyn CoreApi>>`). If the contract is
truly missing something, add a NEW item in a new file `src\api_ext.rs` and say so; never edit api.rs.

## Files you own
`src\lib.rs`, `src\core.rs`, and new modules under `src\`: `time\` (Instant, Cairo DST, parsing),
`store\` (SQLite + append-only journal), `excel\` (calamine read, rust_xlsxwriter export, spelling,
number normalization, preview/import), `speed\`, `backup\`, `auth\`, `gateway\`, `photos\`, `config\`;
integration tests in `tests\backend_*.rs`; the `[dependencies]` backend lines of Cargo.toml (use `cargo add`).
NOT yours: `src\main.rs`, `src\ui\`, `ui\`, `assets\`, `build.rs`, `.github\`, `installer\`.

## What to build
1. Store: SQLite via rusqlite (bundled). Data dir = `api::home_dir()`. Journal is append-only and the source of
   truth; table writes and journal append in ONE transaction; atomic file writes (temp + rename).
   Soft delete only. One data lock (a second process gets a clear error). `shutdown()` stops new writes,
   waits for in-flight saves, releases the lock.
2. Excel start flow, ported from `server\excel_io.py`, `fleet_import.py`, `xlsx_read.py`, `spelling.py`:
   car DB first; one spelling for one thing; numbers normalized before save; preview -> confirm -> import.
   Mirror `tests\test_start_from_excel.py`, `test_excel_io.py`, `test_fleet_import.py` as Rust tests.
   Must import `I:\Yousef-Transportation\tests\fixtures\sample_sep26_synthetic.xlsx`.
3. Odometer speed check (`server\domain.py` ~243 MAX_AVG_KMH=130, `server\tripsvc.py` ~49 MAX_SPEED_KMH=250):
   avg = distance between two readings / real elapsed time (utc_ms). >130 Warn, >250 Impossible. Warn only.
4. Gateway client (port `server\gateway_client.py`): reqwest blocking + rustls, HTTPS only, event ids
   `ev-<uuid>`, ack/mailbox behaviour. URL and keys from local config only; never log or print a key.
   DO NOT call the live gateway; unit-test with a local fake transport.
5. Photos (HARD RULE): images only on local disk `<data>\photos\<photo-id>.jpg`. Outbound only
   {photo id, trip id, reading, time, SHA-256}. `trait PhotoTransfer` with a hash-only local stub.
   Route EVERY outbound request through one `Transport` trait, and add test `tests\backend_no_image_egress.rs`
   that captures every request of a full sync with photos and FAILS if any body/multipart part starts with
   JPEG (FF D8 FF), PNG (89 50 4E 47), WebP (RIFF....WEBP) bytes or has an `image/*` content type.
6. The 15 review fixes, each with a test named `fix_NN_<what>` (NN = finding number in the brief's section 3):
   1 restore/office-wide replace needs all categories; 2 deleted private photos stay private, "public" is
   explicit; 3 restore copies + verifies photo files before success; 4 committed = success, audit copies
   retried separately, idempotent req_id for create/import; 5 import keys rechecked inside the save
   transaction; 6 keep last N complete verified backups, prune only after a verified replacement;
   7 login re-reads account/password generation inside the session transaction; 8 127.0.0.1 by default,
   LAN explicit; 9 bounded xlsx rows/cols, iterate only non-empty rows (a 10,000,000-row sparse sheet must be
   rejected fast); 10 compare absolute instants (repeated Cairo hour 2026-10-29 23:50+03:00 -> 23:10+02:00
   is a valid 20-minute trip and does not overlap 23:10-23:20+03:00); 11 unique normalized plate/driver/place
   keys inside the lock, including duplicates within one batch; 13 "new PC" reset archives the real db files
   (+ -wal/-shm); 15 insights/reports computed once per snapshot, no N^2, trips paged in SQL.
   (12 logout cache and 14 installer belong to the UI agent.)

## Steps
Read -> design modules -> implement in small steps, `cargo test` after each -> fix -> commit on branch
`backend` with clear messages (git commit only; no push, no merge, no other branch).

## Definition of done (run and paste the summary lines)
- `cargo build` and `cargo test` pass in your checkout, no warnings that matter (`cargo clippy` if available).
- `fix_01`..`fix_15` tests (except 12, 14) and `backend_no_image_egress` pass; the sample workbook imports.
- A test proves a 130+ km/h pair is flagged Warn and a 250+ pair Impossible.

## Answer format (short)
Commits (hash + one line each); test summary line; per fix NN: test name + pass; what was ported vs TODO;
"needs" (crates/decisions); "doubts" (anything not verified).
