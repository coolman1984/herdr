package goal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func signed(c Command, secret string) Command {
	if c.At.IsZero() {
		c.At = T(time.Now())
	}
	c.Key, c.Mac = keyID(secret), commandMAC(secret, c)
	return c
}

// Audit #3 (+ round 2 finding 2): the consumer trusts a signature made with a key it holds, never the
// "by" text; the key itself is never written to the inbox.
func TestInboxRoleComesFromSignatureNotClaim(t *testing.T) {
	r := newRig(t, task("T1"))
	write := func(c Command) {
		b, _ := json.Marshal(c)
		f, _ := os.OpenFile(filepath.Join(r.e.L.Dir, "inbox.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		f.Write(append(b, '\n'))
		f.Close()
	}
	write(Command{ID: "c1", Cmd: "stop", By: "owner"}) // forged claim, no signature
	r.tick()
	if r.e.G.State == GoalStopped {
		t.Fatal("a forged owner claim stopped the goal")
	}
	ownerB, _ := os.ReadFile(filepath.Join(r.e.L.Dir, "keys", "owner"))
	owner := string(ownerB)
	bad := signed(Command{ID: "c2", Cmd: "stop"}, owner)
	bad.Cmd = "pause" // signature over another command
	write(bad)
	r.tick()
	if r.e.G.State == GoalPaused {
		t.Fatal("a command whose signature does not match was applied")
	}
	good := signed(Command{ID: "c3", Cmd: "pause", By: "write"}, owner)
	write(good)
	r.tick()
	if r.e.G.State != GoalPaused {
		t.Fatalf("owner signature not honoured: %s", r.e.G.State)
	}
	r.e.apply(Command{Cmd: "resume", By: "owner"})
	write(good) // replay of the same signed line
	r.tick()
	if r.e.G.State == GoalPaused {
		t.Fatal("a replayed command was applied twice")
	}
	if st, _ := os.Stat(filepath.Join(r.e.L.Dir, "keys", "owner")); runtime.GOOS != "windows" && st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("owner key readable by others: %v", st.Mode())
	}
}

func TestEnqueueNeverWritesTheKey(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.tick()
	if _, err := Enqueue(r.e.L.Dir, "decide", "owner", map[string]string{"text": "x"}); err != nil {
		t.Fatal(err)
	}
	r.tick()
	owner, _ := os.ReadFile(filepath.Join(r.e.L.Dir, "keys", "owner"))
	secrets := []string{string(owner)}
	for _, f := range r.b.agents {
		if p := f.spec.Env["HERDR_GOAL_KEY_FILE"]; p != "" {
			b, _ := os.ReadFile(p)
			secrets = append(secrets, string(b))
		}
	}
	if len(secrets) != 2 || len(r.events("decide")) != 1 {
		t.Fatalf("setup: %d secrets, %d decisions", len(secrets), len(r.events("decide")))
	}
	_ = filepath.WalkDir(r.e.L.Dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.Contains(p, string(filepath.Separator)+"keys"+string(filepath.Separator)) {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, s := range secrets {
			if strings.Contains(string(b), s) {
				t.Errorf("a key leaked into %s", p)
			}
		}
		return nil
	})
	if st, _ := os.Stat(filepath.Join(r.e.L.Dir, "inbox.jsonl")); runtime.GOOS != "windows" && st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("inbox readable by others: %v", st.Mode())
	}
}

