package goal

import (
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
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	State    string   `json:"state"`
	License  string   `json:"license"`
	Tags     []string `json:"tags"`
	Requires []string `json:"requires"`
	Profiles []string `json:"profiles"`
	Evidence string   `json:"evidence"`
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
	return &SkillCatalog{Root: filepath.Dir(filepath.Dir(path)), Skills: c.Skills}, nil
}

// ChosenSkill is what a brief gets.
type ChosenSkill struct {
	Name, Path, Hash, Why string
}

// Select picks skills for a task. tools: capability facts (name -> status); profile: device profile.
// Returns the chosen ones and why each other skill was left out (for the event log).
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
	for _, k := range cands {
		if len(out) == MaxSkills {
			skipped = append(skipped, k.e.Name+": over the per-task limit")
			continue
		}
		p := filepath.Join(c.Root, filepath.FromSlash(k.e.Path))
		b, err := os.ReadFile(p)
		if err != nil {
			skipped = append(skipped, k.e.Name+": file missing")
			continue
		}
		h := sha256.Sum256(b)
		out = append(out, ChosenSkill{Name: k.e.Name, Path: p, Hash: hex.EncodeToString(h[:6]),
			Why: fmt.Sprintf("%d matching tags; %s", k.score, k.e.Evidence)})
	}
	return out, skipped
}
