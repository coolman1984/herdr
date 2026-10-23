# Audit findings - 2026-10-23

Summary: 30 findings (S1: 5, S2: 18, S3: 7). Files read: CLAUDE.md, AGENTS.md, LESSONS.md, HISTORY.md, README.md, herdr-skill.md, herdr-agents.ps1, claude-limits.ps1, herdr-hud.ps1, herdr-layout.ps1, herdr-columns.ps1, herdr-spaces.ps1, herdr-open-folder.ps1, herdr-help.ps1, herdr-stop.ps1, config/keys.toml, config/settings.toml, agents.json, sync-skills.ps1, check.ps1, install.ps1, install-custom-build.ps1, Herdr.cmd, install.cmd, herdr-agents.cmd, launcher/HerdrLauncher.cs, launcher/HerdrLauncher.exe (PE header and reflected pure method), launcher/build.ps1, launcher/use-launcher.ps1, skills/herdr-team/SKILL.md, skills/herdr-workspace/SKILL.md, docs/FILES.md, docs/GUIDE.md, docs/SETUP.md, docs/DELEGATION.md, docs/AGENT-ROSTER.md, docs/CLINE-PASS.md, tasks/HANDOFF-TEMPLATE.md, tasks/herdr-launcher.md, tasks/herdr-launcher-fix1.md. Isolated session used: no (optional; chose static tracing and in-memory fake dependencies within the restricted sandbox; no server startup attempted).

Baseline: `pwsh -NoProfile -File check.ps1` exited 1 with exactly these FAIL/WARN lines:

```text
FAIL  skill DIFFERS herdr-team  (run sync-skills.ps1)
FAIL  skill DIFFERS herdr-workspace  (run sync-skills.ps1)
FAIL  docs/FILES.md lists 'herdr/' but it does not exist
WARN  this folder is not on the user PATH
3 FAIL
```

All four are worktree artifacts, not additional toolkit findings. `git ls-files --eol skills` reports `i/lf w/crlf` for both skills. Hashing the local UTF-8 text after replacing CRLF with LF gives exactly the installed hashes: herdr-team `128D55D495134828B919B71A56B7B1F4CDC9F9617D2590400C974B7A8545D82A`, herdr-workspace `D5056C1614266E79D4591B2F943B9EA27B3A81512A47F180F77C1D893EE8D171`. There is no content drift. The upstream clone is deliberately untracked (docs/FILES.md); this new worktree is not the installed folder on PATH. Do not sync/install from this audit to resolve these messages.

PROVEN below means the stated pure function, expression, or argument parsing result was executed, with fake dependencies where indicated. It does not mean a live pane was changed. No prohibited toolkit command was executed; no browser, agent worker, credentials, installs, or live mutations were used. The only Herdr discovery calls were its absolute executable path with `--help`, `agent`, `pane`, and the read-only calls made by the authorized check script. All root and launcher PowerShell scripts parsed without errors. Code line numbers refer to this worktree.

For reproducibility, the extracted-function tests used this loader in a fresh `pwsh -NoProfile`. Each fixture below is independent; mocked file commands never touched disk.

```powershell
function Import-AuditFunctions([string]$File, [string[]]$Names) {
    $ast = [System.Management.Automation.Language.Parser]::ParseFile(
        (Join-Path $PWD $File), [ref]$null, [ref]$null)
    foreach ($f in $ast.FindAll({ param($x)
        $x -is [System.Management.Automation.Language.FunctionDefinitionAst]
    }, $true)) {
        if ($f.Name -in $Names) {
            . ([scriptblock]::Create($f.Extent.Text))
            Set-Item -Path "Function:global:$($f.Name)" -Value (
                Get-Item "Function:$($f.Name)").ScriptBlock
        }
    }
}
```

## F01 [S1] [READ] herdr-agents.ps1:66 - Pane ownership records are shared across independent servers
- Input / situation: Session A records agent `coder` in `w1:p1`; session B also has `w1:p1`, occupied by an unrelated owner pane. Run `stop coder` in B. The shared state can also survive a server/session replacement.
- Expected: Close only a pane this tool started in the current server (CLAUDE.md safety rule 5). herdr-skill.md:91 explicitly says IDs and names are scoped to one server.
- Actual: B reads A's record and closes B's `w1:p1`. Running isolated tests with the toolkit can also replace the live session's ownership records by name.
- Evidence: Line 66: `$statePath = Join-Path $env:APPDATA "herdr\herdr-agents-panes.json"`. No socket/session identity is saved at 325-327. Lines 662-664: `$entry = $state | Where-Object { $_.name -eq $Target } | Select-Object -First 1`; `$r = Invoke-Herdr @("pane", "close", $entry.pane)`. The close is sent to the current socket, not the record's original server. `herdr-spaces.ps1:21` likewise shares its undo record between sessions.
- Proposed fix: Namespace state and side-by-side records by normalized server/socket identity, include that identity in each record, and refuse mismatches in `stop`/`Split-Spaces`. Update HUD/layout readers to use the same state location; do not trust legacy records without validation.
- Risk of the fix: Existing records need a safe migration or explicit re-registration; HUD/account/model lookup must stay aligned.

## F02 [S1] [PROVEN] herdr-agents.ps1:168 - Runs for the same label in one second overwrite earlier task/answer logs
- Input / situation: Two quick `collect` operations for already settled agents, or two callers saving the same agent's result, finish in the same second.
- Expected: Every task and answer remains in the run history (header:164 and HISTORY Phase 13).
- Actual: Both saves return the same filename; `Set-Content` replaces the first report.
- Evidence: `Save-Run` builds `"{0}-{1}.md" -f (Get-Date -Format "HHmmss"), $Label` at 168, then unconditionally writes it at 170. Executed fixture:
  ```powershell
  Import-AuditFunctions herdr-agents.ps1 @('Save-Run')
  $runsPath='C:\fake'; $global:saved=@{}
  function New-Item { param($ItemType,[switch]$Force,$Path) }
  function Get-Date { param($Format)
      if($Format -eq 'yyyy-MM-dd'){'2026-10-23'}
      elseif($Format -eq 'HHmmss'){'120000'}
      else{[datetime]'2026-10-23T12:00:00'}
  }
  function Set-Content { param($LiteralPath,$Value,$Encoding)
      $global:saved[$LiteralPath]=$Value
  }
  $p1=Save-Run coder first DONE first-answer
  $p2=Save-Run coder second DONE second-answer
  "samePath=$($p1 -eq $p2) files=$($saved.Count) firstAnswerKept=$([bool]($saved.Values -match 'first-answer'))"
  ```
  Output: `samePath=True files=1 firstAnswerKept=False`.