func TestLeadKeyProvesLeadOnly(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.e.G.Limits.MaxAgents = 3
	r.tick()
	var key string
	for _, f := range r.b.agents {
		if p := f.spec.Env["HERDR_GOAL_KEY_FILE"]; p != "" {
			b, _ := os.ReadFile(p)
			key = string(b)
		}
		if f.spec.Env["HERDR_GOAL_ROLE"] != "lead" && f.spec.Env["HERDR_GOAL_KEY_FILE"] != "" {
			t.Fatal("a delegate got a command key")
		}
	}
	if key == "" {
		t.Fatal("lead has no key")
	}
	for _, c := range []Command{{ID: "a", Cmd: "stop"}, {ID: "b", Cmd: "decide", Args: json.RawMessage(`{"text":"use sqlite"}`)}} {
		b, _ := json.Marshal(signed(c, key))
		os.WriteFile(filepath.Join(r.e.L.Dir, "inbox.jsonl"), append(b, '\n'), 0o600)
		r.e.G.InboxOff = 0
		r.tick()
	}
	if r.e.G.State == GoalStopped || len(r.events("decide")) != 1 {
		t.Fatalf("lead key: state %s, decisions %d", r.e.G.State, len(r.events("decide")))
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

// Audit #1: one runner by OS lock, a newer generation fences an old writer.
func TestRunnerLockIsExclusiveAndReleased(t *testing.T) {
	dir := t.TempDir()
	_, release, err := takeLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !runnerAlive(dir) {
		t.Fatal("held lock not seen")
	}
	if _, _, err := takeLock(dir); err == nil {
		t.Fatal("two runners took the same goal")
	}
	release()
	if runnerAlive(dir) {
		t.Fatal("released lock still seen as a live runner")
	}
	_, release2, err := takeLock(dir)
	if err != nil {
		t.Fatalf("lock not reusable after release: %v", err)
	}
	release2()
}

func TestOldRunnerGenerationCannotWrite(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.Gen = 1
	r.e.G.RunnerGen = 1
	r.tick()
	newer, _ := r.e.L.Load()
	newer.RunnerGen = 2
	r.e.L.Save(newer)
	r.e.G.Tasks[0].Title = "written by the old runner"
	if err := r.e.Tick(); err != ErrSuperseded {
		t.Fatalf("old runner kept writing: %v", err)
	}
	if g, _ := r.e.L.Load(); g.Tasks[0].Title == "written by the old runner" {
		t.Fatal("old runner's state reached disk")
	}
}

func pidAlive(t *testing.T, pidFile string) bool {
	t.Helper()
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("no pid file: %v", err)
	}
	var pid int
	fmt.Sscan(strings.TrimSpace(string(b)), &pid)
	return processAlive(pid)
}

// Audit #5/#10: cancel and timeout kill the whole process tree, not only the shell.
func TestCancelAndTimeoutKillTheProcessTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh fixture")
	}
	dir := t.TempDir()
	cmd := "sleep 60 & echo $! > child.pid; wait"
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(700 * time.Millisecond); cancel() }()
	start := time.Now()
	ok, out := RunChecksCtx(ctx, dir, []string{cmd}, time.Minute)
	if ok || time.Since(start) > 15*time.Second || !strings.Contains(out, "CANCELLED") {
		t.Fatalf("cancel: ok=%v after %s: %s", ok, time.Since(start), out)
	}
	time.Sleep(200 * time.Millisecond)
	if pidAlive(t, filepath.Join(dir, "child.pid")) {
		t.Fatal("cancelled check left its child running")
	}
	os.Remove(filepath.Join(dir, "child.pid"))
	ok, out = RunChecksCtx(context.Background(), dir, []string{cmd}, 700*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	if ok || !strings.Contains(out, "TIMEOUT") || pidAlive(t, filepath.Join(dir, "child.pid")) {
		t.Fatalf("timeout did not kill the tree: %s", out)
	}
}

func TestCheckOutputIsBounded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh fixture")
	}
	out, _, _ := runBounded(context.Background(), t.TempDir(), time.Minute, 64<<10, "sh", "-c", "head -c 20000000 /dev/zero | tr '\\0' x")
	if len(out) > 70<<10 || !strings.Contains(out, "earlier bytes dropped") {
		t.Fatalf("output not bounded: %d bytes", len(out))
	}
}

