# PowerShell-only document pipeline: for a Windows PC with Office and PowerShell but no usable Python.
#   powershell -NoProfile -File docpipe.ps1 -In <folder> -Out <folder> [-Field Amount] [-Reference "Amount=4301.50"]
#                                           [-ExportApproved] [-RecordsDir <folder of .jsonl>]
# Each file is read only by its authorized Office application (com-extract.ps1); nothing parses file bytes.
# Totals use [decimal] (28-29 significant digits; an overflow is an error, never a rounding). Every value
# under a header row becomes a record; counts are reconciled. Without -ExportApproved (company policy:
# reading is not permission to export) only counts, totals and reference checks are written - no values.
# -RecordsDir uses worker output already produced (tests, or a second pass) instead of opening Office.
# Exit 0 = every file read and every reference matches; 1 = a mismatch or an unread file; 2 = usage.
param(
    [string]$In,
    [Parameter(Mandatory)][string]$Out,
    [string[]]$Field = @(),
    [string[]]$Reference = @(),
    [switch]$ExportApproved,
    [string]$RecordsDir
)
$ErrorActionPreference = 'Stop'
if (-not $In -and -not $RecordsDir) { [Console]::Error.WriteLine('need -In or -RecordsDir'); exit 2 }
New-Item -ItemType Directory -Force -Path $Out | Out-Null
$worker = Join-Path $PSScriptRoot 'com-extract.ps1'
$inv = [Globalization.CultureInfo]::InvariantCulture
$files = @(); $unread = @(); $issues = New-Object System.Collections.Generic.List[string]
$totals = @{}; $counts = @{}; $records = New-Object System.Collections.Generic.List[object]
$cellCount = 0; $underHeaders = 0

function Import-Records([string]$name, [string]$jsonl) {
    $heads = @{}; $hrow = @{}
    foreach ($line in [IO.File]::ReadLines($jsonl)) {
        if (-not $line.Trim()) { continue }
        $r = $line | ConvertFrom-Json
        $script:cellCount++
        $sheet = if ($r.PSObject.Properties['sheet'] -and $r.sheet) { [string]$r.sheet } else { $null }
        if (-not $sheet -or -not $r.PSObject.Properties['row']) { continue }
        $isText = $r.kind -ne 'n'
        if (-not $hrow.ContainsKey($sheet) -and $isText) { $hrow[$sheet] = [int]$r.row; $heads[$sheet] = @{} }
        if (-not $hrow.ContainsKey($sheet)) { continue }
        if ([int]$r.row -eq $hrow[$sheet]) {
            $n = ([string]$r.raw -replace '\s+', ' ').Trim(); $base = $n; $k = 1
            while ($heads[$sheet].Values -contains $n) { $k++; $n = "$base#$k" }
            if ($k -gt 1) { $issues.Add("${name} $($r.loc): duplicate header '$base' is field '$n'") }
            $heads[$sheet][[int]$r.col] = $n
            continue
        }
        $script:underHeaders++
        $f = $heads[$sheet][[int]$r.col]
        if (-not $f) { $f = "column $($r.col)"; $heads[$sheet][[int]$r.col] = $f; $issues.Add("${name} $($r.loc): value under a column with no header") }
        $num = $null
        if ($r.kind -eq 'n' -and -not $r.is_date) { $num = [decimal]::Parse(([double]$r.raw).ToString('R', $inv), [Globalization.NumberStyles]::Float, $inv) }
        $cur = if ($r.PSObject.Properties['currency'] -and $r.currency) { [string]$r.currency } else { '-' }
        $records.Add([pscustomobject]@{ file = $name; sheet = $sheet; row = $r.row; field = $f; number = $num; text = $(if ($null -eq $num) { [string]$r.raw } else { $null }); currency = $cur; source = $r.loc })
        if ($null -ne $num) {
            $key = "$f|$cur"
            if (-not $totals.ContainsKey($key)) { $totals[$key] = [decimal]0; $counts[$key] = 0 }
            $totals[$key] = $totals[$key] + $num   # [decimal] throws on overflow: never rounded silently
            $counts[$key]++
        }
    }
}

if ($RecordsDir) {
    foreach ($j in Get-ChildItem -LiteralPath $RecordsDir -Filter *.jsonl -File | Sort-Object Name) { $files += $j.Name; Import-Records $j.Name $j.FullName }
} else {
    foreach ($f in Get-ChildItem -LiteralPath $In -File | Sort-Object Name) {
        $tmp = Join-Path $Out (".work-" + [guid]::NewGuid().ToString('N') + '.jsonl')
        try {
            & powershell -NoProfile -NonInteractive -File $worker -Path $f.FullName -Out $tmp
            if ($LASTEXITCODE -ne 0) { $unread += "$($f.Name) (worker exit $LASTEXITCODE)"; continue }
            $files += $f.Name
            Import-Records $f.Name $tmp
        } finally { Remove-Item -LiteralPath $tmp -ErrorAction SilentlyContinue; Remove-Item -LiteralPath "$tmp.pid" -ErrorAction SilentlyContinue }
    }
}

$checks = @()
foreach ($ref in $Reference) {
    $k, $v = $ref -split '=', 2
    $want = [decimal]::Parse($v.Trim(), $inv)
    $got = if ($totals.ContainsKey("$($k.Trim())|-")) { $totals["$($k.Trim())|-"] } else { [decimal]0 }
    $checks += [pscustomobject]@{ field = $k.Trim(); reference = $want.ToString($inv); computed = $got.ToString($inv); ok = ($want -eq $got) }
}
if ($records.Count -ne $underHeaders) { $issues.Add("reconcile: $underHeaders values under headers but $($records.Count) records") }
$summary = [ordered]@{
    files = $files; unread = $unread; cells = $cellCount; records = $records.Count
    totals = @($totals.Keys | Sort-Object | ForEach-Object { [ordered]@{ field = ($_ -split '\|')[0]; currency = ($_ -split '\|')[1]; total = $totals[$_].ToString($inv); count = $counts[$_] } })
    checks = $checks; issues = @($issues); export = [bool]$ExportApproved
}
[IO.File]::WriteAllText((Join-Path $Out 'summary.json'), (ConvertTo-Json $summary -Depth 5), (New-Object Text.UTF8Encoding($false)))
if ($ExportApproved) {
    # Spreadsheet-safe export: a text starting with = + - @ is prefixed with ' so Excel never runs it.
    $safe = { param($s) if ($s -is [string] -and $s -match '^[=+\-@\t\r]') { "'" + $s } else { $s } }
    $records | ForEach-Object { [pscustomobject]@{ file = $_.file; sheet = $_.sheet; row = $_.row; field = & $safe $_.field
        number = $(if ($null -ne $_.number) { $_.number.ToString($inv) }); text = & $safe $_.text; currency = $_.currency; source = $_.source } } |
        Export-Csv -LiteralPath (Join-Path $Out 'records.csv') -NoTypeInformation -Encoding UTF8
}
$bad = @($checks | Where-Object { -not $_.ok })
Write-Output ("files {0}, unread {1}, records {2}, totals: {3}" -f $files.Count, $unread.Count, $records.Count, (($summary.totals | ForEach-Object { "$($_.field)=$($_.total)" }) -join ', '))
if ($bad.Count -or $unread.Count) { exit 1 }
exit 0
