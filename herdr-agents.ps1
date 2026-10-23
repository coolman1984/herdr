# herdr-agents: one command for Claude (or you) to run other AI agent CLIs in Herdr panes.
#
#   herdr-agents tools                                   which agents exist on this PC (agents.json)
#   herdr-agents start <agent> [-Model M] [-Name N] [-Direction right|down] [-Cwd PATH]
#   herdr-agents start <agent> -Pane <pane-id>           start in an existing empty shell pane
#   herdr-agents start codex -Effort medium              reasoning effort (agents.json effortArgs)
#   herdr-agents ask <name> "<task>" [-TimeoutSec 900] [-Lines 200]   send a task, wait, print the answer
#   herdr-agents ask <name> -TaskFile task.md                          long task from a file
#   herdr-agents read <name> [-Lines 200] [-Visible]                   what is on its screen
#   herdr-agents status                                  every live agent and its state
#   herdr-agents stop <name>                             close a pane this tool started
#   herdr-agents start <agent> -Worktree <branch> [-Cwd REPO]   own git worktree beside caller
#   herdr-agents diff <name>                             what the agent changed (git, read-only)
#   herdr-agents fanout "<task>" -Agents codex,deepseek  same task to several agents in parallel
#   herdr-agents history [<n>]                           past tasks and answers (n = show one)
#   herdr-agents watch [-Once] [-OpenPane]               live board: who is blocked / working / done
#   herdr-agents team -PlanFile team.json [-LeadShare 0.4]   a team in columns right of me, tasks sent
#     { "agents": [ { "name": "coder", "agent": "codex", "model": "gpt-6.1-sol", "effort": "medium",
#                     "provider": "", "cwd": "D:\\x", "task": "..." or "taskFile": "D:\\x\\t.md" } ] }
#     Page 1 = me + up to 3 agents; agents 4.. go 4 per page into new tabs "team 2", "team 3" ...
#     (-Overflow space = new spaces instead). Up to 16 agents.
#   herdr-agents collect -Agents coder,tester            wait for all, one report (like fanout)
#   herdr-agents login codex -Account work2              sign a separate account in (owner finishes it)
#   herdr-agents start codex -Account work2              agent on that account (own limits)
#   herdr-agents accounts                                every separate account folder
#   herdr-agents usage                                   Claude 5-hour / weekly limits per account, NEXT account
#   herdr-agents start next                              Claude on the account with the least 5-hour use
#   herdr-agents handoff [-TaskFile out.md]              a handoff file for the next lead, facts filled in
#
# Approval: agents.json "args" set each agent's approval mode; since 2026-10-22 the owner
# runs them all without approval prompts (CLAUDE.md rule 3). Secrets and trust stay the owner's.
# When an agent waits for approval, "ask" prints BLOCKED and its screen - a person decides.
[CmdletBinding()]
param(
    [Parameter(Position = 0, Mandatory)]
    [ValidateSet("tools", "start", "ask", "read", "status", "stop", "diff", "fanout", "history", "watch", "team", "collect", "login", "accounts", "usage", "handoff")]
    [string]$Command,
    [Parameter(Position = 1)] [string]$Target,
    [Parameter(Position = 2)] [string]$Task,
    [string]$Model,
    [string]$Effort,
    [string]$Account,
    [string]$Provider,
    [string]$Name,
    [ValidateSet("right", "down")] [string]$Direction = "right",
    [string]$Cwd,
    [string]$TaskFile,
    [string]$PlanFile,
    [double]$LeadShare = 0.4,
    [ValidateSet("tab", "space")] [string]$Overflow = "tab",
    [string]$Pane,
    [string]$Worktree,
    [string[]]$Agents,
    [int]$TimeoutSec = 900,
    [int]$Lines = 200,
    [int]$IntervalSec = 2,
    [switch]$Visible,
    [switch]$Once,
    [switch]$OpenPane
)

$ErrorActionPreference = "Stop"
$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"
$registryPath = Join-Path $PSScriptRoot "agents.json"
. (Join-Path $PSScriptRoot "claude-limits.ps1")
$statePath = Join-Path $env:APPDATA "herdr\herdr-agents-panes.json"
$runsPath = Join-Path $env:APPDATA "herdr\agent-runs"

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

# Herdr gives every pane HERDR_SOCKET_PATH = the default socket, while key commands and old
# records have none: both mean the default server (LESSONS #46).
function ConvertTo-ServerId([string]$Socket) {
    if (-not $Socket -or $Socket -eq "default") { return "default" }
    $id = $Socket.Trim().ToLowerInvariant()
    if ($id -eq (Join-Path $env:APPDATA "herdr\herdr.sock").ToLowerInvariant()) { return "default" }
    $id
}

function Get-ServerIdentity { ConvertTo-ServerId $env:HERDR_SOCKET_PATH }

function Get-RecordServer($Record) {
    if ($Record.PSObject.Properties['server']) { return (ConvertTo-ServerId ([string]$Record.server)) }
    "default"
}

