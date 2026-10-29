
## B1
Feature: Live grid and SSE
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: Timed out: !!document.querySelector("[data-key=\"lab|w2:p1\"]"); UI: 
Time: 2026-10-28T17:40:20.8487458+03:00


## B2
Feature: Prompt box and dispatch ledger
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Cannot set properties of null (setting 'value')
    at <anonymous>:1:86
    at <anonymous>:1:228
Time: 2026-10-28T17:40:20.8806608+03:00


## B3
Feature: Paste without Enter then key button
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Cannot read properties of null (reading 'click')
    at <anonymous>:1:97
Time: 2026-10-28T17:40:20.8952412+03:00


## B4
Feature: Copy screen clipboard read-back
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Cannot read properties of null (reading 'innerText')
    at <anonymous>:1:59
Time: 2026-10-28T17:40:20.9080880+03:00


## B5
Feature: Rename pane dialog and read-back
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Cannot read properties of null (reading 'click')
    at <anonymous>:1:92
Time: 2026-10-28T17:40:20.9179049+03:00


## B6
Feature: Zoom and grid columns
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Cannot read properties of null (reading 'click')
    at <anonymous>:1:56
Time: 2026-10-28T17:40:20.9296563+03:00


## B7
Feature: Split right and close confirmation
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Cannot read properties of null (reading 'click')
    at <anonymous>:1:91
Time: 2026-10-28T17:40:20.9555256+03:00


## B8
Feature: Drag swap via card drop
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: herdr --session lab pane layout w2:p1: unknown option: w2:p1
Time: 2026-10-28T17:40:22.6485481+03:00


## B9
Feature: Divider mouse resize
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: Timed out: !!document.querySelector(".divider.v"); UI: 
Time: 2026-10-28T17:40:42.7019521+03:00


## B10
Feature: Pane focus and Herdr zoom API
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: Error: 502 {"cause":"unknown unless stated in the observed error","error":"pane key belongs to another session or is invalid","kind":"agent_or_request_error","next_step":"Read the exact error and pane output; the cause is not verified."}
    at <anonymous>:1:287
Time: 2026-10-28T17:40:42.7243130+03:00


## B11
Feature: Folder picker and workspace rename focus close
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: SyntaxError: Unexpected token 'import'
Time: 2026-10-28T17:40:51.3922167+03:00


## B12
Feature: Equal columns dialog
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: SyntaxError: Unexpected token 'import'
Time: 2026-10-28T17:40:51.4988231+03:00


## B13
Feature: Layout save and shell-only build
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Failed to fetch
    at <anonymous>:1:64
Time: 2026-10-28T17:41:11.6281690+03:00


## B14
Feature: Spaces join and separate
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: herdr --session lab workspace create --cwd D:\WORK\Software Development\GitHub\HERDR --label R3 join A --no-focus: {"error":{"code":"workspace_create_failed","message":"ghostty error -1"},"id":"cli:workspace:create"}
Time: 2026-10-28T17:41:12.5696490+03:00


## B15
Feature: Git root and paths
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: Error: 400 {"cause":"unknown unless stated in the observed error","error":"pane key belongs to another session or is invalid","kind":"agent_or_request_error","next_step":"Read the exact error and pane output; the cause is not verified."}
    at <anonymous>:1:292
Time: 2026-10-28T17:41:12.7619573+03:00


## B16
Feature: Tasks docs help board and tools
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: Timed out: document.querySelector("#drawer-title").innerText.includes("board"); UI: 
Time: 2026-10-28T17:41:48.3071672+03:00


## B17
Feature: Safe mode rejects writes
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: herdr --session lab pane read w2:p1 --source visible --format plain: invalid read format: plain
Time: 2026-10-28T17:41:50.2973737+03:00


## B18
Feature: Typed-input guard toast and zero writes
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: herdr --session lab pane read w2:p1 --source visible --format plain: invalid read format: plain
Time: 2026-10-28T17:42:01.0574774+03:00


## B19
Feature: HUD status board and blocked inbox
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: TypeError: Cannot read properties of null (reading 'dataset')
    at <anonymous>:1:51
Time: 2026-10-28T17:42:05.4981371+03:00


## B20
Feature: Close finished panes
Severity: pending triage (verification failure)
Steps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.
Expected: behavioral assertion passes against isolated lab.
Observed: JS error: SyntaxError: Unexpected token 'import'
Time: 2026-10-28T17:42:23.8108835+03:00


## Initial-run triage (supersedes B1-B20 as product findings)
The first verifier run used an incorrect hard-coded session key (`lab` instead of the API's absolute socket identity), unsupported CLI `--format plain` and positional `pane layout`, and top-level await unsupported by this CDP evaluate call. These are VERIFIER defects, not established product bugs. Downstream missing-element failures are invalid evidence. Corrected verifier derives the session from state, uses text/--pane and async evaluation. All original entries retained for audit; rerun required. Paging-file exhaustion was also observed starting a diagnostic PowerShell process (0x800705AF); no cause attributed to Deck.

## B21
Feature: GET /api/git root/paths (live)
Severity: medium
Steps: corrected verify-ui.ps1, create shell in repository, request /api/git?key=<state-provided key>.
Expected: repository root and status paths within the handler budget.
Observed: HTTP 502: git.exe timed out or cancelled: context deadline exceeded. Body kind=delivery_unknown and next_step asks to inspect pane/delivery before resend, despite this being a read-only git request. 2026-10-28 17:46 lab run. Recheck once; no cause established.

## Harness triage update
Clipboard error Document is not focused is a headless focus prerequisite; Emulation.setFocusEmulationEnabled added for rerun. Divider test must select a workspace/tab before enabling Splits; All panes deliberately has no single layout. These are not confirmed product defects.
