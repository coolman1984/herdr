package deck

import (
	"fmt"
	"strings"
	"testing"
)

func TestRowDiffReconstructsEveryChangedScreen(t *testing.T) {
	for _, pair := range [][2]string{{"a\nb\nc", "a\nchanged\nc"}, {"a", "a\nb"}, {"a\nb", "b"}, {"a", ""}, {"", "new"}, {"same", "same"}} {
		patch := rowDiff(pair[0], pair[1], 3, 4)
		lines := strings.Split(pair[0], "\n")
		next := append([]string{}, lines[:patch.Start]...)
		next = append(next, patch.Lines...)
		next = append(next, lines[patch.Start+patch.Delete:]...)
		if strings.Join(next, "\n") != pair[1] || patch.From != 3 || patch.Version != 4 {
			t.Fatalf("patch failed: %+v", patch)
		}
	}
}
func TestScrollingDiffRetainsRowsAndReconstructsScreen(t *testing.T) {
	before := []string{}
	for i := range 2000 {
		before = append(before, fmt.Sprintf("stream row %d", i))
	}
	after := append([]string{}, before[20:]...)
	for i := range 20 {
		after = append(after, fmt.Sprintf("new row %d", i))
	}
	patches := rowChanges(strings.Join(before, "\n"), strings.Join(after, "\n"), 10)
	if len(patches) != 2 {
		t.Fatal("scroll was not compact")
	}
	current := before
	for _, p := range patches {
		next := append([]string{}, current[:p.Start]...)
		next = append(next, p.Lines...)
		next = append(next, current[p.Start+p.Delete:]...)
		current = next
	}
	if strings.Join(current, "\n") != strings.Join(after, "\n") || len(patches[1].Lines) > 20 {
		t.Fatal("scroll patch lost or resent rows")
	}
}
func BenchmarkRowDiff2000Lines(b *testing.B) {
	old := strings.Repeat("unchanged terminal row\n", 1999) + "last"
	next := strings.Repeat("unchanged terminal row\n", 1999) + "new output"
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		rowDiff(old, next, 1, 2)
	}
}
func BenchmarkParseSnapshot(b *testing.B) {
	data := []byte(`{"result":{"snapshot":{"panes":[{"pane_id":"w1:p1","tokens":{"ctx":"40%"}}],"agents":[],"tabs":[],"workspaces":[]}}}`)
	b.ReportAllocs()
	for range b.N {
		if _, err := parseSnapshot(data, "lab"); err != nil {
			b.Fatal(err)
		}
	}
}