- Proposed fix: In `Save-Run`, append a unique suffix and create the file exclusively; retain chronological naming for `history`.
- Risk of the fix: History sort order and consumers that assume the exact filename pattern.

## F03 [S1] [READ] herdr-agents.ps1:324 - Rejected starts can register an owner's existing pane for later closure
- Input / situation: `start codex -Pane <occupied-pane> -Name coder` fails because that pane already contains an owner process/agent. Alternatively, the started agent exits, and the owner uses its still-open shell pane for other work before `stop coder`.
- Expected: A failed attempt must not grant ownership of a pre-existing pane; `stop` must not close a replacement occupant (CLAUDE.md safety rule 5).
- Actual: The failed start is recorded unconditionally; later `stop coder` closes that pane without checking its live occupant. Stale successful-start records have the same replacement-occupant problem.
- Evidence: `agent start` is invoked at 322. Lines 325-328 append `{ name = $Name; pane = $pane; ... }` and `Save-State $state` BEFORE `if (-not $started.Ok)` at 330. `stop` at 662-664 checks only the recorded name and sends `pane close`. herdr-skill.md:55,125,137 requires an available shell and distinguishes startup-not-ready from successful startup; line 57 says the identity clears on exit/replacement.
- Proposed fix: Record whether this invocation actually created the pane. For a supplied pane, record only a verified matching agent identity (including NOT READY cases with a retained identity). In `stop`, verify the current occupant still matches; otherwise refuse instead of closing unrelated work.
- Risk of the fix: Deliberately keeping a record of a sign-in/trust-blocked agent must continue to work; empty panes created by the tool still need an explicit cleanup path.

## F04 [S1] [READ] install.ps1:167 - Updating the user's PATH overwrites a registry value without the required backup
- Input / situation: Run an install/update with an existing user PATH whose order/Herdr entries need changing.
- Expected: Back up registry values before overwriting anything outside this folder and report the backup location (CLAUDE.md safety rule 6).
- Actual: The old PATH value and registry kind are read, then the value is overwritten with no durable backup or backup location.
- Evidence: `Update-PathRegistryEntry`: `$value = $EnvironmentKey.GetValue("Path", $null, $options)` (152), `$EnvironmentKey.SetValue("Path", $newValue, $kind)` (167). Its caller at 885-896 creates/disposes the Environment key but does not export/save the previous value. No other installer path backs it up.
- Proposed fix: Before `SetValue`, save the original value, type, and whether it existed to a uniquely named backup under the Herdr backup directory, then print that location.
- Risk of the fix: Backup failure should stop the registry mutation; preserve expandable-string values without expanding them.

