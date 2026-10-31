# Offline behaviour tests; imports definitions only, never connects to Herdr.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot))
$ErrorActionPreference = 'Stop'
$pass = 0; $failed = 0
function Check($Label, [scriptblock]$Body) {
    try { $result = @(& $Body); if ($result.Count -eq 1 -and $result[0] -is [bool] -and $result[0]) { $script:pass++; return }; $reason = 'expected one true result' } catch { $reason = $_.Exception.Message }
    $script:failed++; Write-Output "FAIL $Label : $reason"
}
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile((Join-Path $SourceRoot 'herdr-agents.ps1'), [ref]$null, [ref]$parseErrors)
if ($parseErrors) { throw $parseErrors }
foreach ($def in $ast.FindAll({ param($n) $n -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $true)) {
    . ([scriptblock]::Create($def.Extent.Text.Replace('$PSScriptRoot', "'" + $SourceRoot.Replace("'", "''") + "'")))
}
function Fail([string]$Message) { throw "ERROR: $Message" }
$now = [datetime]'2026-10-31T14:00:00'
foreach ($phrase in @("You've hit your limit", 'usage limit', 'limit reached')) {
    Check "no reset: $phrase" { (Get-LimitFromScreen $phrase $now).Until -eq $now.AddHours(1) }
}
foreach ($sample in @(
    @('Limit reached - resets 3pm', '2026-10-31T15:00:00'),
    @("You've hit your limit - resets 3:30pm (Europe/Paris)", '2026-10-31T15:30:00'),
    @("You've hit your limit - resets Nov 2, 9am", '2026-11-02T09:00:00'),
    @("You've hit your usage limit. Try again at 3:05 PM", '2026-10-31T15:05:00'),
    @('Limit reached - resets 9am', '2026-11-01T09:00:00'),
    @('Claude usage limit reached. Your limit will reset at 3pm (Africa/Cairo).', '2026-10-31T15:00:00'),
    @('5-hour limit reached ∙ resets 4pm', '2026-10-31T16:00:00'))) {
    Check $sample[0] { (Get-LimitFromScreen $sample[0] $now).Until -eq [datetime]$sample[1] }
}
foreach ($normal in @('rate limit handling in the API', 'The word "limit" describes a boundary.', 'The docs say "You''ve hit your limit".', ('Limit reached' + "`n" + ((1..40 | ForEach-Object { 'normal' }) -join "`n")))) {
    Check 'ignore normal or old text' { $null -eq (Get-LimitFromScreen $normal $now) }
}
$registryPath = Join-Path $SourceRoot 'agents.json'
function Get-Command { param($Name, $ErrorAction) if ($Name -ne 'missing') { [pscustomobject]@{ Name = $Name } } }
function Get-ClaudeLimits { @() }
$script:ClaudeLimitThreshold = 95
function Rows { @(Get-LeaderChoices @{} $now) }
# Owner's decision 2026-10-31: the lead starts on another Claude account, never the main sign-in.
Check 'configured chain leads with another Claude account, not the main sign-in' {
    $r = Rows
    $main = (Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json).accountNames.claude
    (($r | ForEach-Object { $_.Item }) -join ',') -eq '2040,1984,2030,codex,gemini' -and
        $r[0].Item -ne $main -and (Select-Leader $r '').Item -eq '2040' -and (Get-NextClaudeLeader @{} $now).Item -eq '2040'
}
Check 'skip current and limited' { $limits = @{ '2040' = @{ until = $now.AddHours(1).ToString('o'); source = 'screen' } }; (Select-Leader (Get-LeaderChoices $limits $now) '2030').Item -eq '1984' }
Check 'expired limit is free' { (Select-Leader (Get-LeaderChoices @{ '2040' = @{ until = $now.AddMinutes(-1).ToString('o'); source = 'screen' } } $now) '2030').Item -eq '2040' }
Check 'explicit target wins even limited' { (Select-Leader (Get-LeaderChoices @{ codex = @{ until = $now.AddHours(1).ToString('o'); source = 'screen' } } $now) '2030' 'codex').Item -eq 'codex' }
Check 'reject target outside chain' { try { $null = Select-Leader (Rows) '2030' 'unknown'; $false } catch { $_ -match 'chain' } }
Check 'uninstalled choice skipped' { $r = Rows; $r[0].Installed = $false; (Select-Leader $r '2030').Item -eq '1984' }
Check 'explicit missing tool fails' { $r = Rows; $r[3].Installed = $false; try { $null = Select-Leader $r '2030' 'codex'; $false } catch { $_ -match 'not installed' } }
Check 'usage LIMIT skipped' {
    function Get-ClaudeLimits { [pscustomobject]@{ Account = '2040'; HasData = $true; Limit = $true; FiveHour = 99; FiveResets = $now.AddHours(2); SevenDay = 0 } }
    (Select-Leader (Get-LeaderChoices @{} $now) '2030').Item -eq '1984'
}
Check 'all limited names earliest reset without writes' {
    $r = Rows; foreach ($row in $r) { $row.Until = $now.AddHours(2) }; $r[3].Until = $now.AddHours(1)
    try { $null = Select-Leader $r '2030'; $false } catch { $_ -match 'codex' -and $_ -match '15:00' }
}
$scratch = Join-Path $PSScriptRoot ('leader-test-' + [guid]::NewGuid().ToString('N'))
$null = New-Item -ItemType Directory -Path $scratch
try {
    Check 'state round trip' {
        $path = Join-Path $scratch 'leader.json'; Save-LeaderJson $path @{ item = '2040'; name = 'claude-2040'; pane_id = 'p1'; cwd = 'I:\herdr'; since = $now.ToString('o') }
        (Read-LeaderJson $path).item -eq '2040'
    }
    Check 'limits round trip' {
        $path = Join-Path $scratch 'leader-limits.json'; Save-LeaderJson $path @{ '2040' = @{ until = $now.AddHours(1).ToString('o'); source = 'screen'; seen = $now.ToString('o') } }
        (Read-LeaderJson $path)['2040'].source -eq 'screen'
    }
    Check 'corrupt state reports filename' {
        $path = Join-Path $scratch 'broken.json'; Set-Content -LiteralPath $path -Value '{bad'
        try { $null = Read-LeaderJson $path; $false } catch { $_ -match 'broken.json' }
    }
    Check 'invalid leader schema reports filename' {
        $path = Join-Path $scratch 'leader.json'; Set-Content -LiteralPath $path -Value '{}'
        try { $null = Read-LeaderJson $path; $false } catch { $_ -match 'leader.json.*Missing current leader fields' }
    }
    Check 'invalid reset reports filename' {
        $path = Join-Path $scratch 'leader-limits.json'; Set-Content -LiteralPath $path -Value '{"2040":{"until":"bad-date","source":"screen","seen":"2026-10-31"}}'
        try { $null = Read-LeaderJson $path; $false } catch { $_ -match 'leader-limits.json' }
    }
    Check 'handoff keeps real items, previous path and only last 80 lines' {
        $PSScriptRoot = $SourceRoot; $Cwd = $SourceRoot; $TaskFile = Join-Path $scratch 'handoff.md'
        function Get-ClaudeAccountNames { @() }; function Get-ClaudeAccountOf { '2030' }; function Get-ClaudeNextAccount { $null }
        function Format-ClaudeLimits { 'no data' }; function Get-LiveAgents { @() }
        function Invoke-Git($Dir, $Arguments) { [pscustomobject]@{ Ok = $true; Text = $(if ($Arguments[0] -eq 'rev-parse') { $SourceRoot } else { 'fixture git state' }) } }
        function Get-ChildItem { [pscustomobject]@{ FullName = 'I:\herdr\tasks\handoff-earlier.md'; LastWriteTimeUtc = $now } }
        $screen = ((1..100 | ForEach-Object { "screen-line-$_" }) -join "`n")
        $null = New-LeaderHandoff '2040' 'codex' $screen
        $text = Get-Content -LiteralPath $TaskFile -Raw
        $text -match '# Handoff: 2040 -> codex' -and $text.Contains('handoff-earlier.md') -and $text.Contains('screen-line-21') -and -not $text.Contains("screen-line-20`n") -and $text.Contains('screen-line-100') -and $text.Contains('TODO') -and -not $text.Contains('You are Claude Code')
    }
    Check 'invalid chain item reports item' {
        $cfg = Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json
        $cfg | Add-Member -Force NoteProperty leaders ([pscustomobject]@{ order = @('unknown') })
        $registryPath = Join-Path $scratch 'agents.json'; $cfg | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $registryPath
        try { $null = Get-Registry; $false } catch { $_ -match 'Invalid leaders.order item.*unknown' }
    }
} finally {
    $resolvedScratch = [IO.Path]::GetFullPath($scratch)
    if (-not $resolvedScratch.StartsWith([IO.Path]::GetFullPath($PSScriptRoot).TrimEnd('\') + '\leader-test-', [StringComparison]::OrdinalIgnoreCase)) { throw 'Unexpected test cleanup path' }
    Remove-Item -LiteralPath $resolvedScratch -Recurse -Force
}
$registryPath = Join-Path $SourceRoot 'agents.json'

# Run real command bodies with fake Herdr replies, ownership and leader-state writes.
$dispatch = $ast.EndBlock.Statements | Where-Object { $_ -is [Management.Automation.Language.SwitchStatementAst] } | Select-Object -Last 1
$startClause = $dispatch.Clauses | Where-Object { $_.Item1.Value -eq 'start' }
$startText = $startClause.Item2.Extent.Text
$startBody = [scriptblock]::Create($startText.Substring(1, $startText.Length - 2))
foreach ($startMode in 'command', 'shared') {
Check "start next without usage starts and records the first free chain Claude ($startMode)" {
    $Target = 'next'; $Cwd = $SourceRoot; $Name = ''; $Account = ''; $Pane = ''; $Worktree = ''; $Effort = ''; $Model = ''; $Provider = ''; $Direction = 'right'; $Lead = $true
    $script:started = $null; $script:leadSaved = $null
    function Get-ClaudeNextAccount { $null }; function Get-LiveAgents { @() }; function Get-State { @() }
    function Read-LeaderJson { @{} }; function Get-ServerIdentity { 'fixture' }
    function Get-LeaderProject($Folder) { $Folder }; function Read-ProjectLeader { @{} }
    function Save-ProjectLeader($Project, $Value) { $script:leadSaved = $Value }
    function Save-LeaderJson($Path, $Value) { $script:leadSaved = $Value }
    function Save-State($Items) { $script:started = $Items }
    function Get-EnvArgs { @() }
    # 2040 is an alias WITH an account folder, so the real Get-Account would touch %USERPROFILE%.
    function Get-Account($Entry, $AccountName) { [pscustomobject]@{ Name = $AccountName; Tool = 'claude'; Dir = "fixture-$AccountName"; Env = @{}; Args = @(); Login = '' } }
    function Invoke-Herdr($Arguments) {
        [pscustomobject]@{ Ok = $true; Text = ''; Json = @{ result = @{ pane = @{ pane_id = 'new' } } } }
    }
    $previous = $env:HERDR_ENV; $previousPane = $env:HERDR_PANE_ID
    try {
        $env:HERDR_ENV = '1'; $env:HERDR_PANE_ID = 'old'
        if ($startMode -eq 'shared') { $null = Start-Agent } else { $null = & $startBody }
        $script:started[0].name -eq 'claude-2040' -and $script:leadSaved.item -eq '2040' -and $script:leadSaved.pane_id -eq 'new'
    } finally { $env:HERDR_ENV = $previous; $env:HERDR_PANE_ID = $previousPane }
}
}

function Run-SwitchFixture([string]$Mode) {
    $script:writes = @{}; $script:prompts = @(); $script:handoffs = 0; $script:startedCount = 0; $script:fixtureMode = $Mode
    $Cwd = $SourceRoot; $To = ''; $TaskFile = ''
    function Get-Date { $now }
    function Get-LeaderProject($Folder) { $Folder }
    function Read-ProjectLeader { Read-LeaderJson 'fixture-leader.json' }
    function Save-ProjectLeader($Project, $Value) { Save-LeaderJson 'fixture-project.json' $Value }
    function Read-LeaderJson($Path) {
        if ($Path.EndsWith('leader.json')) { return @{ item = '2030'; name = 'claude-2030'; pane_id = 'old'; cwd = $SourceRoot; since = $now.ToString('o') } }
        $limits = @{}
        if ($script:fixtureMode -eq 'exhausted') { foreach ($item in @('2040','1984','codex','gemini')) { $limits[$item] = @{ until = $now.AddHours(2).ToString('o'); source = 'screen' } } }
        $limits
    }
    function Save-LeaderJson($Path, $Value) { $script:writes[$Path] = $Value }
    function Get-LiveAgents { [pscustomobject]@{ name = 'claude-2030'; pane_id = 'old'; cwd = $SourceRoot } }
    function Get-State { [pscustomobject]@{ name = 'claude-2030'; pane = 'old' } }
    function Get-ServerIdentity { 'default' }; function Get-RecordServer { 'default' }
    function New-LeaderHandoff { $script:handoffs++; $script:lastHandoff = 'I:\herdr\tasks\fake-handoff.md' }
    function Start-Agent {
        $script:startedCount++; $script:agentStartFailed = $false
        $script:lastStartedLeader = [pscustomobject]@{ item = $Target; name = 'new-lead'; pane_id = 'new'; cwd = $Cwd; since = $now.ToString('o') }
    }
    function Invoke-Herdr($Arguments) {
        if ($Arguments[1] -eq 'prompt') { $script:prompts += ,$Arguments }
        $text = if ($Arguments[2] -eq 'claude-2030') { "You've hit your limit - resets 3pm" }
            elseif ($script:fixtureMode -eq 'dialog') { "Do you trust this folder?`nEnter to confirm" } else { 'Working on the takeover' }
        [pscustomobject]@{ Ok = $true; Text = $text; Json = @{ result = @{ agent = @{ agent_status = 'working' } } } }
    }
    $previous = $env:HERDR_ENV; $previousPane = $env:HERDR_PANE_ID
    try {
        $env:HERDR_PANE_ID = 'focus'; $env:HERDR_ENV = '1'
        $problem = ''; $output = @()
        try { $output = @(Switch-Leader) } catch { $problem = $_.Exception.Message }
        [pscustomobject]@{ Error = $problem; Output = $output; Writes = $script:writes; Prompts = $script:prompts; Handoffs = $script:handoffs; Starts = $script:startedCount }
    } finally { $env:HERDR_ENV = $previous; $env:HERDR_PANE_ID = $previousPane }
}
Check 'switch sends a non-waited lead prompt and records read-back' {
    $r = Run-SwitchFixture success
    $r.Error -eq '' -and $r.Writes.Count -eq 2 -and $r.Handoffs -eq 1 -and $r.Starts -eq 1 -and $r.Prompts.Count -eq 1 -and $r.Prompts[0] -notcontains '--wait' -and ($r.Output -join "`n") -match 'SWITCHED 2030 -> 2040'
}
Check 'exhausted switch changes no state and creates no handoff' {
    $r = Run-SwitchFixture exhausted
    $r.Error -match 'earliest free' -and $r.Writes.Count -eq 0 -and $r.Handoffs -eq 0 -and $r.Starts -eq 0
}
Check 'startup dialog sends nothing and returns BLOCKED 3' {
    $r = Run-SwitchFixture dialog
    $r.Error -eq '' -and $r.Output[-1] -eq 3 -and ($r.Output -join "`n") -match 'BLOCKED' -and $r.Prompts.Count -eq 0 -and $r.Writes.Count -eq 1
}
# Per-project delta: real temporary files and fake Herdr/git only.
$deltaStart = $pass + $failed; $deltaFailedStart = $failed
& {
    $projectScratch = Join-Path $PSScriptRoot ('leader-project-' + [guid]::NewGuid().ToString('N'))
    $priorAppData = $env:APPDATA
    $priorPane = $env:HERDR_PANE_ID; $priorActiveCwd = $env:HERDR_ACTIVE_PANE_CWD
    $priorActivePane = $env:HERDR_ACTIVE_PANE_ID; $priorHerdrEnv = $env:HERDR_ENV
    $env:APPDATA = Join-Path $projectScratch 'state'
    $projectA = Join-Path $projectScratch 'project-a'; $projectB = Join-Path $projectScratch 'project-b'
    $plain = Join-Path $projectScratch 'plain'
    $null = New-Item -ItemType Directory -Path $projectA,$projectB,$plain,(Join-Path $projectA 'src'),(Join-Path $projectA '.git\info'),(Join-Path $projectB '.git\info') -Force
    Set-Content -LiteralPath (Join-Path $projectA 'README.md') -Value '# Project A'
    Set-Content -LiteralPath (Join-Path $projectB 'AGENTS.md') -Value '# Project B rules'
    Set-Content -LiteralPath (Join-Path $projectA '.gitignore') -Value 'original-ignore'
    function Invoke-Git($Dir, $Arguments) {
        if ($Arguments[0] -eq 'rev-parse') {
            if ($Dir -eq $plain -or $Dir.StartsWith($plain + '\', [StringComparison]::OrdinalIgnoreCase)) {
                return [pscustomobject]@{ Ok = $false; Text = 'not a git repository' }
            }
            foreach ($root in @($projectA,$projectB,$SourceRoot)) {
                if ($Dir -eq $root -or $Dir.StartsWith($root + '\', [StringComparison]::OrdinalIgnoreCase)) {
                    return [pscustomobject]@{ Ok = $true; Text = $root.Replace('\','/') }
                }
            }
            return [pscustomobject]@{ Ok = $false; Text = 'not a git repository' }
        }
        [pscustomobject]@{ Ok = $true; Text = 'fixture git facts' }
    }
    function Record($Item, $Project, $AgentName, $PaneId) {
        @{ item = $Item; name = $AgentName; pane_id = $PaneId; cwd = $Project; since = $now.ToString('o'); project = $Project }
    }
    function Get-ClaudeAccountNames { @() }; function Get-ClaudeAccountOf { '2030' }; function Get-ClaudeNextAccount { $null }
    function Format-ClaudeLimits { 'no data' }; function Get-State { @() }
    $script:projectCurrent = Record '2030' $projectA 'claude-2030' 'pane-a'
    function Get-LiveAgents { [pscustomobject]@{ name = $script:projectCurrent.name; pane_id = $script:projectCurrent.pane_id; cwd = $projectA } }
    function Start-Agent {
        $script:agentStartFailed = $false
        $script:projectCurrent = Record $Target $Cwd ($Target + '-a') 'new-a'
        $script:lastStartedLeader = [pscustomobject]$script:projectCurrent
    }
    function Invoke-Herdr($Arguments) {
        if ($Arguments[1] -eq 'prompt') { $script:projectPrompt = $Arguments[3] }
        [pscustomobject]@{ Ok = $true; Text = "You've hit your limit - resets 3pm"; Json = @{ result = @{ agent = @{ agent_status = 'working' }; pane = @{ cwd = $projectB } } } }
    }
    $Cwd = $projectA; $To = 'codex'; $TaskFile = ''; $env:HERDR_PANE_ID = 'pane-a'
    try {
        Check 'project: repo subfolder resolves to top-level' { (Get-LeaderProject (Join-Path $projectA 'src')) -eq $projectA }
        Check 'project: non-git folder resolves to itself' { (Get-LeaderProject $plain) -eq $plain }
        Check 'project: same repo shares a stable hashed state path' {
            $one = Get-LeaderStatePath (Get-LeaderProject $projectA)
            $two = Get-LeaderStatePath (Get-LeaderProject (Join-Path $projectA 'src'))
            $one -eq $two -and $one -match 'leaders\\project-a-[a-f0-9]{8}\.json$'
        }
        Check 'project: case-insensitive paths hash to the same file' {
            (Get-LeaderStatePath $projectA) -eq (Get-LeaderStatePath $projectA.ToUpperInvariant())
        }
        Check 'project: switch A preserves B state and starts in A root' {
            Save-ProjectLeader $projectA $script:projectCurrent
            Save-ProjectLeader $projectB (Record 'gemini' $projectB 'gemini-b' 'pane-b')
            $pathB = Get-LeaderStatePath $projectB; $before = [IO.File]::ReadAllText($pathB)
            $Cwd = Join-Path $projectA 'src'
            $null = Switch-Leader
            $afterA = Read-ProjectLeader $projectA
            [IO.File]::ReadAllText($pathB) -ceq $before -and $afterA.item -eq 'codex' -and $afterA.cwd -eq $projectA -and $afterA.project -eq $projectA
        }
        Check 'project: leaders Cwd B shows B CURRENT and project first' {
            $clause = $dispatch.Clauses | Where-Object { $_.Item1.Value -eq 'leaders' }
            $text = $clause.Item2.Extent.Text; $Cwd = $projectB
            $output = @(& ([scriptblock]::Create($text.Substring(1,$text.Length-2))))
            $output[0] -eq "Project: $projectB" -and ($output -join "`n") -match 'gemini.*CURRENT' -and ($output -join "`n") -notmatch 'codex.*CURRENT'
        }
        Check 'project: two handoffs stay local and exclude is added exactly once' {
            $null = Get-LeaderProject $projectA
            $Cwd = $projectA; $To = 'gemini'; $null = Switch-Leader
            $location = Join-Path $projectA '.herdr\handoffs'
            $files = @(Get-ChildItem -LiteralPath $location -Filter 'handoff-*.md')
            $exclude = @(Get-Content -LiteralPath (Join-Path $projectA '.git\info\exclude'))
            $latest = Get-Content -LiteralPath $script:lastHandoff -Raw
            $files.Count -eq 2 -and @($exclude | Where-Object { $_ -eq '.herdr/' }).Count -eq 1 -and (Get-Content -LiteralPath (Join-Path $projectA '.gitignore') -Raw).Trim() -eq 'original-ignore' -and @($files | Where-Object { $_.FullName -ne $script:lastHandoff -and $latest.Contains($_.FullName) }).Count -eq 1
        }
        Check 'project: existing exclude text survives without a trailing newline' {
            $path = Join-Path $projectB '.git\info\exclude'; [IO.File]::WriteAllText($path, '# existing exclusion')
            $null = Get-LeaderHandoffFolder $projectB -Prepare
            $null = Get-LeaderHandoffFolder $projectB -Prepare
            [IO.File]::ReadAllText($path) -ceq "# existing exclusion`n.herdr/`n" -and -not (Test-Path -LiteralPath (Join-Path $projectB '.gitignore'))
        }
        Check 'project: foreign template names only existing rules and drops shell rule' {
            $text = Get-Content -LiteralPath $script:lastHandoff -Raw
            $text.Contains((Join-Path $projectA 'README.md')) -and -not $text.Contains('CLAUDE.md') -and -not $text.Contains('LESSONS.md') -and -not $text.Contains('PowerShell only')
        }
        Check 'project: takeover paths are absolute and missing rules are absent' {
            $script:projectPrompt.Contains((Join-Path $projectA 'README.md')) -and $script:projectPrompt.Contains($script:lastHandoff) -and $script:projectPrompt.Contains((Join-Path $SourceRoot 'docs\TEAM-PLAYBOOK.md')) -and $script:projectPrompt.Contains((Join-Path $SourceRoot 'docs\DELEGATION.md')) -and $script:projectPrompt.Contains("Work in $projectA.") -and -not $script:projectPrompt.Contains('CLAUDE.md')
        }
        Check 'project: Herdr still uses tasks' { (Get-LeaderHandoffFolder $SourceRoot) -eq (Join-Path $SourceRoot 'tasks') }
        Check 'project: plain handoff uses non-git local folder without creating git metadata' {
            $null = Get-LeaderHandoffFolder $plain
            $Cwd = $plain; $null = New-LeaderHandoff
            $script:lastHandoff.StartsWith((Join-Path $plain '.herdr\handoffs') + '\') -and -not (Test-Path -LiteralPath (Join-Path $plain '.git'))
        }
        Check 'project: focused folder wins over a remembered old cwd' {
            $env:HERDR_ACTIVE_PANE_CWD = $projectB
            (Get-LeaderFolder '') -eq $projectB -and (Get-LeaderFolder $projectA) -eq $projectA
        }
        Check 'project: pane get supplies folder when key cwd is absent' {
            $env:HERDR_ACTIVE_PANE_CWD = ''
            (Get-LeaderFolder '') -eq $projectB
        }
        Check 'project: current location is used outside Herdr' {
            $env:HERDR_ACTIVE_PANE_CWD = ''; $env:HERDR_PANE_ID = ''; $env:HERDR_ACTIVE_PANE_ID = ''
            Push-Location -LiteralPath $plain
            try { (Get-LeaderFolder '') -eq $plain } finally { Pop-Location }
        }
        Check 'project: legacy containment does not confuse neighbouring names' {
            -not (Test-LeaderProjectFolder ($projectA + 'b\src') $projectA)
        }
        Check 'project: mismatched project field fails clearly' {
            Save-LeaderJson (Get-LeaderStatePath $projectA) (Record 'codex' $projectB 'wrong' 'wrong-pane')
            try { $null = Read-ProjectLeader $projectA; $false } catch { $_ -match 'project.*mismatch|mismatch.*project' }
        }
        $legacy = Join-Path $env:APPDATA 'herdr\leader.json'
        Check 'project: foreign legacy file is left alone' {
            Save-LeaderJson $legacy (Record '2030' $projectB 'legacy-b' 'legacy-pane')
            $null = Read-ProjectLeader $plain
            Save-ProjectLeader $plain (Record 'codex' $plain 'plain-lead' 'plain-pane')
            (Test-Path -LiteralPath $legacy) -and (Read-LeaderJson $legacy).name -eq 'legacy-b'
        }
        Check 'project: legacy leader is read without deleting before successful write' {
            $legacyFolder = Join-Path $plain 'child'; $null = New-Item -ItemType Directory -Path $legacyFolder -Force
            Remove-Item -LiteralPath (Get-LeaderStatePath $plain)
            Save-LeaderJson $legacy (Record '2040' $legacyFolder 'legacy-a' 'legacy-pane')
            $value = Read-ProjectLeader $plain
            $value.item -eq '2040' -and (Test-Path -LiteralPath $legacy) -and -not (Test-Path -LiteralPath (Get-LeaderStatePath $plain))
        }
        Check 'project: failed migration write keeps legacy intact' {
            function Save-LeaderJson { throw 'fixture write denied' }
            try { Save-ProjectLeader $plain (Record 'codex' $plain 'new' 'new-pane'); $false }
            catch { $_ -match 'write denied' -and (Test-Path -LiteralPath $legacy) }
        }
        Check 'project: successful migration removes legacy only after new write' {
            $script:projectCurrent = Record '2040' (Join-Path $plain 'child') 'legacy-a' 'legacy-pane'
            $env:HERDR_PANE_ID = 'legacy-pane'
            $Cwd = $plain; $To = '1984'; $null = Switch-Leader
            -not (Test-Path -LiteralPath $legacy) -and (Read-ProjectLeader $plain).item -eq '1984'
        }
        Check 'project: start Lead from a subfolder records the repo project' {
            $subfolder = Join-Path $projectB 'src'; $null = New-Item -ItemType Directory -Path $subfolder
            $Target = 'codex'; $Name = 'starter-b'; $Account = ''; $Pane = ''; $Worktree = ''; $Effort = ''; $Model = ''; $Provider = ''; $Direction = 'right'; $Lead = $true; $Cwd = $subfolder
            $env:HERDR_ENV = '1'; $env:HERDR_PANE_ID = 'caller'
            function Save-State { }
            function Invoke-Herdr { [pscustomobject]@{ Ok = $true; Text = ''; Json = @{ result = @{ pane = @{ pane_id = 'started-b' } } } } }
            $null = & $startBody
            $value = Read-ProjectLeader $projectB
            $value.project -eq $projectB -and $value.cwd -eq $subfolder -and $value.item -eq 'codex' -and $value.name -eq 'starter-b'
        }
    } finally {
        $env:APPDATA = $priorAppData; $env:HERDR_PANE_ID = $priorPane; $env:HERDR_ACTIVE_PANE_CWD = $priorActiveCwd
        $env:HERDR_ACTIVE_PANE_ID = $priorActivePane; $env:HERDR_ENV = $priorHerdrEnv
        $resolved = [IO.Path]::GetFullPath($projectScratch)
        if (-not $resolved.StartsWith([IO.Path]::GetFullPath($PSScriptRoot).TrimEnd('\') + '\leader-project-', [StringComparison]::OrdinalIgnoreCase)) { throw 'Unsafe project fixture cleanup' }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
Write-Output "PROJECT DELTA: $($failed - $deltaFailedStart) failed / $(($pass + $failed) - $deltaStart) tests"
Write-Output "PASS $pass/$($pass + $failed); FAIL $failed"
if ($failed) { exit 1 }
