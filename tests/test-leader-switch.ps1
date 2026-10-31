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
Check 'configured chain and no-data Claude fallback' { $r = Rows; (($r | ForEach-Object { $_.Item }) -join ',') -eq '2030,2040,1984,codex,gemini' -and (Get-NextClaudeLeader @{} $now).Item -eq '2030' }
Check 'skip current and limited' { $limits = @{ '2040' = @{ until = $now.AddHours(1).ToString('o'); source = 'screen' } }; (Select-Leader (Get-LeaderChoices $limits $now) '2030').Item -eq '1984' }
Check 'expired limit is free' { (Select-Leader (Get-LeaderChoices @{ '2040' = @{ until = $now.AddMinutes(-1).ToString('o'); source = 'screen' } } $now) '2030').Item -eq '2040' }
Check 'explicit target wins even limited' { (Select-Leader (Get-LeaderChoices @{ codex = @{ until = $now.AddHours(1).ToString('o'); source = 'screen' } } $now) '2030' 'codex').Item -eq 'codex' }
Check 'reject target outside chain' { try { $null = Select-Leader (Rows) '2030' 'unknown'; $false } catch { $_ -match 'chain' } }
Check 'uninstalled choice skipped' { $r = Rows; $r[1].Installed = $false; (Select-Leader $r '2030').Item -eq '1984' }
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
        function Invoke-Git { [pscustomobject]@{ Ok = $true; Text = 'fixture git state' } }
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
Check "start next without usage starts and records main Claude ($startMode)" {
    $Target = 'next'; $Cwd = $SourceRoot; $Name = ''; $Account = ''; $Pane = ''; $Worktree = ''; $Effort = ''; $Model = ''; $Provider = ''; $Direction = 'right'; $Lead = $true
    $script:started = $null; $script:leadSaved = $null
    function Get-ClaudeNextAccount { $null }; function Get-LiveAgents { @() }; function Get-State { @() }
    function Read-LeaderJson { @{} }; function Get-ServerIdentity { 'fixture' }
    function Save-LeaderJson($Path, $Value) { $script:leadSaved = $Value }
    function Save-State($Items) { $script:started = $Items }
    function Get-EnvArgs { @() }
    function Invoke-Herdr($Arguments) {
        [pscustomobject]@{ Ok = $true; Text = ''; Json = @{ result = @{ pane = @{ pane_id = 'new' } } } }
    }
    $previous = $env:HERDR_ENV; $previousPane = $env:HERDR_PANE_ID
    try {
        $env:HERDR_ENV = '1'; $env:HERDR_PANE_ID = 'old'
        if ($startMode -eq 'shared') { $null = Start-Agent } else { $null = & $startBody }
        $script:started[0].name -eq 'claude-2030' -and $script:leadSaved.item -eq '2030' -and $script:leadSaved.pane_id -eq 'new'
    } finally { $env:HERDR_ENV = $previous; $env:HERDR_PANE_ID = $previousPane }
}
}

function Run-SwitchFixture([string]$Mode) {
    $script:writes = @{}; $script:prompts = @(); $script:handoffs = 0; $script:startedCount = 0; $script:fixtureMode = $Mode
    $Cwd = $SourceRoot; $To = ''; $TaskFile = ''
    function Get-Date { $now }
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
Write-Output "PASS $pass/$($pass + $failed); FAIL $failed"
if ($failed) { exit 1 }
