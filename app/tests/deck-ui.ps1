# Headless UI harness for Herdr Deck: isolated APPDATA (tests\runtime), the Deck in --headless mode,
# ONE headless Edge with a throwaway profile, driven over the DevTools protocol (CDP). Dot-source it:
#   . app\tests\deck-ui.ps1; Start-DeckUi -Lab; Open-DeckPage '?mock=1'; Invoke-DeckJs 'innerWidth'; Stop-DeckUi
# Nothing touches the owner's live Herdr session; it stops only the processes it started.
$ErrorActionPreference = 'Stop'
$script:ui = @{}
$script:uiRoot = $PSScriptRoot
$script:uiHerdr = Join-Path $env:LOCALAPPDATA 'Programs\Herdr\bin\herdr.exe'

function Start-UiProcess([string]$File, [string[]]$ArgList) {
    $info = [Diagnostics.ProcessStartInfo]::new($File)
    $info.UseShellExecute = $false; $info.CreateNoWindow = $true
    foreach ($a in $ArgList) { $info.ArgumentList.Add($a) }
    [Diagnostics.Process]::Start($info)
}
function Invoke-Lab([string[]]$CommandArgs) {
    $info = [Diagnostics.ProcessStartInfo]::new($script:uiHerdr)
    $info.UseShellExecute = $false; $info.CreateNoWindow = $true
    $info.RedirectStandardOutput = $true; $info.RedirectStandardError = $true
    foreach ($part in @('--session', 'lab') + $CommandArgs) { $info.ArgumentList.Add($part) }
    $process = [Diagnostics.Process]::Start($info)
    try {
        $stdout = $process.StandardOutput.ReadToEndAsync(); $stderr = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit(30000)) { $process.Kill($true); throw "lab CLI timed out: $($CommandArgs -join ' ')" }
        $text = $stdout.GetAwaiter().GetResult(); $diagnostic = $stderr.GetAwaiter().GetResult()
        if ($process.ExitCode -ne 0) { throw "herdr --session lab $($CommandArgs -join ' '): $text $diagnostic" }
    } finally { $process.Dispose() }
    if ($text.TrimStart().StartsWith('{')) { $text | ConvertFrom-Json } else { $text }
}
function Start-DeckUi([switch]$Lab, [int]$Port = 9333, [string[]]$EdgeArgs = @()) {
    $script:ui.SavedAppData = $env:APPDATA; $script:ui.SavedSocket = $env:HERDR_SOCKET_PATH
    $script:ui.SavedRoot = $env:HERDR_DECK_ROOT
    $env:APPDATA = Join-Path $script:uiRoot 'runtime'
    $env:HERDR_SOCKET_PATH = Join-Path $env:APPDATA 'herdr\sessions\lab\herdr.sock'
    $env:HERDR_DECK_ROOT = Split-Path -Parent (Split-Path -Parent $script:uiRoot)
    if ($Lab) {
        $alive = $false
        try { Invoke-Lab @('workspace', 'list') | Out-Null; $alive = $true } catch { $alive = $false }
        if ($alive) { throw 'The lab server is already running; refusing to share a server owned by another run.' }
        if (-not $alive) {
            $script:ui.Herdr = Start-UiProcess $script:uiHerdr @('--session', 'lab', 'server')
            $deadline = [DateTime]::UtcNow.AddSeconds(15)
            while ($true) {
                try { Invoke-Lab @('workspace', 'list') | Out-Null; break } catch { if ([DateTime]::UtcNow -gt $deadline) { throw "lab Herdr did not start: $_" } }
                Start-Sleep -Milliseconds 300
            }
        }
    }
    $address = Join-Path $script:uiRoot 'tmp-ui-address.json'
    Remove-Item -LiteralPath $address -ErrorAction SilentlyContinue
    $deck = Join-Path (Split-Path -Parent $script:uiRoot) 'HerdrDeck.exe'
    $script:ui.Deck = Start-UiProcess $deck @('--session', 'lab', '--headless', '--address-file', $address)
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    while (-not (Test-Path -LiteralPath $address)) {
        if ($script:ui.Deck.HasExited -or [DateTime]::UtcNow -gt $deadline) { throw "Deck start failed (exit $($script:ui.Deck.ExitCode))" }
        Start-Sleep -Milliseconds 100
    }
    $uri = [Uri](Get-Content -LiteralPath $address -Raw | ConvertFrom-Json).url
    $script:ui.Origin = $uri.GetLeftPart([UriPartial]::Authority); $script:ui.Query = $uri.Query
    # A headless Deck exits 3 s after its last event-stream client; mock pages open none, so hold one.
    $script:ui.Http = [Net.Http.HttpClient]::new(); $script:ui.Http.Timeout = [Threading.Timeout]::InfiniteTimeSpan
    $script:ui.Stream = $script:ui.Http.GetStreamAsync("$($script:ui.Origin)/api/events$($script:ui.Query)").GetAwaiter().GetResult()
    $edge = @("${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe", "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe") | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
    if (-not $edge) { throw 'msedge.exe not found' }
    $profileDir = Join-Path $script:uiRoot 'tmp-edge'
    $script:ui.Edge = Start-UiProcess $edge @('--headless=new', "--remote-debugging-port=$Port", "--user-data-dir=$profileDir", '--no-first-run',
        '--disable-extensions', '--disable-background-networking', '--disable-sync', '--window-size=1440,900' + $EdgeArgs + 'about:blank')
    $deadline = [DateTime]::UtcNow.AddSeconds(15); $page = $null
    while (-not $page) {
        try { $page = (Invoke-RestMethod "http://127.0.0.1:$Port/json/list") | Where-Object type -eq 'page' | Select-Object -First 1 } catch { $page = $null }
        if (-not $page) { if ([DateTime]::UtcNow -gt $deadline) { throw 'Edge DevTools did not answer' }; Start-Sleep -Milliseconds 200 }
    }
    $script:ui.Ws = [Net.WebSockets.ClientWebSocket]::new()
    $script:ui.Ws.ConnectAsync([Uri]$page.webSocketDebuggerUrl, [Threading.CancellationToken]::None).GetAwaiter().GetResult()
    $script:ui.Id = 0
    Send-Cdp 'Page.enable' | Out-Null; Send-Cdp 'Runtime.enable' | Out-Null
    "Deck $($script:ui.Origin) (pid $($script:ui.Deck.Id)); Edge pid $($script:ui.Edge.Id)"
}
function Send-Cdp([string]$Method, $Params = @{}, [int]$TimeoutMs = 20000) {
    $id = ++$script:ui.Id
    $msg = [Text.Encoding]::UTF8.GetBytes((@{ id = $id; method = $Method; params = $Params } | ConvertTo-Json -Depth 10 -Compress))
    $script:ui.Ws.SendAsync([ArraySegment[byte]]::new($msg), 'Text', $true, [Threading.CancellationToken]::None).GetAwaiter().GetResult()
    $deadline = [DateTime]::UtcNow.AddMilliseconds($TimeoutMs)
    $buf = [byte[]]::new(1 -shl 20)
    while ([DateTime]::UtcNow -lt $deadline) {
        $ms = [IO.MemoryStream]::new()
        do {
            $cts = [Threading.CancellationTokenSource]::new([Math]::Max(1, ($deadline - [DateTime]::UtcNow).TotalMilliseconds))
            $r = $script:ui.Ws.ReceiveAsync([ArraySegment[byte]]::new($buf), $cts.Token).GetAwaiter().GetResult()
            $ms.Write($buf, 0, $r.Count)
        } while (-not $r.EndOfMessage)
        $obj = [Text.Encoding]::UTF8.GetString($ms.ToArray()) | ConvertFrom-Json -Depth 50
        if ($obj.id -eq $id) {
            if ($obj.error) { throw "CDP $Method failed: $($obj.error.message)" }
            return $obj.result
        }
    }
    throw "CDP $Method timed out"
}
function Invoke-DeckJs([string]$Expression) {
    $r = Send-Cdp 'Runtime.evaluate' @{ expression = $Expression; returnByValue = $true; awaitPromise = $true }
    if ($r.exceptionDetails) { throw "JS error: $($r.exceptionDetails.exception.description ?? $r.exceptionDetails.text)" }
    $r.result.value
}
function Set-DeckSize([int]$Width, [int]$Height = 900) {
    Send-Cdp 'Emulation.setDeviceMetricsOverride' @{ width = $Width; height = $Height; deviceScaleFactor = 1; mobile = $false } | Out-Null
}
function Open-DeckPage([string]$Extra = '', [int]$WaitMs = 1500) {
    $url = $script:ui.Origin + '/' + $script:ui.Query + ($Extra -replace '^\?', '&')
    Send-Cdp 'Page.navigate' @{ url = $url } | Out-Null
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    Start-Sleep -Milliseconds 100
    while (-not (Invoke-DeckJs "document.readyState === 'complete' && !!document.querySelector('#grid') && location.href.includes('$($Extra -replace '^\?', '')')")) {
        if ([DateTime]::UtcNow -gt $deadline) { throw "page did not load: $url" }
        Start-Sleep -Milliseconds 100
    }
    Start-Sleep -Milliseconds $WaitMs
}
function Save-DeckShot([string]$Name) {
    $dir = Join-Path $script:uiRoot 'shots'; New-Item -ItemType Directory -Force -Path $dir | Out-Null
    $r = Send-Cdp 'Page.captureScreenshot' @{ format = 'png' }
    $path = Join-Path $dir "$Name.png"
    [IO.File]::WriteAllBytes($path, [Convert]::FromBase64String($r.data))
    $path
}
function Stop-DeckUi {
    if ($script:ui.Ws) { try { $script:ui.Ws.Dispose() } catch { Write-Warning "websocket dispose: $_" } }
    if ($script:ui.Stream) { $script:ui.Stream.Dispose() }
    if ($script:ui.Http) { $script:ui.Http.Dispose() }
    foreach ($k in 'Edge', 'Deck', 'Herdr') {
        $p = $script:ui[$k]
        if (-not $p) { continue }
        if ($k -eq 'Herdr' -and -not $p.HasExited) { try { Invoke-Lab @('server', 'stop') | Out-Null } catch { Write-Warning "lab stop: $_" } }
        if (-not $p.WaitForExit(3000)) { $p.Kill($true) }
        $p.Dispose()
    }
    $env:APPDATA = $script:ui.SavedAppData; $env:HERDR_SOCKET_PATH = $script:ui.SavedSocket
    $env:HERDR_DECK_ROOT = $script:ui.SavedRoot
    foreach ($leaf in 'tmp-edge', 'tmp-ui-address.json') {
        $target = [IO.Path]::GetFullPath((Join-Path $script:uiRoot $leaf))
        $parent = [IO.Path]::GetFullPath($script:uiRoot).TrimEnd('\') + '\'
        if (-not $target.StartsWith($parent, [StringComparison]::OrdinalIgnoreCase)) { throw "Cleanup target outside test directory: $target" }
        if (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Recurse -Force }
    }
    $script:ui = @{}
}
