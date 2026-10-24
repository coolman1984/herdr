# Offline fixtures only. Use -SourceRoot for temporary HEAD copies of the sources/binary.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot))
$ErrorActionPreference = 'Stop'
$script:pass = 0; $script:fail = 0
function Check([string]$Name, [scriptblock]$Test) {
    try {
        $result = @(& $Test)
        if ($result.Count -eq 1 -and $result[0] -is [bool] -and $result[0]) { $script:pass++; return }
    } catch { Write-Output "  $($_.Exception.Message)" }
    $script:fail++; Write-Output "FAIL $Name"
}
$script:asts = @{}
foreach ($file in 'install.ps1', 'herdr-spaces.ps1', 'herdr-open-folder.ps1', 'herdr-stop.ps1') {
    $errors = $null
    $ast = [System.Management.Automation.Language.Parser]::ParseFile(
        (Join-Path $SourceRoot $file), [ref]$null, [ref]$errors)
    if ($errors) { throw "Cannot parse $file" }
    $script:asts[$file] = $ast
    foreach ($f in $ast.FindAll({ param($x)
        $x -is [System.Management.Automation.Language.FunctionDefinitionAst]
    }, $true)) {
        if ($f.Name -in 'Update-PathRegistryEntry','Prepend-PathEntry','Remove-OldReleases',
            'Join-Spaces','Split-Spaces','Get-Running') {
            # Only the registry parameter type is substituted, allowing a fake key;
            # no real registry key is opened or created.
            . ([scriptblock]::Create($f.Extent.Text.Replace('[Microsoft.Win32.RegistryKey]$EnvironmentKey', '[object]$EnvironmentKey')))
        }
    }
}
$teamAst = $script:asts['herdr-open-folder.ps1'].FindAll({ param($x)
    $x -is [System.Management.Automation.Language.IfStatementAst] -and
        $x.Clauses[0].Item1.Extent.Text -eq '$Agent -eq "team"'
}, $true) | Select-Object -Last 1
$team = [scriptblock]::Create('param($PSScriptRoot)' + "`n" + $teamAst.Clauses[0].Item2.Extent.Text.Trim().TrimStart('{').TrimEnd('}'))
$stopText = $script:asts['herdr-stop.ps1'].Extent.Text
$statusStart = $stopText.IndexOf('if ($Status)')
$statusEnd = $stopText.IndexOf('Add-Type', $statusStart)
$statusBranch = [scriptblock]::Create($stopText.Substring($statusStart, $statusEnd - $statusStart).Replace('exit 0', 'return'))
$cmdText = Microsoft.PowerShell.Management\Get-Content -LiteralPath (Join-Path $SourceRoot 'Herdr.cmd') -Raw
$installCmd = Microsoft.PowerShell.Management\Get-Content -LiteralPath (Join-Path $SourceRoot 'install.cmd') -Raw
$launcherBytes = [IO.File]::ReadAllBytes((Join-Path $SourceRoot 'launcher/HerdrLauncher.exe'))
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class BatchBArgv {
    [DllImport("shell32.dll", CharSet=CharSet.Unicode)]
    static extern IntPtr CommandLineToArgvW(string command, out int count);
    [DllImport("kernel32.dll")] static extern IntPtr LocalFree(IntPtr p);
    public static string[] Split(string command) {
        int count; IntPtr p = CommandLineToArgvW(command, out count);
        if (p == IntPtr.Zero) throw new Exception("CommandLineToArgvW failed");
        try {
            string[] result = new string[count];
            for (int i=0;i<count;i++) result[i] = Marshal.PtrToStringUni(Marshal.ReadIntPtr(p, i*IntPtr.Size));
            return result;
        } finally { LocalFree(p); }
    }
}
'@
function Reset-Fixture {
    $script:files = @{}; $script:events = @(); $script:children = @()
    $script:backupFails = $false; $script:deleted = @(); $script:messages = @()
    $script:calls = [Collections.Generic.List[object]]::new()
    $script:listFails = $false; $script:moveFails = @(); $script:getFails = $false
    $script:recordPath = 'C:\fake\record.json'; $script:noWindow = $true
    $script:root = 'C:\fake'; $script:PSScriptRoot = 'C:\fake'
    $script:herdr = 'Fake-Herdr'; $script:n = 0; $script:ratios = @()
    $script:profileJson = '{"agents":{"codex":{"modelFlag":"-m","model":"chosen-model","args":["SHOULD-NOT-PASS"]}}}'
    $script:runningOk = $true; $script:runningAgents = @()
    $script:key = [pscustomobject]@{ Raw = '%USERPROFILE%\bin;C:\old'; Kind = [Microsoft.Win32.RegistryValueKind]::ExpandString }
    $script:key | Add-Member ScriptMethod GetValue { param($name,$default,$options)
        if ($options -ne [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) { throw 'expanded' }; $this.Raw }
    $script:key | Add-Member ScriptMethod GetValueKind { param($name) $this.Kind }
    $script:key | Add-Member ScriptMethod SetValue { param($name,$value,$kind)
        $script:events += 'SetValue'; $script:newPath = $value; $script:newKind = $kind }
}
function Test-Path { param($LiteralPath, $Path, $PathType) $true }
function Get-Date { param($Format)
    $now = [datetime]'2026-10-23T12:00:00'
    if ($Format) { return $now.ToString($Format) }; $now
}
function Get-Content { param($LiteralPath, [switch]$Raw)
    if ($LiteralPath -like '*agents.json') { return $script:profileJson }; $script:files[$LiteralPath] }
function Get-ChildItem { param($LiteralPath, [switch]$Force, [switch]$Directory, $ErrorAction) $script:children }
function New-Item { param($Path, $ItemType, [switch]$Force, $ErrorAction)
    if ($ItemType -eq 'File') {
        if ($script:files.ContainsKey($Path)) { throw [IO.IOException]::new('exists') }
        $script:files[$Path] = ''
    }
}
function Set-Content {
    [CmdletBinding()]
    param($LiteralPath, [Parameter(ValueFromPipeline)]$Value, $Encoding)
    process {
    if ($script:backupFails) { throw 'backup denied' }
    $script:files[$LiteralPath] = $Value; $script:events += 'backup'
    }
}
function Remove-Item { param($LiteralPath, [switch]$Recurse, [switch]$Force, $ErrorAction)
    $script:deleted += $LiteralPath }
function Write-Host { param($Object) $script:messages += [string]$Object }
function Show-Message { param($Text, $Icon) $script:messages += $Text }
function Show-Error { param($Message) throw $Message }
function Update-PaneRecord { param($Old,$New) }
# Preserve full precision here: the recurrence, rather than display rounding, is under test.
function Format-Ratio([double]$Value) { $Value.ToString('R', [cultureinfo]::InvariantCulture) }
function Get-TabPanes($Workspace) { $script:sourcePanes }
function Invoke-Herdr([string[]]$Arguments) {
    $script:calls.Add(@($Arguments))
    $ok = $true; $json = @{ result = @{} }; $text = ''
    switch ("$($Arguments[0]) $($Arguments[1])") {
        'pane list' { $ok = -not $script:listFails; $text = 'server unavailable'
            $json.result.panes = @($script:alive | ForEach-Object { @{ pane_id = $_ } }) }
        'workspace create' { $json.result = @{ workspace = @{ workspace_id = 'host' }; tab = @{ tab_id = 'host-tab' }; root_pane = @{ pane_id = 'holder' } } }
        'pane get' { $ok = -not $script:getFails; $text = 'lookup failed'; $json.result.pane = @{ tab_id = 'dest' } }
        'pane move' {
            $ok = $Arguments[2] -notin $script:moveFails; $text = 'move failed'
            $newId = $Arguments[2] + '-moved'
            $json.result.move_result = @{ pane = @{ pane_id = $newId } }
            if ($Arguments -contains 'down') { $script:ratios += [double]::Parse($Arguments[[array]::IndexOf($Arguments,'--ratio')+1], [cultureinfo]::InvariantCulture) }
        }
    }
    [pscustomobject]@{ Ok = $ok; Text = $text; Json = $json }
}
function Fake-Herdr {
    $script:calls.Add(@($args)); $global:LASTEXITCODE = 0
    if ($args[0] -eq 'agent' -and $args[1] -eq 'list') {
        if (-not $script:runningOk) { $global:LASTEXITCODE = 1; return '' }
        return ConvertTo-Json -InputObject @{ result = @{ agents = $script:runningAgents } } -Depth 5
    }
    '{"result":{"pane":{"pane_id":"right"}}}'
}
function Set-Groups($Groups) {
    $script:files[$recordPath] = ConvertTo-Json -InputObject @{ workspace='host'; tab='host-tab'; groups=@($Groups); when='now' } -Depth 8
    $script:alive = @($Groups | ForEach-Object { $_.panes })
}
$oldAppData = $env:APPDATA
$temp = Join-Path $env:TEMP ('herdr-batch-b-' + [guid]::NewGuid().ToString('N'))
[void][IO.Directory]::CreateDirectory($temp)
try {
    $env:APPDATA = $temp
    Check 'F04 raw backup precedes SetValue and is reported' {
        Reset-Fixture; $null = Update-PathRegistryEntry $key 'C:\new'
        $paths = @($script:files.Keys); if ($paths.Count -ne 1) { return $false }
        $data = $script:files[$paths[0]] | ConvertFrom-Json
        ($events -join ',') -eq 'backup,SetValue' -and $data.existed -eq $true -and
            $data.value -ceq '%USERPROFILE%\bin;C:\old' -and $data.kind -eq 'ExpandString' -and
            ([datetime]$data.time).Year -ge 2026 -and $files[$paths[0]] -match '"time": "\d{4}-\d\d-\d\dT' -and $paths[0] -like "$temp\herdr\backup\user-path-*.json" -and
            ($messages -join '') -like "*PATH backup: $($paths[0])*" -and $newKind -eq $key.Kind
    }
    Check 'F04 backup failure prevents registry mutation' {
        Reset-Fixture; $script:backupFails = $true; $threw = $false
        try { $null = Update-PathRegistryEntry $key 'C:\new' } catch { $threw = $true }
        $threw -and $events -notcontains 'SetValue'
    }
    Check 'F04 absent value and exclusive backup names' {
        Reset-Fixture; $key.Raw = $null
        $null = Update-PathRegistryEntry $key 'C:\new'; $null = Update-PathRegistryEntry $key 'C:\next'
        $data = @($files.Values | ForEach-Object { $_ | ConvertFrom-Json })
        $files.Count -eq 2 -and @($data | Where-Object { $_.existed -eq $false -and $_.kind -eq 'String' }).Count -eq 2
    }
    Check 'F30 rollback retains current plus Keep minus one newest' {
        Reset-Fixture
        $script:children = @(1..4 | ForEach-Object { [pscustomobject]@{ Name="r$_"; FullName="C:\fake\r$_"; LastWriteTimeUtc=([datetime]'2026-10-23').AddDays(-$_) } })
        Remove-OldReleases 'C:\fake' 'C:\fake\r4' 3
        $first = ($deleted -join ',') -eq 'C:\fake\r3'
        $script:deleted = @(); Remove-OldReleases 'C:\fake' 'C:\fake\r4' 1
        $first -and $deleted.Count -eq 3 -and $deleted -notcontains 'C:\fake\r4'
    }
    foreach ($folder in 'D:\','D:\a b\','D:\a b') {
        Check "F21 command line preserves $folder" {
            $escape = [regex]::Match($cmdText, '(?m)^if "%START_DIR:~-1%"=="\\" set "START_DIR=%START_DIR%\\"\r?$').Value.Trim()
            $line = [regex]::Match($cmdText, '(?m)^\s*start "" wt\.exe .*').Value.Trim().Replace('start "" ', 'echo ')
            $fixture = Join-Path $temp 'quote.cmd'
            [IO.File]::WriteAllText($fixture, "@echo off`r`nset `"START_DIR=$folder`"`r`nset `"HERDR_EXE=C:\herdr.exe`"`r`n$escape`r`n$line`r`n")
            $command = (& cmd.exe /d /c $fixture) -join "`n"
            $argv = [BatchBArgv]::Split($command)
            $argv.Count -eq 6 -and $argv[2] -ceq $folder -and $argv[4] -eq 'Herdr' -and $argv[5] -eq 'C:\herdr.exe'
        }
    }
    Check 'F22 pwsh selection fallback message and exit forwarding' {
        $installCmd -match '(?is)where pwsh\.exe\s*>nul 2>&1\s*if %errorlevel%==0\s*\(\s*pwsh\.exe -NoProfile.*?\) else \(\s*echo PowerShell 7 not found - using Windows PowerShell\.\s*powershell\.exe -NoProfile' -and
            $installCmd -match '(?is)\)\s*set "INSTALLER_EXIT=%ERRORLEVEL%".*?exit /b %INSTALLER_EXIT%'
    }
    Check 'F23 rebuilt ResolveFolder returns absolute docs without Main' {
        $previous = [Environment]::CurrentDirectory
        try {
            [Environment]::CurrentDirectory = $SourceRoot
            $assembly = [Reflection.Assembly]::Load($launcherBytes)
            $method = $assembly.GetType('Program').GetMethod('ResolveFolder', [Reflection.BindingFlags]'NonPublic,Static')
            $resolved = $method.Invoke($null, @('docs'))
            [IO.Path]::IsPathFullyQualified($resolved) -and $resolved -eq (Join-Path $SourceRoot 'docs')
        } finally { [Environment]::CurrentDirectory = $previous }
    }
    Check 'F17 team uses shared profile start and records its pane' {
        Reset-Fixture
        function Start-FolderAgent($ProfileName, $PaneId) { $script:calls.Add(@('profile', $ProfileName, $PaneId)); $true }
        $Agent = 'team'; $paneId = 'w1:p1'; $Path = 'C:\project'; $null = . $team 'C:\fake'
        $started = @($calls | Where-Object { $_[0] -eq 'profile' })
        $started.Count -eq 1 -and ($started[0] -join '|') -eq 'profile|codex|right'
    }
    Check 'F17 missing model still uses shared profile start' {
        Reset-Fixture; $script:profileJson = '{"agents":{}}'
        function Start-FolderAgent($ProfileName, $PaneId) { $script:calls.Add(@('profile', $ProfileName, $PaneId)); $true }
        $Agent = 'team'; $paneId = 'w1:p1'; $Path = 'C:\project'; $null = . $team 'C:\fake'
        @($calls | Where-Object { $_[0] -eq 'profile' -and $_[1] -eq 'codex' }).Count -eq 1
    }
    Check 'F18 equal row recurrence for 2 3 and 5 panes' {
        $equal = $true
        foreach ($count in 2,3,5) {
            Reset-Fixture; $script:sourcePanes = @(1..$count | ForEach-Object { [pscustomobject]@{ pane_id="p$_"; cwd='C:\project'; label='named' } })
            $null = Join-Spaces @([pscustomobject]@{ label='A' })
            $heights = @(1.0)
            foreach ($ratio in $ratios) { $h=$heights[-1]; $heights[-1]=$h*$ratio; $heights += $h*(1-$ratio) }
            $equal = $equal -and $heights.Count -eq $count -and @($heights | Where-Object { [math]::Abs($_ - 1.0/$count) -gt 1e-9 }).Count -eq 0
        }; $equal
    }
    Check 'F19 failed discovery preserves record and reports server error' {
        Reset-Fixture; Set-Groups @(@{ label='A'; panes=@('a') }); $before = $files[$recordPath]
        $script:listFails = $true; $null = Split-Spaces
        $deleted.Count -eq 0 -and $files[$recordPath] -ceq $before -and ($messages -join '') -match 'server unavailable' -and $calls.Count -eq 1
    }
    Check 'F19 partial separation saves only unresolved group with new ids and retries' {
        Reset-Fixture; Set-Groups @(@{ label='A'; panes=@('a') }, @{ label='B'; panes=@('b','c','d') })
        $script:moveFails = @('c'); $null = Split-Spaces
        $saved = $files[$recordPath] | ConvertFrom-Json
        $partial = $deleted.Count -eq 0 -and @($saved.groups).Count -eq 1 -and
            $saved.groups[0].label -eq 'B' -and ($saved.groups[0].panes -join ',') -eq 'b-moved,c,d-moved'
        $script:alive = @('a-moved','b-moved','c','d-moved'); $script:moveFails = @(); $script:calls.Clear()
        $null = Split-Spaces
        $moves = @($calls | Where-Object { $_[1] -eq 'move' })
        $partial -and $deleted -contains $recordPath -and $moves.Count -eq 1 -and $moves[0][2] -eq 'c' -and $moves[0] -notcontains '--new-workspace'
    }
    Check 'F19 failed first move and pane lookup remain retryable' {
        Reset-Fixture; Set-Groups @(@{ label='A'; panes=@('a','b') }); $script:moveFails = @('a'); $null = Split-Spaces
        $first = $deleted.Count -eq 0 -and (($files[$recordPath] | ConvertFrom-Json).groups[0].panes -join ',') -eq 'a,b'
        $script:moveFails = @(); $script:getFails = $true; $null = Split-Spaces
        $first -and $deleted.Count -eq 0 -and (($files[$recordPath] | ConvertFrom-Json).groups[0].panes -join ',') -eq 'a-moved,b'
    }
    Check 'F20 running empty server has successful result and running status' {
        Reset-Fixture; $running = Get-Running; $agents = $running.Agents; $Status = $true
        $output = (. $statusBranch) -join "`n"
        $running.Ok -eq $true -and $running.Agents -is [array] -and $running.Agents.Count -eq 0 -and $output -match 'running with 0 agent'
    }
    Check 'F20 failed server differs from zero agents' {
        Reset-Fixture; $script:runningOk = $false; $running = Get-Running; $agents = $running.Agents; $Status = $true
        $output = (. $statusBranch) -join "`n"
        $running.Ok -eq $false -and $running.Agents -is [array] -and $output -match 'not reachable'
    }
    Check 'F20 one and many agents remain arrays' {
        $safe = $true
        foreach ($count in 1,2) {
            Reset-Fixture; $script:runningAgents = @(1..$count | ForEach-Object { @{ name="a$_" } }); $running = Get-Running
            $safe = $safe -and $running.Ok -eq $true -and $running.Agents -is [array] -and $running.Agents.Count -eq $count
        }; $safe
    }
} finally {
    $env:APPDATA = $oldAppData
    # Delete only this suite's checked, unique TEMP fixture directory.
    if ([IO.Path]::GetFullPath($temp).StartsWith([IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\') -and
        [IO.Path]::GetFileName($temp) -like 'herdr-batch-b-*') { [IO.Directory]::Delete($temp, $true) }
}
Write-Output "PASS $script:pass/$($script:pass + $script:fail)"
if ($script:fail) { exit 1 }
