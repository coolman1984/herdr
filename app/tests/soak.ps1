[CmdletBinding()]
param([ValidateRange(1, 3600)][int]$Seconds = 600)
$ErrorActionPreference = 'Stop'
$deckRoot = Split-Path -Parent $PSScriptRoot
$deckHerdr = Join-Path $env:LOCALAPPDATA 'Programs\Herdr\bin\herdr.exe'
$deckOldAppData = $env:APPDATA
$deckOldSocket = $env:HERDR_SOCKET_PATH
$env:APPDATA = Join-Path $PSScriptRoot 'runtime'
$env:HERDR_SOCKET_PATH = Join-Path $env:APPDATA 'herdr\sessions\lab\herdr.sock'
$deckWorkspace = $null
$deckProcess = $null
$deckHttp = [Net.Http.HttpClient]::new()
$deckHttp.Timeout = [Threading.Timeout]::InfiniteTimeSpan
$deckStream = $null
function Invoke-Lab([string[]]$CommandArgs) {
    $deckOutput = & $deckHerdr --session lab @CommandArgs 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($deckOutput -join "`n") }
    ($deckOutput -join "`n") | ConvertFrom-Json
}
try {
    $deckStatus = Invoke-Lab @('status', 'server', '--json')
    if (-not $deckStatus.running) { throw 'Start the workspace-only lab server first; this script never uses the owner session.' }
    $deckCreated = Invoke-Lab @('workspace', 'create', '--cwd', $deckRoot, '--label', 'Deck soak', '--no-focus')
    $deckWorkspace = $deckCreated.result.workspace.workspace_id
    $deckPanes = @($deckCreated.result.root_pane.pane_id)
    1..3 | ForEach-Object {
        $deckSplit = Invoke-Lab @('pane', 'split', $deckPanes[-1], '--direction', 'right', '--cwd', $deckRoot, '--no-focus')
        $deckPanes += $deckSplit.result.pane.pane_id
    }
    $deckAddress = Join-Path $PSScriptRoot 'soak-address.json'
    $deckInfo = [Diagnostics.ProcessStartInfo]::new((Join-Path $deckRoot 'HerdrDeck.exe'))
    $deckInfo.UseShellExecute = $false
    $deckInfo.CreateNoWindow = $true
    foreach ($deckArg in @('--session', 'lab', '--headless', '--address-file', $deckAddress)) { $deckInfo.ArgumentList.Add($deckArg) }
    $deckProcess = [Diagnostics.Process]::Start($deckInfo)
    $deckDeadline = [DateTime]::UtcNow.AddSeconds(8)
    while (-not (Test-Path -LiteralPath $deckAddress)) {
        if ($deckProcess.HasExited -or [DateTime]::UtcNow -gt $deckDeadline) { throw 'Deck did not start.' }
        Start-Sleep -Milliseconds 100
    }
    $deckRecord = Get-Content -LiteralPath $deckAddress -Raw | ConvertFrom-Json
    $deckBase = $deckRecord.url -replace '/\?t=', '/api/'
    $deckUri = [Uri]$deckRecord.url
    $deckOrigin = $deckUri.GetLeftPart([UriPartial]::Authority)
    $deckQuery = $deckUri.Query
    $deckStream = $deckHttp.GetStreamAsync("$deckOrigin/api/events$deckQuery").GetAwaiter().GetResult()
    Start-Sleep -Milliseconds 500
    $deckState = Invoke-RestMethod -Uri "$deckOrigin/api/state$deckQuery"
    $deckKeys = @($deckState.panes | Where-Object { $_.workspace_id -eq $deckWorkspace } | ForEach-Object { $_.key })
    Invoke-RestMethod -Method Post -ContentType 'application/json' -Uri "$deckOrigin/api/visible$deckQuery" -Body (@{ keys=$deckKeys } | ConvertTo-Json) | Out-Null
    $deckCommand = 'Set-PSReadLineOption -HistorySaveStyle SaveNothing; for($deckTick=0;$deckTick -lt 18000;$deckTick++){ Write-Output ("DECK-STREAM " + $deckTick); Start-Sleep -Milliseconds 100 }'
    foreach ($deckPane in $deckPanes[0..1]) { Invoke-Lab @('pane', 'run', $deckPane, $deckCommand) | Out-Null }
    $deckTimer = [Diagnostics.Stopwatch]::StartNew()
    $deckProcess.Refresh()
    $deckLastCPU = $deckProcess.TotalProcessorTime.TotalSeconds
    $deckLastTime = 0.0
    $deckNextSample = 5.0
    $deckBytes = 0L
    $deckBuffer = [byte[]]::new(65536)
    $deckRead = $deckStream.ReadAsync($deckBuffer, 0, $deckBuffer.Length)
    $deckRows = [Collections.Generic.List[object]]::new()
    $deckCsv = Join-Path $PSScriptRoot 'soak.csv'
    while ($deckTimer.Elapsed.TotalSeconds -lt $Seconds) {
        if ($deckRead.Wait(100)) {
            $deckCount = $deckRead.GetAwaiter().GetResult()
            if ($deckCount -eq 0) { throw 'SSE stream closed during soak.' }
            $deckBytes += $deckCount
            $deckRead = $deckStream.ReadAsync($deckBuffer, 0, $deckBuffer.Length)
        }
        if ($deckTimer.Elapsed.TotalSeconds -ge $deckNextSample) {
            $deckProcess.Refresh()
            $deckNow = $deckTimer.Elapsed.TotalSeconds
            $deckCPU = $deckProcess.TotalProcessorTime.TotalSeconds
            $deckHealth = Invoke-RestMethod -Uri "$deckOrigin/api/health$deckQuery"
            $deckRow = [pscustomobject]@{
                seconds=[Math]::Round($deckNow, 2)
                rss_mb=[Math]::Round($deckProcess.WorkingSet64 / 1MB, 3)
                handles=$deckProcess.HandleCount
                goroutines=$deckHealth.goroutines
                cpu_one_core_pct=[Math]::Round(100*($deckCPU-$deckLastCPU)/($deckNow-$deckLastTime), 3)
                sse_bytes=$deckBytes
            }
            $deckRows.Add($deckRow)
            $deckRows | Export-Csv -LiteralPath $deckCsv -NoTypeInformation
            $deckLastCPU=$deckCPU; $deckLastTime=$deckNow; $deckNextSample+=5
        }
    }
    if ($deckRows.Count -lt 2) { throw 'Not enough soak samples.' }
    $deckFirst=$deckRows[0]; $deckLast=$deckRows[-1]
    $deckSlope=($deckLast.rss_mb-$deckFirst.rss_mb)/(($deckLast.seconds-$deckFirst.seconds)/60)
    $deckSummary=[pscustomobject]@{
        duration_sec=$Seconds; samples=$deckRows.Count; four_panes=($deckPanes.Count -eq 4); streaming_panes=2
        rss_start_mb=$deckFirst.rss_mb; rss_end_mb=$deckLast.rss_mb
        rss_slope_mb_per_min=[Math]::Round($deckSlope,4)
        cpu_mean_one_core_pct=[Math]::Round(($deckRows.cpu_one_core_pct | Measure-Object -Average).Average,3)
        cpu_peak_one_core_pct=($deckRows.cpu_one_core_pct | Measure-Object -Maximum).Maximum
        goroutine_growth=($deckLast.goroutines-$deckFirst.goroutines)
        handle_growth=($deckLast.handles-$deckFirst.handles)
        sse_bytes_per_sec=[Math]::Round($deckBytes/$Seconds,2)
    }
    $deckSummary | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'soak-summary.json')
    $deckSummary | Format-List
} finally {
    if ($deckStream) { $deckStream.Dispose() }
    $deckHttp.Dispose()
    if ($deckProcess) {
        if (-not $deckProcess.WaitForExit(6000)) { $deckProcess.Kill($true); Write-Warning 'Deck needed forced cleanup after the SSE closed.' }
        $deckProcess.Dispose()
    }
    if ($deckWorkspace) { Invoke-Lab @('workspace','close',$deckWorkspace) | Out-Null }
    $env:APPDATA=$deckOldAppData
    $env:HERDR_SOCKET_PATH=$deckOldSocket
}
