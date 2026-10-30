package goal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func pct(v float64) *float64 { return &v }

func TestJudgeAtTheNinetyPercentEdge(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	read := func(fh, wk *float64) QuotaReading {
		r := QuotaReading{State: QuotaKnown, Observed: now.Add(-time.Minute)}
		if fh != nil {
			r.Limits = append(r.Limits, Limit{Kind: "five_hour", UsedPct: fh, ResetsAt: now.Add(time.Hour)})
		}
		if wk != nil {
			r.Limits = append(r.Limits, Limit{Kind: "weekly", UsedPct: wk, ResetsAt: now.Add(72 * time.Hour)})
		}
		return r
	}
	cases := []struct {
		name     string
		r        QuotaReading
		eligible bool
		low      bool
	}{
		{"89.9", read(pct(89.9), pct(10)), true, false},
		{"90.0", read(pct(90), pct(10)), false, false},
		{"90.1 weekly", read(pct(5), pct(90.1)), false, false},
		{"weekly missing", read(pct(50), nil), true, false},
		{"unknown", QuotaReading{State: QuotaUnknown}, true, true},
		{"unsupported", QuotaReading{State: QuotaUnsupported}, true, true},
		{"stale", QuotaReading{State: QuotaKnown, Observed: now.Add(-2 * time.Hour), Limits: []Limit{{Kind: "five_hour", UsedPct: pct(10)}}}, true, true},
		{"stale but at limit", QuotaReading{State: QuotaKnown, Observed: now.Add(-2 * time.Hour), Limits: []Limit{{Kind: "five_hour", UsedPct: pct(95), ResetsAt: now.Add(time.Hour)}}}, false, false},
	}
	for _, c := range cases {
		v := Judge(c.r, now)
		if v.Eligible != c.eligible || v.LowTrust != c.low {
			t.Errorf("%s: eligible=%v low=%v (%s), want %v %v", c.name, v.Eligible, v.LowTrust, v.Reason, c.eligible, c.low)
		}
	}
	// a window that reset after the reading is no longer used
	r := read(pct(99), nil)
	r.Limits[0].ResetsAt = now.Add(-time.Minute)
	if !Judge(r, now).Eligible {
		t.Error("a reset window still blocks")
	}
}

func rigQuotas(r *rig, used map[string]float64) {
	src := fixedSource{used: used, now: func() time.Time { return r.now }}
	r.e.Q = &Quotas{Sources: map[string]QuotaSource{"a": src, "b": src}}
}

func TestAccountAtNinetyPercentGetsNoNewWork(t *testing.T) {
	r := newRig(t, task("T1"))
	rigQuotas(r, map[string]float64{"a1": 90})
	r.tick()
	if a := r.agentOf("T1"); a == nil || a.Account != "a2" {
		t.Fatalf("started on %+v, want a2 (a1 is at 90%%)", a)
	}
}

func TestProfilesOnOneSubscriptionShareItsBudget(t *testing.T) {
	r := newRig(t, task("T1"))
	alias := r.e.G.Profiles[0]
	alias.Name, alias.Accounts = "alpha-fast", []string{"a1"}
	r.e.G.Profiles = append([]Profile{alias}, r.e.G.Profiles...) // same provider "a", same account a1
	r.e.G.Task("T1").Writer = "alpha-fast"
	rigQuotas(r, map[string]float64{"a1": 95})
	r.tick()
	if a := r.agentOf("T1"); a == nil || a.Account == "a1" {
		t.Fatalf("an alias profile used the exhausted account: %+v", a)
	}
}

func TestUnknownQuotaMeansOneAgentPerAccount(t *testing.T) {
	r := newRig(t, task("T1"), task("T2"), task("T3"))
	r.e.G.Profiles[0].Accounts = []string{"a1"}
	r.e.G.Profiles[1].Accounts = []string{"b1"}
	r.e.G.Limits.MaxAgents, r.e.G.Limits.MaxProcs = 5, 9
	rigQuotas(r, map[string]float64{"a1": -1, "b1": -1}) // no readings
	r.tick()
	per := map[string]int{}
	for _, a := range r.e.G.Agents {
		if !a.Ended {
			per[a.Account]++
		}
	}
	if per["a1"] > 1 || per["b1"] > 1 {
		t.Fatalf("unknown limits treated as free: %v", per)
	}
}

func TestRunningAgentLeavesAnAccountThatReachesNinety(t *testing.T) {
	r := newRig(t, task("T1"))
	used := map[string]float64{"a1": 50}
	rigQuotas(r, used)
	r.tick()
	a := r.agentOf("T1")
	used["a1"] = 91
	r.tick()
	b := r.agentOf("T1")
	if !a.Ended || b == nil || b.Account == "a1" {
		t.Fatalf("agent kept working on an account at 91%%: old ended=%v new=%+v", a.Ended, b)
	}
}

func TestClaudeStatusFilesReading(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "s1.json"), []byte(`{"config_dir":"C:\\Users\\o\\.herdr-accounts\\claude\\2040","rate_limits":{"five_hour":{"used_percentage":91.5,"resets_at":1793400000},"seven_day":{"used_percentage":40,"resets_at":1793900000}}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "s2.json"), []byte(`{"config_dir":"","rate_limits":{"five_hour":{"used_percentage":12,"resets_at":1793400000}}}`), 0o644)
	r := ClaudeStatusFiles{Dir: dir}.Read("2040")
	if r.State != QuotaKnown || len(r.Limits) != 2 || r.Limits[0].Kind != "five_hour" || *r.Limits[0].UsedPct != 91.5 || r.Limits[1].Kind != "weekly" {
		t.Fatalf("%+v", r)
	}
	if m := (ClaudeStatusFiles{Dir: dir}).Read("main"); m.State != QuotaKnown || *m.Limits[0].UsedPct != 12 {
		t.Fatalf("main: %+v", m)
	}
	if u := (ClaudeStatusFiles{Dir: dir}).Read("1984"); u.State != QuotaUnknown {
		t.Fatalf("an account with no file must be unknown, not 0%%: %+v", u)
	}
}

func TestCodexSessionReading(t *testing.T) {
	home := t.TempDir()
	d := filepath.Join(home, "sessions", "2026", "10", "30")
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "rollout-x.jsonl"), []byte(`{"timestamp":"2026-10-30T11:00:00Z","type":"event_msg","payload":{"type":"token_count"}}
{"timestamp":"2026-10-30T11:50:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":93.0,"window_minutes":300,"resets_in_seconds":1200},"secondary":{"used_percent":20.0,"window_minutes":10080,"resets_in_seconds":200000}}}}
`), 0o644)
	r := CodexSessions{Home: func(string) string { return home }}.Read("default")
	if r.State != QuotaKnown || len(r.Limits) != 2 || r.Limits[0].Kind != "five_hour" || *r.Limits[0].UsedPct != 93 {
		t.Fatalf("%+v", r)
	}
	if !r.Limits[0].ResetsAt.Equal(time.Date(2026, 10, 30, 12, 10, 0, 0, time.UTC)) {
		t.Fatalf("reset %v", r.Limits[0].ResetsAt)
	}
}
