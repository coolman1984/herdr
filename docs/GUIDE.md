# The complete Herdr guide (this PC)

Everything this setup can do, from opening the app to letting Claude run a team of
agents. Herdr = the official 0.9.5 app; everything marked **(ours)** was built in this
folder around it.

Contents: 1 Start · 2 Words · 3 Keys · 4 Folders and layouts · 5 Agents ·
6 Claude leads the team · 7 Notifications · 8 Sessions and machines ·
9 Daily recipes · 10 Tricks · 11 When something goes wrong · 12 Safety

---

## 1. Start Herdr

| Way | What happens |
|---|---|
| Double-click **Herdr** (Desktop or Start menu) **(ours)** | Windows Terminal opens Herdr in `D:\WORK\Software Development\GitHub` |
| Type `herdr` in File Explorer's address bar **(ours)** | Herdr opens in the folder you are looking at |
| `Win + R` -> `herdr` **(ours)** | Herdr opens in the GitHub folder |
| `herdr` in any terminal | Herdr opens in that terminal's folder |

**Closing the window does not stop anything.** Herdr runs a background server: the
agents keep working. Open Herdr again and you are back where you were. Your spaces,
tabs and panes are saved (`session.json`) and come back if the server stops (the
programs inside them start fresh).

Stop everything for real: `herdr server stop` (kills every agent - rarely needed).

## 2. The words

| Word | Meaning |
|---|---|
| **Space** (workspace) | One project. Listed in the sidebar on the left. |
| **Tab** | A page inside a space (tabs bar at the top). |
| **Pane** | One terminal inside a tab. Split a tab into several panes. |
| **Agent** | An AI CLI running in a pane: Claude Code, Codex, Cline, OpenCode, Grok, Antigravity... Herdr recognises it and shows its state. |
| **State** | `working` (busy), `blocked` (waits for YOU: approval or a question), `done` (finished, not yet looked at), `idle` (ready), `unknown`. |
| **Prefix** | `Ctrl+B`. Press it, let go, then press the key. |

## 3. Every key

Press `Ctrl+B`, let go, then the key. `Ctrl+B` `?` shows them inside the app.

### Added by this setup (ours)

| Keys | Action |
|---|---|
| `F` | Pick a folder (Windows dialog) -> new space there |
| `Shift+F` | Same, and start Claude Code in it |
| `T` | Same, with a team: Claude Code left + Codex (GPT-6.1-Sol) right |
| `Shift+V` | Split the pane into 4 equal columns, same folder |
| `Shift+B` | **Layout builder** window (section 4) |
| `A` | **Agent board** in a pane below: every agent, blocked first in red |
| `Shift+S` | **Spaces side by side** / separate again (section 4) |

### Herdr's own

| Keys | Action |
|---|---|
| `V` / `-` | Split side by side / split stacked |
| `X` | Close the pane |
| `Z` | Zoom the pane to full size and back |
| `R` | Resize mode (arrows), `Esc` to leave |
| `H` `J` `K` `L` (or arrows) | Move to the pane left / down / up / right |
| `Shift+H/J/K/L` | Swap the pane with its neighbour |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `C` | New tab |
| `N` / `P` | Next / previous tab |
| `1`..`9` | Go to tab 1..9 |
| `Shift+T` | Rename the tab |
| `Shift+X` | Close the tab |
| `Shift+N` | New space |
| `Shift+W` | Rename the space |
| `Shift+D` | Close the space |
| `W` | Space picker |
| `G` | Go-to navigator: jump to any space, tab or agent |
| `O` | Jump to the agent that just notified you |
| `Shift+P` | Rename the pane |
| `[` | Copy mode (select text with the keyboard) |
| `E` | Open the pane's scrollback in your editor |
| `B` | Hide / show the sidebar |
| `S` | Settings |
| `Shift+R` | Reload the config |
| `Q` | Detach: close the window, everything keeps running |
| `Shift+G` | Herdr's own "new git worktree" - **avoid**, it crashed the server once; use `herdr-agents start -Worktree` instead (LESSONS #19) |

The mouse works too: click a pane to focus it, right-click a space, tab or pane for
its menu, scroll to read back.

## 4. Folders and layouts

### One project
`Ctrl+B` `F` -> pick the folder -> a new space opens there. `Shift+F` adds Claude,
`T` adds Claude + Codex. The dialog starts in the last folder you picked.

