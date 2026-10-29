# Runs one go command in app\ under the shared Go lock, with build.ps1's offline settings.
#   pwsh -NoProfile -File app\tests\go-locked.ps1 test ./internal/deck -run TestX -count=1
# Exit code = go's exit code. No param() block on purpose: it would make this an advanced script
# and PowerShell would take go flags like -o as its own common parameters (-OutVariable ...).
$ErrorActionPreference = 'Stop'
$goArgs = @($args)
. (Join-Path $PSScriptRoot 'go-lock.ps1')
$go = @($env:DECK_GO, 'D:\tools\go\go1.26.10\go\bin\go.exe', 'C:\Users\mflma\tools\go\go1.26.1\go\bin\go.exe') |
    Where-Object { $_ -and (Test-Path -LiteralPath $_ -PathType Leaf) } | Select-Object -First 1
if (-not $go) { $go = (Get-Command go.exe -ErrorAction Stop).Source }
$app = Split-Path -Parent $PSScriptRoot
$env:GOCACHE = Join-Path $PSScriptRoot 'go-cache'; $env:GOTMPDIR = Join-Path $PSScriptRoot 'go-tmp'
$env:GOPROXY = 'off'; $env:GOSUMDB = 'off'; $env:GOTOOLCHAIN = 'local'
New-Item -ItemType Directory -Path $env:GOCACHE, $env:GOTMPDIR -Force | Out-Null
$lock = Enter-GoLock
try {
    Push-Location -LiteralPath $app
    & $go @goArgs
    $code = $LASTEXITCODE
} finally { Pop-Location; Exit-GoLock $lock }
exit $code
