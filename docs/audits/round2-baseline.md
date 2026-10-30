# Round-2 baseline on 4e914a3 (merge of codex/goal-audit-round2-20261030 into cd7b35e), 2026-10-30T18:25:37Z
```
--- FAIL: TestIndependentRound2StaleExhaustionWithoutResetIsNotFree (0.00s)
    independent_round2_test.go:14: stale 99% usage with no reset became eligible
--- FAIL: TestIndependentRound2RemainingZeroIsNotFree (0.00s)
    independent_round2_test.go:23: zero remaining credits became eligible
--- FAIL: TestIndependentRound2MissingUsageIsLowTrust (0.00s)
    independent_round2_test.go:32: a named window with no usage became a trusted free account
--- FAIL: TestIndependentRound2ZeroAgentBudgetDefersWork (0.17s)
    independent_round2_test.go:47: zero cap accepted an agent: 
--- FAIL: TestIndependentRound2ResultWaitsForConfirmedStop (0.30s)
    independent_round2_test.go:60: live writer detached and work advanced: state=checking taskAgent="" agentState=stop-pending
--- FAIL: TestIndependentRound2QuotaRotationWaitsForConfirmedStop (0.20s)
    independent_round2_test.go:77: 2 workers own the same task after unconfirmed stop
FAIL
FAIL	herdrdeck/internal/goal	0.677s
FAIL
FAIL: test_com_1904_workbook_date_system_is_respected (tests.test_independent_round2.IndependentRound2.test_com_1904_workbook_date_system_is_respected)
AssertionError: '1899-12-31' != '1904-01-02'
FAIL: test_company_authorized_route_does_not_open_source_bytes (tests.test_independent_round2.IndependentRound2.test_company_authorized_route_does_not_open_source_bytes)
PermissionError: raw access denied; authorized application may open
AssertionError: protected source was opened before authorized app routing: raw access denied; authorized application may open
FAIL: test_company_csv_requires_authorized_application (tests.test_independent_round2.IndependentRound2.test_company_csv_requires_authorized_application)
AssertionError: 'native' not found in ('com', 'app-pdf', 'skip') : stdlib reader
FAIL: test_duplicate_headers_never_silently_discard_values (tests.test_independent_round2.IndependentRound2.test_duplicate_headers_never_silently_discard_values)
AssertionError: [] is not true : two source values became one report value, marked done with no issue
FAIL: test_iso_date_cell_is_not_accepted_as_empty (tests.test_independent_round2.IndependentRound2.test_iso_date_cell_is_not_accepted_as_empty)
AssertionError: None != '2026-10-30'
FAIL: test_large_decimal_total_keeps_cents (tests.test_independent_round2.IndependentRound2.test_large_decimal_total_keeps_cents)
AssertionError: Decimal('1.234567890123456789012345679E+29') != Decimal('123456789012345678901234567890.01')
FAIL: test_quoted_csv_newline_is_preserved (tests.test_independent_round2.IndependentRound2.test_quoted_csv_newline_is_preserved)
AssertionError: 'helloworld' != 'hello\nworld'
Ran 7 tests in 0.082s
FAILED (failures=7)
```
