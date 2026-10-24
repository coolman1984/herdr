# herdr-autosync.ps1 - keeps GitHub `main` in sync with this folder, automatically.
# Owner decision 2026-10-24: "sync with main all the time automatically".
#   herdr-autosync.ps1                 loop: every -IntervalSeconds, commit + pull --rebase + push
#   herdr-autosync.ps1 -Once           one pass, then exit (prints what it did)
#   herdr-autosync.ps1 -Once -DryRun   print what would be committed, change nothing
#   herdr-autosync.ps1 -Status         last log lines + whether a loop is running
# Safety: never force-push, never reset/stash/checkout; stops (logs CONFLICT) if the rebase
# conflicts; skips files that look like secrets or are > 5 MB; one instance (mutex).
param(
    [int]$IntervalSeconds = 300,
    [switch]$Once,
    [switch]$DryRun,
    [switch]$Status
)
$ErrorActionPreference = "Stop"
$repo = $PSScriptRoot
$logDir = Join-Path $env:APPDATA "herdr"
$log = Join-Path $logDir "autosync.log"
$maxBytes = 5MB
$secretNames = '(^|/)(\.env[^/]*|.*\.pem|.*\.pfx|.*\.key|id_rsa.*|\.credentials\.json|auth\.json|.*secrets?\.(json|toml|ya?ml))$'
$secretText = '(sk-[A-Za-z0-9_-]{20,}|ghp_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY-----|xox[abp]-[A-Za-z0-9-]{10,})'
$coAuthor = "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"

function Write-Log([string]$msg) {
    New-Item -ItemType Directory -Force -Path $logDir | Out-Null
    if ((Test-Path -LiteralPath $log) -and (Get-Item -LiteralPath $log).Length -gt 1MB) {
        Move-Item -LiteralPath $log -Destination "$log.old" -Force
    }
    $line = "{0:yyyy-MM-dd HH:mm:ss} {1}" -f (Get-Date), $msg
    Add-Content -LiteralPath $log -Value $line
    Write-Host $line
}
function Invoke-Git { & git -C $repo @args 2>&1 }

if ($Status) {
    if (Test-Path -LiteralPath $log) { Get-Content -LiteralPath $log -Tail 12 } else { "no log yet" }
    $mutex = $null
    $isNew = $false
    $mutex = New-Object System.Threading.Mutex($false, "Local\herdr-autosync", [ref]$isNew)
    if ($mutex.WaitOne(0)) { $mutex.ReleaseMutex(); "loop: NOT running" } else { "loop: running" }
    return
}

function Invoke-Pass {
    $branch = (Invoke-Git rev-parse --abbrev-ref HEAD | Out-String).Trim()
    if ($branch -ne "main") { Write-Log "SKIP branch is '$branch', not main"; return }
    # a merge/rebase already in progress means a human or agent is working on git: do not touch it
    foreach ($m in "rebase-merge", "rebase-apply", "MERGE_HEAD") {
        if (Test-Path -LiteralPath (Join-Path $repo ".git\$m")) { Write-Log "SKIP git operation in progress ($m)"; return }
    }
    $lines = @(Invoke-Git status --porcelain=v1 -uall | Where-Object { $_ })
    $stage = New-Object System.Collections.Generic.List[string]
    foreach ($l in $lines) {
        $path = $l.Substring(3).Trim('"')
        if ($path -match ' -> ') { $path = ($path -split ' -> ')[-1] }
        $full = Join-Path $repo $path
        if ($path -match $secretNames) { Write-Log "SKIP secret-like name: $path"; continue }
        if ((Test-Path -LiteralPath $full -PathType Leaf)) {
            $item = Get-Item -LiteralPath $full
            if ($item.Length -gt $maxBytes) { Write-Log "SKIP > 5 MB: $path"; continue }
            if ($item.Length -lt 2MB) {
                $hit = Select-String -LiteralPath $full -Pattern $secretText -List -ErrorAction SilentlyContinue
                if ($hit) { Write-Log "SKIP secret-like text in: $path"; continue }
            }
        }
        $stage.Add($path)
    }
    if ($stage.Count -eq 0) { Write-Log "no changes"; if (-not $DryRun) { Push-IfAhead }; return }
    if ($DryRun) { Write-Log "DRYRUN would commit $($stage.Count) file(s)"; $stage | Select-Object -First 40 | ForEach-Object { Write-Host "  $_" }; return }
    foreach ($p in $stage) {
        $out = Invoke-Git add -A -- $p
        if ($LASTEXITCODE -ne 0) { Write-Log "ERROR git add $p : $out"; return }
    }
    $msg = "autosync: $($stage.Count) file(s) $(Get-Date -Format 'yyyy-MM-dd HH:mm')`n`n$coAuthor"
    $out = Invoke-Git commit -q -m $msg
    if ($LASTEXITCODE -ne 0) { Write-Log "NOTE commit: $out"; return }
    Write-Log "COMMIT $($stage.Count) file(s)"
    Push-IfAhead
}

function Push-IfAhead {
    $null = Invoke-Git fetch -q origin main
    if ($LASTEXITCODE -ne 0) { Write-Log "ERROR fetch failed (offline?)"; return }
    $counts = (Invoke-Git rev-list --left-right --count "HEAD...origin/main" | Out-String).Trim() -split '\s+'
    $ahead = [int]$counts[0]; $behind = [int]$counts[1]
    if ($behind -gt 0) {
        $out = Invoke-Git pull --rebase --autostash -q origin main
        if ($LASTEXITCODE -ne 0) {
            Write-Log "CONFLICT pull --rebase failed, aborting rebase: $out"
            $null = Invoke-Git rebase --abort
            return
        }
        Write-Log "PULLED $behind commit(s) from origin/main"
        $ahead = [int]((Invoke-Git rev-list --count "origin/main..HEAD" | Out-String).Trim())
    }
    if ($ahead -gt 0) {
        $out = Invoke-Git push -q origin main
        if ($LASTEXITCODE -ne 0) { Write-Log "ERROR push failed: $out"; return }
        Write-Log "PUSHED $ahead commit(s) to origin/main"
    }
}

if ($Once) { Invoke-Pass; return }

$isNew = $false
$mutex = New-Object System.Threading.Mutex($true, "Local\herdr-autosync", [ref]$isNew)
if (-not $isNew) { Write-Host "autosync already running"; return }
Write-Log "START loop every $IntervalSeconds s"
try {
    while ($true) {
        try { Invoke-Pass } catch { Write-Log "ERROR $($_.Exception.Message)" }
        Start-Sleep -Seconds $IntervalSeconds
    }
} finally { $mutex.ReleaseMutex() }
