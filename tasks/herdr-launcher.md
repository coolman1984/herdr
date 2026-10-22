# Task: build HerdrLauncher.exe - start Herdr with NO console window

## 1. Role and goal
You are a senior Windows/C# engineer working for the lead (Claude).
Goal: a tiny compiled Windows program `HerdrLauncher.exe` that opens Herdr in Windows
Terminal exactly like `Herdr.cmd` does, but without the console window that flashes for a
moment whenever a `.cmd` file runs.
Why: the owner types `herdr` in File Explorer's address bar; that runs `Herdr.cmd` and a
black console flashes. The lead will point the "herdr" App Path and the shortcuts at your exe.

## 2. Context (read these first)
- Folder: `D:\WORK\Software Development\GitHub\HERDR` (PowerShell toolkit around Herdr).
- Read `Herdr.cmd` - your program must do the same, step by step.
- Herdr: `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe`. Toolkit HUD script: `herdr-hud.ps1`
  in the HERDR folder (it already refuses to run twice).
- Compiler: `C:\WINDOWS\Microsoft.NET\Framework64\v4.0.30319\csc.exe` (C# 5, .NET Framework 4).
  No NuGet, no Visual Studio, no dotnet SDK.

## 3. Exact scope
- Create ONLY: `launcher\HerdrLauncher.cs`, `launcher\build.ps1`, and the built
  `launcher\HerdrLauncher.exe`.
- Do NOT change any other file, the registry, shortcuts or Herdr's config. Do NOT run git.
- Do NOT run `HerdrLauncher.exe` without arguments and do NOT start Windows Terminal or Herdr
  (that would open windows on the owner's screen) - use the test mode below.

## 4. Design (decided by the lead - follow it)
- C# console-less program: compile with `/target:winexe` (Windows GUI subsystem = 2), so
  Windows never creates a console for it. `[STAThread] static int Main(string[] args)`.
- References: `System.dll`, `System.Windows.Forms.dll` (only for an error MessageBox).
- Folder rule (same as Herdr.cmd):
  1. `args[0]` if given and it is an existing folder;
  2. else `Environment.CurrentDirectory`;
  3. if that equals the System32 folder (`Environment.SpecialFolder.System`, compare
     case-insensitive, trailing `\` ignored) or does not exist -> `D:\WORK\Software Development\GitHub`
     (if that does not exist: the user profile folder).
- Toolkit folder = the folder of the exe; if that folder is named `launcher`, use its parent.
- If herdr.exe is missing: `MessageBox` "Herdr is not installed at ... Run install.cmd in the
  HERDR folder first." (title "Herdr", error icon), return 1.
- Start the HUD hidden: `conhost.exe --headless pwsh -NoProfile -File "<toolkit>\herdr-hud.ps1"`,
  `WorkingDirectory` = folder, `UseShellExecute = false`, `CreateNoWindow = true`,
  `WindowStyle = Hidden`. Only if the script exists. Ignore failures.
- Start Herdr: `wt.exe -d "<folder>" --title Herdr "<herdr.exe>"` with `UseShellExecute = true`;
  if that throws (no Windows Terminal), start `herdr.exe` itself with WorkingDirectory = folder.
- One helper `static bool Start(string file, string arguments, string folder, bool hidden)`
  that returns false instead of throwing.
- **Test mode** `--dry-run [folder]`: do everything EXCEPT starting processes; instead write
  to `%TEMP%\herdr-launcher-dryrun.txt` three lines: `folder=<folder>`, `hud=<full command>`,
  `herdr=<full command>`; exit 0. (`--dry-run` may be followed by a folder argument.)
- Header comment at the top of the .cs: what it is, why (no console flash), how to build.
- `launcher\build.ps1`: finds csc.exe under `$env:WINDIR\Microsoft.NET\Framework64\v4*`,
  compiles `/nologo /target:winexe /optimize+ /out:<launcher>\HerdrLauncher.exe`
  `/reference:System.Windows.Forms.dll`, throws on failure, prints `Built <path>`. Header
  comment like the other .ps1 scripts in the folder.

## 5. Steps
1. Read `Herdr.cmd` and one other `.ps1` for the header style.
2. Write the two source files. 3. Run `pwsh -NoProfile -File launcher\build.ps1`.
4. Run the checks in section 6 and fix until they pass.

## 6. Definition of done
- [ ] Build prints `Built ...\HerdrLauncher.exe`, no warnings.
- [ ] PE subsystem is 2 (GUI): read the 16-bit value at offset (PE header + 0x5C):
      `$b=[IO.File]::ReadAllBytes('launcher\HerdrLauncher.exe'); $pe=[BitConverter]::ToInt32($b,0x3C); [BitConverter]::ToUInt16($b,$pe+0x5C)` -> `2`.
- [ ] `launcher\HerdrLauncher.exe --dry-run "C:\Windows\System32"` -> the file says
      `folder=D:\WORK\Software Development\GitHub`.
- [ ] `launcher\HerdrLauncher.exe --dry-run "D:\WORK\Software Development\GitHub\HERDR\docs"`
      -> `folder=` that docs folder; `hud=` contains `conhost.exe --headless pwsh` and
      `...\HERDR\herdr-hud.ps1` (the parent of `launcher`); `herdr=` contains
      `wt.exe -d "...\docs" --title Herdr` and the herdr.exe path.
  (A GUI exe returns at once in PowerShell: use `Start-Process -Wait` to get its exit code.)

## 7. Answer format
DONE | BLOCKED | PARTIAL
Changed: <files>
Checks: <each check from section 6 and its result, with the dry-run file contents>
Not done / doubts: <list or "none">
