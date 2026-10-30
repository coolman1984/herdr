package goal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Audit #3: the consumer trusts the token, never the "by" text a sender writes.
func TestInboxRoleComesFromTokenNotClaim(t *testing.T) {
	r := newRig(t, task("T1"))
	write := func(c Command) {
		b, _ := json.Marshal(c)
		f, _ := os.OpenFile(filepath.Join(r.e.L.Dir, "inbox.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		f.Write(append(b, '\n'))
		f.Close()
	}
	write(Command{ID: "c1", Cmd: "stop", By: "owner"}) // forged claim, no token
	r.tick()
	if r.e.G.State == GoalStopped {
		t.Fatal("a forged owner claim stopped the goal")
	}
	owner, _ := os.ReadFile(filepath.Join(r.e.L.Dir, "keys", "owner"))
	write(Command{ID: "c2", Cmd: "pause", By: "write", Token: string(owner)})
	r.tick()
	if r.e.G.State != GoalPaused {
		t.Fatalf("owner token not honoured: %s", r.e.G.State)
	}
	if st, _ := os.Stat(filepath.Join(r.e.L.Dir, "keys", "owner")); runtime.GOOS != "windows" && st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("owner key readable by others: %v", st.Mode())
	}
}

func TestLeadTokenProvesLeadOnly(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.e.G.Limits.MaxAgents = 3
	r.tick()
	var tok string
	for _, f := range r.b.agents {
		if v := f.spec.Env["HERDR_GOAL_TOKEN"]; v != "" {
			tok = v
		}
		if f.spec.Env["HERDR_GOAL_ROLE"] != "lead" && f.spec.Env["HERDR_GOAL_TOKEN"] != "" {
			t.Fatal("a delegate got a command token")
		}
	}
	if tok == "" {
		t.Fatal("lead has no token")
	}
	for _, c := range []Command{{ID: "a", Cmd: "stop", Token: tok}, {ID: "b", Cmd: "decide", Token: tok, Args: json.RawMessage(`{"text":"use sqlite"}`)}} {
		b, _ := json.Marshal(c)
		os.WriteFile(filepath.Join(r.e.L.Dir, "inbox.jsonl"), append(b, '\n'), 0o644)
		r.e.G.InboxOff = 0
		r.tick()
	}
	if r.e.G.State == GoalStopped || len(r.events("decide")) != 1 {
		t.Fatalf("lead token: state %s, decisions %d", r.e.G.State, len(r.events("decide")))
	}
}

func TestDependencyCycleRejectedWithoutEffect(t *testing.T) {
	r := newRig(t, task("T1"), task("T2", "T1"))
	err := r.e.apply(Command{Cmd: "task-update", By: "lead", Args: json.RawMessage(`{"id":"T1","deps":["T2"]}`)})
	if err == nil || !strings.Contains(err.Error(), "cycle") || len(r.e.G.Task("T1").Deps) != 0 {
		t.Fatalf("cycle: %v %v", err, r.e.G.Task("T1").Deps)
	}
}

func TestCancelledPrerequisiteWaitsForReplanThenRuns(t *testing.T) {
	r := newRig(t, task("T1"), task("T2", "T1"))
	r.e.apply(Command{Cmd: "task-cancel", By: "lead", Args: json.RawMessage(`{"id":"T1"}`)})
	r.tick()
	if r.e.G.Task("T2").State != TaskPending || len(r.events("dependency-broken")) != 1 {
		t.Fatal("dependent of a cancelled task ran or was not reported")
	}
	r.e.apply(Command{Cmd: "task-update", By: "lead", Args: json.RawMessage(`{"id":"T2","deps":[]}`)})
	r.tick()
	if r.e.G.Task("T2").State != TaskRunning {
		t.Fatalf("re-planned task did not start: %s", r.e.G.Task("T2").State)
	}
}

func TestPathReservationsNormalised(t *testing.T) {
	for _, c := range [][2]string{{"src", "src/a.go"}, {"./src/", "src/a.go"}, {"src/a.go", "src//a.go"}, {"src\\a.go", "src/a.go"}} {
		if !pathsOverlap(c[0], c[1]) {
			t.Errorf("%q and %q should overlap", c[0], c[1])
		}
	}
	if pathsOverlap("src", "srcx/a.go") {
		t.Error("src and srcx are different folders")
	}
}

// Audit #6: the merged commit is the reviewed one; a push after review starts is reviewed again.
func TestReviewBoundToTheReviewedCommit(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	r.finish("T1", Answer{Status: "done"})
	r.tick()
	r.tick()
	rev := r.agentOf("T1")
	wt := r.e.G.Task("T1").Worktree
	os.WriteFile(filepath.Join(wt, "late.txt"), []byte("unreviewed\n"), 0o644)
	run(t, wt, "git", "add", "-A")
	run(t, wt, "git", "commit", "-qm", "late change")
	b, _ := json.Marshal(Answer{Msg: rev.Msg, Status: "done", Verdict: "pass"})
	os.WriteFile(rev.Answer, b, 0o644)
	r.tick()
	if r.e.G.Task("T1").State == TaskDone || r.e.G.Task("T1").State == TaskMerging {
		t.Fatal("work changed after review began was merged on the old verdict")
	}
	if len(r.events("review-stale")) != 1 {
		t.Fatal("stale review not reported")
	}
}

var _ = time.Second
