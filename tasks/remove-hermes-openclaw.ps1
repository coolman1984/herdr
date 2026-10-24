# Run from an unrestricted PowerShell 7 session. -WhatIf performs read-only discovery.
# Only the explicit allowlist is removed; other discoveries require review.
# Preview: pwsh -NoProfile -File I:\herdr\tasks\remove-hermes-openclaw.ps1 -WhatIf
# Remove (owner authorized full disk access 2026-10-24): pwsh -NoProfile -File I:\herdr\tasks\remove-hermes-openclaw.ps1
# -ScanOnly is read-only. -FileTargetsOnly skips task/service mutations after an access-denied result.
# Administrator recovery, if task/service removal is denied (do not retry here):
# Unregister-ScheduledTask -TaskName 'Clawdbot Gateway' -Confirm:$false
# Unregister-ScheduledTask -TaskName 'Hermes_Gateway' -Confirm:$false
# Stop-Service -Name clawdbot
# sc.exe delete clawdbot
[CmdletBinding(SupportsShouldProcess)]
param([switch]$ScanOnly, [switch]$FileTargetsOnly)
$ErrorActionPreference = 'Stop'
$script:Problems = 0
$script:Hits = 0
$pattern = '(?i)hermes|openclaw|clawdbot|moltbot|nousresearch'
$targets = @(
    'C:\Users\mflma\.clawdbot', 'C:\Users\mflma\.hermes',
    'C:\Users\mflma\Hermes-Mastery-Database', 'C:\Users\mflma\hermes-youtube-stack',
    'C:\Users\mflma\AppData\Roaming\Hermes',
    'C:\Users\mflma\AppData\Local\com.nousresearch.hermes.setup',
    'C:\Users\mflma\AppData\Local\hermes', 'G:\Hermes Secrets',
    'G:\hermes-agent-self-evolution',
    'C:\Users\mflma\scoop\persist\nodejs-lts\bin\openclaw',
    'C:\Users\mflma\scoop\persist\nodejs-lts\bin\openclaw.cmd',
    'C:\Users\mflma\scoop\persist\nodejs-lts\bin\openclaw.ps1',
    'C:\Users\mflma\scoop\persist\nodejs-lts\bin\node_modules\openclaw',
    'C:\Users\mflma\.codex\skills\hermes-agent-operations',
    'C:\Users\mflma\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Hermes.lnk',
    'C:\Users\mflma\Documents\Codex\2026-07-05\understand-the-project-first-then-tell\work\repo-scan\Company-Dashboard-Project\.hermes',
    'C:\Users\mflma\Documents\Codex\2026-06-03\fix-this-error-i-downloaded-hermes\work\openclaw-auth-backup-20260603-213351',
    'C:\Users\mflma\Documents\Codex\2026-06-03\fix-this-error-i-downloaded-hermes\work\openclaw-backup-20260603-212400',
    'C:\Users\mflma\AppData\Local\claude-cli-nodejs\Cache\g--Hermes-Secrets',
    'G:\Skills\Sources\hermes-agent-main',
    'G:\Company-Dashboard-Project\.hermes',
    'G:\EXCEL-AUTOMATION\data\hermes-server-8001.pid',
    'C:\Users\mflma\.gemini\antigravity\scratch\Hermes-Setup.exe',
    'C:\Users\mflma\.gemini\antigravity\code_tracker\active\Moltbot_3cb610f54e81817c88de494e5209e1fbc6025635',
    'C:\Users\mflma\.gemini\antigravity\code_tracker\active\no_repo\9efe97da187b8d55e4f6b5e67655ac70_Launch_Moltbot.bat',
    'C:\Users\mflma\.gemini\antigravity\code_tracker\active\no_repo\ab1ff61fcb323245a6d8aa270bf53414_openclaw.json',
    'C:\Users\mflma\.gemini\antigravity\code_tracker\active\no_repo\ecec9d551386e0658ff2275e8ece6d29_moltbot_tray.py',
    'C:\Users\mflma\.claude\projects\g--Hermes-Secrets',
    'C:\tmp\clawdbot', 'C:\tmp\clawdbot-gateway.log',
    'C:\tmp\hermes_screen_now.jpg', 'C:\tmp\hermes_screen_now.png', 'C:\tmp\openclaw',
    'E:\tmp\hermes_screen_now.jpg', 'E:\tmp\hermes_screen_now.png',
    'E:\Github Projects\Moltbot', 'E:\Github Projects\Online Openclaw',
    'E:\Github Projects\OpenClaw_VPS_Control_Guide.md',
    'E:\Github Projects\tmp\clawdbot', 'E:\Github Projects\tmp\openclaw',
    'E:\Github Projects\tmp\openclaw-install-flow',
    'E:\Github Projects\Open Claw',
    'E:\all my projects\Hermes Secrets',
    'E:\all my projects\hermes-agent-self-evolution',
    'E:\all my projects\Skills\Sources\hermes-agent-main',
    'E:\Github Projects\Agents\stack\openclaw-openai.batch.json',
    'E:\Github Projects\.pnpm-store\v10\index\fe\3b9251bd3bd17cfc2bc0a74a18da650afb095b51696e056d1f77960af584b7-openclaw@2026.3.13.json',
    'E:\Github Projects\.pnpm-store\v10\index\7b\9464d808f9e6c0ffe4bc97f7899d602423ef3a475e8d31219b25fa4d78d3e9-openclaw@2026.3.8.json',
    'E:\Github Projects\.pnpm-store\v10\index\6b\aceb712eafe6d516ab3b0587970db72bb9f93adad545b9caf3ed62145808a4-openclaw@2026.2.24.json',
    'E:\all my projects\Company-Dashboard-Project\.hermes',
    'E:\all my projects\EXCEL-AUTOMATION\data\hermes-server-8001.pid',
    'E:\Downloads\Skills\hermes-agent-main',
    'E:\Downloads\Skills\Hermes-Agent-Skills',
    'E:\Downloads\Skills\Sources\hermes-agent-main.zip',
    'E:\Downloads\hermes-skills-share-2026-06-05',
    'E:\Downloads\hermes-skills-share-2026-06-05.zip',
    'C:\Users\mflma\AppData\Local\Temp\hermes-dm',
    'C:\Users\mflma\.local\state\hermes',
    'C:\Users\mflma\.pm2\pids\openclaw-gateway-2.pid',
    'C:\Users\mflma\.kilocode\cli\workspaces\moltbot-50a7b85a',
    'C:\Users\mflma\.kilocode\cli\workspaces\moltbot-808aec49',
    'C:\Users\mflma\AppData\Roaming\Code\User\globalStorage\kilocode.kilo-code\vector\.clawdbot-bf302d3ab38b5d09',
    'C:\Users\mflma\AppData\Roaming\Code\User\globalStorage\kilocode.kilo-code\vector\Moltbot-808aec492057a93b',
    'C:\Users\mflma\AppData\Roaming\Code\User\globalStorage\github.vscode-pull-request-github\assignableUsers\openclaw',
    'C:\Users\mflma\AppData\Roaming\Code\User\globalStorage\github.vscode-pull-request-github\assignableUsers\coolman1984\Hermes-Secrets.json',
    'C:\Users\mflma\AppData\Roaming\Code\User\globalStorage\github.vscode-pull-request-github\assignableUsers\coolman1984\Moltbot.json',
    'C:\Users\mflma\AppData\Roaming\ai.opencode.desktop\opencode.workspace.G--Hermes-Se.1vziyee.dat',
    'C:\Users\mflma\.gemini\antigravity-ide\code_tracker\active\Moltbot_3cb610f54e81817c88de494e5209e1fbc6025635',
    'C:\Users\mflma\.gemini\antigravity-ide\code_tracker\active\no_repo\9efe97da187b8d55e4f6b5e67655ac70_Launch_Moltbot.bat',
    'C:\Users\mflma\.gemini\antigravity-ide\code_tracker\active\no_repo\ab1ff61fcb323245a6d8aa270bf53414_openclaw.json',
    'C:\Users\mflma\.gemini\antigravity-ide\code_tracker\active\no_repo\ecec9d551386e0658ff2275e8ece6d29_moltbot_tray.py',
    'C:\Users\mflma\.gemini\antigravity-backup\code_tracker\active\Moltbot_3cb610f54e81817c88de494e5209e1fbc6025635',
    'C:\Users\mflma\.gemini\antigravity-backup\code_tracker\active\no_repo\9efe97da187b8d55e4f6b5e67655ac70_Launch_Moltbot.bat',
    'C:\Users\mflma\.gemini\antigravity-backup\code_tracker\active\no_repo\ab1ff61fcb323245a6d8aa270bf53414_openclaw.json',
    'C:\Users\mflma\.gemini\antigravity-backup\code_tracker\active\no_repo\ecec9d551386e0658ff2275e8ece6d29_moltbot_tray.py'
)
function Fail([string]$Label, [string]$Reason) {
    $script:Problems++
    Write-Output "FAIL $Label : $Reason"
}
function Found([string]$Label) {
    $script:Hits++
    Write-Output "LEFT $Label"
}
function Space([string]$Stage) {
    foreach ($letter in 'C','G') {
        try {
            $driveInfo = [IO.DriveInfo]::new("${letter}:\")
            Write-Output ("SPACE {0} {1}: {2} bytes ({3:N3} GiB) free" -f $Stage,$letter,$driveInfo.AvailableFreeSpace,($driveInfo.AvailableFreeSpace/1GB))
        } catch { Fail "space $letter" $_.Exception.Message }
    }
}
function Scan([string]$Stage) {
    Write-Output "SCAN $Stage (drive-root and home names; standard install/shortcut locations; no file contents)"
    $script:Hits = 0
    $locations = @('C:\','G:\','H:\','I:\','E:\','C:\Users\mflma',
        'C:\Users\mflma\AppData\Local','C:\Users\mflma\AppData\Roaming',
        'C:\Users\mflma\.codex\skills','C:\Users\mflma\.agents\skills',
        'C:\Users\mflma\scoop\apps','C:\Users\mflma\scoop\shims',
        'C:\Users\mflma\scoop\persist\nodejs-lts\bin',
        'C:\Users\mflma\scoop\persist\nodejs-lts\bin\node_modules',
        'C:\Users\mflma\AppData\Roaming\npm',
        'C:\Users\mflma\AppData\Roaming\npm\node_modules',
        'C:\Program Files','C:\Program Files (x86)','C:\ProgramData',
        'C:\Users\mflma\AppData\Roaming\uv\tools',
        'C:\Users\mflma\.local\bin', 'C:\Users\mflma\AppData\Roaming\Python',
        'C:\Users\mflma\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup',
        'C:\ProgramData\Microsoft\Windows\Start Menu\Programs\Startup')
    foreach ($location in $locations) {
        try {
            if (Test-Path -LiteralPath $location) {
                foreach ($entry in Get-ChildItem -LiteralPath $location -Force) {
                    if ($entry.Name -match $pattern) { Found $entry.FullName }
                }
            } else { Write-Output "ABSENT scan location $location" }
        } catch { Fail "scan $location" $_.Exception.Message }
    }
    # Check every explicit target, including targets nested below a nonmatching parent.
    foreach ($target in $targets) {
        try { if (Test-Path -LiteralPath $target) { Found $target } }
        catch { Fail "scan $target" $_.Exception.Message }
    }
    foreach ($shortcutRoot in @('C:\Users\mflma\Desktop','C:\Users\Public\Desktop',
        'C:\Users\mflma\AppData\Roaming\Microsoft\Windows\Start Menu',
        'C:\ProgramData\Microsoft\Windows\Start Menu')) {
        try {
            if (!(Test-Path -LiteralPath $shortcutRoot)) { continue }
            $shellObject = New-Object -ComObject WScript.Shell
            foreach ($entry in Get-ChildItem -LiteralPath $shortcutRoot -Recurse -Force -File) {
                $related = $entry.Name -match $pattern
                if ($entry.Extension -eq '.lnk') {
                    $link = $shellObject.CreateShortcut($entry.FullName)
                    $related = $related -or $link.TargetPath -match $pattern -or $link.Arguments -match $pattern
                }
                if ($related) { Found "shortcut $($entry.FullName)" }
            }
        } catch { Fail "shortcuts $shortcutRoot" $_.Exception.Message }
    }
    try {
        $taskCount = 0
        foreach ($task in Get-ScheduledTask) {
            $taskCount++
            if ($task.TaskName -match $pattern -or (($task.Actions | ForEach-Object { $_.Execute; $_.Arguments }) -join ' ') -match $pattern) {
                Found "task $($task.TaskPath)$($task.TaskName)"
            }
        }
        Write-Output "OK task inventory completed: $taskCount tasks inspected"
    } catch { Fail 'task scan' $_.Exception.Message }
    try {
        $serviceCount = 0
        foreach ($service in Get-CimInstance Win32_Service) {
            $serviceCount++
            if ($service.Name -match $pattern -or $service.DisplayName -match $pattern -or $service.PathName -match $pattern) {
                Found "service $($service.Name) state=$($service.State)"
            }
        }
        Write-Output "OK service inventory completed: $serviceCount services inspected"
    } catch { Fail 'service scan' $_.Exception.Message }
    foreach ($keyPath in @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Run',
        'HKLM:\Software\Microsoft\Windows\CurrentVersion\Run',
        'HKCU:\Software\Microsoft\Windows\CurrentVersion\RunOnce',
        'HKLM:\Software\Microsoft\Windows\CurrentVersion\RunOnce',
        'HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run')) {
        try {
            if (Test-Path -LiteralPath $keyPath) {
                $key = Get-Item -LiteralPath $keyPath
                foreach ($valueName in $key.GetValueNames()) {
                    if ($valueName -match $pattern -or [string]$key.GetValue($valueName) -match $pattern) {
                        Found "Run entry $keyPath [$valueName] (value withheld)"
                    }
                }
            }
        } catch { Fail "Run scan $keyPath" $_.Exception.Message }
    }
    foreach ($scope in 'User','Machine','Process') {
        try {
            $variables = [Environment]::GetEnvironmentVariables($scope)
            foreach ($variableName in $variables.Keys) {
                if ($variableName -match $pattern) { Found "environment $scope $variableName (value withheld)" }
            }
            foreach ($part in ([Environment]::GetEnvironmentVariable('Path',$scope) -split ';')) {
                if ($part -match $pattern) { Found "PATH $scope $part" }
            }
        } catch { Fail "environment scan $scope" $_.Exception.Message }
    }
    foreach ($commandName in 'openclaw','hermes') {
        $commands = @(Get-Command $commandName -All -ErrorAction SilentlyContinue)
        if ($commands.Count) { foreach ($command in $commands) { Found "command $commandName $($command.Source)" } }
        else { Write-Output "OK command $commandName absent" }
    }
    # Native inventory output is held in memory and only matching package lines are displayed.
    foreach ($probe in @(
        @{Name='npm'; Arguments=@('ls','--global','--depth=0','--json')},
        @{Name='pip'; Arguments=@('list','--format=json','--disable-pip-version-check')},
        @{Name='uv'; Arguments=@('tool','list')},
        @{Name='scoop'; Arguments=@('list')}
    )) {
        try {
            $tool = Get-Command $probe.Name -ErrorAction SilentlyContinue
            if (!$tool) { Write-Output "UNAVAILABLE package inventory $($probe.Name)"; continue }
            $probeArgs = $probe.Arguments
            $global:LASTEXITCODE = 0
            $output = @(& $tool @probeArgs 2>&1)
            $nativeExit = $LASTEXITCODE
            if ($nativeExit -ne 0) { Fail "package inventory $($probe.Name)" "exit $nativeExit; inventory incomplete"; continue }
            $matching = @($output | Where-Object { [string]$_ -match $pattern })
            if ($matching.Count) {
                # JSON from pip is one line containing all packages: extract names only.
                $names = [regex]::Matches(($matching -join ' '),'(?i)[a-z0-9_.-]*(?:hermes|openclaw|clawdbot|moltbot|nousresearch)[a-z0-9_.-]*')
                foreach ($name in ($names.Value | Sort-Object -Unique)) { Found "package inventory $($probe.Name): $name" }
            } else { Write-Output "OK package inventory $($probe.Name): no matching names" }
        } catch { Fail "package inventory $($probe.Name)" $_.Exception.Message }
    }
    Write-Output "SCAN $Stage : $script:Hits findings (duplicates possible); scan is bounded, not an exhaustive drive-content search"
}
Space 'before'
Scan 'before'
if ($ScanOnly) { Space 'after'; Write-Output "READ ONLY: $script:Problems errors; $script:Hits findings"; exit ([int][bool]($script:Problems -or $script:Hits)) }
if (!$FileTargetsOnly) {
foreach ($taskName in 'Clawdbot Gateway','Hermes_Gateway','OpenClaw Gateway','OpenClawNightly','BirdCLI-Twitter-To-Obsidian','BirdCLI-Twitter-To-Obsidian-DryRun') {
    try {
        $existing = @(Get-ScheduledTask | Where-Object TaskName -eq $taskName)
        foreach ($task in $existing) {
            if ($PSCmdlet.ShouldProcess("$($task.TaskPath)$taskName",'Stop and unregister scheduled task')) {
                if ($task.State -eq 'Running') { Stop-ScheduledTask -TaskName $taskName -TaskPath $task.TaskPath }
                Unregister-ScheduledTask -TaskName $taskName -TaskPath $task.TaskPath -Confirm:$false
                if (Get-ScheduledTask | Where-Object { $_.TaskName -eq $taskName -and $_.TaskPath -eq $task.TaskPath }) { throw 'task remains' }
                Write-Output "OK task $taskName removed"
            }
        }
        if (!$existing.Count) { Write-Output "OK task $taskName absent" }
    } catch { Fail "task $taskName" "$($_.Exception.Message); ADMIN: Unregister-ScheduledTask -TaskName '$taskName' -Confirm:`$false" }
}
try {
    $service = Get-Service -Name 'clawdbot' -ErrorAction SilentlyContinue
    if (!$service) { Write-Output 'OK service clawdbot absent' }
    elseif ($PSCmdlet.ShouldProcess('clawdbot','Stop and delete Windows service (leave ERP files untouched)')) {
        if ($service.Status -ne 'Stopped') { Stop-Service -Name 'clawdbot' }
        $deleteResult = & sc.exe delete clawdbot 2>&1
        if ($LASTEXITCODE -ne 0) { throw "sc.exe delete failed: $deleteResult" }
        if (Get-Service -Name clawdbot -ErrorAction SilentlyContinue) { throw 'service remains or is pending deletion; restart may be needed' }
        Write-Output 'OK service clawdbot removed'
    }
} catch { Fail 'service clawdbot' "$($_.Exception.Message); ADMIN: Stop-Service -Name clawdbot; sc.exe delete clawdbot" }
}
foreach ($target in $targets) {
    try {
        if (!(Test-Path -LiteralPath $target)) { Write-Output "OK absent $target"; continue }
        $absoluteTarget = [IO.Path]::GetFullPath($target).TrimEnd('\')
        if ($absoluteTarget -notin $targets -or $absoluteTarget -like 'G:\Real Accounting ERP System*') { throw 'target failed exact allowlist check' }
        $item = Get-Item -LiteralPath $absoluteTarget -Force
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'target is a link; review its destination before removal' }
        $bytes = 0L
        $files = 0L
        if ($item.PSIsContainer) {
            foreach ($file in Get-ChildItem -LiteralPath $absoluteTarget -Recurse -Force -File) { $bytes += $file.Length; $files++ }
        } else { $bytes = $item.Length; $files = 1 }
        Write-Output ("SIZE {0}: {1} bytes ({2:N3} GiB), {3} files" -f $absoluteTarget,$bytes,($bytes/1GB),$files)
        if ($PSCmdlet.ShouldProcess($absoluteTarget,'Stop processes inside this exact target and permanently delete')) {
            foreach ($process in Get-Process) {
                $processPath = $process.Path
                if ($processPath -and ($processPath.Equals($absoluteTarget,[StringComparison]::OrdinalIgnoreCase) -or $processPath.StartsWith($absoluteTarget+'\',[StringComparison]::OrdinalIgnoreCase))) {
                    Stop-Process -Id $process.Id
                    Write-Output "OK stopped target process $($process.Id)"
                }
            }
            # Keep deletion in PowerShell with LiteralPath; never pass enumerated paths to cmd.
            Remove-Item -LiteralPath $absoluteTarget -Recurse -Force
            if (Test-Path -LiteralPath $absoluteTarget) { throw 'target still exists' }
            Write-Output "OK removed $absoluteTarget ($bytes bytes)"
        }
    } catch { Fail $target $_.Exception.Message }
}
foreach ($variableName in 'HERMES_HOME','HERMES_GIT_BASH_PATH') {
    try {
        if ($PSCmdlet.ShouldProcess("User environment $variableName",'Remove')) {
            $userEnvironment = Get-Item -LiteralPath 'HKCU:\Environment'
            if ($variableName -in $userEnvironment.GetValueNames()) {
                Remove-ItemProperty -LiteralPath 'HKCU:\Environment' -Name $variableName
            }
            if (Test-Path -LiteralPath "Env:\$variableName") { Remove-Item -LiteralPath "Env:\$variableName" }
            if ($variableName -in (Get-Item -LiteralPath 'HKCU:\Environment').GetValueNames()) { throw 'variable remains' }
            Write-Output "OK removed User environment $variableName"
        }
    } catch { Fail "environment $variableName" $_.Exception.Message }
}
try {
    $envKey = Get-Item -LiteralPath 'HKCU:\Environment'
    $oldPath = [string]$envKey.GetValue('Path',$null,[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    $pathParts = $oldPath -split ';'
    $kept = @($pathParts | Where-Object {
        [Environment]::ExpandEnvironmentVariables($_.Trim().Trim('"')).TrimEnd('\') -ine 'C:\Users\mflma\AppData\Local\hermes\bin'
    })
    if ($kept.Count -eq $pathParts.Count) { Write-Output 'OK target user PATH entry absent' }
    elseif ($PSCmdlet.ShouldProcess('User PATH','Back up original raw value and remove only the exact Hermes bin entry')) {
        $backup = 'I:\herdr\tasks\path-backup-2026-10-24.txt'
        # Never overwrite the original recovery copy on a second run.
        $stream = [IO.File]::Open($backup,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write)
        try {
            $data = [Text.Encoding]::UTF8.GetBytes($oldPath)
            $stream.Write($data,0,$data.Length)
        } finally { $stream.Dispose() }
        $kind = $envKey.GetValueKind('Path')
        $writeKey = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment',$true)
        try { $writeKey.SetValue('Path',($kept -join ';'),$kind) } finally { $writeKey.Dispose() }
        $readBack = (Get-Item -LiteralPath 'HKCU:\Environment').GetValue('Path',$null,[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        if ($readBack -cne ($kept -join ';')) { throw 'PATH read-back differs' }
        $env:Path = (($env:Path -split ';' | Where-Object { $_.Trim().Trim('"').TrimEnd('\') -ine 'C:\Users\mflma\AppData\Local\hermes\bin' }) -join ';')
        Write-Output "OK user PATH changed; backup $backup; existing applications retain their inherited PATH"
    }
} catch { Fail 'user PATH' $_.Exception.Message }
Scan 'after'
Space 'after'
if ($WhatIfPreference) { Write-Output 'DRY RUN: no removal performed' }
elseif ($script:Problems -or $script:Hits) { Write-Output "PARTIAL: $script:Problems errors; $script:Hits remaining findings"; exit 1 }
else { Write-Output 'PARTIAL: scoped checks clear; full drive-content scan has not been performed' }
