# herdr-headless: run a Cline agent profile headless (no pane), wait, save the log, report like "ask".
#
#   pwsh -NoProfile -File herdr-headless.ps1 -Profile ds-flash -Task "do the thing"
#   pwsh -NoProfile -File herdr-headless.ps1 -Profile kimi -TaskFile tasks\my-task.md
#     [-Cwd folder] [-Model id] [-Effort high] [-Account name] [-TimeoutSec 1800]
#   -Profile is an alias for -AgentProfile.
#
# Cline models get no pane (CLAUDE.md section 0): the lead drives them headless. Cline then
# reports exactly when it is done (LESSONS #47): the last JSON Lines record is "run_result",
# the record before it the answer - no pane, no idle/done guessing (LESSONS #41). The full JSON
# Lines log is kept under %APPDATA%\herdr\agent-runs next to the pane runs.
# Profiles come from agents.json and must have kind = cline (cline, cline-flash, ds-flash,
# deepseek, kimi). The prompt is the last argument; for cline "-p" means PLAN mode (LESSONS #47).
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [Alias('Profile')] [string]$AgentProfile,
    [string]$Task,
    [string]$TaskFile,
    [string]$Cwd,
    [string]$Model,
    [string]$Effort,
    [string]$Account,
    [ValidateRange(1, 86400)] [int]$TimeoutSec = 1800
)
$ErrorActionPreference = "Stop"
$registryPath = Join-Path $PSScriptRoot "agents.json"
$runsPath = Join-Path $env:APPDATA "herdr\agent-runs"
$accountsRoot = Join-Path $env:USERPROFILE ".herdr-accounts"
# Cline's npm shim is cline.ps1 on Windows and a .ps1 needs PowerShell 7 to run.
$pwshPath = (Get-Command pwsh -ErrorAction SilentlyContinue | Select-Object -First 1).Source
if (-not $pwshPath) {
    $candidate = Join-Path $PSHOME "pwsh.exe"
    if (Test-Path -LiteralPath $candidate) { $pwshPath = $candidate }
}

function Fail([string]$Message) {
    Write-Output "ERROR: $Message"
    exit 1
}

# Only Cline profiles run headless; everything else needs a pane.
function Get-HeadlessProfile($Registry, [string]$Name) {
    $entry = $Registry.agents.PSObject.Properties[$Name]
    if (-not $entry) { Fail "Unknown profile: $Name (see: herdr-agents tools)" }
    $spec = $entry.Value
    if ($spec.kind -ne "cline") { Fail "herdr-headless runs Cline profiles only: $Name" }
    $spec
}

# -TaskFile is sent as a prompt naming its full path, so the agent opens the file itself.
function Get-HeadlessPrompt([string]$Task, [string]$TaskFile) {
    if ($TaskFile) {
        if (-not (Test-Path -LiteralPath $TaskFile -PathType Leaf)) { Fail "Task file not found: $TaskFile" }
        $full = (Resolve-Path -LiteralPath $TaskFile).ProviderPath
        return "Read $full completely and do exactly what it says. Reply in its answer format."
    }
    $Task
}

# The CLI arguments for one headless run, in the order Cline expects them:
# profile args without --tui/-i, provider, model, effort, --json, folder, timeout, account.
# The prompt is added last by the caller.
function Get-HeadlessArgs($Spec, [string]$Model, [string]$Effort, [string]$Cwd, [int]$TimeoutSec, [string[]]$AccountArgs) {
    $out = New-Object System.Collections.Generic.List[string]
    foreach ($argument in @($Spec.args)) {
        if (-not $argument) { continue }
        # --tui/-i open Cline's own interface; headless has no pane.
        if ($argument -in @("--tui", "-i")) { continue }
        $out.Add([string]$argument)
    }
    if ($Spec.PSObject.Properties["providerFlag"] -and $Spec.providerFlag -and $Spec.provider) {
        $out.Add([string]$Spec.providerFlag); $out.Add([string]$Spec.provider)
    }
    $chosen = if ($Model) { $Model } elseif ($Spec.model) { $Spec.model } else { "" }
    if ($chosen -and $Spec.modelFlag) { $out.Add([string]$Spec.modelFlag); $out.Add([string]$chosen) }
    $level = if ($Effort) { $Effort } else { [string]$Spec.effort }
    if ($level -and $Spec.PSObject.Properties["effortArgs"]) {
        foreach ($argument in @($Spec.effortArgs)) { $out.Add(([string]$argument).Replace("{0}", $level)) }
    }
    $out.Add("--json"); $out.Add("-c"); $out.Add($Cwd); $out.Add("-t"); $out.Add($TimeoutSec.ToString())
    foreach ($argument in @($AccountArgs)) { if ($argument) { $out.Add([string]$argument) } }
    $out.ToArray()
}

