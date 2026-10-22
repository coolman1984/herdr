# Verify the whole Herdr setup: install, scripts, agent profiles, key bindings,
# skills, Windows integration and the documentation. Read-only.
#   pwsh -File check.ps1      exit 0 = no FAIL (WARN lines are advice)
$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"
$script:fails = 0

function Ok([string]$m) { Write-Output "OK    $m" }
function Warn([string]$m) { Write-Output "WARN  $m" }
function Bad([string]$m) { Write-Output "FAIL  $m"; $script:fails++ }

# 1. Herdr itself
if (Test-Path -LiteralPath $herdr) {
    Ok "Herdr installed: $((& $herdr --version) -join ' ')"
} else {
    Bad "Herdr not found at $herdr (docs/SETUP.md step 1)"
}

# 2. Every script parses
foreach ($file in Get-ChildItem -LiteralPath $root -Filter *.ps1 -File) {
    $errors = $null
    [System.Management.Automation.Language.Parser]::ParseFile($file.FullName, [ref]$null, [ref]$errors) | Out-Null
    if ($errors) { Bad "$($file.Name) does not parse: $($errors[0].Message)" } else { Ok "$($file.Name) parses" }
}

# 3. Agent profiles
$kinds = @()
if (Test-Path -LiteralPath $herdr) {
    $kindLine = (& $herdr agent 2>&1 | Out-String) -split "`n" | Where-Object { $_ -match '^\s*kinds:' } | Select-Object -First 1
    if ($kindLine) { $kinds = ($kindLine -replace '^\s*kinds:\s*', '').Trim() -split '\|' }
}
$dangerous = @('--yolo', '-y', '--dangerously-skip-permissions', '--dangerously-bypass-approvals-and-sandbox', '--full-auto')
try {
    $agents = (Get-Content -LiteralPath (Join-Path $root "agents.json") -Raw | ConvertFrom-Json).agents
    foreach ($p in $agents.PSObject.Properties) {
        $a = $p.Value
        foreach ($field in 'kind', 'command', 'modelFlag') {
            if (-not $a.$field) { Bad "agents.json '$($p.Name)' has no $field" }
        }
        if ($kinds -and $a.kind -notin $kinds) { Bad "agents.json '$($p.Name)': Herdr does not know kind '$($a.kind)'" }
        $flags = @($a.args)
        $hit = $flags | Where-Object { $_ -in $dangerous }
        if ($hit) { Bad "agents.json '$($p.Name)' auto-approves ($hit) - CLAUDE.md rule 2.3" }
        $i = [array]::IndexOf($flags, '--auto-approve')
        if ($i -ge 0 -and $flags[$i + 1] -ne 'false') { Bad "agents.json '$($p.Name)' sets --auto-approve $($flags[$i + 1])" }
        if ($a.kind -eq 'cline' -and $flags -notcontains '--auto-approve') { Bad "agents.json '$($p.Name)': Cline needs --auto-approve false (LESSONS #12)" }
        if (Get-Command $a.command -ErrorAction SilentlyContinue) { Ok "agent '$($p.Name)' ($($a.command)) installed" }
        else { Warn "agent '$($p.Name)': $($a.command) not installed" }
    }
} catch {
    Bad "agents.json unreadable: $($_.Exception.Message)"
}

# 4. Key bindings present in Herdr's live config
$live = Join-Path $env:APPDATA "herdr\config.toml"
$master = Get-Content -LiteralPath (Join-Path $root "config\keys.toml") -Raw
$liveText = if (Test-Path -LiteralPath $live) { Get-Content -LiteralPath $live -Raw } else { "" }
foreach ($m in [regex]::Matches($master, "(?m)^key = ""([^""]+)""\s*\r?\ntype = ""([^""]+)""\s*\r?\ncommand = '([^']+)'")) {
    $key = $m.Groups[1].Value
    $block = "key = ""$key"""
    if ($liveText.Contains($block) -and $liveText.Contains($m.Groups[3].Value)) { Ok "key $key in Herdr config" }
    else { Bad "key $key from config/keys.toml missing or different in $live" }
}

# 4b. Settings present in Herdr's live config (config/settings.toml)
foreach ($line in (Get-Content -LiteralPath (Join-Path $root "config\settings.toml"))) {
    $line = $line.Trim()
    if (-not $line -or $line.StartsWith("#")) { continue }
    if ($liveText.Contains($line)) { Ok "setting $line in Herdr config" }
    else { Bad "setting '$line' from config/settings.toml missing in $live" }
}

