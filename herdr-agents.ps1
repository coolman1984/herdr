# herdr-agents: one command for Claude (or you) to run other AI agent CLIs in Herdr panes.
#
#   herdr-agents tools                                   which agents exist on this PC (agents.json)
#   herdr-agents start <agent> [-Model M] [-Name N] [-Direction right|down] [-Cwd PATH]
#   herdr-agents start <agent> -Pane <pane-id>           start in an existing empty shell pane
#   herdr-agents ask <name> "<task>" [-TimeoutSec 900] [-Lines 200]   send a task, wait, print the answer
#   herdr-agents ask <name> -TaskFile task.md                          long task from a file
#   herdr-agents read <name> [-Lines 200] [-Visible]                   what is on its screen
#   herdr-agents status                                  every live agent and its state
#   herdr-agents stop <name>                             close a pane this tool started
#
# Safety: agents start with tool approval ON where the CLI allows it (agents.json "args").
# When an agent waits for approval, "ask" prints BLOCKED and its screen - a person decides.
[CmdletBinding()]
param(
    [Parameter(Position = 0, Mandatory)]
    [ValidateSet("tools", "start", "ask", "read", "status", "stop")]
    [string]$Command,
    [Parameter(Position = 1)] [string]$Target,
    [Parameter(Position = 2)] [string]$Task,
    [string]$Model,
    [string]$Provider,
    [string]$Name,
    [ValidateSet("right", "down")] [string]$Direction = "right",
    [string]$Cwd,
    [string]$TaskFile,
    [string]$Pane,
    [int]$TimeoutSec = 900,
    [int]$Lines = 200,
    [switch]$Visible
)

$ErrorActionPreference = "Stop"
$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"
$registryPath = Join-Path $PSScriptRoot "agents.json"
$statePath = Join-Path $env:APPDATA "herdr\herdr-agents-panes.json"

function Invoke-Herdr {
    param([string[]]$Arguments)
    $output = & $herdr @Arguments 2>&1
    $code = $LASTEXITCODE
    $text = ($output | ForEach-Object { $_.ToString() }) -join "`n"
    $json = $null
    try { $json = $text | ConvertFrom-Json } catch { }
    [PSCustomObject]@{ Ok = ($code -eq 0); Code = $code; Text = $text; Json = $json }
}

# Terminal reads carry many empty rows; keep the text, drop the padding.
function Format-Screen([string]$Text) {
    $out = New-Object System.Collections.Generic.List[string]
    foreach ($line in ($Text -split "`r?`n")) {
        $line = $line.TrimEnd()
        if (-not $line -and ($out.Count -eq 0 -or -not $out[$out.Count - 1])) { continue }
        $out.Add($line)
    }
    while ($out.Count -gt 0 -and -not $out[$out.Count - 1]) { $out.RemoveAt($out.Count - 1) }
    $out -join "`n"
}

function Fail([string]$Message) {
    Write-Output "ERROR: $Message"
    exit 1
}

function Get-Registry {
    (Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json).agents
}

function Get-LiveAgents {
    $r = Invoke-Herdr @("agent", "list")
    if (-not $r.Ok) { Fail "herdr agent list failed: $($r.Text)" }
    @($r.Json.result.agents)
}

function Get-State {
    if (Test-Path -LiteralPath $statePath) {
        try { return @(Get-Content -LiteralPath $statePath -Raw | ConvertFrom-Json) } catch { }
    }
    @()
}

function Save-State($Items) {
    ConvertTo-Json -InputObject @($Items) -Depth 4 | Set-Content -LiteralPath $statePath -Encoding utf8
}

if (-not (Test-Path -LiteralPath $herdr)) { Fail "Herdr is not installed at $herdr." }

