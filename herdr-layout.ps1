# Layout builder: a window to open N side-by-side columns, choosing for EACH column
# its project folder and what runs in it - an empty PowerShell, a new agent
# (profiles from agents.json), or an agent that is already running (its pane is
# moved into the column; the agent keeps running). Bound to ctrl+b shift+b.
#
#   pwsh -STA -File herdr-layout.ps1
#   pwsh -File herdr-layout.ps1 -PlanFile plan.json   no window (for agents and tests):
#     { "Where": 0|1, "Name": "work", "WorkspaceId": "w1" (Where 0 only),
#       "Columns": [ { "Folder": "D:\\x", "Type": "shell|new|existing",
#                      "Value": "<profile or pane id>", "Model": "" } ] }
#   pwsh -File herdr-layout.ps1 -Preset morning      a preset saved in the window, no window
#
# Presets ("Save preset..." in the window) live in %APPDATA%\herdr\layouts\<name>.json and
# hold folders and new agents only; a running agent is saved as an empty PowerShell.
[CmdletBinding()]
param([string]$PlanFile, [string]$Preset)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms, System.Drawing, Microsoft.VisualBasic
[System.Windows.Forms.Application]::EnableVisualStyles()

$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"
$agentsTool = Join-Path $PSScriptRoot "herdr-agents.ps1"
$lastPath = Join-Path $env:APPDATA "herdr\layout-last.json"
$presetDir = Join-Path $env:APPDATA "herdr\layouts"
$panesState = Join-Path $env:APPDATA "herdr\herdr-agents-panes.json"
$defaultFolder = "D:\WORK\Software Development\GitHub"
$noWindow = [bool]($PlanFile -or $Preset)

function Show-Message([string]$Text, [string]$Icon = "Error") {
    if ($noWindow) { Write-Output "$($Icon.ToUpper()): $Text"; return }
    [System.Windows.Forms.MessageBox]::Show($Text, "Herdr - layout", "OK", $Icon) | Out-Null
}

function Invoke-Herdr([string[]]$Arguments) {
    $out = & $herdr @Arguments 2>&1
    $text = ($out | ForEach-Object { $_.ToString() }) -join "`n"
    $json = $null
    try { $json = $text | ConvertFrom-Json } catch { }
    [PSCustomObject]@{ Ok = ($LASTEXITCODE -eq 0); Text = $text; Json = $json }
}

if (-not (Test-Path -LiteralPath $herdr)) { Show-Message "Herdr is not installed at $herdr."; exit 1 }

# ---- What exists now -------------------------------------------------------
$current = (Invoke-Herdr @("pane", "current")).Json.result.pane
$workspaceId = if ($current) { $current.workspace_id } else { $null }
$startFolder = if ($current -and $current.cwd) { $current.cwd } else { $defaultFolder }

$labels = @{}
$wsList = Invoke-Herdr @("workspace", "list")
if ($wsList.Ok) { foreach ($w in $wsList.Json.result.workspaces) { $labels[$w.workspace_id] = $w.label } }

