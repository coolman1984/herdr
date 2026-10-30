package goal

import (
    "testing"
    "time"
)

// These specifications expose behavior beyond the six original audit tests.
func TestIndependentRound2StaleExhaustionWithoutResetIsNotFree(t *testing.T) {
    now := time.Now()
    v := Judge(QuotaReading{State: QuotaKnown, Observed: now.Add(-2*time.Hour),
        Limits: []Limit{{Kind: "weekly", UsedPct: pct(99)}}}, now)
    if v.Eligible { t.Fatal("stale 99% usage with no reset became eligible") }
}

func TestIndependentRound2RemainingZeroIsNotFree(t *testing.T) {
    now := time.Now()
    v := Judge(QuotaReading{State: QuotaKnown, Observed: now,
        Limits: []Limit{{Kind: "credits", Left: pct(0), Unit: "credits"}}}, now)
    if v.Eligible { t.Fatal("zero remaining credits became eligible") }
}

func TestIndependentRound2MissingUsageIsLowTrust(t *testing.T) {
    now := time.Now()
    v := Judge(QuotaReading{State: QuotaKnown, Observed: now,
        Limits: []Limit{{Kind: "weekly"}}}, now)
    if v.Eligible && !v.LowTrust { t.Fatal("a named window with no usage became a trusted free account") }
}

func TestIndependentRound2ZeroAgentBudgetDefersWork(t *testing.T) {
    r := newRig(t, task("T1"))
    budget := DeriveBudget(100, 1, 2)
    if budget.MaxAgents != 0 { t.Fatalf("fixture expected zero budget: %+v", budget) }
    r.e.G.Limits.MaxAgents = budget.MaxAgents
    r.e.G.Limits.MaxProcs = budget.MaxProcs
    // Memory readings can become unavailable after preflight. The explicit zero cap must survive.
    r.ram = -1
    if ok, why := r.e.canStart(); ok { t.Fatalf("zero cap accepted an agent: %s", why) }
}

func TestIndependentRound2ResultWaitsForConfirmedStop(t *testing.T) {
    r := newRig(t, task("T1"))
    r.tick()
    a := r.agentOf("T1")
    r.finish("T1", Answer{Status:"done", Summary:"work done"})
    r.e.B = stuckStop{r.b}
    r.e.finished(a, &Answer{Status:"done", Summary:"work done"})
    tt := r.e.G.Task("T1")
    if !a.Ended && (tt.Agent != a.Name || tt.State == TaskChecking || tt.State == TaskMerging || tt.State == TaskDone) {
        t.Fatalf("live writer detached and work advanced: state=%s taskAgent=%q agentState=%s", tt.State, tt.Agent, a.State)
    }
}

func TestIndependentRound2QuotaRotationWaitsForConfirmedStop(t *testing.T) {
    r := newRig(t, task("T1"))
    r.tick()
    a := r.agentOf("T1")
    r.e.B = stuckStop{r.b}
    r.e.accountLimited(a, Sight{Reset:r.now.Add(time.Hour), Line:"limit reached"}, "limit reached")
    active := 0
    for _, worker := range r.e.G.Agents {
        if worker.Task == a.Task && !worker.Ended { active++ }
    }
    if active != 1 { t.Fatalf("%d workers own the same task after unconfirmed stop", active) }
}
