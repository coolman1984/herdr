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

function Start-FolderAgent([string]$ProfileName, [string]$PaneId) {
    $output = & pwsh -NoProfile -File (Join-Path $PSScriptRoot 'herdr-agents.ps1') start $ProfileName -Pane $PaneId 2>&1
    if ($LASTEXITCODE -ne 0) {
        Show-Error "Workspace created, but $ProfileName is not ready:`n$($output -join "`n")"
        return $false
    }
    return $true
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
    if (-not (Start-FolderAgent $first $paneId)) { exit 1 }

    if ($Agent -eq "team") {
        $output = & $herdr pane split $paneId --direction right --cwd $Path --no-focus 2>&1
        $splitCode = $LASTEXITCODE
        $codexPane = $null
        try { $codexPane = (($output -join "`n") | ConvertFrom-Json).result.pane.pane_id } catch { }
        if ($splitCode -ne 0 -or -not $codexPane) {
            Show-Error "Claude started, but the pane for Codex could not be created:`n$($output -join "`n")"
            exit 1
        }
        if (-not (Start-FolderAgent 'codex' $codexPane)) { exit 1 }
    }
}
