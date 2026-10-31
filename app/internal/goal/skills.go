package goal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Skill selection (skills-vendor/POLICY.md): approved skills only, filtered by what the measured machine
// has and the device profile allows, ranked by how many of the task's tags they serve, at most MaxSkills
// per task. The chosen skills and their content hashes are written into the brief and kept on the task.

const MaxSkills = 3

type SkillEntry struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	State     string   `json:"state"`
	License   string   `json:"license"`
	Tags      []string `json:"tags"`
	Requires  []string `json:"requires"`
	Profiles  []string `json:"profiles"`
	Evidence  string   `json:"evidence"`
	SHA256    string   `json:"sha256"`    // pinned full content hash
	Mandatory bool     `json:"mandatory"` // company policy: always injected
}

// MaxSkillBytes bounds what skills add to one brief's context (mandatory ones included).
var MaxSkillBytes = 64 << 10

// FindSkillCatalog looks for the trusted catalog: HERDR_SKILLS_CATALOG, the project, HERDR_HOME, then the
// folders above the herdr-goal executable. "" when none exists.
func FindSkillCatalog(project string) string {
	cands := []string{os.Getenv("HERDR_SKILLS_CATALOG")}
	if project != "" {
		cands = append(cands, filepath.Join(project, "skills", "catalog.json"))
	}
	if h := os.Getenv("HERDR_HOME"); h != "" {
		cands = append(cands, filepath.Join(h, "skills", "catalog.json"))
	}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		for i := 0; i < 5; i++ {
			cands = append(cands, filepath.Join(d, "skills", "catalog.json"))
			d = filepath.Dir(d)
		}
	}
	for _, c := range cands {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

type SkillCatalog struct {
	Root   string // repository root the paths are relative to
	Skills []SkillEntry
}

func LoadSkillCatalog(path string) (*SkillCatalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c struct {
		Skills []SkillEntry `json:"skills"`
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	root, err := filepath.EvalSymlinks(filepath.Dir(filepath.Dir(path)))
	if err != nil {
		return nil, err
	}
	return &SkillCatalog{Root: root, Skills: c.Skills}, nil
}

// ChosenSkill is what a brief gets.
type ChosenSkill struct {
	Name, Path, Hash, Why string
}

// Select picks skills for a task. tools: capability facts (name -> status); profile: device profile.
// Returns the chosen ones and why each other skill was left out (for the event log).
// verified returns a skill's real path and content only if it lies inside the catalog's root (links
// resolved) and its full content hash equals the pinned one.
func (c *SkillCatalog) verified(e SkillEntry) (string, []byte, error) {
	p, err := filepath.EvalSymlinks(filepath.Join(c.Root, filepath.FromSlash(e.Path)))
	if err != nil {
		return "", nil, fmt.Errorf("file missing")
	}
	if rel, err := filepath.Rel(c.Root, p); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", nil, fmt.Errorf("path leaves the catalog folder (%s)", p)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", nil, err
	}
	// Hash the text with LF line ends: a Windows checkout (core.autocrlf) turns LF into CRLF and must not
	// make an unchanged skill look tampered with. Any other change still breaks the pin.
	sum := sha256.Sum256(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")))
	if e.SHA256 == "" || hex.EncodeToString(sum[:]) != e.SHA256 {
		return "", nil, fmt.Errorf("content does not match its pinned sha256 (changed or unpinned): not used")
	}
	return p, b, nil
}

func (c *SkillCatalog) Select(tags []string, tools map[string]Fact, profile string) ([]ChosenSkill, []string) {
	type cand struct {
		e     SkillEntry
		score int
	}
	var cands []cand
	var skipped []string
	want := map[string]bool{}
	for _, t := range tags {
		want[strings.ToLower(t)] = true
	}
	for _, e := range c.Skills {
		score := 0
		for _, t := range e.Tags {
			if want[strings.ToLower(t)] {
				score++
			}
		}
		switch {
		case score == 0:
			continue
		case e.State != "approved":
			skipped = append(skipped, e.Name+": "+e.State+" (never activated automatically)")
			continue
		case len(e.Profiles) > 0 && !has(e.Profiles, profile):
			skipped = append(skipped, e.Name+": not allowed on profile "+profile)
			continue
		}
		missing := ""
		for _, req := range e.Requires {
			ok := false
			for _, alt := range strings.Split(req, "|") {
				if f, found := tools[alt]; found && f.Status == CapTested {
					ok = true
				}
			}
			if !ok {
				missing = req
			}
		}
		if missing != "" {
			skipped = append(skipped, e.Name+": needs "+missing+" (not found on this machine)")
			continue
		}
		cands = append(cands, cand{e, score})
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	var out []ChosenSkill
	used := 0
	take := func(e SkillEntry, why string, mandatory bool) {
		p, b, err := c.verified(e)
		if err != nil {
			skipped = append(skipped, e.Name+": "+err.Error())
			return
		}
		if !mandatory && used+len(b) > MaxSkillBytes {
			skipped = append(skipped, fmt.Sprintf("%s: over the %d-byte skill budget of a brief", e.Name, MaxSkillBytes))
			return
		}
		used += len(b)
		out = append(out, ChosenSkill{Name: e.Name, Path: p, Hash: short(e.SHA256), Why: why})
	}
	for _, e := range c.Skills {
		if e.Mandatory {
			take(e, "mandatory company policy", true)
		}
	}
	optional := 0
	for _, k := range cands {
		if k.e.Mandatory {
			continue
		}
		if optional == MaxSkills {
			skipped = append(skipped, k.e.Name+": over the per-task limit")
			continue
		}
		n := len(out)
		take(k.e, fmt.Sprintf("%d matching tags; %s", k.score, k.e.Evidence), false)
		if len(out) > n {
			optional++
		}
	}
	return out, skipped
}
