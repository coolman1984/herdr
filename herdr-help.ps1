# Help window: searchable shortcuts from this setup and Herdr, plus the user guide.
# Bound to ctrl+b i (the binding is maintained separately in config/keys.toml).
#
#   pwsh -NoProfile -STA -File herdr-help.ps1
#   pwsh -NoProfile -File herdr-help.ps1 -List   print shortcuts without a window
[CmdletBinding()]
param([switch]$List)

$ErrorActionPreference = 'Stop'

function Format-HelpKey([string]$Key) {
    $parts = $Key -split '\+'
    $prefix = $parts[0] -eq 'prefix'
    if ($prefix) { $parts = $parts | Select-Object -Skip 1 }
    $formatted = foreach ($part in $parts) {
        switch ($part.ToLowerInvariant()) {
            'shift' { 'Shift' }
            'ctrl' { 'Ctrl' }
            'alt' { 'Alt' }
            default { $part.ToUpperInvariant() }
        }
    }
    $result = $formatted -join '+'
    if ($prefix) { return "Ctrl+B, then $result" }
    return $result
}

function Get-HelpRows {
    $lines = @(Get-Content -LiteralPath (Join-Path $PSScriptRoot 'config\keys.toml'))
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -notmatch '^\s*\[\[keys\.command\]\]\s*$') { continue }
        $comments = [System.Collections.Generic.List[string]]::new()
        for ($j = $i - 1; $j -ge 0 -and $lines[$j] -match '^\s*#\s?(.*)$'; $j--) {
            $comments.Insert(0, $Matches[1])
        }
        $description = $comments -join ' '
        # "Title: ctrl+b then x -> what it does": keep what follows the first "->"
        # (the keys are already in the Keys column), else what follows "Title: ".
        $arrow = $description.IndexOf('->')
        $colon = $description.IndexOf(': ')
        if ($arrow -ge 0) { $description = $description.Substring($arrow + 2).Trim() }
        elseif ($colon -ge 0) { $description = $description.Substring($colon + 2) }
        if ($description) { $description = $description.Substring(0, 1).ToUpperInvariant() + $description.Substring(1) }
        for ($j = $i + 1; $j -lt $lines.Count -and $lines[$j] -notmatch '^\s*\['; $j++) {
            if ($lines[$j] -match '^\s*key\s*=\s*"([^"]+)"') {
                [pscustomobject]@{ Keys = Format-HelpKey $Matches[1]; 'What it does' = $description; From = 'this setup' }
                break
            }
        }
    }

    $builtIn = @(
        ,@('?', "Herdr's own key list")
        ,@('V', 'Split side by side')
        ,@('-', 'Split stacked')
        ,@('X', 'Close the pane')
        ,@('Z', 'Zoom the pane / back')
        ,@('R', 'Resize mode (arrows, Esc)')
        ,@('H/J/K/L or arrows', 'Move to the pane left/down/up/right')
        ,@('Shift+H/J/K/L', 'Swap with the neighbour')
        ,@('Tab / Shift+Tab', 'Next / previous pane')
        ,@('C', 'New tab')
        ,@('N / P', 'Next / previous tab')
        ,@('1..9', 'Go to tab 1..9')
        ,@('Shift+T', 'Rename the tab')
        ,@('Shift+X', 'Close the tab')
        ,@('Shift+N', 'New space')
        ,@('Shift+W', 'Rename the space')
        ,@('Shift+D', 'Close the space')
        ,@('W', 'Space picker')
        ,@('G', 'Go to any space, tab or agent')
        ,@('O', 'Jump to the agent that notified you')
        ,@('Shift+P', 'Rename the pane (its name shows on the border and in the sidebar)')
        ,@('[', 'Copy mode')
        ,@('E', 'Open the scrollback in an editor')
        ,@('B', 'Hide / show the sidebar')
        ,@('S', 'Settings')
        ,@('Shift+R', 'Reload the config')
        ,@('Q', 'Detach (agents keep running)')
    )
    foreach ($row in $builtIn) {
        [pscustomobject]@{ Keys = "Ctrl+B, then $($row[0])"; 'What it does' = $row[1]; From = 'Herdr' }
    }
    foreach ($row in @(
        ,@('herdr-agents status', 'Who is running and their state')
        ,@('herdr-agents ask <name> "<task>"', 'Give an agent a task and wait for the answer')
        ,@('herdr-agents history', 'Every task and answer saved.')
    )) {
        [pscustomobject]@{ Keys = $row[0]; 'What it does' = $row[1]; From = 'command' }
    }
}

