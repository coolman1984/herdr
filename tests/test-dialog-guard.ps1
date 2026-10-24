# Offline tests for the ask dialog guard (LESSONS #53). -SourceRoot also runs them against HEAD copies.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot))
$ErrorActionPreference = 'Stop'
$script:pass = 0; $script:fail = 0
function Check([string]$Name, [scriptblock]$Test) {
    try { if (& $Test) { $script:pass++; return } } catch { }
    $script:fail++; Write-Output "FAIL $Name"
}

# Import definitions only; never execute a script's initialization or main code.
$errors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile(
    (Join-Path $SourceRoot 'herdr-agents.ps1'), [ref]$null, [ref]$errors)
if ($errors) { throw "Cannot parse herdr-agents.ps1" }
foreach ($f in $ast.FindAll({ param($x)
    $x -is [System.Management.Automation.Language.FunctionDefinitionAst]
}, $true)) {
    . ([scriptblock]::Create($f.Extent.Text))
}

# The ask branch as a scriptblock; each exit code becomes a TEST_EXIT:<code> throw (as in test-batch-a).
$switch = $ast.Find({ param($x)
    $x -is [System.Management.Automation.Language.SwitchStatementAst]
}, $false)
$clause = $switch.Clauses | Where-Object { $_.Item1.Value -eq 'ask' }
$body = $clause.Item2.Extent.Text
$offset = $clause.Item2.Extent.StartOffset
foreach ($e in ($clause.Item2.FindAll({ param($x)
    $x -is [System.Management.Automation.Language.ExitStatementAst]
}, $true) | Sort-Object { $_.Extent.StartOffset } -Descending)) {
    $replacement = 'throw ("TEST_EXIT:" + [string](' + $e.Pipeline.Extent.Text + '))'
    $body = $body.Remove($e.Extent.StartOffset - $offset, $e.Extent.Text.Length).
        Insert($e.Extent.StartOffset - $offset, $replacement)
}
$script:ask = [scriptblock]::Create($body.Substring(1, $body.Length - 2))

# Real screens (LESSONS #53 and the dialog texts of the lead's spec). Footers: the real Antigravity and
# Claude footers; the Codex footer is assumed, not captured (see the answer file's doubts).
$navFooter = '↑/↓ Navigate · enter Confirm'
$claudeBrowser = @(
    'Claude Code v2.1.309',
    '',
    'Yes, use my browser',
    'Enter to confirm · Esc to keep browser tools off') -join "`n"
$antigravityTrust = @(
    'Antigravity CLI',
    '',
    'Do you trust the contents of this project?',
    '',
    '> Yes, I trust this folder',
    '  No, exit',
    '',
    $navFooter) -join "`n"
$modelPick = @(
    'Select a model',
    '',
    '> Haiku 5.5',
    '  Opus 5.5',
    '',
    $navFooter) -join "`n"
$codexTrust = @(
    'Trust this folder?',
    '',
    'Enter to confirm · Esc to cancel') -join "`n"
$claudeIdle = @(
    'Claude Code v2.1.309',
    '',
    '❯ Try "write a test for <filepath>"',
    '',
    '  trial/haiku-xhigh | Opus 5.5 | ctx 3%') -join "`n"
$codexIdle = @(
    '>_ OpenAI Codex (v0.163.0)',
    '',
    '› Ask Codex to do anything',
    '',
    '  gpt-6.1-sol medium  ·  C:\proj') -join "`n"
$oldAnswers = @(1..25 | ForEach-Object { "old answer line $_" })
# Real Claude answer that quotes "Trust this folder?" near the bottom (LESSONS #53 false positive).
$liveAnswer = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'fixture-claude-answer-mentions-trust.txt') -Raw

