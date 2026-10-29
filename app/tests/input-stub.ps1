# Offline input-box fixture. Never starts an AI CLI, tools, or network requests.
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
[Console]::Write("`e[2J`e[HOffline input transport fixture`r`n")
$rule = [string]::new([char]0x2500, 48)
[Console]::Write("$rule`r`n$([char]0x276f)$([char]0x00a0)owner draft`r`n$rule`r`nRECEIVED=0`r`n")
$received = 0
while ($true) {
    $pressed = [Console]::ReadKey($true)
    if ($pressed.Key -eq 'Escape') { break }
    $received++
    [Console]::Write("RECEIVED=$received`r`n")
}
