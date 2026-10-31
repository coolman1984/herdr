# The complete Herdr guide (this PC)

Everything this setup can do, from opening the app to letting Claude run a team of
agents. Herdr = the official 0.9.5 app; everything marked **(ours)** was built in this
folder around it.

Contents: 1 Start · 2 Words · 3 Keys · 4 Folders and layouts · 5 Agents ·
6 Claude leads the team · 7 Notifications · 8 Sessions and machines ·
9 Daily recipes · 10 Tricks · 11 When something goes wrong · 12 Safety ·
**13 How to do everything, step by step**

All shortcuts in one window: `Ctrl+B`, then `I` (search box + this guide).

---

## 1. Start Herdr

| Way | What happens |
|---|---|
| Double-click **Herdr** (Desktop or Start menu) **(ours)** | Windows Terminal opens Herdr in `I:\herdr` |
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
| `I` | **All shortcuts** (type to filter) and this guide in one window |
| `Shift+Q` | **Stop Herdr for real** (asks first): every agent and pane stops. Same as the Desktop icon **Stop Herdr** |
| `Shift+C` | **Switch the lead**: next free choice in agents.json (section 13.20) |

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
| `cline` | Cline Pass | cline-pass/glm-5.3 |
| `cline-flash` | Cline Pass | cline-pass/glm-5.3-flash |
| `deepseek` | Cline Pass | cline-pass/deepseek-v4-pro |
| `kimi` | Cline Pass | cline-pass/kimi-k3 |
| `opencode`, `grok`, `agy` | their CLIs | their default |

All run **without approval prompts** (your decision of 2026-10-22): Claude in auto mode,
Codex never asks but can only write inside its project folder, Cline auto-approve,
Antigravity auto mode, Grok always-approve. Change it in `agents.json` (`args`); a backup
of the old settings is `agents.json.bak`.

### First-time sign-ins (you, once)
- The four Cline Pass profiles: in any pane run `cline auth cline-pass` and finish
  the sign-in yourself. Direct providers remain available with `-Provider <id>` and
  `-Model <model>` after your own provider sign-in (e.g. `cline auth deepseek`).
- Codex asks **"Trust this folder?"** once per repository - answer in its pane.

### What every pane and the sidebar show (ours)
- **Pane top border:** `name · project path · model · ctx 33% (327k/1M) · out 245k`
  - name = the pane's own name (`Ctrl+B` `Shift+P` to rename), else its space's name
  - ctx = how full the agent's context window is (like Claude's and Codex's own status lines)
  - out = tokens the agent has written in this session
  - Codex shows model + effort (`gpt-6.1-sol medium`); Claude its model; others the model
    they were started with (Cline reads the model of its own session, below).
- **Sidebar, agents:** row 1 = state + the agent's own name + its space; row 2 = CLI and
  model; row 3 = context. In a "side by side" view each agent keeps the name of the space
  it came from.
- **Sidebar, spaces:** row 2 = the git branch of the space's folder (e.g. `main`) and
  arrows: `↑2` = 2 commits made here that are not pushed yet; `↓3` = 3 commits on GitHub
  you have not pulled yet. No arrow = in sync.
