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