# Separate accounts (herdr-agents "Get-Account"): the profile's account block points the CLI at
# its own sign-in folder %USERPROFILE%\.herdr-accounts\<tool>\<name>, {dir} = that folder.
# Headless never creates the folder or copies settings - a missing folder means nobody signed
# in there, and the owner does that in the CLI itself (CLAUDE.md safety rule 1).
function Get-HeadlessAccount($Entry, [string]$AccountName) {
    if (-not $Entry.PSObject.Properties["account"] -or -not $Entry.account) { Fail "This agent has no separate-account support (agents.json 'account')." }
    if ($AccountName -notmatch '^[a-z0-9][a-z0-9_-]{0,23}$') { Fail "Account name must match [a-z0-9][a-z0-9_-]{0,23}: $AccountName" }
    $spec = $Entry.account
    $dir = Join-Path $accountsRoot (Join-Path $spec.tool $AccountName)
    if (-not (Test-Path -LiteralPath $dir)) {
        Fail "Account folder not found: $dir (the owner signs in first: $(([string]$spec.login).Replace('{dir}', $dir)))"
    }
    $envs = [ordered]@{}
    foreach ($p in @($spec.env.PSObject.Properties)) { $envs[$p.Name] = ([string]$p.Value).Replace("{dir}", $dir) }
    [PSCustomObject]@{
        Name = $AccountName; Tool = $spec.tool; Dir = $dir; Env = $envs
        Args = @($spec.args | Where-Object { $_ } | ForEach-Object { $_.Replace("{dir}", $dir) })
    }
}

# The profile's command, resolved like herdr-agents does. On Windows an npm CLI is a .ps1 shim
# (cline.ps1): start it with pwsh -NoProfile -File so no console window and no profile runs.
function Get-HeadlessCommand([string]$Command) {
    $found = @(Get-Command $Command -ErrorAction SilentlyContinue) | Select-Object -First 1
    if (-not $found) { Fail "$Command is not installed." }
    $source = $found.Source
    if ($source -like "*.ps1") {
        if (-not $pwshPath -or -not (Test-Path -LiteralPath $pwshPath)) { Fail "PowerShell 7 (pwsh) is needed to run $source but was not found." }
        return [PSCustomObject]@{ File = $pwshPath; Prefix = @("-NoProfile", "-File", $source) }
    }
    [PSCustomObject]@{ File = $source; Prefix = @() }
}

# The run log: %APPDATA%\herdr\agent-runs\<date>\<time>-headless-<profile>.jsonl, never
# overwriting an earlier run (the same rule as Save-Run: -2, -3, ...).
function New-HeadlessLogPath([string]$ProfileName, [datetime]$Now) {
    $day = Join-Path $runsPath $Now.ToString("yyyy-MM-dd")
    New-Item -ItemType Directory -Force -Path $day | Out-Null
    $stem = "{0}-headless-{1}" -f $Now.ToString("HHmmss"), $ProfileName
    $n = 1
    while ($true) {
        $suffix = if ($n -eq 1) { "" } else { "-$n" }
        $file = Join-Path $day "$stem$suffix.jsonl"
        try { New-Item -ItemType File -Path $file -ErrorAction Stop | Out-Null; return $file }
        catch { if (-not (Test-Path -LiteralPath $file)) { throw }; $n++ }
    }
}