# Options for the "Runs" list: index -> @{ Type; Value; Text; Cwd }
$options = New-Object System.Collections.Generic.List[object]
$options.Add(@{ Type = "shell"; Value = ""; Text = "PowerShell (empty)"; Cwd = $null })
$registry = (Get-Content -LiteralPath (Join-Path $PSScriptRoot "agents.json") -Raw | ConvertFrom-Json).agents
foreach ($p in $registry.PSObject.Properties) {
    if (-not (Get-Command $p.Value.command -ErrorAction SilentlyContinue)) { continue }
    $model = if ($p.Value.model) { $p.Value.model } else { "default model" }
    $options.Add(@{ Type = "new"; Value = $p.Name; Text = "New: $($p.Name)  [$model]"; Cwd = $null })
}
$agentList = Invoke-Herdr @("agent", "list")
if ($agentList.Ok) {
    foreach ($a in $agentList.Json.result.agents) {
        $who = if ($a.name) { "$($a.name) ($($a.agent))" } else { $a.agent }
        $space = if ($labels[$a.workspace_id]) { $labels[$a.workspace_id] } else { $a.workspace_id }
        $folder = Split-Path -Leaf ([string]$a.cwd).TrimEnd("\")
        $options.Add(@{ Type = "existing"; Value = $a.pane_id; Cwd = $a.cwd
            Text = "Running: $who - $folder - space $space - $($a.agent_status)" })
    }
}

$last = $null
if (Test-Path -LiteralPath $lastPath) { try { $last = Get-Content -LiteralPath $lastPath -Raw | ConvertFrom-Json } catch { } }

function Get-PresetNames {
    @(Get-ChildItem -LiteralPath $presetDir -Filter *.json -ErrorAction SilentlyContinue | Sort-Object Name | ForEach-Object { $_.BaseName })
}

$plan = $null
if ($PlanFile) {
    $plan = Get-Content -LiteralPath $PlanFile -Raw | ConvertFrom-Json
    if ($plan.PSObject.Properties["WorkspaceId"] -and $plan.WorkspaceId) { $workspaceId = $plan.WorkspaceId }
} elseif ($Preset) {
    $file = Join-Path $presetDir "$Preset.json"
    if (-not (Test-Path -LiteralPath $file)) { Show-Message "No preset '$Preset'. Saved: $((Get-PresetNames) -join ', ')"; exit 1 }
    $p = Get-Content -LiteralPath $file -Raw | ConvertFrom-Json
    $where = if ($workspaceId) { [int]$p.where } else { 1 }
    $plan = [PSCustomObject]@{ Where = $where; Name = $p.name; Columns = @($p.columns | ForEach-Object {
        [PSCustomObject]@{ Folder = $_.folder; Type = $(if ($_.runs) { "new" } else { "shell" }); Value = $_.runs; Model = $_.model }
    }) }
    foreach ($c in $plan.Columns) {
        if (-not (Test-Path -LiteralPath $c.Folder -PathType Container)) { Show-Message "Preset '$Preset': folder not found: $($c.Folder)"; exit 1 }
    }
} else {
# ---- The window ---------------------------------------------------------------
$form =New-Object System.Windows.Forms.Form -Property @{
    Text = "Herdr - open side-by-side columns"; StartPosition = "CenterScreen"; TopMost = $true
    FormBorderStyle = "FixedDialog"; MaximizeBox = $false; MinimizeBox = $false
    Font = New-Object System.Drawing.Font("Segoe UI", 9.5); AutoScaleMode = "Dpi"
    ClientSize = New-Object System.Drawing.Size(980, 200)
}

function New-Label($text, $x, $y, $w = 90) {
    $l = New-Object System.Windows.Forms.Label -Property @{ Text = $text; Left = $x; Top = $y + 4; Width = $w; AutoSize = $false }
    $form.Controls.Add($l); $l
}

New-Label "Columns" 16 14 60 | Out-Null
$count = New-Object System.Windows.Forms.NumericUpDown -Property @{ Left = 80; Top = 14; Width = 50; Minimum = 1; Maximum = 6; Value = 4 }
if ($last -and $last.count) { $count.Value = [Math]::Min(6, [Math]::Max(1, [int]$last.count)) }
$form.Controls.Add($count)

New-Label "Open in" 150 14 55 | Out-Null
$where = New-Object System.Windows.Forms.ComboBox -Property @{ Left = 208; Top = 14; Width = 190; DropDownStyle = "DropDownList" }
[void]$where.Items.AddRange(@("New tab in this space", "New space"))
$where.SelectedIndex = if ($last -and $last.where -eq 1) { 1 } else { 0 }
if (-not $workspaceId) { $where.SelectedIndex = 1; $where.Enabled = $false }
$form.Controls.Add($where)

New-Label "Name" 416 14 42 | Out-Null
$nameBox = New-Object System.Windows.Forms.TextBox -Property @{ Left = 460; Top = 14; Width = 160; Text = "work" }
if ($last -and $last.name) { $nameBox.Text = $last.name }
$form.Controls.Add($nameBox)

New-Label "Preset" 636 14 48 | Out-Null
$presetBox = New-Object System.Windows.Forms.ComboBox -Property @{ Left = 686; Top = 14; Width = 160; DropDownStyle = "DropDownList" }
[void]$presetBox.Items.Add("(none)")
foreach ($n in Get-PresetNames) { [void]$presetBox.Items.Add($n) }
$presetBox.SelectedIndex = 0
$form.Controls.Add($presetBox)
$savePreset = New-Object System.Windows.Forms.Button -Property @{ Text = "Save preset..."; Left = 852; Top = 13; Width = 112; Height = 28 }
$form.Controls.Add($savePreset)
$script:resetRows = $false

$hint = New-Label "Running agents are MOVED into their column and keep running; their folder stays theirs." 16 46 900
$hint.ForeColor = [System.Drawing.Color]::DimGray

$rows = New-Object System.Collections.Generic.List[object]
$ok = New-Object System.Windows.Forms.Button -Property @{ Text = "Open layout"; Width = 120; Height = 32 }
$cancel = New-Object System.Windows.Forms.Button -Property @{ Text = "Cancel"; Width = 100; Height = 32; DialogResult = "Cancel" }
$form.Controls.AddRange(@($ok, $cancel))
$form.AcceptButton = $ok
$form.CancelButton = $cancel

function Update-RowState($row) {
    $opt = $options[$row.Runs.SelectedIndex]
    if ($opt.Type -eq "existing") {
        $row.Folder.Text = $opt.Cwd
        $row.Folder.Enabled = $false; $row.Browse.Enabled = $false; $row.Model.Enabled = $false
    } else {
        $row.Folder.Enabled = $true; $row.Browse.Enabled = $true
        $row.Model.Enabled = ($opt.Type -eq "new")
    }
}

function Build-Rows {
    $keep = if ($script:resetRows) { @() } else { @($rows | ForEach-Object { @{ Folder = $_.Folder.Text; Runs = $_.Runs.SelectedIndex; Model = $_.Model.Text } }) }
    $script:resetRows = $false
    foreach ($r in $rows) { foreach ($c in $r.Controls) { $form.Controls.Remove($c); $c.Dispose() } }
    $rows.Clear()
    $n = [int]$count.Value
    for ($i = 0; $i -lt $n; $i++) {
        $y = 80 + $i * 38
        $label = New-Object System.Windows.Forms.Label -Property @{ Text = "Column $($i + 1)"; Left = 16; Top = $y + 4; Width = 70 }
        $folder = New-Object System.Windows.Forms.TextBox -Property @{ Left = 88; Top = $y; Width = 330 }
        $browse = New-Object System.Windows.Forms.Button -Property @{ Text = "Browse..."; Left = 422; Top = $y - 1; Width = 80; Height = 28 }
        $runs = New-Object System.Windows.Forms.ComboBox -Property @{ Left = 510; Top = $y; Width = 330; DropDownStyle = "DropDownList" }
        foreach ($o in $options) { [void]$runs.Items.Add($o.Text) }
        $model = New-Object System.Windows.Forms.TextBox -Property @{ Left = 848; Top = $y; Width = 116; PlaceholderText = "model (optional)" }

        # Defaults: what this row had, else the last layout, else the current folder.
        $folder.Text = $startFolder
        $runs.SelectedIndex = 0
        if ($i -lt $keep.Count) {
            $folder.Text = $keep[$i].Folder; $runs.SelectedIndex = [Math]::Min($keep[$i].Runs, $options.Count - 1); $model.Text = $keep[$i].Model
        } elseif ($last -and $last.columns -and $i -lt @($last.columns).Count) {
            $col = @($last.columns)[$i]
            if ($col.folder -and (Test-Path -LiteralPath $col.folder)) { $folder.Text = $col.folder }
            if ($col.runs) {
                $match = 0
                for ($k = 0; $k -lt $options.Count; $k++) {
                    if ($options[$k].Type -eq "new" -and $options[$k].Value -eq $col.runs) { $match = $k }
                }
                $runs.SelectedIndex = $match
            }
            if ($col.model) { $model.Text = $col.model }
        }

        $row = [PSCustomObject]@{ Folder = $folder; Browse = $browse; Runs = $runs; Model = $model; Controls = @($label, $folder, $browse, $runs, $model) }
        $browse.Add_Click({
            param($sender, $e)
            $target = $rows | Where-Object { $_.Browse -eq $sender } | Select-Object -First 1
            $dialog = New-Object System.Windows.Forms.FolderBrowserDialog -Property @{
                Description = "Project folder for this column"; UseDescriptionForTitle = $true; ShowNewFolderButton = $true
                InitialDirectory = $target.Folder.Text
            }
            if ($dialog.ShowDialog($form) -eq "OK") { $target.Folder.Text = $dialog.SelectedPath }
            $dialog.Dispose()
        })
        $runs.Add_SelectedIndexChanged({
            param($sender, $e)
            $target = $rows | Where-Object { $_.Runs -eq $sender } | Select-Object -First 1
            if ($target) { Update-RowState $target }
        })
        $form.Controls.AddRange($row.Controls)
        $rows.Add($row)
        Update-RowState $row
    }
    $bottom = 80 + $n * 38 + 12
    $ok.Left = 740; $ok.Top = $bottom
    $cancel.Left = 866; $cancel.Top = $bottom
    $form.ClientSize = New-Object System.Drawing.Size(980, ($bottom + 48))
}

$count.Add_ValueChanged({ Build-Rows })
Build-Rows

# A preset fills the whole window the way the last layout does (same format).
$presetBox.Add_SelectedIndexChanged({
    if ($presetBox.SelectedIndex -le 0) { return }
    $file = Join-Path $presetDir "$($presetBox.SelectedItem).json"
    try { $p = Get-Content -LiteralPath $file -Raw | ConvertFrom-Json } catch { Show-Message "Preset unreadable: $file"; return }
    $script:last = $p
    if ($where.Enabled) { $where.SelectedIndex = [int]$p.where }
    $nameBox.Text = $p.name
    $script:resetRows = $true
    $n = [Math]::Min(6, [Math]::Max(1, [int]$p.count))
    if ($count.Value -ne $n) { $count.Value = $n } else { Build-Rows }
})

$savePreset.Add_Click({
    $default = if ($presetBox.SelectedIndex -gt 0) { [string]$presetBox.SelectedItem } else { $nameBox.Text.Trim() }
    $form.TopMost = $false
    $n = [Microsoft.VisualBasic.Interaction]::InputBox("Preset name (letters, digits, - and _):", "Save layout preset", $default)
    $form.TopMost = $true
    if (-not $n) { return }
    if ($n -notmatch '^[A-Za-z0-9_-]{1,40}$') { Show-Message "Use letters, digits, - and _ only: $n"; return }
    $hadRunning = $false
    $cols = @($rows | ForEach-Object {
        $opt = $options[$_.Runs.SelectedIndex]
        if ($opt.Type -eq "existing") { $hadRunning = $true }
        @{ folder = $_.Folder.Text; runs = $(if ($opt.Type -eq "new") { $opt.Value } else { "" }); model = $(if ($opt.Type -eq "new") { $_.Model.Text.Trim() } else { "" }) }
    })
    New-Item -ItemType Directory -Force -Path $presetDir | Out-Null
    @{ count = $cols.Count; where = $where.SelectedIndex; name = $nameBox.Text.Trim(); columns = $cols } |
        ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $presetDir "$n.json") -Encoding utf8
    if (-not $presetBox.Items.Contains($n)) { [void]$presetBox.Items.Add($n) }
    $note = if ($hadRunning) { "`nRunning agents are saved as an empty PowerShell in their folder." } else { "" }
    Show-Message "Preset '$n' saved. Next time: pick it here, or run`nherdr-layout.ps1 -Preset $n$note" "Information"
})

$ok.Add_Click({
    $columns = @()
    $seen = @{}
    for ($i = 0; $i -lt $rows.Count; $i++) {
        $r = $rows[$i]
        $opt = $options[$r.Runs.SelectedIndex]
        if ($opt.Type -eq "existing") {
            if ($seen.ContainsKey($opt.Value)) { Show-Message "Column $($i + 1): that running agent is already in column $($seen[$opt.Value])."; return }
            $seen[$opt.Value] = $i + 1
        } elseif (-not (Test-Path -LiteralPath $r.Folder.Text -PathType Container)) {
            Show-Message "Column $($i + 1): folder not found:`n$($r.Folder.Text)"; return
        }
        $columns += [PSCustomObject]@{ Folder = $r.Folder.Text; Type = $opt.Type; Value = $opt.Value; Model = $r.Model.Text.Trim() }
    }
    $script:plan = [PSCustomObject]@{ Where = $where.SelectedIndex; Name = $nameBox.Text.Trim(); Columns = $columns }
    $form.DialogResult = "OK"
    $form.Close()
})

$form.Add_Shown({ $form.Activate() })
if ($form.ShowDialog() -ne "OK" -or -not $plan) { exit 0 }
$form.Dispose()

# Remember the choices (new agents and folders; running agents are not remembered).
@{
    count = $plan.Columns.Count; where = $plan.Where; name = $plan.Name
    columns = @($plan.Columns | ForEach-Object {
        @{ folder = $_.Folder; runs = $(if ($_.Type -eq "new") { $_.Value } else { "" }); model = $_.Model }
    })
} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $lastPath -Encoding utf8
}