### Columns
- `Ctrl+B` `V`: two panes side by side.
- `Ctrl+B` `Shift+V`: four equal columns, all in the current folder.
- Pressing `V` repeatedly halves the pane each time (1/2, 1/4, 1/8) - use `Shift+V`
  or the layout builder for equal columns.

### Several spaces side by side (`Ctrl+B` `Shift+S`)
Herdr shows one space at a time. To watch e.g. `Digital- Factory`, `MR.Park` and
`opening-nerp-tcode` together:
1. `Ctrl+B` `Shift+S` -> a window lists every space, all ticked. Untick what you do not
   want; the top one becomes the left column.
2. **Side by side** -> a new space `side by side` opens with one equal column per space.
   Each space's panes move in (a space with 2 panes gets both, stacked in its column);
   the agents keep running and keep their names. The emptied spaces leave the sidebar.
3. Done? `Ctrl+B` `Shift+S` -> **Separate again**: every group goes back to its own space
   with its old name, and `side by side` closes itself.

Without the window: `pwsh -File herdr-spaces.ps1 -Spaces "MR.Park,opening-nerp-tcode"` and
`pwsh -File herdr-spaces.ps1 -Separate`. Only the active tab of each space moves.

### The layout builder (`Ctrl+B` `Shift+B`)
A window where you choose:
- **Columns** 1-6, equal widths.
- **Open in** a new tab in this space, or a new space; **Name** for it.
- For **each column**: its **folder** (type it or **Browse...**), what it **Runs**:
  - `PowerShell (empty)`
  - `New: claude / codex / cline / deepseek / opencode / grok / agy` + optional **model**
  - `Running: <agent>` - an agent already running anywhere is **moved** into this column
    and keeps its conversation (its old space closes if it was the only pane there).
- **Preset** / **Save preset...**: keep a layout under a name ("morning", "review") and
  open it next time with one pick. Running agents are saved as an empty PowerShell.

It remembers the last layout you opened. Without the window:
`pwsh -File herdr-layout.ps1 -Preset morning`.

## 5. Agents

### Profiles (`agents.json`) (ours)

| Profile | CLI | Default model |
|---|---|---|
| `claude` | Claude Code | claude-opus-5-5 |
| `codex` | Codex | gpt-6.1-sol |
| `cline` | Cline, any provider | Cline's choice |
| `deepseek` | Cline on DeepSeek | chosen at `cline auth deepseek` |
| `opencode`, `grok`, `agy` | their CLIs | their default |

All start with **tool approval ON**. Cline is forced to `--auto-approve false`.

### First-time sign-ins (you, once)
- DeepSeek / other Chinese models through Cline: in any pane run `cline auth deepseek`
  and paste YOUR key there (never into a chat).
- Codex asks **"Trust this folder?"** once per repository - answer in its pane.

### Reading an agent at a glance
The sidebar shows each agent's state. `Ctrl+B` `A` opens the board; `Ctrl+B` `G` jumps to
any agent; `Ctrl+B` `O` jumps to the one that just notified you.

## 6. Claude leads the team (ours)

Inside Herdr, in a Claude Code pane, **just ask in plain words**:
- "Start Codex and DeepSeek, give both this bug, compare their fixes."
- "Let Codex implement the login fix on its own branch, then review its diff."
- "Ask Codex what this function does and summarise."
- "Open 3 columns: Codex in project A, DeepSeek in B, move my running Claude into the third."

Claude uses the `herdr-team` skill, which runs `herdr-agents`. You can run it yourself:

| Command | Does |
|---|---|
| `herdr-agents tools` | Which agents and models exist on this PC |
| `herdr-agents status` | Who runs, their state, pane and folder |
| `herdr-agents watch` | The same as a live board (`-Once` = print once) |
| `herdr-agents start codex` | Codex in a new pane next to you |
| `... -Model <m>` / `-Provider <p>` / `-Name reviewer` / `-Direction down` / `-Cwd <dir>` | Choose model, provider, name, side, folder |
| `herdr-agents start codex -Worktree fix-login` | Codex on its **own git branch and folder**, in a new space - your main copy stays untouched |
| `herdr-agents ask codex "<task>"` | Send a task, wait, print the answer |
| `herdr-agents ask codex -TaskFile task.md` | Long task from a file |
| `herdr-agents fanout "<task>" -Agents codex,deepseek` | Same task to several agents at once, one report |
| `herdr-agents read codex` (`-Visible`) | Its transcript (or the screen now) |
| `herdr-agents diff codex` | What it changed: commits since its base branch, files (read-only) |
| `herdr-agents history` / `history 3` | Every task and answer saved / show one |
| `herdr-agents stop codex` | Close an agent this tool started (worktrees are kept) |

