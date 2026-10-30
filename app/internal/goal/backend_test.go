package goal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeHerdr is a shell script standing in for the herdr CLI: it logs its arguments and answers like
// Herdr's JSON. The real CLI was not available to this test (see docs/GOAL-ENGINE.md, doubts).
const fakeHerdr = `#!/bin/sh
echo "$*" >> "$LOG"
case "$1 $2" in
"pane split") echo '{"result":{"pane":{"pane_id":"p-7"}}}' ;;
"pane get") if [ -f "$DIR/closed" ]; then echo 'error: pane_not_found' >&2; exit 1; fi
            echo '{"result":{"pane":{"pane_id":"p-7","agent":"claude","agent_status":"working"}}}' ;;
"pane read") printf '● Working on it\n> \n' ;;
"pane close") touch "$DIR/closed" ;;
esac
`

func TestHerdrBackendCommandsAndPaneMemory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "herdr")
	os.WriteFile(exe, []byte(fakeHerdr), 0o755)
	t.Setenv("LOG", filepath.Join(dir, "log"))
	t.Setenv("DIR", dir)
	h := &HerdrBackend{Exe: exe, LeadPane: "p-1", Dir: dir, Timeout: 10 * time.Second}
	if err := h.Start(StartSpec{Name: "w-T1-3", Cwd: "/work/wt/T1", Argv: []string{"claude", "Read the brief"}, Env: map[string]string{"CLAUDE_CONFIG_DIR": "/acc/2040"}}); err != nil {
		t.Fatal(err)
	}
	// A new backend (a restarted runner) finds the pane from panes.json.
	h2 := &HerdrBackend{Exe: exe, Dir: dir, Timeout: 10 * time.Second}
	o := h2.Observe("w-T1-3")
	if !o.Exists || o.Status != StatusWorking || !strings.Contains(o.Screen, "Working on it") {
		t.Fatalf("observe %+v", o)
	}
	h2.Send("w-T1-3", "continue")
	h2.Keys("w-T1-3", "1")
	h2.Stop("w-T1-3")
	if o := h2.Observe("w-T1-3"); o.Exists || o.Err != nil {
		t.Fatalf("closed pane observed as %+v", o)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "log"))
	want := []string{
		"pane split p-1 --direction right --cwd /work/wt/T1 --no-focus",
		"pane run p-7 CLAUDE_CONFIG_DIR=/acc/2040 claude 'Read the brief'",
		"pane get p-7", "pane read p-7 --source visible",
		"pane send-text p-7 continue", "pane send-keys p-7 Enter", "pane send-keys p-7 1", "pane close p-7",
	}
	for _, w := range want {
		if !strings.Contains(string(b), w+"\n") {
			t.Errorf("herdr was not called as %q; log:\n%s", w, b)
		}
	}
}

func TestProcBackendScreenIsOutputSinceLastInput(t *testing.T) {
	log := "trust?\n1. Yes\nEnter to confirm\n" + inputMark + "\"1\"\ntrusted.\n"
	if s := lastScreen(log); strings.Contains(s, "trust?") || !strings.Contains(s, "trusted.") {
		t.Fatalf("screen %q", s)
	}
}
