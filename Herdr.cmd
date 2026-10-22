@echo off
rem One-click Herdr: opens Windows Terminal with Herdr in the current folder
rem (typing "herdr" in File Explorer's address bar opens it in that folder).
rem Uses the installer's stable alias, so "herdr update" never breaks this launcher.
setlocal
set "HERDR_EXE=%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe"
set "START_DIR=%CD%"
rem Started from Win+R or a shortcut without a folder: use the GitHub folder.
if /i "%START_DIR%"=="%SystemRoot%\System32" set "START_DIR=D:\WORK\Software Development\GitHub"
if not "%~1"=="" set "START_DIR=%~1"

if not exist "%HERDR_EXE%" (
    echo Herdr is not installed. Run install.cmd in this folder first.
    pause
    exit /b 1
)

rem Pane HUD (name, path, model, context on every pane border); one copy only.
start "" /min pwsh -NoProfile -WindowStyle Hidden -File "%~dp0herdr-hud.ps1"

where wt.exe >nul 2>&1
if %errorlevel%==0 (
    start "" wt.exe -d "%START_DIR%" --title Herdr "%HERDR_EXE%"
) else (
    cd /d "%START_DIR%"
    start "Herdr" "%HERDR_EXE%"
)
