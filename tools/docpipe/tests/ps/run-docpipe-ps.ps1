# Runs docpipe.ps1 (the PowerShell-only pipeline) on recorded worker output and checks the result.
# It exercises the real PowerShell code path; it does NOT open Office (records come from a fixture).
$ErrorActionPreference = 'Stop'
$here = $PSScriptRoot
$pipe = Join-Path (Split-Path (Split-Path $here)) 'docpipe.ps1'
$fail = 0
function Check($ok, $what) { if ($ok) { "ok   $what" } else { "FAIL $what"; $script:fail++ } }
$out = Join-Path ([IO.Path]::GetTempPath()) ("dp-" + [guid]::NewGuid().ToString('N'))
& $pipe -RecordsDir (Join-Path $here 'records') -Out $out -Reference 'Amount=1350.75','Amount#2=0.3' -ExportApproved
Check ($LASTEXITCODE -eq 0) "exit 0 when references match (got $LASTEXITCODE)"
$s = Get-Content -Raw (Join-Path $out 'summary.json') | ConvertFrom-Json
$t = @{}; foreach ($x in $s.totals) { $t[$x.field] = $x.total }
Check ($t['Amount'] -eq '1350.75') "Amount total exact: $($t['Amount'])"
Check ($t['Amount#2'] -eq '0.3') "0.1 + 0.2 = 0.3 exactly: $($t['Amount#2'])"
Check ($s.records -eq 9) "every value under a header is a record (9): $($s.records)"
Check (@($s.issues | Where-Object { $_ -match 'duplicate header' }).Count -eq 1) 'duplicate header reported'
Check (@($s.issues | Where-Object { $_ -match 'no header' }).Count -eq 1) 'value without header reported'
Check (@($s.issues | Where-Object { $_ -match 'reconcile' }).Count -eq 0) 'records reconcile with values under headers'
$csv = Get-Content -Raw (Join-Path $out 'records.csv')
Check ($csv -match "'=HYPERLINK") 'formula-like text is neutralised in the CSV'
Check ($csv -notmatch ',"=HYPERLINK') 'no raw formula in the CSV'
$out2 = Join-Path ([IO.Path]::GetTempPath()) ("dp-" + [guid]::NewGuid().ToString('N'))
& $pipe -RecordsDir (Join-Path $here 'records') -Out $out2 -Reference 'Amount=1350.74'
Check ($LASTEXITCODE -eq 1) 'a reference mismatch exits 1'
Check (-not (Test-Path (Join-Path $out2 'records.csv'))) 'no value export without -ExportApproved'
Remove-Item -Recurse -Force $out, $out2
"$fail failure(s)"
exit $fail
