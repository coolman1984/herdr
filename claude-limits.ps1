# Claude usage limits per account, for herdr-agents (usage, start next) and the HUD.
# Dot-source it:  . "$PSScriptRoot\claude-limits.ps1"; Get-ClaudeLimits
#
# Source (read-only): %APPDATA%\herdr\claude-cost\<session_id>.json, written by the owner's
# status line script ~\.claude\statusline.js (every account copies it) on each refresh:
#   rate_limits.five_hour / seven_day = { used_percentage, resets_at (unix seconds) },
#   config_dir ("" = the main sign-in, else %USERPROFILE%\.herdr-accounts\claude\<name>), cost.
# Limits belong to the ACCOUNT, so the newest file of an account wins. A window whose reset
# time has passed counts as 0% (the account has not been used since it reset).

$script:ClaudeLimitThreshold = 95
$script:claudeLimitCache = @{}

function Get-ClaudeAccountNames {
    # Main sign-in first (display name from agents.json "accountNames.claude", default "main").
    $main = "main"
    try {
        $reg = Get-Content -LiteralPath (Join-Path $PSScriptRoot "agents.json") -Raw | ConvertFrom-Json
        if ($reg.PSObject.Properties["accountNames"] -and $reg.accountNames.claude) { $main = [string]$reg.accountNames.claude }
    } catch { }
    $list = @([PSCustomObject]@{ Name = $main; Dir = ""; Main = $true })
    $root = Join-Path $env:USERPROFILE ".herdr-accounts\claude"
    foreach ($d in Get-ChildItem -LiteralPath $root -Directory -ErrorAction SilentlyContinue | Sort-Object Name) {
        $list += [PSCustomObject]@{ Name = $d.Name; Dir = $d.FullName; Main = $false }
    }
    $list
}

# Which account a config_dir belongs to ("" or missing = the main sign-in).
function Get-ClaudeAccountOf([string]$ConfigDir, $Accounts) {
    $dir = ([string]$ConfigDir).TrimEnd('\')
    foreach ($a in $Accounts) {
        if (($a.Main -and -not $dir) -or ($a.Dir -and [string]::Equals($a.Dir.TrimEnd('\'), $dir, [StringComparison]::OrdinalIgnoreCase))) { return $a.Name }
    }
    if ($dir) { return (Split-Path -Leaf $dir) }
    $null
}

function Get-ClaudeLimits {
    $accounts = @(Get-ClaudeAccountNames)
    $now = [DateTimeOffset]::UtcNow
    $costDir = Join-Path $env:APPDATA "herdr\claude-cost"
    $newest = @{}; $costToday = @{}
    foreach ($f in Get-ChildItem -LiteralPath $costDir -Filter *.json -File -ErrorAction SilentlyContinue) {
        $c = $script:claudeLimitCache[$f.FullName]
        if (-not $c -or $c.Time -ne $f.LastWriteTimeUtc) {
            try { $c = @{ Time = $f.LastWriteTimeUtc; Data = (Get-Content -LiteralPath $f.FullName -Raw | ConvertFrom-Json) } } catch { continue }
            $script:claudeLimitCache[$f.FullName] = $c
        }
        $d = $c.Data
        $acc = Get-ClaudeAccountOf ([string]$d.config_dir) $accounts
        if (-not $acc) { continue }
        if ($f.LastWriteTime.Date -eq (Get-Date).Date) { $costToday[$acc] += [double]$d.cost }
        if ($d.rate_limits -and (-not $newest[$acc] -or $f.LastWriteTimeUtc -gt $newest[$acc].Time)) {
            $newest[$acc] = @{ Time = $f.LastWriteTimeUtc; Data = $d }
        }
    }
    foreach ($a in $accounts) {
        $n = $newest[$a.Name]
        $row = [ordered]@{
            Account = $a.Name; Main = $a.Main; HasData = [bool]$n; FiveHour = $null; FiveResets = $null
            SevenDay = $null; SevenResets = $null; CostToday = [double]$costToday[$a.Name]; AgeMin = $null; Limit = $false
        }
        if ($n) {
            foreach ($w in @(@("five_hour", "FiveHour", "FiveResets"), @("seven_day", "SevenDay", "SevenResets"))) {
                $r = $n.Data.rate_limits.($w[0])
                if (-not $r) { continue }
                $reset = if ($r.resets_at) { [DateTimeOffset]::FromUnixTimeSeconds([long]$r.resets_at) } else { $null }
                $row[$w[1]] = if ($reset -and $reset -le $now) { 0 } else { [int]$r.used_percentage }
                $row[$w[2]] = if ($reset) { $reset.LocalDateTime } else { $null }
            }
            $row.AgeMin = [int]((Get-Date).ToUniversalTime() - $n.Time).TotalMinutes
            $row.Limit = ($row.FiveHour -ge $script:ClaudeLimitThreshold) -or ($row.SevenDay -ge $script:ClaudeLimitThreshold)
        }
        [PSCustomObject]$row
    }
}

# Text table (herdr-agents usage, handoff files): one line per account, then NEXT.
function Format-ClaudeLimits($Limits) {
    $t = "ddd HH:mm"
    "{0,-8} {1,-6} {2,-12} {3,-7} {4,-12} {5,-9} {6}" -f "account", "5h", "5h resets", "weekly", "week resets", "cost today", "data age"
    foreach ($l in $Limits) {
        $name = if ($l.Main) { "$($l.Account)*" } else { $l.Account }
        if (-not $l.HasData) { "{0,-8} no data (open it once)" -f $name; continue }
        $five = if ($null -ne $l.FiveHour) { "$($l.FiveHour)%" } else { "-" }
        $week = if ($null -ne $l.SevenDay) { "$($l.SevenDay)%" } else { "-" }
        $line = "{0,-8} {1,-6} {2,-12} {3,-7} {4,-12} {5,-9} {6}" -f $name, $five,
            $(if ($l.FiveResets) { $l.FiveResets.ToString($t) } else { "-" }), $week,
            $(if ($l.SevenResets) { $l.SevenResets.ToString($t) } else { "-" }),
            ([string]::Format([cultureinfo]::InvariantCulture, '${0:0.00}', $l.CostToday)), "$($l.AgeMin) min"
        if ($l.Limit) { $line += "  LIMIT" }
        $line
    }
    "(* = the main sign-in; LIMIT = at or above $($script:ClaudeLimitThreshold)% of the 5-hour or weekly limit)"
    $next = Get-ClaudeNextAccount $Limits
    if ($next) { "NEXT: $($next.Account)" } else { "NEXT: none - every account with data is at its limit or has no data" }
}

# The account to continue on: below the threshold, lowest 5-hour use (then lowest weekly).
# Accounts without data are not chosen (their use is unknown) - open one once to give it data.
function Get-ClaudeNextAccount($Limits) {
    @($Limits | Where-Object { $_.HasData -and -not $_.Limit } | Sort-Object FiveHour, SevenDay) | Select-Object -First 1
}
