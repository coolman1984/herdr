# Build HerdrLauncher.exe - compile launcher\HerdrLauncher.cs into a windowless exe.
# Uses the .NET Framework 4 C# compiler (csc.exe, C# 5) found under
# $env:WINDIR\Microsoft.NET\Framework64\v4*; no NuGet, no Visual Studio, no dotnet SDK.
# /target:winexe = GUI subsystem, so Windows never opens a console for the launcher.
# Run: pwsh -NoProfile -File launcher\build.ps1   -> prints "Built <path>" or throws.

$ErrorActionPreference = "Stop"

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$source = Join-Path $here "HerdrLauncher.cs"
$outExe = Join-Path $here "HerdrLauncher.exe"

$pattern = Join-Path $env:WINDIR "Microsoft.NET\Framework64\v4*\csc.exe"
$csc = Get-ChildItem -Path $pattern -ErrorAction SilentlyContinue |
    Sort-Object -Property VersionInfo.FileVersion -Descending |
    Select-Object -First 1
if ($null -eq $csc) { throw "csc.exe not found under $pattern" }

$framework = Split-Path -Parent $csc.FullName
& $csc.FullName "/nologo" "/target:winexe" "/optimize+" "/out:$outExe" `
    "/reference:$framework\System.dll" `
    "/reference:$framework\System.Windows.Forms.dll" `
    "$source"
if ($LASTEXITCODE -ne 0) { throw "csc.exe failed with exit code $LASTEXITCODE" }

Write-Output "Built $outExe"
