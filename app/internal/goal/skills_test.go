package goal

import (
	"crypto/sha256"
	"encoding/hex"
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
	if len(got) != 2 || got[0].Name != "herdr-environment-preflight" || got[1].Name != "herdr-documents" || got[1].Hash == "" {
		t.Fatalf("chosen %+v", got)
	}
	if !strings.Contains(strings.Join(skipped, ";"), "systematic-debugging: candidate-not-evaluated") {
		t.Fatalf("an unevaluated candidate must be reported as left out: %v", skipped)
	}
	if got, skipped := c.Select([]string{"excel"}, map[string]Fact{}, "personal"); len(got) != 1 || !strings.Contains(strings.Join(skipped, ";"), "needs python3|python|py") {
		t.Fatalf("skill chosen on a machine without its tools: %+v %v", got, skipped)
	}
	if got, _ := c.Select([]string{"nothing-matches"}, py, "personal"); len(got) != 1 || got[0].Name != "herdr-environment-preflight" {
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
	if !strings.Contains(string(b), "herdr-documents") || !strings.Contains(string(b), "herdr-environment-preflight") || len(r.e.G.Task("T1").Skills) != 2 {
		t.Fatalf("brief without the chosen skill:\n%s", b)
	}
}

func TestMandatoryPolicyAlwaysAndPinnedHashesEnforced(t *testing.T) {
	c := repoCatalog(t)
	got, _ := c.Select([]string{"web"}, map[string]Fact{}, "personal")
	if len(got) == 0 || got[0].Name != "herdr-environment-preflight" {
		t.Fatalf("mandatory policy skill missing for an unrelated task: %+v", got)
	}
	// a skill whose file changed after pinning is refused
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "skills", "x"), 0o755)
	os.WriteFile(filepath.Join(dir, "skills", "x", "SKILL.md"), []byte("changed"), 0o644)
	os.WriteFile(filepath.Join(dir, "evil.md"), []byte("outside"), 0o644)
	cat := filepath.Join(dir, "skills", "catalog.json")
	os.WriteFile(cat, []byte(`{"skills":[
	 {"name":"x","path":"skills/x/SKILL.md","state":"approved","tags":["a"],"sha256":"00"},
	 {"name":"y","path":"skills/../../`+filepath.Base(dir)+`/evil.md","state":"approved","tags":["a"],"sha256":"00"}]}`), 0o644)
	c2, err := LoadSkillCatalog(cat)
	if err != nil {
		t.Fatal(err)
	}
	got, skipped := c2.Select([]string{"a"}, map[string]Fact{}, "personal")
	if len(got) != 0 || !strings.Contains(strings.Join(skipped, ";"), "pinned sha256") {
		t.Fatalf("changed/unpinned skill used: %+v %v", got, skipped)
	}
}

func TestCatalogIsFoundWithoutAnEnvironmentVariable(t *testing.T) {
	proj := t.TempDir()
	os.MkdirAll(filepath.Join(proj, "skills"), 0o755)
	os.WriteFile(filepath.Join(proj, "skills", "catalog.json"), []byte(`{"skills":[]}`), 0o644)
	t.Setenv("HERDR_SKILLS_CATALOG", "")
	if got := FindSkillCatalog(proj); got != filepath.Join(proj, "skills", "catalog.json") {
		t.Fatalf("catalog not discovered: %q", got)
	}
}

func TestSkillBudgetBoundsTheBrief(t *testing.T) {
	c := repoCatalog(t)
	old := MaxSkillBytes
	MaxSkillBytes = 10
	defer func() { MaxSkillBytes = old }()
	got, skipped := c.Select([]string{"excel"}, map[string]Fact{"python3": {Status: CapTested}}, "personal")
	for _, g := range got {
		if g.Name == "herdr-documents" {
			t.Fatal("optional skill over the byte budget was added")
		}
	}
	if !strings.Contains(strings.Join(skipped, ";"), "skill budget") {
		t.Fatalf("budget refusal not explained: %v", skipped)
	}
}

func TestSkillPathsMustStayInsideTheCatalog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture")
	}
	root, outside := t.TempDir(), t.TempDir()
	body := []byte("outside skill")
	os.WriteFile(filepath.Join(outside, "evil.md"), body, 0o644)
	os.MkdirAll(filepath.Join(root, "skills"), 0o755)
	os.Symlink(filepath.Join(outside, "evil.md"), filepath.Join(root, "skills", "link.md"))
	sum := sha256.Sum256(body)
	h := hex.EncodeToString(sum[:])
	rel, _ := filepath.Rel(root, filepath.Join(outside, "evil.md"))
	cat := filepath.Join(root, "skills", "catalog.json")
	os.WriteFile(cat, []byte(`{"skills":[
	 {"name":"up","path":"`+filepath.ToSlash(rel)+`","state":"approved","tags":["a"],"sha256":"`+h+`"},
	 {"name":"link","path":"skills/link.md","state":"approved","tags":["a"],"sha256":"`+h+`"}]}`), 0o644)
	c, err := LoadSkillCatalog(cat)
	if err != nil {
		t.Fatal(err)
	}
	got, skipped := c.Select([]string{"a"}, map[string]Fact{}, "personal")
	if len(got) != 0 || strings.Count(strings.Join(skipped, ";"), "leaves the catalog folder") != 2 {
		t.Fatalf("a skill outside the catalog was used: %+v %v", got, skipped)
	}
}

// Windows CI (run 41171161928): a CRLF checkout of an unchanged skill must still match its pin.
func TestPinnedHashSurvivesCRLFCheckout(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "skills", "s"), 0o755)
	lf := []byte("line one\nline two\n")
	sum := sha256.Sum256(lf)
	os.WriteFile(filepath.Join(root, "skills", "s", "SKILL.md"), []byte("line one\r\nline two\r\n"), 0o644)
	cat := filepath.Join(root, "skills", "catalog.json")
	os.WriteFile(cat, []byte(`{"skills":[{"name":"s","path":"skills/s/SKILL.md","state":"approved","tags":["a"],"sha256":"`+hex.EncodeToString(sum[:])+`"}]}`), 0o644)
	c, _ := LoadSkillCatalog(cat)
	if got, skipped := c.Select([]string{"a"}, map[string]Fact{}, "personal"); len(got) != 1 {
		t.Fatalf("CRLF copy of an unchanged skill refused: %v", skipped)
	}
	os.WriteFile(filepath.Join(root, "skills", "s", "SKILL.md"), []byte("line one\r\nline 2\r\n"), 0o644)
	if got, _ := c.Select([]string{"a"}, map[string]Fact{}, "personal"); len(got) != 0 {
		t.Fatal("a changed skill was accepted")
	}
}
