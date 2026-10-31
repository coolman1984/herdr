# On-device gates for the owner's authorized Windows PC. Cloud CI cannot run these (no Office, no NASCA,
# no signed-in agent CLIs). Each gate writes evidence to -Out; nothing is uploaded anywhere.
#   powershell -NoProfile -File tools\device-harness\run-device-gates.ps1 -Out C:\herdr-evidence
#       [-ProtectedFile <NASCA file> -ProtectedField Amount -ProtectedTotal 1234.50] [-Profiles docs\goal-profiles.example.json]
# Gates: 1 environment, 2 Office COM round trip (synthetic workbook made by Excel itself, Arabic, dates,
# currency, formula), 3 the owner's open workbook survives a worker run, 4 a killed worker leaves no Office
# process it created and does not touch the owner's, 5 PowerShell-only pipeline with Python hidden,
# 6 NASCA-protected file (only when -ProtectedFile is given), 7 real quota readings per account.
# The device's execution policy applies (run it the way the owner runs approved scripts).
param(
    [Parameter(Mandatory)][string]$Out,
    [string]$ProtectedFile, [string]$ProtectedField = 'Amount', [string]$ProtectedTotal,
    [string]$Profiles
)
$ErrorActionPreference = 'Stop'
$root = Split-Path (Split-Path $PSScriptRoot)
$worker = Join-Path $root 'tools\docpipe\com-extract.ps1'
$pipe = Join-Path $root 'tools\docpipe\docpipe.ps1'
New-Item -ItemType Directory -Force -Path $Out | Out-Null
$results = [ordered]@{ started = (Get-Date).ToString('o'); host = $env:COMPUTERNAME; ps = $PSVersionTable.PSVersion.ToString() }
function Gate([string]$name, [scriptblock]$body) {
    try { $r = & $body; $results[$name] = [ordered]@{ result = 'pass'; detail = $r } ; "PASS $name" }
    catch { $results[$name] = [ordered]@{ result = 'fail'; detail = $_.Exception.Message }; "FAIL $name : $($_.Exception.Message)" }
}
function Assert($cond, $msg) { if (-not $cond) { throw $msg } }

Gate 'g1-environment' {
    $hg = Get-Command herdr-goal -ErrorAction SilentlyContinue
    if ($hg) { & $hg.Source env --project $root | Set-Content (Join-Path $Out 'capabilities.json') -Encoding UTF8; 'herdr-goal env saved' }
    else { 'herdr-goal not on PATH: build it (cd app; go build ./cmd/herdr-goal) to record capabilities' }
}

