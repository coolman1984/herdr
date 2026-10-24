# AGENTS.md

Codex, Cline, Antigravity, OpenCode and every other agent: the rules for this folder are in
[CLAUDE.md](CLAUDE.md). Read it completely, then [HISTORY.md](HISTORY.md) and
[LESSONS.md](LESSONS.md), before changing anything.

## If you were started by Claude (the lead) through `herdr-agents.ps1`

You are a delegate. Your task spec (a file in `tasks/`) is the contract; where it is silent,
these rules apply.

1. **Deliver working, verified results, not a plan.** Implement, run the checks, fix, then
   answer. Act on reasonable assumptions; stop only when truly blocked (sign-in, approval,
   a question only the owner can answer) and say BLOCKED.
2. **Stay in scope.** Change only the files the spec lists. Never revert, reformat or "tidy"
   changes you did not make; other agents may work in the same folder. If you see unexpected
   changes, leave them and mention them.
3. **Prove every claim by running it.** A finding, a cause or "it works" needs the command and
   its output. Write "not verified" when you could not run something. Never state a cause you
   only reasoned about as a fact.
4. **Tests must fail on the old code and test behaviour.** Run your tests against the previous
   version (`git show HEAD:<file>` into a temp folder) and report how many fail there. Never
   test for the presence of a code line.
5. **No silent failure.** No broad `try { } catch { }` that hides errors, no fallbacks that
   pretend success, no swallowed exit codes. Every error reaches the caller with a clear message.
6. **Copy the surrounding style.** Reuse existing helpers (`Invoke-Herdr`, `Fail`, ...) before
   writing new ones. PowerShell 7; `-LiteralPath`; absolute herdr path; never shadow automatic
   variables (`$profile`, `$args`, `$input`, `$matches`, `$error`, `$host`; names ignore case).
7. **Safety.** Never read or print secrets (`auth.json`, credentials, tokens). Never touch the
   owner's live Herdr session except as the spec allows; test in an isolated session. No git
   commit / push / reset / stash / checkout unless the spec says so. Do not start other agents.
   **No browser tools** (no Chrome/Edge automation, no browser extension, no opening web pages):
   the PC is short on RAM (LESSONS #56). Use the CLI's own server-side web search if research is needed.
8. **Do not loop.** If you re-read or re-edit the same files twice without progress, stop and
   report what blocks you and the one question that would unblock you.
9. **Answer exactly in the spec's format**, findings first and ordered by severity for reviews,
   and list anything you could not verify under "doubts". Be brief.
