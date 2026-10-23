# Offline regressions for herdr-headless.ps1 (no live Cline run). -SourceRoot runs the same
# fixtures against another copy of the script, e.g. a HEAD copy in a temp folder.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot))
$ErrorActionPreference = 'Stop'
$script:pass = 0; $script:fail = 0
function Check([string]$Name, [scriptblock]$Test) {
    if ($script:missing) { $script:fail++; Write-Output "FAIL $Name"; return }
    try { if (& $Test) { $script:pass++; return } } catch { Write-Output "  $($_.Exception.Message)" }
    $script:fail++; Write-Output "FAIL $Name"
}

# Import definitions only; never execute the script's initialization or main code.
$script:missing = $false
$headlessPath = Join-Path $SourceRoot 'herdr-headless.ps1'
if (-not (Test-Path -LiteralPath $headlessPath)) {
    $script:missing = $true
    Write-Output "herdr-headless.ps1 not found in $SourceRoot - every check fails."
} else {
    $errors = $null
    $ast = [System.Management.Automation.Language.Parser]::ParseFile($headlessPath, [ref]$null, [ref]$errors)
    if ($errors) { throw "Cannot parse herdr-headless.ps1: $errors" }
    foreach ($f in $ast.FindAll({ param($x)
        $x -is [System.Management.Automation.Language.FunctionDefinitionAst]
    }, $false)) {
        # Top level only: a definition nested in another function would not exist at run time.
        . ([scriptblock]::Create($f.Extent.Text))
    }
}
# The real Fail ends the host; inside the harness it must only end the one check.
function Fail([string]$Message) { throw "ERROR: $Message" }

# The real JSON Lines shapes (LESSONS #47, Cline 3.0.73).
$doneLine = '{"ts":"2026-10-23T12:00:00.000Z","type":"agent_event","event":{"type":"done","reason":"completed","text":"the final answer"}}'
$resultLine = '{"ts":"2026-10-23T12:00:00.500Z","type":"run_result","finishReason":"completed","iterations":4,"usage":{"inputTokens":50180,"outputTokens":555,"cacheReadTokens":36366,"cacheWriteTokens":0,"totalCost":0},"aggregateUsage":{"inputTokens":50180,"outputTokens":555}}'
$clineSpec = [pscustomobject]@{
    args = @('--tui', '--auto-approve', 'true'); providerFlag = '-P'; provider = 'cline-pass'
    modelFlag = '-m'; model = 'cline-pass/deepseek-v4.1-flash'
    effortArgs = @('--thinking', '{0}'); effort = 'high'
}

