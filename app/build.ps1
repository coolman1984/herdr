[CmdletBinding()]
# -Output: where the exe goes (default app\HerdrDeck.exe); agents build to their own path.
param([string]$Output)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'tests\go-lock.ps1')
if (-not $Output) { $Output = Join-Path $PSScriptRoot 'HerdrDeck.exe' }
$Output = [IO.Path]::GetFullPath($Output)
$deckCandidates = @()
if ($env:DECK_GO) { $deckCandidates += $env:DECK_GO }
$deckCandidates += 'C:\Users\mflma\tools\go\go1.26.1\go\bin\go.exe'
$deckCandidates += 'D:\tools\go\go1.26.10\go\bin\go.exe'
$deckGo = $null
foreach ($deckCandidate in $deckCandidates) {
    if ($deckCandidate -and (Test-Path -LiteralPath $deckCandidate -PathType Leaf)) { $deckGo = $deckCandidate; break }
}
if (-not $deckGo) {
    $deckFound = Get-Command go.exe -ErrorAction SilentlyContinue
    if ($deckFound) { $deckGo = $deckFound.Source }
}
if (-not $deckGo) {
    throw "Go not found: set DECK_GO to go.exe (tried: $($deckCandidates -join ', '), go.exe on PATH)"
}
Write-Output "Go: $deckGo"
$deckOldLocation = Get-Location
$deckOldCache = $env:GOCACHE
$deckOldTemp = $env:GOTMPDIR
$deckOldProxy = $env:GOPROXY
$deckOldSum = $env:GOSUMDB
$deckOldToolchain = $env:GOTOOLCHAIN
$deckLock = Enter-GoLock
try {
    Set-Location -LiteralPath $PSScriptRoot
    $env:GOCACHE = Join-Path $PSScriptRoot 'tests\go-cache'
    $env:GOTMPDIR = Join-Path $PSScriptRoot 'tests\go-tmp'
    $env:GOPROXY = 'off'
    $env:GOSUMDB = 'off'
    $env:GOTOOLCHAIN = 'local'
    New-Item -ItemType Directory -Path $env:GOCACHE, $env:GOTMPDIR -Force | Out-Null
    & $deckGo fmt ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go formatting failed.' }
    & $deckGo vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed.' }
    & $deckGo test ./... -count=1
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed.' }
    & $deckGo build -trimpath -ldflags '-H=windowsgui -s -w' -o $Output .
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
    $deckExe = Get-Item -LiteralPath $Output
    if ($deckExe.Length -ge 12MB) { throw "Deck is too large: $($deckExe.Length) bytes." }
    $deckHash = (Get-FileHash -LiteralPath $deckExe.FullName -Algorithm SHA256).Hash
    Write-Output "Built: $($deckExe.FullName)"
    Write-Output "Size: $($deckExe.Length) bytes ($([Math]::Round($deckExe.Length / 1MB, 2)) MiB)"
    Write-Output "SHA256: $deckHash"
} finally {
    Exit-GoLock $deckLock
    Set-Location -LiteralPath $deckOldLocation
    $env:GOCACHE = $deckOldCache
    $env:GOTMPDIR = $deckOldTemp
    $env:GOPROXY = $deckOldProxy
    $env:GOSUMDB = $deckOldSum
    $env:GOTOOLCHAIN = $deckOldToolchain
}
