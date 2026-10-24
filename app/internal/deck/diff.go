package deck

import "strings"

type RowPatch struct {
	From    uint64   `json:"from_version"`
	Version uint64   `json:"version"`
	Start   int      `json:"start"`
	Delete  int      `json:"delete"`
	Lines   []string `json:"lines"`
}

func rowDiff(old, new string, from, to uint64) RowPatch {
	before := strings.Split(old, "\n")
	after := strings.Split(new, "\n")
	start := 0
	for start < len(before) && start < len(after) && before[start] == after[start] {
		start++
	}
	endOld, endNew := len(before), len(after)
	for endOld > start && endNew > start && before[endOld-1] == after[endNew-1] {
		endOld--
		endNew--
	}
	return RowPatch{From: from, Version: to, Start: start, Delete: endOld - start, Lines: after[start:endNew]}
}

// A scrolling viewport needs a top drop and a tail update. Two ordinary patches
// preserve the existing browser contract and avoid resending 2000 retained rows.
func rowChanges(old, new string, from uint64) []RowPatch {
	before := strings.Split(old, "\n")
	after := strings.Split(new, "\n")
	if len(before) > 64 && len(after) > 64 {
		maximum := len(before) - 1
		if maximum > 256 {
			maximum = 256
		}
		for shift := 1; shift <= maximum; shift++ {
			if before[shift] != after[0] {
				continue
			}
			same := 0
			for same+shift < len(before) && same < len(after) && before[same+shift] == after[same] {
				same++
			}
			if same > len(before)/2 {
				return []RowPatch{{From: from, Version: from + 1, Start: 0, Delete: shift, Lines: []string{}}, {From: from + 1, Version: from + 2, Start: same, Delete: len(before) - shift - same, Lines: after[same:]}}
			}
		}
	}
	return []RowPatch{rowDiff(old, new, from, from+1)}
}

type PaneUpdate struct {
	Pane
	Screen *string   `json:"screen,omitempty"`
	Patch  *RowPatch `json:"screen_patch,omitempty"`
}
