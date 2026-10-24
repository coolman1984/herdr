# Copy the master skills in .\skills to ~/.claude/skills (Claude Code loads them there).
#   pwsh -File sync-skills.ps1          copy (backs up a changed installed copy first)
#   pwsh -File sync-skills.ps1 -Check   only report differences; exit 1 when any
[CmdletBinding()]
param([switch]$Check)

$ErrorActionPreference = "Stop"
$source = Join-Path $PSScriptRoot "skills"
$target = Join-Path $env:USERPROFILE ".claude\skills"
$drift = 0

foreach ($skill in Get-ChildItem -LiteralPath $source -Directory) {
    $from = Join-Path $skill.FullName "SKILL.md"
    $to = Join-Path $target "$($skill.Name)\SKILL.md"
    $same = (Test-Path -LiteralPath $to) -and
        ((Get-FileHash -LiteralPath $from).Hash -eq (Get-FileHash -LiteralPath $to).Hash)
    if ($same) {
        Write-Output "OK      $($skill.Name)"
        continue
    }
    $drift++
    if ($Check) {
        Write-Output "DIFFERS $($skill.Name)  (run sync-skills.ps1)"
        continue
    }
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $to) | Out-Null
    if (Test-Path -LiteralPath $to) {
        $backup = "$to.$(Get-Date -Format yyyyMMdd-HHmmss).bak"
        Copy-Item -LiteralPath $to -Destination $backup
        Write-Output "BACKUP  $backup"
    }
    Copy-Item -LiteralPath $from -Destination $to
    Write-Output "COPIED  $($skill.Name)"
}
if ($Check -and $drift) { exit 1 }
