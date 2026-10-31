# Task: leader switching per project (the project the leader works in, not only I:\herdr)

Follow-up to `tasks/leader-switch.md` (committed as a00d9fa8; the lead then changed: limit regex,
`leader.json` saved before the takeover prompt, `-Notify` message box). Same rules, scope style,
tests and report format as that spec. No git commands; the lead commits.

## Why
The owner leads agents in many project folders (e.g. `I:\GitHub\<repo>`). Today `switch` keeps ONE
`%APPDATA%\herdr\leader.json` for the whole PC, writes every handoff into `I:\herdr\tasks\`, and the
takeover prompt names I:\herdr files (HISTORY.md, docs/TEAM-PLAYBOOK.md relative). Two projects led at
once overwrite each other's leader, and a leader in another project reads the wrong files.

## Design (decided by the lead)
1. **Project** = the git top-level of the folder (`git rev-parse --show-toplevel`, as a Windows path),
   else the folder itself. Folder = `-Cwd`, else the focused pane's cwd (`HERDR_ACTIVE_PANE_CWD` /
   `pane get`), else the current location. One helper: `Get-LeaderProject([string]$Folder)`.
2. **Leader state per project**: `%APPDATA%\herdr\leaders\<name>-<hash8>.json` (name = leaf folder name,
   sanitized `[a-z0-9_-]`; hash8 = first 8 hex of SHA-256 of the lower-cased full project path), same
   fields as today plus `"project": "<path>"`. A file whose `project` differs from the expected path is
   an error, not a silent match. The old global `leader.json`: if present and its `cwd` is inside the
   project, use it once as the old leader and migrate it to the new file; then delete the old file only
   after the new one is written. Say so in HISTORY.
3. **Limits stay global** (`leader-limits.json`): limits belong to accounts, not projects. Unchanged.
4. **Handoff location**: project = `I:\herdr` (`$PSScriptRoot`) -> `tasks\` as today. Any other project
   -> `<project>\.herdr\handoffs\handoff-<time>-<id>.md`, and add `.herdr/` to `<project>\.git\info\exclude`
   (create the file if missing, add the line once, never touch `.gitignore` or tracked files). Not a git
   folder -> no exclude. "Earlier handoff" = newest earlier handoff in that same location.
   Plain `handoff` (no switch) uses the same location rule for its folder.
5. **Template text**: `HANDOFF-TEMPLATE.md` says "PowerShell only" and names CLAUDE.md / HISTORY.md /
   LESSONS.md. For a project other than I:\herdr, replace those lines with: "Read the project's rules
   files that exist: <list of the ones present among CLAUDE.md, AGENTS.md, README.md, HISTORY.md,
   CHANGELOG.md, absolute paths>", and drop "PowerShell only". Keep the I:\herdr text unchanged.
6. **Takeover prompt**: absolute paths only. Name the handoff, the project's present rules files (as in 5),
   and `I:\herdr\docs\TEAM-PLAYBOOK.md` + `I:\herdr\docs\DELEGATION.md` (how to lead with herdr-agents).
   "Work in <project>." The new leader starts in the project folder.
7. **`leaders`** shows CURRENT for the project of `-Cwd` / the focused pane / the current location, and
   prints `Project: <path>` first. `start -Lead` records the leader for the project of its `-Cwd`.
8. Old-leader detection fallback (focused pane) unchanged; it now looks up that project's file.

## Tests (extend `tests/test-leader-switch.ps1`)
- Two projects: switching in A leaves B's leader file unchanged; `leaders -Cwd B` shows B's CURRENT.
- Subfolder of a repo maps to the repo's leader file; a non-git folder maps to itself.
- Handoff for a non-herdr project lands in `<project>\.herdr\handoffs\`; `.git\info\exclude` gets
  `.herdr/` exactly once after two switches; no `.gitignore` change; I:\herdr project still uses `tasks\`.
- Template text for another project lists only rules files that exist and has no "PowerShell only".
- Prompt contains only absolute paths for those files.
- Legacy `leader.json` migration: used once, migrated, old file removed only after the new write;
  legacy file for another project is left alone.
- Mismatched `project` in a state file -> clear error.
Report how many new tests fail on HEAD (`git show HEAD:...` copies). Run all tests in `tests/` and
`pwsh -File check.ps1`; all must pass. Update HISTORY (new phase 58a is wrong - use Phase 59),
docs/GUIDE.md 13.20, docs/FILES.md (`leaders\` folder, `<project>\.herdr\handoffs`), skills/herdr-team
SKILL.md (one line). LESSONS only if you learned a durable fact.
