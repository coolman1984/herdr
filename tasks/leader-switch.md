# Task: one-step leader switch with a chain of leader choices (Claude accounts, then other models)

## 1. Role and goal
You are a senior PowerShell 7 engineer working for the lead (Claude).
Goal: when the lead agent (the "leader" / Architect) hits its usage limit, the owner presses ONE key
(`Ctrl+B` `Shift+C`) or runs `herdr-agents switch`, and the work continues on the next free leader:
another Claude account first, then other models (Codex GPT-6.1-Sol, Gemini) when every Claude account
is limited. The owner chooses and orders the leaders in `agents.json`.
Why: today the owner forgets which account the leader used, `herdr-agents usage` shows "no data" for
every account, and `start next` then refuses. Switching takes four manual steps.

Root cause found by the lead (2026-10-31, verified):
- `~\.claude\statusline.js` no longer exists on this PC; `~\.claude\settings.json` `statusLine` now runs
  Orca's `claude-statusline.cmd`, which posts rate limits to Orca only. The account folders
  `%USERPROFILE%\.herdr-accounts\claude\2040` and `1984` have no status line at all.
- So `%APPDATA%\herdr\claude-cost\` is never written, `Get-ClaudeLimits` has no data, and
  `Get-ClaudeNextAccount` returns nothing. Do NOT try to fix the status line (it is the owner's
  Claude setting, CLAUDE.md rule 4). Design around it: limits are learned from the screen.

## 2. Context you need
- Folder: `I:\herdr`. Read first: `CLAUDE.md`, `AGENTS.md`, the last 3 phases of `HISTORY.md`,
  `LESSONS.md` #42 and the lessons about `agent prompt` / `agent read`.
- `herdr-agents.ps1` (874 lines): one `switch ($Command)`; helpers `Invoke-Herdr`, `Fail`,
  `Format-Screen`, `Get-AgentDialog`, `Get-LiveAgents`, `Get-State`/`Save-State`, `Save-Run`,
  `Get-Account`, `Invoke-Git`. Commands of interest: `start` (aliases from `agents.json` `aliases`:
  `2030` = main sign-in, `2040`, `1984`; `start next`; key-started runs get
  `HERDR_ACTIVE_PANE_ID` / `_CWD`), `handoff` (fills `tasks\HANDOFF-TEMPLATE.md`), `usage`.
- `claude-limits.ps1`: `Get-ClaudeLimits`, `Get-ClaudeNextAccount`, `Format-ClaudeLimits`,
  `Get-ClaudeAccountNames`, `Get-ClaudeAccountOf`.
- `agents.json`: profiles under `agents` (claude, codex = gpt-6.1-sol, gemini = agy gemini-4-argon-high, ...),
  `accountNames`, `aliases`.
- `config/keys.toml`: `prefix+shift+c` runs `herdr-agents.ps1 start next` via `conhost.exe --headless`.
- Style to copy: `herdr-agents.ps1` itself (comments, `Fail`, `-LiteralPath`, absolute herdr path).
- Tests style: `tests/test-batch-a.ps1` (offline, `-SourceRoot` for baseline copies).

## 3. Exact scope
- Change: `herdr-agents.ps1`, `claude-limits.ps1` (only if a helper belongs there), `agents.json`
  (add `leaders`), `config/keys.toml` (Shift+C line and its comment), `AGENTS.md` (one rule, below),
  `skills/herdr-team/SKILL.md` (a short "switching the lead" section), `docs/GUIDE.md` 13.20,
  `docs/DELEGATION.md`, `docs/FILES.md` (new state files), `README.md` (Shift+C row / limits line),
  `HISTORY.md` (new phase), `LESSONS.md` (new lesson: the status-line root cause above).
- Create: `tests/test-leader-switch.ps1`, fixtures `tests/fixture-limit-*.txt` if useful.
- Do NOT touch: `herdr-app/`, `app/`, `~\.claude\*`, `%APPDATA%\herdr\config.toml` (the lead applies the
  key change to the live config), `.wt/`, the owner's live Herdr session.
- No git commit / push / reset / stash / checkout. Do not start agents. No browser tools.

## 4. Design (decided by the lead - follow it)
### 4.1 Leader chain (the owner's choices)
`agents.json` gets:
```json
"leaders": {
  "_comment": "Who leads, in order. herdr-agents switch takes the first one that is installed, not the current leader and not limited. Each item is an alias or a profile name. Reorder to change the choice.",
  "order": ["2030", "2040", "1984", "codex", "gemini"]
}
```
Validate it in `Get-Registry` style: every item must be an alias or profile; fail clearly otherwise.

### 4.2 State (in `%APPDATA%\herdr\`)
- `leader.json`: `{ "item": "2040", "name": "claude-2040", "pane_id": "...", "cwd": "...", "since": "<iso>" }`
  - the current leader.
- `leader-limits.json`: `{ "<item>": { "until": "<iso>", "source": "screen|usage|default", "seen": "<iso>" } }`.
  An entry whose `until` has passed counts as free. Write both files atomically (temp file + move).

### 4.3 Limit text from the screen
`Get-LimitFromScreen([string]$Screen, [datetime]$Now)` returns `$null` or
`[PSCustomObject]@{ Until = <datetime>; Line = <matched line> }`. Look only at the last 40 lines.
Match the limit messages of Claude Code ("You've hit your limit", "usage limit", "limit reached",
"Limit reached ... resets 3pm", "resets 3:30pm (Europe/Paris)", "resets Nov 2, 9am") and Codex
("You've hit your usage limit ... try again at 3:05 PM"). Parse the reset time (today, or tomorrow if
already passed; ignore the time-zone name, use local time). A limit line with no parsable time ->
Until = Now + 1 hour. A normal answer that only mentions the word "limit" (e.g. "rate limit handling in
the API") must NOT match: require the phrasing of a limit message. Say in your report which strings you
took from a real screen and which are assumptions.

### 4.4 `herdr-agents leaders`
Prints one row per chain item: order, item, kind + model, installed yes/no, state
(`CURRENT` / `limited until ddd HH:mm (screen|usage|default)` / `free` / `not installed`), and
the 5-hour % from `Get-ClaudeLimits` when it has data. Last line `NEXT: <item>` or
`NEXT: none - every leader is limited or missing (earliest free: <item> at <time>)`.

### 4.5 `herdr-agents switch [-To <item>] [-Cwd PATH]`
One step, prints each step on one line:
1. **Old leader** = `leader.json` if its pane is still live (`Get-LiveAgents`), else the focused pane
   (`HERDR_PANE_ID`, or `HERDR_ACTIVE_PANE_ID` when started by a key - same handling as `start`).
   Folder = `-Cwd`, else the old leader's cwd, else the focused pane's cwd.
2. **Mark it limited**: read its screen (`agent read --source recent`); `Get-LimitFromScreen` time
   (source `screen`); else `Get-ClaudeLimits` reset time if that account is at LIMIT (source `usage`);
   else Now + 1 hour (source `default`, the owner pressed switch for a reason).
3. **Handoff file**: factor the `handoff` body into a function and reuse it. Add to the file the last
   80 lines of the old leader's screen (in a fenced block) and the path of the newest earlier
   `tasks\handoff-*.md`. `{{FROM}}` / `{{TO}}` name the real items. TODOs stay (the old leader may be
   unable to fill them); the new leader fills what it can.
4. **Pick the next**: `-To` if given (fail if not in the chain or not installed; a limited `-To` is
   allowed with a warning). Else the first chain item that is installed, not the old leader, not
   limited in `leader-limits.json`, and for Claude items not at LIMIT in `Get-ClaudeLimits` when it has
   data. None -> fail with the earliest free time; change nothing else.
5. **Start it** with the existing `start` code path (refactor `start` into a function if needed; do
   not shell out to a second pwsh) beside the focused pane, in the folder from step 1.
6. **First prompt**, sent WITHOUT waiting for an answer (`agent prompt` without `--wait`; read back
   that the agent began, per CLAUDE.md "DONE only after reading the result back"):
   "You are now the LEAD of this Herdr setup (not a delegate), taking over from <old item>, which hit
   its limit. Read <handoff file> first, then CLAUDE.md, the last 3 phases of HISTORY.md and
   docs/TEAM-PLAYBOOK.md. Fill the TODOs in the handoff from git, HISTORY and
   `herdr-agents history`, then continue the work. Use herdr-agents to lead the other agents."
   A startup dialog (trust, sign-in) -> do not send; print BLOCKED with the screen, exit 3 (as `ask`).
7. **Record**: write `leader.json` for the new leader. Leave the old pane open (CLAUDE.md rule 5:
   close only what you started); print `Old leader still open in pane <id>: herdr-agents stop <name>`
   when this tool started it.
8. Final line: `SWITCHED <old> -> <new> (handoff: <path>)`.

### 4.6 Also
- `start next` with no usage data: instead of failing, pick the first free Claude item of the chain.
- `start` records `leader.json` when called with a new switch `-Lead` (so the owner can say who leads:
  `herdr-agents start 2040 -Lead`). `switch` uses the same writer.
- `config/keys.toml`: `prefix+shift+c` runs `herdr-agents.ps1 switch` (keep `conhost.exe --headless`).
  Comment: "Switch the lead to the next free leader (agents.json leaders)".
- `AGENTS.md`: one rule - "If your first prompt says you are the LEAD (from `herdr-agents switch`),
  the lead rules in CLAUDE.md apply to you, not the delegate rules above."
- Update the header usage comment of `herdr-agents.ps1` and the `ValidateSet`.

## 5. Tests (`tests/test-leader-switch.ps1`, offline, no Herdr server)
Fake `Invoke-Herdr` / the clock as the existing tests do. Cover:
- `Get-LimitFromScreen`: each Claude and Codex limit phrasing; reset time today vs tomorrow; no time ->
  +1 h; a normal answer about "rate limit handling" and an answer quoting "limit" -> no match.
- Chain choice: skips the current leader, a limited item, an uninstalled item, a Claude item at LIMIT;
  an expired limit counts as free; all limited -> failure naming the earliest free time and no state
  change; `-To` picks exactly that item.
- State: `leader.json` / `leader-limits.json` round-trip; a corrupt file gives a clear error, not a
  silent reset.
- `start next` with no usage data picks the first free Claude chain item.
- Invalid `leaders.order` item fails with a clear message.
Run the new tests against the old code (`git show HEAD:herdr-agents.ps1` etc. into a temp folder, the
`-SourceRoot` pattern) and report how many fail there. Run all existing tests in `tests/` and
`pwsh -File check.ps1`; all must pass.

## 6. Done means
- `pwsh -File tests/test-leader-switch.ps1` passes; the existing tests pass; `pwsh -File check.ps1` passes.
- `pwsh -File herdr-agents.ps1 leaders` runs on this PC (it needs no pane) and shows the chain.
- HISTORY phase (Symptom / Cause / Fix / Lesson) and the LESSONS entry are written.

## 7. Report format
1. Files changed (one line each). 2. Test results with counts (new tests on new code, on old code;
existing tests; check.ps1). 3. Limit strings: which are from a real screen, which assumed.
4. Design decisions you made where this spec was silent. 5. Doubts / not verified.
