# herdr-trial: copy the model trial kit, run one profile, grade and append its result.
#   pwsh -NoProfile -File herdr-trial.ps1 -AgentProfile ds-flash [-Model id] [-Effort high]
#     [-Run name] [-TrialRoot path] [-TimeoutMin 60] [-PollSec 15]
#   -Profile remains an alias for -AgentProfile.
# Leaves the agent running so the lead can read REVIEW.md before stopping it.
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [Alias('Profile')] [string]$AgentProfile,
    [string]$Model,
    [string]$Effort,
    [string]$Run,
    [string]$TrialRoot = 'D:\WORK\Software Development\GitHub\HERDR-trials\model-trial',
    [ValidateRange(1, 1440)] [int]$TimeoutMin = 60,
    [ValidateRange(1, 3600)] [int]$PollSec = 15
)
$ErrorActionPreference = "Stop"
$herdr = Join-Path $env:LOCALAPPDATA 'Programs\Herdr\bin\herdr.exe'

function Fail([string]$Message) { Write-Output "ERROR: $Message"; exit 1 }

# Redirect both streams and create no console, including when invoked from a detached pane.
function Invoke-TrialProcess([string]$File, [string[]]$Arguments) {
    $info = [Diagnostics.ProcessStartInfo]::new()
    $info.FileName = $File
    $info.UseShellExecute = $false; $info.CreateNoWindow = $true
    $info.RedirectStandardOutput = $true; $info.RedirectStandardError = $true
    foreach ($argument in $Arguments) { $info.ArgumentList.Add($argument) }
    $process = [Diagnostics.Process]::new(); $process.StartInfo = $info
    try {
        [void]$process.Start()
        $stdout = $process.StandardOutput.ReadToEndAsync(); $stderr = $process.StandardError.ReadToEndAsync()
        $process.WaitForExit()
        [pscustomobject]@{ Code = $process.ExitCode; Text = ($stdout.GetAwaiter().GetResult() + "`n" + $stderr.GetAwaiter().GetResult()).TrimEnd() }
    } finally { $process.Dispose() }
}

function Invoke-Herdr([string[]]$Arguments) {
    $r = Invoke-TrialProcess $herdr $Arguments
    $json = $null
    try { $json = $r.Text | ConvertFrom-Json } catch { }
    [pscustomobject]@{ Ok = ($r.Code -eq 0); Text = $r.Text; Json = $json }
}

function New-TrialRun([string]$TrialRoot, [string]$Run) {
    if ($Run -cnotmatch '^[a-z0-9][a-z0-9._-]{0,48}$') { Fail ('Run name must match ^[a-z0-9][a-z0-9._-]{0,48}$: ' + $Run) }
    $folder = Join-Path $TrialRoot (Join-Path 'runs' $Run)
    if (Test-Path -LiteralPath $folder) { Fail "Run folder exists: $folder" }
    $kit = Join-Path $TrialRoot 'kit'
    if (-not (Test-Path -LiteralPath $kit -PathType Container)) { Fail "Trial kit not found: $kit" }
    [IO.Directory]::CreateDirectory((Join-Path $TrialRoot 'runs')) | Out-Null
    Copy-Item -LiteralPath $kit -Destination $folder -Recurse
    (Resolve-Path -LiteralPath $folder).ProviderPath
}

function Start-TrialAgent([Alias('Profile')] [string]$AgentProfile, [string]$Folder, [string]$Model, [string]$Effort) {
    $live = Invoke-Herdr @('agent', 'list')
    if (-not $live.Ok) { Fail "Could not list agents: $($live.Text)" }
    $names = @($live.Json.result.agents | ForEach-Object { $_.name })
    $n = 1
    while ($names -contains "trial-$n") { $n++ }
    $name = "trial-$n"
    $arguments = @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'herdr-agents.ps1'), 'start', $AgentProfile, '-Name', $name, '-Cwd', $Folder)
    if ($Model) { $arguments += @('-Model', $Model) }
    if ($Effort) { $arguments += @('-Effort', $Effort) }
    $started = Invoke-TrialProcess (Join-Path $PSHOME 'pwsh.exe') $arguments
    if ($started.Code -eq 2) {
        Write-Host $started.Text
        Fail "Agent not ready - the owner answers its question in the pane, then rerun with -Run <existing> is NOT supported: stop it and start again."
    }
    if ($started.Code -ne 0) { Fail "Agent start failed: $($started.Text)" }
    $name
}

function Send-TrialPrompt([string]$AgentName) {
    $prompt = 'Read TASKS.md in your current folder completely and do every task in it. Follow its rules exactly. When finished, reply with the answer format from TASKS.md.'
    $sent = Invoke-Herdr @('agent', 'prompt', $AgentName, $prompt)
    if (-not $sent.Ok) { Fail "Trial prompt failed: $($sent.Text)" }
}

function Wait-TrialReport([string]$Folder, [int]$TimeoutMin, [int]$PollSec) {
    $deadline = (Get-Date).AddMinutes($TimeoutMin)
    $report = Join-Path $Folder 'REPORT.md'
    $previous = $null
    while ($true) {
        if (Test-Path -LiteralPath $report -PathType Leaf) {
            $size = (Get-Item -LiteralPath $report).Length
            if ($null -ne $previous -and $size -eq $previous) { return 'DONE' }
            $previous = $size
        } else { $previous = $null }
        $remaining = ($deadline - (Get-Date)).TotalSeconds
        if ($remaining -le 0) { return 'TIMEOUT' }
        Start-Sleep -Seconds ([Math]::Min($PollSec, $remaining))
    }
}

