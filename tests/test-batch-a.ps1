# Offline regression tests. -SourceRoot also runs the same fixtures against HEAD copies.
param([string]$SourceRoot = (Split-Path -Parent $PSScriptRoot))
$ErrorActionPreference = 'Stop'
$script:pass = 0; $script:fail = 0
function Check([string]$Name, [scriptblock]$Test) {
    try { if (& $Test) { $script:pass++; return } } catch { }
    $script:fail++; Write-Output "FAIL $Name"
}

# Import definitions only; never execute a script's initialization or main code.
$script:asts = @{}
foreach ($file in 'herdr-agents.ps1', 'claude-limits.ps1', 'check.ps1') {
    $errors = $null
    $ast = [System.Management.Automation.Language.Parser]::ParseFile(
        (Join-Path $SourceRoot $file), [ref]$null, [ref]$errors)
    if ($errors) { throw "Cannot parse $file" }
    $script:asts[$file] = $ast
    foreach ($f in $ast.FindAll({ param($x)
        $x -is [System.Management.Automation.Language.FunctionDefinitionAst]
    }, $true)) {
        . ([scriptblock]::Create($f.Extent.Text))
    }
}
$script:branches = @{}
$switch = $script:asts['herdr-agents.ps1'].Find({ param($x)
    $x -is [System.Management.Automation.Language.SwitchStatementAst]
}, $false)
foreach ($clause in $switch.Clauses) {
    $body = $clause.Item2.Extent.Text
    # Preserve production exit codes without terminating the test host.
    $offset = $clause.Item2.Extent.StartOffset
    foreach ($e in ($clause.Item2.FindAll({ param($x)
        $x -is [System.Management.Automation.Language.ExitStatementAst]
    }, $true) | Sort-Object { $_.Extent.StartOffset } -Descending)) {
        $replacement = 'throw ("TEST_EXIT:" + [string](' + $e.Pipeline.Extent.Text + '))'
        $body = $body.Remove($e.Extent.StartOffset - $offset, $e.Extent.Text.Length).
            Insert($e.Extent.StartOffset - $offset, $replacement)
    }
    $script:branches[$clause.Item1.Value] = [scriptblock]::Create($body.Substring(1, $body.Length - 2))
}
# Checker fixtures execute only the named section, with all dependencies mocked.
$checkText = $script:asts['check.ps1'].Extent.Text
function Get-CheckSection([string]$From, [string]$To) {
    $start = $checkText.IndexOf($From)
    [scriptblock]::Create($checkText.Substring($start, $checkText.IndexOf($To, $start) - $start))
}
$script:profiles = Get-CheckSection '# 3. Agent profiles' '# 4. Key bindings'
$script:keys = Get-CheckSection '# 4. Key bindings' '# 4b. Settings'
$script:docs = Get-CheckSection '# 7. Documentation' 'foreach ($file in Get-ChildItem'

