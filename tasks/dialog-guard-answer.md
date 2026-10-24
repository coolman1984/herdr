DONE
Changed: herdr-agents.ps1 (Get-AgentDialog: trust, browser and model need a selection footer (`enter to confirm|enter confirm|navigate|esc to`) in the last 6 non-empty lines, with the phrase in the last 20; sign-in needs its phrase in the last 6 and no footer; trust = `do you trust|trust this folder`). tests/test-dialog-guard.ps1 (fixtures given real footers; new tests D1 live-answer fixture, D6 footer window, D7 no-footer, D8 sign-in window, A7 live answer through ask). tests/test-round2-agents.ps1 unchanged since last round. No commit. HISTORY.md and LESSONS.md not touched. The fixture tests/fixture-claude-answer-mentions-trust.txt was used as given, not changed.
Tests: test-dialog-guard.ps1 - suite 26/26; on HEAD: 23 fail (A4, A5 and A7 pass on HEAD by design: normal asks must still send). test-round2-agents.ps1 - suite 18/18 (on HEAD sources with this test file: 3 fail, unchanged from last round). test-batch-a.ps1 - suite 48/48. herdr-agents.ps1 parses with 0 errors.
Mutation: with the footer rule removed in a scratch copy, 4 tests fail (D1 live answer, D6 footer window, D7, A7), so the live-answer tests catch the false positive.
Doubts:
- Codex trust footer: the Codex fixture's footer ("Enter to confirm · Esc to cancel") is assumed, not captured. If the real Codex trust screen has no footer from the list, it will not be blocked. Needs the real screen.
- `esc to` also matches "esc to interrupt", which Claude shows while working. A working Claude whose last 6 lines show that footer and whose last 20 lines quote "do you trust" or "pick a model" would still be blocked. The live fixture does not hit this.
- The model-pick footer in the test is Antigravity's footer, not a captured model-picker screen.
- When a pattern matches several lines, the top-most matching line is returned (spec silent).
- The check is a snapshot: a dialog that appears between the read and the prompt is not caught.
- The live `ask haiku` check is the lead's; I did not run a live ask.
- check.ps1 was not rerun this round. Last run: 1 FAIL, "docs/FILES.md lists 'herdr/' but it does not exist". Pre-existing: herdr/ exists only in the main checkout, not in this worktree.
