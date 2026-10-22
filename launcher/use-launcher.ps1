# Point "herdr" (Explorer address bar / Win+R) and the Desktop + Start-menu Herdr.lnk at
# HerdrLauncher.exe, so starting Herdr never flashes a console (HISTORY Phase 23).
# The owner runs this once:  pwsh -NoProfile -File launcher\use-launcher.ps1
#   -Undo   puts the old values back from the newest backup.
# Backups: %APPDATA%\herdr\backup\launcher-<time>\ (App Path .reg + both .lnk files).
# Herdr.cmd stays as the fallback; nothing is deleted.
param([switch]$Undo)
$ErrorActionPreference = "Stop"

$exe = Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "HerdrLauncher.exe"
$regKey = "HKCU\Software\Microsoft\Windows\CurrentVersion\App Paths\herdr.exe"
$psKey = "Registry::$regKey"
$links = [ordered]@{
    "desktop-Herdr.lnk"   = Join-Path ([Environment]::GetFolderPath("Desktop")) "Herdr.lnk"
    "startmenu-Herdr.lnk" = Join-Path ([Environment]::GetFolderPath("Programs")) "Herdr.lnk"
}
$backupRoot = Join-Path $env:APPDATA "herdr\backup"
$sh = New-Object -ComObject WScript.Shell

if ($Undo) {
    $bk = Get-ChildItem -LiteralPath $backupRoot -Directory -Filter "launcher-*" -ErrorAction SilentlyContinue |
        Sort-Object Name | Select-Object -Last 1
    if (-not $bk) { throw "No launcher backup under $backupRoot" }
    reg import (Join-Path $bk.FullName "apppath-herdr.reg") 2>$null
    if ($LASTEXITCODE -ne 0) { throw "reg import failed" }
    foreach ($name in $links.Keys) {
        $saved = Join-Path $bk.FullName $name
        if (Test-Path -LiteralPath $saved) { Copy-Item -LiteralPath $saved $links[$name] -Force }
    }
    Write-Output "Restored from $($bk.FullName)"
} else {
    if (-not (Test-Path -LiteralPath $exe)) { throw "Not built: $exe (pwsh -File launcher\build.ps1)" }
    $bk = Join-Path $backupRoot ("launcher-" + (Get-Date -Format "yyyyMMdd-HHmmss"))
    New-Item -ItemType Directory -Force -Path $bk | Out-Null
    reg export $regKey (Join-Path $bk "apppath-herdr.reg") /y | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "reg export failed" }
    foreach ($name in $links.Keys) {
        $p = $links[$name]
        if (-not (Test-Path -LiteralPath $p)) { Write-Output "skip (missing): $p"; continue }
        Copy-Item -LiteralPath $p (Join-Path $bk $name)
        $l = $sh.CreateShortcut($p)
        $l.TargetPath = $exe          # icon, working folder and description are kept
        $l.Arguments = ""
        $l.Save()
    }
    Set-ItemProperty -LiteralPath $psKey -Name "(default)" -Value $exe
    Write-Output "Backup: $bk"
}

# Read the result back.
Write-Output ("App Path: " + (Get-ItemProperty -LiteralPath $psKey)."(default)")
foreach ($p in $links.Values) {
    if (Test-Path -LiteralPath $p) {
        $l = $sh.CreateShortcut($p)
        Write-Output ("{0}: {1} {2} (icon {3})" -f $p, $l.TargetPath, $l.Arguments, $l.IconLocation)
    }
}