if ($List) {
    foreach ($row in Get-HelpRows) {
        '{0} | {1} | {2}' -f $row.Keys, $row.'What it does', $row.From
    }
    exit 0
}

Add-Type -AssemblyName System.Windows.Forms, System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

function Show-HelpError([string]$Text) {
    if ($form -and -not $form.IsDisposed) {
        [System.Windows.Forms.MessageBox]::Show($form, $Text, 'Herdr - help', 'OK', 'Error') | Out-Null
    } else {
        [System.Windows.Forms.MessageBox]::Show($Text, 'Herdr - help', 'OK', 'Error') | Out-Null
    }
}

try {
    $rows = @(Get-HelpRows)
    $guidePath = Join-Path $PSScriptRoot 'docs\GUIDE.md'
    $rawGuide = Get-Content -LiteralPath $guidePath -Raw
    $guideHtml = $null
    try {
        $body = (ConvertFrom-Markdown -Path $guidePath).Html
        $guideHtml = @"
<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta http-equiv="X-UA-Compatible" content="IE=edge">
<style>
body { font-family: Segoe UI, sans-serif; font-size: 14px; line-height: 1.55; max-width: 860px; margin: 24px auto; padding: 0 20px; color: #222; }
table { border-collapse: collapse; width: 100%; margin: 12px 0; }
th, td { border: 1px solid #bbb; padding: 7px 10px; text-align: left; }
th { background: #eee; } code, pre { font-family: Consolas, monospace; }
pre { padding: 12px; background: #f3f3f3; overflow: auto; }
</style></head><body>$body</body></html>
"@
    } catch { Show-HelpError "The guide could not be rendered. Showing Markdown instead.`n$($_.Exception.Message)" }

    $form = [System.Windows.Forms.Form]::new()
    $form.Text = 'Herdr - shortcuts and guide'
    $form.Size = [System.Drawing.Size]::new(900, 640)
    $form.MinimumSize = [System.Drawing.Size]::new(620, 400)
    $form.StartPosition = 'CenterScreen'
    $form.TopMost = $true
    $form.Font = [System.Drawing.Font]::new('Segoe UI', 9.5)
    $form.AutoScaleMode = 'Dpi'
    $form.KeyPreview = $true
    $form.Add_KeyDown({ if ($_.KeyCode -eq 'Escape') { $_.SuppressKeyPress = $true; $form.Close() } })
    $form.Add_Shown({ $form.Activate() })

    $tabs = [System.Windows.Forms.TabControl]::new()
    $tabs.Dock = 'Fill'
    $shortcuts = [System.Windows.Forms.TabPage]::new('Shortcuts')
    $guide = [System.Windows.Forms.TabPage]::new('Guide')
    $tabs.TabPages.AddRange(@($shortcuts, $guide))

    $search = [System.Windows.Forms.TextBox]::new()
    $search.Dock = 'Top'
    $search.PlaceholderText = 'Type to filter, e.g. split, agent, folder'
    $view = [System.Windows.Forms.ListView]::new()
    $view.Dock = 'Fill'
    $view.View = 'Details'
    $view.FullRowSelect = $true
    $view.GridLines = $true
    $view.Columns.Add('Keys', 180) | Out-Null
    $view.Columns.Add('What it does', 570) | Out-Null
    $view.Columns.Add('From', 90) | Out-Null
    $view.Add_Resize({ $view.Columns[1].Width = [Math]::Max(150, $view.ClientSize.Width - 180 - 90 - 24) })
    $shortcuts.Controls.Add($view)
    $shortcuts.Controls.Add($search)

    function Update-HelpRows {
        $view.BeginUpdate()
        try {
            $view.Items.Clear()
            foreach ($row in $rows) {
                [string[]]$values = @([string]$row.Keys, [string]$row.'What it does', [string]$row.From)
                if ($search.Text -and -not ($values | Where-Object { $_.IndexOf($search.Text, [StringComparison]::OrdinalIgnoreCase) -ge 0 })) { continue }
                $item = [System.Windows.Forms.ListViewItem]::new($values)
                $view.Items.Add($item) | Out-Null
            }
        } finally { $view.EndUpdate() }
    }
    $search.Add_TextChanged({ try { Update-HelpRows } catch { Show-HelpError $_.Exception.Message } })
    Update-HelpRows

    $script:helpBrowser = $null
    $script:helpRawShown = $false
    function Show-RawHelpGuide {
        if ($script:helpRawShown) { return }
        $script:helpRawShown = $true
        if ($script:helpBrowser) { $guide.Controls.Remove($script:helpBrowser) }
        $raw = [System.Windows.Forms.TextBox]::new()
        $raw.Dock = 'Fill'
        $raw.Multiline = $true
        $raw.ReadOnly = $true
        $raw.ScrollBars = 'Both'
        $raw.WordWrap = $false
        $raw.Text = $rawGuide
        $guide.Controls.Add($raw)
    }
    if ($guideHtml) {
        try {
            $script:helpBrowser = [System.Windows.Forms.WebBrowser]::new()
            $script:helpBrowser.Dock = 'Fill'
            $script:helpBrowser.AllowWebBrowserDrop = $false
            $script:helpBrowser.ScriptErrorsSuppressed = $true
            $script:helpBrowser.Add_Navigating({
                if ($_.Url.AbsoluteUri -ne 'about:blank') { $_.Cancel = $true }
            })
            $script:helpBrowser.Add_DocumentCompleted({
                try {
                    if (-not $script:helpBrowser.Document -or -not $script:helpBrowser.Document.Body) { Show-RawHelpGuide }
                } catch { Show-RawHelpGuide }
            })
            $guide.Controls.Add($script:helpBrowser)
            $script:helpBrowser.DocumentText = $guideHtml
        } catch { Show-RawHelpGuide }
    } else { Show-RawHelpGuide }

    # Some WebBrowser failures are asynchronous and never raise DocumentCompleted.
    $guideTimer = [System.Windows.Forms.Timer]::new()
    $guideTimer.Interval = 5000
    $guideTimer.Add_Tick({
        $guideTimer.Stop()
        try {
            if (-not $script:helpRawShown -and (-not $script:helpBrowser.Document -or -not $script:helpBrowser.Document.Body)) { Show-RawHelpGuide }
        } catch { Show-RawHelpGuide }
    })
    $form.Add_Shown({ $guideTimer.Start() })

    $buttons = [System.Windows.Forms.FlowLayoutPanel]::new()
    $buttons.Dock = 'Bottom'
    $buttons.Height = 48
    $buttons.Padding = [System.Windows.Forms.Padding]::new(8)
    $buttons.FlowDirection = 'RightToLeft'
    $close = [System.Windows.Forms.Button]::new()
    $close.Text = 'Close'
    $close.AutoSize = $true
    $close.Add_Click({ $form.Close() })
    $form.CancelButton = $close
    $open = [System.Windows.Forms.Button]::new()
    $open.Text = 'Open guide in browser'
    $open.AutoSize = $true
    $open.Enabled = [bool]$guideHtml
    $open.Add_Click({
        try {
            $htmlPath = Join-Path $env:TEMP 'herdr-guide.html'
            Set-Content -LiteralPath $htmlPath -Value $guideHtml -Encoding utf8
            Start-Process -FilePath $htmlPath -ErrorAction Stop | Out-Null
        } catch { Show-HelpError $_.Exception.Message }
    })
    $buttons.Controls.AddRange(@($close, $open))
    $form.Controls.Add($tabs)
    $form.Controls.Add($buttons)
    try { $form.ShowDialog() | Out-Null }
    finally { $guideTimer.Dispose(); $form.Dispose(); if ($script:helpBrowser) { $script:helpBrowser.Dispose() } }
} catch { Show-HelpError $_.Exception.Message }
