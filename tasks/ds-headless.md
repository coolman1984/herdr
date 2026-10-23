# Task: `herdr-headless.ps1` - run a Cline profile headless and report like `ask`

## 1. Role and goal
You are a senior PowerShell 7 engineer working for the lead (Claude). Read `AGENTS.md` first.
Goal: a new script `herdr-headless.ps1` runs a Cline agent profile from `agents.json` WITHOUT a
pane, waits for it to finish, saves the full log, and prints a short result like `herdr-agents
ask` does.
Why: owner's rule - Cline models get no pane; the lead drives them headless (CLAUDE.md section 0).
Headless Cline reports exactly when it is done (LESSONS #47), unlike Herdr's pane state (#41).

## 2. Context
- Folder: your current folder (worktree, branch `round2`). Read `agents.json` (Cline profiles:
  `cline`, `cline-flash`, `ds-flash`, `deepseek`, `kimi`: fields `command`, `providerFlag`,
  `provider`, `modelFlag`, `model`, `args`, `effortArgs`, `effort`, `account`), `LESSONS.md` #47,
  and `herdr-agents.ps1` (`Get-Account`, `Save-Run`, style).
- Headless command (verified by the lead): `cline -P cline-pass -m <model> --thinking high
  --auto-approve true -c <folder> --json -t <seconds> "<prompt>"`. Output is JSON Lines. The
  LAST line of a finished run is like
  `{"ts":"...","type":"run_result","finishReason":"completed","iterations":4,"usage":{"inputTokens":50180,"outputTokens":555,"cacheReadTokens":36366,"cacheWriteTokens":0,"totalCost":0},"aggregateUsage":{...}}`
  and the line before it holds the final answer:
  `{"ts":"...","type":"agent_event","event":{"type":"done","reason":"completed","text":"<final answer>"}}`.
  `-p` is PLAN mode in cline - never use it for the prompt.
- Other agents work in this folder: Codex on `herdr-agents.ps1` and `tests/test-round2-agents.ps1`.
  Never touch those, nor `HISTORY.md`, `LESSONS.md`, `CLAUDE.md`, `docs/`.

## 3. Exact scope
- Create: `herdr-headless.ps1`, `tests/test-headless.ps1`. Nothing else.
- Do not run cline yourself (the lead runs the live test). No git commit. No agents.

## 4. Design (decided by the lead)
- CLI: `herdr-headless.ps1 -Profile <name> (-Task <text> | -TaskFile <path>) [-Cwd <folder>]
  [-Model <id>] [-Effort <level>] [-Account <name>] [-TimeoutSec 1800]`
  - Use `[Alias('Profile')] [string]$AgentProfile` (never a variable named `$profile`).
  - Profile must exist and have `kind` = `cline`, else `Fail "herdr-headless runs Cline profiles only: <name>"`.
  - `-Cwd` default: current folder; must exist (`Fail "Folder not found: ..."`), resolved to a full path.
  - `-TaskFile`: must exist; the prompt sent is
    `Read <full path of the task file> completely and do exactly what it says. Reply in its answer format.`
- Arguments, built by `Get-HeadlessArgs($Spec, $Model, $Effort, $Cwd, $TimeoutSec, $AccountArgs) -> string[]`:
  profile `args` WITHOUT `--tui`/`-i`; `providerFlag provider` if set; `modelFlag model` (`-Model`
  overrides); effort = `-Effort`, else profile `effort`, via `effortArgs` (`{0}`); then
  `--json`, `-c <cwd>`, `-t <TimeoutSec>`; account args (from `Get-Account`-style `{dir}`
  replacement; you may copy the logic into this script); the prompt last.
- Run: `System.Diagnostics.Process`, `UseShellExecute=$false`, `CreateNoWindow=$true`, stdout
  and stderr redirected (async reads), `ArgumentList` (no manual quoting). The command is the
  profile's `command` resolved with `Get-Command` (`.Source`; on Windows `cline` is `cline.ps1`
  from npm - if the source ends with `.ps1`, run it as `pwsh -NoProfile -File <source> <args>`).
- Log: write every stdout line to `%APPDATA%\herdr\agent-runs\<yyyy-MM-dd>\<HHmmss>-headless-<profile>.jsonl`
  (unique name: add `-2`, `-3` if it exists), stderr to the same name `.stderr.txt` when not empty.
- Result, `Read-HeadlessResult([string[]]$Lines) -> pscustomobject @{ Finish; Text; Fresh; Out; Iterations }`:
  last `run_result` line -> finishReason, usage (Fresh = inputTokens - cacheReadTokens, Out =
  outputTokens); last `agent_event` with `event.type = "done"` -> Text. Lines that are not JSON
  are skipped.
- Output and exit code:
  - exit 0 and Finish `completed` -> print `DONE: <profile> (<model>) in <n> s - in <fresh> out <out>`
    then `--- answer ---`, the Text, then `--- log: <path> ---`; exit 0.
  - otherwise -> `FAILED (<finish or 'exit N'>)`, the last 20 log lines, the log path; exit 1.
  - timeout: cline's own `-t` ends it; if the process is still alive 30 s after TimeoutSec, kill
    it and report `FAILED (timeout)`.
- Header comment like the other scripts (usage lines, LESSONS #47, why no pane).

### Tests: `tests/test-headless.ps1`
Same style as `tests/test-batch-a.ps1` (AST loader, `Check`, `PASS n/m`, exit 1 on failure).
Test the pure functions with fixtures: `H1 args drop --tui and add json cwd timeout`,
`H2 profile effort is default and -Effort overrides`, `H3 -Model overrides profile model`,
`H4 result parsed from run_result and done event`, `H5 non-JSON lines skipped`,
`H6 missing run_result gives Finish null`, `H7 non-cline profile refused`,
`H8 task file prompt uses the full path`. Use the real JSON line shapes from section 2.

## 5. Definition of done
- [ ] `pwsh -NoProfile -File tests/test-headless.ps1` -> `PASS n/n`.
- [ ] Both new files parse (Parser::ParseFile, 0 errors).
- [ ] `git status --porcelain` shows your two files (others' files may appear - ignore them).

## 6. Answer format
DONE | PARTIAL | BLOCKED
Changed: <files>
Tests: <names> - suite <n/n>
Doubts: <list or none>