function Reply([string]$Status = 'done', [string]$ErrorCode = '') {
    [pscustomobject]@{ Ok = -not $ErrorCode; Text = $(if ($ErrorCode) { $ErrorCode } else { 'answer' })
        Json = [pscustomobject]@{ result = [pscustomobject]@{ agent = [pscustomobject]@{ agent_status = $Status } }
            error = [pscustomobject]@{ code = $ErrorCode } } }
}
function Reset-Fixture {
    $script:records = @(); $script:savedState = $null; $script:liveAgents = @()
    $script:calls = [System.Collections.Generic.List[object]]::new()
    $script:files = @{}; $script:tick = 0; $script:closeFails = $false
    $script:startFails = $false; $script:missingPane = $false; $script:missingFolder = $false
    $script:startIdentity = @()
    $script:readError = ''; $script:statuses = @('done'); $script:waitStatus = 'done'; $script:waitError = ''
    $script:listFails = $false
    $script:paneCwd = 'C:\pane'; $script:created = @(); $script:children = @()
    $script:registryPath = 'C:\fake\agents.json'; $script:runsPath = 'C:\fake\runs'
    $script:root = 'C:\fake'; $script:herdr = 'C:\fake\herdr.exe'
    $script:accountsRoot = 'C:\fake\accounts'; $script:Lines = 200; $script:TimeoutSec = 20
    $script:ClaudeLimitThreshold = 95; $script:claudeLimitCache = @{}
    $script:badMessages = @(); $script:fails = 0
    $script:registryJson = '{"agents":{"codex":{"kind":"codex","command":"fake-cli","modelFlag":"-m","args":[]},"claude":{"kind":"claude","command":"fake-cli","modelFlag":"--model","args":[]}},"aliases":{"2040":{"profile":"claude","name":"claude-2040"}}}'
    $script:planJson = '{"agents":[{"agent":"codex","name":"coder"}]}'
    $env:HERDR_ENV = '1'; $env:HERDR_PANE_ID = 'caller'; $env:HERDR_ACTIVE_PANE_ID = ''
    $env:HERDR_ACTIVE_PANE_CWD = ''
    $env:HERDR_SOCKET_PATH = 'C:\Fake\Session.sock'
}
function Fail([string]$Message) { throw "ERROR: $Message" }
function Get-Date { param([string]$Format)
    $now = [datetime]'2026-10-23T12:00:00'
    if ($Format) { return $now.ToString($Format) }
    $now.AddSeconds($script:tick)
}
function Start-Sleep { param($Seconds, $Milliseconds) $script:tick += [int]$Seconds }
function Test-Path { param($LiteralPath, $PathType)
    if ($LiteralPath -eq $herdr) { return $false }
    if ($LiteralPath -like '*worktrees*') { return $false }
    if ($LiteralPath -eq 'missing' -or ($script:missingFolder -and $LiteralPath -eq 'C:\missing')) { return $false }
    if ($LiteralPath -like '*second*') { return $false }
    if ($LiteralPath -like '*.md') { return $script:files.ContainsKey($LiteralPath) }
    $true
}
function Resolve-Path { param($LiteralPath)
    [pscustomobject]@{ Path = $(if ($LiteralPath -eq 'relative') { 'C:\full\relative' } else { $LiteralPath }); ProviderPath = $(if ($LiteralPath -eq 'relative') { 'C:\full\relative' } else { $LiteralPath }) }
}
function Get-Location { [pscustomobject]@{ ProviderPath = 'C:\caller' } }
function Get-Command { param($Name, $ErrorAction) [pscustomobject]@{ Name = $Name } }
function Get-Content { param($LiteralPath, [switch]$Raw, $Encoding, $TotalCount)
    if ($LiteralPath -eq $registryPath -or $LiteralPath -like '*agents.json') { return $script:registryJson }
    if ($LiteralPath -eq 'plan.json') { return $script:planJson }
    $script:files[$LiteralPath]
}
function Get-ChildItem { param($LiteralPath, $Filter, [switch]$File, [switch]$Recurse, $ErrorAction) $script:children }
function New-Item { param($ItemType, [switch]$Force, $Path, $ErrorAction)
    if ($ItemType -eq 'File') {
        if ($script:files.ContainsKey($Path) -and -not $Force) { throw [System.IO.IOException]::new('File exists') }
        $script:files[$Path] = ''; $script:created += $Path
    }
}
function Set-Content { param($LiteralPath, $Value, $Encoding) $script:files[$LiteralPath] = $Value }
function Get-State { $script:records }
function Save-State($Items) { $script:savedState = @($Items) }
function Get-LiveAgents { $script:liveAgents }
function Invoke-Git([string]$Dir, [string[]]$Arguments) {
    [pscustomobject]@{ Ok = ($Arguments[0] -ne 'show-ref'); Text = $(if ($Arguments[0] -eq 'rev-parse' -and $Arguments[1] -eq '--show-toplevel') { 'C:\repo' } else { 'main' }) }
}
function Invoke-Herdr([string[]]$Arguments) {
    $script:calls.Add(@($Arguments))
    switch ("$($Arguments[0]) $($Arguments[1])") {
        'agent get' {
            $status = $script:statuses[0]
            if ($script:statuses.Count -gt 1) { $script:statuses = $script:statuses[1..($script:statuses.Count - 1)] }
            return Reply $status
        }
        'agent wait' { return Reply $script:waitStatus $script:waitError }
        'agent read' { return Reply '' $script:readError }
        'agent start' {
            $script:liveAgents += $script:startIdentity
            return Reply '' $(if ($script:startFails) { 'agent_not_ready' } else { '' })
        }
        'pane get' { return [pscustomobject]@{ Ok = -not $script:missingPane; Text = 'pane lookup failed'
            Json = @{ result = @{ pane = @{ pane_id = $Arguments[2]; cwd = $script:paneCwd; workspace_id = 'w1' } } } } }
        'pane list' { return [pscustomobject]@{ Ok = -not $script:listFails; Json = @{ result = @{ panes = @($(if (-not $script:missingPane) { @{ pane_id = 'old' } })) } } } }
        'pane close' { return Reply '' $(if ($script:closeFails) { 'server_error' } else { '' }) }
        default { return [pscustomobject]@{ Ok = $true; Text = 'screen'; Json = @{ result = @{ pane = @{ pane_id = 'new' }; root_pane = @{ pane_id = 'space' } } } } }
    }
}
function pwsh { $script:childArgs = @($args); $global:LASTEXITCODE = 0; 'STARTED name=coder' }
function Bad([string]$m) { $script:fails++; $script:badMessages += $m }
function Ok([string]$m) { }
function Warn([string]$m) { }
function Run-Branch([string]$Branch, [hashtable]$Values = @{}) {
    $Target = 'codex'; $Name = ''; $Cwd = ''; $Pane = ''; $Worktree = ''; $Account = ''
    $Model = ''; $Provider = ''; $Effort = ''; $Direction = 'right'; $PlanFile = 'plan.json'
    $LeadShare = 0.4; $Overflow = 'tab'; $PSCommandPath = 'fake-script'
    foreach ($k in $Values.Keys) { Set-Variable -Name $k -Value $Values[$k] }
    $output = @(); $problem = ''
    try { $output = @(switch ('fixture') { default { . $script:branches[$Branch] } }) } catch { $problem = $_.Exception.Message }
    [pscustomobject]@{ Output = $output -join "`n"; Error = $problem }
}
function Make-Record([string]$Server = 'c:\fake\session.sock', [string]$Kind = 'codex') {
    [pscustomobject]@{ name = 'codex'; pane = 'old'; server = $Server; agentKind = $Kind; cwd = 'C:\pane' }
}
function Read-Limits([string]$RateLimits) {
    $script:files['C:\fake\usage.json'] = '{"config_dir":"","cost":21,"rate_limits":' + $RateLimits + '}'
    $script:children = @([pscustomobject]@{ FullName = 'C:\fake\usage.json'; LastWriteTime = (Get-Date); LastWriteTimeUtc = (Get-Date).ToUniversalTime() })
    @(Get-ClaudeLimits)
}
function Get-ClaudeAccountNames { [pscustomobject]@{ Name = 'main'; Main = $true; Dir = '' } }