## F05 [S1] [READ] skills/herdr-workspace/SKILL.md:26 - The maintenance skill still directs tests into the owner's live session
- Input / situation: An agent follows the skill's "Live tests" procedure while working in the owner's normal session.
- Expected: Headless isolated-session testing, with its own socket and explicit teardown (CLAUDE.md engineering rule 3; HISTORY Phase 13; LESSONS #20 and #24).
- Actual: The procedure creates a workspace on the inherited live socket, changes caller pane variables, runs tests there, and closes it. No isolated server/socket is established.
- Evidence: Lines 26-28: `Live tests: create a workspace with workspace create --no-focus --label <test>, set HERDR_ENV=1 and HERDR_PANE_ID to its pane, test, then workspace close it.` HISTORY Phase 13 explicitly replaced live testing after a live server crash. This is not an exception for a feature that isolation cannot show.
- Proposed fix: Replace that procedure with absolute-path `--session <unique-name> server`, an explicit isolated `HERDR_SOCKET_PATH`, headless tests, and `finally` teardown. Keep the owner's socket out of the test process.
- Risk of the fix: Installed skill copies must eventually be updated through the approved sync workflow; account copies are a separate, documented limitation (#43).

## F06 [S2] [PROVEN] check.ps1:46 - The safety check accepts Codex's unsandboxed mode
- Input / situation: A Codex profile has `args = ['-a','never','-s','danger-full-access']`.
- Expected: FAIL for removing Codex's workspace sandbox, as expressly required by CLAUDE.md safety rule 3 and check.ps1:43-44.
- Actual: The blacklist matches nothing and the checker prints the normal owner-approved auto-mode OK line.
- Evidence: The forbidden list at 33 contains bypass switches but not the `-s`/`--sandbox` value. Running its predicates with those fake flags produced `hits=0 auto=True`: `$hit=@($flags | Where-Object { $_ -in $dangerous }); $auto=[bool]($flags | Where-Object { $_ -in @('auto','never','--always-approve') })`.
- Proposed fix: In the agent-profile check, parse Codex sandbox options, including `--sandbox=...`, and reject `danger-full-access` as well as the bypass switches. Do not change the owner's approved workspace-write configuration.
- Risk of the fix: Legitimate restricted sandbox modes and the explicit Antigravity exception must remain accepted.

## F07 [S2] [PROVEN] check.ps1:67 - Swapped key commands pass verification
- Input / situation: The live configuration has the F and Shift+F command strings exchanged between their respective blocks.
- Expected: FAIL: the shortcuts now do each other's jobs.
- Actual: All ten keys pass, because each key and each command occurs somewhere in the file.
- Evidence: Exact predicate: `if ($liveText.Contains($block) -and $liveText.Contains($m.Groups[3].Value))`. Tested with the actual master text in memory, replacing the first two command captures by the other's value (from the higher index first using `String.Remove(...).Insert(...)`), then evaluating this predicate for every regex match. Output: `check swapped F and Shift+F commands: passed=10 of 10`. No live config was changed.
- Proposed fix: Parse/compare each key's own command block, including type; reject duplicate ambiguous bindings.
- Risk of the fix: Tolerate harmless TOML formatting/comments instead of relying on one exact line arrangement.

## F08 [S2] [PROVEN] check.ps1:107 - Documentation verification skips additional paths in a table cell
- Input / situation: AGENTS.md is missing, while CLAUDE.md exists; the FILES table lists both in one cell. The two skill-directory cell has the same issue.
- Expected: FAIL for every listed file/directory that is absent.
- Actual: Only the first backtick-delimited path is checked; the subsequent split cannot recover the others.
- Evidence: Execute the exact extraction on a real-shaped cell:
  ```powershell
  $text='| `CLAUDE.md`, `AGENTS.md` | rules |'
  foreach($m in [regex]::Matches($text,'\|\s*`([^`]+)`')) {
      $m.Groups[1].Value -split '`, `'
  }
  ```
  Output: `CLAUDE.md`. The capture already stops before the comma and second filename. The later root-file loop only inspects files that exist and cannot detect missing AGENTS.md.
- Proposed fix: Restrict to the first table cell, then enumerate every backtick span inside that cell before checking paths.
- Risk of the fix: Avoid treating prose/code examples in other table columns as filesystem paths.

## F09 [S2] [READ] claude-limits.ps1:53 - "cost today" counts an active session's lifetime cost
- Input / situation: A Claude session cost $20 yesterday, then $1 today; its status-line file is refreshed today with cumulative `cost = 21`.
- Expected: `usage`'s documented "cost today" is $1 (docs/GUIDE.md:434-435).
- Actual: It reports $21, including yesterday. A session only needs a refresh today to have its entire lifetime cost counted today.
- Evidence: Line 53: `if ($f.LastWriteTime.Date -eq (Get-Date).Date) { $costToday[$acc] += [double]$d.cost }`. The source is cumulative session cost (HUD header:14 and LESSONS #30), not a daily delta. No per-day baseline is maintained.
- Proposed fix: Either rename the metric accurately to cumulative cost of sessions updated today (smallest honest fix), or maintain daily per-session baselines/deltas if actual daily cost is required.
- Risk of the fix: Baselines need midnight/restart handling; a label-only fix changes the documented meaning but avoids inventing unavailable data.

## F10 [S2] [PROVEN] claude-limits.ps1:54 - Accounts with unknown five-hour usage are selected as NEXT
- Input / situation: A status-line file has `rate_limits = {}` or only a seven_day window; five_hour is absent.
- Expected: Unknown usage is not eligible for NEXT, per `Get-ClaudeNextAccount`'s comment at 100-101 and HISTORY Phase 24.
- Actual: An empty object is truthy, so HasData becomes true. FiveHour remains null, Limit is false, and NEXT selects the account (null also sorts before numeric usage).
- Evidence: Extract `Get-ClaudeLimits`, `Get-ClaudeAccountOf`, `Get-ClaudeNextAccount`; set threshold 95/cache empty. Mock `Get-ClaudeAccountNames` to one main account named `test`; `Get-ChildItem` to one file dated now; `Get-Content` to `{"config_dir":"","cost":25,"rate_limits":{}}`. `Get-ClaudeNextAccount @(Get-ClaudeLimits)` returned account `test`; the row was `HasData=true, FiveHour=null, SevenDay=null, Limit=false`. With `{"seven_day":{"used_percentage":20,"resets_at":4102444800}}` it likewise returned `test` with FiveHour null.
- Proposed fix: Require an actual known five-hour percentage for NEXT eligibility; distinguish partial/empty rate-limit data from a known zero. Preserve expired known windows becoming zero.
- Risk of the fix: Accounts that genuinely omit the weekly field should still be usable when their five-hour data is valid.

## F11 [S2] [PROVEN] herdr-agents.ps1:125 - Asking an already BLOCKED agent returns FAILED/1 instead of BLOCKED/3
- Input / situation: `ask` encounters Herdr's `agent_blocked` rejection before any text is sent.
- Expected: BLOCKED, visible screen, exit 3 (script header:32 and docs/DELEGATION.md result table).
- Actual: `Complete-Ask` returns Outcome FAILED and exit 1; the same classification propagates to fanout.
- Evidence: Extract `Complete-Ask` and `Format-Screen`; mock `Invoke-Herdr` to return visible text. Invoke `Complete-Ask test ([pscustomobject]@{Ok=$false;Json=('{"error":{"code":"agent_blocked"}}'|ConvertFrom-Json);Text='agent_blocked'})`. Observed `Outcome=FAILED, Exit=1`, text starting `FAILED (agent_blocked): agent_blocked`. Lines 125-132 return before the BLOCKED branch at 152-156. herdr-skill.md:145 documents the pre-send rejection.
- Proposed fix: Handle `agent_blocked` separately as BLOCKED/3, keeping its visible-screen output and preserving timeout/error failure handling.
- Risk of the fix: Fanout/collect exit precedence must continue to prefer real failures over blocked agents.

## F12 [S2] [PROVEN] herdr-agents.ps1:159 - Completion reports DONE even when reading the answer fails
- Input / situation: The wait succeeds with agent status done, then the agent exits or the server fails before `agent read`.
- Expected: DONE only after reading the result back (CLAUDE.md engineering rule 3), and no success claim when the answer could not be obtained.
- Actual: The failed read's error text becomes the "answer"; Outcome remains DONE and exit remains 0.
- Evidence: With extracted `Complete-Ask`/`Format-Screen`, `$Lines=200`, mock `Invoke-Herdr` as `{ param([string[]]$Arguments) [pscustomobject]@{Ok=$false;Text='READ FAILED';Json=$null} }`. Call `Complete-Ask test ([pscustomobject]@{Ok=$true;Json=('{"result":{"agent":{"agent_status":"done"}}}'|ConvertFrom-Json);Text='ok'})`. Output: `{"Name":"test","Outcome":"DONE","Exit":0,"Text":"DONE: test is done\n--- recent output of test ---\nREAD FAILED"}`. Line 160 uses only `.Text`; the read's `.Ok` is discarded.
- Proposed fix: In `Complete-Ask`, retain/check the read response before forming DONE; return FAILED/nonzero with the read error if the transcript is unavailable.
- Risk of the fix: Do not suppress the BLOCKED screen or imply that a read failure means the submitted task was not delivered.

## F13 [S2] [READ] herdr-agents.ps1:245 - Repeated account aliases and the NEXT shortcut fail on a fixed live name
- Input / situation: `claude-2040` is already live and 2040 remains the least-used account; press Ctrl+B Shift+C, or run `start 2040` again without an explicit Name.
- Expected: The documented NEXT action opens another Claude beside the focused pane; unspecified names receive numeric suffixes (skill herdr-team:54-55).
- Actual: Alias resolution supplies fixed Name `claude-2040`; the explicit-name collision branch aborts rather than generating a suffix. The headless key action consequently opens nothing.
- Evidence: `start next` chooses the alias at 241; lines 247-248 assign `$Account = $al.account` and `$Name = $al.name`. Suffix generation only occurs inside `if (-not $Name)` at 262-266; the next branch at 267-268 is `Fail "An agent named '$Name' is already running."`. All three account aliases define a fixed name in agents.json:253-268.
- Proposed fix: Track whether Name came from the user. Treat an alias's default name as a base for normal suffix generation; keep explicit user-name collision errors.
- Risk of the fix: Existing scripts expecting the first unsuffixed account name must still receive it when it is available.

## F14 [S2] [READ] herdr-agents.ps1:271 - Starting in an existing pane records the caller's folder instead of that pane's folder
- Input / situation: From folder A, use the documented `start codex -Pane P` where the empty shell P is in folder B. The agent later exits; run `diff` using its retained record.
- Expected: Record/review the folder where that agent actually ran.
- Actual: Cwd defaults to A and is recorded as A, while the supplied-pane branch neither changes nor reads B. After the live agent disappears, `diff` reviews A.
- Evidence: Line 271: `if (-not $Cwd) { $Cwd = (Get-Location).ProviderPath }`. The `elseif ($Pane)` branch at 306-313 only selects the pane and optionally sends account environment assignments. Line 325 records `cwd = $Cwd`. `diff` at 584 falls back to `$entry.cwd` when the agent is no longer live. Explicit `-Cwd` with `-Pane` is likewise never applied to that shell.
- Proposed fix: Read the supplied pane's actual cwd and record it. If explicit Cwd is supported with Pane, apply/verify it before starting; otherwise reject a mismatch clearly.
- Risk of the fix: Layout/team callers already pass matching folders; don't inject a directory-change command into an occupied pane.

## F15 [S2] [PROVEN] herdr-agents.ps1:402 - A team whose lead is at a drive root starts in a drive-relative folder
- Input / situation: The lead's cwd is `D:\`, and a team member omits cwd (documented default: use the lead's folder).
- Expected: Pass `D:\` to pane creation and the child start.
- Actual: The code passes `D:`, which is drive-relative and can resolve to a previously visited D: directory rather than the root.
- Evidence: Line 402: `$here = if ($lead -and $lead.cwd) { ([string]$lead.cwd).TrimEnd('\') } else { ... }`. Executed `([string]'D:\').TrimEnd('\')`; output: `D:`. `$cwdOf` at 403 passes this value into the splits and child process. This is separate from native trailing-backslash quoting: it removes the path separator before argument passing.
- Proposed fix: Preserve the full cwd from Herdr; do not trim its root separator in the team branch.
- Risk of the fix: None to ordinary folders; rely on normal native argument passing rather than altering path meaning.

## F16 [S2] [READ] herdr-agents.ps1:665 - Failed pane closure deletes the ownership record and prevents retry
- Input / situation: `stop coder` encounters a temporary server/API error on `pane close`.
- Expected: Preserve the record when closure did not succeed, so the tool can retry safely.
- Actual: State is saved without coder before checking the close response. The next stop says the agent was not started by this tool even though it remains running.
- Evidence: Lines 664-666: `$r = Invoke-Herdr @("pane", "close", $entry.pane)`; `Save-State @($state | Where-Object { $_.name -ne $Target })`; `if (-not $r.Ok) { Fail ... }`.
- Proposed fix: Remove the record only after confirmed closure (or a confirmed already-gone pane); retain it on an unconfirmed error, together with the identity validation from F01/F03.
- Risk of the fix: A lost API response may follow a successful close; inspect live state before attempting another close.

## F17 [S2] [READ] herdr-open-folder.ps1:110 - The team shortcut does not request its promised Codex model
- Input / situation: The owner's Codex default model differs from gpt-6.1-sol; use Ctrl+B T / `-Agent team`.
- Expected: Claude left and Codex SOL 6.1 right, as the script header:5, config/keys.toml:16, README keys, and GUIDE promise.
- Actual: `agent start` receives no model arguments and uses the CLI's saved/default model. Editing agents.json's model does not affect this shortcut either.
- Evidence: Line 110: `$output = & $herdr agent start $name --kind codex --pane $codexPane --timeout 90000 2>&1`. There is no `-- -m ...`, registry lookup, or delegation-tool call in this branch. agents.json:33 defines `gpt-6.1-sol`, but this file never reads it.
- Proposed fix: Pass the documented model explicitly, preferably reading the Codex profile's model/modelFlag from agents.json; keep all owner approval-mode choices unchanged.
- Risk of the fix: Decide whether profile edits should update the shortcut promise; avoid accidentally changing pane ownership/stop behavior by replacing the whole start pathway.

## F18 [S2] [PROVEN] herdr-spaces.ps1:111 - Three or more panes in a joined space do not get the claimed equal rows
- Input / situation: Join a space whose active tab has three panes with another space.
- Expected: Equal rows inside that space's column (HISTORY Phase 14 explicitly says the down-ratios produce equal rows).
- Actual: For three panes the heights are 1/2, 1/3, 1/6 of the column; more panes become still less balanced.
- Evidence: Each iteration splits `$below` (the newest/bottom pane), retaining `1.0 * $k / ($k + 1)` in the old pane. Executed that exact recurrence from an initial height 1: `for($k=1;$k -lt 3;$k++){ $r=1.0*$k/($k+1); $h=$rects[-1]; $rects[-1]=$h*$r; $rects+=$h*(1-$r) }`, with `$rects=@(1.0)`. Output: `0.5 0.333333 0.166667`. LESSONS #17 establishes that ratio belongs to the original pane.
- Proposed fix: In `Join-Spaces`, split the bottom remainder using `1.0 / ($src.Panes.Count - $k + 1)`, as the equal-column builders already do.
- Risk of the fix: One/two-pane columns must stay unchanged; terminal cell rounding will still cause at most normal one-cell differences.

## F19 [S2] [PROVEN] herdr-spaces.ps1:151 - Separate destroys its recovery record when the server cannot be read
- Input / situation: An existing side-by-side view has a saved record, but `pane list` fails. Move failures after some groups separate also lose their remaining recovery instructions.
- Expected: Report the API failure and retain enough state to retry Separate.
- Actual: A failed list is treated as every pane being gone; the undo file is removed unconditionally. The warning blames closed/moved panes instead of the server failure.
- Evidence: Extract `Split-Spaces`. Mock `Test-Path` true, `Get-Content` as `{"groups":[{"label":"A","panes":["w1:p1"]}]}`, `Invoke-Herdr` as `{Ok=$false;Text='server unavailable';Json=$null}`, `Remove-Item` to set an in-memory deleted flag, and `Show-Message` to capture text. `Split-Spaces` produced `recordDeleted=True`, message `Separated, with problems: A: its panes are gone (closed or moved)`. Code 132-137 continues with empty `$alive` on failure; 151 always removes the record.
- Proposed fix: Abort without deleting anything if discovery fails. After moves, persist only unresolved groups/panes with any new IDs, and delete the record only once all groups are confirmed separated.
- Risk of the fix: Partial-success retries must not move already separated panes a second time; retain names/IDs transactionally.

## F20 [S2] [PROVEN] herdr-stop.ps1:16 - A running server with zero agents cannot be stopped through Stop Herdr
- Input / situation: Herdr has only shell panes (possibly running builds/commands), with no recognized agents.
- Expected: Status says running with zero agents; the dialog still offers to stop the server and all panes (GUIDE 13.15).
- Actual: The empty array is unrolled out of `Get-Running`, becoming null. Both Status and the dialog say Herdr is not running; the stop pathway is unreachable.
- Evidence: Extract `Get-Running`; define `function Fake-Herdr { $global:LASTEXITCODE=0; '{"result":{"agents":[]}}' }`; set `$herdr='Fake-Herdr'; $agents=Get-Running`. Observed `$null -eq $agents` => `True`, selecting `Herdr is not running.`. Line 16 returns `@((... ).result.agents)` through a function pipeline; lines 21 and 29 use null as the server-not-running sentinel.
- Proposed fix: Return a wrapper containing separate success/running and Agents fields, or otherwise preserve the empty collection while distinguishing it from an API failure.
- Risk of the fix: Connection failures must still avoid a misleading confirmation dialog; one-agent and many-agent lists must remain printable.

## F21 [S2] [PROVEN] Herdr.cmd:23 - The fallback launcher breaks when opened from a drive root
- Input / situation: Run the documented fallback from `D:\`, or supply a folder ending with a backslash.
- Expected: Windows Terminal receives that folder as one complete `-d` argument, just as the compiled launcher does.
- Actual: The closing quote is escaped under Windows argument parsing; following options/path text becomes part of the directory argument.
- Evidence: Line 23: `start "" wt.exe -d "%START_DIR%" --title Herdr "%HERDR_EXE%"`. Passed the resulting command `wt.exe -d "D:\" --title Herdr "C:\Program Files\Herdr\herdr.exe"` to `shell32!CommandLineToArgvW` (P/Invoke in scratch PowerShell, no processes started). Output argv: `["wt.exe","-d","D:\" --title Herdr C:\\Program","Files\\Herdr\\herdr.exe"]`. LESSONS #40 and the compiled `Quote()` already explain/fix this exact native-argument defect; the batch fallback still contains it.
- Proposed fix: Double trailing backslashes when constructing the native wt directory argument, or forward to the compiled launcher's tested quoting path. Preserve batch fallback behavior when wt is unavailable.
- Risk of the fix: Do not double ordinary interior backslashes or change the fallback `cd /d` path; test roots, spaces, UNC paths, and trailing separators separately.

## F22 [S2] [READ] install.cmd:18 - The installer entry point invokes Windows PowerShell instead of required PowerShell 7
- Input / situation: Run the documented `install.cmd -Channel stable` with pwsh installed.
- Expected: Repository scripts run under PowerShell 7, per CLAUDE.md engineering rule 3.
- Actual: The batch entry point explicitly launches Windows PowerShell 5.1, regardless of installed pwsh.
- Evidence: Line 18: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%INSTALLER_PATH%" %*`. There is no pwsh selection/fallback check before it. The deliberate batch console flash is not this finding; the wrong PowerShell host is.
- Proposed fix: Launch `pwsh.exe -NoProfile ...`; give an actionable prerequisite message if PowerShell 7 is absent.
- Risk of the fix: Fresh machines need a documented PowerShell 7 prerequisite; preserve installer exit propagation and cleanup.

## F23 [S2] [PROVEN] launcher/HerdrLauncher.cs:98 - Relative folder arguments are resolved twice when starting Windows Terminal
- Input / situation: From the toolkit folder, launch with existing relative argument `docs`.
- Expected: Open the toolkit's docs folder (the design accepts any existing folder argument).
- Actual: `ResolveFolder` returns the relative string `docs`. Start sets WorkingDirectory to docs and also passes `-d "docs"`, so wt resolves the second docs relative to the first: toolkit\docs\docs, which does not exist here.
- Evidence: Read the binary bytes and load them with `[Reflection.Assembly]::Load($bytes)` (no Main invocation). Invoke private static `Program.ResolveFolder('docs')`. Observed `folder=[docs]`; `[IO.Path]::GetFullPath((Join-Path $folder $folder))` = `C:\Users\m.labib\.herdr\worktrees\HERDR\fix-bug-audit\docs\docs`, `exists=False`. Source lines 96-98 return `argFolder` unchanged; 83 supplies folder to both `-d` and `Start`; 155 assigns `psi.WorkingDirectory = folder`. No process was launched.
- Proposed fix: Normalize the accepted existing folder to an absolute path in `ResolveFolder` before returning it.
- Risk of the fix: Preserve the deliberate System32 fallback and trailing-root handling; validate UNC and drive-relative inputs too.

## F24 [S3] [PROVEN] claude-limits.ps1:69 - Rounding usage before comparison triggers LIMIT below 95 percent
- Input / situation: A valid five-hour window reports `used_percentage = 94.6` and has not reset.
- Expected: Not at LIMIT until actual use reaches 95 percent (HISTORY Phase 24).
- Actual: The integer cast rounds it to 95; Limit becomes true and NEXT excludes it early.
- Evidence: Extract `Get-ClaudeLimits`/`Get-ClaudeAccountOf`, set threshold 95 and empty cache. Mock one main account/file with current timestamps and JSON `{"config_dir":"","cost":25,"rate_limits":{"five_hour":{"used_percentage":94.6,"resets_at":4102444800},"seven_day":{"used_percentage":20,"resets_at":4102444800}}}`. Output: `FiveHour=95 LIMIT=True`. Line 69 stores `[int]$r.used_percentage`; line 73 compares the rounded value.
- Proposed fix: Keep the percentage as double/decimal for decisions and round only the displayed text.
- Risk of the fix: Display a suitable fractional precision near the threshold so the table and decision do not look contradictory.

## F25 [S3] [READ] docs/CLINE-PASS.md:202 - The control reference still declares re-authentication necessary after its own verified sign-in update
- Input / situation: Follow "How to Switch the Saved Default Model" after reading the lead's verification at the start of this same document.
- Expected: Instructions consistent with the verified successful Cline Pass setup; historical failures labeled historical.
- Actual: The section says the PC currently requires re-authentication and still uses the retired model; line 59 also says live testing could not happen. The lead's note at 6-10 explicitly records successful owner authentication, four successful models, and the bare-ID rejection.
- Evidence: Line 202: `Because cline-pass on this PC currently reports "requires re-authentication" ... and still has the retired stealth/ox-alpha saved`. Its AGENT-ROSTER citation now says Cline Pass works. This is an internal stale-state contradiction; no claim about today's remote service health is needed.
- Proposed fix: Mark the old state as the pre-sign-in history and retain the model-switch commands without asserting current authentication failure; update the obsolete caveat at 59.
- Risk of the fix: Do not turn the documented four-model test into a claim that all listed models were tested; keep the unverified benchmark caveat.

## F26 [S3] [READ] docs/DELEGATION.md:92 - Cline provider/model instructions disagree with the shipped profiles
- Input / situation: Follow the delegation/guide's DeepSeek setup recipe or use the stated model defaults to select an agent.
- Expected: The recipe configures the provider the `deepseek` profile actually uses; defaults match agents.json.
- Actual: The docs tell the owner to configure direct DeepSeek and say the model comes from that login; shipped `deepseek` explicitly uses Cline Pass with `cline-pass/deepseek-v4-pro`. `cline` explicitly requests glm-5.3, not Cline's saved choice. Configuring only direct DeepSeek does not configure Cline Pass for this profile.
- Evidence: docs/DELEGATION.md:92-93 says `Cline's choice` / `chosen at cline auth deepseek` and `-P deepseek`. agents.json:88,95,117,124 instead has `cline-pass/glm-5.3`, provider `cline-pass`, `cline-pass/deepseek-v4-pro`, provider `cline-pass`. GUIDE:145-146,155,399-400 and SETUP:30 repeat the old recipe; skills/herdr-team/SKILL.md:54 says all others use CLI defaults despite explicit Gemini/Cline model fields.
- Proposed fix: Update these current-profile descriptions and DeepSeek-profile sign-in recipe to Cline Pass. Keep direct-provider instructions separately for the supported explicit `-Provider` override.
- Risk of the fix: Existing direct-provider users still need the override documented; do not change profiles or approval choices as a documentation fix.

## F27 [S3] [READ] docs/GUIDE.md:173 - The guide says Claude dollar cost is absent while describing it elsewhere
- Input / situation: Use section 5 to interpret the HUD's dollar figure.
- Expected: One consistent explanation of the shipped HUD.
- Actual: Section 5 says Claude's dollar cost is not shown; section 13.7 says it is shown and explains it.
- Evidence: Lines 172-173: `Claude's dollar cost is not shown (only Claude Code itself knows it ...)`. Lines 332-334 describe `$cost` on the border; herdr-hud.ps1:217-220 reads the status-line cost file and appends the dollar cost. HISTORY Phase 16 implemented this bridge.
- Proposed fix: Replace the obsolete section-5 sentence with the status-line-file explanation and its availability condition.
- Risk of the fix: Do not promise a dollar figure before Claude has written that session's cost file.

## F28 [S3] [READ] docs/SETUP.md:24 - Rebuild instructions install the superseded launcher pathway and omit the HUD startup setup
- Input / situation: Rebuild this setup by following "Install on a PC (or rebuild this one)" exactly.
- Expected: The current windowless launcher and self-starting HUD described by README, FILES, and HISTORY Phases 22-23.
- Actual: Step 6 points shortcuts directly at wt, which bypasses HUD startup; step 7 points Explorer at Herdr.cmd, reintroducing its deliberate console flash. There is no Startup HUD shortcut creation step. The key-binding step also still says three blocks although the master contains ten.
- Evidence: Lines 23-28 prescribe wt and `full path of Herdr.cmd`; lines 19-20 say `add the three` blocks. launcher/HerdrLauncher.cs:76-83 is the current HUD-plus-wt launch path, launcher/use-launcher.ps1 switches the existing registrations, and check.ps1:92-93 expects `Herdr HUD.lnk`. The intentional availability of Herdr.cmd as a fallback is fine; documenting it as the current rebuild target is the defect.
- Proposed fix: Document build/current launcher registration, all current key blocks, and the required headless HUD/Stop shortcut setup with backups. Retain fallback instructions explicitly as fallback.
- Risk of the fix: `use-launcher.ps1` only edits existing shortcuts/registry values, so fresh-machine instructions must create the prerequisites before invoking it.

## F29 [S3] [PROVEN] herdr-agents.ps1:270 - Name validation accepts uppercase names that the Herdr contract rejects
- Input / situation: `start codex -Name Coder`, or a team plan with that name.
- Expected: Reject the invalid name before creating panes; the documented alphabet is lowercase `[a-z][a-z0-9_-]{0,31}`.
- Actual: PowerShell's case-insensitive regex accepts it; the native agent-name validation then rejects startup after layout mutations. Team's preflight has the same mistake at 398.
- Evidence: Executed `'Coder' -notmatch '^[a-z][a-z0-9_-]{0,31}$'`; result `False` (accepted). The printed rule at 270 and herdr-skill.md:57 require lowercase names. This proves the local validation mismatch; the native rejection is traced from the documented contract, not tested against a live pane.
- Proposed fix: Use `-cnotmatch` for both name validations, or normalize deliberately before both uniqueness checks and startup.
- Risk of the fix: Preserve automatic lowercase names and explicit duplicate-name protection; avoid silently renaming user targets inconsistently.

## F30 [S3] [PROVEN] install.ps1:655 - Retention keeps one extra release when the activated release is old
- Input / situation: Four releases ordered newest r1 through oldest r4; activate r4 (a rollback/local package), with Retain/Keep = 3.
- Expected: Keep the current release plus two other releases: three in total.
- Actual: r1-r3 are kept before the loop sees current r4; r4 is always kept and increments the count, so all four remain.
- Evidence: Extract `Remove-OldReleases`. Mock `Test-Path` true, `Get-ChildItem` to directory objects `C:\fake\r1` ... r4 with descending dates, and `Remove-Item` to an in-memory removal list. `Remove-OldReleases 'C:\fake' 'C:\fake\r4' 3` produced `deleted=0 kept=4`. At 657-659 the current release is counted only when encountered, after the keep budget may already have been spent.
- Proposed fix: Reserve one retention slot for current first, then retain only Keep-1 newest non-current releases.
- Risk of the fix: Never delete the activated release; preserve the exclusion of staging/backup directories and the existing Keep<1 behavior unless separately decided.

## Checked and fine

- CLAUDE.md / AGENTS.md / LESSONS.md / HISTORY.md: read against every suspected defect; deliberate approval modes, stalled-prompt UNCONFIRMED/exit 0, headless conhost, server-environment inheritance, account skill-copy limitations, and the parked custom build were not reported as bugs. Phase/lesson numbering is continuous.
- herdr-skill.md: read in full; JSON creation/move shapes, original-pane ratio semantics, scoped identifiers, available-shell requirement, pre-send BLOCKED rejection, and read-only discovery were used as the contract.
- herdr-agents.ps1 tools: normal installed/missing CLI and empty-model/provider branches traced; registry enumeration and literal registry path are correct.
- herdr-agents.ps1 start/worktree: normal profile, missing profile/CLI, unsupported effort, worktree creation/error and existing-directory guards traced; plain Git avoids the documented Herdr worktree crash; worktrees are kept on stop. Findings above cover the ownership, supplied-pane, name, and alias failures.
- herdr-agents.ps1 account/login/accounts: normal account, absent account support, invalid/empty name, empty env/args and no account directories traced; explicit copy lists contain no sign-in filenames. Login leaves authentication to the owner. No login/authentication files were opened or commands run.
- herdr-agents.ps1 ask/fanout/collect: normal DONE, blocked, timeout/stall, missing names/task, comma-separated names and one-agent collect traced; threaded native responses retain individual exit codes. Findings above cover wrong blocked/read outcomes and run filename collision. Cline's unreliable state is already an explicit lesson, not a new detection bug.
- herdr-agents.ps1 read/history: visible/recent selection, zero/one/many history files, out-of-range selection, screen-padding stripping, and saved task-line offset traced; no additional defect found.
- herdr-agents.ps1 diff/watch/status: live versus recorded lookup, non-Git folder, zero/one/many agents, board sort/colors and one-shot loop traced. Read-only Git commands and caller paths use argument arrays. Moving baseline branches are not reported as a defect because the docs explicitly describe comparison with a base branch.
- herdr-agents.ps1 handoff / tasks/HANDOFF-TEMPLATE.md: replacement tokens agree; no-overwrite guard, no next account, no agents, non-Git folder and missing HISTORY fallback traced; TODOs intentionally remain for the lead. No handoff file was generated.
- claude-limits.ps1: main/account matching, trailing separators/case, newest file per account, expired reset -> zero, no files, and valid-data NEXT tie sorting traced; the data-quality, daily-cost and rounding defects are above. No real usage/session message text was read.
- herdr-hud.ps1: read top to bottom; zero/one/many panes, pane labels from pane list, missing session/cost files, incremental complete-line reads, truncation reset, Claude message-id deduplication, Codex usage fields, token formatting, TTL refresh, -Once notification suppression, and mutex loop traced. Integer formatting is display-only here; no separate HUD finding asserted without live metadata-merge evidence. F01's state-file fix must update its readers too. HUD was never executed.
- herdr-layout.ps1: shell/new/existing column pathways, one/six columns, duplicate running-agent UI guard, missing preset, invalid saved folder, per-column account propagation, remembered choices and placeholder/new-ID updates traced. Correct equal-column recurrence; normal root paths here are passed without the team's trimming defect. No layout was executed.
- herdr-columns.ps1: Count validation 2..8, right/down, explicit/focused pane, same-cwd propagation, native failure checks and 1/N remainder recurrence traced. Read-only installed CLI help confirms `pane current --pane ID` is valid.
- herdr-spaces.ps1: active-tab filtering, empty sources, label/ID lookup, duplicate selection, two-pane case, pane-name preservation, placeholder closure and move-result IDs traced. Equal-row and failed-Separate recovery defects are above; only fake Split-Spaces dependencies were executed.
- herdr-open-folder.ps1: cancel, missing folder, spaces/brackets in literal paths, remembered folder, workspace JSON root pane, native workspace/run errors, team split failure and root label fallback traced. Team's model omission is above. No picker or workspace was opened.
- herdr-help.ps1: key formatting, adjacent-comment parsing, current ten custom bindings, built-in rows, empty/nonempty filter, Markdown fallback, browser failure timer and disposal traced. The existing default-handler implementation was deliberately chosen in HISTORY Phase 16/LESSONS #31; it was not retested or reported as a new browser feature defect. No help/browser window was opened.
- herdr-stop.ps1: one/many agents, working/blocked warning and No-default confirmation traced; zero agents is F20. Neither the real Status branch nor stop dialog/server stop was run.
- config/keys.toml / config/settings.toml: all ten key commands have conhost --headless and quoted script paths; dialog commands have -STA; bindings, sidebar token references and toast setting correspond to the scripts. Installed fixed repository paths are intentional, not defects merely because this audit is in a worktree.
- agents.json: JSON parses; all profiles/kinds/model/provider/effort and account placeholders read; official supported kinds were verified with absolute-path `herdr.exe agent`. Owner approval decisions were preserved and not treated as findings. Fixed alias-name collision is F13; no secrets were read.
- sync-skills.ps1: equal/changed/missing installed copy and Check versus copy branches traced; changed copies are backed up before overwrite. Hash-only CRLF/LF drift in this worktree is explained in the baseline, not an installed-content bug. Sync was not run independently; only check.ps1's authorized -Check child ran.
- check.ps1: baseline run completed; root parsers, actual profile checks, live key/settings presence, skill drift, registration/shortcut existence, PATH and phase/lesson checks read. Launcher scripts were also parsed separately (both zero errors). Three checker false-negative cases are documented above; no baseline failure was hidden.
- install.ps1: all 935 lines read, including all functions and main flow. Zero/four local-package parameters, channel validation, manifest string/object assets, BOM conversion, SHA-256 checks, ZIP/EXE validation, release locking/staging, rollback after activation failure, junction refusal, PATH normalization, and native exit checks traced. No download/install/registry mutation executed. Findings cover PATH backup, retention and the batch entry host.
- install-custom-build.ps1: normal build, missing output/official backup, SkipBuild and Restore traced; native cargo/version exit codes checked, original/previous binaries kept. Build incompatibility and unavailable upstream clone/toolchain are already deliberate/parked facts; the script was not run.
- Herdr.cmd / herdr-agents.cmd / install.cmd: dispatch, spaces in script paths, arguments/exit forwarding, missing Herdr, wt fallback and download failure cleanup read. The fallback batch's console flash is explicitly documented, not a new finding. Root native quoting and installer host are above; none of the batch entry points was run.
- launcher/HerdrLauncher.cs / launcher/HerdrLauncher.exe: source read completely; existing/missing absolute folder, System32 fallback, missing Herdr, missing HUD, wt fallback, dry-run non-start pathway and trailing-backslash Quote traced. Binary PE subsystem was read as 2 (GUI); only pure ResolveFolder was invoked by reflection, with the relative-path failure above. Main/dry-run was not executed and no temporary report was created.
- launcher/build.ps1 / launcher/use-launcher.ps1: source read completely; missing compiler/build failure, existing registration backup, missing shortcut skip, Undo newest-backup selection and result readback traced. Compiler native exit checked; these mutation scripts were not run.
- skills/herdr-team/SKILL.md: commands/parameters exist, plan fields agree with team code, command result contracts checked, roster/handoff references valid; stale defaults are included in F26. No delegation was started.
- skills/herdr-workspace/SKILL.md: maintenance/check/history/master-skill workflow read; unsafe live-test procedure is F05. Its general maintenance/commit steps were not executed during this audit-only task.
- docs/FILES.md / README.md: tracked locations/links and current launcher/key feature claims compared with code; missing untracked clone in this worktree is explained. Launcher/shortcut and profile discrepancies are recorded at their actionable sources above.
- docs/GUIDE.md / docs/DELEGATION.md / docs/SETUP.md: read completely against implementations and result contracts; current feature walkthroughs traced, defects above. GUIDE's old "approval ON" sentence is explicitly recorded as an intentionally unedited line in HISTORY Phase 18, so it is not another finding.
- docs/AGENT-ROSTER.md / docs/CLINE-PASS.md: read completely; profiles/routing/effort/account examples cross-checked with agents.json. Cline's stale pre-auth state is F25. Remote availability, prices, benchmark claims and untested model IDs were not re-verified: browser/network research was expressly out of scope, and the lead already marked the research limitations.
- Final scope: only this findings file was created; no fixes, commits, pushes, branch/index operations, live pane changes, worker starts, or isolated servers. Git-status verification showed only `?? tasks/audit-findings.md`.
