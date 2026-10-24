# Handoff to Claude 2040 (from Claude 2030, whose 5-hour limit is at 99%)

You are Claude Code on the owner's second account ("2040"). You take over as LEAD for the
owner's Herdr toolkit. Work in `I:\herdr`.
**First read** `CLAUDE.md`, the last 3 phases of `HISTORY.md`, `LESSONS.md` #21, #24, #36-#39,
`docs\AGENT-ROSTER.md`, and the skill `herdr-team` (your `skills` folder has it). Reply to the
owner in English. PowerShell only. Commit with a HISTORY entry; run `pwsh -File check.ps1`.

## A. Finish HerdrLauncher.exe (written by GLM-5.3 Flash, reviewed by 2030)
State: `launcher\HerdrLauncher.cs`, `launcher\build.ps1`, `launcher\HerdrLauncher.exe` exist
(uncommitted). Lead's hidden tests: `I:\HERDR-trials\launcher-tests\test-launcher.ps1`
-> 21/21 pass in dry-run mode (first run found the real bug: paths ending in `\`, incl. `D:\`,
broke the wt command line; Flash fixed it with a Quote() helper).
1. Real launch test: `$env:LAUNCHER_REAL="1"; pwsh -File <that test script>` - it opens ONE real
   Windows Terminal window with Herdr (tell the owner first) and checks every 40 ms for 6 s that
   no console window appears. Report the result.
2. If it passes: point the App Path at the exe
   (`HKCU:\Software\Microsoft\Windows\CurrentVersion\App Paths\herdr.exe`, default value =
   `I:\herdr\launcher\HerdrLauncher.exe`; back up the old value
   `...\Herdr.cmd` in HISTORY) and the Desktop + Start-menu `Herdr.lnk` (Target = the exe, no
   args, keep the icon). Keep `Herdr.cmd` as a fallback.
3. Docs: `docs\FILES.md` (launcher\ rows), README "Start" table, GUIDE 13.1, HISTORY Phase 23
   (Flash wrote it; 2030's hidden tests 16/20 -> fix -> 21/21; Herdr reported Cline "idle" while
   it was still thinking - wait on its screen), LESSONS: #40 Windows quoting (`\"` = escaped
   quote; double trailing backslashes), #41 Herdr's state for Cline is unreliable.
4. Stop the Flash agent (`herdr-agents stop flash`).

## B. Usage limits + automatic switching (owner's new request)
The owner wants: see each account's limits/usage; when the account in use passes **95%** of
its 5-hour limit, switch to the available account with the LEAST 5-hour usage; and the main
Claude account is named **2030** (the second one is **2040**).
Facts found by 2030:
- Claude Code's status line input contains `rate_limits.five_hour.used_percentage`,
  `.resets_at` (unix seconds) and `seven_day` likewise. `~\.claude\statusline.js`
  (`shareCostWithHerdr`, owner-approved) now writes them to
  `%APPDATA%\herdr\claude-cost\<session_id>.json` with `config_dir` ("" = main account 2030,
  else `...\.herdr-accounts\claude\<name>`). Account 2040 uses the same statusline.js.
- At 16:41 today: 2030 = 5-hour 99% (resets 18:10), weekly 28% (resets Tue 27 Oct 10:00).
- Codex logs (`token_count.rate_limits.primary/secondary`) were null on this plan.
Build:
1. `agents.json`: alias `"2030"` -> profile claude, no account (the main sign-in), name
   `claude-2030`; and `"accountNames": { "claude": "2030" }` = display name of the default sign-in.
2. `herdr-agents usage`: for each Claude account (2030 + every folder in
   `%USERPROFILE%\.herdr-accounts\claude`) the newest file in claude-cost with that
   config_dir: 5-hour %, weekly %, reset times (local), cost, age of the data; mark accounts
   >= 95% "LIMIT"; print `NEXT: <account with the lowest 5-hour %>`. Accounts with no data yet:
   "no data (open it once)".
3. HUD (`herdr-hud.ps1`): for Claude panes add `5h N%` to the border title and a `$limit`
   token; show `account 2030` for the main sign-in; when an account crosses 95%, one Herdr
   notification (`herdr notification show "<account> at 96% - switch: Ctrl+B Shift+C" --sound request`).
4. Key/alias: `herdr-agents start 2030` works; GUIDE + skill (`herdr-team` 1b): "before a big
   task run `herdr-agents usage`; at >= 95% write a handoff file like this one and continue on
   NEXT". Update Claude's memory file `reference_claude_2040_account.md` to also name 2030.
5. Test, document (HISTORY Phase 24, LESSONS #42 rate_limits in the status line), check, commit.

## Rules
Never print or read secrets (.credentials.json, auth files). Tests never open windows except
the one real launch in A.1 (owner told first). Show the owner a short report at the end.

## Outcome (Claude 2040, 2026-10-22)
- A: done - dry-run 21/21, real launch 23/23 (no console, new Herdr client); App Path + both
  `Herdr.lnk` switched by `launcher\use-launcher.ps1` (backup, `-Undo`); docs; HISTORY 23,
  LESSONS #40-#41. Flash's pane was already closed.
- B: done - `claude-limits.ps1`, `herdr-agents usage` / `start next` / `start 2030` / `handoff`,
  HUD `5h N%` + 95% notification, `ctrl+b shift+c` = `start next`; third account `1984`.
  HISTORY 24, LESSONS #42-#43.
- Next handoffs: `herdr-agents handoff` + `tasks\HANDOFF-TEMPLATE.md` (more detailed than this file).