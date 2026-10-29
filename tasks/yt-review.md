# Task: review Yousef-Transportation (Trip Orders) - READ ONLY, report only

Folder: I:\Yousef-Transportation (Python server `server/app.py` + web UI `js/` `css/`, Windows build
`tools/build_windows.py` with Nuitka + Inno Setup `installer/trip-orders.iss`). Read its CLAUDE.md first.
Users are non-technical dispatchers; data must never be lost (see CLAUDE.md "Never").

DO NOT change any file except the report. DO NOT commit, push, or touch git. No browser tools.

## Part 1 - Review (findings ordered by severity)
Find real bugs, data-loss risks, security problems, race conditions, and edge cases
(empty data, huge data, Arabic/RTL text, dates/time zones, port in use, second instance,
crash mid-write, disk full, antivirus/locked files, offline, no admin rights).
Focus first on: `server/app.py`, `server/store.py`, `server/journal.py`, `server/backup.py`,
`server/sync.py`, `server/auth.py`, `server/excel_io.py` / `xlsx_*.py`, `server/training.py`.
For each finding: file:line, what goes wrong, a concrete trigger, the fix. Prove with a command
or test where you can; write "not verified" where you cannot. Never state a guessed cause as fact.

## Part 2 - Performance
Measure, do not guess. Startup time of the server (import cost, startup backup, demo data),
slow request paths, large-table cost, front-end load (bundle size, fonts, render blocking).
Give numbers (before) and the specific change that would help (expected gain).

## Part 3 - Best way to ship a real desktop app
Owner wants an .exe that opens like a professional app: its OWN window, not a browser tab,
and opens FAST. Today `main()` in `server/app.py` runs an HTTP server and calls `webbrowser.open`
(about lines 1650-1700), and the exe is a console program ("Keep this window open").
Compare, for THIS code base (Python server already exists, Windows 10/11, offline, non-admin installs):
 a) Edge/Chrome `--app=http://localhost:PORT` window (no extra dependency)
 b) pywebview + WebView2 (works with Nuitka?)
 c) Tauri / Electron wrapper
 d) anything better you find
Criteria: opens speed, exe size, build risk with Nuitka, WebView2 runtime present on Win10/11,
no console window, tray/close behaviour (closing the window must stop or keep the server clearly),
single instance (second launch focuses the existing window), antivirus false positives, file
downloads/Excel export/print working, Arabic RTL. Recommend ONE option with the exact code
changes (function names, files, config keys) and how to test it. Do not implement it yet.

## Answer format
Write the full report to `I:\herdr\tasks\yt-review-report.md` (your sandbox is I:\herdr; the project folder is read-only for you, so run measurements with read-only commands or copies under I:\herdr\tasks\tmp).
Reply with: top 10 findings (one line each), the performance numbers, the recommended exe option
in 5 lines, and anything under "doubts". Be brief.
