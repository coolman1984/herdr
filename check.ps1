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
        # Owner decision 2026-10-22 (CLAUDE.md rule 3): no approval prompts, but nothing that
        # removes Codex's sandbox or skips every permission outright.
        # Owner decision 2026-10-22 (HISTORY Phase 19): Antigravity may skip all permissions.
        $hit = $flags | Where-Object { $_ -in $dangerous -and -not ($a.kind -eq 'agy' -and $_ -eq '--dangerously-skip-permissions') }
        if ($a.kind -eq 'codex') {
            for ($j = 0; $j -lt $flags.Count; $j++) {
                if (($flags[$j] -in @('-s', '--sandbox') -and $j + 1 -lt $flags.Count -and $flags[$j + 1] -eq 'danger-full-access') -or
                    $flags[$j] -match '^(?:-s|--sandbox)=danger-full-access$') { $hit += @('danger-full-access') }
            }
        }
        if ($a.kind -eq 'agy' -and $flags -contains '--dangerously-skip-permissions') { $flags += 'auto' }
        if ($hit) { Bad "agents.json '$($p.Name)' uses $hit - wider than the owner's decision (CLAUDE.md rule 3)" }
        $auto = @($flags | Where-Object { $_ -in @('auto', 'never', '--always-approve') })
        $i = [array]::IndexOf($flags, '--auto-approve')
        if ($i -ge 0 -and $flags[$i + 1] -eq 'true') { $auto += '--auto-approve true' }
        if ($auto) { Ok "agent '$($p.Name)' runs without approval prompts (owner decision, CLAUDE.md rule 3)" }
        if (Get-Command $a.command -ErrorAction SilentlyContinue) { Ok "agent '$($p.Name)' ($($a.command)) installed" }
        else { Warn "agent '$($p.Name)': $($a.command) not installed" }
    }
} catch {
    Bad "agents.json unreadable: $($_.Exception.Message)"
}

# 4. Key bindings present in Herdr's live config
function Get-KeyCommands([string]$Text, [string]$Source) {
    $map = @{}
    foreach ($b in [regex]::Matches($Text, '(?ms)^[ \t]*\[\[keys\.command\]\][ \t]*(?:#[^\r\n]*)?\r?\n(.*?)(?=^[ \t]*\[|\z)')) {
        $values = @{}
        foreach ($m in [regex]::Matches($b.Groups[1].Value, '(?m)^\s*(key|type|command)\s*=\s*(?:"((?:\\.|[^"\\])*)"|''([^'']*)'')\s*(?:#.*)?$')) {
            # Double-quoted TOML strings use JSON-compatible escapes in these configs.
            $value = if ($m.Groups[2].Success) { ('"' + $m.Groups[2].Value + '"') | ConvertFrom-Json } else { $m.Groups[3].Value }
            $values[$m.Groups[1].Value] = $value
        }
        $key = $values['key']
        if (-not $key) { Bad "$Source has a keys.command block without a key"; continue }
        if ($map.ContainsKey($key)) { Bad "duplicate key $key in $Source"; continue }
        $map[$key] = $values
    }
    return ,$map
}
$live = Join-Path $env:APPDATA "herdr\config.toml"
$master = Get-Content -LiteralPath (Join-Path $root "config\keys.toml") -Raw
$liveText = if (Test-Path -LiteralPath $live) { Get-Content -LiteralPath $live -Raw } else { "" }
$masterKeys = Get-KeyCommands $master 'config/keys.toml'
$liveKeys = Get-KeyCommands $liveText $live
foreach ($key in $masterKeys.Keys) {
    if ($liveKeys.ContainsKey($key) -and $masterKeys[$key]['type'] -and $masterKeys[$key]['command'] -and
        $liveKeys[$key]['type'] -ceq $masterKeys[$key]['type'] -and $liveKeys[$key]['command'] -ceq $masterKeys[$key]['command']) { Ok "key $key in Herdr config" }
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
$hudLink = Join-Path ([Environment]::GetFolderPath('Startup')) "Herdr HUD.lnk"
if (Test-Path -LiteralPath $hudLink) { Ok "pane HUD starts at sign-in ($hudLink)" } else { Warn "pane HUD not started at sign-in: $hudLink missing" }
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
foreach ($row in [regex]::Matches($inFolder, '(?m)^\|([^|]*)\|')) {
    foreach ($m in [regex]::Matches($row.Groups[1].Value, '`([^`]+)`')) {
        $name = $m.Groups[1].Value
        $path = Join-Path $root ($name.TrimEnd('/') -replace '/', '\')
        if (-not (Test-Path -LiteralPath $path)) { Bad "docs/FILES.md lists '$name' but it does not exist" }
    }
}
foreach ($file in Get-ChildItem -LiteralPath $root -File | Where-Object { $_.Name -notlike '.*' -and $_.Extension -ne '.bak' }) {
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
