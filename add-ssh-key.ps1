# add-ssh-key.ps1 - let a device log in to this PC over SSH with a key instead of the password (HISTORY 65).
#
# The device makes its own key pair; only the PUBLIC key (one line "ssh-ed25519 AAAA... name") comes here.
# A private key is refused. From the phone (Termux), after `ssh-keygen -t ed25519`:
#   cat ~/.ssh/id_ed25519.pub | ssh mflma@fawzy "pwsh -NoProfile -File I:/herdr/add-ssh-key.ps1"
# (an SSH password login of an administrator is already elevated, so it can write the admin key file).
# Or on this PC:  pwsh -File I:\herdr\add-ssh-key.ps1 -Key "ssh-ed25519 AAAA... phone"
# Members of Administrators use C:\ProgramData\ssh\administrators_authorized_keys (sshd_config
# "Match Group administrators"), which must belong to Administrators + SYSTEM only; others use
# ~\.ssh\authorized_keys. -Path overrides the file (tests). A key already present is not added twice.
param(
    [string]$Key = '',
    [string]$Path = ''
)
$ErrorActionPreference = 'Stop'

if (-not $Key -and [Console]::IsInputRedirected) { $Key = [Console]::In.ReadToEnd() }
$Key = ($Key -split "`r?`n" | Where-Object { $_.Trim() } | Select-Object -First 1)
if ($Key) { $Key = $Key.Trim() }
if (-not $Key) { Write-Error "No key given. Pass -Key `"ssh-ed25519 AAAA... name`" or pipe the .pub file in."; exit 1 }
if ($Key -match 'PRIVATE KEY') { Write-Error "That is a PRIVATE key - it must never leave the device. Send the .pub file instead."; exit 1 }
if ($Key -notmatch '^(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp(256|384|521)|sk-ssh-ed25519@openssh\.com|sk-ecdsa-sha2-nistp256@openssh\.com) ([A-Za-z0-9+/]+={0,2})( .*)?$') {
    Write-Error "Not an SSH public key line: $($Key.Substring(0, [Math]::Min(40, $Key.Length)))..."; exit 1
}
$blob = $Matches[3]

$adminFile = $false
if (-not $Path) {
    $me = [Security.Principal.WindowsIdentity]::GetCurrent()
    $inAdmins = [bool]($me.Groups | Where-Object { $_.Value -eq 'S-1-5-32-544' })
    if ($inAdmins) { $Path = Join-Path $env:ProgramData 'ssh\administrators_authorized_keys'; $adminFile = $true }
    else { $Path = Join-Path $env:USERPROFILE '.ssh\authorized_keys' }
}
if ($adminFile -and -not ([Security.Principal.WindowsPrincipal]$me).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    # Interactive window without admin rights: ask Windows (the owner clicks Yes), key passed along.
    try { Start-Process -FilePath (Get-Command pwsh).Source -ArgumentList @('-NoProfile', '-NoExit', '-File', "`"$PSCommandPath`"", '-Key', "`"$Key`"") -Verb RunAs }
    catch { Write-Error "Admin rights were refused, nothing changed: $($_.Exception.Message)"; exit 1 }
    Write-Output "Continuing in the new administrator window."
    exit 0
}

$null = New-Item -ItemType Directory -Path (Split-Path -Parent $Path) -Force
$existing = if (Test-Path -LiteralPath $Path) { @(Get-Content -LiteralPath $Path) } else { @() }
if ($existing | Where-Object { ($_ -split ' ')[1] -eq $blob }) {
    Write-Output "Key already allowed in $Path - nothing changed."
    exit 0
}
Add-Content -LiteralPath $Path -Value $Key -Encoding utf8NoBOM
if ($adminFile) {
    # sshd ignores this file unless only Administrators and SYSTEM can write it.
    $out = icacls $Path /inheritance:r /grant '*S-1-5-32-544:F' /grant '*S-1-5-18:F' 2>&1
    if ($LASTEXITCODE -ne 0) { Write-Error "Key added, but setting the file's permissions failed: $out"; exit 1 }
}
$label = (($Key -split ' ', 3) + @('', '', ''))[2]
Write-Output "Key added to $Path ($(@($existing).Count + 1) key(s) now)$(if ($label) { ": $label" })"