# Mocks: no Herdr, no files outside the script's fake folders.
function Fail([string]$Message) { throw "ERROR: $Message" }
function Get-Date { param([string]$Format)
    $now = [datetime]'2026-10-24T12:00:00'
    if ($Format) { return $now.ToString($Format) }
    $now
}
function Start-Sleep { param($Seconds) }
function New-Item { param($ItemType, [switch]$Force, $Path, $ErrorAction)
    if ($ItemType -eq 'File') {
        if ($script:files.ContainsKey($Path)) { throw [System.IO.IOException]::new('File exists') }
        $script:files[$Path] = ''
    }
}
function Set-Content { param($LiteralPath, $Value, $Encoding) $script:files[$LiteralPath] = $Value }
function Invoke-Herdr([string[]]$Arguments) {
    $script:calls.Add(@($Arguments))
    if ($Arguments[0] -eq 'agent' -and $Arguments[1] -eq 'read' -and $Arguments -contains 'visible' -and $script:readFails) {
        return [pscustomobject]@{ Ok = $false; Code = 1; Text = 'read failed'; Json = $null }
    }
    [pscustomobject]@{ Ok = $true; Code = 0; Text = $script:herdrScreen; Json = $null }
}
function Reset-Fixture([string]$Screen = '') {
    $script:calls = [System.Collections.Generic.List[object]]::new()
    $script:files = @{}; $script:herdrScreen = $Screen; $script:readFails = $false
    $script:runsPath = 'C:\fake\runs'
}
function Run-Branch([hashtable]$Values = @{}) {
    $Target = 'codex'; $Task = 'do task'; $TaskFile = ''; $Fresh = $false; $TimeoutSec = 20; $Lines = 200
    foreach ($k in $Values.Keys) { Set-Variable -Name $k -Value $Values[$k] }
    $printed = [System.Collections.Generic.List[string]]::new(); $problem = ''
    try { & $script:ask | ForEach-Object { $printed.Add([string]$_) } } catch { $problem = $_.Exception.Message }
    [pscustomobject]@{ Output = ($printed -join "`n"); Error = $problem }
}
function Prompts { @($script:calls | Where-Object { $_[1] -eq 'prompt' }).Count }

Check 'D1 real Antigravity trust dialog is found' {
    $d = Get-AgentDialog $antigravityTrust
    $d.Name -ceq 'trust' -and $d.Line -ceq 'Do you trust the contents of this project?'
}
Check 'D1 real Claude browser dialog is found' {
    $d = Get-AgentDialog $claudeBrowser
    $d.Name -ceq 'browser' -and $d.Line -ceq 'Yes, use my browser'
}
Check 'D1 real idle Claude and Codex screens are not dialogs' {
    $null -eq (Get-AgentDialog $claudeIdle) -and $null -eq (Get-AgentDialog $codexIdle)
}
Check 'D1 Codex trust prompt is found' {
    $d = Get-AgentDialog $codexTrust
    $d.Name -ceq 'trust' -and $d.Line -ceq 'Trust this folder?'
}
Check 'D1 the live answer that quotes Trust this folder? is not a dialog' {
    $liveAnswer.Contains('Trust this folder?') -and $null -eq (Get-AgentDialog $liveAnswer)
}
Check 'D2 sign-in wording is found' {
    foreach ($text in @('Not logged in', 'You are not signed in', 'Please log in to continue', 'Please sign in', 'Run /login to continue')) {
        $d = Get-AgentDialog $text
        if (-not $d -or $d.Name -cne 'sign-in') { return $false }
    }
    $true
}
Check 'D2 model pick wording is found' {
    foreach ($text in @('Select a model', 'Choose a model to continue', 'pick a model')) {
        $d = Get-AgentDialog "$text`n$navFooter"
        if (-not $d -or $d.Name -cne 'model') { return $false }
    }
    $true
}
Check 'D2 browser tools wording is found' {
    (Get-AgentDialog 'Esc to keep browser tools off').Name -ceq 'browser'
}
Check 'D2 trust wording is case-insensitive' {
    (Get-AgentDialog "DO YOU TRUST THIS FOLDER?`nENTER TO CONFIRM").Name -ceq 'trust'
}
Check 'D3 trust wins over sign-in when both show' {
    $d = Get-AgentDialog "Please log in`nDo you trust this folder?`n$navFooter"
    $d.Name -ceq 'trust' -and $d.Line -ceq 'Do you trust this folder?'
}
Check 'D3 browser wins over model' {
    (Get-AgentDialog "Select a model`nUse my browser`n$navFooter").Name -ceq 'browser'
}
Check 'D4 the matching line is trimmed' {
    (Get-AgentDialog "   Do you trust this folder?   `n   $navFooter   ").Line -ceq 'Do you trust this folder?'
}
Check 'D5 empty or null screen is no dialog' {
    $null -eq (Get-AgentDialog '') -and $null -eq (Get-AgentDialog $null) -and $null -eq (Get-AgentDialog "`n`n   `n")
}
Check 'D6 dialog text above the last 20 lines is ignored' {
    $null -eq (Get-AgentDialog ((@('Do you trust the contents of this project?') + $oldAnswers + @($navFooter)) -join "`n"))
}
Check 'D6 the last 20 non-empty lines still count' {
    $edge = @('Do you trust this folder?') + @($oldAnswers | Select-Object -First 18) + @($navFooter)
    $past = @('Do you trust this folder?') + @($oldAnswers | Select-Object -First 19) + @($navFooter)
    (Get-AgentDialog ($edge -join "`n")).Name -ceq 'trust' -and $null -eq (Get-AgentDialog ($past -join "`n"))
}
Check 'D6 blank padding is not counted as a line' {
    $inside = (@('Do you trust this folder?') + @($oldAnswers | Select-Object -First 18) + @($navFooter)) -join "`n"
    $outside = (@('Do you trust this folder?') + @($oldAnswers | Select-Object -First 19) + @($navFooter)) -join "`n"
    (Get-AgentDialog ($inside + ("`n" * 40))).Name -ceq 'trust' -and $null -eq (Get-AgentDialog ($outside + ("`n" * 40)))
}
Check 'D6 the footer must be in the last 6 non-empty lines' {
    $footerIn = (@('Do you trust this folder?', $navFooter) + @($oldAnswers | Select-Object -First 5)) -join "`n"
    $footerOut = (@('Do you trust this folder?', $navFooter) + @($oldAnswers | Select-Object -First 6)) -join "`n"
    (Get-AgentDialog ($footerIn + ("`n" * 40))).Name -ceq 'trust' -and $null -eq (Get-AgentDialog $footerOut)
}
Check 'D7 a dialog phrase without a footer is no dialog' {
    $null -eq (Get-AgentDialog 'Do you trust this folder?') -and $null -eq (Get-AgentDialog 'Use my browser') -and
        $null -eq (Get-AgentDialog 'Pick a model')
}
Check 'D8 sign-in must be in the last 6 non-empty lines' {
    $in = (@('Please log in') + @($oldAnswers | Select-Object -First 5)) -join "`n"
    $out = (@('Please log in') + @($oldAnswers | Select-Object -First 6)) -join "`n"
    (Get-AgentDialog $in).Name -ceq 'sign-in' -and $null -eq (Get-AgentDialog $out)
}

