# Task: SMD G-MES tools as a server package (build it, do not run it live)

The owner will copy the SMD tools to the G-MES server **106.139.69.145** (Windows Server, opened by
Remote Desktop) into `D:\GMES Automation\` - that folder already holds `GMES_Automation_server_package`
(+ .zip, 17 Oct), `SETUP_UI_AUTOMATION (copy to the server D-drive)`, `SETUP_WEEKLY_REPORTS`,
`test_runs`, `GMES_Tool_20261010_153117`, `opening-nerp-tcode`. The tools must run THERE on their
schedule and save to the SAME Data Hub paths they use now (`\\106.139.69.145\DataHub Shared Folder\...`
- on the server that share is local; decide UNC vs local path and justify it from the code/docs).

## Scope
SMD only: `OneClick_Tools/SMD_VD` (daily job + Drop Dashboard D every 30 min) and whatever it needs
(`_source`, the `_engine` copy - it holds click_tab / NEVER_WEEKDAYS that the root engine lacks -,
recordings `data\sites\smd-vd\screens`, site.json, popups.json, the SMD calendar logic).

## Do
1. Find how the server runs Python today (the existing `GMES_Automation_server_package` /
   `SETUP_UI_AUTOMATION` docs and build scripts in this repo: embedded Python? frozen exe? installed?)
   and reuse that way - the server has no internet; nothing may need pip at install time.
2. Build ONE self-contained folder (+ a zip of it) at
   `D:\WORK\Software Development\GitHub\Computer Use\opening-nerp-tcode\dist\SMD_Server_Package\`
   whose layout, when copied to `D:\GMES Automation\SMD_Tools\` on the server, just works:
   runner + engine + SMD tool.json + recordings + .bat files (Run / Check / Days / Status /
   Schedule_ON / Schedule_OFF, Drop Dashboard variants) with **relative** paths only (no
   `D:\WORK\...` or user-profile paths of this PC baked in - grep the package to prove it).
3. Scheduled tasks: the Schedule_ON .bat registers the same tasks/times (Sun-Thu 08:15 daily, Drop
   Dashboard every 30 min) for the server user; say which user/"run whether logged on" choice it needs
   and that the owner answers any password prompt himself.
4. `SERVER_SETUP.md` inside the package: copy steps, first sign-in (the OWNER signs in to G-MES on the
   server; the tool never stores or asks for passwords in files), Check.bat first, then Run.bat,
   then Schedule_ON; how to see it worked (log + files in the Data Hub folders); how to turn the old
   PC's / GMES Automation's duplicate schedules for the same screens OFF so files are not downloaded
   twice; how to roll back.
5. Also write your full SMD routine table (previous answer, all sections) to
   `...\opening-nerp-tcode\dist\SMD_Server_Package\SMD-ROUTINE.md`.

## Rules
- Write ONLY inside `dist\SMD_Server_Package\` (and its zip). Do not edit the repo's source, the
  OneClick_Tools folders or the owner's uncommitted changes. No git writes.
- Run nothing that signs in to G-MES, opens a browser, registers a scheduled task on THIS PC or touches
  the Data Hub share. You may run offline checks (python -m py_compile, `Days.bat`-style dry logic if it
  makes no network/browser call - say which).
- Never copy credentials, cookies, browser profiles, `gmes_credentials` data, logs with personal data or
  screenshots into the package. List every file in the package and its size.

## Answer
DONE | PARTIAL | BLOCKED; package path + zip size; file list; how Python runs on the server; the
exact paths it saves to; the offline checks you ran and their output; what the owner must do on the
server, step by step; doubts. Then `READY - waiting for the next task.`
