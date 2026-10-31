# herdr-keepalive.ps1 - keep Orca and Herdr running all the time (HISTORY 66, GUIDE 13.22).
#
# Run every minute by the Task Scheduler task "Herdr Keepalive" (at sign-in + every 1 minute,
# `conhost.exe --headless`, no window). Each run:
#   - Orca desktop app missing (closed from the tray, crashed): start Orca.exe again.
#   - Herdr server not running: run herdr-boot.ps1 (server headless, a lead, the watcher).
# Edge cases:
#   - An app seen down only once is started on the NEXT run (an update or restart may be under way);
#     on the first run of a new sign-in (no run in the last 5 minutes) both start at once.
#   - Orca's updater running: Orca is left alone.
#   - Owner wants it stopped: `-Pause 120` (minutes, both apps) or `-Pause 120 -App herdr`; `-Resume`.
#     herdr-stop.ps1 pauses Herdr for only 15 minutes after it stops the server (a stop by someone
#     else at home does not last).
#   - Automatic Windows sign-in (Winlogon AutoAdminLogon=1): the first run after the sign-in locks the
#     PC once the apps are started, so nobody at home uses the owner's open desktop.
#   - One run at a time (named mutex); a run that finds another running exits.
# State: %APPDATA%\herdr\keepalive-state.json; log %APPDATA%\herdr\keepalive.log (below 100 KB).
#   pwsh -File herdr-keepalive.ps1 -Status     print state, pauses and what runs
#   pwsh -File herdr-keepalive.ps1 -Install    register the task (no admin); -Uninstall removes it
param(
    [int]$Pause = 0,
    [ValidateSet('both', 'orca', 'herdr')][string]$App = 'both',
    [switch]$Resume,
    [switch]$Status,
    [switch]$Install,
    [switch]$Uninstall,
    [switch]$DryRun,
    [string]$OrcaExe = (Join-Path $env:LOCALAPPDATA 'Programs\orca\Orca.exe')   # tests point it elsewhere
)
$ErrorActionPreference = 'Stop'
$taskName = 'Herdr Keepalive'
$herdr = Join-Path $env:LOCALAPPDATA 'Programs\Herdr\bin\herdr.exe'
$boot = Join-Path $PSScriptRoot 'herdr-boot.ps1'
$dataDir = Join-Path $env:APPDATA 'herdr'
$statePath = Join-Path $dataDir 'keepalive-state.json'
$logPath = Join-Path $dataDir 'keepalive.log'
$null = New-Item -ItemType Directory -Path $dataDir -Force

function Write-KeepLog([string]$Text) {
    $line = "{0:yyyy-MM-dd HH:mm:ss} {1}" -f (Get-Date), $Text
    Write-Output $line
    Add-Content -LiteralPath $logPath -Value $line -Encoding utf8
    if ((Get-Item -LiteralPath $logPath).Length -ge 100KB) {
        $rows = @([IO.File]::ReadAllLines($logPath))
        [IO.File]::WriteAllLines($logPath, $rows[[int]($rows.Count / 2)..($rows.Count - 1)])
    }
}

function Read-KeepState {
    $s = @{ lastRun = $null; orcaDown = 0; herdrDown = 0; pause = @{}; lockedSession = $null }
    if (Test-Path -LiteralPath $statePath) {
        try { $j = Get-Content -LiteralPath $statePath -Raw | ConvertFrom-Json -AsHashtable; foreach ($k in $j.Keys) { $s[$k] = $j[$k] } }
        catch { Write-KeepLog "state unreadable, starting fresh: $($_.Exception.Message)" }
    }
    if (-not $s.pause) { $s.pause = @{} }
    $s
}

function Save-KeepState($State) {
    $tmp = "$statePath.tmp"
    $State | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $tmp -Encoding utf8
    Move-Item -LiteralPath $tmp -Destination $statePath -Force
}

function Test-Paused($State, [string]$Name) {
    $until = $State.pause[$Name]
    $until -and ([datetime]$until) -gt (Get-Date)
}

function Get-OrcaApp {
    @(Get-Process -Name Orca -ErrorAction SilentlyContinue | Where-Object { $_.Path -and $_.Path -ieq $orcaExe })
}

function Test-OrcaUpdating {
    [bool](Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $_.ProcessName -match '(?i)^orca[ -._]?setup' -or ($_.Path -and $_.Path -match '(?i)\\orca-updater\\')
    })
}

function Test-HerdrServer {
    $null = & $herdr workspace list 2>&1
    $LASTEXITCODE -eq 0
}

