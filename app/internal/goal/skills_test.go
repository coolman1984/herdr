package goal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoCatalog(t *testing.T) *SkillCatalog {
	_, me, _, _ := runtime.Caller(0)
	c, err := LoadSkillCatalog(filepath.Join(filepath.Dir(me), "..", "..", "..", "skills", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSkillSelectionFollowsPolicyAndMachine(t *testing.T) {
	c := repoCatalog(t)
	py := map[string]Fact{"python3": {Status: CapTested}}
	got, skipped := c.Select([]string{"excel", "totals", "debug"}, py, "company_nasca")
	if len(got) != 1 || got[0].Name != "herdr-documents" || got[0].Hash == "" {
		t.Fatalf("chosen %+v", got)
	}
	if !strings.Contains(strings.Join(skipped, ";"), "systematic-debugging: candidate-not-evaluated") {
		t.Fatalf("an unevaluated candidate must be reported as left out: %v", skipped)
	}
	if got, skipped := c.Select([]string{"excel"}, map[string]Fact{}, "personal"); len(got) != 0 || !strings.Contains(strings.Join(skipped, ";"), "needs python3|python|py") {
		t.Fatalf("skill chosen on a machine without its tools: %+v %v", got, skipped)
	}
	if got, _ := c.Select([]string{"nothing-matches"}, py, "personal"); len(got) != 0 {
		t.Fatalf("unrelated task got skills: %+v", got)
	}
}

func TestBriefCarriesChosenSkillsAndTaskRecordsThem(t *testing.T) {
	r := newRig(t, Task{ID: "T1", Title: "totals", Spec: "sum the sheets", Tags: []string{"excel", "totals"}, Accept: []string{"true"}})
	r.e.Skills = repoCatalog(t)
	r.e.Caps = &CapReport{Tools: map[string]Fact{"python3": {Status: CapTested}}, Policy: DevicePolicy{Profile: "personal"}}
	r.tick()
	a := r.agentOf("T1")
	b, _ := os.ReadFile(filepath.Join(r.e.L.Dir, "briefs", a.Name+".md"))
	if !strings.Contains(string(b), "herdr-documents") || len(r.e.G.Task("T1").Skills) != 1 {
		t.Fatalf("brief without the chosen skill:\n%s", b)
	}
}