Answers: `DONE` (finished), `BLOCKED` (waits for you - look at its pane), `UNCONFIRMED`
(answered but Herdr never saw it working - check it is complete), `FAILED`, `NOT READY`
(needs a sign-in or model choice in its pane). Details: [DELEGATION.md](DELEGATION.md).

### After a worktree agent finishes
1. `herdr-agents diff <name>` - read what changed; run the tests in its folder.
2. Merge yourself: `git merge <branch>` in your main checkout.
3. `herdr-agents stop <name>`, then the `git worktree remove ...` line it prints.

## 7. Notifications

- **Windows notification** when an agent in a space or tab you are NOT looking at
  finishes or needs you (setting `[ui.toast] delivery = "system"`, ours).
- **Sound** for the same events (Herdr default).
- The sidebar marks `done` agents until you look at them.
- `Ctrl+B` `O` jumps straight to the agent behind the last notification.

## 8. Sessions and other machines

- **Named sessions**: `herdr --session lab` is a completely separate Herdr (own spaces,
  own server). Good for experiments; `herdr --session lab server stop` ends it.
- **Detach / attach**: `Ctrl+B` `Q` closes the window; `herdr` brings it back.
- **Remote**: `herdr --remote user@host` attaches to Herdr on another machine over SSH;
  `herdr machine ...` saves machines so Claude can control agents there too.
- **Update**: `herdr update` (check `herdr status` afterwards).

## 9. Daily recipes

| Goal | Do |
|---|---|
| Start a work day | `Ctrl+B` `Shift+B` -> pick your preset -> Open layout |
| New project, me + Claude | Explorer -> project folder -> type `herdr` -> `Ctrl+B` `Shift+F` |
| Second opinion | Ask Claude: "fanout this question to Codex and DeepSeek and compare" |
| Safe code change by another agent | "Codex: implement X on its own branch" -> review `diff` -> merge |
| Cheap bulk work | DeepSeek through Cline drafts; Claude checks |
| See who needs me | `Ctrl+B` `A` (board) or wait for the Windows notification |
| Find what an agent said yesterday | `herdr-agents history` |
| Watch four repos | Layout builder: 4 columns, a folder each |

## 10. Tricks

- **Zoom** a pane (`Ctrl+B` `Z`) to read a long answer, press again to return.
- **Rename** panes (`Shift+P`) and tabs (`Shift+T`) - names show in the board and navigator.
- **Move a running agent** to another place: layout builder -> `Running: <agent>`.
- **Long answer?** Ask the agent to write Markdown to a file and reply with the path.
- **Copy text**: `Ctrl+B` `[` for keyboard selection, or select with the mouse.
- **Scrollback in an editor**: `Ctrl+B` `E`.
- **Two Codex at once**: `start codex` twice -> `codex`, `codex-2`.
- **A different model just for one run**: `start codex -Model <model>`.
- **Agents in the background**: close the window - they keep working; the Windows
  notification tells you when they are done.

## 11. When something goes wrong

| Problem | Fix |
|---|---|
| `herdr` not found in an old terminal | Use the full path `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe` or open a new terminal |
| A key does nothing | `herdr server reload-config`, then `pwsh -File check.ps1` |
| Window closed unexpectedly | Double-click Herdr - the layout comes back; restart the agents |
| Agent shows BLOCKED | Open its pane, read the question, decide yourself |
| Agent NOT READY | Sign in / choose a model in its pane |
| Is everything installed right? | `pwsh -File check.ps1` in the HERDR folder |

More: [SETUP.md](SETUP.md) (install, update, restore).

## 12. Safety (what the tools never do)

- Never type or store your API keys or passwords - you type them in the agent's pane.
- Never approve a tool or answer a question for you; BLOCKED waits for you.
- Agents start with approval ON; no `--yolo` / auto-approve unless you ask.
- Close only what they started; never `herdr server stop` on their own.
- Never delete a worktree or merge a branch for you unless you ask.
- Back up Herdr's config before every change (`config.toml.<time>.bak`).
