# Task: review AND fix the Yousef Slint backend (branch `backend`)

You are a senior Rust reviewer and fixer working for the lead (Claude). The backend was written by
several cheaper agents in turn (Codex Sol, Codex Terra, Cline MiniMax M3.1 Flash, Cline DeepSeek V4.1 Flash).
Your job: find what is wrong or missing, fix it, prove it with tests, and score the last writer (DeepSeek).

## Where
- Checkout: `I:\yousef-slint\.wt\backend` (git worktree, branch `backend`). Work and commit ONLY there
  (git commit allowed; no push, no merge, no other branch).
- Before every cargo command: `. I:\yousef-slint\env.ps1` (Rust paths for this shell only).
- The task the writers had: `I:\herdr\tasks\ys-backend.md` (read it fully; it is the spec you review against).
- Owner brief: `C:\Users\mflma\Downloads\yousef-slint-prompt.md`. Review findings to fix by design:
  `I:\herdr\tasks\yt-review-report.md`. Python source of truth, READ-ONLY: `I:\Yousef-Transportation`.
- The contract `src\api.rs` is the lead's: do not change it (additions go in `src\api_ext.rs`).
- Do not touch `src\main.rs`, `src\ui\`, `ui\`, `assets\`, `build.rs` (UI branch owns them).
- Follow `I:\herdr\AGENTS.md` (delegate rules). No browser tools. Do not start other agents.

## Steps
1. `git log --oneline main..HEAD`, `git status`, `cargo build`, `cargo test` - record the starting state.
2. Review against ys-backend.md "What to build" 1-6 and "Definition of done". For every item: done /
   partial / missing / wrong, with file:line. Look hard at: data loss, transactions (journal + table in one
   transaction), idempotent req_id, stale-preview double import, time comparisons in absolute instants,
   the no-image-egress test really intercepting EVERY request (a test that cannot fail is worthless:
   prove it fails when an image is sent), secrets never logged.
3. Fix by severity: wrong/dangerous first, then missing Definition-of-done items. Each fix with a test
   that fails without the fix (show it). Commit per fix.
4. Run the Definition of done: `cargo build`, `cargo test`, `cargo clippy` (if installed), the sample
   workbook `I:\Yousef-Transportation\tests\fixtures\sample_sep26_synthetic.xlsx` imports, a 130+ km/h pair
   is Warn and 250+ is Impossible, `backend_no_image_egress` passes.

## Answer format (short, plain words)
1. Starting state (build/test lines). 2. Findings by severity (file:line, what, fixed? + test name).
3. Final test summary line + clippy. 4. Per fix 01-15 (except 12, 14): status + test. 5. Still missing.
6. **Score for DeepSeek V4.1 Flash, 0-10**, with 3 reasons (what it did well/badly, honesty of its report).
7. Doubts (not verified).
