[CmdletBinding()]
param([switch]$SkipRestart)
$ErrorActionPreference = 'Stop'
$deckAppRoot=Split-Path -Parent $PSScriptRoot
$deckHerdr=Join-Path $env:LOCALAPPDATA 'Programs\Herdr\bin\herdr.exe'
$deckSavedAppData=$env:APPDATA; $deckSavedSocket=$env:HERDR_SOCKET_PATH
$env:APPDATA=Join-Path $PSScriptRoot 'runtime'
$env:HERDR_SOCKET_PATH=Join-Path $env:APPDATA 'herdr\sessions\lab\herdr.sock'
$deckClient=[Net.Http.HttpClient]::new(); $deckClient.Timeout=[Threading.Timeout]::InfiniteTimeSpan
$deckStream=$null; $deckAppProcess=$null; $deckOwnedWorkspace=$null
$deckEvidence=Join-Path $PSScriptRoot 'evidence'
New-Item -ItemType Directory -Path $deckEvidence -Force | Out-Null
$deckResults=[Collections.Generic.List[string]]::new()
function Lab([string[]]$CommandArgs) {
    $deckOut=& $deckHerdr --session lab @CommandArgs 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($deckOut -join "`n") }
    ($deckOut -join "`n") | ConvertFrom-Json
}
function DeckGet([string]$Route) { Invoke-RestMethod -Uri ($script:deckOrigin+$Route+$script:deckQuery) }
function DeckPost([string]$Route,$Data) {
    Invoke-RestMethod -Method Post -ContentType 'application/json' -Uri ($script:deckOrigin+$Route+$script:deckQuery) -Body ($Data|ConvertTo-Json -Depth 12 -Compress)
}
function Proof([string]$Text) { $deckResults.Add('PASS '+$Text); Write-Output ('PASS '+$Text) }
try {
    $deckCreated=Lab @('workspace','create','--cwd',$deckAppRoot,'--label','Deck verification','--no-focus')
    $deckOwnedWorkspace=$deckCreated.result.workspace.workspace_id
    $deckFirst=$deckCreated.result.root_pane.pane_id
    $deckSplit=Lab @('pane','split',$deckFirst,'--direction','right','--cwd',$deckAppRoot,'--no-focus')
    $deckSecond=$deckSplit.result.pane.pane_id
    $deckAddress=Join-Path $PSScriptRoot 'verify-address.json'
    $deckInfo=[Diagnostics.ProcessStartInfo]::new((Join-Path $deckAppRoot 'HerdrDeck.exe'))
    $deckInfo.UseShellExecute=$false; $deckInfo.CreateNoWindow=$true
    foreach($deckArg in @('--session','lab','--headless','--address-file',$deckAddress)){ $deckInfo.ArgumentList.Add($deckArg) }
    $deckAppProcess=[Diagnostics.Process]::Start($deckInfo)
    $deckDeadline=[DateTime]::UtcNow.AddSeconds(8)
    while(-not(Test-Path -LiteralPath $deckAddress)) { if($deckAppProcess.HasExited -or [DateTime]::UtcNow -gt $deckDeadline){throw 'Deck start failed'};Start-Sleep -Milliseconds 100 }
    $deckRecord=Get-Content -LiteralPath $deckAddress -Raw | ConvertFrom-Json
    $deckUri=[Uri]$deckRecord.url; $script:deckOrigin=$deckUri.GetLeftPart([UriPartial]::Authority); $script:deckQuery=$deckUri.Query
    $deckStream=$deckClient.GetStreamAsync("$deckOrigin/api/events$deckQuery").GetAwaiter().GetResult()
    Start-Sleep -Milliseconds 700
    $deckState=DeckGet '/api/state'; $deckPane=$deckState.panes | Where-Object pane_id -eq $deckFirst
    if(-not $deckPane){throw 'Pane missing from API state'}
    $deckKey=$deckPane.key
    DeckPost '/api/visible' @{keys=@($deckKey)} | Out-Null
    & curl.exe --silent --fail "$deckOrigin/api/state$deckQuery" --output (Join-Path $deckEvidence 'state.json')
    if($LASTEXITCODE -ne 0){throw 'curl state check failed'}
    Proof 'curl /api/state lists the real isolated lab panes'
    $deckSSEBuffer=[byte[]]::new(16384)
    $deckSSERead=$deckStream.ReadAsync($deckSSEBuffer,0,$deckSSEBuffer.Length)
    if(-not $deckSSERead.Wait(5000)){throw 'No SSE event received'}
    $deckSSECount=$deckSSERead.GetAwaiter().GetResult()
    [Text.Encoding]::UTF8.GetString($deckSSEBuffer,0,$deckSSECount) | Set-Content -LiteralPath (Join-Path $deckEvidence 'sse-sample.txt')
    Proof 'SSE state/diff stream received'
    DeckPost '/api/pane/prompt' @{key=$deckKey;text="Write-Output ('DECK-PROMPT-' + 'OK')"} | Out-Null
    Lab @('pane','wait-output',$deckFirst,'--match','DECK-PROMPT-OK','--timeout','6000') | Out-Null
    Proof 'shell prompt executed (computed marker appears in output)'
    DeckPost '/api/pane/text' @{key=$deckKey;text="Write-Output ('DECK-KEYS-' + 'OK')"} | Out-Null
    DeckPost '/api/pane/keys' @{key=$deckKey;keys=@('Enter')} | Out-Null
    Lab @('pane','wait-output',$deckFirst,'--match','DECK-KEYS-OK','--timeout','6000') | Out-Null
    Proof 'literal paste and Enter key executed'
    $deckGo='C:\Users\mflma\tools\go\go1.26.1\go\bin\go.exe'
    $env:GOCACHE=Join-Path $PSScriptRoot 'go-cache';$env:GOTMPDIR=Join-Path $PSScriptRoot 'go-tmp'
    $deckStub=Join-Path $PSScriptRoot 'codex.exe'
    & $deckGo build -o $deckStub (Join-Path $PSScriptRoot 'agentstub\main.go')
    if($LASTEXITCODE -ne 0){throw 'Transport fixture build failed'}
    Lab @('pane','run',$deckFirst,('& '+"'"+$deckStub.Replace("'","''")+"'")) | Out-Null
    Lab @('pane','wait-output',$deckFirst,'--match','transport fixture','--timeout','6000') | Out-Null
    Lab @('pane','report-agent',$deckFirst,'--source','deck-test','--agent','codex','--state','idle') | Out-Null
    DeckPost '/api/pane/prompt' @{key=$deckKey;text="Write-Output ('DECK-AGENT-' + 'OK')"} | Out-Null
    Lab @('pane','wait-output',$deckFirst,'--match','DECK-AGENT-OK','--timeout','6000') | Out-Null
    Proof 'agent prompt reaches the offline codex.exe transport fixture (no AI CLI/network/tools launched)'
    Lab @('pane','report-metadata',$deckFirst,'--title','Deck test HUD','--token','model=gpt-6.1-sol','--token','ctx=12%','--token','out=42','--ttl-ms','120000') | Out-Null
    Lab @('pane','run',$deckFirst,'BLOCK') | Out-Null
    Lab @('pane','report-agent',$deckFirst,'--source','deck-test','--agent','codex','--state','blocked') | Out-Null
    Start-Sleep -Milliseconds 2500
    $deckState=DeckGet '/api/state'; $deckPane=$deckState.panes|Where-Object pane_id -eq $deckFirst
    if(-not $deckPane.blocked -or $deckPane.tokens.ctx -ne '12%'){throw 'Blocked/HUD state missing'}
    $deckBlocked=Invoke-WebRequest -SkipHttpErrorCheck -Method Post -ContentType 'application/json' -Uri "$deckOrigin/api/pane/prompt$deckQuery" -Body (@{key=$deckKey;text='THIS MUST NOT BE SENT'}|ConvertTo-Json)
    if([int]$deckBlocked.StatusCode -ne 409){throw 'Blocked prompt was not rejected'}
    $deckStill=Lab @('pane','get',$deckFirst)
    if($deckStill.result.pane.agent_status -ne 'blocked'){throw 'Blocked pane was changed without explicit input'}
    Proof 'blocked state + model/context tokens shown; prompt rejected 409; waiting pane left unanswered'
    DeckPost '/api/pane/keys' @{key=$deckKey;keys=@('C-c')} | Out-Null
    Lab @('pane','release-agent',$deckFirst,'--source','deck-test','--agent','codex') | Out-Null
    DeckPost '/api/pane/rename' @{key=$deckKey;label='Deck verified shell'} | Out-Null
    $deckRenamed=Lab @('pane','get',$deckFirst)
    if($deckRenamed.result.pane.label -ne 'Deck verified shell'){throw 'Rename read-back failed'}
    $deckNew=DeckPost '/api/pane/split' @{key=$deckKey;direction='down';cwd=$deckAppRoot}
    $deckNewID=($deckNew.message|ConvertFrom-Json).result.pane.pane_id
    if(-not $deckNewID){throw 'Split returned no ID'}
    DeckPost '/api/pane/close' @{key=($deckState.session+'|'+$deckNewID)} | Out-Null
    $deckListed=Lab @('pane','list')
    if($deckNewID -in $deckListed.result.panes.pane_id){throw 'Close read-back failed'}
    Proof 'rename, downward split and close verified by Herdr JSON'
    DeckPost '/api/read-only' @{enabled=$true} | Out-Null
    $deckRefused=Invoke-WebRequest -SkipHttpErrorCheck -Method Post -ContentType 'application/json' -Uri "$deckOrigin/api/pane/text$deckQuery" -Body (@{key=$deckKey;text='blocked write'}|ConvertTo-Json)
    if([int]$deckRefused.StatusCode -ne 403){throw 'Read-only mode bypassed'}
    DeckPost '/api/read-only' @{enabled=$false} | Out-Null
    Proof 'backend read-only mode rejects mutation with 403'
    $deckTasks=DeckGet '/api/tasks'; if($deckTasks.tasks.Count -lt 1){throw 'Task list empty'}
    $deckHealth=DeckGet '/api/health'; if($deckHealth.goroutines -lt 1 -or -not $deckHealth.disks){throw 'Health missing'}
    Proof ('task allow-list and Windows RAM/disk/process health read back; goroutines='+$deckHealth.goroutines)
    $deckPE=[IO.File]::ReadAllBytes((Join-Path $deckAppRoot 'HerdrDeck.exe'));$deckPEOffset=[BitConverter]::ToInt32($deckPE,0x3c);$deckSubsystem=[BitConverter]::ToUInt16($deckPE,$deckPEOffset+24+68)
    if($deckSubsystem -ne 2){throw 'Executable is not Windows GUI subsystem'}
    Proof 'PE subsystem=2 (Windows GUI); hidden child console absence covered by executable-stub test'
    $deckAgain=[Diagnostics.Process]::Start($deckInfo)
    if(-not $deckAgain.WaitForExit(5000) -or $deckAgain.ExitCode -ne 0){throw 'Second launch did not reopen first instance'}
    $deckAgain.Dispose(); Proof 'second launch reaches the existing instance and exits 0'
    if(-not $SkipRestart){
        Lab @('server','stop') | Out-Null
        Start-Sleep -Milliseconds 2400
        $deckOffline=DeckGet '/api/state';if($deckOffline.herdr.running -or -not $deckOffline.error){throw 'Stopped state not visible'}
        $deckServerInfo=[Diagnostics.ProcessStartInfo]::new($deckHerdr);$deckServerInfo.UseShellExecute=$false;$deckServerInfo.CreateNoWindow=$true
        foreach($deckArg in @('--session','lab','server')){$deckServerInfo.ArgumentList.Add($deckArg)}
        $deckServerProcess=[Diagnostics.Process]::Start($deckServerInfo)
        $deckDeadline=[DateTime]::UtcNow.AddSeconds(12)
        do{Start-Sleep -Milliseconds 700;$deckBack=DeckGet '/api/state'}while(-not $deckBack.herdr.running -and [DateTime]::UtcNow -lt $deckDeadline)
        if(-not $deckBack.herdr.running){throw 'Deck did not recover after lab restart'}
        Proof 'stopped lab displays error and Deck recovers after lab restart'
    }
    $deckResults | Set-Content -LiteralPath (Join-Path $deckEvidence 'lab-results.txt')
} finally {
    if($deckStream){$deckStream.Dispose()};$deckClient.Dispose()
    if($deckAppProcess){if(-not $deckAppProcess.WaitForExit(6000)){$deckAppProcess.Kill($true);Write-Warning 'Deck needed forced cleanup'};$deckAppProcess.Dispose()}
    if($deckOwnedWorkspace){$deckRemaining=Lab @('workspace','list');if($deckOwnedWorkspace -in $deckRemaining.result.workspaces.workspace_id){Lab @('workspace','close',$deckOwnedWorkspace)|Out-Null}}
    $deckResults | Set-Content -LiteralPath (Join-Path $deckEvidence 'lab-results.txt')
    $env:APPDATA=$deckSavedAppData;$env:HERDR_SOCKET_PATH=$deckSavedSocket
}