# Run the CLI with no console window and both streams redirected, write the log, and return
# what came back. Cline's own -t ends the run (LESSONS #47); 30 s more and we kill the tree
# ourselves, because the shim's node child would otherwise keep running.
function Start-HeadlessRun([string]$File, [string[]]$Arguments, [string]$LogPath, [int]$TimeoutSec, $Environment) {
    $info = [Diagnostics.ProcessStartInfo]::new()
    $info.FileName = $File
    $info.UseShellExecute = $false; $info.CreateNoWindow = $true
    $info.RedirectStandardOutput = $true; $info.RedirectStandardError = $true
    foreach ($argument in $Arguments) { $info.ArgumentList.Add($argument) }
    if ($Environment) { foreach ($key in $Environment.Keys) { $info.Environment[$key] = [string]$Environment[$key] } }
    $process = [Diagnostics.Process]::new(); $process.StartInfo = $info
    $began = Get-Date
    try {
        [void]$process.Start()
        $stdout = $process.StandardOutput.ReadToEndAsync()
        $stderr = $process.StandardError.ReadToEndAsync()
        $timedOut = -not $process.WaitForExit(($TimeoutSec + 30) * 1000)
        if ($timedOut -and -not $process.HasExited) {
            $process.Kill($true)
            [void]$process.WaitForExit(10000)
        }
        $outText = $stdout.GetAwaiter().GetResult()
        $errText = $stderr.GetAwaiter().GetResult()
        $code = $null
        if ($process.HasExited) { $code = $process.ExitCode }
    } finally { $process.Dispose() }
    $text = $outText.TrimEnd("`r", "`n")
    $lines = if ($text) { @($text -split "`r?`n") } else { @() }
    Set-Content -LiteralPath $LogPath -Value $lines -Encoding utf8
    if ($errText.Trim()) { Set-Content -LiteralPath "$LogPath.stderr.txt" -Value $errText.TrimEnd() -Encoding utf8 }
    [PSCustomObject]@{
        Code = $code; TimedOut = $timedOut; Lines = $lines; Log = $LogPath
        Seconds = [int][math]::Round(((Get-Date) - $began).TotalSeconds)
    }
}

# Cline's JSON Lines: the last "run_result" record carries the outcome and the token counts,
# the last "agent_event" with event.type = "done" carries the answer. Anything else is noise.
function Read-HeadlessResult([string[]]$Lines) {
    $result = [PSCustomObject]@{ Finish = $null; Text = ""; Fresh = $null; Out = $null; Iterations = $null }
    foreach ($line in @($Lines)) {
        if (-not $line -or -not $line.Trim()) { continue }
        $item = $null
        try { $item = $line | ConvertFrom-Json } catch { continue }
        if (-not $item -or -not $item.PSObject.Properties["type"]) { continue }
        if ($item.type -eq "run_result") {
            $result.Finish = [string]$item.finishReason
            $result.Iterations = $item.iterations
            $usage = $item.usage
            if ($usage) {
                $result.Fresh = [long]$usage.inputTokens - [long]$usage.cacheReadTokens
                $result.Out = [long]$usage.outputTokens
            }
        } elseif ($item.type -eq "agent_event" -and $item.event -and $item.event.type -eq "done") {
            $result.Text = [string]$item.event.text
        }
    }
    $result
}

$registry = Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json
$spec = Get-HeadlessProfile $registry $AgentProfile

if (-not $Cwd) { $Cwd = (Get-Location).ProviderPath }
if (-not (Test-Path -LiteralPath $Cwd -PathType Container)) { Fail "Folder not found: $Cwd" }
$Cwd = (Resolve-Path -LiteralPath $Cwd).ProviderPath

$prompt = Get-HeadlessPrompt $Task $TaskFile
if (-not $prompt) { Fail "herdr-headless needs a task: -Task <text> or -TaskFile <path>." }

# $acc, not $account: PowerShell names ignore case, so $account would wipe the -Account parameter.
$acc = $null
if ($Account) { $acc = Get-HeadlessAccount $spec $Account }

$command = Get-HeadlessCommand $spec.command
$arguments = @(Get-HeadlessArgs $spec $Model $Effort $Cwd $TimeoutSec $(if ($acc) { $acc.Args } else { @() })) + $prompt
$log = New-HeadlessLogPath $AgentProfile (Get-Date)
$run = Start-HeadlessRun $command.File ($command.Prefix + $arguments) $log $TimeoutSec $(if ($acc) { $acc.Env } else { $null })

$result = Read-HeadlessResult $run.Lines
$used = if ($Model) { $Model } elseif ($spec.model) { $spec.model } else { "(CLI default)" }
if ($run.Code -eq 0 -and -not $run.TimedOut -and $result.Finish -eq "completed") {
    Write-Output "DONE: $AgentProfile ($used) in $($run.Seconds) s - in $($result.Fresh) out $($result.Out)"
    Write-Output "--- answer ---"
    Write-Output $result.Text
    Write-Output "--- log: $($run.Log) ---"
    exit 0
}
$why = if ($run.TimedOut) { "timeout" } elseif ($result.Finish) { $result.Finish } else { "exit $($run.Code)" }
Write-Output "FAILED ($why)"
foreach ($line in @($run.Lines | Select-Object -Last 20)) { Write-Output $line }
Write-Output "--- log: $($run.Log) ---"
exit 1

