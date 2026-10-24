# Offline regressions; -SourceRoot and -OnlyA exercise unchanged HEAD copies.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot), [switch]$OnlyA)
$ErrorActionPreference = 'Stop'
$script:pass = 0; $script:fail = 0
function Check([string]$Name, [scriptblock]$Test) {
    try { if (& $Test) { $script:pass++; return } } catch { Write-Output "  $($_.Exception.Message)" }
    $script:fail++; Write-Output "FAIL $Name"
}
$asts = @{}
foreach ($file in @('herdr-agents.ps1') + $(if (-not $OnlyA) { 'herdr-trial.ps1' })) {
    if (-not (Test-Path -LiteralPath (Join-Path $SourceRoot $file))) { continue }
    $errors = $null
    $ast = [System.Management.Automation.Language.Parser]::ParseFile((Join-Path $SourceRoot $file), [ref]$null, [ref]$errors)
    if ($errors) { throw "Cannot parse ${file}: $errors" }
    $asts[$file] = $ast
    foreach ($f in $ast.FindAll({ param($x) $x -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $true)) {
        # Keep the defining script directory when importing into this harness.
        $definition = $f.Extent.Text.Replace('$PSScriptRoot', "'" + $SourceRoot.Replace("'", "''") + "'")
        . ([scriptblock]::Create($definition))
    }
}
$switch = $asts['herdr-agents.ps1'].Find({ param($x) $x -is [System.Management.Automation.Language.SwitchStatementAst] }, $false)
$clause = $switch.Clauses | Where-Object { $_.Item1.Value -eq 'ask' }
$body = $clause.Item2.Extent.Text
$offset = $clause.Item2.Extent.StartOffset
foreach ($e in ($clause.Item2.FindAll({ param($x) $x -is [System.Management.Automation.Language.ExitStatementAst] }, $true) | Sort-Object { $_.Extent.StartOffset } -Descending)) {
    $body = $body.Remove($e.Extent.StartOffset - $offset, $e.Extent.Text.Length).Insert($e.Extent.StartOffset - $offset, 'throw ("TEST_EXIT:" + [string](' + $e.Pipeline.Extent.Text + '))')
}
$ask = [scriptblock]::Create($body.Substring(1, $body.Length - 2))
function Fail([string]$Message) { throw "ERROR: $Message" }
$freshScreen = "╭──────────────────────────────────────╮`n│ >_ OpenAI Codex (v0.163.0)            │`n╰──────────────────────────────────────╯`n`n› Ask Codex to do anything"
$menuScreen = "Where should the new conversation run?`n› 1. Use current Git worktree`n  2. Create new Git worktree"
function Reset-Fixture {
    $script:tick = 0; $script:screens = @($freshScreen); $script:kind = 'codex'
    $script:calls = [Collections.Generic.List[object]]::new()
}
function Get-Date { param([string]$Format)
    $now = ([datetime]'2026-10-23T12:00:00').AddSeconds($script:tick)
    if ($Format) { $now.ToString($Format) } else { $now }
}
function Start-Sleep { param($Seconds) $script:tick += $Seconds }
function Invoke-Herdr([string[]]$Arguments) {
    $script:calls.Add($Arguments)
    $session = @{ value = 'old-session' } # /new keeps the ID until the first message.
    $screen = ''
    if ($Arguments[1] -eq 'read') {
        $screen = $script:screens[0]
        if ($script:screens.Count -gt 1) { $script:screens = $script:screens[1..($script:screens.Count - 1)] }
    }
    [pscustomobject]@{ Ok = $true; Text = $screen; Json = @{ result = @{ agent = @{ agent = $script:kind; agent_session = $session }; agents = @(@{ name = 'trial-1' }, @{ name = 'trial-3' }) } } }
}
function Complete-Ask { [pscustomobject]@{ Outcome = 'DONE'; Text = 'answer'; Exit = 0 } }
function Save-Run { 'mock-log' }
function Run-Branch {
    $Target = 'coder'; $Task = 'do task'; $TaskFile = ''; $Fresh = $true; $TimeoutSec = 30
    try { & $ask | Out-Null; 'NO_EXIT' } catch { $_.Exception.Message }
}
Check 'A1 fresh sends /new then waits for ready' {
    Reset-Fixture; $script:screens = @($freshScreen, 'loading', $freshScreen)
    $result = Run-Branch
    $prompts = @($script:calls | Where-Object { $_[1] -eq 'prompt' })
    $result -eq 'TEST_EXIT:0' -and $prompts.Count -eq 2 -and $prompts[0][3] -eq '/new' -and
        $prompts[0] -notcontains '--wait' -and $prompts[1][3] -eq 'do task' -and $script:tick -ge 1
}
Check 'A2 fresh answers the worktree menu once' {
    Reset-Fixture; $script:screens = @($freshScreen, "$freshScreen`n$menuScreen", $menuScreen, $freshScreen)
    $result = Run-Branch
    $keys = @($script:calls | Where-Object { $_[1] -eq 'send-keys' })
    $result -eq 'TEST_EXIT:0' -and $keys.Count -eq 1 -and $keys[0][3] -ceq 'Enter' -and $script:tick -ge 2
}
Check 'A3 fresh times out with the read hint' {
    Reset-Fixture; $script:screens = @('loading')
    $result = Run-Branch
    $result -like '*Codex did not open a fresh conversation in 60 s; see: herdr-agents read coder -Visible*' -and
        $script:tick -eq 60 -and @($script:calls | Where-Object { $_[1] -eq 'prompt' }).Count -eq 1
}
Check 'A4 fresh refuses non-codex' {
    Reset-Fixture; $script:kind = 'cline'
    (Run-Branch) -like '*-Fresh is for Codex agents only.*' -and @($script:calls | Where-Object { $_[1] -eq 'prompt' }).Count -eq 0
}
Check 'A5 old conversation screen with bullets does not count' {
    $rejected = 0
    foreach ($transcript in @('• Ran Get-Date', '  • Edited herdr-agents.ps1', '• DONE', 'Worked for 33s')) {
        Reset-Fixture; $script:screens = @("$freshScreen`n$transcript")
        $result = Run-Branch
        if ($result -like '*Codex did not open a fresh conversation in 60 s; see: herdr-agents read coder -Visible*' -and
            $script:tick -eq 60 -and @($script:calls | Where-Object { $_[1] -eq 'prompt' }).Count -eq 1) { $rejected++ }
    }
    $rejected -eq 4
}
Check 'A6 fresh banner screen without bullets returns' {
    Reset-Fixture
    $script:screens = @('› Ask Codex to do anything', '>_ OpenAI Codex (v0.163.0)', $freshScreen)
    $result = Start-FreshConversation 'coder'
    $result -eq $true -and $script:tick -eq 2 -and $script:calls[0][1] -eq 'prompt' -and
        @($script:calls | Where-Object { $_[1] -in @('get', 'send-keys') }).Count -eq 0
}
Check 'A7 menu answered then fresh screen returns' {
    Reset-Fixture
    $script:screens = @($freshScreen, "$freshScreen`n$menuScreen", $menuScreen, $freshScreen)
    $result = Run-Branch
    $keys = @($script:calls | Where-Object { $_[1] -eq 'send-keys' })
    $result -eq 'TEST_EXIT:0' -and $script:tick -eq 2 -and $keys.Count -eq 1 -and $keys[0][3] -ceq 'Enter'
}
if (-not $OnlyA) {
    $tempRoot = Join-Path $env:TEMP ('round2-' + [guid]::NewGuid().ToString('N'))
    [IO.Directory]::CreateDirectory($tempRoot) | Out-Null
    try {
        Check 'B1 run name validation and existing folder' {
            $kit = Join-Path $tempRoot 'kit'; [IO.Directory]::CreateDirectory($kit) | Out-Null
            Set-Content -LiteralPath (Join-Path $kit 'TASKS.md') -Value 'fixture'
            $bad = 0
            foreach ($name in '../escape', 'Bad', '', ('a' * 50)) {
                try { New-TrialRun $tempRoot $name | Out-Null } catch { if ($_.Exception.Message -like '*Run name*') { $bad++ } }
            }
            $folder = New-TrialRun $tempRoot 'valid-run'
            $exists = $false
            try { New-TrialRun $tempRoot 'valid-run' | Out-Null } catch { $exists = $_.Exception.Message -like '*Run folder exists:*' }
            $bad -eq 4 -and $exists -and (Get-Content -LiteralPath (Join-Path $folder 'TASKS.md')) -eq 'fixture'
        }
        Check 'B2 report counts only when size is stable' {
            Reset-Fixture
            function Test-Path { param($LiteralPath, $PathType) $script:tick -ge 1 }
            function Get-Item { param($LiteralPath) [pscustomobject]@{ Length = $(if ($script:tick -lt 2) { 10 } else { 20 }) } }
            (Wait-TrialReport 'C:\fake' 1 1) -eq 'DONE' -and $script:tick -eq 3
        }
        Check 'B3 grade lines parsed' {
            function Invoke-TrialProcess { [pscustomobject]@{ Code = 0; Text = "noise`nLEDGER 18/20`nCOLUMNS 21/21`nSCOPE VIOLATED: extra x`nREVIEW.md present`nREPORT.md MISSING" } }
            $g = Invoke-TrialGrade 'C:\fake' 'C:\fake\run'
            $g.ledger -eq '18/20' -and $g.columns -eq '21/21' -and $g.scope -eq 'VIOLATED: extra x' -and $g.review -eq 'present' -and $g.report -eq 'MISSING'
        }
        Check 'B4 cline tokens from session matching cwd' {
            $fixtureHome = Join-Path $tempRoot 'user'; $runPath = Join-Path $tempRoot 'run'
            foreach ($entry in @(@('old', '.cline\data\sessions', $runPath, 50, 20), @('new', '.herdr-accounts\cline\work\data\sessions', ($runPath.ToUpperInvariant() + '\'), 200, 80), @('other', '.cline\data\sessions', 'C:\other', 999, 0))) {
                $dir = Join-Path $fixtureHome (Join-Path $entry[1] $entry[0]); [IO.Directory]::CreateDirectory($dir) | Out-Null
                $file = Join-Path $dir ($entry[0] + '.json')
                @{ cwd = $entry[2]; metadata = @{ usage = @{ inputTokens = $entry[3]; cacheReadTokens = $entry[4]; outputTokens = 12 } } } | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $file
                (Get-Item -LiteralPath $file).LastWriteTimeUtc = ([datetime]'2026-10-23').AddMinutes([array]::IndexOf(@('old','new','other'), $entry[0]))
            }
            $t = Get-TrialTokens 'cline' $runPath $fixtureHome
            $empty = Get-TrialTokens 'codex' $runPath $fixtureHome
            $t.fresh_in -eq 120 -and $t.out -eq 12 -and $empty.fresh_in -eq '' -and $empty.out -eq ''
        }
        Check 'B5 csv header once and rows appended' {
            $row = [pscustomobject][ordered]@{ date = 'today'; run = 'one'; profile = 'cline'; model = 'm,quoted'; effort = 'high'; outcome = 'DONE'; ledger = '1/1'; columns = '1/1'; scope = 'ok'; review = 'present'; report = 'present'; minutes = '1.2'; fresh_in = 20; out = 10 }
            Add-TrialResult $tempRoot $row
            $file = Join-Path $tempRoot 'results.csv'; $before = [IO.File]::ReadAllText($file)
            $row.run = 'two'; Add-TrialResult $tempRoot $row
            $rows = @(Import-Csv -LiteralPath $file)
            $rows.Count -eq 2 -and $rows[1].run -eq 'two' -and $rows[0].model -eq 'm,quoted' -and [IO.File]::ReadAllText($file).StartsWith($before) -and @(Get-Content -LiteralPath $file).Count -eq 3
        }
        Check 'B6 timeout gives TIMEOUT row' {
            Reset-Fixture
            function New-TrialRun { 'C:\fake\run' }
            function Start-TrialAgent { 'trial-2' }
            function Send-TrialPrompt { }
            function Test-Path { param($LiteralPath, $PathType) $false }
            function Invoke-TrialGrade { @{ ledger = '0/1'; columns = '0/1'; scope = 'ok'; review = 'MISSING'; report = 'MISSING' } }
            function Get-TrialTokens { @{ fresh_in = ''; out = '' } }
            function Get-Content { param($LiteralPath, [switch]$Raw) '{"agents":{"codex":{"kind":"codex","model":"default-model"}}}' }
            function Add-TrialResult($TrialRoot, $Row) { $script:resultRow = $Row }
            $script:resultRow = $null
            Invoke-Trial 'codex' '' '' 'timeout' 'C:\fake' 1 15 | Out-Null
            $script:resultRow.outcome -eq 'TIMEOUT' -and $script:resultRow.minutes -eq '1.0' -and $script:resultRow.model -eq 'default-model' -and $script:tick -eq 60
        }
        Check 'B7 startup selects lowest free name and forwards overrides' {
            Reset-Fixture
            function Invoke-TrialProcess($File, $Arguments) {
                $script:startArgs = $Arguments
                [pscustomobject]@{ Code = 0; Text = 'STARTED' }
            }
            $name = Start-TrialAgent 'cline' 'C:\trial space\run' 'custom-model' 'high'
            $expected = @('-NoProfile', '-File', (Join-Path $SourceRoot 'herdr-agents.ps1'), 'start', 'cline', '-Name', 'trial-2', '-Cwd', 'C:\trial space\run', '-Model', 'custom-model', '-Effort', 'high')
            $name -eq 'trial-2' -and ($script:startArgs -join '|') -ceq ($expected -join '|')
        }
        Check 'B8 NOT READY prints screen and stops without prompting' {
            Reset-Fixture
            function Invoke-TrialProcess { [pscustomobject]@{ Code = 2; Text = 'SCREEN: owner question' } }
            function Write-Host($Object) { $script:screenPrinted = $Object }
            $caught = ''
            try { Start-TrialAgent 'codex' 'C:\fake' '' '' | Out-Null } catch { $caught = $_.Exception.Message }
            $caught -like '*Agent not ready*rerun with -Run <existing> is NOT supported: stop it and start again.*' -and
                $script:screenPrinted -eq 'SCREEN: owner question' -and @($script:calls | Where-Object { $_[1] -eq 'prompt' }).Count -eq 0
        }
        Check 'B9 standard prompt exact and no wait' {
            Reset-Fixture; Send-TrialPrompt 'trial-2'
            $script:calls.Count -eq 1 -and ($script:calls[0] -join '|') -ceq 'agent|prompt|trial-2|Read TASKS.md in your current folder completely and do every task in it. Follow its rules exactly. When finished, reply with the answer format from TASKS.md.'
        }
        Check 'B10 defaults, DONE row, invariant minutes and stop reminder' {
            Reset-Fixture
            function New-TrialRun($TrialRoot, $Run) { $script:runName = $Run; 'C:\fake\run' }
            function Start-TrialAgent { 'trial-2' }
            function Send-TrialPrompt { }
            function Wait-TrialReport { $script:tick = 90; 'DONE' }
            function Invoke-TrialGrade { @{ ledger = '20/20'; columns = '21/21'; scope = 'ok'; review = 'present'; report = 'present' } }
            function Get-TrialTokens { @{ fresh_in = 100; out = 10 } }
            function Get-Content { param($LiteralPath, [switch]$Raw) '{"agents":{"ds-flash":{"kind":"cline","model":"default-model","effort":"high"}}}' }
            function Add-TrialResult($TrialRoot, $Row) { $script:resultRow = $Row }
            $culture = [Threading.Thread]::CurrentThread.CurrentCulture
            try {
                [Threading.Thread]::CurrentThread.CurrentCulture = [Globalization.CultureInfo]'fr-FR'
                $output = @(Invoke-Trial 'ds-flash' '' '' '' 'C:\fake' 1 15)
            } finally { [Threading.Thread]::CurrentThread.CurrentCulture = $culture }
            $script:runName -eq 'ds-flash-default-20261023-1200' -and $script:resultRow.minutes -eq '1.5' -and
                $script:resultRow.outcome -eq 'DONE' -and $script:resultRow.effort -eq 'high' -and $output[-1] -eq 'Review REVIEW.md yourself, then: herdr-agents stop trial-2'
        }
        Check 'B11 -Profile alias still works' {
            # Exercise the real script parameter binding and final call, without its main work.
            $trialAst = $asts['herdr-trial.ps1']
            $binding = [scriptblock]::Create($trialAst.ParamBlock.Extent.Text + "`n" +
                $trialAst.EndBlock.Statements[-1].Extent.Text + "`n" + '[pscustomobject]@{ AutomaticProfile = $PROFILE }')
            function Invoke-Trial($AgentProfile) { [pscustomobject]@{ Selected = $AgentProfile } }
            $automaticProfile = $PROFILE
            $aliasResult = @(& $binding -Profile 'ds-flash')
            $namedResult = @(& $binding -AgentProfile 'ds-flash')
            $aliasResult[0].Selected -eq 'ds-flash' -and $namedResult[0].Selected -eq 'ds-flash' -and
                $aliasResult[1].AutomaticProfile -eq $automaticProfile -and $namedResult[1].AutomaticProfile -eq $automaticProfile
        }
    } finally {
        # Only this test's freshly created temporary directory is removed.
        $resolved = [IO.Path]::GetFullPath($tempRoot)
        if ($resolved.StartsWith([IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\', [StringComparison]::OrdinalIgnoreCase)) { Remove-Item -LiteralPath $resolved -Recurse -Force }
    }
}
Write-Output "PASS $script:pass/$($script:pass + $script:fail)"
if ($script:fail) { exit 1 }
