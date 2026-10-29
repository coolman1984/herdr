# One Go build/test at a time on this PC (two at once ran out of RAM, LESSONS #88 / Phase 50).
# Dot-source, then: $lock = Enter-GoLock; try { ... } finally { Exit-GoLock $lock }
# The lock is an exclusively opened file; Windows releases it when the holder exits or crashes.
function Enter-GoLock([int]$TimeoutMinutes = 30) {
    $path = Join-Path $PSScriptRoot 'go.lock'
    $deadline = [DateTime]::UtcNow.AddMinutes($TimeoutMinutes)
    $said = $false
    while ($true) {
        try { return [IO.File]::Open($path, 'OpenOrCreate', 'ReadWrite', 'None') }
        catch [IO.IOException] {
            if ([DateTime]::UtcNow -gt $deadline) { throw "Go lock busy for $TimeoutMinutes min: $path (another build or test is running)" }
            # Write-Host, not Write-Output: output would become part of the returned lock.
            if (-not $said) { Write-Host 'Waiting for another Go build/test to finish...'; $said = $true }
            Start-Sleep -Seconds 2
        }
    }
}
function Exit-GoLock($Lock) { if ($Lock) { $Lock.Dispose() } }