$book = Join-Path $Out 'synthetic.xlsx'
Gate 'g2-office-com-roundtrip' {
    $xl = New-Object -ComObject Excel.Application
    try {
        $xl.DisplayAlerts = $false
        $wb = $xl.Workbooks.Add(); $ws = $wb.Worksheets.Item(1); $ws.Name = 'مبيعات'
        $ws.Cells.Item(1,1).Value2 = 'الفرع'; $ws.Cells.Item(1,2).Value2 = 'Amount'; $ws.Cells.Item(1,3).Value2 = 'Date'
        $ws.Cells.Item(2,1).Value2 = 'القاهرة'; $ws.Cells.Item(2,2).Value2 = 1250.5; $ws.Cells.Item(2,3).Value2 = 46325; $ws.Cells.Item(2,3).NumberFormat = 'yyyy-mm-dd'
        $ws.Cells.Item(3,1).Value2 = 'الجيزة'; $ws.Cells.Item(3,2).Value2 = 1200.25; $ws.Cells.Item(3,2).NumberFormat = '[$EGP] #,##0.00'
        $ws.Cells.Item(4,1).Value2 = 'المجموع'; $ws.Cells.Item(4,2).Formula = '=SUM(B2:B3)'
        $wb.SaveAs($book, 51); $wb.Close($false)
    } finally { $xl.Quit(); [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($xl) }
    $jl = Join-Path $Out 'synthetic.jsonl'
    & powershell -NoProfile -File $worker -Path $book -Out $jl
    Assert ($LASTEXITCODE -eq 0) "worker exit $LASTEXITCODE"
    $recs = Get-Content $jl | ForEach-Object { $_ | ConvertFrom-Json }
    $b4 = $recs | Where-Object { $_.loc -like '*!B4' }
    Assert ($b4.raw -eq 2450.75 -and $b4.formula) "formula value via COM: $($b4.raw)"
    Assert (($recs | Where-Object { $_.loc -like '*!C2' }).is_date) 'date cell flagged'
    Assert (($recs | Where-Object { $_.loc -like '*!B3' }).currency -match 'EGP') 'currency from number format'
    Assert (-not (Get-Process EXCEL -ErrorAction SilentlyContinue | Where-Object { $_.StartTime -gt (Get-Date).AddMinutes(-1) -and -not $_.MainWindowTitle })) 'no hidden Excel left'
    "$($recs.Count) records; B4=$($b4.raw)"
}

Gate 'g3-owner-workbook-survives' {
    $xl = New-Object -ComObject Excel.Application; $xl.Visible = $true
    $mine = $xl.Workbooks.Add(); $mine.Worksheets.Item(1).Cells.Item(1,1).Value2 = 'owner is typing here'
    try {
        & powershell -NoProfile -File $worker -Path $book -Out (Join-Path $Out 'g3.jsonl')
        Assert ($LASTEXITCODE -eq 0) "worker exit $LASTEXITCODE"
        Assert ($mine.Worksheets.Item(1).Cells.Item(1,1).Value2 -eq 'owner is typing here') "the owner's open workbook changed or closed"
        'owner workbook untouched'
    } finally { $mine.Close($false); $xl.Quit(); [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($xl) }
}

Gate 'g4-killed-worker-cleanup' {
    $big = Join-Path $Out 'big.xlsx'
    $xl = New-Object -ComObject Excel.Application
    try { $wb = $xl.Workbooks.Add(); $wb.Worksheets.Item(1).Range('A1:J200000').Formula = '=ROW()*COLUMN()'; $wb.SaveAs($big, 51); $wb.Close($false) }
    finally { $xl.Quit(); [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($xl) }
    $before = @(Get-Process EXCEL -ErrorAction SilentlyContinue | ForEach-Object Id)
    $jl = Join-Path $Out 'g4.jsonl'
    $p = Start-Process powershell -ArgumentList '-NoProfile','-File',$worker,'-Path',$big,'-Out',$jl -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 8; Stop-Process -Id $p.Id -Force
    Start-Sleep -Seconds 2
    $owned = if (Test-Path "$jl.pid") { [int](Get-Content "$jl.pid") } else { $null }
    if ($owned) { Stop-Process -Id $owned -Force -ErrorAction SilentlyContinue }  # what docpipe does on a timeout
    Start-Sleep -Seconds 2
    $left = @(Get-Process EXCEL -ErrorAction SilentlyContinue | ForEach-Object Id | Where-Object { $before -notcontains $_ })
    Assert ($left.Count -eq 0) "Office processes left after the kill: $($left -join ',')"
    "worker killed; owned pid $owned cleaned; $(@($before).Count) pre-existing Excel process(es) untouched"
}

Gate 'g5-powershell-only' {
    $saved = $env:PATH
    $env:PATH = ($env:PATH -split ';' | Where-Object { $_ -notmatch 'python|Python' }) -join ';'
    try {
        Assert (-not (Get-Command python -ErrorAction SilentlyContinue)) 'python still visible'
        $in = Join-Path $Out 'ps-only-in'; New-Item -ItemType Directory -Force $in | Out-Null; Copy-Item $book $in
        & powershell -NoProfile -File $pipe -In $in -Out (Join-Path $Out 'ps-only-out') -Reference 'Amount=2450.75'
        Assert ($LASTEXITCODE -eq 0) "docpipe.ps1 exit $LASTEXITCODE (Amount must be 1250.5 + 1200.25 = 2450.75; the SUM row is a formula and is not added again)"
        Get-Content (Join-Path $Out 'ps-only-out\summary.json') -Raw
    } finally { $env:PATH = $saved }
}

if ($ProtectedFile) {
    Gate 'g6-nasca-protected-file' {
        $in = Join-Path $Out 'nasca-in'; New-Item -ItemType Directory -Force $in | Out-Null
        Copy-Item -LiteralPath $ProtectedFile $in   # a copy made by Windows, still protected by the DRM client
        $o = Join-Path $Out 'nasca-out'
        $args = @('-NoProfile','-File',$pipe,'-In',$in,'-Out',$o)
        if ($ProtectedTotal) { $args += @('-Reference', "$ProtectedField=$ProtectedTotal") }
        & powershell @args
        $code = $LASTEXITCODE
        Assert (-not (Get-ChildItem $o -Filter '.work-*' -ErrorAction SilentlyContinue)) 'extracted content left in the work folder'
        Assert (-not (Test-Path (Join-Path $o 'records.csv'))) 'values exported without -ExportApproved'
        "exit $code; summary: " + (Get-Content (Join-Path $o 'summary.json') -Raw)
    }
} else { $results['g6-nasca-protected-file'] = @{ result = 'not-run'; detail = 'no -ProtectedFile given: NASCA gate unverified' } }

if ($Profiles) {
    Gate 'g7-quota-readings' {
        $hg = (Get-Command herdr-goal -ErrorAction Stop).Source
        & $hg quota --profiles $Profiles | Tee-Object (Join-Path $Out 'quota.jsonl')
    }
} else { $results['g7-quota-readings'] = @{ result = 'not-run'; detail = 'no -Profiles given' } }

$results['finished'] = (Get-Date).ToString('o')
[IO.File]::WriteAllText((Join-Path $Out 'device-gates.json'), (ConvertTo-Json $results -Depth 6), (New-Object Text.UTF8Encoding($false)))
"evidence: $(Join-Path $Out 'device-gates.json')"
if (@($results.Values | Where-Object { $_ -is [Collections.IDictionary] -and $_.result -eq 'fail' }).Count) { exit 1 }
exit 0