# ---- Build it --------------------------------------------------------------------
$cols = $plan.Columns
$label = if ($plan.Name) { $plan.Name } else { "work" }
$first = $cols[0].Folder
if ($plan.Where -eq 0) {
    $made = Invoke-Herdr @("tab", "create", "--workspace", $workspaceId, "--cwd", $first, "--label", $label, "--focus")
} else {
    $made = Invoke-Herdr @("workspace", "create", "--cwd", $first, "--label", $label, "--focus")
}
if (-not $made.Ok) { Show-Message "Herdr could not open the $(if ($plan.Where -eq 0) { 'tab' } else { 'space' }):`n$($made.Text)"; exit 1 }
$tabId = $made.Json.result.tab.tab_id
$panes = @($made.Json.result.root_pane.pane_id)

# Equal columns: Herdr's ratio is the share the original pane keeps (LESSONS #17).
$target = $panes[0]
for ($i = 1; $i -lt $cols.Count; $i++) {
    $remaining = $cols.Count - $i + 1
    $ratio = [string]::Format([cultureinfo]::InvariantCulture, "{0:0.####}", 1.0 / $remaining)
    $split = Invoke-Herdr @("pane", "split", $target, "--direction", "right", "--ratio", $ratio, "--cwd", $cols[$i].Folder, "--no-focus")
    if (-not $split.Ok) { Show-Message "Column $($i + 1) could not be opened:`n$($split.Text)"; exit 1 }
    $target = $split.Json.result.pane.pane_id
    $panes += $target
}

