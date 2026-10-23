# Spaces side by side: show several spaces at once as equal columns in one space, and
# separate them again. Panes are MOVED (agents keep running, nothing restarts).
# Bound to ctrl+b shift+s.
#
#   pwsh -STA -File herdr-spaces.ps1                      window: tick spaces, Side by side / Separate again
#   pwsh -File herdr-spaces.ps1 -Spaces "MR.Park,wG"      no window (labels or workspace ids, in order)
#   pwsh -File herdr-spaces.ps1 -Separate                 no window: undo the last side-by-side
#
# A new space "side by side" gets N equal columns; each chosen space's active-tab panes
# move into its column (first pane = the column, more panes stacked inside it), and the
# emptied spaces close (LESSONS #18). The record in %APPDATA%\herdr\side-by-side.json lets
# "Separate" move each group back into a new space with its old name.
[CmdletBinding()]
param([string]$Spaces, [switch]$Separate)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"
$recordPath = Join-Path $env:APPDATA "herdr\side-by-side.json"
$panesState = Join-Path $env:APPDATA "herdr\herdr-agents-panes.json"
$noWindow = [bool]($Spaces -or $Separate)

function Show-Message([string]$Text, [string]$Icon = "Error") {
    if ($noWindow) { Write-Output "$($Icon.ToUpper()): $Text"; return }
    [System.Windows.Forms.MessageBox]::Show($Text, "Herdr - spaces side by side", "OK", $Icon) | Out-Null
}

function Invoke-Herdr([string[]]$Arguments) {
    $out = & $herdr @Arguments 2>&1
    $text = ($out | ForEach-Object { $_.ToString() }) -join "`n"
    $json = $null
    try { $json = $text | ConvertFrom-Json } catch { }
    [PSCustomObject]@{ Ok = ($LASTEXITCODE -eq 0); Text = $text; Json = $json }
}

function Format-Ratio([double]$Value) { [string]::Format([cultureinfo]::InvariantCulture, "{0:0.####}", $Value) }

# Keep herdr-agents' record of panes it started pointing at a moved pane's new id.
function Update-PaneRecord([string]$Old, [string]$New) {
    if ($New -and (Test-Path -LiteralPath $panesState)) {
        $raw = Get-Content -LiteralPath $panesState -Raw
        if ($raw.Contains("""$Old""")) { Set-Content -LiteralPath $panesState -Value ($raw.Replace("""$Old""", """$New""")) -Encoding utf8 }
    }
}

# Panes of a workspace's active tab, left-to-right / top-to-bottom.
function Get-TabPanes($Workspace) {
    $list = Invoke-Herdr @("pane", "list", "--workspace", $Workspace.workspace_id)
    if (-not $list.Ok) { return @() }
    $panes = @($list.Json.result.panes | Where-Object { $_.tab_id -eq $Workspace.active_tab_id })
    if (-not $panes) { return @() }
    $layout = Invoke-Herdr @("pane", "layout", "--pane", $panes[0].pane_id)
    $rects = @{}
    if ($layout.Ok) { foreach ($p in $layout.Json.result.layout.panes) { $rects[$p.pane_id] = $p.rect } }
    # The pane's own name (ctrl+b shift+p) is "label" in pane list; absent when it has none.
    foreach ($p in $panes) { if (-not $p.PSObject.Properties["label"]) { $p | Add-Member -NotePropertyName label -NotePropertyValue $null } }
    @($panes | Sort-Object { if ($rects[$_.pane_id]) { $rects[$_.pane_id].x } else { 0 } }, { if ($rects[$_.pane_id]) { $rects[$_.pane_id].y } else { 0 } })
}