# 5. Skills installed = master copies
$sync = & pwsh -NoProfile -File (Join-Path $root "sync-skills.ps1") -Check
foreach ($line in $sync) {
    if ($line -like "OK*") { Ok "skill $($line.Substring(8).Trim()) in sync" } else { Bad "skill $line" }
}

# 6. Windows integration
$appPath = "HKCU:\Software\Microsoft\Windows\CurrentVersion\App Paths\herdr.exe"
$target = (Get-ItemProperty -LiteralPath $appPath -ErrorAction SilentlyContinue).'(default)'
if ($target -and (Test-Path -LiteralPath $target)) { Ok "'herdr' in Explorer -> $target" } else { Bad "App Path herdr.exe missing or broken (docs/SETUP.md step 7)" }
foreach ($lnk in @((Join-Path ([Environment]::GetFolderPath('Desktop')) "Herdr.lnk"), (Join-Path ([Environment]::GetFolderPath('Programs')) "Herdr.lnk"))) {
    if (Test-Path -LiteralPath $lnk) { Ok "shortcut $lnk" } else { Warn "shortcut missing: $lnk" }
}
$userPath = [Environment]::GetEnvironmentVariable("Path", "User") -split ";"
if ($userPath -contains $root) { Ok "this folder on user PATH (herdr-agents)" } else { Warn "this folder is not on the user PATH" }
if (Test-Path -LiteralPath $herdr) {
    $integration = (& $herdr integration status 2>&1 | Out-String) -split "`n" | Where-Object { $_ -match '^(claude|codex):' }
    foreach ($line in $integration) {
        if ($line -match 'not installed') { Warn "integration $($line.Trim()) - owner runs: herdr integration install $($line.Split(':')[0])" }
        else { Ok "integration $($line.Trim())" }
    }
}

# 7. Documentation agrees with the folder
$files = Get-Content -LiteralPath (Join-Path $root "docs\FILES.md") -Raw
$inFolder = ($files -split '## Outside this folder')[0]
foreach ($m in [regex]::Matches($inFolder, '\|\s*`([^`]+)`')) {
    foreach ($name in ($m.Groups[1].Value -split '`, `')) {
        $path = Join-Path $root ($name.TrimEnd('/') -replace '/', '\')
        if (-not (Test-Path -LiteralPath $path)) { Bad "docs/FILES.md lists '$name' but it does not exist" }
    }
}
foreach ($file in Get-ChildItem -LiteralPath $root -File | Where-Object { $_.Name -notlike '.*' }) {
    if (-not $files.Contains("``$($file.Name)``")) { Bad "$($file.Name) is not described in docs/FILES.md" }
}
$lessons = Get-Content -LiteralPath (Join-Path $root "LESSONS.md") -Raw
$lessonNumbers = [regex]::Matches($lessons, '(?m)^(\d+)\.\s') | ForEach-Object { [int]$_.Groups[1].Value }
for ($i = 0; $i -lt $lessonNumbers.Count; $i++) {
    if ($lessonNumbers[$i] -ne $i + 1) { Bad "LESSONS.md numbering breaks at $($lessonNumbers[$i])"; break }
}
$referencing = @("HISTORY.md", "CLAUDE.md", "docs\SETUP.md", "docs\DELEGATION.md") + (Get-ChildItem -LiteralPath (Join-Path $root "skills") -Recurse -Filter SKILL.md | ForEach-Object { $_.FullName.Substring($root.Length + 1) })
foreach ($doc in $referencing) {
    $text = Get-Content -LiteralPath (Join-Path $root $doc) -Raw
    foreach ($m in [regex]::Matches($text, '(?:LESSONS |Lessons?:[^\n]*?)#(\d+)')) {
        if ([int]$m.Groups[1].Value -notin $lessonNumbers) { Bad "$doc points to LESSONS #$($m.Groups[1].Value), which does not exist" }
    }
}
$history = Get-Content -LiteralPath (Join-Path $root "HISTORY.md") -Raw
$phases = [regex]::Matches($history, '(?m)^## Phase (\d+)') | ForEach-Object { [int]$_.Groups[1].Value }
for ($i = 0; $i -lt $phases.Count; $i++) {
    if ($phases[$i] -ne $i + 1) { Bad "HISTORY.md phase numbering breaks at Phase $($phases[$i])"; break }
}
Ok "docs: $($phases.Count) history phases, $($lessonNumbers.Count) lessons"

Write-Output ""
if ($script:fails) { Write-Output "$($script:fails) FAIL"; exit 1 }
Write-Output "All checks passed."
