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
	return rowDiffRows(before, after, from, to)
}

func rowDiffRows(before, after []string, from, to uint64) RowPatch {
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
	return rowChangesRows(before, after, from)
}

func rowChangesRows(before, after []string, from uint64) []RowPatch {
	patch := rowDiffRows(before, after, from, from+1)
	// Ordinary edits, particularly repeated-row tails, must never look like scrolls.
	if patch.Start == 0 && patch.Delete > len(before)/2 && len(before) > 64 && len(after) > 64 {
		maximum := len(before) - 1
		if maximum > 256 {
			maximum = 256
		}
		budget := 4 * (len(before) + len(after))
		for shift := 1; shift <= maximum && budget > 0; shift++ {
			if before[shift] != after[0] {
				continue
			}
			same := 0
			for same+shift < len(before) && same < len(after) && budget > 0 && before[same+shift] == after[same] {
				budget--
				same++
			}
			if same > len(before)/2 && len(after)-same+2 < len(patch.Lines) {
				return []RowPatch{{From: from, Version: from + 1, Start: 0, Delete: shift, Lines: []string{}}, {From: from + 1, Version: from + 2, Start: same, Delete: len(before) - shift - same, Lines: after[same:]}}
			}
		}
	}
	return []RowPatch{patch}
}

type PaneUpdate struct {
	Pane
	Screen *string   `json:"screen,omitempty"`
	Patch  *RowPatch `json:"screen_patch,omitempty"`
}

// Validate coordinates against the evolving retained row count before publishing.
func validPatches(patches []RowPatch, count, want int, version uint64) bool {
	if len(patches) == 0 {
		return false
	}
	for _, p := range patches {
		if p.From != version || p.Version != version+1 || p.Start < 0 || p.Delete < 0 || p.Start > count || p.Delete > count-p.Start {
			return false
		}
		count += len(p.Lines) - p.Delete
		version = p.Version
		if count > 2000 {
			return false
		}
	}
	return count == want
}
