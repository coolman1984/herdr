# Task spec template (Claude -> subagent)

Claude is the lead engineer: it owns the plan, the design and the result. A subagent gets
a spec it can execute without guessing. Fill EVERY section; "n/a" is allowed, silence is
not. One file per task: `tasks\<short-name>.md`. Sources: OpenAI's Codex best practices
(context, constraints, definition of done, verification; durable rules in AGENTS.md) and
Anthropic's prompting guidance (be explicit, give the reason, show the format).

---

# Task: <one line: the outcome>

## 1. Role and goal
You are <role, e.g. "a senior PowerShell engineer"> working for the lead (Claude).
Goal: <the outcome in 1-3 sentences, as the user will see it>.
Why: <the reason - it lets you make the right small decisions>.

## 2. Context you need (nothing else is needed)
- Folder: `<absolute path>`; read first: `<CLAUDE.md / AGENTS.md / the files listed>`.
- How the surrounding code works: <2-6 bullets: the modules, the data, the conventions>.
- Style to copy: `<file>` (header comment, naming, error handling).

## 3. Exact scope
- Create: `<file>` | Change: `<file>` (only `<function>`) | Do NOT touch: `<files>`.
- Do not: run git, install anything, change settings, touch other folders, <...>.

## 4. Design (decided by the lead - follow it)
- Technology: <language/version, libraries, APIs; e.g. "pwsh 7, Windows Forms, no modules">.
- Entry point / CLI: `<script> [-Param <type>] ...` with the meaning of each parameter.
- Functions to write, with signatures and behaviour:
  - `Get-Thing([string]$Path) -> [pscustomobject[]]` - <what it returns, edge cases>.
  - `Show-Thing(...)` - <UI layout: controls, sizes, events>.
- Data formats: <exact JSON / file layouts with an example>.
- Errors: <what to do on each failure: message text, exit code>.
- Known traps: <e.g. "PowerShell names ignore case", "array unrolling in returns">.

## 5. Steps
1. <first concrete step>
2. <...>
3. Run the checks in section 6 and fix until they pass.

## 6. Definition of done (all must be true)
- [ ] `<command>` prints `<expected>` / exits 0.
- [ ] `<parse / test / lint command>` -> 0 errors.
- [ ] <behaviour check the lead will repeat>.

## 7. Answer format
Reply with exactly:
DONE | BLOCKED | PARTIAL
Changed: <files>
Checks: <each check from section 6 and its result>
Not done / doubts: <list, or "none">
