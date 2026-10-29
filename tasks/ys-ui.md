# Task: Yousef Slint - Slint UI, app shell and release

You are a senior Rust + Slint UI engineer working for the lead (Claude). A second agent (Codex) builds
the backend at the same time in another worktree. You never touch its files; it never touches yours.

## Why
The owner is replacing the Python/web office app of Trip Orders with a native desktop app that opens
in its own window, fast, like a professional program. Users are non-technical dispatchers.
Arabic first (RTL), English second.

## Where
- Your checkout: `I:\yousef-slint\.wt\ui` (git worktree, branch `ui`). Work and commit ONLY there.
- Before EVERY cargo command, in the same shell: `. I:\yousef-slint\env.ps1` (Rust paths for this shell only,
  CARGO_TARGET_DIR=<your checkout>\target). Do NOT set permanent environment variables, do not install
  anything, no admin rights, nothing on C:. Crates are pre-downloaded; if you need a new one, `cargo add` it
  (UI lines of Cargo.toml only) and say so.
- Source of truth, READ-ONLY: `I:\Yousef-Transportation` (`CLAUDE.md`, `LAUNCH_SCOPE.md`).
- The full owner brief: `C:\Users\mflma\Downloads\yousef-slint-prompt.md` (sections 2.1, 3 items 12/14 and
  "Also", 4). The review: `I:\herdr\tasks\yt-review-report.md`.
- Do not start other agents or subagents. No browser tools.

## Contract (do not change)
`src\api.rs` (trait `CoreApi`, types) is the lead's contract. Get the backend with
`yousef_slint::core::open_core(&yousef_slint::api::home_dir())` -> `Arc<dyn CoreApi>`. On your branch
`src\core.rs` is a stub returning empty data; for screenshots/demo write your OWN fake in `src\ui\fake.rs`
(fills sample trips incl. one 130+ km/h pair) selected by env `YOUSEF_FAKE=1`. Never edit `src\api.rs`,
`src\core.rs`, `src\lib.rs` or backend modules. Call CoreApi off the UI thread
(`std::thread` + `slint::invoke_from_event_loop`).

## Files you own
`build.rs`, `src\main.rs`, `src\ui\` (Rust glue: models, paging, fake), `src\i18n.rs`, `src\shell\`
(single instance, close, error dialog), `ui\` (app.slint, theme.slint, screens\*.slint), `assets\` (fonts,
icons, i18n json), `tools\i18n_to_json.*` if needed, `tests\ui_*.rs`, `.github\workflows\release.yml`,
`installer\` (optional), `README.md`, the UI lines of Cargo.toml.

## What to build
1. Design from the web app: `index.html`, `css\tokens.css`, `css\base.css`, `js\views\*.js`,
   `docs\ui-lab\*-1366.png` screenshots. Copy colors/spacing/radius/type scale into a Slint `Theme` global.
2. Fonts: copy from `I:\Yousef-Transportation\fonts\` to `assets\fonts\` with `fonts\LICENSES`; IBM Plex Sans
   Arabic default; Cairo/Kufi/Tajawal optional (Settings). Register with Slint (`import "..."` in .slint).
3. i18n: strings come from `js\i18n\ar.js` and `en.js` (JS objects). Convert to `assets\i18n\ar.json`/`en.json`
   at build time in build.rs (or a checked-in converter + test that the JSON matches the JS key set).
   Arabic default + RTL layout (Slint: mirror layouts / `layout-direction` as supported by the Slint version
   in Cargo.lock; check its docs in `.cargo-home\registry\src`). Language switch in Settings, no restart.
   Show `Msg { key, args }` from the backend through the same i18n.
4. MVP screens, in order: Overview, Trips (paged + filters via `TripQuery`, never all rows; speed Warn/
   Impossible badges), Excel start/import (file picker -> preview counts + rows -> confirm -> result;
   use a native open-file dialog: `rfd` crate if cached, else Win32 via windows-sys), Vehicles and Drivers
   (list + add/edit with req_id = new uuid per save attempt), Settings (language, font, LAN mode with a
   clear warning, data/backup folder display, sign-out). Other web screens appear in the menu as
   "قريبًا / next" (list them in README).
5. Shell: `#![windows_subsystem = "windows"]` (no console); build.rs adds icon + version info via
   `winresource` (icon: convert `I:\Yousef-Transportation\icons` or `assets` source to .ico; only when
   building for Windows). Single instance: named mutex + named pipe per Windows user; a second launch
   brings the existing window to front and exits 0. Close: run `CoreApi::shutdown()` (finishes in-flight
   saves) then quit. Startup error: native MessageBox (Arabic + English) and exit code non-zero.
6. Fix 12: sign-out/logout clears ALL cached UI state (models, filters, preview, version) - with a test.
   Fix 14: per-user non-admin: data in `%LOCALAPPDATA%\YousefSlint` (already in api::home_dir); optional
   Inno Setup script `installer\yousef-slint.iss` with `PrivilegesRequired=lowest` (TODO if not done).
7. `.github\workflows\release.yml`: windows-latest, toolchain `stable-x86_64-pc-windows-msvc`;
   `cargo test` -> `cargo build --release` -> upload exe artifact. Publish a GitHub Release ONLY on a pushed
   `v*` tag, with `RELEASE-PROOF.txt` (version, commit, SHA-256 of the exe). Do not push anything.

## Steps
Read -> theme + fonts + i18n -> shell -> screens one by one, `cargo build` after each -> tests ->
run the app with `YOUSEF_FAKE=1` and check it opens (launch it detached, confirm the process stays up and a
window exists via its process MainWindowTitle, then stop ONLY that process) -> commit on branch `ui`
(git commit only; no push, no merge).

## Definition of done (run and paste the summary lines)
- `cargo build` and `cargo test` pass in your checkout, no warnings that matter.
- The exe starts without a console, shows the window in Arabic RTL with the 5 MVP screens; a second launch
  does not open a second window; closing calls shutdown.
- Tests: i18n key-set test, logout-clears-state test, paging model test.

## Answer format (short)
Commits (hash + one line); test summary line; what each MVP screen does; what is TODO; "needs" (crates,
contract gaps for the lead); "doubts" (not verified).
