# Independent round-two evidence, 2026-10-30

Product source: cd7b35e5f27fa4e87c4adc2febbc89aae0f5760b
Test commit: a3a2a81e4a95b9cd4fd38fb3cfd2050816c209dd
Run: https://github.com/coolman1984/herdr/actions/runs/41103968280
Linux job: 124228199682
Windows job: 124228200368

Outcome: six NEW Go failures and seven NEW Python failures on BOTH platforms; original six independent Go checks pass; original goal tests, build and 42 web behavioral tests pass. Python runs 21 cases with 7 failures and 1 LibreOffice skip. Windows Office applications are NOT registered. No real Office/NASCA/authenticated-agent certification is claimed.

These excerpts are selected from retrieved job logs; complete logs/artifacts are linked from the run. No product code was modified by this review.

## Linux

```text
ok  	herdrdeck/internal/goal	19.407s
--- PASS: TestIndependentAuditExitedAgentGetsNoInput (0.06s)
--- PASS: TestIndependentAuditBlockedReviewCannotApprove (0.04s)
--- PASS: TestIndependentAuditCancelledPrerequisiteIsNotSuccess (0.04s)
--- PASS: TestIndependentAuditDirectoryOwnershipIncludesChildren (0.03s)
--- PASS: TestIndependentAuditConsumerRejectsDelegateStop (0.03s)
--- PASS: TestIndependentAuditInvalidUpdateLeavesWorkerRunning (0.06s)
=== RUN   TestIndependentRound2StaleExhaustionWithoutResetIsNotFree
    independent_round2_test.go:13: stale 99% usage with no reset became eligible
--- FAIL: TestIndependentRound2StaleExhaustionWithoutResetIsNotFree (0.00s)
=== RUN   TestIndependentRound2RemainingZeroIsNotFree
    independent_round2_test.go:20: zero remaining credits became eligible
--- FAIL: TestIndependentRound2RemainingZeroIsNotFree (0.00s)
=== RUN   TestIndependentRound2MissingUsageIsLowTrust
    independent_round2_test.go:27: a named window with no usage became a trusted free account
--- FAIL: TestIndependentRound2MissingUsageIsLowTrust (0.00s)
=== RUN   TestIndependentRound2ZeroAgentBudgetDefersWork
    independent_round2_test.go:38: zero cap accepted an agent: 
--- FAIL: TestIndependentRound2ZeroAgentBudgetDefersWork (0.03s)
=== RUN   TestIndependentRound2ResultWaitsForConfirmedStop
    independent_round2_test.go:50: live writer detached and work advanced: state=checking taskAgent="" agentState=stop-pending
--- FAIL: TestIndependentRound2ResultWaitsForConfirmedStop (0.06s)
=== RUN   TestIndependentRound2QuotaRotationWaitsForConfirmedStop
    independent_round2_test.go:64: 2 workers own the same task after unconfirmed stop
--- FAIL: TestIndependentRound2QuotaRotationWaitsForConfirmedStop (0.05s)
ℹ tests 42
ℹ pass 42
ℹ fail 0
Ran 21 tests in 1.123s
FAILED (failures=7, skipped=1)
```

## Windows

```text
ok  	herdrdeck/internal/goal	98.059s
--- PASS: TestIndependentAuditExitedAgentGetsNoInput (0.82s)
--- PASS: TestIndependentAuditBlockedReviewCannotApprove (0.63s)
--- PASS: TestIndependentAuditCancelledPrerequisiteIsNotSuccess (0.58s)
--- PASS: TestIndependentAuditDirectoryOwnershipIncludesChildren (0.61s)
--- PASS: TestIndependentAuditConsumerRejectsDelegateStop (0.61s)
--- PASS: TestIndependentAuditInvalidUpdateLeavesWorkerRunning (0.79s)
=== RUN   TestIndependentRound2StaleExhaustionWithoutResetIsNotFree
    independent_round2_test.go:13: stale 99% usage with no reset became eligible
--- FAIL: TestIndependentRound2StaleExhaustionWithoutResetIsNotFree (0.00s)
=== RUN   TestIndependentRound2RemainingZeroIsNotFree
    independent_round2_test.go:20: zero remaining credits became eligible
--- FAIL: TestIndependentRound2RemainingZeroIsNotFree (0.00s)
=== RUN   TestIndependentRound2MissingUsageIsLowTrust
    independent_round2_test.go:27: a named window with no usage became a trusted free account
--- FAIL: TestIndependentRound2MissingUsageIsLowTrust (0.00s)
=== RUN   TestIndependentRound2ZeroAgentBudgetDefersWork
    independent_round2_test.go:38: zero cap accepted an agent: 
--- FAIL: TestIndependentRound2ZeroAgentBudgetDefersWork (0.59s)
=== RUN   TestIndependentRound2ResultWaitsForConfirmedStop
    independent_round2_test.go:50: live writer detached and work advanced: state=checking taskAgent="" agentState=stop-pending
--- FAIL: TestIndependentRound2ResultWaitsForConfirmedStop (1.08s)
=== RUN   TestIndependentRound2QuotaRotationWaitsForConfirmedStop
    independent_round2_test.go:64: 2 workers own the same task after unconfirmed stop
--- FAIL: TestIndependentRound2QuotaRotationWaitsForConfirmedStop (0.85s)
ℹ tests 42
ℹ pass 42
ℹ fail 0
Ran 21 tests in 3.338s
FAILED (failures=7, skipped=1)
Excel.Application registered: False
Word.Application registered: False
PowerPoint.Application registered: False
```