# Turn the result of "agent prompt --wait" into DONE / BLOCKED / FAILED with the text to show.
function Complete-Ask([string]$AgentName, $Result) {
    # Not "$lines": PowerShell names ignore case, and $Lines is a parameter.
    $out = New-Object System.Collections.Generic.List[string]
    $status = if ($Result.Json -and $Result.Json.result.agent) { $Result.Json.result.agent.agent_status } else { $null }
    $outcome = "DONE"
    # A settled prompt result can precede the turn's first working state (LESSONS #44).
    if ($Result.Ok -and $status -in @("done", "idle")) {
        $began = Get-Date
        $deadline = $began.AddSeconds([Math]::Min(10, $TimeoutSec))
        while ((Get-Date) -lt $deadline) {
            Start-Sleep -Seconds 2
            $g = Invoke-Herdr @("agent", "get", $AgentName)
            $observed = if ($g.Ok -and $g.Json.result.agent) { $g.Json.result.agent.agent_status } else { "unknown" }
            if ($observed -in @("working", "blocked")) {
                $remaining = [Math]::Max(1, [int](($TimeoutSec - ((Get-Date) - $began).TotalSeconds) * 1000))
                $Result = Invoke-Herdr @("agent", "wait", $AgentName, "--timeout", $remaining.ToString())
                $status = if ($Result.Json.result.agent) { $Result.Json.result.agent.agent_status } else { "unknown" }
                break
            }
        }
    }
    if (-not $Result.Ok) {
        $errCode = if ($Result.Json -and $Result.Json.error) { $Result.Json.error.code } else { "error" }
        if ($errCode -eq "agent_blocked") {
            $status = "blocked"
        } elseif ($errCode -ne "agent_prompt_stalled") {
            $out.Add("FAILED ($errCode): $($Result.Text)")
            if ($errCode -eq "timeout") {
                $out.Add("--- screen of $AgentName ---")
                $out.Add((Format-Screen (Invoke-Herdr @("agent", "read", $AgentName, "--source", "visible")).Text))
                $out.Add("--- Do not resend blindly: a timeout does not prove the task was not delivered. ---")
            }
            return [PSCustomObject]@{ Name = $AgentName; Outcome = "FAILED"; Exit = 1; Text = ($out -join "`n") }
        } else {
            # The prompt was delivered but Herdr saw no "working" within 5 s (LESSONS #21). Its
            # state can lag behind the agent, so "idle" right now proves nothing: watch the state
            # for up to 30 s; once it shows working, wait until it settles.
            $deadline = (Get-Date).AddSeconds(30)
            $status = "unknown"
            while ((Get-Date) -lt $deadline) {
                $g = Invoke-Herdr @("agent", "get", $AgentName)
                $status = if ($g.Json -and $g.Json.result.agent) { $g.Json.result.agent.agent_status } else { "unknown" }
                if ($status -in @("working", "blocked")) {
                    $w = Invoke-Herdr @("agent", "wait", $AgentName, "--timeout", ($TimeoutSec * 1000).ToString())
                    $status = if ($w.Json -and $w.Json.result.agent) { $w.Json.result.agent.agent_status } else { "unknown" }
                    break
                }
                Start-Sleep -Seconds 2
            }
            $outcome = "UNCONFIRMED"
            $out.Add("UNCONFIRMED: Herdr did not see $AgentName start working; it is now $status. Check the answer below is complete.")
        }
    }
    if ($status -eq "blocked") {
        $out.Add("BLOCKED: $AgentName is asking for approval or an answer. Show this to the user; do not approve on their behalf.")
        $out.Add("--- screen of $AgentName ---")
        $out.Add((Format-Screen (Invoke-Herdr @("agent", "read", $AgentName, "--source", "visible")).Text))
        return [PSCustomObject]@{ Name = $AgentName; Outcome = "BLOCKED"; Exit = 3; Text = ($out -join "`n") }
    }
    $read = Invoke-Herdr @("agent", "read", $AgentName, "--source", "recent-unwrapped", "--lines", $Lines.ToString())
    if (-not $read.Ok) {
        if ($read.Json.error.code -eq "agent_not_idle") {
            return [PSCustomObject]@{ Name = $AgentName; Outcome = "UNCONFIRMED"; Exit = 0
                Text = "UNCONFIRMED: $AgentName is still working; read it later with: herdr-agents read $AgentName" }
        }
        return [PSCustomObject]@{ Name = $AgentName; Outcome = "FAILED"; Exit = 1; Text = "FAILED reading ${AgentName}: $($read.Text)" }
    }
    if ($outcome -eq "DONE") { $out.Add("DONE: $AgentName is $status") }
    $out.Add("--- recent output of $AgentName ---")
    $out.Add((Format-Screen $read.Text))
    [PSCustomObject]@{ Name = $AgentName; Outcome = $outcome; Exit = 0; Text = ($out -join "`n") }
}

# Every task and its answer is kept as Markdown: %APPDATA%\herdr\agent-runs\<date>\<time>-<name>.md
function Save-Run([string]$Label, [string]$TaskText, [string]$Outcome, [string]$Body) {
    $day = Join-Path $runsPath (Get-Date -Format "yyyy-MM-dd")
    New-Item -ItemType Directory -Force -Path $day | Out-Null
    $stem = "{0}-{1}" -f (Get-Date -Format "HHmmss"), $Label
    $n = 1
    while ($true) {
        $suffix = if ($n -eq 1) { "" } else { "-$n" }
        $file = Join-Path $day "$stem$suffix.md"
        try { New-Item -ItemType File -Path $file -ErrorAction Stop | Out-Null; break }
        catch { if (-not (Test-Path -LiteralPath $file)) { throw }; $n++ }
    }
    $text = "# $Label - $Outcome`n`nTime: $((Get-Date).ToString('s'))`n`n## Task`n`n$TaskText`n`n## Answer`n`n$Body`n"
    Set-Content -LiteralPath $file -Value $text -Encoding utf8
    $file
}

