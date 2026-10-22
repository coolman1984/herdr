# Rules for AI agents working on this folder

Read this file completely before your first tool call. Read [HISTORY.md](HISTORY.md)
and [LESSONS.md](LESSONS.md) before changing a script, a key binding or a skill.

This folder is the owner's personal Herdr setup: the tools that let Claude Code lead
the other agent CLIs on this PC. It touches the owner's Claude settings, PATH,
registry and agent CLIs, so mistakes reach outside the folder.

## 1. The update rule

**Every change of behaviour gets a HISTORY.md entry in the same commit**
(Symptom / Cause / Fix / Lesson, observed behaviour, not intention). A durable
property of Herdr, a CLI or Windows also goes into LESSONS.md as a numbered lesson.
Then run `pwsh -File check.ps1` - it must pass.

## 2. Safety rules

1. **Never handle secrets.** API keys, passwords and sign-ins are typed by the owner
   in the agent's own pane (`cline auth <provider>`, `/login` ...). Never put a key in
   a file, an argument, an environment variable, a log or a chat.
2. **Never approve for the owner.** When an agent is BLOCKED (tool approval or a
   question), show its screen and let the owner decide.
3. **Agents start with approval ON.** Do not add `--yolo`, `--auto-approve true`,
   `--dangerously-skip-permissions` or similar to `agents.json` unless the owner asks
   for that agent, in that run.
4. **Claude's own settings are the owner's.** `~/.claude/settings.json` and Herdr
   integrations (`herdr integration install ...`) change Claude Code itself - give the
   owner the exact command to run, do not run it (LESSONS #3).
5. **Close only what you started.** Never close other panes, tabs or workspaces;
   never run `herdr server stop` (it kills every pane) unless the owner asks.
6. **Back up before you overwrite** anything outside this folder (config.toml,
   registry values, skills) and say where the backup is.
7. **Never edit `herdr/`** for a quick fix: it is the upstream clone. Changes there
   need Zig 0.16 + MSVC Build Tools to build (LESSONS #9) and are kept on a branch.

## 3. Engineering rules

- PowerShell 7 (`pwsh`) only; scripts must also work when started by Herdr
  (`cmd.exe /d /c ...`, no console, detached).
- Call Herdr by its stable path `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe`,
  never bare `herdr`: old shells and panes do not have it on PATH (LESSONS #1).
- Read IDs (workspace, pane, agent names) from Herdr's JSON; never guess them.
- Test against the live Herdr in a **separate test workspace**, then close it.
- A tool reports DONE only after reading the result back (screen, file, JSON).
- Skills: edit the master copy in `skills/`, then `pwsh -File sync-skills.ps1`.

## 4. Where things are

See [docs/FILES.md](docs/FILES.md).
