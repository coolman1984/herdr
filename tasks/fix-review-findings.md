# Task: fix the review findings in the Herdr toolkit (2026-10-24)

## 1. Role and goal
You are a senior PowerShell engineer working for the lead (Claude). Fix the defects an independent
Sonnet 5.5 review found, with tests, without changing behaviour that works.
Why: the owner said "fix all". Read `CLAUDE.md`, `AGENTS.md`, `LESSONS.md` first; copy the style of the
surrounding code; reuse `Invoke-Herdr`, `Fail`.

## 2. Findings (file:line are from the review; re-check them, lines may have moved)
MEDIUM
1. `check.ps1:67-92,112-116`: it compares key-command text but never tests that the script each key
   runs exists; it tests that `.lnk` files exist, not where they point. Add: each `[[keys.command]]`
   target script path must `Test-Path` (FAIL otherwise); each `Herdr.lnk`, `Herdr HUD.lnk`,
   `Stop Herdr.lnk` target must exist (WARN/FAIL as the neighbouring checks do).
2. `sync-skills.ps1:9`, `check.ps1:103-106`: only `~\.claude\skills` is synced/checked. Account copies
   under `%USERPROFILE%\.herdr-accounts\claude\<name>\skills` are stale (1984 still names `D:\WORK`;
   2040 has none). Make sync-skills sync every account folder that has (or should have, per
   `agents.json` account `copy` list) a skills folder, and make check/`-Check` report drift per account.
   Do NOT run the sync into account folders yourself (outside your sandbox) - the lead will.
3. `install-custom-build.ps1:19,36,44`: source path `$PSScriptRoot\herdr` is gone; the source is
   `herdr-app\` (HISTORY 34, docs/FILES.md:44). Point it at `herdr-app`, keep its checks. Also fix the
   wording in `CLAUDE.md:65` and `.gitignore:2` that still calls `herdr/` the upstream clone where it is
   now `herdr-app/` (keep `herdr/` ignored if it can still appear locally).
4. `herdr-open-folder.ps1:93,114-116`: prefix+t and prefix+shift+f start bare `claude` and `codex -m`
   and ignore `agents.json` args (no approval flags, contradicts Phase 18 / CLAUDE.md rule 3) and skip
   the pane record (`herdr-agents stop` later refuses). Make them start through the same profile logic
   as `herdr-agents start` (or call it), so args and pane record apply. Do not widen approval beyond
   what `agents.json` already says.
LOW
5. `herdr-agents.ps1:283,367-377`: account regex case-insensitive, name check case-sensitive; the
   account folder is created (286-293) before the name check fails. Validate first, create after.
6. `herdr-hud.ps1:227-241,369`: records are mapped to panes by pane id only, but records carry `server`
   (ids repeat between servers); and `catch { $ok = $false }` hides every error. Match on server too;
   log the error (to a file under `%APPDATA%\herdr\`) instead of silent.
7. `herdr-stop.ps1:29,41`: `server stop 2>&1 | Out-Null` drops result/exit code. Check and report it
   (read back that the server is really gone); bring the confirm dialog to front like
   `herdr-open-folder.ps1:32-33`.
8. `herdr-hud.ps1:349-355`: `-Once` ("one pass, printed") still sends `pane report-metadata` to live
   panes. With `-Once` print only.
9. `herdr-trial.ps1:18,83-97,149-152`: default root `I:\HERDR-trials\model-trial` does not exist;
   `Wait-TrialReport` waits the whole 60 min even if the agent is blocked/dead; a Fail after the agent
   started leaves its pane open. Make the root derive from the script folder or a clear error, stop
   waiting when the agent is blocked/gone, close the pane (only the one it opened) on Fail.
10. `herdr-agents.ps1:191-205` via `collect` (620-645): each already-done agent costs up to 10 s of
    polling for "working", one at a time. Poll all agents together or skip done ones.
11. `read-modify-write of herdr-agents-panes.json` (herdr-agents.ps1): an unreadable state file becomes
    `@()` and the next save drops ownership records; a bad `agents.json` makes the account name fall
    back to "main" instead of "2030". Fail with a clear message instead of falling back silently.
12-14. Also re-review the root scripts yourself for any further LOW defect of the same kinds (silent
    catch, wrong id mapping, missing read-back) and fix those you can prove. List each one.

## 3. Scope
- Change: the files named above plus their tests under `tests/`. New tests go in a new
  `tests/test-review-fixes.ps1` (offline, no live Herdr).
- Do NOT touch: `herdr-app/`, `~/.claude/settings.json`, accounts' folders, the live Herdr session
  (isolated `--session lab` only), approval settings beyond item 4. No git. Do not open panes.
- If your sandbox blocks a write outside `I:\herdr`, do not bypass it: list it for the lead.

## 4. Proof (AGENTS.md rule 4)
- New tests must FAIL on the old code: run them against `git show HEAD:<file>` copies in a temp folder
  and report how many fail there; they must pass on the new code.
- Run every existing suite under `tests/` and `pwsh -File check.ps1`; all must pass.
- Add a HISTORY phase (next number) and LESSONS entries for durable facts.

## 5. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Checks: <suite results; tests failing on old code: N of M>
Not done / doubts: <list or none>
