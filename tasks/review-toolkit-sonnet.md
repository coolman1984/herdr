# Task: review the Herdr toolkit code in I:\herdr (read-only)

## 1. Role and goal
You are a senior PowerShell/Windows reviewer working for the lead (Claude Opus/Sonnet lead session).
Goal: find real defects in the toolkit scripts and report them, ordered by severity.
Why: the owner wants an independent review after today's changes (paths rebuilt, shortcuts, PATH).

## 2. Context
- Folder `I:\herdr`; read `CLAUDE.md`, `AGENTS.md`, then `LESSONS.md` before judging anything.
- Review: `herdr-agents.ps1` (+ `.cmd`), `herdr-headless.ps1`, `herdr-trial.ps1`, `herdr-open-folder.ps1`,
  `herdr-layout.ps1`, `herdr-help.ps1`, `herdr-stop.ps1`, `herdr-spaces.ps1`, `herdr-columns.ps1`,
  `herdr-hud.ps1`, `claude-limits.ps1`, `check.ps1`, `sync-skills.ps1`, `install*.ps1`, `launcher/`,
  `config/keys.toml`, `config/settings.toml`, `agents.json`.
- Out of scope: `herdr-app/` (upstream Rust source), `herdr/`.

## 3. Scope - READ ONLY
- Change nothing. No git, no installs, no settings, no panes, no live Herdr session (isolated
  session `--session lab` only if you must run something). Never read or print secrets
  (`auth.json`, `.credentials.json`, tokens).
- Look for: wrong paths after the move to `I:\`, silent failures (`try{}catch{}`), shadowed
  automatic variables (`$args`, `$input`, `$matches`, `$error`, `$host`, `$profile`), quoting/injection
  in `Invoke-Herdr` and `pane run` strings, race conditions, missing `-LiteralPath`, scripts that flash a
  console window (LESSONS #39), unsafe deletes, mismatches between docs and code, `agents.json` errors
  (e.g. the new `cursor` profile, `gemini` default `gemini-4-argon-high`).

## 4. Steps
1. Read the rules, then the scripts. Parse-check each with `[System.Management.Automation.Language.Parser]::ParseFile`.
2. For every finding: file:line, what is wrong, a concrete failing scenario, severity (high/medium/low).
3. Prove claims by running a read-only command where possible; write "not verified" otherwise.

## 5. Answer format
Findings first, ordered by severity, max 15 findings:
`SEVERITY | file:line | defect | failing scenario | proof or "not verified"`
Then: Doubts: <list or none>. Then: Verdict: PASS | NEEDS FIX.