// Audit #5: a check result for an older task generation is never applied.
func TestStaleCheckResultIsDropped(t *testing.T) {
	r := newRig(t, Task{ID: "T1", Title: "t", Spec: "s", Accept: []string{"sleep 2"}})
	r.e.Sync = false
	r.tick()
	r.finish("T1", Answer{Status: "done"})
	r.tick() // checks start in the background
	if r.e.G.Task("T1").State != TaskChecking {
		t.Fatalf("state %s", r.e.G.Task("T1").State)
	}
	if err := r.e.apply(Command{Cmd: "task-update", By: "lead", Args: json.RawMessage(`{"id":"T1","spec":"new spec"}`)}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	r.tick()
	if len(r.events("task-review")) != 0 || r.e.G.Task("T1").State == TaskReview {
		t.Fatal("a check of the old spec moved the changed task to review")
	}
}

// Audit #4: one job at a time owns the integration worktree.
func TestIntegrationWorktreeHasOneOwner(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.Sync = false
	release := make(chan struct{})
	if j := r.e.startJob("merge:A:0", true, func(ctx context.Context, j *job) { <-release }); j != nil {
		t.Fatal("job finished early")
	}
	var ran atomic.Bool
	j := r.e.startJob("revert:B", true, func(ctx context.Context, j *job) { ran.Store(true) })
	time.Sleep(200 * time.Millisecond)
	if j != nil || ran.Load() {
		t.Fatal("a second job touched the integration worktree while a merge held it")
	}
	close(release)
	time.Sleep(100 * time.Millisecond)
	r.e.startJob("merge:A:0", true, nil)
	time.Sleep(50 * time.Millisecond)
	for i := 0; i < 20 && !ran.Load(); i++ {
		r.e.startJob("revert:B", true, func(ctx context.Context, j *job) { ran.Store(true) })
		time.Sleep(20 * time.Millisecond)
	}
	if !ran.Load() {
		t.Fatal("worktree never freed")
	}
}

// Audit #10: offline dependencies in the integration worktree survive a merge; only files git names as
// blocking are removed.
func TestMergeKeepsUntrackedDependencies(t *testing.T) {
	r := newRig(t, task("T1"))
	main := r.e.W.MainDir()
	os.MkdirAll(filepath.Join(main, "node_modules", "lib"), 0o755)
	os.WriteFile(filepath.Join(main, "node_modules", "lib", "index.js"), []byte("offline\n"), 0o644)
	os.WriteFile(filepath.Join(main, "T1.txt"), []byte("left by a test run\n"), 0o644) // T1 adds a tracked T1.txt
	r.tick()
	r.finish("T1", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.finish("T1", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	if r.e.G.Task("T1").State != TaskDone {
		t.Fatalf("merge blocked by an untracked file: %s %v", r.e.G.Task("T1").State, r.events("merge-failed"))
	}
	if _, err := os.Stat(filepath.Join(main, "node_modules", "lib", "index.js")); err != nil {
		t.Fatal("merge deleted untracked offline dependencies")
	}
}

// Soak round 5: a restart loop is activity, not progress; the no-progress limit still fires.
func TestRestartLoopIsNotProgress(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	for i := 0; i < 50 && r.e.G.State == GoalRunning; i++ {
		r.advance(time.Minute)
		if a := r.agentOf("T1"); a != nil {
			a.Begun = true
			r.b.agents[a.Name].exists = false // dies every minute after working a while
		}
		r.tick()
	}
	if r.e.G.State != GoalStopped {
		t.Fatalf("restart loop kept the goal alive: %s", r.e.G.State)
	}
}

// Soak round 5: a replacement lead is told about the open problems again.
func TestNewLeadIsToldAboutOpenProblems(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.e.G.Limits.MaxAgents = 3
	r.e.G.Task("T1").State = TaskFailed
	nudged := func() int {
		n := 0
		for _, a := range r.e.G.Agents {
			if a.Role == "lead" {
				for _, s := range r.b.agents[a.Name].sent {
					if strings.Contains(s, "Next step needed") {
						n++
					}
				}
			}
		}
		return n
	}
	idle := func() {
		for _, a := range r.e.G.Agents {
			if a.Role == "lead" && !a.Ended {
				a.Begun = true
				f := r.b.agents[a.Name]
				f.status, f.screen = StatusIdle, "> "+a.Name
			}
		}
	}
	r.tick()
	idle()
	r.tick()
	if nudged() != 1 {
		t.Fatalf("first lead nudges %d", nudged())
	}
	for _, a := range r.e.G.Agents {
		if a.Role == "lead" && !a.Ended {
			r.b.agents[a.Name].exists = false
		}
	}
	r.tick()
	idle()
	r.tick()
	if nudged() != 2 {
		t.Fatalf("replacement lead was never told about the failed task (nudges %d)", nudged())
	}
}

func TestAutosaveSkipsCachesSecretsAndHugeFiles(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	wt := r.agentOf("T1").Cwd
	files := map[string]int{"src/app.py": 10, "__pycache__/app.cpython-313.pyc": 10, ".env": 10, "keys/server.pem": 10, "node_modules/x/i.js": 10, "big.bin": 6 << 20}
	for f, n := range files {
		os.MkdirAll(filepath.Dir(filepath.Join(wt, f)), 0o755)
		os.WriteFile(filepath.Join(wt, f), []byte(strings.Repeat("x", n)), 0o644)
	}
	if ok, err := r.e.W.SaveWork(r.e.G.Task("T1"), "w"); !ok || err != nil {
		t.Fatal(ok, err)
	}
	got, _ := git(wt, "show", "--name-only", "--format=", "HEAD")
	if got != "src/app.py" {
		t.Fatalf("autosave committed %q, want only src/app.py", got)
	}
}

func TestFailureBudgetSurvivesReplans(t *testing.T) {
	r := newRig(t, task("T1"))
	for i := 0; i < 20; i++ {
		tt := r.e.G.Task("T1")
		if strings.HasPrefix(tt.Note, "BUDGET:") {
			break
		}
		if tt.State == TaskFailed {
			r.advance(time.Second)
			r.e.apply(Command{Cmd: "research", By: "lead", Args: json.RawMessage(`{"task":"T1","source":"offline","why":"retry after failure"}`)})
			if err := r.e.apply(Command{Cmd: "task-retry", By: "lead", Args: json.RawMessage(`{"id":"T1"}`)}); err != nil {
				t.Fatal(err)
			}
		}
		r.e.taskAttemptFailed(&AgentRun{Role: "write", Task: "T1", Profile: "alpha", Ended: true}, "boom")
	}
	if !strings.HasPrefix(r.e.G.Task("T1").Note, "BUDGET:") {
		t.Fatal("lead retries reset the failure budget for ever")
	}
	if err := r.e.apply(Command{Cmd: "task-retry", By: "lead", Args: json.RawMessage(`{"id":"T1"}`)}); err == nil {
		t.Fatal("lead retried past the budget")
	}
}

// Audit #7: trust is decided on the folder the prompt shows, not on where we started the agent.
func TestTrustUsesTheFolderOnScreen(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	f := r.screen("T1", StatusBlocked, strings.Replace(claudeTrust, "/work/project", t.TempDir(), 1))
	r.tick()
	if len(f.keys) != 0 || r.e.G.Task("T1").State != TaskBlocked {
		t.Fatalf("trusted a folder outside the project because the agent started inside it: %v", f.keys)
	}
}

// Audit #8: no blind default on a menu with a risky option; the decider does not block the watchdog.
func TestRiskyMenuIsNotAnsweredByDefault(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	f := r.screen("T1", StatusBlocked, " Apply the change?\n ❯ 1. Push to origin now\n   2. Keep local\n Enter to confirm · Esc to cancel\n")
	r.tick()
	if len(f.keys) != 0 || len(r.events("escalated")) == 0 {
		t.Fatalf("risky menu answered by default: %v", f.keys)
	}
}

func TestDeciderRunsInTheBackground(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.Sync = false
	release := make(chan struct{})
	r.e.Decide = func(g *Goal, a *AgentRun, screen string, opts []Option) ([]string, string, error) {
		<-release
		return []string{"2"}, "json", nil
	}
	r.tick()
	f := r.screen("T1", StatusBlocked, " Which format?\n ❯ 1. Plain\n   2. JSON\n Enter to confirm · Esc to cancel\n")
	done := make(chan bool)
	go func() { r.tick(); done <- true }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the tick waited for the decider")
	}
	close(release)
	time.Sleep(100 * time.Millisecond)
	r.tick()
	if len(f.keys) != 1 || f.keys[0][0] != "2" {
		t.Fatalf("decider pick not sent after it finished: %v", f.keys)
	}
}

// Audit #9: an unconfirmed stop keeps the agent owned; nothing restarts on top of it.
type stuckStop struct{ *fakeBackend }

func (s stuckStop) Stop(n string) error { return nil } // reports success, process lives on

func TestUnconfirmedStopKeepsOwnership(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	r.e.B = stuckStop{r.b}
	r.e.restartOrFail(a, "test")
	if a.Ended || a.State != "stop-pending" || len(r.b.starts) != 1 {
		t.Fatalf("stop not confirmed but agent released: ended=%v state=%s starts=%d", a.Ended, a.State, len(r.b.starts))
	}
}

// Audit #9: an account limit on one agent does not stop healthy agents on other accounts.
func TestAccountLimitDoesNotStopHealthyAgents(t *testing.T) {
	r := newRig(t, task("T1"), task("T2"))
	r.e.G.Profiles[0].Accounts = []string{"a1"}
	r.e.G.Tasks[1].Writer = "beta"
	r.tick()
	healthy := r.agentOf("T2")
	r.e.G.Accounts[r.e.realAccount("beta", "b1")] = &Account{LimitedUntil: T(r.now.Add(time.Hour))} // no free account left anywhere
	r.screen("T1", StatusIdle, "You've hit your limit · resets in 1h 0m\n")
	r.tick()
	if healthy.Ended || !r.b.agents[healthy.Name].exists {
		t.Fatal("a healthy agent was stopped because another account hit its limit")
	}
	if r.e.G.State == GoalWaitingLimit {
		t.Fatal("goal waits while an agent still works")
	}
}

// Audit #15: a reused event number keeps the later line; a damaged line in the middle is skipped.
func TestEventsDeduplicateAndSkipDamagedLines(t *testing.T) {
	l := Ledger{Dir: t.TempDir()}
	os.WriteFile(filepath.Join(l.Dir, "events.jsonl"), []byte(`{"seq":1,"kind":"a"}
{"seq":2,"kind":"b-before-crash"}
{"seq":2,"kind":"to
{"seq":2,"kind":"b-after-restart"}
{"seq":3,"kind":"c"}
`), 0o644)
	evs, err := l.Events()
	if err != nil || len(evs) != 3 || evs[1].Kind != "b-after-restart" {
		t.Fatalf("%v %+v", err, evs)
	}
}

// Audit #16: a damaged pane map is an error, not "no panes"; newlines never reach a pane.
func TestHerdrBackendRefusesDamagedPaneMapAndNewlines(t *testing.T) {
	dir := t.TempDir()
	h := &HerdrBackend{Exe: "/nonexistent/herdr", LeadPane: "p1", Dir: dir}
	if err := h.Start(StartSpec{Name: "a", Argv: []string{"claude", "line1\nrm -rf ~"}}); err == nil || !strings.Contains(err.Error(), "newline") {
		t.Fatalf("newline argument accepted: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "panes.json"), []byte(`{"a": "p-`), 0o644)
	if err := h.Start(StartSpec{Name: "b", Argv: []string{"claude"}}); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Fatalf("damaged panes.json read as empty: %v", err)
	}
}

// Audit #17: the process budget counts the lead and background jobs.
func TestProcessBudgetCountsLeadAndJobs(t *testing.T) {
	r := newRig(t, task("T1"), task("T2"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.e.G.Limits.MaxAgents, r.e.G.Limits.MaxProcs = 5, 3
	r.e.Sync = false
	release := make(chan struct{})
	defer close(release)
	r.e.startJob("check:X:0", false, func(ctx context.Context, j *job) { <-release })
	r.tick() // lead: 1 proc + 1 job = 2; one delegate may start (3), not two
	if n := r.e.liveAgents(""); n != 1 {
		t.Fatalf("%d delegates started; lead + job + delegates must stay within 3", n)
	}
}

func TestProbeDropsAProfileThatDoesNotAnswer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh fixture")
	}
	ps := []Profile{{Name: "ok", Probe: []string{"sh", "-c", "echo OK"}}, {Name: "signed-out", Probe: []string{"sh", "-c", "echo 'Not logged in' >&2; exit 1"}}, {Name: "none"}}
	kept, notes := ProbeProfiles(ps, 10*time.Second)
	if len(kept) != 2 || kept[0].Name != "ok" || kept[1].Name != "none" || !strings.Contains(strings.Join(notes, ";"), "signed-out: DROPPED") {
		t.Fatalf("%+v %v", kept, notes)
	}
}

// Audit #18: the zone printed by the CLI decides, not the PC's.
func TestResetTimeUsesThePrintedZone(t *testing.T) {
	now := time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) // 13:00 in Cairo (UTC+3 summer? use LoadLocation)
	cairo, err := time.LoadLocation("Africa/Cairo")
	if err != nil {
		t.Skip("no tz database")
	}
	s := MustPrompts().Classify("You've hit your limit · resets 3pm (Africa/Cairo)", now)
	want := time.Date(2026, 10, 30, 15, 0, 0, 0, cairo)
	if !s.Reset.Equal(want) {
		t.Fatalf("reset %v, want %v", s.Reset, want)
	}
}

// Audit #18: files changing counts as progress even with an identical screen.
func TestWorktreeChangesAreProgress(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	f := r.screen("T1", StatusWorking, "● working\n✻ Thinking (3s)\n>\nctx\nfoot\nx")
	for i := 0; i < 30; i++ {
		os.WriteFile(filepath.Join(a.Cwd, "out.txt"), []byte(strings.Repeat("x", i+1)), 0o644)
		r.advance(time.Minute)
		r.tick()
	}
	if len(f.keys) != 0 || len(r.events("stalled")) != 0 {
		t.Fatal("an agent writing files every minute was called stalled")
	}
}

// Audit #5: checks that ran on an older commit than the branch now holds are not accepted.
func TestCheckOfAnOlderCommitIsDropped(t *testing.T) {
	r := newRig(t, Task{ID: "T1", Title: "t", Spec: "s", Accept: []string{"sleep 2"}})
	r.e.Sync = false
	r.tick()
	r.finish("T1", Answer{Status: "done"})
	r.tick() // checks start on the current head
	wt := r.e.G.Task("T1").Worktree
	os.WriteFile(filepath.Join(wt, "later.txt"), []byte("x\n"), 0o644)
	run(t, wt, "git", "add", "-A")
	run(t, wt, "git", "commit", "-qm", "pushed while checks ran")
	time.Sleep(3 * time.Second)
	r.tick()
	if len(r.events("stale-result")) != 1 || r.e.G.Task("T1").State == TaskReview {
		t.Fatal("checks of an older commit moved the task to review")
	}
}

// Soak 3 round 11: an agent blocked on a secret request was killed, its replacement finished the task and
// the owner's question was never answered. A blocked task stays blocked when its agent dies.
func TestOwnerBlockedTaskIsNotResumedByARestart(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	r.screen("T1", StatusIdle, "Please enter your API key:")
	r.tick()
	r.b.agents[a.Name].exists = false // killed while waiting for the owner
	r.tick()
	r.tick()
	if n := len(r.b.starts); n != 1 {
		t.Fatalf("%d starts: a replacement carried on past the owner's question", n)
	}
	if r.e.G.Task("T1").State != TaskBlocked || r.e.G.State != GoalNeedsOwner {
		t.Fatalf("task %s goal %s", r.e.G.Task("T1").State, r.e.G.State)
	}
}