Check 'A1 a trust dialog blocks the ask before any prompt' {
    Reset-Fixture $antigravityTrust; $r = Run-Branch
    $lines = $r.Output -split "`n"
    $r.Error -eq 'TEST_EXIT:3' -and (Prompts) -eq 0 -and
        $lines[0] -ceq 'BLOCKED: codex shows a trust dialog ("Do you trust the contents of this project?") - the owner answers it in the pane.' -and
        $lines[1] -ceq '--- screen of codex ---' -and $r.Output.Contains('> Yes, I trust this folder') -and
        $r.Output.Contains('--- saved: ') -and @($script:files.Values | Where-Object { $_.Contains('# codex - BLOCKED') }).Count -eq 1
}
Check 'A2 every startup dialog kind blocks the ask' {
    foreach ($screen in @($claudeBrowser, 'Not logged in', $modelPick, $antigravityTrust, $codexTrust)) {
        Reset-Fixture $screen; $r = Run-Branch
        if ($r.Error -ne 'TEST_EXIT:3' -or (Prompts) -ne 0) { return $false }
    }
    $true
}
Check 'A3 a clean screen is read first, then the task is sent' {
    Reset-Fixture $claudeIdle; $r = Run-Branch
    $order = @($script:calls | ForEach-Object { if ($_ -contains 'visible') { 'screen' } else { $_[1] } })
    $r.Error -eq 'TEST_EXIT:0' -and (Prompts) -eq 1 -and $order[0] -eq 'screen' -and
        [array]::IndexOf($order, 'screen') -lt [array]::IndexOf($order, 'prompt')
}
Check 'A4 a failed screen read does not stop the ask' {
    Reset-Fixture $antigravityTrust; $script:readFails = $true; $r = Run-Branch
    $r.Error -eq 'TEST_EXIT:0' -and (Prompts) -eq 1
}
Check 'A5 old dialog text above the window does not block the ask' {
    Reset-Fixture ((@('Do you trust the contents of this project?') + $oldAnswers) -join "`n"); $r = Run-Branch
    $r.Error -eq 'TEST_EXIT:0' -and (Prompts) -eq 1
}
Check 'A6 -Fresh sends nothing into a dialog' {
    Reset-Fixture $antigravityTrust; $r = Run-Branch @{ Fresh = $true }
    $r.Error -eq 'TEST_EXIT:3' -and (Prompts) -eq 0 -and @($script:calls | Where-Object { $_[1] -eq 'get' }).Count -eq 0
}
Check 'A7 the live answer that quotes Trust this folder? does not block the ask' {
    Reset-Fixture $liveAnswer; $r = Run-Branch
    $r.Error -eq 'TEST_EXIT:0' -and (Prompts) -eq 1
}

Write-Output "PASS $script:pass/$($script:pass + $script:fail)"
if ($script:fail) { exit 1 }
