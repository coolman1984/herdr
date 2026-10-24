# Herdr "open folder": pick a folder with the normal Windows dialog and open it
# as a new Herdr workspace. Bound in %APPDATA%\herdr\config.toml:
#   prefix+f        -> new workspace in the chosen folder
#   prefix+shift+f  -> same, then start Claude Code in it
#   prefix+t        -> team: Claude Code on the left, Codex (SOL 6.1) on the right
#
# Manual use:  pwsh -STA -File herdr-open-folder.ps1 [-Agent claude|codex|team] [-Path D:\some\folder]
[CmdletBinding()]
param(
    [ValidateSet("", "claude", "codex", "team")]
    [string]$Agent = "",
    [string]$Path = ""
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms

$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"
$stateFile = Join-Path $env:APPDATA "herdr\open-folder-last.txt"
$defaultFolder = $PSScriptRoot

function Show-Error([string]$Message) {
    [System.Windows.Forms.MessageBox]::Show($Message, "Herdr - open folder", "OK", "Error") | Out-Null
}

function Select-Folder([string]$Start) {
    # A hidden top-most owner keeps the dialog in front of Windows Terminal.
    $owner = New-Object System.Windows.Forms.Form -Property @{
        TopMost = $true; ShowInTaskbar = $false; Opacity = 0
        StartPosition = "CenterScreen"; Width = 1; Height = 1
    }
    $owner.Show()
    $owner.Activate()
    $dialog = New-Object System.Windows.Forms.FolderBrowserDialog -Property @{
        Description = "Open a folder in a new Herdr workspace"
        UseDescriptionForTitle = $true
        ShowNewFolderButton = $true
        InitialDirectory = $Start
    }
    try {
        if ($dialog.ShowDialog($owner) -eq [System.Windows.Forms.DialogResult]::OK) {
            return $dialog.SelectedPath
        }
        return $null
    } finally {
        $dialog.Dispose()
        $owner.Close()
    }
}

if (-not (Test-Path -LiteralPath $herdr)) {
    Show-Error "Herdr is not installed at $herdr."
    exit 1
}

if (-not $Path) {
    $start = $defaultFolder
    if (Test-Path -LiteralPath $stateFile) {
        $last = (Get-Content -LiteralPath $stateFile -TotalCount 1).Trim()
        if ($last -and (Test-Path -LiteralPath $last -PathType Container)) { $start = $last }
    }
    $Path = Select-Folder $start
    if (-not $Path) { exit 0 }   # cancelled
}

if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
    Show-Error "Folder not found:`n$Path"
    exit 1
}
$Path = (Resolve-Path -LiteralPath $Path).ProviderPath
$label = Split-Path -Leaf $Path.TrimEnd("\")
if (-not $label) { $label = $Path }

$output = & $herdr workspace create --cwd $Path --label $label --focus 2>&1
if ($LASTEXITCODE -ne 0) {
    Show-Error "Herdr could not create the workspace:`n$($output -join "`n")"
    exit 1
}
Set-Content -LiteralPath $stateFile -Value $Path -Encoding utf8

if ($Agent) {
    $paneId = $null
    try {
        $json = ($output -join "`n") | ConvertFrom-Json
        $result = if ($json.PSObject.Properties["result"]) { $json.result } else { $json }
        $paneId = $result.root_pane.pane_id
    } catch { }
    if (-not $paneId) {
        Show-Error "The workspace was created, but its pane id was not in Herdr's answer, so $Agent was not started.`n`n$($output -join "`n")"
        exit 1
    }
    $first = if ($Agent -eq "team") { "claude" } else { $Agent }
    $output = & $herdr pane run $paneId $first 2>&1
    if ($LASTEXITCODE -ne 0) {
        Show-Error "The workspace was created, but $first could not be started:`n$($output -join "`n")"
        exit 1
    }

    if ($Agent -eq "team") {
        # Codex goes in a pane to the right, named so Claude can address it
        # ("codex-w3" for workspace w3): herdr agent prompt codex-w3 "...".
        $output = & $herdr pane split $paneId --direction right --cwd $Path --no-focus 2>&1
        $codexPane = $null
        try { $codexPane = (($output -join "`n") | ConvertFrom-Json).result.pane.pane_id } catch { }
        if (-not $codexPane) {
            Show-Error "Claude started, but the pane for Codex could not be created:`n$($output -join "`n")"
            exit 1
        }
        $name = "codex-" + ($paneId -split ":")[0]
        $modelArgs = @()
        $profilesPath = Join-Path $PSScriptRoot "agents.json"
        if (Test-Path -LiteralPath $profilesPath) {
            $codexProfile = (Get-Content -LiteralPath $profilesPath -Raw | ConvertFrom-Json).agents.codex
            if ($codexProfile.modelFlag -and $codexProfile.model) { $modelArgs = @("--", $codexProfile.modelFlag, $codexProfile.model) }
        }
        $output = & $herdr agent start $name --kind codex --pane $codexPane --timeout 90000 @modelArgs 2>&1
        if ($LASTEXITCODE -ne 0) {
            Show-Error "Claude started, but Codex did not become ready in its pane:`n$($output -join "`n")"
            exit 1
        }
    }
}
