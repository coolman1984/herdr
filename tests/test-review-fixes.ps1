# Offline behavioural regressions. Definitions and command branches only; no live Herdr.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot))
$ErrorActionPreference = 'Stop'
$script:passed = 0; $script:failed = 0
$asts = @{}
foreach ($file in 'check.ps1','sync-skills.ps1','herdr-agents.ps1','herdr-hud.ps1','herdr-stop.ps1','herdr-trial.ps1','herdr-open-folder.ps1') {
    $parseErrors = $null
    $ast = [Management.Automation.Language.Parser]::ParseFile((Join-Path $SourceRoot $file), [ref]$null, [ref]$parseErrors)
    if ($parseErrors) { throw "Parse error in ${file}: $parseErrors" }
    $asts[$file] = $ast
}
# String.Replace has no count overload; use a regex for the scope prefix.
function Import-Functions([string]$File, [string[]]$Names) {
    foreach ($f in $asts[$File].FindAll({ param($x) $x -is [Management.Automation.Language.FunctionDefinitionAst] }, $true)) {
        if ($f.Name -in $Names) { . ([scriptblock]::Create(([regex]::new('^function ')).Replace($f.Extent.Text.Replace('$PSScriptRoot', "'" + $SourceRoot.Replace("'", "''") + "'"), 'function script:', 1))) }
    }
}
function Check([string]$Name, [scriptblock]$Test) {
    try { $result = @(& $Test); if ($result.Count -eq 1 -and $result[0] -is [bool] -and $result[0]) { $script:passed++; Write-Output "PASS $Name"; return } }
    catch { Write-Output "  $($_.Exception.Message)" }
    $script:failed++; Write-Output "FAIL $Name"
}
function Fail([string]$Message) { throw $Message }
$fixture = Join-Path $PSScriptRoot ('.review-' + [guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($fixture) | Out-Null
$previousHome = $env:USERPROFILE; $previousAppData = $env:APPDATA; $previousSocket = $env:HERDR_SOCKET_PATH
try {
    $env:USERPROFILE = $fixture; $env:APPDATA = $fixture; $env:HERDR_SOCKET_PATH = ''
    Import-Functions 'check.ps1' @('Test-KeyTarget','Test-Shortcut')
    Check 'missing key script fails' {
        $script:bad = @(); function Bad($m) { $script:bad += $m }; function Ok { }
        Test-KeyTarget 'pwsh -File "C:\missing\file.ps1"' 'key'
        $bad.Count -eq 1 -and $bad[0] -like '*target missing*'
    }
    Check 'existing key script with spaces succeeds' {
        $file = Join-Path $fixture 'space name.ps1'; Set-Content -LiteralPath $file -Value '# fixture'
        $script:ok = @(); function Ok($m) { $script:ok += $m }; function Bad { throw 'unexpected failure' }
        Test-KeyTarget "pwsh -File `"$file`" -Once" 'key'
        $ok.Count -eq 1
    }
    Check 'shortcut existing link with absent destination warns' {
        function Test-Path { param($LiteralPath,$PathType) $LiteralPath -eq 'link.lnk' }
        function New-Object { [pscustomobject]@{} | Add-Member ScriptMethod CreateShortcut { param($p) [pscustomobject]@{TargetPath='missing.exe'} } -PassThru }
        $script:warnings = @(); function Warn($m) { $script:warnings += $m }; function Ok { }
        Test-Shortcut 'link.lnk'
        $warnings.Count -eq 1 -and $warnings[0] -like '*target missing*'
    }
    Import-Functions 'sync-skills.ps1' @('Get-SkillTargets')
    Check 'account with missing skills is included by copy list' {
        $PSScriptRoot = $SourceRoot
        [IO.Directory]::CreateDirectory((Join-Path $fixture '.herdr-accounts\claude\1984')) | Out-Null
        $targets = @(Get-SkillTargets)
        $targets -contains (Join-Path $fixture '.herdr-accounts\claude\1984\skills')
    }
    Check 'account skill drift is reported without writing' {
        $PSScriptRoot = $SourceRoot
        $body = $asts['sync-skills.ps1'].EndBlock.Statements | Where-Object { $_ -is [Management.Automation.Language.ForEachStatementAst] } | Select-Object -Last 1
        # Execute the actual copy/check loop with fixture destinations only.
        $targets = @((Join-Path $fixture '.claude\skills'), (Join-Path $fixture '.herdr-accounts\claude\1984\skills'))
        $source = Join-Path $SourceRoot 'skills'; $Check = $true; $drift = 0
        $out = @(& ([scriptblock]::Create($body.Extent.Text)))
        @($out | Where-Object { $_ -like 'DIFFERS*1984*' }).Count -eq 2 -and -not (Test-Path -LiteralPath $targets[1])
    }
    Import-Functions 'herdr-agents.ps1' @('Get-State','Get-Account','Complete-Ask')
    Check 'invalid ownership JSON aborts rather than returning empty' {
        $statePath = Join-Path $fixture 'state.json'; Set-Content -LiteralPath $statePath -Value '{broken'
        try { Get-State; $false } catch { $_.Exception.Message -like '*ownership state*' }
    }
    Check 'uppercase account rejected before creating folder' {
        $accountsRoot = Join-Path $fixture 'accounts'
        $entry = [pscustomobject]@{account=[pscustomobject]@{tool='claude';copy=@();env=[pscustomobject]@{};args=@();login='login'}}
        try { Get-Account $entry 'Upper'; $false } catch { -not (Test-Path -LiteralPath (Join-Path $accountsRoot 'claude\Upper')) }
    }
    Check 'collect settled result skips prompt-start polling' {
        $TimeoutSec = 10; $Lines = 20
        function Start-Sleep { throw 'unexpected poll' }
        function Invoke-Herdr($Arguments) { [pscustomobject]@{Ok=$true;Text='finished answer';Json=$null} }
        function Format-Screen($Text) { $Text }
        $r = [pscustomobject]@{Ok=$true;Json=@{result=@{agent=@{agent_status='done'}}}}
        $answer = Complete-Ask 'done-agent' $r -Collected
        $answer.Outcome -eq 'DONE' -and $answer.Text -like '*finished answer*'
    }
    Import-Functions 'herdr-hud.ps1' @('ConvertTo-HudServer','Get-PaneRecords','Get-PaneAccounts','Get-StartedModels','Write-HudError','Update-Panes')
    Check 'HUD duplicate pane id uses current server model' {
        $script:panesState = Join-Path $fixture 'panes.json'
        @(@{pane='p1';model='default-model';account='2030';server='default'},@{pane='p1';model='lab-model';account='1984';server='lab.sock'}) | ConvertTo-Json | Set-Content -LiteralPath $panesState
        $env:HERDR_SOCKET_PATH = ''
        (Get-StartedModels)['p1'] -eq 'default-model' -and (Get-PaneAccounts)['p1'] -eq '2030'
    }
    Check 'HUD normalizes default socket for old records' {
        $env:HERDR_SOCKET_PATH = Join-Path $env:APPDATA 'herdr\herdr.sock'
        (Get-StartedModels)['p1'] -eq 'default-model'
    }
    Check 'HUD errors reach diagnostic file' {
        Write-HudError 'fixture failure'
        (Get-Content -LiteralPath (Join-Path $fixture 'herdr\hud-errors.log') -Raw) -like '*fixture failure*'
    }
    Check 'HUD Once prints without metadata writes' {
        $Once = $true; $last = @{}; $lastSent = @{}; $ttlMs = 120000
        $script:sent = @()
        function Invoke-Herdr($Arguments) {
            $script:sent += $Arguments -join '|'
            $result = switch ($Arguments[0]) {
                'workspace' { @{workspaces=@(@{workspace_id='w1';label='test'})} }
                'agent' { @{agents=@()} }
                'pane' { @{panes=@(@{pane_id='p1';workspace_id='w1';cwd='C:\test'})} }
            }
            [pscustomobject]@{Ok=$true;Json=@{result=$result}}
        }
        function Get-PaneAccounts { @{} }; function Get-StartedModels { @{} }
        function Get-ClaudeAccountNames { @() }; function Get-ClaudeLimits { @() }; function Send-LimitNotices { }
        function Format-Path($Path) { $Path }
        $printed = @(Update-Panes)
        ($printed -join '') -like '*test*' -and @($sent | Where-Object { $_ -like '*report-metadata*' }).Count -eq 0
    }
    Check 'HUD discovery failure reaches caller' {
        function Invoke-Herdr { [pscustomobject]@{Ok=$false;Text='discovery failure'} }
        try { Update-Panes | Out-Null; $false } catch { $_.Exception.Message -like '*discovery failure*' }
    }
    Check 'HUD metadata failure reaches caller' {
        $Once = $false; $last = @{}; $lastSent = @{}; $ttlMs = 120000
        function Invoke-Herdr($Arguments) {
            if ($Arguments[1] -eq 'report-metadata') { return [pscustomobject]@{Ok=$false;Text='metadata denied'} }
            $r = switch ($Arguments[0]) {
                'workspace' { @{workspaces=@(@{workspace_id='w1';label='test'})} }
                'agent' { @{agents=@()} }
                'pane' { @{panes=@(@{pane_id='p1';workspace_id='w1';cwd='C:\test'})} }
            }
            [pscustomobject]@{Ok=$true;Json=@{result=$r}}
        }
        function Get-PaneAccounts { @{} }; function Get-StartedModels { @{} }
        function Get-ClaudeAccountNames { @() }; function Get-ClaudeLimits { @() }; function Send-LimitNotices { }
        function Format-Path($Path) { $Path }
        try { Update-Panes | Out-Null; $false } catch { $_.Exception.Message -like '*metadata denied*' }
    }
    Import-Functions 'herdr-stop.ps1' @('Stop-HerdrServer')
    Check 'failed stop is reported' {
        $herdr = 'Fake-Stop'; function Fake-Stop { $global:LASTEXITCODE = 1; 'stop denied' }
        try { Stop-HerdrServer; $false } catch { $_.Exception.Message -like '*stop denied*' }
    }
    Check 'successful stop must read stopped status' {
        $herdr = 'Fake-Stop'; $script:stopCalls = @()
        function Fake-Stop { $script:stopCalls += $args -join '|'; $global:LASTEXITCODE=0; '{"status":"not_running"}' }
        Stop-HerdrServer | Out-Null
        ($stopCalls -join ',') -eq 'server|stop,status|server|--json'
    }
    Import-Functions 'herdr-trial.ps1' @('Wait-TrialReport','Remove-TrialAgent')
    Check 'blocked trial stops waiting immediately' {
        function Invoke-Herdr { [pscustomobject]@{Ok=$true;Json=@{result=@{agent=@{agent_status='blocked'}}}} }
        function Start-Sleep { throw 'must not wait' }
        try { Wait-TrialReport $fixture 60 15 'trial-1'; $false } catch { $_.Exception.Message -like '*cannot continue*' }
    }
    Check 'missing trial agent stops waiting immediately' {
        function Invoke-Herdr { [pscustomobject]@{Ok=$false;Text='agent gone'} }
        function Start-Sleep { throw 'must not wait' }
        try { Wait-TrialReport $fixture 60 15 'trial-1'; $false } catch { $_.Exception.Message -like '*gone or unreadable*' }
    }
    Check 'trial cleanup stops only its recorded name' {
        $script:trialAgent='trial-own'; $script:cleanup=@()
        function Invoke-TrialProcess($File,$Arguments) { $script:cleanup=$Arguments; [pscustomobject]@{Code=0;Text='STOPPED'} }
        Remove-TrialAgent
        $cleanup[-2] -eq 'stop' -and $cleanup[-1] -eq 'trial-own' -and -not $script:trialAgent
    }
    Import-Functions 'herdr-open-folder.ps1' @('Start-FolderAgent')
    Check 'folder start uses profile command with pane' {
        $PSScriptRoot = $SourceRoot; $script:profileCall=@()
        function pwsh { $script:profileCall=$args; $global:LASTEXITCODE=0; 'STARTED' }
        (Start-FolderAgent 'claude' 'p7') -and ($profileCall -join '|') -like '*herdr-agents.ps1|start|claude|-Pane|p7'
    }
    Check 'folder start failure is surfaced' {
        $PSScriptRoot = $SourceRoot; $script:shown=''
        function pwsh { $global:LASTEXITCODE=2; 'NOT READY' }
        function Show-Error($Message) { $script:shown=$Message }
        -not (Start-FolderAgent 'codex' 'p8') -and $shown -like '*NOT READY*'
    }
    Check 'custom build uses existing tracked source folder' {
        $buildAst = [Management.Automation.Language.Parser]::ParseFile((Join-Path $SourceRoot 'install-custom-build.ps1'), [ref]$null, [ref]$null)
        $assignment = $buildAst.EndBlock.Statements | Where-Object { $_ -is [Management.Automation.Language.AssignmentStatementAst] -and $_.Left.Extent.Text -eq '$source' } | Select-Object -First 1
        $PSScriptRoot = $SourceRoot
        . ([scriptblock]::Create($assignment.Extent.Text.Replace('$PSScriptRoot', "'" + $SourceRoot.Replace("'", "''") + "'")))
        $source -eq (Join-Path $SourceRoot 'herdr-app')
    }
    Check 'invalid agent name cannot create account folder' {
        $branch = $asts['herdr-agents.ps1'].Find({param($x) $x -is [Management.Automation.Language.SwitchStatementAst]}, $false).Clauses | Where-Object { $_.Item1.Value -eq 'start' }
        $text = $branch.Item2.Extent.Text
        $start = [scriptblock]::Create($text.Substring(1, $text.Length - 2))
        $registryPath = Join-Path $SourceRoot 'agents.json'; $accountsRoot = Join-Path $fixture 'invalid-start'
        $Target='claude'; $Name='Invalid'; $Account='valid'; $Pane='p1'; $Cwd=$fixture; $Worktree=''
        function Get-LiveAgents { @() }; function Get-State { @() }
        function Get-Registry { (Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json).agents }
        function Get-Command { $true }
        try { & $start | Out-Null; $false } catch {
            $_.Exception.Message -like '*Name must match*' -and -not (Test-Path -LiteralPath (Join-Path $accountsRoot 'claude\valid'))
        }
    }
    Check 'account setup failure prevents agent start' {
        $branch = $asts['herdr-agents.ps1'].Find({param($x) $x -is [Management.Automation.Language.SwitchStatementAst]}, $false).Clauses | Where-Object { $_.Item1.Value -eq 'start' }
        $envClause = $branch.Item2.FindAll({param($x) $x -is [Management.Automation.Language.IfStatementAst] -and $x.Clauses[0].Item1.Extent.Text -eq '$set'}, $true) | Select-Object -First 1
        $set='set env'; $pane='p1'; $script:configuredCalls=0
        function Invoke-Herdr { $script:configuredCalls++; [pscustomobject]@{Ok=$false;Text='environment denied'} }
        function Start-Sleep { }
        try { & ([scriptblock]::Create($envClause.Extent.Text)) | Out-Null; $false }
        catch { $_.Exception.Message -like '*environment denied*' -and $configuredCalls -eq 1 }
    }
    Import-Functions 'herdr-agents.ps1' @('Get-Registry')
    Check 'bad registry does not relabel main account' {
        function Get-Content { throw 'bad registry' }
        try { Get-Registry; $false } catch { $_.Exception.Message -like '*Cannot read agents.json*' }
    }
} finally {
    $env:USERPROFILE=$previousHome; $env:APPDATA=$previousAppData; $env:HERDR_SOCKET_PATH=$previousSocket
    $resolved = [IO.Path]::GetFullPath($fixture)
    if ($resolved.StartsWith([IO.Path]::GetFullPath($PSScriptRoot).TrimEnd('\') + '\.review-', [StringComparison]::OrdinalIgnoreCase)) { Remove-Item -LiteralPath $resolved -Recurse -Force }
}
Write-Output "PASS $script:passed/$($script:passed + $script:failed)"
if ($script:failed) { exit 1 }
