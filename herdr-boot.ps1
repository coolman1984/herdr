# herdr-boot.ps1 - at Windows sign-in: Herdr server (headless), a lead for the home project, the lead watcher.
#
# Started by herdr-keepalive.ps1 (task "Herdr Keepalive", at sign-in and whenever the server is gone;
# conhost.exe --headless, no window; CLAUDE.md). Phase 64's Startup shortcut "Herdr Boot.lnk" is removed
# by `herdr-keepalive -Install`.
# - Server not running: start it headless, open a workspace "Lead" in -Cwd and run
#   `herdr-agents start next -Lead` there (first free leader of agents.json leaders.order; it also starts
#   the watcher). Clients (Windows Terminal, `herdr --remote <pc>` over SSH) attach to this server later.
# - Server already running (the owner opened Herdr first): start nothing but the watcher; its mutex
#   keeps one copy.
# Log: %APPDATA%\herdr\boot.log. -DryRun prints the steps that would start agents or the watcher.
# -Session <name> uses an isolated Herdr session (tests: `-Session lab -DryRun`).
param(
    [string]$Cwd = $PSScriptRoot,
    [string]$Session = '',
    [int]$WaitSec = 30,
    [switch]$DryRun
)
$ErrorActionPreference = 'Stop'
$herdr = Join-Path $env:LOCALAPPDATA 'Programs\Herdr\bin\herdr.exe'
$agents = Join-Path $PSScriptRoot 'herdr-agents.ps1'
$logPath = Join-Path $env:APPDATA 'herdr\boot.log'

function Write-BootLog([string]$Text) {
    $line = "{0:yyyy-MM-dd HH:mm:ss} {1}" -f (Get-Date), $Text
    Write-Output $line
    $null = New-Item -ItemType Directory -Path (Split-Path -Parent $logPath) -Force
    Add-Content -LiteralPath $logPath -Value $line -Encoding utf8
    if ((Get-Item -LiteralPath $logPath).Length -ge 100KB) {
        $rows = @([IO.File]::ReadAllLines($logPath))
        [IO.File]::WriteAllLines($logPath, $rows[[int]($rows.Count / 2)..($rows.Count - 1)])
    }
}

function Invoke-Herdr([string[]]$Arguments) {
    $all = if ($Session) { @('--session', $Session) + $Arguments } else { $Arguments }
    $output = & $herdr @all 2>&1
    $code = $LASTEXITCODE
    $text = ($output | ForEach-Object { $_.ToString() }) -join "`n"
    $json = $null
    try { $json = $text | ConvertFrom-Json } catch { }
    [PSCustomObject]@{ Ok = ($code -eq 0); Text = $text; Json = $json }
}

function Fail([string]$Text) { Write-BootLog "FAILED: $Text"; exit 1 }

if (-not (Test-Path -LiteralPath $herdr)) { Fail "Herdr is not installed at $herdr" }
if (-not (Test-Path -LiteralPath $Cwd -PathType Container)) { Fail "Folder not found: $Cwd" }
$Cwd = (Resolve-Path -LiteralPath $Cwd).ProviderPath
if ($Session) { $env:HERDR_SOCKET_PATH = Join-Path $env:APPDATA "herdr\sessions\$Session\herdr.sock" }

if ((Invoke-Herdr @('workspace', 'list')).Ok) {
    Write-BootLog "server already running$(if ($Session) { " (session $Session)" }); starting only the watcher"
    if ($DryRun) { Write-BootLog "dry run: would start the watcher"; exit 0 }
    Start-Process -FilePath 'conhost.exe' -ArgumentList @('--headless', 'pwsh', '-NoProfile', '-File', "`"$agents`"", 'autoswitch') -WindowStyle Hidden
    exit 0
}

$serverArgs = @('--headless', "`"$herdr`"") + $(if ($Session) { @('--session', $Session) }) + @('server')
Start-Process -FilePath 'conhost.exe' -ArgumentList $serverArgs -WindowStyle Hidden
$deadline = (Get-Date).AddSeconds($WaitSec)
do { Start-Sleep -Seconds 1; $up = (Invoke-Herdr @('workspace', 'list')).Ok } while (-not $up -and (Get-Date) -lt $deadline)
if (-not $up) { Fail "Herdr server did not answer within $WaitSec s" }
Write-BootLog "server started$(if ($Session) { " (session $Session)" })"

$ws = Invoke-Herdr @('workspace', 'create', '--cwd', $Cwd, '--label', 'Lead', '--no-focus')
if (-not $ws.Ok) { Fail "Could not open the Lead workspace: $($ws.Text)" }
$pane = $ws.Json.result.root_pane.pane_id
if (-not $pane) { Fail "Herdr gave no pane id for the Lead workspace: $($ws.Text)" }
Write-BootLog "Lead workspace pane $pane in $Cwd"

if ($DryRun) { Write-BootLog "dry run: would run herdr-agents start next -Lead -Pane $pane -Cwd $Cwd"; exit 0 }
$result = & pwsh -NoProfile -File $agents start next -Lead -Pane $pane -Cwd $Cwd 2>&1
$code = $LASTEXITCODE
Write-BootLog ("lead start exit {0}: {1}" -f $code, (($result | ForEach-Object { $_.ToString() }) -join ' | '))
exit $code
