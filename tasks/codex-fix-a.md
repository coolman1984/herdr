# Task: fix batch A - herdr-agents.ps1, claude-limits.ps1, check.ps1 (+ regression tests)

## 1. Role and goal
You are a senior PowerShell 7 engineer working for the lead (Claude). Fix the approved defects
below, each with the smallest correct change, and prove every fix with a regression test.
Why: the owner's delegation tool misreports results and can close the wrong pane. The lead
reviews every line of your diff; a fix without a test, or a change outside the list, is rejected.

## 2. Context you need
- Folder: your current folder, a git worktree on branch `fix/bug-audit`
  (`C:\Users\m.labib\.herdr\worktrees\HERDR\fix-bug-audit`).
- Read first: `CLAUDE.md`, `AGENTS.md`, `LESSONS.md`, and `tasks/audit-findings.md` (YOUR earlier
  audit - the evidence and proposed fixes are there; this spec overrides it where they differ).
- Style to copy: `herdr-agents.ps1` itself (short comments, `Invoke-Herdr`, `Fail`, `-LiteralPath`).
- Herdr CLI contract: `herdr-skill.md`.

## 3. Exact scope
- Change: `herdr-agents.ps1`, `claude-limits.ps1`, `check.ps1`, `HISTORY.md` (one new phase at the
  end), `LESSONS.md` (append #44 only), `docs/FILES.md` (add the tests file row).
- Create: `tests/test-batch-a.ps1`.
- Do NOT touch anything else (batch B comes later: herdr-spaces, herdr-stop, herdr-open-folder,
  Herdr.cmd, install*, launcher/*, skills/*, other docs). Do NOT run git commit/push/reset/stash.
- Same safety limits as your audit: never run live-mutating commands against the owner's session;
  no secrets; no other agents; no browser.

## 4. Design (decided by the lead - follow it)

### herdr-agents.ps1
- **M1 (lead) false DONE right after a prompt.** Observed today: `ask codex -TaskFile ...` printed
  `DONE: codex is done` seconds after sending, while Codex had just started; the following
  `agent read` failed with `agent_not_idle` ("cannot read 200 lines while codex is working").
  Fix in `Complete-Ask`: when the prompt result is Ok and status is `done`/`idle`, confirm it:
  poll `agent get` every 2 s for up to 10 s; if it ever shows `working`/`blocked`, call `agent wait`
  (remaining `$TimeoutSec`) and use that result (blocked -> the BLOCKED branch). Also, if the final
  `agent read` fails with error code `agent_not_idle`, do not report DONE: report
  `UNCONFIRMED: <name> is still working; read it later with: herdr-agents read <name>`, exit 0.
  Keep the existing stalled-prompt branch.
- **F11** `agent_blocked` -> BLOCKED / exit 3 with the visible screen (before the FAILED branch).
- **F12** the answer read must succeed for DONE; otherwise FAILED / exit 1 with the read error
  (except `agent_not_idle` -> M1 rule above).
- **F02** `Save-Run`: unique file name - if `<HHmmss>-<label>.md` exists, use `-2`, `-3`, ...
  (create with `New-Item -ItemType File` without `-Force` in a loop so a race cannot overwrite).
  `history` must still list them in time order.
- **M2 (lead) `-Worktree` must open as a split to the RIGHT of the caller's pane** (owner's rule:
  every agent beside the lead, never a new space). After `git worktree add`, when
  `HERDR_ENV=1` and `HERDR_PANE_ID` are set, use the same `pane split $env:HERDR_PANE_ID
  --direction $Direction --cwd $wtPath --no-focus` path as a normal start. Only when not inside
  a Herdr pane, keep `workspace create`. Update the header line 12 comment and the STARTED text
  (`space=new` only when a space was really created).
- **M3 (lead)** `start -Cwd <path>`: fail before creating anything if the folder does not exist:
  `Fail "Folder not found: <path>"`. Resolve it to a full path (`Resolve-Path -LiteralPath`).
- **M4 (lead) + F01 + F03 + F16 - pane ownership.** Records get two new fields: `server`
  (the socket identity: `$env:HERDR_SOCKET_PATH` if set, else `"default"`, normalised lower-case)
  and `agentKind`. Rules:
  - `start`: save the record only if this invocation created the pane (split/workspace) OR the
    agent start succeeded in a supplied `-Pane`. A failed start in a supplied pane saves nothing.
  - `stop <name>`: only records with `server` equal to the current identity (legacy records
    without `server` count as `"default"`). Find the live agent by NAME in `agent list`:
    if live, close ITS current `pane_id` (it may have moved - LESSONS #18) - but only if its kind
    equals the record's `agentKind` (legacy: skip the kind check); if the name is not live and
    the recorded pane exists with no agent in it, close it (the tool created it); if the recorded
    pane has a different agent, refuse: `Fail "Pane <id> now holds another agent; not closing it."`.
  - Remove the record only after the close succeeded or the pane is confirmed gone (F16).
  - `status`/`diff`/HUD readers must keep working with old and new records.
- **F13** alias names (`start next` / `2040`): the alias name is a base name - if taken, use the
  normal suffix rule (`claude-2040-2`, ...). A user-given `-Name` that is taken still fails.
- **F14** `start -Pane P`: read P's cwd with `pane get` and record that; if `-Cwd` was also given
  and differs, `Fail "-Cwd differs from pane <P>'s folder <cwd>."`.
- **F15** team: do not `TrimEnd('\')` the lead's cwd (keep `D:\`).
- **F29** name validation with `-cnotmatch` (both in start and team).

### claude-limits.ps1
- **F10** an account is NEXT-eligible only with a numeric `five_hour.used_percentage`
  (expired window -> 0 still counts as known). `{}` or only `seven_day` -> HasData for display,
  but not eligible; the table shows `?` for the unknown 5h value.
- **F24** keep percentages as `[double]` for decisions; round only for display (`{0:0}`).
- **F09** rename the column/label "cost today" -> "session cost (updated today)" in `usage`
  output and the docs line that defines it (docs/GUIDE.md is batch B - leave a note in your answer).

### check.ps1
- **F06** fail a codex profile whose args contain `-s`/`--sandbox` with value `danger-full-access`
  (both `-s X` and `--sandbox=X` forms).
- **F07** compare each key's own command block: parse every `[[keys.command]]` block of the master
  `config/keys.toml` and of the live config into key -> command (and type) maps; FAIL per key that
  is missing or different; FAIL on duplicate keys.
- **F08** check every backtick path in the FIRST cell of each FILES.md table row.

### Tests: `tests/test-batch-a.ps1`
- Plain pwsh, no Pester, same `Check "<name>" { <bool> }` style as below, prints `PASS n/m`, exits
  1 on any failure. Load functions from the scripts with the AST loader from your audit (do not
  dot-source whole scripts that run main code). Mock `Invoke-Herdr` / `Get-Date` / file commands.
- At least one test per item above (M1-M4, F01-F03, F06-F16, F24, F29), each named after its id,
  e.g. `Check "M1 done-then-working waits" { ... }`. Each test must FAIL on the old code - run the
  suite once against `git show HEAD:<file>` copies and report how many failed there.
```powershell
$script:pass = 0; $script:fail = 0
function Check([string]$Name, [scriptblock]$Test) {
    try { if (& $Test) { $script:pass++; return } } catch { }
    $script:fail++; Write-Output "FAIL $Name"
}
```

### HISTORY.md / LESSONS.md
- New phase `## Phase 25 - Audit batch A: delegation tool, limits, check (2026-10-23)` in the
  existing Symptom / Cause / Fix / Proof / Lesson style; one bullet per item id.
- LESSONS #44: `agent prompt --wait` can return `done` within seconds while the agent has just
  started working (seen with Codex 0.163, 2026-10-23); confirm with `agent get` before trusting it.

## 5. Steps
1. Read section 2 files. 2. Write the tests first, see them fail on the current code.
3. Fix item by item. 4. Run section 6. 5. Answer.

## 6. Definition of done
- [ ] `pwsh -NoProfile -File tests/test-batch-a.ps1` -> `PASS n/n`, exit 0.
- [ ] Same suite against the HEAD versions -> every test id fails at least once (report the count).
- [ ] All `.ps1` parse (Parser::ParseFile, 0 errors).
- [ ] `pwsh -NoProfile -File check.ps1` -> no NEW FAIL besides the 3 worktree artifacts you
      explained in the audit baseline.
- [ ] `git status --porcelain` shows only the files of section 3.

## 7. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Per item: <id> - <one line: what changed> - test <name>
Checks: <each check of section 6 and its result>
Not done / doubts: <list, or "none">
