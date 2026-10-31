# setup-remote.ps1 - one-time setup so the owner reaches this PC's Herdr from anywhere (Plan A, HISTORY 64).
#
# Run ONCE in an administrator PowerShell 7:  pwsh -File I:\herdr\setup-remote.ps1
# What it does (needs admin, so the owner runs it, not an agent):
#   1. OpenSSH server: start now and at every boot.
#   2. Firewall: SSH answers only from the Tailscale network (100.64.0.0/10), never the open internet.
#   3. SSH sessions open PowerShell 7, so `herdr` / `herdr --remote <pc>` work over SSH.
#      Old DefaultShell value is printed first (backup) and restored with -Undo.
# Sign-ins stay with the owner: `tailscale up --unattended` on this PC, the Tailscale app on the phone.
param([switch]$Undo)
$ErrorActionPreference = 'Stop'
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    # Ask Windows for admin rights (the owner clicks Yes) and run again in a window that stays open.
    $relaunch = @('-NoProfile', '-NoExit', '-File', "`"$PSCommandPath`"") + $(if ($Undo) { @('-Undo') })
    try { Start-Process -FilePath (Get-Command pwsh).Source -ArgumentList $relaunch -Verb RunAs }
    catch { Write-Error "Admin rights were refused, nothing changed: $($_.Exception.Message)"; exit 1 }
    Write-Output "Continuing in the new administrator window."
    exit 0
}

$key = 'HKLM:\SOFTWARE\OpenSSH'
$backup = Join-Path $env:APPDATA 'herdr\openssh-defaultshell.bak.txt'
$rules = @(Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object { $_.Name -like 'OpenSSH*' -or $_.DisplayName -like 'OpenSSH*' })

if ($Undo) {
    Set-Service sshd -StartupType Manual; Stop-Service sshd
    foreach ($r in $rules) { Set-NetFirewallRule -Name $r.Name -RemoteAddress Any }
    $old = if (Test-Path -LiteralPath $backup) { (Get-Content -LiteralPath $backup -Raw).Trim() } else { '' }
    if ($old) { Set-ItemProperty -Path $key -Name DefaultShell -Value $old } else { Remove-ItemProperty -Path $key -Name DefaultShell -ErrorAction SilentlyContinue }
    Write-Output "Undone: sshd manual + stopped, firewall rule open to Any, DefaultShell restored to '$old'."
    exit 0
}

# 1. SSH server on, now and at boot.
Set-Service sshd -StartupType Automatic
Start-Service sshd
Write-Output "sshd: $((Get-Service sshd).Status), start type $((Get-Service sshd).StartType)"

# 2. Only the Tailscale network may connect.
if (-not $rules) {
    New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -LocalPort 22 -Action Allow -RemoteAddress '100.64.0.0/10' | Out-Null
    Write-Output "firewall: new rule, port 22 from 100.64.0.0/10 only"
} else {
    foreach ($r in $rules) { Set-NetFirewallRule -Name $r.Name -RemoteAddress '100.64.0.0/10' }
    Write-Output "firewall: $($rules.Count) OpenSSH rule(s) now allow 100.64.0.0/10 only"
}

# 3. PowerShell 7 as the SSH shell (backup of the old value first).
$pwsh = (Get-Command pwsh).Source
$old = (Get-ItemProperty -Path $key -Name DefaultShell -ErrorAction SilentlyContinue).DefaultShell
$null = New-Item -ItemType Directory -Path (Split-Path -Parent $backup) -Force
Set-Content -LiteralPath $backup -Value ([string]$old) -Encoding utf8
New-ItemProperty -Path $key -Name DefaultShell -Value $pwsh -PropertyType String -Force | Out-Null
Write-Output "ssh shell: $pwsh (old value '$old' saved in $backup)"
Write-Output "Done. Next: 'tailscale up --unattended' here (your sign-in), then from the phone: ssh $env:USERNAME@<pc name in Tailscale>"
