# Read one Office file through the authorized Office application (COM) and write its values as JSON lines
# (one record per line) with exact locations. For company_nasca devices: the protected file is opened by
# Excel / Word / PowerPoint itself (the DRM client decrypts for the authorized app); this script never touches
# the file's bytes and never saves, exports or copies the document.
#   powershell -NoProfile -File com-extract.ps1 -Path <file> -Out <jsonl> [-MaxCells N]
# The device's execution policy applies: this script is not run with -ExecutionPolicy Bypass. A policy that
# blocks it is a capability fact for the owner, not something to work around.
# Ownership: the Office process this script creates is written to <Out>.pid so the caller can kill exactly
# that process on a timeout. If no new process appeared (Office handed us an instance that was already
# running, e.g. the owner's PowerPoint), the script closes only its own document and never calls Quit().
# Exit 0 = records written; 2 = Office not available; 3 = open refused (protection / permissions / dialog);
# 4 = unsupported type; 5 = over -MaxCells (nothing read). Windows PowerShell 5.1 and PowerShell 7.
param(
    [Parameter(Mandatory)][string]$Path,
    [Parameter(Mandatory)][string]$Out,
    [int]$MaxCells = 2000000
)
$ErrorActionPreference = 'Stop'
$msoAutomationSecurityForceDisable = 3
$ext = [IO.Path]::GetExtension($Path).ToLowerInvariant()
$full = (Resolve-Path -LiteralPath $Path).ProviderPath
$writer = $null
$exit = 0

