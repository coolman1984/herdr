# Review: Goal engine M1 (Codex Sol high, read-only)

Read I:\herdr\AGENTS.md (delegate rules) and docs\SPEC-GOAL-ENGINE.md. Review `app/internal/goal` in your
checkout I:\herdr\.wt\goal-m1 (branch goal-engine). You are a different vendor from the writer on purpose.

Do NOT change code. Another agent is editing this folder at the same time; do not run `go build` output
into it, use your own `-o` temp path; ignore half-written files if a test fails oddly and say so.

This engine presses keys in live agent CLIs with no human present. Find what makes it press the wrong
thing, or hang, or quietly fail. Focus:
1. Trust and secret gates: can a screen, path, junction, or wrapped line get a wrong "yes"? Can a secret
   prompt get any answer? Can an agent's own output fake a prompt?
2. Sensing: false done, false dead, false stall; what happens on a Herdr error or a slow CLI.
3. Crash and resume: kill at each point of Tick; lost or doubled actions, double-started agents.
4. Loops with no cap; anything that needs a human when the owner's rule says it must not.
Already known, skip: rate-limit swap loop, stall hash over the timer, no-answer-file agents, status-only secret
match, acceptance run inside the tick, non-git goal folder.

Prove each finding: a failing test or a concrete input, not a guess. Findings first, by severity:
file:line, what happens, input, fix. Write to I:\herdr\.wt\goal-m1\tasks\goal-m1-review.md, then answer "DONE".
List what you could not verify under "doubts".
