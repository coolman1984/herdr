package deck

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadinessWarnsWithoutInventingInstructions(t *testing.T) {
	result := readiness("Do something useful")
	if result["ready"] != false || len(result["warnings"].([]string)) != 4 {
		t.Fatalf("missing warning: %+v", result)
	}
	complete := "Allowed files: app/**. Answer format: DONE. Definition of done: tests. Stop when blocked."
	if readiness(complete)["ready"] != true {
		t.Fatal("complete task rejected")
	}
}
func TestOwnershipOverlapsAreCaseNormalizedAndCannotEscape(t *testing.T) {
	a, err := normalizeOwnership(`I:\herdr`, `app/**`)
	if err != nil {
		t.Fatal(err)
	}
	b, err := normalizeOwnership(`I:\herdr`, `APP\main.go`)
	if err != nil || !overlaps(a, b) {
		t.Fatal("overlap missed")
	}
	if _, err = normalizeOwnership(`I:\herdr`, `..\outside`); err == nil {
		t.Fatal("escaped repo")
	}
	if overlaps(a, `i:\herdr\application\main.go`) {
		t.Fatal("prefix false positive")
	}
}
func TestHelperAndResearchGuardsHoldUnknownOrUnsafeLaunches(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.profiles["claude"] = Profile{Kind: "claude"}
	for _, p := range []paneRequest{{Profile: "claude", Helper: true}, {Profile: "claude", Helper: true, Account: "2030", LeadAccount: "2030"}, {Profile: "claude", Purpose: "research", Route: "browser"}} {
		if err := a.launchGuard(p); err == nil {
			t.Fatalf("unsafe launch admitted: %+v", p)
		}
	}
}
func TestProofAndCitationPresenceNeverBecomesVerification(t *testing.T) {
	result := auditProof("Checks run: claimed. https://openai.com/test https://example.org/post", []string{"openai.com"})
	proof := result["proof"].(map[string]string)
	if proof["checks run"] != "mentioned; not verified" || proof["mutation"] != "not verified" {
		t.Fatal("unproven claim marked verified")
	}
	urls := result["citations"].([]map[string]any)
	if len(urls) != 2 || urls[0]["official_domain"] != true || urls[1]["official_domain"] != false {
		t.Fatalf("bad citation audit: %+v", urls)
	}
}

type failingTransport struct{ fakeHerdr }

func (f *failingTransport) Call(ctx context.Context, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("timed out after possible delivery")
}
func TestUncertainDeliveryIsRecordedWithoutReplay(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	a := newApp(&failingTransport{}, "lab", "test-token", "http://127.0.0.1:9876", "", map[string]Profile{})
	_, dispatch, err := a.deliver(context.Background(), "lab|w1:p1", "w1:p1", "task", "agent", "prompt", "coder")
	if err == nil || !strings.Contains(err.Error(), "delivery unknown") {
		t.Fatal("uncertain delivery hidden")
	}
	if dispatch == "" {
		t.Fatal("uncertain delivery has no dispatch id")
	}
	if len(a.guards.Deliveries) != 1 || a.guards.Deliveries[0].State != "unknown" || a.guards.Deliveries[0].Key != "lab|w1:p1" {
		t.Fatalf("missing ledger: %+v", a.guards.Deliveries)
	}
}
func TestScriptBridgePassesJSONAsDataAndSwitchesAsBooleans(t *testing.T) {
	root := t.TempDir()
	script := `param([string]$Task,[switch]$Fresh,[string[]]$Agents) @{task=$Task;fresh=[bool]$Fresh;agents=$Agents}|ConvertTo-Json -Compress; exit 0`
	os.WriteFile(filepath.Join(root, "bridge-test.ps1"), []byte(script), 0600)
	cli := &CLI{Root: root}
	task := "\"quoted\"\n$(never execute) ; I:\\work\\"
	out, err := cli.RunScript(context.Background(), "bridge-test.ps1", map[string]any{"Task": task, "Fresh": true, "Agents": []string{"one", "two"}}, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Task   string
		Fresh  bool
		Agents []string
	}
	if err = json.Unmarshal([]byte(out), &result); err != nil || result.Task != task || !result.Fresh || len(result.Agents) != 2 {
		t.Fatalf("bridge changed data: %q %v", out, err)
	}
}