# Separate accounts: a profile's "account" block in agents.json says how the CLI is pointed at
# its own sign-in folder %USERPROFILE%\.herdr-accounts\<tool>\<name> (env vars and/or args,
# {dir} = that folder). On first use the folder is created and the listed settings are copied
# from the CLI's normal folder - never sign-in files (LESSONS #38).
$accountsRoot = Join-Path $env:USERPROFILE ".herdr-accounts"
$defaultHomes = @{ claude = (Join-Path $env:USERPROFILE ".claude"); codex = (Join-Path $env:USERPROFILE ".codex") }
function Get-Account($Entry, [string]$AccountName) {
    if (-not $Entry.PSObject.Properties["account"] -or -not $Entry.account) { Fail "This agent has no separate-account support (agents.json 'account')." }
    if ($AccountName -notmatch '^[a-z0-9][a-z0-9_-]{0,23}$') { Fail "Account name must match [a-z0-9][a-z0-9_-]{0,23}: $AccountName" }
    $spec = $Entry.account
    $dir = Join-Path $accountsRoot (Join-Path $spec.tool $AccountName)
    if (-not (Test-Path -LiteralPath $dir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
        $from = $defaultHomes[$spec.tool]
        foreach ($item in @($spec.copy)) {
            if ($from -and $item -and (Test-Path -LiteralPath (Join-Path $from $item))) {
                Copy-Item -LiteralPath (Join-Path $from $item) -Destination (Join-Path $dir $item) -Recurse -Force
            }
        }
    }
    $envs = [ordered]@{}
    foreach ($p in @($spec.env.PSObject.Properties)) { $envs[$p.Name] = $p.Value.Replace("{dir}", $dir) }
    [PSCustomObject]@{
        Name = $AccountName; Tool = $spec.tool; Dir = $dir; Env = $envs
        Args = @($spec.args | Where-Object { $_ } | ForEach-Object { $_.Replace("{dir}", $dir) })
        Login = ([string]$spec.login).Replace("{dir}", $dir)
    }
}

function Get-EnvArgs($Acc) {
    $out = @()
    if ($Acc) { foreach ($k in $Acc.Env.Keys) { $out += @("--env", "$k=$($Acc.Env[$k])") } }
    $out
}

function Invoke-Git([string]$Dir, [string[]]$Arguments) {
    $out = & git -C $Dir @Arguments 2>&1
    [PSCustomObject]@{ Ok = ($LASTEXITCODE -eq 0); Text = (($out | ForEach-Object { $_.ToString() }) -join "`n").TrimEnd() }
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
        # Started from a Herdr key ([[keys.command]]): Herdr passes the focused pane as
        # HERDR_ACTIVE_PANE_ID / _CWD instead of the caller's HERDR_PANE_ID.
        if (-not $env:HERDR_PANE_ID -and $env:HERDR_ACTIVE_PANE_ID) {
            $env:HERDR_PANE_ID = $env:HERDR_ACTIVE_PANE_ID; $env:HERDR_ENV = "1"
            if (-not $Pane -and -not $Cwd -and $env:HERDR_ACTIVE_PANE_CWD) { $Cwd = $env:HERDR_ACTIVE_PANE_CWD }
        }
        if ($Cwd) {
            if (-not (Test-Path -LiteralPath $Cwd -PathType Container)) { Fail "Folder not found: $Cwd" }
            $Cwd = (Resolve-Path -LiteralPath $Cwd).ProviderPath
        }
        $explicitName = [bool]$Name
        # Short names (agents.json "aliases"): e.g. "2040" = profile claude on account 2040.
        $aliases = (Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json).aliases
        # "next" = Claude on the account with the least 5-hour use, below the limit.
        if ($Target -eq "next") {
            $next = Get-ClaudeNextAccount (Get-ClaudeLimits)
            if (-not $next) { Fail "No Claude account below $($script:ClaudeLimitThreshold)% with usage data. See: herdr-agents usage" }
            $Target = if ($aliases -and $aliases.PSObject.Properties[$next.Account]) { $next.Account } else { "claude" }
            if ($Target -eq "claude" -and -not $next.Main -and -not $Account) { $Account = $next.Account }
            Write-Output "NEXT account: $($next.Account) (5h $($next.FiveHour)%, weekly $($next.SevenDay)%)"
        }
        if ($Target -and $aliases -and $aliases.PSObject.Properties[$Target]) {
            $al = $aliases.$Target
            if (-not $Account -and $al.account) { $Account = $al.account }
            if (-not $Name) { $Name = $al.name }
            $Target = $al.profile
        }
        if (-not $Pane -and -not $Worktree -and ($env:HERDR_ENV -ne "1" -or -not $env:HERDR_PANE_ID)) {
            Fail "start must run inside a Herdr pane (HERDR_ENV=1); it opens the agent next to the caller."
        }
        if (-not $Target) { Fail "start needs an agent, e.g. herdr-agents start codex. See: herdr-agents tools" }
        $entryProfile = (Get-Registry).PSObject.Properties[$Target]
        if (-not $entryProfile) { Fail "Unknown agent '$Target'. See: herdr-agents tools" }
        $a = $entryProfile.Value
        if (-not (Get-Command $a.command -ErrorAction SilentlyContinue)) { Fail "$($a.command) is not installed." }

        $acc = if ($Account) { Get-Account $a $Account } else { $null }
        $live = @(Get-LiveAgents | ForEach-Object { $_.name } | Where-Object { $_ })
        if (-not $explicitName) {
            if (-not $Name) { $Name = if ($acc) { "$Target-$Account" } else { $Target } }
            $baseName = $Name
            $n = 2
            while ($live -contains $Name) { $Name = "$baseName-$n"; $n++ }
        } elseif ($live -contains $Name) {
            Fail "An agent named '$Name' is already running."
        }
        if ($Name -cnotmatch '^[a-z][a-z0-9_-]{0,31}$') { Fail "Name must match [a-z][a-z0-9_-]{0,31}: $Name" }
        if ($Pane -and -not $Worktree) {
            $p = Invoke-Herdr @("pane", "get", $Pane)
            if (-not $p.Ok -or -not $p.Json.result.pane.cwd) { Fail "Could not read pane ${Pane}: $($p.Text)" }
            $paneCwd = [string]$p.Json.result.pane.cwd
            if ($Cwd -and -not [string]::Equals([IO.Path]::TrimEndingDirectorySeparator($Cwd), [IO.Path]::TrimEndingDirectorySeparator($paneCwd), [StringComparison]::OrdinalIgnoreCase)) {
                Fail "-Cwd differs from pane $Pane's folder $paneCwd."
            }
            $Cwd = $paneCwd
        }
        if (-not $Cwd) {
            $Cwd = (Get-Location).ProviderPath
            if (-not (Test-Path -LiteralPath $Cwd -PathType Container)) { Fail "Folder not found: $Cwd" }
            $Cwd = (Resolve-Path -LiteralPath $Cwd).ProviderPath
        }

        $agentArgs = @($a.args)
        $prov = if ($Provider) { $Provider } elseif ($a.PSObject.Properties["provider"]) { $a.provider } else { "" }
        if ($prov) { $agentArgs += @($a.providerFlag, $prov) }
        $mdl = if ($Model) { $Model } else { $a.model }
        if ($mdl) { $agentArgs += @($a.modelFlag, $mdl) }
        # Reasoning effort (low / medium / high ...): agents.json "effortArgs", {0} = the value;
        # the profile's "effort" is the default when -Effort is not given.
        if (-not $Effort -and $a.PSObject.Properties["effort"] -and $a.effort) { $Effort = $a.effort }
        if ($Effort) {
            if (-not $a.PSObject.Properties["effortArgs"]) { Fail "$Target has no effort setting in agents.json." }
            $agentArgs += @($a.effortArgs | ForEach-Object { $_ -f $Effort })
        }
        if ($acc) { $agentArgs += $acc.Args }
        $envArgs = @(Get-EnvArgs $acc)

        $repoRoot = $null; $baseBranch = $null; $createdPane = $false; $createdSpace = $false
        if ($Worktree) {
            # The agent gets its own branch and folder beside the caller. Plain git:
            # Herdr's own "worktree create" crashed the server once (LESSONS #19).
            $top = Invoke-Git $Cwd @("rev-parse", "--show-toplevel")
            if (-not $top.Ok) { Fail "-Worktree needs a git repository; $Cwd is not one." }
            $repoRoot = [System.IO.Path]::GetFullPath($top.Text)
            if (-not (Invoke-Git $repoRoot @("check-ref-format", "--branch", $Worktree)).Ok) { Fail "Not a valid branch name: $Worktree" }
            $baseBranch = (Invoke-Git $repoRoot @("rev-parse", "--abbrev-ref", "HEAD")).Text
            $wtPath = Join-Path $env:USERPROFILE (".herdr\worktrees\{0}\{1}" -f (Split-Path -Leaf $repoRoot), ($Worktree -replace '[\\/]', '-'))
            if (Test-Path -LiteralPath $wtPath) { Fail "Worktree folder already exists: $wtPath" }
            $exists = (Invoke-Git $repoRoot @("show-ref", "--verify", "--quiet", "refs/heads/$Worktree")).Ok
            $add = if ($exists) { Invoke-Git $repoRoot @("worktree", "add", $wtPath, $Worktree) }
                   else { Invoke-Git $repoRoot @("worktree", "add", "-b", $Worktree, $wtPath) }
            if (-not $add.Ok) { Fail "git worktree add failed: $($add.Text)" }
            $Cwd = $wtPath
            if ($env:HERDR_ENV -eq "1" -and $env:HERDR_PANE_ID) {
                $split = Invoke-Herdr (@("pane", "split", $env:HERDR_PANE_ID, "--direction", $Direction, "--cwd", $wtPath, "--no-focus") + $envArgs)
                if (-not $split.Ok) { Fail "Worktree made at $wtPath, but Herdr could not open a pane: $($split.Text)" }
                $pane = $split.Json.result.pane.pane_id
            } else {
                $ws = Invoke-Herdr (@("workspace", "create", "--cwd", $wtPath, "--label", "$Name $Worktree", "--no-focus") + $envArgs)
                if (-not $ws.Ok) { Fail "Worktree made at $wtPath, but Herdr could not open a space: $($ws.Text)" }
                $pane = $ws.Json.result.root_pane.pane_id
                $createdSpace = $true
            }
            $createdPane = $true
        } elseif ($Pane) {
            $pane = $Pane
            if ($acc) {
                # An existing pane already has its environment: set the account's variables in
                # its PowerShell before the agent starts there.
                $set = ($acc.Env.Keys | ForEach-Object { "`$env:$_='$($acc.Env[$_].Replace("'", "''"))'" }) -join "; "
                if ($set) { [void](Invoke-Herdr @("pane", "run", $pane, $set)); Start-Sleep -Milliseconds 800 }
            }
        } else {
            $split = Invoke-Herdr (@("pane", "split", $env:HERDR_PANE_ID, "--direction", $Direction, "--cwd", $Cwd, "--no-focus") + $envArgs)
            if (-not $split.Ok) { Fail "Could not open a pane: $($split.Text)" }
            $pane = $split.Json.result.pane.pane_id
            $createdPane = $true
        }

        $startArgs = @("agent", "start", $Name, "--kind", $a.kind, "--pane", $pane, "--timeout", "120000")
        if ($agentArgs.Count -gt 0) { $startArgs += @("--") + $agentArgs }
        $started = Invoke-Herdr $startArgs

        $recordPane = $createdPane -or $started.Ok
        if (-not $recordPane -and $Pane) {
            # Herdr keeps the identity even when sign-in or trust prevents readiness.
            $recordPane = [bool](Get-LiveAgents | Where-Object { $_.name -eq $Name -and $_.pane_id -eq $pane } | Select-Object -First 1)
        }
        if ($recordPane) {
            $server = Get-ServerIdentity
            $state = @(Get-State | Where-Object { $_.name -ne $Name -or (Get-RecordServer $_) -ne $server })
            $state += [PSCustomObject]@{ name = $Name; pane = $pane; agent = $Target; agentKind = $a.kind; server = $server; model = $mdl; cwd = $Cwd; started = (Get-Date).ToString("s")
                repo = $repoRoot; branch = $(if ($Worktree) { $Worktree } else { $null }); base = $baseBranch
                account = $(if ($acc) { $acc.Name } else { $null }); accountTool = $(if ($acc) { $acc.Tool } else { $null }); accountDir = $(if ($acc) { $acc.Dir } else { $null }) }
            Save-State $state
        }

        if (-not $started.Ok) {
            $screen = (Invoke-Herdr @("pane", "read", $pane, "--source", "visible")).Text
            Write-Output "NOT READY: $Name in pane $pane ($($started.Text))"
            Write-Output "--- screen ---"
            Write-Output (Format-Screen $screen)
            Write-Output "--- If it asks to sign in or pick a model, the USER must do that in the pane. ---"
            exit 2
        }
        $where = if ($Worktree) { " branch=$Worktree (from $baseBranch)" } else { "" }
        if ($createdSpace) { $where += " space=new" }
        if ($acc) { $where += " account=$($acc.Name)" }
        Write-Output "STARTED name=$Name agent=$Target kind=$($a.kind) pane=$pane model=$(if ($mdl) { $mdl } else { '(CLI default)' }) cwd=$Cwd$where"
    }

    "ask" {
        if (-not $Target) { Fail "ask needs an agent name. See: herdr-agents status" }
        if ($TaskFile) { $Task = Get-Content -LiteralPath $TaskFile -Raw }
        if (-not $Task) { Fail "ask needs a task (text or -TaskFile)." }
        $r = Invoke-Herdr @("agent", "prompt", $Target, $Task, "--wait", "--timeout", ($TimeoutSec * 1000).ToString())
        $done = Complete-Ask $Target $r
        $log = Save-Run $Target $Task $done.Outcome $done.Text
        Write-Output $done.Text
        Write-Output "--- saved: $log ---"
        exit $done.Exit
    }

    "fanout" {
        # The same task to several agents at once; one report with every answer.
        $task = if ($TaskFile) { Get-Content -LiteralPath $TaskFile -Raw } elseif ($Target) { $Target } else { $null }
        if (-not $task) { Fail "fanout needs a task (text or -TaskFile)." }
        $names = @($Agents | ForEach-Object { $_ -split ',' } | ForEach-Object { $_.Trim() } | Where-Object { $_ } | Select-Object -Unique)
        if ($names.Count -lt 2) { Fail "fanout needs two or more agents: -Agents codex,deepseek (see: herdr-agents status)" }
        $live = @(Get-LiveAgents | ForEach-Object { $_.name } | Where-Object { $_ })
        $missing = @($names | Where-Object { $live -notcontains $_ })
        if ($missing) { Fail "Not running: $($missing -join ', '). Start them first (herdr-agents start ...)." }
        $ms = ($TimeoutSec * 1000).ToString()
        $jobs = foreach ($n in $names) {
            Start-ThreadJob -ArgumentList $herdr, $n, $task, $ms -ScriptBlock {
                param($h, $n, $t, $ms)
                $out = & $h agent prompt $n $t --wait --timeout $ms 2>&1
                $text = ($out | ForEach-Object { $_.ToString() }) -join "`n"
                $json = $null
                try { $json = $text | ConvertFrom-Json } catch { }
                [PSCustomObject]@{ Name = $n; Ok = ($LASTEXITCODE -eq 0); Text = $text; Json = $json }
            }
        }
        $results = @($jobs | Wait-Job | Receive-Job)
        $jobs | Remove-Job
        $answers = foreach ($n in $names) { Complete-Ask $n ($results | Where-Object { $_.Name -eq $n } | Select-Object -First 1) }
        $body = ($answers | ForEach-Object { "### $($_.Name): $($_.Outcome)`n`n``````text`n$($_.Text)`n```````n" }) -join "`n"
        $summary = ($answers | ForEach-Object { "$($_.Name)=$($_.Outcome)" }) -join " "
        $log = Save-Run ("fanout-" + ($names -join "-")) $task $summary $body
        foreach ($x in $answers) { Write-Output "===== $($x.Name): $($x.Outcome) ====="; Write-Output $x.Text; Write-Output "" }
        Write-Output "FANOUT $summary"
        Write-Output "--- saved: $log ---"
        $worst = ($answers | Measure-Object -Property Exit -Maximum).Maximum
        exit $(if ($answers | Where-Object { $_.Outcome -eq "FAILED" }) { 1 } else { $worst })
    }

    "team" {
        # The lead (the caller, usually Claude) keeps the left part of its pane; the team
        # gets equal columns to the right; every agent starts and gets its task, no waiting.
        if ($env:HERDR_ENV -ne "1" -or -not $env:HERDR_PANE_ID) { Fail "team must run inside a Herdr pane (HERDR_ENV=1)." }
        if (-not $PlanFile -or -not (Test-Path -LiteralPath $PlanFile)) { Fail "team needs -PlanFile team.json (format in this script's header)." }
        $members = @((Get-Content -LiteralPath $PlanFile -Raw | ConvertFrom-Json).agents)
        if ($members.Count -lt 1 -or $members.Count -gt 16) { Fail "A team has 1 to 16 agents." }
        $registry = Get-Registry
        foreach ($m in $members) {
            if (-not $registry.PSObject.Properties[$m.agent]) { Fail "Unknown agent '$($m.agent)' for '$($m.name)'. See: herdr-agents tools" }
            if ($m.name -and $m.name -cnotmatch '^[a-z][a-z0-9_-]{0,31}$') { Fail "Name must match [a-z][a-z0-9_-]{0,31}: $($m.name)" }
        }
        # Agents without a "cwd" work in the lead pane's folder.
        $lead = (Invoke-Herdr @("pane", "get", $env:HERDR_PANE_ID)).Json.result.pane
        $here = if ($lead -and $lead.cwd) { [string]$lead.cwd } else { (Get-Location).ProviderPath }
        $cwdOf = { param($m) if ($m.PSObject.Properties["cwd"] -and $m.cwd) { $m.cwd } else { $here } }
        $fmt = { param($v) [string]::Format([cultureinfo]::InvariantCulture, "{0:0.####}", $v) }
        # Split $First into $Count equal columns to its right... (ratio 1/remaining, LESSONS #17).
        $fill = {
            param($first, $group)
            $ids = @($first)
            for ($k = 1; $k -lt $group.Count; $k++) {
                $s = Invoke-Herdr @("pane", "split", $ids[-1], "--direction", "right", "--ratio", (& $fmt (1.0 / ($group.Count - $k + 1))), "--cwd", (& $cwdOf $group[$k]), "--no-focus")
                if (-not $s.Ok) { Fail "Could not open a column for $($group[$k].name): $($s.Text)" }
                $ids += $s.Json.result.pane.pane_id
            }
            $ids
        }
        # Page 1 = the lead + up to 3 agents (4 columns). More agents: 4 per extra page, each
        # page a new tab of the lead's space ("team 2", "team 3" ...), or a new space with
        # -Overflow space.
        $firstGroup = @($members | Select-Object -First 3)
        $split = Invoke-Herdr @("pane", "split", $env:HERDR_PANE_ID, "--direction", "right", "--ratio", (& $fmt ([Math]::Min(0.8, [Math]::Max(0.2, $LeadShare)))), "--cwd", (& $cwdOf $firstGroup[0]), "--no-focus")
        if (-not $split.Ok) { Fail "Could not open the team area: $($split.Text)" }
        $panes = @(& $fill $split.Json.result.pane.pane_id $firstGroup)
        $rest = @($members | Select-Object -Skip 3)
        $page = 2
        while ($rest.Count -gt 0) {
            $group = @($rest | Select-Object -First 4)
            $rest = @($rest | Select-Object -Skip 4)
            $label = "team $page"
            $made = if ($Overflow -eq "space") {
                Invoke-Herdr @("workspace", "create", "--cwd", (& $cwdOf $group[0]), "--label", "$((Invoke-Herdr @('workspace', 'get', $lead.workspace_id)).Json.result.workspace.label) $label", "--no-focus")
            } else {
                Invoke-Herdr @("tab", "create", "--workspace", $lead.workspace_id, "--cwd", (& $cwdOf $group[0]), "--label", $label, "--no-focus")
            }
            if (-not $made.Ok) { Fail "Could not open page ${page}: $($made.Text)" }
            $panes += @(& $fill $made.Json.result.root_pane.pane_id $group)
            Write-Output "PAGE $page ($Overflow): $(($group | ForEach-Object { $_.name }) -join ', ')"
            $page++
        }
        $exit = 0
        for ($i = 0; $i -lt $members.Count; $i++) {
            $m = $members[$i]
            $startArgs = @("-NoProfile", "-File", $PSCommandPath, "start", $m.agent, "-Pane", $panes[$i], "-Cwd", (& $cwdOf $m))
            foreach ($f in @("name", "model", "effort", "provider", "account")) {
                if ($m.PSObject.Properties[$f] -and $m.$f) { $startArgs += @("-$((Get-Culture).TextInfo.ToTitleCase($f))", $m.$f) }
            }
            $out = (& pwsh @startArgs 2>&1) -join "`n"
            $code = $LASTEXITCODE
            Write-Output $out
            if ($code -ne 0) { $exit = 2; continue }
            $who = if ($out -match 'STARTED name=(\S+)') { $Matches[1] } else { $null }
            if ($who) { [void](Invoke-Herdr @("pane", "rename", $panes[$i], $who)) }
            $t = if ($m.PSObject.Properties["taskFile"] -and $m.taskFile) { Get-Content -LiteralPath $m.taskFile -Raw } elseif ($m.PSObject.Properties["task"]) { $m.task } else { $null }
            if ($who -and $t) {
                $sent = Invoke-Herdr @("agent", "prompt", $who, $t)
                Write-Output $(if ($sent.Ok) { "SENT task to $who" } else { "NOT SENT to ${who}: $($sent.Text)" })
                if (-not $sent.Ok) { $exit = 1 }
            }
        }
        Write-Output "TEAM ready. Watch: herdr-agents read <name> -Visible   Collect: herdr-agents collect -Agents <names>"
        exit $exit
    }

    "collect" {
        # Wait for every named agent to settle, then one report (as fanout, without sending).
        $names = @($Agents | ForEach-Object { $_ -split ',' } | ForEach-Object { $_.Trim() } | Where-Object { $_ } | Select-Object -Unique)
        if (-not $names) { Fail "collect needs -Agents a,b,c (see: herdr-agents status)" }
        $ms = ($TimeoutSec * 1000).ToString()
        $jobs = foreach ($n in $names) {
            Start-ThreadJob -ArgumentList $herdr, $n, $ms -ScriptBlock {
                param($h, $n, $ms)
                $out = & $h agent wait $n --timeout $ms 2>&1
                $text = ($out | ForEach-Object { $_.ToString() }) -join "`n"
                $json = $null
                try { $json = $text | ConvertFrom-Json } catch { }
                [PSCustomObject]@{ Name = $n; Ok = ($LASTEXITCODE -eq 0); Text = $text; Json = $json }
            }
        }
        $results = @($jobs | Wait-Job | Receive-Job)
        $jobs | Remove-Job
        $answers = foreach ($n in $names) { Complete-Ask $n ($results | Where-Object { $_.Name -eq $n } | Select-Object -First 1) }
        $body = ($answers | ForEach-Object { "### $($_.Name): $($_.Outcome)`n`n``````text`n$($_.Text)`n```````n" }) -join "`n"
        $summary = ($answers | ForEach-Object { "$($_.Name)=$($_.Outcome)" }) -join " "
        $log = Save-Run ("collect-" + ($names -join "-")) "(collect)" $summary $body
        foreach ($x in $answers) { Write-Output "===== $($x.Name): $($x.Outcome) ====="; Write-Output $x.Text; Write-Output "" }
        Write-Output "COLLECT $summary"
        Write-Output "--- saved: $log ---"
        $worst = ($answers | Measure-Object -Property Exit -Maximum).Maximum
        exit $(if ($answers | Where-Object { $_.Outcome -eq "FAILED" }) { 1 } else { $worst })
    }

    "login" {
        # Open a pane with the account's own folder and run the CLI's sign-in there; the owner
        # completes it (browser / device code). Never types a key or password for them.
        if ($env:HERDR_ENV -ne "1" -or -not $env:HERDR_PANE_ID) { Fail "login must run inside a Herdr pane (HERDR_ENV=1)." }
        if (-not $Target -or -not $Account) { Fail "login needs a profile and -Account, e.g. herdr-agents login codex -Account work2" }
        $entry = (Get-Registry).PSObject.Properties[$Target]
        if (-not $entry) { Fail "Unknown agent '$Target'. See: herdr-agents tools" }
        $acc = Get-Account $entry.Value $Account
        $split = Invoke-Herdr (@("pane", "split", $env:HERDR_PANE_ID, "--direction", "right", "--cwd", $acc.Dir, "--no-focus") + @(Get-EnvArgs $acc))
        if (-not $split.Ok) { Fail "Could not open a pane: $($split.Text)" }
        $p = $split.Json.result.pane.pane_id
        [void](Invoke-Herdr @("pane", "rename", $p, "login $($acc.Tool) $($acc.Name)"))
        [void](Invoke-Herdr @("pane", "run", $p, $acc.Login))
        Write-Output "LOGIN pane=$p tool=$($acc.Tool) account=$($acc.Name) folder=$($acc.Dir)"
        Write-Output "The owner finishes the sign-in in that pane (browser / code), then closes it (ctrl+b x)."
        Write-Output "Use it: herdr-agents start $Target -Account $($acc.Name)"
    }

    "accounts" {
        # Every separate account folder, per tool, and the profiles that can use it.
        $registry = Get-Registry
        $byTool = @{}
        foreach ($p in $registry.PSObject.Properties) {
            if ($p.Value.PSObject.Properties["account"] -and $p.Value.account) { $byTool[$p.Value.account.tool] += @($p.Name) }
        }
        Write-Output "Account folders: $accountsRoot"
        foreach ($tool in ($byTool.Keys | Sort-Object)) {
            $names = @(Get-ChildItem -LiteralPath (Join-Path $accountsRoot $tool) -Directory -ErrorAction SilentlyContinue | ForEach-Object { $_.Name })
            Write-Output ("{0,-9} accounts: {1,-30} profiles: {2}" -f $tool, $(if ($names) { $names -join ', ' } else { "(none - default sign-in only)" }), ($byTool[$tool] -join ', '))
        }
        $none = @($registry.PSObject.Properties | Where-Object { -not ($_.Value.PSObject.Properties["account"] -and $_.Value.account) } | ForEach-Object { $_.Name })
        if ($none) { Write-Output "No separate accounts: $($none -join ', ')" }
        Write-Output "New account: herdr-agents login <profile> -Account <name>   then: herdr-agents start <profile> -Account <name>"
    }

    "usage" {
        # Claude limits per account (claude-limits.ps1); the owner switches at 95% (HISTORY 24).
        Format-ClaudeLimits (Get-ClaudeLimits)
    }

    "handoff" {
        # A lead -> lead handoff file (tasks\HANDOFF-TEMPLATE.md) with the facts filled in;
        # the writer fills the TODO sections. Never overwrites a file.
        $template = Join-Path $PSScriptRoot "tasks\HANDOFF-TEMPLATE.md"
        $folder = if ($Cwd) { $Cwd } else { (Get-Location).ProviderPath }
        $out = if ($TaskFile) { $TaskFile } else { Join-Path $PSScriptRoot ("tasks\handoff-{0}.md" -f (Get-Date -Format "yyyyMMdd-HHmm")) }
        if (Test-Path -LiteralPath $out) { Fail "Exists already, not overwritten: $out" }
        $limits = @(Get-ClaudeLimits)
        $me = Get-ClaudeAccountOf $env:CLAUDE_CONFIG_DIR (Get-ClaudeAccountNames)
        $mine = $limits | Where-Object { $_.Account -eq $me } | Select-Object -First 1
        $next = Get-ClaudeNextAccount ($limits | Where-Object { $_.Account -ne $me })
        $why = if ($mine -and $mine.HasData) { "Account $me is at $($mine.FiveHour)% of its 5-hour limit (weekly $($mine.SevenDay)%). TODO: or another reason." } else { "TODO: why (limit / context full / owner asked)." }
        $agentRows = @(Get-LiveAgents | ForEach-Object { "| {0} | {1} | {2} | {3} | {4} |" -f $_.name, $_.agent, $_.agent_status, $_.pane_id, $_.cwd })
        $agentsText = if ($agentRows) { (@("| name | kind | state | pane | folder |", "|---|---|---|---|---|") + $agentRows) -join "`n" } else { "(none)" }
        $git = Invoke-Git $folder @("status", "--short", "--branch")
        $gitText = if ($git.Ok) { "Last commit: " + (Invoke-Git $folder @("log", "-1", "--oneline")).Text + "`n`n" + '```' + "`n" + $git.Text + "`n" + '```' } else { "(not a git folder)" }
        $hist = Join-Path $folder "HISTORY.md"
        $lastPhase = if (Test-Path -LiteralPath $hist) { (Select-String -LiteralPath $hist -Pattern '^## Phase' | Select-Object -Last 1).Line -replace '^## ', '' } else { "TODO" }
        $text = (Get-Content -LiteralPath $template -Raw).
            Replace("{{FROM}}", "Claude $me").Replace("{{TO}}", $(if ($next) { "Claude $($next.Account)" } else { "the next lead" })).
            Replace("{{TIME}}", (Get-Date -Format "yyyy-MM-dd HH:mm")).Replace("{{FOLDER}}", $folder).Replace("{{WHY}}", $why).
            Replace("{{LAST_PHASE}}", $lastPhase).Replace("{{AGENTS}}", $agentsText).Replace("{{GIT}}", $gitText).
            Replace("{{USAGE}}", ('```' + "`n" + ((Format-ClaudeLimits $limits) -join "`n") + "`n" + '```'))
        Set-Content -LiteralPath $out -Value $text -Encoding utf8
        Write-Output "HANDOFF $out"
        Write-Output "Fill every TODO, then the next lead starts with: herdr-agents start next   and reads that file."
    }

    "history" {
        $files = @(Get-ChildItem -LiteralPath $runsPath -Recurse -Filter *.md -ErrorAction SilentlyContinue | Sort-Object LastWriteTimeUtc, FullName -Descending)
        if (-not $files) { Write-Output "No runs saved yet ($runsPath)."; break }
        if ($Target) {
            $i = 0
            if (-not [int]::TryParse($Target, [ref]$i) -or $i -lt 1 -or $i -gt $files.Count) { Fail "history <n>: n from 1 (newest) to $($files.Count)" }
            Write-Output (Get-Content -LiteralPath $files[$i - 1].FullName -Raw)
            break
        }
        $i = 0
        foreach ($f in ($files | Select-Object -First 20)) {
            $i++
            $head = (Get-Content -LiteralPath $f.FullName -TotalCount 1) -replace '^# ', ''
            $taskLine = (Get-Content -LiteralPath $f.FullName | Select-Object -Skip 6 -First 1)
            if ($taskLine.Length -gt 70) { $taskLine = $taskLine.Substring(0, 70) + "..." }
            Write-Output ("{0,2}. {1} {2,-40} {3}" -f $i, $f.Directory.Name, $head, $taskLine)
        }
        Write-Output "Show one: herdr-agents history <n>    Folder: $runsPath"
    }

    "diff" {
        # Read-only review gate: what the agent changed in its folder.
        if (-not $Target) { Fail "diff needs an agent name." }
        $agent = Get-LiveAgents | Where-Object { $_.name -eq $Target } | Select-Object -First 1
        $entry = Get-State | Where-Object { $_.name -eq $Target -and (Get-RecordServer $_) -eq (Get-ServerIdentity) } | Select-Object -First 1
        $dir = if ($agent) { $agent.cwd } elseif ($entry) { $entry.cwd } else { $null }
        if (-not $dir) { Fail "No agent '$Target' running or recorded." }
        $top = Invoke-Git $dir @("rev-parse", "--show-toplevel")
        if (-not $top.Ok) { Fail "$dir is not in a git repository." }
        Write-Output "=== $Target in $($top.Text) ==="
        Write-Output (Invoke-Git $dir @("status", "--short", "--branch")).Text
        $base = if ($entry -and $entry.PSObject.Properties["base"] -and $entry.base) { $entry.base } else { $null }
        if ($base) {
            Write-Output "--- commits since $base ---"
            Write-Output (Invoke-Git $dir @("log", "--oneline", "$base..HEAD")).Text
            Write-Output "--- files changed since $base (committed + uncommitted) ---"
            Write-Output (Invoke-Git $dir @("diff", "--stat", $base)).Text
        } else {
            Write-Output "--- uncommitted changes ---"
            Write-Output (Invoke-Git $dir @("diff", "--stat", "HEAD")).Text
        }
        Write-Output "Full patch: git -C `"$dir`" diff $(if ($base) { $base } else { 'HEAD' })"
    }

    "watch" {
        if ($OpenPane) {
            # Open the board in a new pane below the focused one (ctrl+b a).
            $cur = (Invoke-Herdr @("pane", "current")).Json.result.pane
            if (-not $cur) { Fail "No focused pane." }
            $split = Invoke-Herdr @("pane", "split", $cur.pane_id, "--direction", "down", "--ratio", "0.7", "--cwd", $cur.cwd, "--no-focus")
            if (-not $split.Ok) { Fail "Could not open a pane: $($split.Text)" }
            [void](Invoke-Herdr @("pane", "run", $split.Json.result.pane.pane_id, "pwsh -NoProfile -File `"$PSCommandPath`" watch"))
            break
        }
        $labels = @{}
        $order = @{ blocked = 0; working = 1; done = 2; idle = 3; unknown = 4 }
        $colors = @{ blocked = "Red"; working = "Yellow"; done = "Green"; idle = "Gray"; unknown = "DarkGray" }
        do {
            $ws = Invoke-Herdr @("workspace", "list")
            if ($ws.Ok) { foreach ($w in $ws.Json.result.workspaces) { $labels[$w.workspace_id] = $w.label } }
            $list = Invoke-Herdr @("agent", "list")
            if (-not $Once) { Clear-Host }
            if (-not $list.Ok) { Write-Host "Herdr is not answering: $($list.Text)" -ForegroundColor Red }
            else {
                $all = @($list.Json.result.agents | Sort-Object { $order[[string]$_.agent_status] }, name)
                $counts = ($all | Group-Object agent_status | ForEach-Object { "$($_.Count) $($_.Name)" }) -join ", "
                Write-Host ("AGENTS {0}   {1}" -f (Get-Date -Format "HH:mm:ss"), $(if ($counts) { $counts } else { "none running" }))
                Write-Host ("{0,-9} {1,-14} {2,-9} {3,-16} {4}" -f "STATE", "NAME", "KIND", "SPACE", "FOLDER") -ForegroundColor DarkCyan
                foreach ($a in $all) {
                    $st = [string]$a.agent_status
                    $space = if ($labels[$a.workspace_id]) { $labels[$a.workspace_id] } else { $a.workspace_id }
                    $nm = if ($a.name) { $a.name } else { "-" }
                    $color = if ($colors[$st]) { $colors[$st] } else { "White" }
                    Write-Host ("{0,-9} {1,-14} {2,-9} {3,-16} {4}" -f $st.ToUpper(), $nm, $a.agent, $space, $a.cwd) -ForegroundColor $color
                }
                if ($all | Where-Object { $_.agent_status -eq "blocked" }) {
                    Write-Host "BLOCKED agents wait for YOU: open their pane and decide." -ForegroundColor Red
                }
            }
            if (-not $Once) { Write-Host "(refreshes every $IntervalSec s - Ctrl+C to close)" -ForegroundColor DarkGray; Start-Sleep -Seconds $IntervalSec }
        } while (-not $Once)
    }

    "read" {
        if (-not $Target) { Fail "read needs an agent name." }
        $source = if ($Visible) { "visible" } else { "recent-unwrapped" }
        $r = Invoke-Herdr @("agent", "read", $Target, "--source", $source, "--lines", $Lines.ToString())
        if (-not $r.Ok) { Fail $r.Text }
        Write-Output (Format-Screen $r.Text)
    }

    "status" {
        $mine = @(Get-State | Where-Object { (Get-RecordServer $_) -eq (Get-ServerIdentity) })
        foreach ($a in Get-LiveAgents) {
            $rec = $mine | Where-Object { $_.pane -eq $a.pane_id } | Select-Object -First 1
            $tag = if ($rec) { "started-by-tool" + $(if ($rec.PSObject.Properties["account"] -and $rec.account) { " account=$($rec.account)" } else { "" }) } else { "" }
            Write-Output ("{0,-14} {1,-9} {2,-8} pane={3,-7} {4} {5}" -f ($(if ($a.name) { $a.name } else { "(unnamed)" })), $a.agent, $a.agent_status, $a.pane_id, $a.cwd, $tag)
        }
    }

    "stop" {
        if (-not $Target) { Fail "stop needs an agent name." }
        $state = @(Get-State)
        $server = Get-ServerIdentity
        $entry = $state | Where-Object { $_.name -eq $Target -and (Get-RecordServer $_) -eq $server } | Select-Object -First 1
        if (-not $entry) { Fail "'$Target' was not started by this tool; close it yourself in Herdr." }
        $live = @(Get-LiveAgents)
        $agent = $live | Where-Object { $_.name -eq $Target } | Select-Object -First 1
        $pane = $entry.pane
        $gone = $false
        if ($agent) {
            if ($entry.PSObject.Properties['agentKind'] -and $entry.agentKind -ne $agent.agent) {
                Fail "Agent '$Target' now has another kind; not closing it."
            }
            $pane = $agent.pane_id
        } else {
            if ($live | Where-Object { $_.pane_id -eq $pane }) { Fail "Pane $pane now holds another agent; not closing it." }
            $p = Invoke-Herdr @("pane", "get", $pane)
            if (-not $p.Ok) {
                $list = Invoke-Herdr @("pane", "list")
                if (-not $list.Ok -or $null -eq $list.Json.result.panes -or ($list.Json.result.panes | Where-Object { $_.pane_id -eq $pane })) {
                    Fail "Could not inspect pane ${pane}: $($p.Text)"
                }
                $gone = $true
            }
        }
        if (-not $gone) {
            $r = Invoke-Herdr @("pane", "close", $pane)
            if (-not $r.Ok) {
                $list = Invoke-Herdr @("pane", "list")
                if (-not $list.Ok -or $null -eq $list.Json.result.panes -or ($list.Json.result.panes | Where-Object { $_.pane_id -eq $pane })) {
                    Fail "Could not close pane ${pane}: $($r.Text)"
                }
            }
        }
        Save-State @($state | Where-Object { $_.name -ne $Target -or (Get-RecordServer $_) -ne $server })
        Write-Output "STOPPED $Target (pane $pane closed or gone)"
        if ($entry.PSObject.Properties["branch"] -and $entry.branch) {
            # Never deleted automatically: the branch may hold work nobody has merged yet.
            Write-Output "Worktree kept: $($entry.cwd) (branch $($entry.branch)). After merging it:"
            Write-Output "  git -C `"$($entry.repo)`" worktree remove `"$($entry.cwd)`""
        }
    }
}
