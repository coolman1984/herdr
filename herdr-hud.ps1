# Pane HUD: keeps every pane's top border showing
#     <name> · <project path> · <model> · ctx <used>% (<tokens>/<window>) · out <tokens>
# and gives the sidebar the tokens $name, $model, $ctx (config/settings.toml rows).
# Runs in the background (started by Herdr.cmd, one copy only); -Once = one pass, printed.
#
# Where the numbers come from (read-only, usage fields only - never message text):
#   Codex  ~\.codex\sessions\**\rollout-*-<session>.jsonl : turn_context (model, effort),
#          token_count (last input = context used, model_context_window, total output)
#   Claude ~\.claude\projects\*\<session>.jsonl : assistant usage (input + cache = context,
#          output summed per message id), model; window from agents.json "contextWindow"
#   Others the model herdr-agents started them with, else the profile's default.
#   Claude's account and its 5-hour limit ("5h N%", token $limit): claude-limits.ps1; at 95%
#          one Herdr notification per account and window (HISTORY Phase 24).
#   Claude's $ cost: %APPDATA%\herdr\claude-cost\<session>.json, written by the owner's
#          status line script ~\.claude\statusline.js (only Claude Code knows its cost).
# <name> = the pane's own name (ctrl+b shift+p), else its space's name.
[CmdletBinding()]
param([int]$IntervalSec = 5, [switch]$Once)

$ErrorActionPreference = "Stop"
$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"
$registry = (Get-Content -LiteralPath (Join-Path $PSScriptRoot "agents.json") -Raw | ConvertFrom-Json).agents
$panesState = Join-Path $env:APPDATA "herdr\herdr-agents-panes.json"
. (Join-Path $PSScriptRoot "claude-limits.ps1")
$source = "herdr-hud"
$ttlMs = 120000

function Invoke-Herdr([string[]]$Arguments) {
    $out = & $herdr @Arguments 2>&1
    $text = ($out | ForEach-Object { $_.ToString() }) -join "`n"
    $json = $null
    try { $json = $text | ConvertFrom-Json } catch { }
    [PSCustomObject]@{ Ok = ($LASTEXITCODE -eq 0); Text = $text; Json = $json }
}

function Format-Tokens([double]$n) {
    if ($n -ge 1000000) { return [string]::Format([cultureinfo]::InvariantCulture, "{0:0.#}M", $n / 1000000) }
    if ($n -ge 1000) { return [string]::Format([cultureinfo]::InvariantCulture, "{0:0.#}k", $n / 1000) }
    [string][int]$n
}

