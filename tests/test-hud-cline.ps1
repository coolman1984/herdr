# Offline tests for the HUD's Cline branch (model + fresh input/output tokens).
# Real temp session folders under $env:TEMP; -SourceRoot also runs them against HEAD copies.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot))
$ErrorActionPreference = 'Stop'
$script:pass = 0; $script:fail = 0
function Check([string]$Name, [scriptblock]$Test) {
    try { if (& $Test) { $script:pass++; return } } catch { Write-Output "  $($_.Exception.Message)" }
    $script:fail++; Write-Output "FAIL $Name"
}

# Import definitions only; never execute the HUD's initialization or main loop.
$errors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile(
    (Join-Path $SourceRoot 'herdr-hud.ps1'), [ref]$null, [ref]$errors)
if ($errors) { throw 'Cannot parse herdr-hud.ps1' }
foreach ($f in $ast.FindAll({ param($x)
    $x -is [System.Management.Automation.Language.FunctionDefinitionAst]
}, $true)) {
    . ([scriptblock]::Create($f.Extent.Text))
}

# Fake Cline session tree: <root>\<id>\<id>.json
$script:root = Join-Path $env:TEMP ("hud-cline-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $script:root -Force | Out-Null
$clineSessionRoots = @($script:root)          # the HUD reads this variable (test hook)
$panesState = Join-Path $script:root 'panes.json'   # absent: no started models, no accounts
$registry = [pscustomobject]@{ claude = [pscustomobject]@{ contextWindow = 200000 } }
$ttlMs = 120000; $source = 'herdr-hud'; $Once = $false
$last = @{}; $lastSent = @{}
$script:clineSessionCache = @{}

function New-Session([string]$Id, [int]$ProcId, [string]$Cwd, [string]$Status,
    [hashtable]$Usage, [string]$Model = 'cline-pass/deepseek-v4.1-flash', [int]$AgeMin = 0) {
    $dir = Join-Path $script:root $Id
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    $file = Join-Path $dir "$Id.json"
    $o = [ordered]@{ pid = $ProcId; cwd = $Cwd; model = $Model; status = $Status; started_at = '2026-10-23T10:00:00.000Z' }
    if ($Usage) { $o.metadata = @{ usage = $Usage } }
    ($o | ConvertTo-Json -Depth 6) | Set-Content -LiteralPath $file -Encoding utf8
    (Get-Item -LiteralPath $file).LastWriteTime = (Get-Date).AddMinutes(-$AgeMin)
    $file
}
function Reply($Result) {
    [pscustomobject]@{ Ok = $true; Text = 'ok'; Json = [pscustomobject]@{ result = $Result } }
}
function Invoke-Herdr([string[]]$Arguments) {
    $script:calls.Add(@($Arguments))
    switch ("$($Arguments[0]) $($Arguments[1])") {
        'workspace list' { return Reply ([pscustomobject]@{ workspaces = @([pscustomobject]@{ workspace_id = 'wR'; label = 'HERDR' }) }) }
        'agent list' { return Reply ([pscustomobject]@{ agents = @($script:agents) }) }
        'pane list' { return Reply ([pscustomobject]@{ panes = @($script:panes) }) }
        'pane process-info' { return $script:processReply }
    }
    Reply ([pscustomobject]@{})
}
# claude-limits.ps1 is not loaded here: the Claude side is not under test.
function Get-ClaudeAccountNames { @() }
function Get-ClaudeLimits { @() }
function Send-LimitNotices { param($Limits) }
function Get-ClaudeAccountOf { param($Dir, $Accounts) $null }
function Find-SessionFile { param($Kind, $Id) $null }
function Reset-Fixture {
    $script:clineSessionCache = @{}
    # $script: so the HUD's own $last/$lastSent (script scope) are really cleared: a plain
    # assignment here would only make a local copy and leave the dedupe state behind.
    $script:last = @{}; $script:lastSent = @{}; $script:agents = @(); $script:panes = @()
    $script:calls = [System.Collections.Generic.List[object]]::new()
    $script:processReply = [pscustomobject]@{ Ok = $false; Text = 'pane_not_found'; Json = $null }
    Get-ChildItem -LiteralPath $script:root -Directory | Remove-Item -Recurse -Force
}
function Call-Args([string]$Command) {
    $script:calls | Where-Object { $_[0] -eq 'pane' -and $_[1] -eq $Command }
}
function Token-Of([string[]]$Call, [string]$Key) {
    for ($i = 0; $i -lt $Call.Count - 1; $i++) {
        if ($Call[$i] -eq '--token' -and $Call[$i + 1] -like "$Key=*") { return $Call[$i + 1].Substring($Key.Length + 1) }
    }
    $null
}

try {
    Check 'C1 pid match wins over newer cwd match' {
        Reset-Fixture
        $newer = New-Session 'a' 111 'C:\other' 'running' @{ inputTokens = 10; outputTokens = 1 }
        $older = New-Session 'b' 222 'C:\work' 'running' @{ inputTokens = 20; outputTokens = 2 } -AgeMin 10
        (Find-ClineSession @(222) 'C:\other') -eq $older -and $newer -ne $older
    }
    Check 'C2 cwd fallback ignores case and trailing backslash' {
        Reset-Fixture
        $file = New-Session 'c' 999 'C:\Work\Proj\' 'running' @{ inputTokens = 10; outputTokens = 1 }
        (Find-ClineSession @(1) 'c:\work\proj') -eq $file
    }
    Check 'C3 exited sessions are not used by cwd fallback' {
        Reset-Fixture
        $exited = New-Session 'd' 1 'C:\x' 'exited' @{ inputTokens = 10; outputTokens = 1 }
        $live = New-Session 'e' 2 'C:\x' 'running' @{ inputTokens = 20; outputTokens = 2 } -AgeMin 10
        $done = New-Session 'f' 3 'C:\only' 'done' @{ inputTokens = 30; outputTokens = 3 }
        (Find-ClineSession @(9) 'C:\x') -eq $live -and $null -eq (Find-ClineSession @(9) 'C:\only') -and
            $exited -ne $live -and $done
    }
    Check 'C4 usage fresh = input minus cache read' {
        Reset-Fixture
        $file = New-Session 'g' 7 'C:\g' 'running' @{ inputTokens = 1000; outputTokens = 40; cacheReadTokens = 750; cacheWriteTokens = 0 }
        $u = Get-ClineUsage $file
        $u.fresh -eq 250 -and $u.out -eq 40 -and $u.model -eq 'cline-pass/deepseek-v4.1-flash'
    }
    Check 'C5 missing usage gives nulls, bad json gives null' {
        Reset-Fixture
        $noUsage = New-Session 'i' 7 'C:\i' 'running' $null
        $bad = Join-Path (Join-Path $script:root 'j') 'j.json'
        New-Item -ItemType Directory -Path (Join-Path $script:root 'j') -Force | Out-Null
        Set-Content -LiteralPath $bad -Value '{oops' -Encoding utf8
        $u = Get-ClineUsage $noUsage
        $null -eq $u.fresh -and $null -eq $u.out -and $u.model -eq 'cline-pass/deepseek-v4.1-flash' -and
            $null -eq (Get-ClineUsage $bad) -and $null -eq (Get-ClineUsage (Join-Path $script:root 'nope.json'))
    }
    Check 'C6 process-info failure gives empty pids' {
        Reset-Fixture
        $failed = @(Get-ClinePids 'wR:p1').Count
        $script:processReply = Reply ([pscustomobject]@{ process_info = [pscustomobject]@{
            foreground_processes = @([pscustomobject]@{ pid = 100 }, [pscustomobject]@{ pid = 101 }); shell_pid = 102 } })
        $pids = @(Get-ClinePids 'wR:p1')
        $call = @(Call-Args 'process-info')[0]
        $failed -eq 0 -and $pids.Count -eq 3 -and $pids -contains 102 -and
            ($call -join '|') -eq 'pane|process-info|--pane|wR:p1'
    }
    Check 'C7 Update-Panes adds in/out tokens and session model for a cline pane only' {
        Reset-Fixture
        $file = New-Session 'k' 222 'C:\work' 'running' @{ inputTokens = 1000; outputTokens = 40; cacheReadTokens = 750 }
        $script:agents = @([pscustomobject]@{ pane_id = 'wR:p1'; agent = 'cline' },
            [pscustomobject]@{ pane_id = 'wR:p2'; agent = 'codex' })
        $script:panes = @([pscustomobject]@{ pane_id = 'wR:p1'; cwd = 'C:\work'; workspace_id = 'wR' },
            [pscustomobject]@{ pane_id = 'wR:p2'; cwd = 'C:\other'; workspace_id = 'wR' })
        $script:processReply = Reply ([pscustomobject]@{ process_info = [pscustomobject]@{
            foreground_processes = @([pscustomobject]@{ pid = 222 }); shell_pid = 5 } })
        $null = Update-Panes
        $p1 = @(Call-Args 'report-metadata') | Where-Object { $_[2] -eq 'wR:p1' }
        $p2 = @(Call-Args 'report-metadata') | Where-Object { $_[2] -eq 'wR:p2' }
        $title = $p1[([array]::IndexOf($p1, '--title') + 1)]
        (Token-Of $p1 'model') -eq 'cline-pass/deepseek-v4.1-flash' -and (Token-Of $p1 'in') -eq '250' -and
            (Token-Of $p1 'out') -eq '40' -and $title -match 'in 250' -and $title -match 'out 40' -and
            $title -notmatch 'ctx' -and $null -eq (Token-Of $p2 'in') -and $null -eq (Token-Of $p2 'out')
    }
    Check 'C8 cache reused while pids unchanged' {
        Reset-Fixture
        $file = New-Session 'l' 222 'C:\work' 'running' @{ inputTokens = 10; outputTokens = 1 }
        $first = Find-ClineSession @(222) 'C:\work' 'wR:p1'
        $cached = $script:clineSessionCache['wR:p1']
        (Get-Item -LiteralPath $file).LastWriteTime = (Get-Date).AddDays(-30)
        $again = Find-ClineSession @(222) 'C:\work' 'wR:p1'
        $changed = Find-ClineSession @(333) 'C:\work' 'wR:p1'
        $first -eq $file -and $cached.File -eq $file -and "$($cached.Pids)" -eq '222' -and
            $again -eq $file -and $null -eq $changed
    }
    Check 'C9 completed and failed are skipped' {
        Reset-Fixture
        $completed = New-Session 'm' 1 'C:\x' 'completed' @{ inputTokens = 10; outputTokens = 1 }
        $failed = New-Session 'n' 2 'C:\x' 'failed' @{ inputTokens = 20; outputTokens = 2 }
        $live = New-Session 'o' 3 'C:\x' 'running' @{ inputTokens = 30; outputTokens = 3 } -AgeMin 10
        $deadOnly = New-Session 'p' 4 'C:\dead' 'completed' @{ inputTokens = 40; outputTokens = 4 }
        (Find-ClineSession @(9) 'C:\x') -eq $live -and $null -eq (Find-ClineSession @(9) 'C:\dead') -and
            $completed -and $failed -and $deadOnly
    }
    Check 'C10 newer live session in the same folder replaces the cached one' {
        Reset-Fixture
        $old = New-Session 'q' 222 'C:\work' 'running' @{ inputTokens = 10; outputTokens = 1 } -AgeMin 20
        $first = Find-ClineSession @(222) 'C:\work' 'wR:p1'
        $new = New-Session 'r' 222 'C:\work' 'running' @{ inputTokens = 20; outputTokens = 2 } -AgeMin 5
        $second = Find-ClineSession @(222) 'C:\work' 'wR:p1'
        $dead = New-Session 's' 222 'C:\work' 'completed' @{ inputTokens = 30; outputTokens = 3 }
        $third = Find-ClineSession @(222) 'C:\work' 'wR:p1'
        $first -eq $old -and $second -eq $new -and $third -eq $new -and $dead -ne $third
    }
    Check 'C11 in comes before out' {
        Reset-Fixture
        $file = New-Session 't' 222 'C:\work' 'running' @{ inputTokens = 1000; outputTokens = 40; cacheReadTokens = 750 }
        $script:agents = @([pscustomobject]@{ pane_id = 'wR:p1'; agent = 'cline' })
        $script:panes = @([pscustomobject]@{ pane_id = 'wR:p1'; cwd = 'C:\work'; workspace_id = 'wR' })
        $script:processReply = Reply ([pscustomobject]@{ process_info = [pscustomobject]@{
            foreground_processes = @([pscustomobject]@{ pid = 222 }); shell_pid = 5 } })
        $null = Update-Panes
        $p1 = @(Call-Args 'report-metadata') | Where-Object { $_[2] -eq 'wR:p1' }
        $in = [array]::IndexOf($p1, 'in=250'); $out = [array]::IndexOf($p1, 'out=40')
        $in -ge 0 -and $out -ge 0 -and $in -lt $out -and $file
    }
} finally {
    if (Test-Path -LiteralPath $script:root) { Remove-Item -LiteralPath $script:root -Recurse -Force }
}
Write-Output "PASS $script:pass/$($script:pass + $script:fail)"
if ($script:fail) { exit 1 }
