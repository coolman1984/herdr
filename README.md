# HERDR - one place for all the AI agents on this PC

Personal setup of [Herdr](https://github.com/herdrdev/herdr) (a terminal workspace
for coding agents) so that **Claude Code (Opus 5.5) leads every other agent CLI on
this PC** - Codex on GPT-6.1-Sol, Cline (any provider: DeepSeek and other Chinese
models), OpenCode, Grok, Antigravity - and so that opening a project is one click.

Herdr itself is the **official 0.9.5 build**, unmodified. Everything here is built
*around* it: launchers, a folder picker, a delegation tool, Claude skills and docs.

## Start

| Way | What happens |
|---|---|
| Double-click **Herdr** (Desktop / Start menu) | Windows Terminal opens Herdr in `I:\herdr` |
| Type `herdr` in File Explorer's address bar | Herdr opens in the folder you are looking at |
| `Win + R` -> `herdr` | Herdr opens in the GitHub folder |

All three run `launcher\HerdrLauncher.exe` - no console window flashes (`Herdr.cmd` is the fallback).
**Leader switching:** `herdr-agents leaders` shows the chain; `Ctrl+B` `Shift+C` (or
`herdr-agents switch`) saves a handoff and starts the next free leader - another Claude account (2040, then 1984), never the main sign-in. Claude usage numbers
are optional; limits are learned from the screen (docs/GUIDE.md 13.20).

Close the window any time: the agents keep running (Herdr's server runs on its own).
Open Herdr again and they are still working. **Stop everything for real:** `Ctrl+B`
`Shift+Q` or the Desktop icon **Stop Herdr** - both ask first. **All shortcuts:** `Ctrl+B` `I`.

## Keys inside Herdr

Press `Ctrl+B`, let go, then the key.

| Keys | Action |
|---|---|
| `Ctrl+B` `F` | Pick a folder (Windows dialog) -> new workspace there |
| `Ctrl+B` `Shift+F` | Same, and start Claude Code in it |
| `Ctrl+B` `T` | Same, with a team: Claude Code left + Codex (GPT-6.1-Sol) right |
| `Ctrl+B` `V` | Split the pane in two, side by side |
| `Ctrl+B` `Shift+V` | Split the pane into 4 equal columns, all in the same folder |
| `Ctrl+B` `Shift+B` | **Layout builder**: 1-6 equal columns, each with its own folder and an empty PowerShell, a new agent (any model), or an already-running agent moved in. **Save preset...** keeps a layout under a name; pick it next time from **Preset** |
| `Ctrl+B` `A` | **Agent board** in a pane below: every agent, blocked ones first in red |
| `Ctrl+B` `I` | **All shortcuts** (type to filter) and the full guide in one window |
| `Ctrl+B` `Shift+S` | **Spaces side by side**: tick spaces from the sidebar -> one view, a column each (agents keep running); **Separate again** puts them back |
| `Ctrl+B` `-` / `C` | Split down / new tab |
| `Ctrl+B` `Q` | Detach (agents keep running) |
| `Ctrl+B` `?` | All keys |

## Delegation: Claude controls the other agents

In a Claude pane inside Herdr, just ask: *"start Codex and DeepSeek, give both
this bug, compare their fixes"*. Claude uses the `herdr-team` skill, which runs
`herdr-agents.ps1`. Full guide: [docs/DELEGATION.md](docs/DELEGATION.md).

```
herdr-agents tools      # agents and models on this PC
herdr-agents status     # who is running, working / blocked / idle
herdr-agents watch      # the same as a live board (ctrl+b a)
herdr-agents start codex -Worktree fix-login   # Codex on its own git branch, in a new space
herdr-agents diff codex                        # what it changed, before you merge
herdr-agents fanout "<question>" -Agents codex,deepseek   # same task to both, one report
herdr-agents history    # every task and answer, kept as Markdown
```

Every pane's top border shows **name · project path · model · context used · tokens**
(`herdr-hud.ps1`, starts by itself), and the sidebar shows each agent's own name, model
and context. Rename a pane with `Ctrl+B` `Shift+P`.

Windows notifications are on: when an agent in a space you are not looking at
finishes or needs you, Windows tells you.

## Documents

| File | What it is |
|---|---|
| [docs/GUIDE.md](docs/GUIDE.md) | **The complete guide**: every key, feature, recipe and trick |
| [CLAUDE.md](CLAUDE.md) | Rules for any AI agent working on this folder - read first |
| [HISTORY.md](HISTORY.md) | Every change: Symptom / Cause / Fix / Lesson |
| [LESSONS.md](LESSONS.md) | Numbered lessons about Herdr, the CLIs and Windows |
| [docs/SETUP.md](docs/SETUP.md) | Install, update, rebuild on a new PC, restore, troubleshooting |
| [docs/DELEGATION.md](docs/DELEGATION.md) | The delegation tool and the agent profiles |
| [docs/FILES.md](docs/FILES.md) | Every file here and everything this setup changed outside the folder |

Check that everything is in place: `pwsh -File check.ps1`.
