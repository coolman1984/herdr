# Rules for AI agents working on this folder

Read this file completely before your first tool call. Read [HISTORY.md](HISTORY.md)
and [LESSONS.md](LESSONS.md) before changing a script, a key binding or a skill.

**Delegates** (any agent started by the lead, Claude helpers included) also follow the delegate
rules in @AGENTS.md - Claude Code reads AGENTS.md only through this import when a CLAUDE.md exists.

This folder is the owner's personal Herdr setup: the tools that let Claude Code lead
the other agent CLIs on this PC. It touches the owner's Claude settings, PATH,
registry and agent CLIs, so mistakes reach outside the folder.

## 0. The first rule: every experience goes into the docs

Owner's first rule (2026-10-23): whenever an agent learns something new - a CLI or Herdr
behaviour, a bug and its real cause, a model's strength or failure, a technique that worked, a
trap - it writes it into the docs at once, by itself, without being asked: durable facts in
LESSONS.md, behaviour changes in HISTORY.md, agent strengths and steering in
docs/AGENT-ROSTER.md and docs/TEAM-PLAYBOOK.md, delegate rules in AGENTS.md. Nothing valuable
stays only in a chat.

**Agent panes (owner's rule, 2026-10-23; Cline added back 2026-10-29):** Codex, Antigravity, Claude
and Cline agents get panes, split vertically to the right of the lead. (2026-10-23 to 2026-10-29 Cline ran
headless with no pane, LESSONS #47; `herdr-headless.ps1` stays available for one-shot Cline runs.)
Only coding, review and fix work gets a pane; research runs headless.
**Close finished panes (owner's rule, 2026-10-24):** the PC is short on RAM and disk; the lead
stops every pane it opened as soon as the task is done and no more work is planned, and removes
merged worktrees and its temp copies.

**Claude helpers and research (owner's rules, 2026-10-24):** the lead never uses subagents of
its own Claude session. Claude helpers run on the owner's OTHER Claude accounts (`-Account 1984`
/ `2040`, `start next`) in panes beside the lead, so each account's usage stays separate.
Research is done headless by Codex GPT-6.1-Sol medium and Gemini 4 Argon High (results in
files); the lead searches in parallel and adjusts its decisions when their results arrive.

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
3. **Approval mode is the owner's decision.** Owner decision 2026-10-22 (HISTORY Phase 18):
   every profile in `agents.json` runs without approval prompts - Claude
   `--permission-mode auto`, Codex `-a never -s workspace-write` (still sandboxed to its
   folder), Cline `--auto-approve true`, Antigravity `--dangerously-skip-permissions` (its
   `--mode auto` still asked before shell commands; owner's choice, Phase 19), Grok
   `--always-approve`.
   The owner made this change; agents must not change approval or trust settings on their
   own, and must not widen them further (e.g. Codex's full sandbox bypass) unless the owner
   asks. Secrets, sign-ins and folder-trust questions still go to the owner.
4. **Claude's own settings are the owner's.** `~/.claude/settings.json` and Herdr
   integrations (`herdr integration install ...`) change Claude Code itself - give the
   owner the exact command to run, do not run it (LESSONS #3).
5. **Close only what you started.** Never close other panes, tabs or workspaces;
   never run `herdr server stop` (it kills every pane) unless the owner asks.
6. **Back up before you overwrite** anything outside this folder (config.toml,
   registry values, skills) and say where the backup is.
7. **Never edit `herdr-app/`** for a quick fix: it is the upstream clone. Changes there
   need Zig 0.16 + MSVC Build Tools to build (LESSONS #9) and are kept on a branch.

## 3. Engineering rules

- PowerShell 7 (`pwsh`) only; scripts must also work when started by Herdr
  (`cmd.exe /d /c ...`, no console, detached).
- **Nothing may flash a console window** (owner's rule, Phase 22): every key command,
  shortcut and background start is `conhost.exe --headless pwsh -NoProfile ...` (dialogs
  still show; LESSONS #39). Never `-WindowStyle Hidden` alone, never a VBS hidden launcher.
- **Tests stay invisible**: never open a terminal window for a test session (it cannot
  attach from inside a Herdr pane and only shows the owner an error - LESSONS #24).
- Call Herdr by its stable path `%LOCALAPPDATA%\Programs\Herdr\bin\herdr.exe`,
  never bare `herdr`: old shells and panes do not have it on PATH (LESSONS #1).
- Read IDs (workspace, pane, agent names) from Herdr's JSON; never guess them.
- Test in an **isolated Herdr session**, never the owner's: `herdr --session lab server`
  (headless), point the CLI and scripts at it with
  `$env:HERDR_SOCKET_PATH = "$env:APPDATA\herdr\sessions\lab\herdr.sock"`, and end it
  with `herdr --session lab server stop` (LESSONS #20). A test once crashed the owner's
  server and every agent in it (Phase 13). Use a test workspace in the live session
  only for something the isolated session cannot show.
- A tool reports DONE only after reading the result back (screen, file, JSON).
- Skills: edit the master copy in `skills/`, then `pwsh -File sync-skills.ps1`.

## 4. Where things are

See [docs/FILES.md](docs/FILES.md).