Check 'H1 args drop --tui and add json cwd timeout' {
    $arguments = @(Get-HeadlessArgs $clineSpec '' '' 'C:\work' 60 @())
    ($arguments -join '|') -ceq '--auto-approve|true|-P|cline-pass|-m|cline-pass/deepseek-v4.1-flash|--thinking|high|--json|-c|C:\work|-t|60'
}
Check 'H2 profile effort is default and -Effort overrides' {
    $default = (@(Get-HeadlessArgs $clineSpec '' '' 'C:\work' 60 @()) -join '|')
    $override = (@(Get-HeadlessArgs $clineSpec '' 'low' 'C:\work' 60 @()) -join '|')
    $default.Contains('--thinking|high') -and $override.Contains('--thinking|low') -and -not $override.Contains('high')
}
Check 'H3 -Model overrides profile model' {
    $default = (@(Get-HeadlessArgs $clineSpec '' '' 'C:\work' 60 @()) -join '|')
    $override = (@(Get-HeadlessArgs $clineSpec 'cline-pass/kimi-k3' '' 'C:\work' 60 @()) -join '|')
    $default.Contains('-m|cline-pass/deepseek-v4.1-flash') -and $override.Contains('-m|cline-pass/kimi-k3') -and
        -not $override.Contains('deepseek-v4.1-flash')
}
Check 'H4 result parsed from run_result and done event' {
    $result = Read-HeadlessResult @($doneLine, $resultLine)
    $result.Finish -ceq 'completed' -and $result.Text -ceq 'the final answer' -and
        $result.Fresh -eq 13814 -and $result.Out -eq 555 -and $result.Iterations -eq 4
}
Check 'H5 non-JSON lines skipped' {
    $result = Read-HeadlessResult @('Cline 3.0.73 starting', $doneLine, '{ not json', '', 'more noise', $resultLine, 'bye')
    $result.Finish -ceq 'completed' -and $result.Text -ceq 'the final answer' -and $result.Fresh -eq 13814 -and $result.Out -eq 555
}
Check 'H6 missing run_result gives Finish null' {
    $result = Read-HeadlessResult @('noise', $doneLine)
    $null -eq $result.Finish -and $result.Text -ceq 'the final answer' -and $null -eq $result.Fresh -and $null -eq $result.Out
}
Check 'H7 non-cline profile refused' {
    $registry = [pscustomobject]@{ agents = [pscustomobject]@{
        'ds-flash' = [pscustomobject]@{ kind = 'cline'; model = 'cline-pass/deepseek-v4.1-flash' }
        'codex'    = [pscustomobject]@{ kind = 'codex' }
    } }
    $refused = ''
    try { Get-HeadlessProfile $registry 'codex' | Out-Null } catch { $refused = $_.Exception.Message }
    $unknown = ''
    try { Get-HeadlessProfile $registry 'nope' | Out-Null } catch { $unknown = $_.Exception.Message }
    $refused -ceq 'ERROR: herdr-headless runs Cline profiles only: codex' -and
        $unknown -ceq 'ERROR: Unknown profile: nope (see: herdr-agents tools)' -and
        (Get-HeadlessProfile $registry 'ds-flash').kind -ceq 'cline'
}
Check 'H8 task file prompt uses the full path' {
    $root = Join-Path (Resolve-Path -LiteralPath $env:TEMP).ProviderPath ('herdr-headless-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $root | Out-Null
    try {
        Set-Content -LiteralPath (Join-Path $root 'task.md') -Value '# Task' -Encoding utf8
        Push-Location $root
        try { $prompt = Get-HeadlessPrompt '' 'task.md' } finally { Pop-Location }
        $missing = ''
        try { Get-HeadlessPrompt '' (Join-Path $root 'nope.md') | Out-Null } catch { $missing = $_.Exception.Message }
        $prompt -eq "Read $(Join-Path $root 'task.md') completely and do exactly what it says. Reply in its answer format." -and
            $missing -eq "ERROR: Task file not found: $(Join-Path $root 'nope.md')" -and
            (Get-HeadlessPrompt 'just do it' '') -ceq 'just do it'
    } finally { Remove-Item -LiteralPath $root -Recurse -Force }
}
Check 'H9 main flow adds account args and reports DONE' {
    $registryJson = '{"agents":{"ds-flash":{"kind":"cline","command":"cline","args":["--tui","--auto-approve","true"],"providerFlag":"-P","provider":"cline-pass","modelFlag":"-m","model":"cline-pass/deepseek-v4.1-flash","effortArgs":["--thinking","{0}"],"effort":"high","account":{"tool":"cline","env":{},"args":["--config","{dir}","--data-dir","{dir}\\data"]}}}}'
    function Get-Content { param($LiteralPath, [switch]$Raw) $registryJson }
    function Get-HeadlessCommand { param($Command)
        [pscustomobject]@{ File = 'C:\fake\cline.ps1'; Prefix = @('-NoProfile', '-File', 'C:\fake\cline.ps1') } }
    function New-HeadlessLogPath { param($Profile, $Now) 'C:\fake\headless.jsonl' }
    function Get-HeadlessAccount { param($Entry, $AccountName)
        [pscustomobject]@{ Name = $AccountName; Tool = 'cline'; Dir = 'C:\acc\cline\work1'; Env = @{}
            Args = @('--config', 'C:\acc\cline\work1', '--data-dir', 'C:\acc\cline\work1\data') } }
    function Start-HeadlessRun { param($File, $Arguments, $LogPath, $TimeoutSec, $Environment)
        $script:ran = [pscustomobject]@{ File = $File; Arguments = $Arguments; Log = $LogPath; Timeout = $TimeoutSec; Environment = $Environment }
        [pscustomobject]@{ Code = 0; TimedOut = $false; Lines = @(); Log = $LogPath; Seconds = 7 } }
    function Read-HeadlessResult { param($Lines)
        [pscustomobject]@{ Finish = 'completed'; Text = 'ok'; Fresh = 11; Out = 22; Iterations = 1 } }
    $script:ran = $null
    # The main body runs with every side effect mocked; its exits become throws.
    $lastFunction = @($ast.EndBlock.Statements | Where-Object {
        $_ -is [System.Management.Automation.Language.FunctionDefinitionAst]
    }) | Select-Object -Last 1
    $offset = $lastFunction.Extent.EndOffset
    $body = $ast.Extent.Text.Substring($offset)
    $exits = @($ast.EndBlock.FindAll({ param($y)
        $y -is [System.Management.Automation.Language.ExitStatementAst]
    }, $true) | Where-Object { $_.Extent.StartOffset -ge $offset } | Sort-Object { $_.Extent.StartOffset } -Descending)
    foreach ($x in $exits) {
        $body = $body.Remove($x.Extent.StartOffset - $offset, $x.Extent.Text.Length).
            Insert($x.Extent.StartOffset - $offset, 'throw ("TEST_EXIT:" + [string](' + $x.Pipeline.Extent.Text + '))')
    }
    $main = [scriptblock]::Create($ast.ParamBlock.Extent.Text + "`n" + $body)
    $registryPath = 'C:\fake\agents.json'
    $stopped = ''
    try { & $main -Profile ds-flash -Account work1 -Task 'x' | Out-Null } catch { $stopped = $_.Exception.Message }
    $ran = $script:ran
    $stopped -eq 'TEST_EXIT:0' -and $null -ne $ran -and $ran.File -eq 'C:\fake\cline.ps1' -and
        ($ran.Arguments -join '|').Contains('--config|C:\acc\cline\work1|--data-dir|C:\acc\cline\work1\data|') -and
        $ran.Arguments[-1] -eq 'x'
}
Write-Output "PASS $script:pass/$($script:pass + $script:fail)"
if ($script:fail) { exit 1 }
