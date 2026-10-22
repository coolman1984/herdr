# Handoff: {{FROM}} -> {{TO}} ({{TIME}})

<!-- Lead -> lead handoff, written when the lead's account nears its limit (>= 95% of the
5-hour window: `herdr-agents usage`), its context fills, or the owner asks.
Made by `herdr-agents handoff`: everything in "Facts" is filled in automatically; fill EVERY
section marked TODO ("n/a" is allowed, silence is not), then delete these comments.
The next lead must be able to continue WITHOUT asking you anything: you will be gone. -->

You are Claude Code taking over as LEAD for the owner. Reply to the owner in English
(terminal). Work in `{{FOLDER}}`. PowerShell only.

## 0. Why this handoff
{{WHY}}
Continue on the account named under NEXT in section 3 (`herdr-agents start next`).

## 1. Read first (in this order)
1. `{{FOLDER}}\CLAUDE.md` (the rules), then `HISTORY.md` - the last 3 phases (newest: {{LAST_PHASE}}).
2. `LESSONS.md` #TODO (the ones this work depends on).
3. TODO: specs, test files, docs this work uses (absolute paths).

## 2. The owner's request (verbatim where possible)
TODO: quote the owner's words; add the date. Note anything they changed mid-way.

## 3. Facts at handoff time (filled in automatically)

### Claude usage limits
{{USAGE}}

### Live agents (Herdr)
{{AGENTS}}

### Git ({{FOLDER}})
{{GIT}}

## 4. State of the work
### Done - with proof (what was run, what it printed)
- TODO

### In progress - exactly where it stopped
- TODO: the file, the step, the last command and its result. Half-written files?

### Not started
- TODO

## 5. Decisions already made (do not re-open)
- TODO: owner decisions (with date) and lead decisions (with the reason).

## 6. Blocked / refused / waiting on the owner
- TODO: actions the permission classifier refused (do NOT work around them - give the owner
  the exact command), sign-ins the owner must finish, questions asked and not answered.

## 7. Next steps (numbered; each with how to verify it)
1. TODO - verify: `<command>` -> `<expected>`.

## 8. Traps found on the way
- TODO: what failed and why; what NOT to try again.

## 9. Rules for this work
- Never read or print secrets (`.credentials.json`, auth files, keys).
- Tests never open windows; use an isolated Herdr session (CLAUDE.md section 3).
- Close only panes you started. Back up before overwriting anything outside the folder.
- Every behaviour change: HISTORY entry + `pwsh -File check.ps1` + commit.
- TODO: rules specific to this work.

## 10. Definition of done and the report to the owner
- [ ] TODO: each check that must pass.
- [ ] `pwsh -File check.ps1` -> no FAIL; committed with a HISTORY entry.
Report to the owner: a short list - done (with proof), not done (and why), what they must do.
