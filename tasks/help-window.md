# Task for Codex: the Herdr help window (herdr-help.ps1)

You work in `D:\WORK\Software Development\GitHub\HERDR` (a personal PowerShell toolkit around
the Herdr terminal app). Create ONE new file: `herdr-help.ps1`. Do not change any other file.
Do not run git. Do not install anything. PowerShell 7 (`pwsh`) only.

## What it does
A pop-up window (Windows Forms) that the owner opens with a Herdr key. It shows every
keyboard shortcut and the user guide.

- Window title "Herdr - shortcuts and guide", about 900x640, CenterScreen, TopMost, resizable,
  font Segoe UI 9.5, `AutoScaleMode = "Dpi"`. Esc closes it. Activate it on Shown.
- A TabControl with two tabs:
  1. **Shortcuts**: a search TextBox at the top (placeholder "Type to filter, e.g. split,
     agent, folder") and a ListView (Details view, full-row select, grid lines) with columns
     **Keys** (180 px), **What it does** (rest), **From** (90 px: "this setup" or "Herdr").
     Typing filters rows (case-insensitive, any column). Group or sort so "this setup" rows
     come first. Double-click a row: nothing needed.
  2. **Guide**: shows `docs\GUIDE.md` rendered as HTML. Use
     `(ConvertFrom-Markdown -Path <file>).Html` inside a `System.Windows.Forms.WebBrowser`
     with a short inline CSS (system font, max-width 860px, readable tables with borders,
     code in Consolas). If the WebBrowser control fails for any reason, fall back to a
     read-only multiline TextBox with the raw Markdown.
- Bottom buttons: **Open guide in browser** (write the same HTML to
  `$env:TEMP\herdr-guide.html` and `Start-Process` it) and **Close**.

## Where the shortcuts come from
- "This setup": parse `config\keys.toml` at runtime. It contains blocks like:
  ```
  # Open folder: ctrl+b then f -> Windows folder picker -> new workspace there.
  [[keys.command]]
  key = "prefix+f"
  ```
  For each `[[keys.command]]`, take the `key = "..."` value and the comment line(s) directly
  above the block (join them, strip `# `). Show the keys as `Ctrl+B, then F`:
  `prefix` -> `Ctrl+B, then`, `shift+x` -> `Shift+X`, letters upper-case.
  Description = the comment text after the first ": " if present, else the whole comment.
- "Herdr": a table inside the script with these (Keys | What it does):
  Ctrl+B, then ? | Herdr's own key list ; Ctrl+B, then V | Split side by side ;
  Ctrl+B, then - | Split stacked ; Ctrl+B, then X | Close the pane ; Ctrl+B, then Z | Zoom the
  pane / back ; Ctrl+B, then R | Resize mode (arrows, Esc) ; Ctrl+B, then H/J/K/L or arrows |
  Move to the pane left/down/up/right ; Ctrl+B, then Shift+H/J/K/L | Swap with the neighbour ;
  Ctrl+B, then Tab / Shift+Tab | Next / previous pane ; Ctrl+B, then C | New tab ;
  Ctrl+B, then N / P | Next / previous tab ; Ctrl+B, then 1..9 | Go to tab 1..9 ;
  Ctrl+B, then Shift+T | Rename the tab ; Ctrl+B, then Shift+X | Close the tab ;
  Ctrl+B, then Shift+N | New space ; Ctrl+B, then Shift+W | Rename the space ;
  Ctrl+B, then Shift+D | Close the space ; Ctrl+B, then W | Space picker ;
  Ctrl+B, then G | Go to any space, tab or agent ; Ctrl+B, then O | Jump to the agent that
  notified you ; Ctrl+B, then Shift+P | Rename the pane (its name shows on the border and in the
  sidebar) ; Ctrl+B, then [ | Copy mode ; Ctrl+B, then E | Open the scrollback in an editor ;
  Ctrl+B, then B | Hide / show the sidebar ; Ctrl+B, then S | Settings ;
  Ctrl+B, then Shift+R | Reload the config ; Ctrl+B, then Q | Detach (agents keep running).
- Also add three "this setup" command rows (From = "command"):
  `herdr-agents status` | Who is running and their state ;
  `herdr-agents ask <name> "<task>"` | Give an agent a task and wait for the answer ;
  `herdr-agents history` | Every task and answer saved.

## Also
- A `-List` switch: no window; print the shortcut table as text (Keys, What it does, From),
  one row per line, then exit 0. This is how it will be tested.
- Use `$PSScriptRoot` for paths. Errors in the window: show a MessageBox, do not crash.
- Header comment at the top like the other scripts in the folder (look at herdr-layout.ps1
  for the style: what it does, how to run it, bound to "ctrl+b i").

## Check before you answer
1. `pwsh -NoProfile -File herdr-help.ps1 -List` prints all rows (the 8 keys from keys.toml
   plus the Herdr and command rows).
2. The file parses: `pwsh -NoProfile -Command "[System.Management.Automation.Language.Parser]::ParseFile('herdr-help.ps1',[ref]$null,[ref]$e) | Out-Null; $e.Count"` prints 0.

Answer with: DONE, the number of rows `-List` printed, and anything you could not do.