switch ($Command) {
    "tools" {
        $registry = Get-Registry
        foreach ($p in $registry.PSObject.Properties) {
            $a = $p.Value
            $installed = [bool](Get-Command $a.command -ErrorAction SilentlyContinue)
            $model = if ($a.model) { $a.model } else { "(CLI default)" }
            $provider = if ($a.PSObject.Properties["provider"] -and $a.provider) { " provider=$($a.provider)" } else { "" }
            Write-Output ("{0,-9} kind={1,-8} installed={2,-5} model={3}{4}" -f $p.Name, $a.kind, $installed, $model, $provider)
        }
    }

    "start" {
        if (-not $Pane -and ($env:HERDR_ENV -ne "1" -or -not $env:HERDR_PANE_ID)) {
            Fail "start must run inside a Herdr pane (HERDR_ENV=1); it opens the agent next to the caller."
        }
        if (-not $Target) { Fail "start needs an agent, e.g. herdr-agents start codex. See: herdr-agents tools" }
        $entryProfile = (Get-Registry).PSObject.Properties[$Target]
        if (-not $entryProfile) { Fail "Unknown agent '$Target'. See: herdr-agents tools" }
        $a = $entryProfile.Value
        if (-not (Get-Command $a.command -ErrorAction SilentlyContinue)) { Fail "$($a.command) is not installed." }

        $live = @(Get-LiveAgents | ForEach-Object { $_.name } | Where-Object { $_ })
        if (-not $Name) {
            $Name = $Target
            $n = 2
            while ($live -contains $Name) { $Name = "$Target-$n"; $n++ }
        } elseif ($live -contains $Name) {
            Fail "An agent named '$Name' is already running."
        }
        if ($Name -notmatch '^[a-z][a-z0-9_-]{0,31}$') { Fail "Name must match [a-z][a-z0-9_-]{0,31}: $Name" }
        if (-not $Cwd) { $Cwd = (Get-Location).ProviderPath }

        $agentArgs = @($a.args)
        $prov = if ($Provider) { $Provider } elseif ($a.PSObject.Properties["provider"]) { $a.provider } else { "" }
        if ($prov) { $agentArgs += @($a.providerFlag, $prov) }
        $mdl = if ($Model) { $Model } else { $a.model }
        if ($mdl) { $agentArgs += @($a.modelFlag, $mdl) }

        if ($Pane) {
            $pane = $Pane
        } else {
            $split = Invoke-Herdr @("pane", "split", $env:HERDR_PANE_ID, "--direction", $Direction, "--cwd", $Cwd, "--no-focus")
            if (-not $split.Ok) { Fail "Could not open a pane: $($split.Text)" }
            $pane = $split.Json.result.pane.pane_id
        }

        $startArgs = @("agent", "start", $Name, "--kind", $a.kind, "--pane", $pane, "--timeout", "120000")
        if ($agentArgs.Count -gt 0) { $startArgs += @("--") + $agentArgs }
        $started = Invoke-Herdr $startArgs

        $state = @(Get-State | Where-Object { $_.name -ne $Name })
        $state += [PSCustomObject]@{ name = $Name; pane = $pane; agent = $Target; model = $mdl; cwd = $Cwd; started = (Get-Date).ToString("s") }
        Save-State $state

        if (-not $started.Ok) {
            $screen = (Invoke-Herdr @("pane", "read", $pane, "--source", "visible")).Text
            Write-Output "NOT READY: $Name in pane $pane ($($started.Text))"
            Write-Output "--- screen ---"
            Write-Output (Format-Screen $screen)
            Write-Output "--- If it asks to sign in or pick a model, the USER must do that in the pane. ---"
            exit 2
        }
        Write-Output "STARTED name=$Name agent=$Target kind=$($a.kind) pane=$pane model=$(if ($mdl) { $mdl } else { '(CLI default)' }) cwd=$Cwd"
    }

    "ask" {
        if (-not $Target) { Fail "ask needs an agent name. See: herdr-agents status" }
        if ($TaskFile) { $Task = Get-Content -LiteralPath $TaskFile -Raw }
        if (-not $Task) { Fail "ask needs a task (text or -TaskFile)." }
        $r = Invoke-Herdr @("agent", "prompt", $Target, $Task, "--wait", "--timeout", ($TimeoutSec * 1000).ToString())
        $status = if ($r.Json -and $r.Json.result.agent) { $r.Json.result.agent.agent_status } else { $null }
        if (-not $r.Ok) {
            $errCode = if ($r.Json -and $r.Json.error) { $r.Json.error.code } else { "error" }
            Write-Output "FAILED ($errCode): $($r.Text)"
            if ($errCode -in @("agent_blocked", "timeout", "agent_prompt_stalled")) {
                Write-Output "--- screen of $Target ---"
                Write-Output (Format-Screen (Invoke-Herdr @("agent", "read", $Target, "--source", "visible")).Text)
                Write-Output "--- Do not resend blindly: a timeout does not prove the task was not delivered. ---"
            }
            exit 1
        }
        if ($status -eq "blocked") {
            Write-Output "BLOCKED: $Target is asking for approval or an answer. Show this to the user; do not approve on their behalf."
            Write-Output "--- screen of $Target ---"
            Write-Output (Format-Screen (Invoke-Herdr @("agent", "read", $Target, "--source", "visible")).Text)
            exit 3
        }
        Write-Output "DONE: $Target is $status"
        Write-Output "--- recent output of $Target ---"
        Write-Output (Format-Screen (Invoke-Herdr @("agent", "read", $Target, "--source", "recent-unwrapped", "--lines", $Lines.ToString())).Text)
    }

    "read" {
        if (-not $Target) { Fail "read needs an agent name." }
        $source = if ($Visible) { "visible" } else { "recent-unwrapped" }
        $r = Invoke-Herdr @("agent", "read", $Target, "--source", $source, "--lines", $Lines.ToString())
        if (-not $r.Ok) { Fail $r.Text }
        Write-Output (Format-Screen $r.Text)
    }

    "status" {
        $mine = Get-State
        foreach ($a in Get-LiveAgents) {
            $tag = if ($mine | Where-Object { $_.pane -eq $a.pane_id }) { "started-by-tool" } else { "" }
            Write-Output ("{0,-14} {1,-9} {2,-8} pane={3,-7} {4} {5}" -f ($(if ($a.name) { $a.name } else { "(unnamed)" })), $a.agent, $a.agent_status, $a.pane_id, $a.cwd, $tag)
        }
    }

    "stop" {
        if (-not $Target) { Fail "stop needs an agent name." }
        $state = @(Get-State)
        $entry = $state | Where-Object { $_.name -eq $Target } | Select-Object -First 1
        if (-not $entry) { Fail "'$Target' was not started by this tool; close it yourself in Herdr." }
        $r = Invoke-Herdr @("pane", "close", $entry.pane)
        Save-State @($state | Where-Object { $_.name -ne $Target })
        if (-not $r.Ok) { Fail "Could not close pane $($entry.pane): $($r.Text)" }
        Write-Output "STOPPED $Target (pane $($entry.pane) closed)"
    }
}
