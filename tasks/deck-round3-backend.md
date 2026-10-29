# Herdr Deck round 3 - BACKEND (Codex GPT-6.1-Sol medium)

You continue the Herdr Deck backend you built on 2026-10-24 on the owner's other PC. Same app, new PC.
Read first: `CLAUDE.md`, `AGENTS.md`, `tasks/herdr-deck-app.md` (main spec, still rules),
`tasks/herdr-deck-split.md` (ownership), then the STOP HANDOFF blocks at the top of `app/API.md`,
`app/README.md`, `app/PERF.md`, and `app/API-REQUESTS.md` (R17-R24).

## This PC is different (M-LABIB03)
- Repo: `D:\WORK\Software Development\GitHub\HERDR` (old docs say `I:\herdr` - that was the other PC).
- Go 1.26.10 portable: `D:\tools\go\go1.26.10\go\bin\go.exe`. Not on PATH. The lead built the current
  tree with it: `app\build.ps1` PASSES (fmt, vet, tests, exe 8.2 MiB).
- Disk is short: C: ~2.7 GB free, D: ~10 GB. Keep Go caches inside `app\tests\` (build.ps1 does) and
  delete large temp output you create. No downloads (company proxy blocks most sites; Go stdlib only).
- RAM is short: never more than one lab Herdr server and one Deck process at a time; stop both when done.

## You own (write only these)
`app/main.go`, `app/internal/**`, `app/build.ps1`, `app/tests/**`, `app/API.md`, `app/README.md`,
`app/PERF.md` (backend part), `docs/FILES.md`, and the final `HISTORY.md` / `LESSONS.md` entries.
The frontend (Claude Sonnet 5.5, pane next to you) owns `app/web/**`, `app/FEATURES.md`,
`app/API-REQUESTS.md`. Never edit its files; re-read `app/API-REQUESTS.md` before each work step and
answer its requests in `app/API.md`.

## Work, in this order
1. `build.ps1`: find Go in this order - `$env:DECK_GO`, the old FAWZY path, `D:\tools\go\go1.26.10\go\bin\go.exe`,
   then PATH. Clear error if none.
2. Your NEXT STEPS 2-3 from the API.md STOP HANDOFF: `/api/herdr/start` must target the named
   `--session` (not the default launcher); main-account alias normalization; next-account helper
   protection (the lead account is never handed to a helper; lead-account refusals = HTTP 409);
   R19 reservation JSON + release route + 409 conflict shape; R21 `ram_free_mb`; R22
   `GET /api/headless/routes` (empty verified list - never claim research works); R23 dispatch_id /
   text_sha / HTTP 504 for unknown delivery; R24 git `paths`/`root` fields; `/fonts` immutable cache + MIME.
3. **New guard (owner incident today):** the lead's prompt was appended to text the owner was typing in
   an agent's input box. Any Deck route that types into a pane (`/api/pane/prompt`, `/text`,
   `agents/ask`, broadcasts) must first read the pane and refuse with HTTP 409
   `{"error":"input_not_empty", ...}` when the agent's input line already holds text (Claude Code: the
   line after `❯` between the two `─` rules; Codex: its prompt line). Grey suggestion text is a known
   false-positive risk - find how Herdr's ANSI output marks it (dim / faint SGR 2 or 90) and ignore dim
   text. Test with real captured screens (fixtures) - both a typed box and a suggestion-only box.
4. Rebuild; lab-only verification (`app/tests/verify-lab.ps1`), `mutation.ps1`, then `soak.ps1` for a
   COMPLETE 600 s. Report numbers against `tasks/herdr-deck-performance.md` budgets.
5. Docs: API.md status table updated (DONE only with evidence), README, PERF, FILES.md; a HISTORY phase
   (next free number - read the end of HISTORY.md; fix the duplicate Phase 43 / lesson 66 numbering the
   handoff mentions) and LESSONS for durable facts. Run `pwsh -File check.ps1`; report each failing
   step and its cause (do not repair owner settings or the external App Path - just report).

## Rules
- Test writes ONLY in an isolated lab session (README "Lab isolation": APPDATA = `app\tests\runtime`).
  The owner's live Herdr session is read-only for you. Never start paid AI agents for tests.
- No git writes, no commits. The lead reviews and commits.
- AGENTS.md rules apply (tests fail on old code, prove every claim, no silent catch, no browser tools
  except headless Edge against the lab Deck for screenshots).
- Work through the list without stopping between items; stop only when BLOCKED or done.

## Answer
DONE | PARTIAL | BLOCKED
Changed: files
Checks: build, go test count, tests that fail on the old code (how many), mutation, lab verify, soak numbers, check.ps1
Not done / doubts: list
