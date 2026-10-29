# Task: the SMD work routine table (read only, same rules as before)

The owner wants the full work routine of the SMD projects: every UI screen that runs, when, and
where its output is saved. Source of truth: `OneClick_Tools/SMD_VD/tool.json` (and its `data\`
recordings, site.json, logs, state.json, .bat files), the SMD Part Work Calendar logic
(`gmes_workcal`, the `_engine` copy), and any other SMD screens in the Departments_Daily pool or
the GMES Automation schedules (`schedules/`). Read files only; run nothing live.

## Answer (English, tables first)
1. **Routine table**, one row per UI per job:
   job | UI code | screen title | G-MES base/site (SEEG/HQ, workspace) | day rule (D-1/D/D+1 and how
   working days are decided) | schedule (task name, days, time, every-N-min) | export type
   (xlsx/csv) | exact save folder (full path incl. `<CODE> - <title>` subfolder) | file name pattern |
   filters/division/inputs from the recording | .bat that runs it.
2. **Daily timeline**: what runs at what time, Sun-Thu, including the 30-minute Drop Dashboard loop
   (start/end).
3. **Where everything else is**: logs, run reports (`data\logs\batches`), state.json, lock, recordings
   folder, screenshots/diagnostics.
4. **How to check it ran**: which file/log proves each job succeeded today, and the last real run
   you can find in the logs (date + result) - quote the log line.
5. **Gaps**: anything missing, overlapping with other tools/schedules, or unverified (mark it).

Then write `READY - waiting for the next task.` and stop.
