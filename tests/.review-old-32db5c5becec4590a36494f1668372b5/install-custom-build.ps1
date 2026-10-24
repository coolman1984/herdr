# Build our Herdr (the source in .\herdr, with the "menu" feature) and make it
# the herdr.exe that the Herdr shortcut, the "herdr" command and PATH run.
#
# - Builds with cargo in release mode (first build takes several minutes).
# - Keeps the official binary next to it as herdr-official.exe (restore: run with -Restore).
# - Works while Herdr is running: Windows allows renaming a running exe. New Herdr
#   windows use the new build at once; the background server keeps the old one
#   until it restarts (herdr server stop, then open Herdr again - this closes panes).
#
# After "herdr update" installs a newer official version, run this script again
# (after pulling the new upstream source into .\herdr) to put the menus back.
[CmdletBinding()]
param(
    [switch]$Restore,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$source = Join-Path $PSScriptRoot "herdr"
$alias = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin"
$target = (Get-Item -LiteralPath $alias -Force).Target   # the active release folder
if (-not $target) { $target = $alias }
$live = Join-Path $target "herdr.exe"
$official = Join-Path $target "herdr-official.exe"

if ($Restore) {
    if (-not (Test-Path -LiteralPath $official)) { throw "No herdr-official.exe in $target." }
    $old = Join-Path $target ("herdr-custom-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".exe")
    Rename-Item -LiteralPath $live -NewName (Split-Path -Leaf $old)
    Copy-Item -LiteralPath $official -Destination $live
    Write-Host "Restored the official Herdr: $live"
    exit 0
}

if (-not $SkipBuild) {
    Push-Location $source
    try {
        cargo build --release --locked
        if ($LASTEXITCODE -ne 0) { throw "cargo build failed." }
    } finally {
        Pop-Location
    }
}
$built = Join-Path $source "target\release\herdr.exe"
if (-not (Test-Path -LiteralPath $built)) { throw "Build output not found: $built" }

$version = (& $built --version) -join " "
if ($LASTEXITCODE -ne 0) { throw "The new build does not run: $built --version" }

if (-not (Test-Path -LiteralPath $official)) {
    Copy-Item -LiteralPath $live -Destination $official
    Write-Host "Kept the official binary as $official"
}
# Rename (allowed while running), then copy the new one in.
$previous = Join-Path $target ("herdr-previous-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".exe")
Rename-Item -LiteralPath $live -NewName (Split-Path -Leaf $previous)
Copy-Item -LiteralPath $built -Destination $live
Write-Host "Installed our build ($version) at $live"
Write-Host "Old copy: $previous (safe to delete once no Herdr process uses it)"
