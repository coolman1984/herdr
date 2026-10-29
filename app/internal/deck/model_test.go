package deck

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSnapshotUsesRealEnvelopeAndMetadata(t *testing.T) {
	raw := []byte(`{"result":{"snapshot":{"workspaces":[{"workspace_id":"w7","label":"Work"}],"tabs":[],"panes":[{"pane_id":"w7:p8","tab_id":"w7:t1","workspace_id":"w7","agent_status":"blocked","tokens":{"model":"sol","ctx":"42%","out":"120"},"cwd":"I:\\work"}],"agents":[{"pane_id":"w7:p8","name":"reviewer","agent_status":"blocked"}]}}}`)
	s, err := parseSnapshot(raw, "lab")
	if err != nil {
		t.Fatal(err)
	}
	p := s.Panes[0]
	if p.ID != "w7:p8" || p.Key != "lab|w7:p8" || p.Status != "blocked" || p.Name != "reviewer" || p.Tokens["ctx"] != "42%" || p.Cwd != `I:\work` {
		t.Fatalf("lost real metadata: %+v", p)
	}
}
func TestSnapshotRejectsInvalidSuccess(t *testing.T) {
	for _, raw := range []string{`{}`, `{"result":{}}`, `{"result":{"snapshot":{}}}`, `{"error":{"code":"server_not_running","message":"offline"}}`, `broken`} {
		if _, err := parseSnapshot([]byte(raw), "default"); err == nil {
			t.Fatalf("accepted invalid response: %s", raw)
		}
	}
}
func TestScreenReadAndCap(t *testing.T) {
	raw := []byte(`{"result":{"output":"hello\nworld\n"}}`)
	got, err := parseScreen(raw, 40)
	if err != nil || got != "hello\nworld" {
		t.Fatalf("%q %v", got, err)
	}
	text := strings.Repeat("row\n", 4000)
	data, _ := json.Marshal(map[string]any{"result": map[string]any{"output": text}})
	got, err = parseScreen(data, 40)
	if err != nil || len(strings.Split(got, "\n")) != 2000 {
		t.Fatalf("cap: %d %v", len(strings.Split(got, "\n")), err)
	}
	if _, err = parseScreen([]byte(`{"result":{}}`), 40); err == nil {
		t.Fatal("missing output accepted")
	}
}
func TestDialogGuardNeedsFooter(t *testing.T) {
	if blockedScreen("An answer mentions Trust this folder?\nAll finished.") {
		t.Fatal("answer falsely blocked")
	}
	if !blockedScreen("Trust this folder?\nEnter to confirm · Esc to cancel") {
		t.Fatal("trust dialog missed")
	}
	if !blockedScreen("Please sign in") {
		t.Fatal("sign-in missed")
	}
}