function Join-Spaces($Chosen) {
    $n = $Chosen.Count
    $sources = @($Chosen | ForEach-Object { [PSCustomObject]@{ Space = $_; Panes = @(Get-TabPanes $_) } })
    $empty = @($sources | Where-Object { -not $_.Panes })
    if ($empty) { Show-Message "Could not read the panes of: $(($empty | ForEach-Object { $_.Space.label }) -join ', ')"; return }

    # 1. A new space with N equal placeholder columns (ratios 1/N .. 1/2, LESSONS #17).
    $made = Invoke-Herdr @("workspace", "create", "--cwd", $sources[0].Panes[0].cwd, "--label", "side by side", "--no-focus")
    if (-not $made.Ok) { Show-Message "Herdr could not open the side-by-side space:`n$($made.Text)"; return }
    $wsId = $made.Json.result.workspace.workspace_id
    $tabId = $made.Json.result.tab.tab_id
    $holders = @($made.Json.result.root_pane.pane_id)
    for ($i = 1; $i -lt $n; $i++) {
        $split = Invoke-Herdr @("pane", "split", $holders[-1], "--direction", "right", "--ratio", (Format-Ratio (1.0 / ($n - $i + 1))), "--cwd", $sources[$i].Panes[0].cwd, "--no-focus")
        if (-not $split.Ok) { Show-Message "Column $($i + 1) could not be opened:`n$($split.Text)"; return }
        $holders += $split.Json.result.pane.pane_id
    }

    # 2. Each space's first pane takes its placeholder's place (move next to it, close
    #    the placeholder - LESSONS #18). Its space closes when its last pane leaves.
    $problems = @()
    $groups = @()
    $columns = @{}
    for ($i = 0; $i -lt $n; $i++) {
        $src = $sources[$i]
        $moved = Invoke-Herdr @("pane", "move", $src.Panes[0].pane_id, "--tab", $tabId, "--split", "right", "--target-pane", $holders[$i], "--ratio", "0.5", "--no-focus")
        if (-not $moved.Ok) { $problems += "$($src.Space.label): $($moved.Text)"; continue }
        $col = $moved.Json.result.move_result.pane.pane_id
        [void](Invoke-Herdr @("pane", "close", $holders[$i]))
        Update-PaneRecord $src.Panes[0].pane_id $col
        $columns[$i] = $col
    }

    # Every moved pane keeps a name: its own, else the space it came from - so the sidebar
    # and its border do not all say "side by side". Rename any time with ctrl+b shift+p.
    function Set-PaneName([string]$Id, $Pane, [string]$Space) {
        if (-not $Pane.label) { [void](Invoke-Herdr @("pane", "rename", $Id, $Space)) }
    }
    for ($i = 0; $i -lt $n; $i++) {
        if ($columns.ContainsKey($i)) { Set-PaneName $columns[$i] $sources[$i].Panes[0] $sources[$i].Space.label }
    }

    # 3. Extra panes of a space are stacked inside ITS column only.
    for ($i = 0; $i -lt $n; $i++) {
        if (-not $columns.ContainsKey($i)) { continue }
        $src = $sources[$i]
        $ids = @($columns[$i])
        $below = $columns[$i]
        for ($k = 1; $k -lt $src.Panes.Count; $k++) {
            $m = Invoke-Herdr @("pane", "move", $src.Panes[$k].pane_id, "--tab", $tabId, "--split", "down", "--target-pane", $below, "--ratio", (Format-Ratio (1.0 / ($src.Panes.Count - $k + 1))), "--no-focus")
            if (-not $m.Ok) { $problems += "$($src.Space.label) pane $($k + 1): $($m.Text)"; continue }
            $below = $m.Json.result.move_result.pane.pane_id
            Update-PaneRecord $src.Panes[$k].pane_id $below
            Set-PaneName $below $src.Panes[$k] $src.Space.label
            $ids += $below
        }
        $groups += [PSCustomObject]@{ label = $src.Space.label; panes = $ids }
    }

    ConvertTo-Json -InputObject ([PSCustomObject]@{ workspace = $wsId; tab = $tabId; groups = $groups; when = (Get-Date).ToString("s") }) -Depth 5 |
        Set-Content -LiteralPath $recordPath -Encoding utf8
    [void](Invoke-Herdr @("workspace", "focus", $wsId))
    if ($problems) { Show-Message ("Side by side opened, with problems:`n`n" + ($problems -join "`n")) "Warning" }
    elseif ($noWindow) { Write-Output "SIDE BY SIDE: $(($Chosen | ForEach-Object { $_.label }) -join ' | ')" }
}

function Split-Spaces {
    if (-not (Test-Path -LiteralPath $recordPath)) { Show-Message "Nothing to separate: no side-by-side view was opened." "Information"; return }
    $rec = Get-Content -LiteralPath $recordPath -Raw | ConvertFrom-Json
    $alive = @{}
    $all = Invoke-Herdr @("pane", "list")
    if (-not $all.Ok) { Show-Message "Could not read panes:`n$($all.Text)"; return }
    foreach ($p in $all.Json.result.panes) { $alive[$p.pane_id] = $true }
    $problems = @()
    $remaining = @()
    foreach ($g in @($rec.groups)) {
        $ids = @($g.panes | Where-Object { $alive[$_] })
        if (-not $ids) { $problems += "$($g.label): its panes are gone (closed or moved)"; $remaining += $g; continue }
        # A partial retry reuses its destination and skips panes already moved there.
        $separated = @()
        if ($g.PSObject.Properties["separated"]) { $separated = @($g.separated | Where-Object { $alive[$_] }) }
        if (-not $separated) {
            $first = Invoke-Herdr @("pane", "move", $ids[0], "--new-workspace", "--label", $g.label, "--no-focus")
            if (-not $first.Ok) { $problems += "$($g.label): $($first.Text)"; $remaining += $g; continue }
            $newFirst = $first.Json.result.move_result.pane.pane_id
            Update-PaneRecord $ids[0] $newFirst
            $ids[0] = $newFirst
            $separated = @($newFirst)
        }
        $lookup = Invoke-Herdr @("pane", "get", $separated[0])
        if (-not $lookup.Ok) {
            $problems += "$($g.label): $($lookup.Text)"
            $remaining += [pscustomobject]@{ label = $g.label; panes = $ids; separated = $separated }
            continue
        }
        $info = $lookup.Json.result.pane
        $below = $separated[-1]
        $failed = $false
        for ($k = 0; $k -lt $ids.Count; $k++) {
            if ($ids[$k] -in $separated) { continue }
            $m = Invoke-Herdr @("pane", "move", $ids[$k], "--tab", $info.tab_id, "--split", "down", "--target-pane", $below, "--ratio", "0.5", "--no-focus")
            if (-not $m.Ok) { $problems += "$($g.label) pane $($k + 1): $($m.Text)"; $failed = $true; continue }
            $below = $m.Json.result.move_result.pane.pane_id
            Update-PaneRecord $ids[$k] $below
            $ids[$k] = $below
            $separated += $below
        }
        if ($failed) {
            $remaining += [pscustomobject]@{ label = $g.label; panes = $ids; separated = $separated }
        }
    }
    if ($remaining.Count) {
        $rec.groups = @($remaining)
        ConvertTo-Json -InputObject $rec -Depth 5 | Set-Content -LiteralPath $recordPath -Encoding utf8
    } else { Remove-Item -LiteralPath $recordPath }
    if ($problems) { Show-Message ("Separated, with problems:`n`n" + ($problems -join "`n")) "Warning" }
    elseif ($noWindow) { Write-Output "SEPARATED: $((@($rec.groups) | ForEach-Object { $_.label }) -join ', ') back in their own spaces" }
}

