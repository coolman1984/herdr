// HerdrLauncher - one-click start of Herdr with NO console window flash.
//
// What: starts Herdr in Windows Terminal in the chosen folder, plus the toolkit's pane
//       HUD (herdr-hud.ps1) hidden in a headless conhost. Folder rules match Herdr.cmd:
//       the argument if it is an existing folder, else the current directory, else
//       (when that is System32 or missing) the toolkit folder.
// Why:  Herdr.cmd is a batch file, so Windows creates a console that flashes on screen
//       for a moment every time it runs (e.g. typing "herdr" in Explorer's address bar).
//       This is a GUI-subsystem exe (/target:winexe), so Windows never creates a console.
// Build: pwsh -NoProfile -File launcher\build.ps1
//        (compiles with the .NET Framework 4 C# compiler; no NuGet, no SDK needed)
// Test: launcher\HerdrLauncher.exe --dry-run [folder]
//        does everything except starting processes; writes %TEMP%\herdr-launcher-dryrun.txt
//
// C# 5 (the .NET Framework 4 csc.exe): no string interpolation, no ?. operator.

using System;
using System.Diagnostics;
using System.IO;
using System.Windows.Forms;

static class Program
{
    static string DefaultFolder { get { return ToolkitFolder(); } }

    [STAThread]
    static int Main(string[] args)
    {
        bool dryRun = args.Length >= 1 &&
            string.Equals(args[0], "--dry-run", StringComparison.OrdinalIgnoreCase);

        string argFolder = null;
        if (dryRun)
        {
            if (args.Length >= 2)
            {
                argFolder = args[1];
            }
        }
        else if (args.Length >= 1)
        {
            argFolder = args[0];
        }

        string folder = ResolveFolder(argFolder);

        string toolkit = ToolkitFolder();
        string hudScript = Path.Combine(toolkit, "herdr-hud.ps1");
        string herdr = Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
            "Programs", "Herdr", "bin", "herdr.exe");

        string hudCommand = "conhost.exe --headless pwsh -NoProfile -File " + Quote(hudScript);
        string herdrCommand = "wt.exe -d " + Quote(folder) + " --title Herdr " + Quote(herdr);

        if (dryRun)
        {
            string hudValue = File.Exists(hudScript) ? hudCommand : "(herdr-hud.ps1 not found)";
            File.WriteAllText(
                Path.Combine(Path.GetTempPath(), "herdr-launcher-dryrun.txt"),
                "folder=" + folder + Environment.NewLine +
                "hud=" + hudValue + Environment.NewLine +
                "herdr=" + herdrCommand + Environment.NewLine);
            return 0;
        }

        if (!File.Exists(herdr))
        {
            MessageBox.Show(
                "Herdr is not installed at " + herdr +
                ". Run install.cmd in the HERDR folder first.",
                "Herdr", MessageBoxButtons.OK, MessageBoxIcon.Error);
            return 1;
        }

        if (File.Exists(hudScript))
        {
            Start("conhost.exe",
                  "--headless pwsh -NoProfile -File " + Quote(hudScript),
                  folder, true);
        }

        if (!Start("wt.exe", "-d " + Quote(folder) + " --title Herdr " + Quote(herdr), folder, false))
        {
            Start(herdr, "", folder, false);
        }
        return 0;
    }

    // 1. argument if it is an existing folder; 2. current directory;
    // 3. when that is System32 (same as Herdr.cmd) or missing -> the toolkit folder,
    //    or the user profile if that is missing too.
    static string ResolveFolder(string argFolder)
    {
        string folder = null;
        if (!string.IsNullOrEmpty(argFolder) && Directory.Exists(argFolder))
        {
            folder = Path.GetFullPath(argFolder);
        }
        if (folder == null)
        {
            folder = Environment.CurrentDirectory;
        }
        if (!Directory.Exists(folder) ||
            SameFolder(folder, Environment.GetFolderPath(Environment.SpecialFolder.System)))
        {
            if (Directory.Exists(DefaultFolder))
            {
                return DefaultFolder;
            }
            return Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
        }
        return folder;
    }

    static bool SameFolder(string a, string b)
    {
        return string.Equals(a.TrimEnd('\\'), b.TrimEnd('\\'), StringComparison.OrdinalIgnoreCase);
    }

    // Folder of this exe; if it is named "launcher", use its parent (the HERDR folder).
    static string ToolkitFolder()
    {
        string dir = Path.GetDirectoryName(typeof(Program).Assembly.Location);
        if (string.Equals(Path.GetFileName(dir), "launcher", StringComparison.OrdinalIgnoreCase))
        {
            dir = Path.GetDirectoryName(dir);
        }
        return dir;
    }

    // Wraps a path in double quotes for a command line, doubling every backslash in the
    // run of backslashes at the END of the path (D:\ -> "D:\\", C:\x\ -> "C:\\x\\").
    // Without this, CommandLineToArgvW reads a trailing \" as an escaped quote, so
    // "D:\" would parse as the literal D: followed by a broken command line.
    static string Quote(string s)
    {
        int trailing = 0;
        for (int i = s.Length - 1; i >= 0 && s[i] == '\\'; i--)
        {
            trailing++;
        }
        string backslashes = new string('\\', trailing);
        return "\"" + s.Substring(0, s.Length - trailing) + backslashes + backslashes + "\"";
    }

    // Starts a process; returns false instead of throwing.
    static bool Start(string file, string arguments, string folder, bool hidden)
    {
        try
        {
            ProcessStartInfo psi = new ProcessStartInfo();
            psi.FileName = file;
            psi.Arguments = arguments;
            psi.WorkingDirectory = folder;
            psi.UseShellExecute = !hidden;
            if (hidden)
            {
                psi.CreateNoWindow = true;
                psi.WindowStyle = ProcessWindowStyle.Hidden;
            }
            Process.Start(psi);
            return true;
        }
        catch (Exception)
        {
            return false;
        }
    }
}
