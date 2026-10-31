# setup-nosleep.ps1 - this PC never sleeps, so the owner can always reach it (HISTORY 66, GUIDE 13.22).
#
# Owner's rule (2026-10-31): anyone at home may SHUT DOWN or restart normally; the PC must never SLEEP.
# After a shutdown, automatic sign-in + the "Herdr Keepalive" task bring Orca and Herdr back.
# The owner runs it ONCE: pwsh -File I:\herdr\setup-nosleep.ps1   (it asks Windows for admin rights).
# -Undo puts every value back from the backup it wrote (%APPDATA%\herdr\nosleep-backup.json).
# On mains power:
#   1. Sleep after: never. Hibernate after: never.
#   2. Start menu power button: Shut down instead of Sleep; power / sleep buttons: Shut down / nothing
#      (only where this PC exposes those settings).
#   3. "Sleep" and "Hibernate" are removed from the Start menu power menu (policy ShowSleepOption = 0).
#   4. Windows Update does not restart by itself while someone is signed in (NoAutoRebootWithLoggedOnUsers),
#      so it never cuts the owner's remote work; a shutdown by a person still works.
param([switch]$Undo)
$ErrorActionPreference = 'Stop'
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    $relaunch = @('-NoProfile', '-NoExit', '-File', "`"$PSCommandPath`"") + $(if ($Undo) { @('-Undo') })
    try { Start-Process -FilePath (Get-Command pwsh).Source -ArgumentList $relaunch -Verb RunAs }
    catch { Write-Error "Admin rights were refused, nothing changed: $($_.Exception.Message)"; exit 1 }
    Write-Output "Continuing in the new administrator window."
    exit 0
}

$backupPath = Join-Path $env:APPDATA 'herdr\nosleep-backup.json'
$explorerKey = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Explorer'
$auKey = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU'
# Value to set: 0 = do nothing / never; Start menu button 2 = Shut down; power button 3 = Shut down.
$powerItems = @(
    @{ Sub = 'SUB_SLEEP'; Setting = 'STANDBYIDLE'; Value = 0 },
    @{ Sub = 'SUB_SLEEP'; Setting = 'HIBERNATEIDLE'; Value = 0 },
    @{ Sub = 'SUB_BUTTONS'; Setting = 'UIBUTTON_ACTION'; Value = 2 },
    @{ Sub = 'SUB_BUTTONS'; Setting = 'PBUTTONACTION'; Value = 3 },
    @{ Sub = 'SUB_BUTTONS'; Setting = 'SBUTTONACTION'; Value = 0 }
)
$regItems = @(
    @{ Key = $explorerKey; Name = 'ShowSleepOption'; Value = 0 },
    @{ Key = $explorerKey; Name = 'ShowHibernateOption'; Value = 0 },
    @{ Key = $auKey; Name = 'NoAutoRebootWithLoggedOnUsers'; Value = 1 }
)

# $null = this PC does not expose the setting (e.g. no power-button setting on some desktops).
function Get-PowerAc([string]$Sub, [string]$Setting) {
    $text = (powercfg /q SCHEME_CURRENT $Sub $Setting 2>&1) -join "`n"
    if ($text -match '(?im)AC Power Setting Index:\s*0x([0-9a-f]+)') { [Convert]::ToInt32($Matches[1], 16) } else { $null }
}
function Set-PowerAc([string]$Sub, [string]$Setting, [int]$Value) {
    $out = powercfg /setacvalueindex SCHEME_CURRENT $Sub $Setting $Value 2>&1
    if ($LASTEXITCODE -ne 0) { throw "powercfg could not set ${Setting}: $out" }
}
function Get-RegValue([string]$Key, [string]$Name) {
    $v = Get-ItemProperty -Path $Key -Name $Name -ErrorAction SilentlyContinue
    if ($v) { $v.$Name } else { $null }
}
function Set-RegValue([string]$Key, [string]$Name, $Value) {
    if ($null -eq $Value) { Remove-ItemProperty -Path $Key -Name $Name -ErrorAction SilentlyContinue; return }
    if (-not (Test-Path -LiteralPath $Key)) { $null = New-Item -Path $Key -Force }
    New-ItemProperty -Path $Key -Name $Name -Value $Value -PropertyType DWord -Force | Out-Null
}

if ($Undo) {
    if (-not (Test-Path -LiteralPath $backupPath)) { Write-Error "No backup at $backupPath - nothing to undo."; exit 1 }
    $b = Get-Content -LiteralPath $backupPath -Raw | ConvertFrom-Json
    foreach ($p in $b.power) { if ($null -ne $p.Value) { Set-PowerAc $p.Sub $p.Setting ([int]$p.Value) } }
    powercfg /setactive SCHEME_CURRENT
    foreach ($r in $b.registry) { Set-RegValue $r.Key $r.Name $r.Value }
    Write-Output "Undone from $backupPath."
    exit 0
}

# Backup first; a second run keeps the first backup (the real old values).
if (-not (Test-Path -LiteralPath $backupPath)) {
    $backup = [ordered]@{
        power = @($powerItems | ForEach-Object { [ordered]@{ Sub = $_.Sub; Setting = $_.Setting; Value = (Get-PowerAc $_.Sub $_.Setting) } })
        registry = @($regItems | ForEach-Object { [ordered]@{ Key = $_.Key; Name = $_.Name; Value = (Get-RegValue $_.Key $_.Name) } })
        saved = (Get-Date).ToString('o')
    }
    $null = New-Item -ItemType Directory -Path (Split-Path -Parent $backupPath) -Force
    $backup | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $backupPath -Encoding utf8
    Write-Output "backup: $backupPath"
} else { Write-Output "backup kept from the first run: $backupPath" }

foreach ($p in $powerItems) {
    if ($null -eq (Get-PowerAc $p.Sub $p.Setting)) { Write-Output "power: $($p.Setting) not on this PC - skipped"; continue }
    Set-PowerAc $p.Sub $p.Setting $p.Value
    Write-Output "power: $($p.Setting) = $(Get-PowerAc $p.Sub $p.Setting)"
}
powercfg /setactive SCHEME_CURRENT
foreach ($r in $regItems) { Set-RegValue $r.Key $r.Name $r.Value; Write-Output "policy: $($r.Name) = $(Get-RegValue $r.Key $r.Name)" }
Write-Output "Done: no sleep, shutdown and restart still work. Undo: pwsh -File I:\herdr\setup-nosleep.ps1 -Undo"