# Long paths keep the drive and the last two folders: D:\...\GitHub\MR.Park
function Format-Path([string]$Path) {
    $p = $Path.TrimEnd('\')
    if ($p.Length -le 40) { return $p }
    $parts = $p -split '\\'
    if ($parts.Count -le 3) { return $p }
    "$($parts[0])\...\$($parts[-2])\$($parts[-1])"
}

# Session logs are read incrementally: only lines added since the last pass.
$sessions = @{}
function Read-NewLines([string]$File) {
    $s = $sessions[$File]
    if (-not $s) { $s = @{ Pos = 0L; Data = @{} }; $sessions[$File] = $s }
    $lines = New-Object System.Collections.Generic.List[string]
    $fs = [System.IO.File]::Open($File, 'Open', 'Read', 'ReadWrite')
    try {
        if ($fs.Length -lt $s.Pos) { $s.Pos = 0L; $s.Data = @{} }
        $fs.Seek($s.Pos, 'Begin') | Out-Null
        $reader = New-Object System.IO.StreamReader($fs)
        $text = $reader.ReadToEnd()
        $end = $text.LastIndexOf("`n")
        if ($end -ge 0) {
            foreach ($l in $text.Substring(0, $end).Split("`n")) { if ($l.Trim()) { $lines.Add($l) } }
            $s.Pos += [System.Text.Encoding]::UTF8.GetByteCount($text.Substring(0, $end + 1))
        }
    } finally { $fs.Dispose() }
    # Two outputs on purpose: the caller splits them with "$s, $lines = Read-NewLines ...".
    $s
    , $lines
}

$fileCache = @{}
function Find-SessionFile([string]$Kind, [string]$Id) {
    $key = "$Kind/$Id"
    if ($fileCache.ContainsKey($key) -and $fileCache[$key] -and (Test-Path -LiteralPath $fileCache[$key])) { return $fileCache[$key] }
    $file = $null
    # Separate accounts (herdr-agents -Account) keep their logs in their own folders.
    $accounts = Join-Path $env:USERPROFILE ".herdr-accounts"
    if ($Kind -eq "codex") {
        $roots = @(Join-Path $env:USERPROFILE ".codex\sessions") + @(Get-ChildItem -LiteralPath (Join-Path $accounts "codex") -Directory -ErrorAction SilentlyContinue | ForEach-Object { Join-Path $_.FullName "sessions" })
        foreach ($root in $roots | Where-Object { Test-Path -LiteralPath $_ }) {
            $file = Get-ChildItem -LiteralPath $root -Recurse -Filter "*$Id.jsonl" -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
            if ($file) { break }
        }
    } elseif ($Kind -eq "claude") {
        $roots = @(Join-Path $env:USERPROFILE ".claude\projects") + @(Get-ChildItem -LiteralPath (Join-Path $accounts "claude") -Directory -ErrorAction SilentlyContinue | ForEach-Object { Join-Path $_.FullName "projects" })
        foreach ($root in $roots | Where-Object { Test-Path -LiteralPath $_ }) {
            $file = Get-ChildItem -LiteralPath $root -Directory | ForEach-Object { Join-Path $_.FullName "$Id.jsonl" } |
                Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
            if ($file) { break }
        }
    }
    $fileCache[$key] = $file
    $file
}

function Get-CodexUsage([string]$File) {
    $s, $lines = Read-NewLines $File
    foreach ($l in $lines) {
        if ($l -notmatch '"(turn_context|token_count)"') { continue }
        try { $o = $l | ConvertFrom-Json } catch { continue }
        if ($o.type -eq "turn_context") {
            $s.Data.model = $o.payload.model; $s.Data.effort = $o.payload.effort
        } elseif ($o.payload.type -eq "token_count" -and $o.payload.info) {
            $s.Data.ctx = [double]$o.payload.info.last_token_usage.input_tokens
            $s.Data.window = [double]$o.payload.info.model_context_window
            $s.Data.out = [double]$o.payload.info.total_token_usage.output_tokens
        }
    }
    $s.Data
}

function Get-ClaudeUsage([string]$File, [double]$Window) {
    $s, $lines = Read-NewLines $File
    if (-not $s.Data.seen) { $s.Data.seen = @{}; $s.Data.out = 0.0 }
    foreach ($l in $lines) {
        if ($l -notmatch '"usage"') { continue }
        try { $o = $l | ConvertFrom-Json } catch { continue }
        $m = $o.message
        if (-not $m -or -not $m.usage) { continue }
        $u = $m.usage
        $s.Data.ctx = [double]$u.input_tokens + [double]$u.cache_creation_input_tokens + [double]$u.cache_read_input_tokens
        if ($m.model -and $m.model -notlike "<*") { $s.Data.model = $m.model }
        # One reply is written as several lines with the same message id: count it once.
        if ($m.id -and -not $s.Data.seen.ContainsKey($m.id)) { $s.Data.seen[$m.id] = $true; $s.Data.out += [double]$u.output_tokens }
    }
    $s.Data.window = $Window
    $s.Data
}

function Get-PaneAccounts {
    $map = @{}
    if (Test-Path -LiteralPath $panesState) {
        try { foreach ($e in @(Get-Content -LiteralPath $panesState -Raw | ConvertFrom-Json)) { if ($e.PSObject.Properties["account"] -and $e.account) { $map[$e.pane] = $e.account } } } catch { }
    }
    $map
}

function Get-StartedModels {
    $map = @{}
    if (Test-Path -LiteralPath $panesState) {
        try { foreach ($e in @(Get-Content -LiteralPath $panesState -Raw | ConvertFrom-Json)) { if ($e.model) { $map[$e.pane] = $e.model } } } catch { }
    }
    $map
}

# One Herdr notification when an account reaches the limit: once per account and reset
# window (a HUD restart may repeat it once). -Once never notifies.
$noticed = @{}
function Send-LimitNotices($Limits) {
    if ($Once) { return }
    foreach ($l in $Limits | Where-Object { $_.HasData -and $_.FiveHour -ge $script:ClaudeLimitThreshold }) {
        $key = "$($l.Account)|$($l.FiveResets)"
        if ($noticed[$key]) { continue }
        $noticed[$key] = $true
        $next = Get-ClaudeNextAccount $Limits
        $body = if ($next) { "Least used: $($next.Account) ($($next.FiveHour)%). Resets $($l.FiveResets.ToString('HH:mm'))." } else { "No other account below the limit. Resets $($l.FiveResets.ToString('HH:mm'))." }
        [void](Invoke-Herdr @("notification", "show", "$($l.Account) at $($l.FiveHour)% - switch: Ctrl+B Shift+C", "--body", $body, "--sound", "request"))
    }
}

$last = @{}
$lastSent = @{}
function Update-Panes {
    $ws = Invoke-Herdr @("workspace", "list")
    if (-not $ws.Ok) { return $false }
    $labels = @{}
    foreach ($w in $ws.Json.result.workspaces) { $labels[$w.workspace_id] = $w.label }
    $agents = @{}
    $al = Invoke-Herdr @("agent", "list")
    if ($al.Ok) { foreach ($a in $al.Json.result.agents) { $agents[$a.pane_id] = $a } }
    $pl = Invoke-Herdr @("pane", "list")
    if (-not $pl.Ok) { return $false }
    $panes = @($pl.Json.result.panes)
    # Pane names (ctrl+b shift+p): "label" in pane list, absent when the pane has none.
    $names = @{}
    foreach ($p in $panes) { if ($p.PSObject.Properties["label"] -and $p.label) { $names[$p.pane_id] = $p.label } }
    $started = Get-StartedModels
    $paneAccounts = Get-PaneAccounts
    $claudeAccounts = @(Get-ClaudeAccountNames)
    $claudeLimits = @(Get-ClaudeLimits)
    Send-LimitNotices $claudeLimits
    $now = Get-Date
    foreach ($p in $panes) {
        $space = $labels[$p.workspace_id]
        $name = if ($names[$p.pane_id]) { $names[$p.pane_id] } else { $space }
        $parts = @($name, (Format-Path ([string]$p.cwd)))
        $tokens = [ordered]@{ path = (Format-Path ([string]$p.cwd)) }
        if ($names[$p.pane_id] -and $names[$p.pane_id] -ne $space) { $tokens.name = $names[$p.pane_id] }
        $a = $agents[$p.pane_id]
        if ($a) {
            $kind = [string]$a.agent
            $usage = $null
            $sid = if ($a.agent_session -and $a.agent_session.kind -eq "id") { [string]$a.agent_session.value } else { $null }
            if ($sid -and $kind -in @("codex", "claude")) {
                $file = Find-SessionFile $kind $sid
                if ($file) {
                    try {
                        $usage = if ($kind -eq "codex") { Get-CodexUsage $file }
                                 else { Get-ClaudeUsage $file ([double]($registry.claude.PSObject.Properties["contextWindow"] ? $registry.claude.contextWindow : 0)) }
                    } catch { $usage = $null }
                }
            }
            $model = if ($usage -and $usage.model) { $usage.model } elseif ($started[$p.pane_id]) { $started[$p.pane_id] }
                     elseif ($registry.PSObject.Properties[$kind] -and $registry.$kind.model) { $registry.$kind.model } else { $kind }
            if ($usage -and $usage.effort) { $model = "$model $($usage.effort)" }
            $parts += $model
            $tokens.model = $model
            # Claude's dollar cost and its account's 5-hour limit: written by
            # ~\.claude\statusline.js for its session (config_dir names the account).
            if ($kind -eq "claude" -and $sid) {
                $costFile = Join-Path $env:APPDATA "herdr\claude-cost\$sid.json"
                if (Test-Path -LiteralPath $costFile) {
                    try {
                        $c = Get-Content -LiteralPath $costFile -Raw | ConvertFrom-Json
                        $cost = [string]::Format([cultureinfo]::InvariantCulture, '${0:0.00}', [double]$c.cost)
                        $parts += $cost
                        $tokens.cost = $cost
                        $acc = Get-ClaudeAccountOf ([string]$c.config_dir) $claudeAccounts
                        if ($acc -and -not $paneAccounts[$p.pane_id]) { $paneAccounts[$p.pane_id] = $acc }
                        $lim = $claudeLimits | Where-Object { $_.Account -eq $acc } | Select-Object -First 1
                        if ($lim -and $null -ne $lim.FiveHour) { $parts += "5h $($lim.FiveHour)%"; $tokens.limit = "5h $($lim.FiveHour)%" }
                    } catch { }
                }
            }
            if ($usage -and $usage.ctx) {
                $ctx = if ($usage.window) { "ctx {0}% ({1}/{2})" -f [int](100 * $usage.ctx / $usage.window), (Format-Tokens $usage.ctx), (Format-Tokens $usage.window) }
                       else { "ctx $(Format-Tokens $usage.ctx)" }
                $parts += $ctx
                $tokens.ctx = $ctx
                if ($usage.out) { $parts += "out $(Format-Tokens $usage.out)"; $tokens.out = Format-Tokens $usage.out }
            }
        }
        # Which separate account the agent runs on (herdr-agents -Account).
        if ($paneAccounts[$p.pane_id]) { $parts += "account $($paneAccounts[$p.pane_id])"; $tokens.account = $paneAccounts[$p.pane_id] }
        $title = $parts -join " · "
        $sig = "$title|$(($tokens.GetEnumerator() | ForEach-Object { "$($_.Key)=$($_.Value)" }) -join '|')"
        if ($Once) { Write-Output ("{0,-8} {1}" -f $p.pane_id, $title) }
        # Send on change, and again before the TTL runs out (stale data disappears if the HUD stops).
        if ($last[$p.pane_id] -eq $sig -and $lastSent[$p.pane_id] -and ($now - $lastSent[$p.pane_id]).TotalMilliseconds -lt ($ttlMs / 2)) { continue }
        $args_ = @("pane", "report-metadata", $p.pane_id, "--source", $source, "--title", $title, "--ttl-ms", "$ttlMs")
        if (-not $tokens.Contains("name")) { $args_ += @("--clear-token", "name") }
        foreach ($t in $tokens.GetEnumerator()) { $args_ += @("--token", "$($t.Key)=$($t.Value)") }
        if ((Invoke-Herdr $args_).Ok) { $last[$p.pane_id] = $sig; $lastSent[$p.pane_id] = $now }
    }
    $true
}

if (-not (Test-Path -LiteralPath $herdr)) { Write-Output "Herdr is not installed at $herdr."; exit 1 }
if ($Once) { Update-Panes | Where-Object { $_ -isnot [bool] }; exit 0 }

# One HUD per Windows session (started at sign-in and by Herdr.cmd). While Herdr is not
# running it only checks every 30 s.
$mutex = New-Object System.Threading.Mutex($false, "Local\herdr-hud")
if (-not $mutex.WaitOne(0)) { exit 0 }
while ($true) {
    $ok = $false
    try { $ok = Update-Panes } catch { $ok = $false }
    Start-Sleep -Seconds $(if ($ok) { $IntervalSec } else { 30 })
}
