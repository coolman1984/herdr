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
| Double-click **Herdr** (Desktop / Start menu) | Windows Terminal opens Herdr in `D:\WORK\Software Development\GitHub` |
| Type `herdr` in File Explorer's address bar | Herdr opens in the folder you are looking at |
| `Win + R` -> `herdr` | Herdr opens in the GitHub folder |

Close the window any time: the agents keep running. Open Herdr again to come back.
Stop everything: `herdr server stop`.

## Keys inside Herdr

Press `Ctrl+B`, let go, then the key.

| Keys | Action |
|---|---|
| `Ctrl+B` `F` | Pick a folder (Windows dialog) -> new workspace there |
| `Ctrl+B` `Shift+F` | Same, and start Claude Code in it |
| `Ctrl+B` `T` | Same, with a team: Claude Code left + Codex (GPT-6.1-Sol) right |
| `Ctrl+B` `V` | Split the pane in two, side by side |
| `Ctrl+B` `Shift+V` | Split the pane into 4 equal columns, all in the same folder |
| `Ctrl+B` `Shift+B` | **Layout builder**: 1-6 equal columns, each with its own folder and an empty PowerShell, a new agent (any model), or an already-running agent moved in |
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
```

## Documents

| File | What it is |
|---|---|
| [CLAUDE.md](CLAUDE.md) | Rules for any AI agent working on this folder - read first |
| [HISTORY.md](HISTORY.md) | Every change: Symptom / Cause / Fix / Lesson |
| [LESSONS.md](LESSONS.md) | Numbered lessons about Herdr, the CLIs and Windows |
| [docs/SETUP.md](docs/SETUP.md) | Install, update, rebuild on a new PC, restore, troubleshooting |
| [docs/DELEGATION.md](docs/DELEGATION.md) | The delegation tool and the agent profiles |
| [docs/FILES.md](docs/FILES.md) | Every file here and everything this setup changed outside the folder |

Check that everything is in place: `pwsh -File check.ps1`.
