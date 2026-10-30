package goal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBudgetFollowsTheMeasuredMachine(t *testing.T) {
	cases := []struct {
		free   int
		cpus   float64
		agents int
	}{
		{15000, 4, 2}, // strong: the ceiling
		{4200, 4, 2},  // (4200-2636)/600 = 2
		{3500, 4, 1},  // one agent
		{2600, 8, 0},  // too little: defer
		{-1, 4, 1},    // unknown RAM: one agent, never "unlimited"
		{15000, 1, 1}, // one CPU: one agent
	}
	for _, c := range cases {
		b := DeriveBudget(c.free, c.cpus, 2)
		if b.MaxAgents != c.agents {
			t.Errorf("free %d cpus %.0f: %d agents (%s), want %d", c.free, c.cpus, b.MaxAgents, b.Reason, c.agents)
		}
		if b.MaxProcs != b.MaxAgents+2 || b.BatchRows < 500 {
			t.Errorf("procs/batch: %+v", b)
		}
	}
}

func TestDiscoverMarksUnknownAndReadsDevicePolicy(t *testing.T) {
	dir := t.TempDir()
	pol := filepath.Join(dir, "device-policy.json")
	os.WriteFile(pol, []byte(`{"profile":"company_nasca","allow_packages":false}`), 0o644)
	r := Discover(dir, pol)
	if r.Policy.Profile != "company_nasca" || r.Policy.AllowPackages == nil || *r.Policy.AllowPackages {
		t.Fatalf("policy %+v", r.Policy)
	}
	if r.Tools["git"].Status != CapTested {
		t.Fatalf("git: %+v", r.Tools["git"])
	}
	if f := r.Tools["no-such-tool-xyz"]; f.Status != "" {
		t.Fatal("invented a tool")
	}
	r2 := Discover(dir, filepath.Join(dir, "missing.json"))
	if r2.Policy.Profile != "unknown" {
		t.Fatalf("missing policy must be unknown, got %q", r2.Policy.Profile)
	}
}