- Updated every 5 s by `herdr-hud.ps1`, which starts at sign-in and with Herdr. Claude's
  dollar cost appears after its status line writes that session's cost file under
  `%APPDATA%\herdr\claude-cost\`; it is the session's cumulative cost.
- **Cline panes** show the model of the session the pane is running and `in 12k · out 3.1k`:
  `in` = the fresh tokens the model read (input minus cache reads), `out` = the tokens it wrote
  this session. Cline keeps no context window in that data, so there is no `ctx` part. The numbers
  come from `%USERPROFILE%\.cline\data\sessions\<id>\<id>.json` (usage fields only; a Cline run on
  a separate account keeps its own under `%USERPROFILE%\.herdr-accounts\cline\<account>\`), found
  by the pane's own processes, else by the newest running session in the pane's folder.

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
| `herdr-agents start codex -Worktree fix-login` | Codex on its **own git branch and folder**, beside your pane (new space only outside Herdr) - your main copy stays untouched |
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

## 13. How to do everything, step by step

Every feature, as exact steps. "Press `Ctrl+B` `X`" means: press `Ctrl+B`, let go,
then press `X`. Commands can be typed in any pane (PowerShell).

### 13.1 Open Herdr
1. Double-click **Herdr** on the Desktop or in the Start menu - or in File Explorer
   go to a project folder, click the address bar, type `herdr`, press Enter.
2. Herdr opens in Windows Terminal. If it was already running you get your spaces back.
3. To leave without stopping anything: `Ctrl+B` `Q` (or close the window).
- All three ways start `launcher\HerdrLauncher.exe`: a windowless program, so no black console
  window flashes (it also starts the pane HUD, one copy only). `Herdr.cmd` does the same and
  stays as the fallback; `pwsh -File launcher\use-launcher.ps1 -Undo` points everything back at it.

### 13.2 Open a project
1. Press `Ctrl+B` `F` (empty terminal), `Ctrl+B` `Shift+F` (with Claude) or `Ctrl+B` `T`
   (Claude left + Codex right).
2. A folder dialog opens at the last folder you used. Choose the project, click
   **Select Folder**.
3. A new space named after the folder appears in the sidebar and is focused.

### 13.3 Split, move, resize, zoom, close panes
- Split side by side: `Ctrl+B` `V`. Split stacked: `Ctrl+B` `-`.
- Four equal columns in this folder: `Ctrl+B` `Shift+V`.
- Move between panes: `Ctrl+B` then `H` `J` `K` `L` (or arrows), or click the pane.
- Swap a pane with its neighbour: `Ctrl+B` `Shift+H/J/K/L`.
- Resize: `Ctrl+B` `R`, then arrows, `Esc` to finish.
- Zoom one pane to full size and back: `Ctrl+B` `Z`.
- Close the pane: `Ctrl+B` `X` (an agent in it stops).

### 13.4 Build a layout with a folder and an agent per column
1. `Ctrl+B` `Shift+B`.
2. **Columns**: 1-6. **Open in**: new tab here, or a new space. **Name**: what it is called.
3. For each column: type or **Browse...** a folder; in **Runs** pick `PowerShell (empty)`,
   `New: <agent>` (optional **model**), or `Running: <agent>` to move a running agent in.
4. Click **Open layout**.
5. To keep it: before opening, click **Save preset...** and give a name. Next time pick
   it from **Preset** and click **Open layout**.

### 13.5 See several spaces side by side, then separate them
1. `Ctrl+B` `Shift+S`. Every space is listed and ticked; untick what you do not want.
   The top one becomes the left column.
2. Click **Side by side**. A space `side by side` opens with one column per space; each
   pane keeps the name of the space it came from; agents keep running.
3. To undo: `Ctrl+B` `Shift+S` -> **Separate again**. Each goes back to its own space.

### 13.6 Rename things
- A pane (shows on its border and in the sidebar): `Ctrl+B` `Shift+P`, type, Enter.
- A tab: `Ctrl+B` `Shift+T`. A space: `Ctrl+B` `Shift+W`.

### 13.7 Read a pane's border and the sidebar
- Border: `name · path · model · $cost · ctx 33% (327k/1M) · out 245k`.
  `$cost` = Claude's cost so far (Claude only); `ctx` = how full its memory (context
  window) is; `out` = tokens it has written.
- Sidebar agents: state dot + name + space / CLI + model / context.
- Sidebar spaces: `main` = git branch; `↑2` = 2 commits not pushed; `↓3` = 3 commits on
  GitHub not pulled.
- The values refresh every 5 s. If a border shows nothing: run
  `pwsh -File "I:\herdr\herdr-hud.ps1"` once in any
  pane (it starts by itself at sign-in and with Herdr).

### 13.8 Start an agent yourself
- In a pane at the prompt type the CLI: `claude`, `codex`, `cline`, `opencode`, `grok`, `agy`.
- Or next to your pane, with a model/effort:
  `herdr-agents start codex -Model gpt-6.1-sol -Effort medium`
  `herdr-agents start claude -Model claude-opus-5-5 -Name reviewer`
  `herdr-agents start deepseek -Model <model-id>`
  Add `-Direction down` to open it below, `-Cwd <folder>` for another folder.

### 13.9 Let Claude lead other agents
1. Open Claude in Herdr (`Ctrl+B` `Shift+F`).
2. Ask in plain words, e.g. *"Start Codex on SOL 6.1 medium next to you, make it write X,
   watch its screen and check its work."*
3. Claude starts the agent in a pane next to it, sends the task, reads its screen, and
   checks the result. If the agent asks for approval, Claude shows you; you decide.

### 13.10 Give a task and get the answer (yourself)
1. `herdr-agents status` - see names (`codex`, `codex-2`, ...).
2. `herdr-agents ask codex "Explain what herdr-hud.ps1 does in 5 lines"`.
3. It waits and prints DONE + the answer (or BLOCKED: go to its pane and decide).
4. Long task: write it in a .md file, `herdr-agents ask codex -TaskFile task.md`.
5. See its screen any time: `herdr-agents read codex -Visible`.

### 13.11 Ask several agents the same thing
1. Start them (13.8), e.g. `codex` and `deepseek`.
2. `herdr-agents fanout "Which approach is safer, A or B? Why?" -Agents codex,deepseek`.
3. Both answer in parallel; one report is printed and saved.

### 13.12 Let an agent change code safely (own branch)
1. In the project: `herdr-agents start codex -Worktree fix-login -Cwd <repo folder>`.
   A pane opens beside you on branch `fix-login` in its own folder (a new space only
   when called outside Herdr).
2. Codex may ask "Trust this folder?" - answer it in its pane (once per repository).
3. Give the task: `herdr-agents ask codex "Fix the login timeout; run the tests"`.
4. Review: `herdr-agents diff codex` (commits and files changed).
5. Happy? In your main folder: `git merge fix-login`.
6. `herdr-agents stop codex`, then run the `git worktree remove ...` line it prints.

### 13.13 Find an old answer
`herdr-agents history` lists the last 20 tasks; `herdr-agents history 3` shows number 3.
Files: `%APPDATA%\herdr\agent-runs\<date>\`.

### 13.14 Watch all agents
- `Ctrl+B` `A` opens the live board below the current pane (blocked = red, on top).
  Close it with `Ctrl+C`, then `Ctrl+B` `X`.
- Away from Herdr: a Windows notification tells you when an agent in another space
  finishes or needs you; `Ctrl+B` `O` jumps to it.

### 13.15 Close Herdr without losing work, and stop it for real
- **Closing the window never stops anything.** Herdr's server runs on its own; agents keep
  working. Reopen Herdr (Desktop icon, `herdr` in Explorer) and they are still there.
  `Ctrl+B` `Q` does the same (detach).
- After a Windows restart or sign-out the server is gone; Herdr restores your spaces and
  resumes Claude and Codex conversations, but a task that was running must be sent again.
- **Stop everything for real:** `Ctrl+B` `Shift+Q`, or double-click **Stop Herdr** on the
  Desktop. A window lists every agent (and warns about working ones); **No** is the
  default. `pwsh -File herdr-stop.ps1 -Status` shows what is running.
- One agent this tool started: `herdr-agents stop <name>`. Any agent: exit it in its pane
  (`/exit` or `Ctrl+C`) or `Ctrl+B` `X`.

### 13.16 Use DeepSeek or another Chinese model (Cline)
1. Once: in any pane run `cline auth cline-pass` and finish the sign-in yourself.
2. `herdr-agents start deepseek` uses `cline-pass/deepseek-v4-pro` (or pick `New: deepseek`
   in the layout builder). `cline`, `cline-flash`, `kimi` use GLM-5.3, GLM-5.3 Flash,
   Kimi K3 through the same Cline Pass sign-in. Cline starts with auto-approve (your setting).
3. For direct DeepSeek instead: sign in yourself with `cline auth deepseek`, then
   `herdr-agents start cline -Provider deepseek -Model <direct-model-id>`.

### 13.17 Show all shortcuts
`Ctrl+B` `I`: a window with every shortcut (type to filter) and this guide. `Ctrl+B` `?`
shows Herdr's own key list.

### 13.18 Check, repair, update
- Check everything: `pwsh -File "I:\herdr\check.ps1"`.
- A key stopped working: `herdr server reload-config`.
- Update Herdr: `herdr update`, then run the check.
- Restore an older config: `%APPDATA%\herdr\config.toml.<time>.bak` (copy it over
  `config.toml`, then reload).

### 13.19 Use several accounts of the same tool (beat the 5-hour limits)
Each pane can run an agent on its own sign-in, so a second Codex / Claude / Cline Pass /
Grok / OpenCode subscription (another email) has its own limits.
1. Once per account, sign it in: `herdr-agents login codex -Account work2` - a pane opens
   with that account's own folder and runs the tool's sign-in; finish it there (browser or
   code), then close the pane (`Ctrl+B` `X`).
2. Use it: `herdr-agents start codex -Account work2` (the agent is named `codex-work2`), or
   type `work2` in the **account** box of a column in the layout builder (`Ctrl+B`
   `Shift+B`), or `"account": "work2"` in a team plan.
3. See them all: `herdr-agents accounts`. The border shows `account work2`.
- Account folders: `%USERPROFILE%\.herdr-accounts\<tool>\<name>`; your normal sign-in is
  untouched. A new account gets a copy of your settings (Claude/Codex config, skills), never
  your sign-in.
- Works for: Claude, Codex, Cline (all Cline Pass profiles share the `cline` accounts), OpenCode,
  Grok. Not for Antigravity (its sign-in cannot be separated).
- Check each provider's terms for using several accounts.

### 13.20 Usage limits and switching the lead

**Automatic switching:** `start -Lead` and a successful `switch` ensure one hidden watcher is
running for this Windows user. Every 60 seconds it checks registered project leaders on its
Herdr server. A live leader that is not working and shows a limit message is handed off through
the same switch command; no key press is needed. Working panes and ordinary mentions of limits
are skipped. The same-email account checks still apply.

`herdr-agents autoswitch` starts the watcher explicitly; `-IntervalSec 60` changes the check
interval and `-Once` makes one pass. A second watcher exits with `autoswitch already running`.
`leaders` prints `Watcher: running` or `Watcher: not running (starts with start -Lead or switch)`.
Notifications say **Lead switched**, **Lead switch needs you** (answer the new pane's screen),
or **Lead switch failed**. A blocked project waits 10 minutes; a failed project waits for the
earliest known successor reset, or 30 minutes when unknown. The same outcome is not notified
again unless the leader, blocked pane or error changes. Other projects continue independently.
The watcher exits after 30 minutes without a live leader. Five consecutive ticks with errors
stop it with **Lead watcher stopped**. See `autoswitch.log` in `%APPDATA%\herdr` for the cause;
the log stays below 200 KB by dropping older lines. Start a lead or run `autoswitch` to restart.

1. `herdr-agents leaders` shows the ordered choices, installed tools, remembered limits and NEXT.
2. Press `Ctrl+B` `Shift+C`, or run `herdr-agents switch`. It remembers the old lead's reset,
   writes a handoff with git facts and its last 80 screen lines, opens the next free leader,
   and sends the takeover prompt. It checks that the new leader began, without waiting for its answer.
   From the key, the result (SWITCHED / BLOCKED / ERROR) shows in a message box (`switch -Notify`).
   Focus the old leader's pane before pressing the key, the first time (later it remembers).
3. The default order in `agents.json` `leaders.order` is 2040, 1984, 2030, Codex GPT-6.1-Sol,
   Gemini. The lead works on another Claude account (2040, then 1984) and never on the main sign-in
   2030 by default (owner's decision, 2026-10-31): 2030 is the last Claude fallback in the chain and
   stays reachable with `switch -To 2030`. Reorder or replace these aliases/profile names to choose
   your leaders.
4. `herdr-agents start 2040 -Lead` records the lead explicitly; otherwise switch uses the focused agent.
   `switch -To codex` chooses that chain item, even if limited (with a warning).
5. The old pane stays open. A sign-in or trust dialog prints BLOCKED: answer it in that pane.
   An exhausted chain reports the earliest reset and changes no state or handoff files.

Each project remembers its own leader. `leaders -Cwd <folder>` shows that project's CURRENT
leader and prints its project path first. `switch -Cwd <folder>` selects that project;
without it, the focused pane's folder is used, then the current folder. A folder inside a
git repo uses the repo root; a non-git folder is its own project. `start -Lead` records the
project of the started agent's folder. Account limits remain shared across projects.

Herdr's own handoffs stay in `I:\herdr\tasks`. Other projects keep them in
`<project>\.herdr\handoffs`; `.herdr/` is added once to the local git exclude file, without
changing `.gitignore`. A takeover starts at the project root and names the handoff and
existing project rules by absolute path, plus Herdr's team playbook and delegation guide.
The old shared leader record is used only when its folder belongs to this project; it is
removed only after the replacement project record is saved successfully.

`herdr-agents usage` still shows Claude percentages when available. The owner's current Orca
status line sends data to Orca only, so opening Claude does not guarantee usage data here.
Switch reads limit messages from the screen; without a reset time it remembers one hour.
Time-zone labels are ignored; reset times use local time. `start next` uses the first free
Claude in the chain when no eligible usage account is available. With usage data `start next` takes the least-used account, which can be the main sign-in - to move the lead use `switch` (the chain), not `start next`.
The new leader reads the handoff, fills its TODOs from history and continues the work.

### 13.21 Run this PC as a 24-hour server and reach it from anywhere

The PC never sleeps on AC power. Three parts make Herdr run day and night and answer from a phone or laptop:

1. **Start at sign-in (done by the lead):** the Startup shortcut `Herdr Boot.lnk` runs `herdr-boot.ps1`
   hidden. If no Herdr server runs, it starts one headless, opens a workspace **Lead** in `I:\herdr`
   and runs `herdr-agents start next -Lead` there (the first free leader in the chain; this also starts
   the watcher). If Herdr already runs, it only makes sure the watcher runs. Log: `%APPDATA%\herdr\boot.log`.
   Opening Herdr later (desktop icon) attaches to this same server.
2. **Sign in by itself after a restart (owner):** Startup items run only after Windows signs in. Turn on
   automatic sign-in yourself (it needs your password, so no agent does it), e.g. Sysinternals Autologon.
3. **Reach it from anywhere (owner, once):**
   - In an administrator PowerShell: `pwsh -File I:\herdr\setup-remote.ps1` - SSH server on at boot,
     answering only the Tailscale network (100.64.0.0/10), with PowerShell 7 as its shell
     (`-Undo` reverses it). Tailscale is a private network between your own devices.
   - On this PC: `tailscale up --unattended` and sign in (unattended keeps it on when you sign out).
   - On the phone or laptop: install Tailscale, sign in with the same account, then
     `ssh <user>@<pc name>` and run `herdr`, or from a PC with Herdr: `herdr --remote <user>@<pc name>`.
     Termux works on Android.

Not proven yet: an SSH login and `herdr --remote` from another device (needs the two Tailscale sign-ins).

**Key login (no password):** on the phone (Termux) run `pkg install openssh`, then `ssh-keygen -t ed25519`
(Enter for the questions), then
`cat ~/.ssh/id_ed25519.pub | ssh mflma@fawzy "pwsh -NoProfile -File I:/herdr/add-ssh-key.ps1"` (your
password one last time). After that `ssh mflma@fawzy` logs in with no password. `add-ssh-key.ps1 -Key "<.pub line>"`
does the same on this PC.

### 13.22 Keep Orca and Herdr on all the time (work server)

People at home may shut down or restart the PC normally; it never sleeps, and Orca and Herdr come back
by themselves.

1. `pwsh -File I:\herdr\herdr-keepalive.ps1 -Install` (once, no admin): the task **Herdr Keepalive**
   checks every minute. Orca closed from the tray or crashed -> started again; Herdr server gone -> the
   server, a lead and the watcher start again (`herdr-boot.ps1`). Down twice in a row before a restart,
   so updates are not disturbed. `-Status` shows what runs; `-Pause 120` (minutes) stops the guard for you,
   `-App orca|herdr` for one app, `-Resume` ends it. "Stop Herdr" keeps Herdr stopped for 15 minutes only.
2. `pwsh -File I:\herdr\setup-nosleep.ps1` (once, asks for admin rights; `-Undo`): no sleep or hibernate,
   the Start menu power button shuts down, Sleep is gone from the power menu, and Windows Update does not
   restart the PC by itself while someone is signed in.
3. Automatic Windows sign-in (owner, needs the password, e.g. Sysinternals Autologon): after a shutdown
   or power cut the PC signs in, the task starts both apps and then LOCKS the PC.
4. Optional, in the PC's firmware (BIOS/UEFI) setup: "Restore on AC power loss" = Power On, so the PC
   starts again by itself after a power cut.
5. Better for the family: a separate standard Windows account for the children (Settings > Accounts >
   Other users). Your apps keep running in your locked account while they use theirs.
