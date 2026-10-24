$ErrorActionPreference='Stop'
$deckRoot=Split-Path -Parent $PSScriptRoot
$deckGo='C:\Users\mflma\tools\go\go1.26.1\go\bin\go.exe'
$deckParser=Join-Path $deckRoot 'internal\deck\model.go'
$deckMutation=Join-Path $PSScriptRoot 'parser-mutant.txt'
$deckOverlay=Join-Path $PSScriptRoot 'parser-overlay.json'
$deckOriginal=[IO.File]::ReadAllText($deckParser)
$deckBroken=$deckOriginal.Replace('if p.ID == "" {','if p.ID != "" {')
if($deckBroken -eq $deckOriginal){throw 'Parser mutation location not found'}
[IO.File]::WriteAllText($deckMutation,$deckBroken)
@{Replace=@{$deckParser=$deckMutation}}|ConvertTo-Json -Depth 4|Set-Content -LiteralPath $deckOverlay
$deckSaved=Get-Location
try{
    Set-Location -LiteralPath $deckRoot
    $env:GOCACHE=Join-Path $PSScriptRoot 'go-cache'
    $env:GOTMPDIR=Join-Path $PSScriptRoot 'go-tmp'
    $deckOutput=& $deckGo test -overlay $deckOverlay ./internal/deck -run 'TestSnapshot|TestRefreshDisplaysBlocked' -count=1 2>&1
    $deckExit=$LASTEXITCODE
    $deckOutput | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'mutation-result.txt')
    $deckFailCount=@($deckOutput|Where-Object{$_ -match '^--- FAIL:'}).Count
    if($deckExit -eq 0 -or $deckFailCount -lt 1){throw 'Tests did not catch the deliberately broken parser'}
    Write-Output "Mutation caught: $deckFailCount behavioral tests failed; original source was never edited."
}finally{Set-Location -LiteralPath $deckSaved}