function Get-TaskCommand {
    $pwsh = (Get-Command pwsh).Source
    [pscustomobject]@{ Exe = Join-Path $env:WINDIR 'System32\conhost.exe'; Args = "--headless `"$pwsh`" -NoProfile -File `"$PSCommandPath`"" }
}

# ---- one-off commands ----
if ($Install) {
    $cmd = Get-TaskCommand
    $action = New-ScheduledTaskAction -Execute $cmd.Exe -Argument $cmd.Args -WorkingDirectory $PSScriptRoot
    $atLogon = New-ScheduledTaskTrigger -AtLogOn -User "$env:USERDOMAIN\$env:USERNAME"
    $every = New-ScheduledTaskTrigger -Once -At (Get-Date).AddMinutes(1) -RepetitionInterval (New-TimeSpan -Minutes 1)
    $settings = New-ScheduledTaskSettingsSet -MultipleInstances IgnoreNew -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
        -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 10) -Hidden
    $principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -LogonType Interactive -RunLevel Limited
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger @($atLogon, $every) -Settings $settings -Principal $principal -Force | Out-Null
    $t = Get-ScheduledTask -TaskName $taskName
    Write-KeepLog "task '$taskName' registered: state $($t.State), $($t.Triggers.Count) triggers"
    # The task's sign-in trigger replaces Phase 64's Startup shortcut; two starters could race for the server.
    $oldLink = Join-Path ([Environment]::GetFolderPath('Startup')) 'Herdr Boot.lnk'
    if (Test-Path -LiteralPath $oldLink) { Remove-Item -LiteralPath $oldLink; Write-KeepLog "removed Startup shortcut $oldLink (the task replaces it)" }
    Start-ScheduledTask -TaskName $taskName
    Write-KeepLog "task started once; see -Status"
    exit 0
}
if ($Uninstall) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    Write-KeepLog "task '$taskName' removed"
    exit 0
}
$state = Read-KeepState
if ($Pause -gt 0 -or $Resume) {
    $names = if ($App -eq 'both') { @('orca', 'herdr') } else { @($App) }
    foreach ($n in $names) {
        if ($Resume) { $state.pause.Remove($n) } else { $state.pause[$n] = (Get-Date).AddMinutes($Pause).ToString('o') }
    }
    Save-KeepState $state
    Write-KeepLog $(if ($Resume) { "resumed: $($names -join ', ')" } else { "paused $Pause min: $($names -join ', ')" })
    exit 0
}
if ($Status) {
    $t = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    Write-Output "Task:   $(if ($t) { "$($t.State)" } else { 'not installed (run -Install)' })"
    Write-Output "Orca:   $(if (Get-OrcaApp) { 'running' } else { 'not running' })$(if (Test-Paused $state 'orca') { " (paused until $($state.pause.orca))" })"
    Write-Output "Herdr:  $(if (Test-HerdrServer) { 'server running' } else { 'server not running' })$(if (Test-Paused $state 'herdr') { " (paused until $($state.pause.herdr))" })"
    Write-Output "Last run: $($state.lastRun)"
    exit 0
}

# ---- the check ----
$mutex = New-Object Threading.Mutex($false, 'Local\herdr-keepalive')
$owned = $false
try { $owned = $mutex.WaitOne(0) } catch [Threading.AbandonedMutexException] { $owned = $true }
if (-not $owned) { $mutex.Dispose(); exit 0 }
try {
    $fresh = -not $state.lastRun -or ((Get-Date) - [datetime]$state.lastRun).TotalMinutes -gt 5
    $need = if ($fresh) { 1 } else { 2 }
    $started = $false

    if (Test-Paused $state 'orca') { $state.orcaDown = 0 }
    elseif (Get-OrcaApp) { $state.orcaDown = 0 }
    elseif (Test-OrcaUpdating) { $state.orcaDown = 0; Write-KeepLog "orca not running, updater busy - waiting" }
    else {
        $state.orcaDown++
        if ($state.orcaDown -ge $need) {
            if (-not (Test-Path -LiteralPath $orcaExe)) { Write-KeepLog "orca not installed at $orcaExe" }
            elseif ($DryRun) { Write-KeepLog "dry run: would start Orca" }
            else { Start-Process -FilePath $orcaExe; $started = $true; Write-KeepLog "orca was not running - started" }
            $state.orcaDown = 0
        } else { Write-KeepLog "orca not running - start on the next check" }
    }

    if (Test-Paused $state 'herdr') { $state.herdrDown = 0 }
    elseif (Test-HerdrServer) { $state.herdrDown = 0 }
    else {
        $state.herdrDown++
        if ($state.herdrDown -ge $need) {
            if ($DryRun) { Write-KeepLog "dry run: would run herdr-boot.ps1" }
            else {
                $out = & (Get-Command pwsh).Source -NoProfile -File $boot 2>&1
                $started = $true
                Write-KeepLog ("herdr server was not running - herdr-boot exit {0}: {1}" -f $LASTEXITCODE, (($out | ForEach-Object { $_.ToString() }) -join ' | '))
            }
            $state.herdrDown = 0
        } else { Write-KeepLog "herdr server not running - start on the next check" }
    }

    # Automatic sign-in leaves the owner's desktop open: lock once per sign-in after starting the apps.
    $mySession = (Get-Process -Id $PID).SessionId
    $session = (Get-Process -Name explorer -ErrorAction SilentlyContinue | Where-Object SessionId -eq $mySession |
        Sort-Object StartTime | Select-Object -First 1).StartTime
    $sessionKey = if ($session) { ([datetime]$session).ToString('o') } else { '' }
    $auto = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon' -Name AutoAdminLogon -ErrorAction SilentlyContinue).AutoAdminLogon
    if ($fresh -and $auto -eq '1' -and $sessionKey -and $state.lockedSession -ne $sessionKey) {
        $state.lockedSession = $sessionKey
        if ($DryRun) { Write-KeepLog "dry run: would lock the PC after automatic sign-in" }
        else { Start-Sleep -Seconds $(if ($started) { 20 } else { 0 }); rundll32.exe user32.dll,LockWorkStation; Write-KeepLog "automatic sign-in: PC locked" }
    }

    $state.lastRun = (Get-Date).ToString('o')
    Save-KeepState $state
} catch {
    Write-KeepLog "ERROR: $($_.Exception.Message)"
    exit 1
} finally {
    $mutex.ReleaseMutex(); $mutex.Dispose()
}
