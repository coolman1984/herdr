# Split the focused Herdr pane into N equal columns (or rows), each a new shell in
# the same folder. Bound to ctrl+b shift+v (4 columns) in Herdr's config.
#
#   pwsh -File herdr-columns.ps1                 4 equal columns
#   pwsh -File herdr-columns.ps1 -Count 3        3 equal columns
#   pwsh -File herdr-columns.ps1 -Direction down 4 equal rows
#
# Herdr's --ratio is the share the ORIGINAL pane keeps (LESSONS #17), so N equal
# parts = split with 1/N, then split the new pane with 1/(N-1), ... down to 1/2.
[CmdletBinding()]
param(
    [ValidateRange(2, 8)] [int]$Count = 4,
    [ValidateSet("right", "down")] [string]$Direction = "right",
    [string]$Pane
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"

function Show-Error([string]$Message) {
    [System.Windows.Forms.MessageBox]::Show($Message, "Herdr - columns", "OK", "Error") | Out-Null
    exit 1
}

$currentArgs = @("pane", "current")
if ($Pane) { $currentArgs += @("--pane", $Pane) }
$output = & $herdr @currentArgs 2>&1
if ($LASTEXITCODE -ne 0) { Show-Error "No focused pane:`n$($output -join "`n")" }
$current = (($output -join "`n") | ConvertFrom-Json).result.pane
$cwd = $current.cwd

$target = $current.pane_id
for ($remaining = $Count; $remaining -ge 2; $remaining--) {
    $ratio = [string]::Format([cultureinfo]::InvariantCulture, "{0:0.####}", 1.0 / $remaining)
    $splitArgs = @("pane", "split", $target, "--direction", $Direction, "--ratio", $ratio, "--no-focus")
    if ($cwd) { $splitArgs += @("--cwd", $cwd) }
    $output = & $herdr @splitArgs 2>&1
    if ($LASTEXITCODE -ne 0) { Show-Error "Split failed after $($Count - $remaining) of $($Count - 1):`n$($output -join "`n")" }
    $target = (($output -join "`n") | ConvertFrom-Json).result.pane.pane_id
}