$oldEnv = @{}
foreach ($k in 'HERDR_ENV','HERDR_PANE_ID','HERDR_ACTIVE_PANE_ID','HERDR_ACTIVE_PANE_CWD','HERDR_SOCKET_PATH') { $oldEnv[$k] = [Environment]::GetEnvironmentVariable($k) }
try {
    Check 'M1 done-then-working waits' {
        Reset-Fixture; $script:statuses = @('idle', 'working')
        $r = Complete-Ask coder (Reply)
        $wait = @($script:calls | Where-Object { $_[1] -eq 'wait' })
        $r.Outcome -eq 'DONE' -and $wait.Count -eq 1 -and [int]$wait[0][-1] -lt 20000
    }
    Check 'M1 confirmation detects blocked' {
        Reset-Fixture; $script:statuses = @('blocked'); $script:waitStatus = 'blocked'
        $r = Complete-Ask coder (Reply); $r.Outcome -eq 'BLOCKED' -and $r.Exit -eq 3
    }
    Check 'M1 non-idle answer is unconfirmed' {
        Reset-Fixture; $script:readError = 'agent_not_idle'
        $r = Complete-Ask coder (Reply)
        $r.Outcome -eq 'UNCONFIRMED' -and $r.Exit -eq 0 -and $r.Text -eq 'UNCONFIRMED: coder is still working; read it later with: herdr-agents read coder'
    }
    Check 'M1 failed confirmation wait cannot report DONE' {
        Reset-Fixture; $script:statuses = @('working'); $script:waitError = 'timeout'
        $r = Complete-Ask coder (Reply)
        $r.Outcome -eq 'FAILED' -and $r.Exit -eq 1 -and $r.Text -match 'timeout' -and $r.Text -notmatch 'DONE:'
    }
    Check 'M1 settled state is checked for ten seconds' {
        Reset-Fixture; $r = Complete-Ask coder (Reply)
        $r.Outcome -eq 'DONE' -and $script:tick -eq 10 -and @($script:calls | Where-Object { $_[1] -eq 'get' }).Count -eq 5
    }
    Check 'F11 blocked rejection shows screen' {
        Reset-Fixture; $r = Complete-Ask coder (Reply '' 'agent_blocked')
        $r.Outcome -eq 'BLOCKED' -and $r.Exit -eq 3 -and $r.Text -match 'screen of coder' -and @($script:calls | Where-Object { $_ -contains 'visible' }).Count -eq 1
    }
    Check 'F12 read failure cannot report DONE' {
        Reset-Fixture; $script:readError = 'server_error'; $r = Complete-Ask coder (Reply)
        $r.Outcome -eq 'FAILED' -and $r.Exit -eq 1 -and $r.Text -match 'server_error' -and $r.Text -notmatch 'DONE:'
    }
    Check 'F02 same-second runs are exclusive' {
        Reset-Fixture
        $one = Save-Run coder first DONE first-answer; $two = Save-Run coder second DONE second-answer
        $one -ne $two -and $two -like '*-2.md' -and $script:files[$one] -match 'first-answer' -and $script:created.Count -eq 2
    }
    Check 'F02 history sorts collision suffixes by time' {
        Reset-Fixture
        $script:children = @(1, 2, 10 | ForEach-Object {
            $p = "C:\fake\runs\120000-coder-$_.md"; $script:files[$p] = "# run$_`n`nTime: t`n`n## Task`n`ntask"
            [pscustomobject]@{ FullName = $p; LastWriteTimeUtc = ([datetime]'2026-10-23').AddSeconds($_); Directory = @{ Name = 'day' } }
        })
        (Run-Branch history @{ Target = '1' }).Output -match '^# run10'
    }
    Check 'M2 worktree splits beside caller' {
        Reset-Fixture; $r = Run-Branch start @{ Worktree = 'feature'; Cwd = 'C:\repo' }
        $split = @($script:calls | Where-Object { $_[1] -eq 'split' })
        $beside = $r.Error -eq '' -and $split.Count -eq 1 -and $split[0][2] -eq 'caller' -and $split[0] -contains '--no-focus' -and $split[0][4] -eq 'right' -and $r.Output -notmatch 'space=new'
        Reset-Fixture; $env:HERDR_ENV = ''; $env:HERDR_PANE_ID = ''
        $outside = Run-Branch start @{ Worktree = 'feature'; Cwd = 'C:\repo' }
        $beside -and $outside.Error -eq '' -and $outside.Output -match 'space=new'
    }
    Check 'M3 missing cwd fails before creation' {
        Reset-Fixture; $script:missingFolder = $true; $r = Run-Branch start @{ Cwd = 'C:\missing' }
        $r.Error -eq 'ERROR: Folder not found: C:\missing' -and $script:calls.Count -eq 0
    }
    Check 'M3 relative cwd is resolved' {
        Reset-Fixture; $r = Run-Branch start @{ Cwd = 'relative' }
        $r.Error -eq '' -and $script:savedState[0].cwd -eq 'C:\full\relative'
    }
    Check 'M4 start records server and kind' {
        Reset-Fixture; $r = Run-Branch start
        $recorded = $r.Error -eq '' -and $script:savedState[0].server -ceq 'c:\fake\session.sock' -and $script:savedState[0].agentKind -eq 'codex'
        Reset-Fixture; $script:startFails = $true; $null = Run-Branch start
        $recorded -and $script:savedState.Count -eq 1
    }
    Check 'F01 another server cannot close a pane' {
        Reset-Fixture; $script:records = @(Make-Record 'other'); $r = Run-Branch stop
        $r.Error -like 'ERROR:*not started*' -and $script:calls.Count -eq 0 -and $null -eq $script:savedState
    }
    Check 'F01 same-name records on other servers survive' {
        Reset-Fixture; $script:records = @((Make-Record), (Make-Record 'other'))
        $null = Run-Branch start
        $script:savedState.Count -eq 2 -and @($script:savedState | Where-Object server -eq other).Count -eq 1
    }
    Check 'F01 legacy default records stay on default server' {
        Reset-Fixture; $legacy = [pscustomobject]@{ name = 'codex'; pane = 'old'; cwd = 'C:\pane' }
        $script:records = @($legacy); $wrongServer = Run-Branch stop
        $refused = $wrongServer.Error -like 'ERROR:*not started*' -and $script:calls.Count -eq 0
        $env:HERDR_SOCKET_PATH = ''; $r = Run-Branch stop
        $refused -and $r.Error -eq '' -and $script:savedState.Count -eq 0
    }
    Check 'F03 failed supplied-pane start saves nothing' {
        Reset-Fixture; $script:startFails = $true; $r = Run-Branch start @{ Pane = 'supplied' }
        $r.Error -eq 'TEST_EXIT:2' -and $null -eq $script:savedState
    }
    Check 'R1 not-ready agent in supplied pane is recorded' {
        Reset-Fixture; $script:startFails = $true
        $script:startIdentity = @([pscustomobject]@{ name = 'worker'; pane_id = 'supplied'; agent = 'codex' })
        $r = Run-Branch start @{ Pane = 'supplied'; Name = 'worker' }
        $r.Error -eq 'TEST_EXIT:2' -and
            $script:savedState.Count -eq 1 -and $script:savedState[0].name -eq 'worker' -and
            $script:savedState[0].pane -eq 'supplied' -and $script:savedState[0].cwd -eq 'C:\pane' -and
            $script:savedState[0].server -ceq 'c:\fake\session.sock' -and $script:savedState[0].agentKind -eq 'codex'
    }
    Check 'R1 failed start without identity saves nothing' {
        $safe = $true
        foreach ($identity in @($null,
            [pscustomobject]@{ name = 'worker'; pane_id = 'other'; agent = 'codex' },
            [pscustomobject]@{ name = 'other'; pane_id = 'supplied'; agent = 'codex' })) {
            Reset-Fixture; $script:startFails = $true
            $script:startIdentity = @($identity | Where-Object { $null -ne $_ })
            $r = Run-Branch start @{ Pane = 'supplied'; Name = 'worker' }
            $safe = $safe -and $r.Error -eq 'TEST_EXIT:2' -and $null -eq $script:savedState
        }
        $safe
    }
    Check 'M4 moved live agent closes current pane' {
        Reset-Fixture; $script:records = @(Make-Record)
        $script:liveAgents = @([pscustomobject]@{ name = 'codex'; pane_id = 'moved'; agent = 'codex' })
        $r = Run-Branch stop; $close = @($script:calls | Where-Object { $_[1] -eq 'close' })
        $r.Error -eq '' -and $close[0][2] -eq 'moved'
    }
    Check 'M4 changed kind is refused' {
        Reset-Fixture; $script:records = @(Make-Record)
        $script:liveAgents = @([pscustomobject]@{ name = 'codex'; pane_id = 'moved'; agent = 'claude' })
        $r = Run-Branch stop
        $r.Error -like 'ERROR:*' -and @($script:calls | Where-Object { $_[1] -eq 'close' }).Count -eq 0 -and $null -eq $script:savedState
    }
    Check 'F03 replacement occupant is refused' {
        Reset-Fixture; $script:records = @(Make-Record)
        $script:liveAgents = @([pscustomobject]@{ name = 'owner'; pane_id = 'old'; agent = 'claude' })
        $r = Run-Branch stop
        $r.Error -eq 'ERROR: Pane old now holds another agent; not closing it.' -and $null -eq $script:savedState
    }
    Check 'F16 failed close retains ownership' {
        Reset-Fixture; $script:records = @(Make-Record); $script:closeFails = $true
        $r = Run-Branch stop; $r.Error -like 'ERROR:*Could not close*' -and $null -eq $script:savedState
    }
    Check 'F16 confirmed gone removes without closing' {
        Reset-Fixture; $script:records = @(Make-Record); $script:missingPane = $true
        $r = Run-Branch stop
        $r.Error -eq '' -and $script:savedState.Count -eq 0 -and @($script:calls | Where-Object { $_[1] -eq 'close' }).Count -eq 0
    }
    Check 'F16 failed discovery preserves ownership' {
        Reset-Fixture; $script:records = @(Make-Record); $script:missingPane = $true; $script:listFails = $true
        $r = Run-Branch stop
        $r.Error -like 'ERROR:*Could not inspect*' -and $null -eq $script:savedState -and @($script:calls | Where-Object { $_[1] -eq 'close' }).Count -eq 0
    }
    Check 'F13 alias names receive suffixes' {
        Reset-Fixture; $script:liveAgents = @([pscustomobject]@{ name = 'claude-2040' })
        $r = Run-Branch start @{ Target = '2040' }
        $suffixed = $r.Error -eq '' -and $script:savedState[0].name -eq 'claude-2040-2'
        $explicit = Run-Branch start @{ Target = '2040'; Name = 'claude-2040' }
        $suffixed -and $explicit.Error -like 'ERROR:*already running*'
    }
    Check 'F14 supplied pane records its cwd' {
        Reset-Fixture; $r = Run-Branch start @{ Pane = 'supplied' }
        $r.Error -eq '' -and $script:savedState[0].cwd -eq 'C:\pane'
    }
    Check 'F14 mismatched cwd fails before start' {
        Reset-Fixture; $r = Run-Branch start @{ Pane = 'supplied'; Cwd = 'C:\other' }
        $r.Error -eq "ERROR: -Cwd differs from pane supplied's folder C:\pane." -and @($script:calls | Where-Object { $_[1] -eq 'start' }).Count -eq 0
    }
    Check 'R2 active-pane cwd ignored with -Pane' {
        Reset-Fixture; $env:HERDR_PANE_ID = ''; $env:HERDR_ACTIVE_PANE_ID = 'focused'
        $env:HERDR_ACTIVE_PANE_CWD = 'C:\other'
        $r = Run-Branch start @{ Pane = 'supplied' }
        $r.Error -eq '' -and $script:savedState[0].cwd -eq 'C:\pane' -and
            @($script:calls | Where-Object { $_[1] -eq 'start' }).Count -eq 1 -and
            @($script:calls | Where-Object { $_[1] -eq 'split' }).Count -eq 0
    }
    Check 'F15 team preserves drive root' {
        Reset-Fixture; $script:paneCwd = 'D:\'; $r = Run-Branch team
        $split = @($script:calls | Where-Object { $_[1] -eq 'split' })
        $split[0][[array]::IndexOf($split[0], '--cwd') + 1] -eq 'D:\' -and $script:childArgs -contains 'D:\'
    }
    Check 'F29 start rejects uppercase before creation' {
        Reset-Fixture; $r = Run-Branch start @{ Name = 'Coder' }
        $r.Error -like 'ERROR: Name must match*' -and $script:calls.Count -eq 0
    }
    Check 'F29 team rejects uppercase before creation' {
        Reset-Fixture; $script:planJson = '{"agents":[{"agent":"codex","name":"Coder"}]}'
        $r = Run-Branch team; $r.Error -like 'ERROR: Name must match*' -and $script:calls.Count -eq 0
    }
    Check 'F09 usage names cumulative session cost' {
        Reset-Fixture; $rows = Read-Limits '{"five_hour":{"used_percentage":10}}'
        ((Format-ClaudeLimits $rows) -join "`n") -match 'session cost \(updated today\)'
    }
    Check 'F10 empty limits are unknown and ineligible' {
        Reset-Fixture; $rows = Read-Limits '{}'
        $unknown = $rows[0].HasData -and $null -eq (Get-ClaudeNextAccount $rows) -and ((Format-ClaudeLimits $rows) -join "`n") -match 'main\*\s+\?'
        Reset-Fixture; $expired = Read-Limits '{"five_hour":{"used_percentage":99,"resets_at":1}}'
        $unknown -and $expired[0].FiveHour -eq 0 -and (Get-ClaudeNextAccount $expired).Account -eq 'main'
    }
    Check 'F10 weekly-only limits are ineligible' {
        Reset-Fixture; $rows = Read-Limits '{"seven_day":{"used_percentage":20}}'
        $rows[0].HasData -and $null -eq (Get-ClaudeNextAccount $rows)
    }
    Check 'F10 nonnumeric five-hour value is unknown' {
        Reset-Fixture; $rows = Read-Limits '{"five_hour":{"used_percentage":"no"}}'
        $rows[0].HasData -and $null -eq $rows[0].FiveHour -and $null -eq (Get-ClaudeNextAccount $rows)
    }
    Check 'F24 threshold uses fractional percentages' {
        Reset-Fixture; $rows = Read-Limits '{"five_hour":{"used_percentage":94.6}}'
        $rows[0].FiveHour -eq 94.6 -and -not $rows[0].Limit -and (Get-ClaudeNextAccount $rows).Account -eq 'main'
    }
    foreach ($flags in @(@('-s','danger-full-access'), @('--sandbox=danger-full-access'), @('--sandbox','danger-full-access'), @('-s=danger-full-access'))) {
        Check "F06 rejects $($flags -join ' ')" {
            Reset-Fixture
            $script:registryJson = @{ agents = @{ codex = @{ kind = 'codex'; command = 'fake-cli'; modelFlag = '-m'; args = $flags } } } | ConvertTo-Json -Depth 5
            . $script:profiles
            $script:fails -gt 0 -and ($script:badMessages -join '') -match 'danger-full-access'
        }
    }
    Check 'F07 swapped commands fail per key' {
        Reset-Fixture
        $script:files['C:\fake\config\keys.toml'] = "[[keys.command]]`nkey = `"prefix+f`"`ntype = `"shell`"`ncommand = 'one'`n[[keys.command]]`nkey = `"prefix+t`"`ntype = `"shell`"`ncommand = 'two'"
        $script:files[(Join-Path $env:APPDATA 'herdr\config.toml')] = $script:files['C:\fake\config\keys.toml'].Replace("'one'", "'hold'").Replace("'two'", "'one'").Replace("'hold'", "'two'")
        . $script:keys; $script:fails -eq 2
    }
    Check 'F07 duplicate keys fail' {
        Reset-Fixture
        $block = "[[keys.command]]`nkey = `"prefix+f`"`ntype = `"shell`"`ncommand = 'one'`n"
        $script:files['C:\fake\config\keys.toml'] = $block
        $script:files[(Join-Path $env:APPDATA 'herdr\config.toml')] = $block + $block
        . $script:keys
        $duplicate = $script:fails -gt 0 -and ($script:badMessages -join '') -match 'duplicate'
        Reset-Fixture
        $script:files['C:\fake\config\keys.toml'] = $block
        $script:files[(Join-Path $env:APPDATA 'herdr\config.toml')] = "[[keys.command]] # comment`n command = `"one`" # command`n type = 'shell'`n key = 'prefix+f'"
        . $script:keys
        $duplicate -and $script:fails -eq 0
    }
    Check 'F07 duplicate master keys fail' {
        Reset-Fixture
        $block = "[[keys.command]]`nkey = `"prefix+f`"`ntype = `"shell`"`ncommand = 'one'`n"
        $script:files['C:\fake\config\keys.toml'] = $block + $block
        $script:files[(Join-Path $env:APPDATA 'herdr\config.toml')] = $block
        . $script:keys; $script:fails -gt 0 -and ($script:badMessages -join '') -match 'duplicate'
    }
    Check 'F07 command type must match' {
        Reset-Fixture
        $script:files['C:\fake\config\keys.toml'] = "[[keys.command]]`nkey = `"prefix+f`"`ntype = `"shell`"`ncommand = 'one'"
        $script:files[(Join-Path $env:APPDATA 'herdr\config.toml')] = $script:files['C:\fake\config\keys.toml'].Replace('"shell"','"send"')
        . $script:keys; $script:fails -eq 1
    }
    Check 'F08 every path in first cell is checked' {
        Reset-Fixture; $script:files['C:\fake\docs\FILES.md'] = '| `first`, `second` | description `ignored` |'
        . $script:docs; $script:fails -eq 1 -and ($script:badMessages -join '') -match 'second'
    }
} finally {
    foreach ($k in $oldEnv.Keys) { [Environment]::SetEnvironmentVariable($k, $oldEnv[$k]) }
}
Write-Output "PASS $script:pass/$($script:pass + $script:fail)"
if ($script:fail) { exit 1 }
