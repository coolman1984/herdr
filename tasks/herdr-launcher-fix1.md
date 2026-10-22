# Review fix 1 for HerdrLauncher (lead's tests: 16/20; this is the real defect)

Change only `launcher\HerdrLauncher.cs`, then rebuild with `launcher\build.ps1`.

**Bug:** a folder ending in a backslash breaks the command line. From your own `--dry-run`:
- `wt.exe -d "D:\" --title Herdr "...herdr.exe"`
- `wt.exe -d "C:\some\folder\" --title Herdr ...`

Windows (`CommandLineToArgvW`) reads `\"` as an escaped quote, so Windows Terminal receives
`D:" --title Herdr ...` as the folder. This happens for real when Herdr is opened from a drive
root in File Explorer.

**Fix:** add `static string Quote(string s)`: wrap `s` in double quotes, and DOUBLE every
backslash in the run of backslashes at the END of `s` (`D:\` -> `"D:\\"`,
`C:\x\` -> `"C:\x\\"`, `C:\x` -> `"C:\x"`). Use it for every path you put on a command line:
the `wt -d` folder, the herdr.exe path, the HUD script path. Keep everything else the same.

**Check, then answer:** rebuild; run `launcher\HerdrLauncher.exe --dry-run "D:\"` and
`--dry-run` with a real folder ending in `\` (e.g. `D:\WORK\Software Development\GitHub\HERDR\docs\`),
using `Start-Process -Wait`; paste both dry-run files.
Answer: DONE + what changed + the two dry-run files.
