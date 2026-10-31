package goal

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// G5/G6 in every path: no transition while the old agent may still run.
func TestStuckStopBlocksEveryReleasePath(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	r.e.B = stuckStop{r.b}
	if err := r.e.apply(Command{Cmd: "task-update", By: "lead", Args: json.RawMessage(`{"id":"T1","spec":"new"}`)}); err == nil {
		t.Fatal("task-update went ahead while the agent did not stop")
	}
	if r.e.G.Task("T1").Spec == "new" || r.e.G.Task("T1").Agent != a.Name {
		t.Fatal("a rejected update changed the task or released it")
	}
	if err := r.e.apply(Command{Cmd: "task-cancel", By: "lead", Args: json.RawMessage(`{"id":"T1"}`)}); err == nil || r.e.G.Task("T1").State == TaskCancelled {
		t.Fatal("cancel went ahead while the agent did not stop")
	}
	r.e.taskAttemptFailed(a, "boom")
	if r.e.G.Task("T1").Attempts != 0 || r.e.G.Task("T1").Agent != a.Name {
		t.Fatal("a failed attempt released a writer that is still alive")
	}
	r.tick()
	if len(r.b.starts) != 1 {
		t.Fatalf("%d starts: a second writer on the same task", len(r.b.starts))
	}
}

func TestResultIsHandledOnlyAfterTheStopIsConfirmed(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	r.finish("T1", Answer{Status: "done"})
	r.e.B = stuckStop{r.b}
	r.tick()
	if r.e.G.Task("T1").State != TaskRunning || a.State != "stop-pending" {
		t.Fatalf("state %s / %s", r.e.G.Task("T1").State, a.State)
	}
	r.e.B = r.b // the process finally goes
	r.b.agents[a.Name].exists = false
	r.tick()
	if st := r.e.G.Task("T1").State; st != TaskReview && st != TaskChecking {
		t.Fatalf("result not handled after the confirmed stop: %s", st)
	}
}

// Audit round 2, finding 1: a cancelled job stays counted and owns the worktree until it has finished.
func TestCancelledJobStaysOwnedUntilJoined(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.Sync = false
	release := make(chan struct{})
	r.e.startJob("merge:T1:0", true, func(ctx context.Context, j *job) { <-release })
	r.e.cancelJobs("")
	if r.e.runningJobs() != 1 {
		t.Fatal("a cancelled job that still runs is not counted")
	}
	if j := r.e.startJob("revert:T9", true, func(ctx context.Context, j *job) {}); j != nil {
		t.Fatal("a second worktree job started while the cancelled one still runs")
	}
	close(release)
	time.Sleep(100 * time.Millisecond)
	r.e.startJob("noop", false, func(ctx context.Context, j *job) {})
	time.Sleep(50 * time.Millisecond)
	if r.e.jobsBusy("merge") != 0 {
		t.Fatal("finished cancelled job still counted")
	}
	_ = strings.TrimSpace
	_ = os.Getenv
}

// Round-2 finding 16: research is a runtime gate before a plan runs and after repeated failures.
func TestResearchGates(t *testing.T) {
	probe := Probes{FreeRAMMB: func() int { return 8000 }, FreeDiskMB: func(string) int { return 1 << 20 }, LookPath: func(s string) (string, error) { return s, nil }}
	r := newRig(t)
	_, _, err := Create(NewGoal{ID: "g2", Text: "x", Project: r.e.G.Project, Accept: []string{"true"}, Profiles: testProfiles(),
		Tasks: []Task{task("T1")}}, probe, time.Now())
	if err == nil || !strings.Contains(err.Error(), "research") {
		t.Fatalf("a plan without research was accepted: %v", err)
	}
	r2 := newRig(t, task("T1"))
	tt := r2.e.G.Task("T1")
	for i := 0; i < 2; i++ {
		r2.e.taskAttemptFailed(&AgentRun{Role: "write", Task: "T1", Profile: "alpha", Ended: true}, "same error")
	}
	tt.State = TaskFailed
	if err := r2.e.apply(Command{Cmd: "task-retry", By: "lead", Args: json.RawMessage(`{"id":"T1"}`)}); err == nil {
		t.Fatal("a twice-failed task was retried without research")
	}
	r2.advance(time.Second)
	if err := r2.e.apply(Command{Cmd: "research", By: "lead", Args: json.RawMessage(`{"task":"T1","source":"https://docs.python.org/3/library/csv.html","version":"3.13","why":"newline handling"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := r2.e.apply(Command{Cmd: "task-retry", By: "lead", Args: json.RawMessage(`{"id":"T1"}`)}); err != nil {
		t.Fatalf("retry after research refused: %v", err)
	}
}

// Round-2 finding 3: protected acceptance files cannot be changed by a task; the reviewer works in a
// detached copy of exactly the reviewed commit.
func TestProtectedFilesAndReviewerSnapshot(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Protected = []string{"tests/acceptance"}
	r.tick()
	a := r.agentOf("T1")
	os.MkdirAll(a.Cwd+"/tests/acceptance", 0o755)
	os.WriteFile(a.Cwd+"/tests/acceptance/check.sh", []byte("exit 0\n"), 0o644)
	r.finish("T1", Answer{Status: "done"})
	r.tick()
	r.tick()
	rev := r.agentOf("T1")
	if rev == nil || rev.Role != "review" || rev.Cwd == a.Cwd || !strings.Contains(rev.Cwd, "review-T1") {
		t.Fatalf("reviewer not in a snapshot: %+v", rev)
	}
	head, _ := git(rev.Cwd, "rev-parse", "HEAD")
	if head != r.e.G.Task("T1").ReviewHead {
		t.Fatal("snapshot is not the reviewed commit")
	}
	r.finish("T1", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	if r.e.G.Task("T1").State == TaskDone || len(r.events("protected-change")) != 1 {
		t.Fatalf("a change to protected tests was merged: %s", r.e.G.Task("T1").State)
	}
}
