# Task: `ask -Fresh` for Codex, and `herdr-trial.ps1` (automatic model trials)

## 1. Role and goal
You are a senior PowerShell 7 engineer working for the lead (Claude).
Goal: (A) `herdr-agents ask <codex-agent> -Fresh ...` gives Codex a fresh conversation before
the task; (B) a new script `herdr-trial.ps1` runs the lead's model trial for any agent profile
end-to-end and appends one result row. Both with offline regression tests.
Why: (A) the company gateway rejects long Codex conversations (LESSONS #45) - today the lead
had to type `/new`, answer Codex's "Where should the new conversation run?" menu and resend by
hand. (B) the lead compares models (docs/AGENT-ROSTER.md section 7) by hand; the owner wants
that repeatable so cheap models can take more work.

## 2. Context you need
- Folder: your current folder, git worktree on branch `round2`. Read first: `CLAUDE.md`,
  `AGENTS.md`, `LESSONS.md` (#21, #41, #44, #45, #46), `herdr-skill.md` (agent prompt /
  send-keys / read syntax), `herdr-agents.ps1` (style, `Invoke-Herdr`, `Fail`, `Complete-Ask`),
  `tests/test-batch-a.ps1` (copy its AST loader, `Check`, mocks, `Run-Branch`).
- Another agent (DeepSeek) works in this SAME folder on `herdr-hud.ps1`, `tests/test-hud-cline.ps1`
  and `docs/GUIDE.md`. Never touch those files.
- Trial kit (outside this repo, read-only for you): `D:\WORK\Software Development\GitHub\HERDR-trials\model-trial\`
  - `kit\` (the task folder copied per run), `grade.ps1 -Run <folder>` prints lines
    `LEDGER a/b`, `COLUMNS a/b`, `SCOPE ok|VIOLATED: ...`, `REVIEW.md present|MISSING`, `REPORT.md ...`.
  - The standard prompt (exact): `Read TASKS.md in your current folder completely and do every task in it. Follow its rules exactly. When finished, reply with the answer format from TASKS.md.`
  - Runs live in `runs\<run-name>\`. Do NOT run a real trial; never start agents yourself.
- Codex screen texts seen today (visible screen via `herdr agent read <name> --source visible`):
  after `/new` in a git worktree: `Where should the new conversation run?` /
  `› 1. Use current Git worktree` / `2. Create new Git worktree`; `Enter` (send-keys key name
  `Enter`) picks 1. Ready screen contains `Ask Codex to do anything`.
- Cline session files (for token counts): `%USERPROFILE%\.cline\data\sessions\<id>\<id>.json`
  with top-level `pid`, `cwd`, `model`, `status`, and `metadata.usage.inputTokens`,
  `.outputTokens`, `.cacheReadTokens` (fresh input = inputTokens - cacheReadTokens).

## 3. Exact scope
- Change: `herdr-agents.ps1` (only: param block, header comments, the `ask` branch, one new
  helper function), `docs/DELEGATION.md` (one short `-Fresh` paragraph), `docs/FILES.md` (rows for
  the new files), `skills/herdr-team/SKILL.md` (two table rows: `ask ... -Fresh`, `herdr-trial.ps1`).
- Create: `herdr-trial.ps1`, `tests/test-round2-agents.ps1`.
- Do NOT touch: `HISTORY.md`, `LESSONS.md` (the lead writes them), `herdr-hud.ps1`,
  `docs/GUIDE.md`, anything else. No git commit/push/reset/stash/checkout. No live-mutating herdr
  calls, no agents, no secrets (never open `auth.json` or credential files), no browser.

## 4. Design (decided by the lead - follow it)
### A. `ask -Fresh`
- New switch `[switch]$Fresh`. Valid only for `ask`; if the target's live `agent` kind is not
  `codex`: `Fail "-Fresh is for Codex agents only."`.
- New function `Start-FreshConversation([string]$AgentName)` -> `$true` or `Fail`:
  1. `agent prompt <name> "/new"` (no --wait).
  2. Poll `agent read <name> --source visible` every 1 s for up to 20 s:
     - screen contains `Where should the new conversation run?` -> `agent send-keys <name> Enter`
       (once), keep polling;
     - screen contains `Ask Codex to do anything` and NOT the menu text -> return `$true`.
  3. Timeout -> `Fail "Codex did not open a fresh conversation in 20 s; see: herdr-agents read <name> -Visible"`.
- In `ask`: when `-Fresh`, call it before the existing prompt; everything after is unchanged
  (Complete-Ask etc.). Header comment line: `herdr-agents ask codex -Fresh -TaskFile t.md   fresh
  conversation first (LESSONS #45)`.

### B. `herdr-trial.ps1`
- `herdr-trial.ps1 -Profile <name> [-Model <id>] [-Effort <lvl>] [-Run <name>]
  [-TrialRoot <path>] [-TimeoutMin 60] [-PollSec 15]`
  - `-TrialRoot` default `D:\WORK\Software Development\GitHub\HERDR-trials\model-trial`.
  - `-Run` default: `<profile>-<effort or 'default'>-<yyyyMMdd-HHmm>`; must match
    `^[a-z0-9][a-z0-9._-]{0,48}$`; if `runs\<Run>` exists -> `Fail "Run folder exists: ..."`.
  - Agent name: `trial-<n>` with the lowest free n (from `agent list`).
- Steps (functions, each testable): `New-TrialRun` (copy `kit` -> `runs\<Run>` with
  `Copy-Item -Recurse`), `Start-TrialAgent` (calls `pwsh -NoProfile -File <herdr-agents.ps1> start
  <Profile> -Name <n> -Cwd <run> [-Model] [-Effort]`; exit 2 (NOT READY) -> print the screen text,
  `Fail "Agent not ready - the owner answers its question in the pane, then rerun with -Run <existing>
  is NOT supported: stop it and start again."`), `Send-TrialPrompt` (`agent prompt <n> "<standard
  prompt>"`, no --wait), `Wait-TrialReport` (poll for `runs\<Run>\REPORT.md` every `-PollSec`; a
  file only counts when its size is stable over two polls; timeout -> outcome `TIMEOUT`),
  `Invoke-TrialGrade` (run `grade.ps1 -Run <folder>`, parse the four lines into fields),
  `Get-TrialTokens` (Cline only: newest session json whose `cwd` equals the run folder,
  trailing `\` and case ignored, under `%USERPROFILE%\.cline\data\sessions` and
  `%USERPROFILE%\.herdr-accounts\cline\*\data\sessions`; returns fresh_in / out; other kinds -> empty).
- Result row appended to `<TrialRoot>\results.csv` (create with header if missing; never rewrite
  existing rows): `date,run,profile,model,effort,outcome,ledger,columns,scope,review,report,minutes,fresh_in,out`
  (`outcome` = DONE | TIMEOUT; `minutes` = prompt -> stable REPORT.md, one decimal, invariant culture).
- Prints the row and the line `Review REVIEW.md yourself, then: herdr-agents stop <n>`. It does
  NOT stop the agent (the lead reads its review first).
- Style: same header comment style as other scripts; `$ErrorActionPreference = "Stop"`;
  `-LiteralPath` everywhere; absolute herdr path (LESSONS #1); no console windows.

### Tests: `tests/test-round2-agents.ps1`
Same style as batch A (AST loader, `Check`, `PASS n/m`, exit 1 on failure, all herdr/file calls
mocked, real files only under `$env:TEMP`). At least:
`A1 fresh sends /new then waits for ready`, `A2 fresh answers the worktree menu once`,
`A3 fresh times out with the read hint`, `A4 fresh refuses non-codex`,
`B1 run name validation and existing folder`, `B2 report counts only when size is stable`,
`B3 grade lines parsed`, `B4 cline tokens from session matching cwd`,
`B5 csv header once and rows appended`, `B6 timeout gives TIMEOUT row`.
Run the A tests against the `HEAD` copy of `herdr-agents.ps1` too (temp folder) and report
how many fail there (all A tests must).

## 5. Steps
1. Read section 2. 2. Write tests, see them fail. 3. Implement A, then B. 4. Section 6. 5. Answer.

## 6. Definition of done
- [ ] `pwsh -NoProfile -File tests/test-round2-agents.ps1` -> `PASS n/n`.
- [ ] `tests/test-batch-a.ps1` 48/48 and `tests/test-batch-b.ps1` 18/18 still pass.
- [ ] All `.ps1` parse; `pwsh -NoProfile -File check.ps1` -> only the known worktree FAILs.
- [ ] `git status --porcelain` shows only your section 3 files (DeepSeek's files may also appear - ignore them).

## 7. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Per item: <A or B part> - <one line> - test <names>
Checks: <each check of section 6 and its result>
Not done / doubts: <list, or "none">