function Release($o) { if ($null -ne $o) { [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($o) } }

function Write-Record($r) { $writer.WriteLine((ConvertTo-Json -InputObject $r -Compress -Depth 3)) }

# Start the application and find the process it created (none = we were attached to an existing one).
function Start-Office([string]$ProgId, [string]$ProcName) {
    $before = @(Get-Process -Name $ProcName -ErrorAction SilentlyContinue | ForEach-Object { $_.Id })
    try { $app = New-Object -ComObject $ProgId } catch { [Console]::Error.WriteLine("Office not available: $ProgId ($($_.Exception.Message))"); return $null }
    Start-Sleep -Milliseconds 300
    $new = @(Get-Process -Name $ProcName -ErrorAction SilentlyContinue | Where-Object { $before -notcontains $_.Id } | ForEach-Object { $_.Id })
    $owned = $null
    if ($new.Count -eq 1) { $owned = $new[0]; [IO.File]::WriteAllText("$Out.pid", [string]$owned) }
    return @{ App = $app; Owned = $owned }
}

$app = $null; $doc = $null; $owned = $null
try {
    $writer = New-Object IO.StreamWriter($Out, $false, (New-Object Text.UTF8Encoding($false)))
    switch -Regex ($ext) {
        '^\.(xlsx|xlsm|xls|xlsb|csv|tsv|txt|prn)$' {
            $s = Start-Office 'Excel.Application' 'EXCEL'
            if ($null -eq $s) { $exit = 2; break }
            $app = $s.App; $owned = $s.Owned
            $app.Visible = $false; $app.DisplayAlerts = $false; $app.ScreenUpdating = $false
            $app.AutomationSecurity = $msoAutomationSecurityForceDisable
            $app.AskToUpdateLinks = $false
            try {
                # Open(FileName, UpdateLinks=0, ReadOnly=$true, ..., IgnoreReadOnlyRecommended, AddToMru=$false)
                $doc = $app.Workbooks.Open($full, 0, $true, [Type]::Missing, [Type]::Missing, [Type]::Missing, $true, [Type]::Missing, [Type]::Missing, $false, $false, [Type]::Missing, $false)
            } catch { [Console]::Error.WriteLine("Excel refused to open the file: $($_.Exception.Message)"); $exit = 3; break }
            $date1904 = [bool]$doc.Date1904
            $total = 0
            foreach ($ws in $doc.Worksheets) { $total += [double]$ws.UsedRange.Count }
            if ($total -gt $MaxCells) { [Console]::Error.WriteLine("$total cells is over -MaxCells $MaxCells; nothing read"); $exit = 5; break }
            foreach ($ws in $doc.Worksheets) {
                $used = $ws.UsedRange
                $r0 = $used.Row; $c0 = $used.Column
                $rows = $used.Rows.Count; $cols = $used.Columns.Count
                # Read in blocks of 1000 rows so memory stays bounded on big sheets.
                for ($b = 0; $b -lt $rows; $b += 1000) {
                    $n = [Math]::Min(1000, $rows - $b)
                    $block = $ws.Range($ws.Cells.Item($r0 + $b, $c0), $ws.Cells.Item($r0 + $b + $n - 1, $c0 + $cols - 1))
                    $vals = $block.Value2
                    for ($r = 1; $r -le $n; $r++) {
                        $row = $r0 + $b + $r - 1
                        $rowHidden = [bool]$ws.Rows.Item($row).Hidden
                        for ($c = 1; $c -le $cols; $c++) {
                            $v = if ($n -eq 1 -and $cols -eq 1) { $vals } else { $vals[$r, $c] }
                            if ($null -eq $v) { continue }
                            $cell = $ws.Cells.Item($row, $c0 + $c - 1)
                            $isNum = $v -is [double]
                            $fmt = [string]$cell.NumberFormat
                            $cur = $null
                            if ($isNum -and $fmt -match '\[\$([^\]-]+)') { $cur = $Matches[1] }
                            Write-Record ([ordered]@{
                                loc = "$($ws.Name)!$($cell.Address($false, $false))"; sheet = $ws.Name; row = $row; col = $c0 + $c - 1
                                raw = $v; kind = $(if ($isNum) { 'n' } else { 's' })
                                formula = [bool]$cell.HasFormula; date1904 = $date1904; currency = $cur
                                is_date = ($isNum -and ($fmt -match '[dmy]') -and -not ($fmt -match '0\.0|#'))
                                text = [string]$cell.Text; row_hidden = $rowHidden; sheet_hidden = ($ws.Visible -ne -1)
                            })
                            Release $cell
                        }
                    }
                    Release $block
                }
                Release $used; Release $ws
            }
        }
        '^\.(docx|docm|doc|rtf)$' {
            $s = Start-Office 'Word.Application' 'WINWORD'
            if ($null -eq $s) { $exit = 2; break }
            $app = $s.App; $owned = $s.Owned
            $app.Visible = $false; $app.DisplayAlerts = 0
            $app.AutomationSecurity = $msoAutomationSecurityForceDisable
            try { $doc = $app.Documents.Open($full, $false, $true, $false) } catch { [Console]::Error.WriteLine("Word refused to open the file: $($_.Exception.Message)"); $exit = 3; break }
            $i = 0
            foreach ($p in $doc.Paragraphs) {
                $i++
                if ($p.Range.Information(12)) { Release $p; continue } # wdWithInTable: table text comes once, from the table
                $t = ([string]$p.Range.Text).Trim([char]13, [char]7, ' ')
                if ($t) { Write-Record ([ordered]@{ loc = "p$i"; raw = $t; kind = 's'; rtl = ($p.ReadingOrder -eq 0) }) }
                Release $p
            }
            $ti = 0
            foreach ($tb in $doc.Tables) {
                $ti++
                foreach ($cell in $tb.Range.Cells) {
                    $t = ([string]$cell.Range.Text).Trim([char]13, [char]7, ' ')
                    if ($t) { Write-Record ([ordered]@{ loc = "t$($ti)r$($cell.RowIndex)c$($cell.ColumnIndex)"; sheet = "table$ti"; row = $cell.RowIndex; col = $cell.ColumnIndex; raw = $t; kind = 's' }) }
                    Release $cell
                }
                Release $tb
            }
        }
        '^\.(pptx|pptm|ppt)$' {
            $s = Start-Office 'PowerPoint.Application' 'POWERPNT'
            if ($null -eq $s) { $exit = 2; break }
            $app = $s.App; $owned = $s.Owned
            $app.AutomationSecurity = $msoAutomationSecurityForceDisable
            # Open(FileName, ReadOnly, Untitled, WithWindow=msoFalse)
            try { $doc = $app.Presentations.Open($full, -1, 0, 0) } catch { [Console]::Error.WriteLine("PowerPoint refused to open the file: $($_.Exception.Message)"); $exit = 3; break }
            foreach ($sl in $doc.Slides) {
                foreach ($sh in $sl.Shapes) {
                    if ($sh.HasTable) {
                        $tb = $sh.Table
                        for ($r = 1; $r -le $tb.Rows.Count; $r++) { for ($c = 1; $c -le $tb.Columns.Count; $c++) {
                            $t = [string]$tb.Cell($r, $c).Shape.TextFrame.TextRange.Text
                            if ($t.Trim()) { Write-Record ([ordered]@{ loc = "s$($sl.SlideIndex):$($sh.Name):r$($r)c$c"; sheet = "s$($sl.SlideIndex):$($sh.Name)"; row = $r; col = $c; raw = $t.Trim(); kind = 's' }) }
                        } }
                    } elseif ($sh.HasTextFrame -and $sh.TextFrame.HasText) {
                        $pi = 0
                        foreach ($para in $sh.TextFrame.TextRange.Paragraphs()) {
                            $pi++
                            $t = ([string]$para.Text).Trim()
                            if ($t) { Write-Record ([ordered]@{ loc = "s$($sl.SlideIndex):$($sh.Name):p$pi"; raw = $t; kind = 's' }) }
                        }
                    }
                    Release $sh
                }
                Release $sl
            }
        }
        default { [Console]::Error.WriteLine("unsupported type $ext"); $exit = 4 }
    }
} finally {
    if ($writer) { $writer.Dispose() }
    # Close only what this script opened, without saving. Quit only an instance this script created.
    if ($doc) { try { if ($ext -match 'xls|csv|tsv|txt|prn') { $doc.Close($false) } elseif ($ext -match 'ppt') { $doc.Close() } else { $doc.Close(0) } } catch { } ; Release $doc }
    if ($app) { if ($owned) { try { $app.Quit() } catch { } } ; Release $app }
    [GC]::Collect(); [GC]::WaitForPendingFinalizers()
    if ($exit -ne 0) { Remove-Item -LiteralPath $Out -ErrorAction SilentlyContinue }
}
exit $exit
