# Read one Office file through the authorized Office application (COM) and write its values as JSON
# records with exact locations. For company_nasca devices: the protected file is opened by Excel / Word /
# PowerPoint itself (the DRM client decrypts for the authorized app); this script never touches the
# file's bytes and never saves, exports or copies the document.
#   powershell -NoProfile -ExecutionPolicy Bypass -File com-extract.ps1 -Path <file> -Out <json> [-MaxCells N]
# Runs in its own process; the caller (docpipe) kills it on timeout. It creates its OWN Office instance and
# closes only that one - never the owner's open windows. Macros are forced off; links are not updated.
# Exit 0 = JSON written; 2 = Office not available; 3 = open refused (protection / permissions / dialog);
# 4 = unsupported type. Works with Windows PowerShell 5.1 and PowerShell 7.
param(
    [Parameter(Mandatory)][string]$Path,
    [Parameter(Mandatory)][string]$Out,
    [int]$MaxCells = 5000000
)
$ErrorActionPreference = 'Stop'
$msoAutomationSecurityForceDisable = 3
$ext = [IO.Path]::GetExtension($Path).ToLowerInvariant()
$full = (Resolve-Path -LiteralPath $Path).ProviderPath
$records = New-Object System.Collections.Generic.List[object]

function Release($o) { if ($null -ne $o) { [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($o) } }

function New-Office([string]$ProgId) {
    try { New-Object -ComObject $ProgId } catch { Write-Error "Office not available: $ProgId ($($_.Exception.Message))"; exit 2 }
}

$app = $null; $doc = $null
try {
    switch -Regex ($ext) {
        '^\.(xlsx|xlsm|xls|xlsb|csv)$' {
            $app = New-Office 'Excel.Application'
            $app.Visible = $false; $app.DisplayAlerts = $false; $app.ScreenUpdating = $false
            $app.AutomationSecurity = $msoAutomationSecurityForceDisable
            $app.AskToUpdateLinks = $false
            try {
                # Open(FileName, UpdateLinks=0, ReadOnly=$true, ..., IgnoreReadOnlyRecommended, AddToMru=$false)
                $doc = $app.Workbooks.Open($full, 0, $true, [Type]::Missing, [Type]::Missing, [Type]::Missing, $true, [Type]::Missing, [Type]::Missing, $false, $false, [Type]::Missing, $false)
            } catch { Write-Error "Excel refused to open the file: $($_.Exception.Message)"; exit 3 }
            $count = 0
            foreach ($ws in $doc.Worksheets) {
                $used = $ws.UsedRange
                $vals = $used.Value2            # values as Excel holds them (computed by Excel itself)
                $r0 = $used.Row; $c0 = $used.Column
                $rows = $used.Rows.Count; $cols = $used.Columns.Count
                for ($r = 1; $r -le $rows; $r++) {
                    $rowHidden = [bool]$ws.Rows.Item($r0 + $r - 1).Hidden
                    for ($c = 1; $c -le $cols; $c++) {
                        $v = if ($rows -eq 1 -and $cols -eq 1) { $vals } else { $vals[$r, $c] }
                        if ($null -eq $v) { continue }
                        $cell = $ws.Cells.Item($r0 + $r - 1, $c0 + $c - 1)
                        $addr = $cell.Address($false, $false)
                        $isNum = $v -is [double]
                        $records.Add([ordered]@{
                            loc = "$($ws.Name)!$addr"; sheet = $ws.Name; row = $r0 + $r - 1; col = $c0 + $c - 1
                            raw = $v; kind = $(if ($isNum) { 'n' } else { 's' })
                            formula = [bool]$cell.HasFormula
                            is_date = ($isNum -and ([string]$cell.NumberFormat -match '[dmy]') -and -not ([string]$cell.NumberFormat -match '0\.0'))
                            text = [string]$cell.Text; row_hidden = $rowHidden; sheet_hidden = ($ws.Visible -ne -1)
                        })
                        Release $cell
                        if (++$count -ge $MaxCells) { throw "more than $MaxCells cells: stopped (raise -MaxCells)" }
                    }
                }
                Release $used; Release $ws
            }
        }
        '^\.(docx|docm|doc|rtf)$' {
            $app = New-Office 'Word.Application'
            $app.Visible = $false; $app.DisplayAlerts = 0
            $app.AutomationSecurity = $msoAutomationSecurityForceDisable
            try { $doc = $app.Documents.Open($full, $false, $true, $false) } catch { Write-Error "Word refused to open the file: $($_.Exception.Message)"; exit 3 }
            $i = 0
            foreach ($p in $doc.Paragraphs) {
                $i++
                $t = ([string]$p.Range.Text).Trim([char]13, [char]7, ' ')
                if ($t) { $records.Add([ordered]@{ loc = "p$i"; raw = $t; kind = 's'; rtl = ($p.ReadingOrder -eq 0) }) }
                Release $p
            }
            $ti = 0
            foreach ($tb in $doc.Tables) {
                $ti++
                foreach ($cell in $tb.Range.Cells) {
                    $t = ([string]$cell.Range.Text).Trim([char]13, [char]7, ' ')
                    if ($t) { $records.Add([ordered]@{ loc = "t$($ti)r$($cell.RowIndex)c$($cell.ColumnIndex)"; raw = $t; kind = 's' }) }
                    Release $cell
                }
                Release $tb
            }
        }
        '^\.(pptx|pptm|ppt)$' {
            $app = New-Office 'PowerPoint.Application'
            $app.AutomationSecurity = $msoAutomationSecurityForceDisable
            # Open(FileName, ReadOnly, Untitled, WithWindow=msoFalse)
            try { $doc = $app.Presentations.Open($full, -1, 0, 0) } catch { Write-Error "PowerPoint refused to open the file: $($_.Exception.Message)"; exit 3 }
            foreach ($s in $doc.Slides) {
                foreach ($sh in $s.Shapes) {
                    if ($sh.HasTable) {
                        $tb = $sh.Table
                        for ($r = 1; $r -le $tb.Rows.Count; $r++) { for ($c = 1; $c -le $tb.Columns.Count; $c++) {
                            $t = [string]$tb.Cell($r, $c).Shape.TextFrame.TextRange.Text
                            if ($t.Trim()) { $records.Add([ordered]@{ loc = "s$($s.SlideIndex):$($sh.Name):r$($r)c$c"; raw = $t.Trim(); kind = 's' }) }
                        } }
                    } elseif ($sh.HasTextFrame -and $sh.TextFrame.HasText) {
                        $pi = 0
                        foreach ($para in $sh.TextFrame.TextRange.Paragraphs()) {
                            $pi++
                            $t = ([string]$para.Text).Trim()
                            if ($t) { $records.Add([ordered]@{ loc = "s$($s.SlideIndex):$($sh.Name):p$pi"; raw = $t; kind = 's' }) }
                        }
                    }
                    Release $sh
                }
                Release $s
            }
        }
        default { Write-Error "unsupported type $ext"; exit 4 }
    }
    $json = ConvertTo-Json -InputObject @($records) -Depth 4 -Compress
    [IO.File]::WriteAllText($Out, $json, (New-Object Text.UTF8Encoding($false)))
    exit 0
} finally {
    # Close only what this script opened, without saving.
    if ($doc) { try { if ($ext -match 'xls|csv') { $doc.Close($false) } elseif ($ext -match 'ppt') { $doc.Close() } else { $doc.Close(0) } } catch { } ; Release $doc }
    if ($app) { try { $app.Quit() } catch { } ; Release $app }
    [GC]::Collect(); [GC]::WaitForPendingFinalizers()
}
