DONE

Changed: `herdr-hud.ps1`, `docs/GUIDE.md` (HUD section only), `tests/test-hud-cline.ps1` (new),
`tasks/ds-hud-cline-answer.md` (this file).

Per item: <function / change> - test <names>
- `Get-ClineSessionRoots` + `$clineSessionRoots` test hook (`~\.cline\data\sessions` and
  `~\.herdr-accounts\cline\*\data\sessions`) - all of C1-C11 point it at a temp root.
- `Get-ClinePids` (foreground pids + shell_pid; failure -> empty) - C6.
- `Test-ClineLive` (new, shared by both paths: cwd matches the pane's folder, case- and
  trailing-`\`-insensitive, and status is not `exited`/`done`/`completed`/`failed` - **D1**) -
  C2, C3, C9.
- `Find-ClineSession` (pid match wins over a newer cwd match; else the newest live session for the
  folder; 7-day mtime filter; per-pane cache `@{File;Pids;Cwd}`). Cache reuse now also requires
  that no **newer live session for the same folder** exists: candidate times are compared first and
  only newer files are parsed, so a Cline restart in the same pane (same pids, shared daemon pid)
  takes over - **D2** - C1, C2, C3, C8, C9, C10.
- `Get-ClineUsage` (model; fresh = inputTokens - cacheReadTokens; out; nulls when `metadata.usage`
  is missing; `$null` on bad/missing json) - C4, C5.
- `Update-Panes` Cline branch (session model wins over the started/registry fallback, adds
  `in <fresh> · out <out>` to the title and the `in`/`out` tokens **in that order** - **D3** -, no
  `ctx`, one extra `pane process-info` + file read only for Cline panes, whole block in try/catch) -
  C7, C11.
- `docs/GUIDE.md`: one paragraph under "What every pane and the sidebar show (ours)" describing
  Cline's model + `in`/`out` and where they come from, plus one clarifying line in the pane-border
  bullet - docs, no test.

Checks: <each check of section 6 and its result>
- [x] `pwsh -NoProfile -File tests/test-hud-cline.ps1` -> `PASS 11/11`, exit 0 (C1-C11, incl. the
      three new ones: C9 completed/failed skipped, C10 newer live session replaces the cache,
      C11 `in` before `out`).
- [x] Against a temp HEAD copy (`git show HEAD:herdr-hud.ps1` -> `%TEMP%\hud-head-cline`,
      `-SourceRoot`): `PASS 0/11` - all 11 fail there (C1-C6, C8-C10: `Find-ClineSession` /
      `Get-ClineUsage` / `Get-ClinePids` not recognized; C7/C11: no `in`/`out` tokens). The temp
      copy was deleted afterwards.
- [x] `tests/test-batch-a.ps1` -> `PASS 48/48`; `tests/test-batch-b.ps1` -> `PASS 18/18`.
- [x] `herdr-hud.ps1` parses under pwsh (host 7.6.7): `Parser::ParseFile` -> 0 errors. File is still
      UTF-8 without BOM (first bytes `# P`) with its `·` separators intact. (A run under Windows
      PowerShell 5.1 reports 19 errors - all from the pre-existing `?:` ternary on line 296, which
      5.1 cannot parse; that line is HEAD's, untouched, and the HUD runs under pwsh.)
- [x] Real status values in `~\.cline\data\sessions` (read-only): `failed` 42, `completed` 16,
      `idle` 4, `running` 2 - so `exited`/`done` never occur and D1 was the important half; the 58
      dead sessions are exactly what the old cwd fallback would have shown.
- [x] `git status --porcelain` -> only my files (`herdr-hud.ps1`, `docs/GUIDE.md`,
      `tests/test-hud-cline.ps1`, this answer) plus Codex's (`herdr-agents.ps1`, `herdr-trial.ps1`,
      `tests/test-round2-agents.ps1`, `docs/DELEGATION.md`, `docs/FILES.md`,
      `skills/herdr-team/SKILL.md`), your `docs/TEAM-PLAYBOOK.md` (not touched, out of scope) and
      your task specs.
- [x] Extra, read-only (no pane metadata written): `Get-ClinePids` on the real pane returned the
      node pid + shell pid; `Find-ClineSession` with those pids and the pane's cwd returned my own
      session file; `Get-ClineUsage` on the real idle session `1792754391890_vxfxi` returned
      model `cline-pass/deepseek-v4.1-flash`, fresh `98755` (1359183 - 1260428), out `16808`.
- [x] Token order seen in the real `pane report-metadata` arguments: `--token path=C:\work |
      --token model=cline-pass/deepseek-v4.1-flash | --token in=250 | --token out=40` (C11 asserts
      this order, C7 asserts the values and that `ctx` is absent).

Not done / doubts: <list or none>
- D3 needed no code change and my earlier doubt was wrong: `[ordered]@{ path = ... }` is the only
  declaration, `$usage` stays `$null` for a Cline pane (line 289-299 only fills it for codex/claude),
  and `$tokens.in` is assigned immediately before `$tokens.out` - so `in` already preceded `out`.
  I added the ordering to the comment and locked it with C11 instead of moving code.
- Writing C11 exposed a harness bug, now fixed: `Reset-Fixture` did `$last = @{}` / `$lastSent = @{}`
  (plain assignment = local), so the HUD's script-scope dedupe state survived between tests and
  C11's pane was skipped as "unchanged". Now `$script:last` / `$script:lastSent`. C1-C8 are
  unaffected by this (they were already independent).
- The pid match rarely fires in practice: Cline's session `pid` is the shared `cline-hub-daemon`
  pid (5872 here), not the pane's `node.exe` pid, so the **cwd fallback** does the real work. Two
  Cline panes in the same folder can therefore show the same (newest) session's numbers.
- Reading of D2: while a cache entry is warm, only a **newer live session in the same folder**
  replaces it. A newer file that merely carries a pid from `$Pids` in a *different* folder does not
  (with the shared daemon pid that file usually belongs to another pane) - the pid rule still
  applies on the full scan, i.e. on first sight of a pane. Say so if you want pid matches to
  override a warm cache too.
- I added `Cwd` to the cache entry (`@{File;Pids;Cwd}`) so a pane that changes folder is not shown
  the old folder's session; D2's cost rule is unchanged (dir listing + `LastWriteTime` compare, JSON
  parsed only for files newer than the cached one).
- A session that has not written `metadata.usage` yet shows no `in`/`out` (my own running session
  is one); the numbers appear once Cline writes them.
- Values go through `Format-Tokens` (250 -> `250`, 98755 -> `98.8k`).
- Not done by design: no real HUD run, no git commit/push/reset/stash/checkout, no agents, no
  secrets, no browser, nothing outside section 3 touched (HISTORY/LESSONS untouched).
