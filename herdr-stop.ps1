# Stop Herdr for real: asks first, then stops the background server and every agent and
# terminal in it. Closing Herdr's window never does this - agents keep working (LESSONS #36).
# Bound to ctrl+b shift+q; also the Desktop shortcut "Stop Herdr".
#
#   pwsh -STA -File herdr-stop.ps1           ask, then stop
#   pwsh -File herdr-stop.ps1 -Status        no window: print what is running
[CmdletBinding()]
param([switch]$Status)

$ErrorActionPreference = "Stop"
$herdr = Join-Path $env:LOCALAPPDATA "Programs\Herdr\bin\herdr.exe"

function Get-Running {
    $out = & $herdr agent list 2>$null
    if ($LASTEXITCODE -ne 0) { return [pscustomobject]@{ Ok = $false; Agents = @() } }
    [pscustomobject]@{ Ok = $true; Agents = @(($out | ConvertFrom-Json).result.agents) }
}

function Stop-HerdrServer {
    $output = & $herdr server stop 2>&1
    if ($LASTEXITCODE -ne 0) { throw "Herdr server stop failed: $($output -join "`n")" }
    # status server does not start a server; verify its explicit stopped status.
    for ($attempt = 0; $attempt -lt 10; $attempt++) {
        $output = & $herdr status server --json 2>&1
        $code = $LASTEXITCODE
        $text = $output -join "`n"
        if ($code -eq 0 -and (($text | ConvertFrom-Json).status -in @('stopped', 'not_running'))) {
            Write-Output 'Herdr server stopped.'
            return
        }
        Start-Sleep -Milliseconds 200
    }
    throw "Could not verify Herdr server stopped: $text"
}

$running = Get-Running
$agents = $running.Agents
if ($Status) {
    if (-not $running.Ok) { Write-Output "Herdr server is not reachable."; exit 0 }
    Write-Output "Herdr is running with $($agents.Count) agent(s):"
    foreach ($a in $agents) { Write-Output ("  {0,-12} {1,-8} {2}" -f $(if ($a.name) { $a.name } else { $a.agent }), $a.agent_status, $a.cwd) }
    exit 0
}

Add-Type -AssemblyName System.Windows.Forms
$owner = New-Object System.Windows.Forms.Form -Property @{ TopMost = $true; ShowInTaskbar = $false; Opacity = 0; Width = 1; Height = 1 }
$owner.Show()
$owner.Activate()
if (-not $running.Ok) {
    [System.Windows.Forms.MessageBox]::Show($owner, "Herdr server is not reachable.", "Stop Herdr", "OK", "Information") | Out-Null
    exit 0
}
$working = @($agents | Where-Object { $_.agent_status -in @("working", "blocked") })
$list = ($agents | ForEach-Object { "  - {0} ({1}): {2}" -f $(if ($_.name) { $_.name } else { $_.agent }), $_.agent_status, (Split-Path -Leaf ([string]$_.cwd).TrimEnd('\')) }) -join "`n"
$text = "Stop Herdr completely?`n`nThis closes EVERY space, pane and agent:`n$list`n"
if ($working) { $text += "`n$($working.Count) agent(s) are still WORKING or waiting for you - their current task stops.`n" }
$text += "`nYour layout is saved and comes back next time; Claude and Codex resume their conversations, the task that was running must be sent again.`n`n(To just close the window and keep everything running, close it normally or press Ctrl+B then Q.)"
$answer = [System.Windows.Forms.MessageBox]::Show($owner, $text, "Stop Herdr", "YesNo", "Warning", "Button2")
if ($answer -ne "Yes") { exit 0 }
try { Stop-HerdrServer } catch {
    [System.Windows.Forms.MessageBox]::Show($owner, $_.Exception.Message, "Stop Herdr", "OK", "Error") | Out-Null
    exit 1
} finally { $owner.Dispose() }