function Invoke-TrialGrade([string]$TrialRoot, [string]$Folder) {
    $result = Invoke-TrialProcess (Join-Path $PSHOME 'pwsh.exe') @('-NoProfile', '-File', (Join-Path $TrialRoot 'grade.ps1'), '-Run', $Folder)
    if ($result.Code -ne 0) { Fail "Trial grader failed: $($result.Text)" }
    $fields = [ordered]@{}
    foreach ($entry in @(@('ledger', 'LEDGER', '\d+/\d+'), @('columns', 'COLUMNS', '\d+/\d+'), @('scope', 'SCOPE', 'ok|VIOLATED: .+'), @('review', 'REVIEW\.md', 'present|MISSING'), @('report', 'REPORT\.md', 'present|MISSING'))) {
        $match = [regex]::Match($result.Text, '(?m)^' + $entry[1] + ' (' + $entry[2] + ')\r?$')
        if (-not $match.Success) { Fail "Grader did not report $($entry[0]): $($result.Text)" }
        $fields[$entry[0]] = $match.Groups[1].Value.TrimEnd("`r")
    }
    [pscustomobject]$fields
}

function Get-TrialTokens([string]$Kind, [string]$Folder, [string]$UserRoot = $env:USERPROFILE) {
    $empty = [pscustomobject]@{ fresh_in = ''; out = '' }
    if ($Kind -ne 'cline') { return $empty }
    $roots = @((Join-Path $UserRoot '.cline\data\sessions'))
    $accounts = Join-Path $UserRoot '.herdr-accounts\cline'
    if (Test-Path -LiteralPath $accounts) {
        foreach ($account in Get-ChildItem -LiteralPath $accounts -Directory) { $roots += Join-Path $account.FullName 'data\sessions' }
    }
    $files = foreach ($root in $roots) {
        if (-not (Test-Path -LiteralPath $root)) { continue }
        foreach ($session in Get-ChildItem -LiteralPath $root -Directory) {
            $file = Join-Path $session.FullName ($session.Name + '.json')
            if (Test-Path -LiteralPath $file -PathType Leaf) { Get-Item -LiteralPath $file }
        }
    }
    foreach ($file in ($files | Sort-Object LastWriteTimeUtc -Descending)) {
        try {
            $session = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json
            if (-not [string]::Equals(([string]$session.cwd).TrimEnd('\', '/'), $Folder.TrimEnd('\', '/'), [StringComparison]::OrdinalIgnoreCase)) { continue }
            $usage = $session.metadata.usage
            if ($null -eq $usage.inputTokens -or $null -eq $usage.outputTokens) { return $empty }
            return [pscustomobject]@{ fresh_in = ([long]$usage.inputTokens - [long]$usage.cacheReadTokens); out = [long]$usage.outputTokens }
        } catch { continue } # A session can be in the middle of a write.
    }
    $empty
}

function Add-TrialResult([string]$TrialRoot, $Row) {
    $Row | Export-Csv -LiteralPath (Join-Path $TrialRoot 'results.csv') -NoTypeInformation -Append -Encoding utf8
}

function Invoke-Trial([Alias('Profile')] [string]$AgentProfile, [string]$Model, [string]$Effort, [string]$Run, [string]$TrialRoot, [int]$TimeoutMin, [int]$PollSec) {
    $registry = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'agents.json') -Raw | ConvertFrom-Json
    $entry = $registry.agents.PSObject.Properties[$AgentProfile]
    if (-not $entry) { Fail "Unknown profile: $AgentProfile" }
    $spec = $entry.Value
    if (-not $Run) { $Run = "$AgentProfile-$(if ($Effort) { $Effort } else { 'default' })-$(Get-Date -Format 'yyyyMMdd-HHmm')" }
    $folder = New-TrialRun $TrialRoot $Run
    $name = Start-TrialAgent $AgentProfile $folder $Model $Effort
    $began = Get-Date
    Send-TrialPrompt $name
    $outcome = Wait-TrialReport $folder $TimeoutMin $PollSec
    $minutes = ((Get-Date) - $began).TotalMinutes.ToString('F1', [Globalization.CultureInfo]::InvariantCulture)
    $grade = Invoke-TrialGrade $TrialRoot $folder
    $tokens = Get-TrialTokens $spec.kind $folder
    $row = [pscustomobject][ordered]@{
        date = $began.ToString('s'); run = $Run; profile = $AgentProfile
        model = $(if ($Model) { $Model } else { [string]$spec.model })
        effort = $(if ($Effort) { $Effort } else { [string]$spec.effort })
        outcome = $outcome; ledger = $grade.ledger; columns = $grade.columns; scope = $grade.scope
        review = $grade.review; report = $grade.report; minutes = $minutes; fresh_in = $tokens.fresh_in; out = $tokens.out
    }
    Add-TrialResult $TrialRoot $row
    $row | ConvertTo-Csv -NoTypeInformation
    Write-Output "Review REVIEW.md yourself, then: herdr-agents stop $name"
}

Invoke-Trial $AgentProfile $Model $Effort $Run $TrialRoot $TimeoutMin $PollSec