$problems = @()
for ($i = 0; $i -lt $cols.Count; $i++) {
    $col = $cols[$i]
    switch ($col.Type) {
        "new" {
            $startArgs = @("-NoProfile", "-File", $agentsTool, "start", $col.Value, "-Pane", $panes[$i], "-Cwd", $col.Folder)
            if ($col.Model) { $startArgs += @("-Model", $col.Model) }
            $result = (& pwsh @startArgs 2>&1) -join "`n"
            if ($LASTEXITCODE -ne 0) { $problems += "Column $($i + 1) ($($col.Value)): $result" }
        }
        "existing" {
            # Move the running agent next to its placeholder, then close the placeholder:
            # the moved pane takes the placeholder's place and width.
            $moved = Invoke-Herdr @("pane", "move", $col.Value, "--tab", $tabId, "--split", "right", "--target-pane", $panes[$i], "--ratio", "0.5", "--no-focus")
            if (-not $moved.Ok) { $problems += "Column $($i + 1): could not move the running agent: $($moved.Text)"; continue }
            $newId = $moved.Json.result.move_result.pane.pane_id
            [void](Invoke-Herdr @("pane", "close", $panes[$i]))
            # Keep herdr-agents' record of panes it started pointing at the new id.
            if ($newId -and (Test-Path -LiteralPath $panesState)) {
                $raw = Get-Content -LiteralPath $panesState -Raw
                Set-Content -LiteralPath $panesState -Value ($raw.Replace("""$($col.Value)""", """$newId""")) -Encoding utf8
            }
        }
    }
}
[void](Invoke-Herdr @("tab", "focus", $tabId))
if ($problems) { Show-Message ("Layout opened, with problems:`n`n" + ($problems -join "`n`n")) "Warning" }
