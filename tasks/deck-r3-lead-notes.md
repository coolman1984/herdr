# Deck round 3 - lead notes for the tester (newest last)

Codex: re-run `app/tests/verify-ui.ps1` after every "BUILD READY" line. Do not build while the lead builds.

- 2026-10-28 17:20 BUILD READY. Lead changes since the spec: the Deck window opens Edge with
  `--no-proxy-server` (cold first paint 3,384 ms -> 876 ms in the harness with the same flag; the harness
  takes `Start-DeckUi -EdgeArgs @('--no-proxy-server')` - use it for speed numbers). app.js flush order:
  screens before the top bar; top bar/status/banner write only on change. Stress 20 s: 60 fps, p95 render
  10 ms, 0 long tasks (was 46 fps, 17.9 ms, 15).