if (-not (Test-Path -LiteralPath $herdr)) { Show-Message "Herdr is not installed at $herdr."; exit 1 }

if ($Separate) { Split-Spaces; exit 0 }

$wsList = Invoke-Herdr @("workspace", "list")
if (-not $wsList.Ok) { Show-Message "Herdr is not answering:`n$($wsList.Text)"; exit 1 }
$workspaces = @($wsList.Json.result.workspaces | Sort-Object number)

if ($Spaces) {
    $chosen = New-Object System.Collections.Generic.List[object]
    foreach ($s in ($Spaces -split ',' | ForEach-Object { $_.Trim() } | Where-Object { $_ })) {
        $w = $workspaces | Where-Object { $_.workspace_id -eq $s -or $_.label -eq $s } | Select-Object -First 1
        if (-not $w) { Show-Message "No space '$s'. Spaces: $(($workspaces | ForEach-Object { $_.label }) -join ', ')"; exit 1 }
        if (-not ($chosen | Where-Object { $_.workspace_id -eq $w.workspace_id })) { $chosen.Add($w) }
    }
    if ($chosen.Count -lt 2) { Show-Message "Choose two or more spaces."; exit 1 }
    Join-Spaces $chosen
    exit 0
}

# ---- The window ------------------------------------------------------------------
$form = New-Object System.Windows.Forms.Form -Property @{
    Text = "Herdr - spaces side by side"; StartPosition = "CenterScreen"; TopMost = $true
    FormBorderStyle = "FixedDialog"; MaximizeBox = $false; MinimizeBox = $false
    Font = New-Object System.Drawing.Font("Segoe UI", 9.5); AutoScaleMode = "Dpi"
    ClientSize = New-Object System.Drawing.Size(520, 360)
}
$hint = New-Object System.Windows.Forms.Label -Property @{
    Left = 16; Top = 12; Width = 490; Height = 40; ForeColor = [System.Drawing.Color]::DimGray
    Text = "Tick the spaces to see side by side (top = left column). Their panes move into one new space; agents keep running."
}
$box = New-Object System.Windows.Forms.CheckedListBox -Property @{ Left = 16; Top = 56; Width = 488; Height = 230; CheckOnClick = $true }
foreach ($w in $workspaces) {
    [void]$box.Items.Add(("{0}   ({1}, {2} pane{3})" -f $w.label, $w.agent_status, $w.pane_count, $(if ($w.pane_count -ne 1) { "s" } else { "" })), $true)
}
$join = New-Object System.Windows.Forms.Button -Property @{ Text = "Side by side"; Left = 16; Top = 304; Width = 130; Height = 34 }
$split = New-Object System.Windows.Forms.Button -Property @{ Text = "Separate again"; Left = 156; Top = 304; Width = 140; Height = 34; Enabled = (Test-Path -LiteralPath $recordPath) }
$cancel = New-Object System.Windows.Forms.Button -Property @{ Text = "Cancel"; Left = 404; Top = 304; Width = 100; Height = 34; DialogResult = "Cancel" }
$form.Controls.AddRange(@($hint, $box, $join, $split, $cancel))
$form.CancelButton = $cancel
$form.AcceptButton = $join

$script:action = $null
$join.Add_Click({
    if ($box.CheckedIndices.Count -lt 2) { Show-Message "Tick two or more spaces." "Information"; return }
    $script:action = "join"; $form.DialogResult = "OK"; $form.Close()
})
$split.Add_Click({ $script:action = "split"; $form.DialogResult = "OK"; $form.Close() })
$form.Add_Shown({ $form.Activate() })
if ($form.ShowDialog() -ne "OK") { exit 0 }
$picked = @($box.CheckedIndices | ForEach-Object { $workspaces[$_] })
$form.Dispose()

if ($script:action -eq "split") { Split-Spaces } else { Join-Spaces $picked }
