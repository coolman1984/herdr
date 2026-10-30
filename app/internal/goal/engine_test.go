package goal

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- fakes ----------

type fakeAgent struct {
	spec   StartSpec
	exists bool
	status string
	screen string
	sent   []string
	keys   [][]string
}

type fakeBackend struct {
	agents map[string]*fakeAgent
	starts []string
	err    error
}

func newFake() *fakeBackend { return &fakeBackend{agents: map[string]*fakeAgent{}} }

func (f *fakeBackend) Start(s StartSpec) error {
	f.starts = append(f.starts, s.Name)
	f.agents[s.Name] = &fakeAgent{spec: s, exists: true, status: StatusWorking, screen: "starting " + s.Name}
	return nil
}
func (f *fakeBackend) Observe(n string) Observation {
	if f.err != nil {
		return Observation{Err: f.err}
	}
	a := f.agents[n]
	if a == nil || !a.exists {
		return Observation{Exists: false, Status: StatusExited}
	}
	return Observation{Exists: true, Status: a.status, Screen: a.screen}
}
func (f *fakeBackend) Send(n, text string) error {
	f.agents[n].sent = append(f.agents[n].sent, text)
	return nil
}
func (f *fakeBackend) Keys(n string, k ...string) error {
	f.agents[n].keys = append(f.agents[n].keys, k)
	return nil
}
func (f *fakeBackend) Stop(n string) error {
	if a := f.agents[n]; a != nil {
		a.exists = false
	}
	return nil
}
func (f *fakeBackend) Live() ([]string, error) { return nil, nil }

type rig struct {
	t   *testing.T
	e   *Engine
	b   *fakeBackend
	now time.Time
	ram int
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
}

func testProfiles() []Profile {
	return []Profile{
		{Name: "alpha", Kind: "a", Roles: []string{"write", "review"}, Accounts: []string{"a1", "a2"}, Command: []string{"sh", "{prompt}"}},
		{Name: "beta", Kind: "b", Roles: []string{"write", "review"}, Accounts: []string{"b1"}, Command: []string{"sh", "{prompt}"}},
	}
}

// newRig makes a git project and a goal with the given tasks; checks and merges run inside the tick.
func newRig(t *testing.T, tasks ...Task) *rig {
	t.Helper()
	proj := t.TempDir()
	run(t, proj, "git", "init", "-q", "-b", "main")
	run(t, proj, "git", "config", "user.email", "t@x")
	run(t, proj, "git", "config", "user.name", "t")
	os.WriteFile(filepath.Join(proj, "README.md"), []byte("x\n"), 0o644)
	run(t, proj, "git", "add", "-A")
	run(t, proj, "git", "commit", "-qm", "init")
	r := &rig{t: t, b: newFake(), now: time.Date(2026, 10, 30, 12, 0, 0, 0, time.Local), ram: 8000}
	probe := Probes{FreeRAMMB: func() int { return r.ram }, FreeDiskMB: func(string) int { return 100000 }, LookPath: func(s string) (string, error) { return s, nil }}
	lim := DefaultLimits()
	lim.MaxAgents = 2
	g, l, err := Create(NewGoal{ID: "g1", Text: "test goal", Project: proj, Accept: []string{"test -f README.md"},
		Profiles: testProfiles(), Limits: lim, Tasks: tasks}, probe, r.now)
	if err != nil {
		t.Fatal(err)
	}
	r.e = &Engine{L: l, W: Workspace{Project: g.Project, Ledger: l}, B: r.b, P: MustPrompts(), Now: func() time.Time { return r.now },
		Probe: probe, Sync: true, G: g, jobs: map[string]*job{}}
	return r
}

func task(id string, deps ...string) Task {
	return Task{ID: id, Title: "task " + id, Spec: "do " + id, Deps: deps, Accept: []string{"true"}}
}

func (r *rig) tick() {
	r.t.Helper()
	if err := r.e.Tick(); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) advance(d time.Duration) { r.now = r.now.Add(d) }

// agentOf returns the current live agent of a task.
func (r *rig) agentOf(id string) *AgentRun {
	t := r.e.G.Task(id)
	if t == nil || t.Agent == "" {
		return nil
	}
	return r.e.G.Agents[t.Agent]
}

// finish makes the task's agent do its work: a commit (writers) and its result file.
func (r *rig) finish(id string, ans Answer) {
	r.t.Helper()
	a := r.agentOf(id)
	if a == nil {
		r.t.Fatalf("task %s has no agent", id)
	}
	if a.Role == "write" {
		os.WriteFile(filepath.Join(a.Cwd, id+".txt"), []byte(id+" "+a.Name+"\n"), 0o644)
		run(r.t, a.Cwd, "git", "add", "-A")
		run(r.t, a.Cwd, "git", "commit", "-qm", "work "+id)
	}
	ans.Msg = a.Msg
	b, _ := json.Marshal(ans)
	os.WriteFile(a.Answer, b, 0o644)
}

func (r *rig) screen(id, status, screen string) *fakeAgent {
	a := r.agentOf(id)
	f := r.b.agents[a.Name]
	f.status, f.screen = status, screen
	return f
}

func (r *rig) events(kind string) []Event {
	evs, _ := r.e.L.Events()
	var out []Event
	for _, e := range evs {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// ---------- screens ----------

const claudeTrust = `
 Do you trust the files in this folder?

 /work/project

   1. Yes, proceed
 ❯ 2. No, exit

 Enter to confirm · Esc to exit
`

func TestTrustNeedsMenuFooter(t *testing.T) {
	p := MustPrompts()
	if s := p.Classify(claudeTrust, time.Now()); s.Kind != "trust" {
		t.Fatalf("trust prompt not seen: %+v", s)
	}
	b, err := os.ReadFile("testdata/fixture-claude-answer-mentions-trust.txt")
	if err != nil {
		t.Fatal(err)
	}
	if s := p.Classify(string(b), time.Now()); s.Kind != "" {
		t.Fatalf("an answer that mentions trust was read as %s (%q)", s.Kind, s.Line)
	}
	// The prompt quoted higher up, with ordinary output below it, is not a live prompt.
	old := claudeTrust + strings.Repeat("● working on step\n", 12)
	if s := p.Classify(old, time.Now()); s.Kind == "trust" {
		t.Fatalf("scrolled-away trust prompt was answered")
	}
}

func TestTrustKeysPickYesWhenNoIsHighlighted(t *testing.T) {
	s := MustPrompts().Classify(claudeTrust, time.Now())
	if k := TrustKeys(s.Options); len(k) != 1 || k[0] != "1" {
		t.Fatalf("keys %v, want [1]", k)
	}
	agy := "Do you trust the contents of this project?\n> Yes, I trust this folder\n  No, exit\nenter to confirm"
	s = MustPrompts().Classify(agy, time.Now())
	if k := TrustKeys(s.Options); len(k) != 1 || k[0] != "Enter" {
		t.Fatalf("antigravity keys %v (%+v)", k, s)
	}
	onlyNo := "Do you trust this?\n❯ 1. No, exit\nEnter to confirm"
	if k := TrustKeys(MustPrompts().Classify(onlyNo, time.Now()).Options); k != nil {
		t.Fatalf("answered a menu with no yes option: %v", k)
	}
}

func TestSecretQuestionButNotPasswordTalk(t *testing.T) {
	p := MustPrompts()
	for _, s := range []string{"Please enter your API key:", "Paste your access token here:", "Password:", "Not logged in · Please run /login"} {
		if k := p.Classify("● some output\n"+s+"\n", time.Now()).Kind; k != "secret" {
			t.Errorf("%q read as %q, want secret", s, k)
		}
	}
	for _, s := range []string{"I added password hashing with bcrypt.", "The token parser now handles empty input.", "Tests for the API key rotation pass."} {
		if k := p.Classify("● done\n"+s+"\n> ", time.Now()).Kind; k == "secret" {
			t.Errorf("normal reply %q read as a secret request", s)
		}
	}
}

func TestParseReset(t *testing.T) {
	now := time.Date(2026, 10, 30, 14, 0, 0, 0, time.Local)
	cases := map[string]time.Time{
		"You've hit your limit · resets 3pm":      time.Date(2026, 10, 30, 15, 0, 0, 0, time.Local),
		"usage limit reached, try again at 13:30": time.Date(2026, 10, 31, 13, 30, 0, 0, time.Local),
		"You've hit your limit · resets in 0h 1m": now.Add(time.Minute),
	}
	for text, want := range cases {
		s := MustPrompts().Classify(text, now)
		if s.Kind != "rate-limited" || !s.Reset.Equal(want) {
			t.Errorf("%q: %s %v, want %v", text, s.Kind, s.Reset, want)
		}
	}
}

func TestProgressHashIgnoresTimerAndSpinner(t *testing.T) {
	a := "● Reading files\n  src/main.go\n✻ Thinking… (12s · 340 tokens · esc to interrupt)\n> \n  ctx 41%\n  ⏵⏵ auto mode"
	b := "● Reading files\n  src/main.go\n✶ Thinking… (13s · 351 tokens · esc to interrupt)\n> \n  ctx 41%\n  ⏵⏵ auto mode"
	if ProgressHash(a) != ProgressHash(b) {
		t.Fatal("a running timer counts as progress: stall can never fire")
	}
	c := "● Reading files\n  src/main.go\n● Edited src/util.go\n✶ Thinking… (13s)\n> \n  ctx 41%\n  ⏵⏵ auto mode"
	if ProgressHash(a) == ProgressHash(c) {
		t.Fatal("new output is not progress")
	}
}

// ---------- watchdog ----------

func TestTrustInsideProjectIsAnswered(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	f := r.screen("T1", StatusBlocked, claudeTrust)
	r.tick()
	if len(f.keys) != 1 || f.keys[0][0] != "1" {
		t.Fatalf("keys %v, want [[1]]", f.keys)
	}
	r.tick() // same screen again: the answer is not repeated while the CLI reacts
	if len(f.keys) != 1 {
		t.Fatalf("answered the same screen twice: %v", f.keys)
	}
}

func TestTrustOutsideProjectGoesToOwner(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	a.Cwd = t.TempDir() // not inside the project
	f := r.screen("T1", StatusBlocked, claudeTrust)
	r.tick()
	if len(f.keys) != 0 || len(f.sent) != 0 {
		t.Fatalf("answered a trust prompt outside the project: %v %v", f.keys, f.sent)
	}
	if r.e.G.Task("T1").State != TaskBlocked {
		t.Fatalf("task %s, want blocked", r.e.G.Task("T1").State)
	}
}

func TestNothingIsTypedIntoASecretScreen(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	f := r.screen("T1", StatusIdle, "● Setting up the deploy\nPlease enter your API key:")
	for i := 0; i < 5; i++ {
		r.advance(time.Minute)
		r.tick()
	}
	if len(f.keys) != 0 || len(f.sent) != 0 {
		t.Fatalf("typed into a secret prompt: keys %v sent %v", f.keys, f.sent)
	}
	esc := r.e.G.Escalations[0]
	// The lead may not answer the owner's escalation; the owner's own answer is still refused at send time.
	if err := r.e.apply(Command{Cmd: "answer", By: "lead", Args: json.RawMessage(`{"esc":"` + esc.ID + `","text":"sk-123"}`)}); err == nil {
		t.Fatal("the lead answered an owner escalation")
	}
	r.e.apply(Command{Cmd: "answer", By: "owner", Args: json.RawMessage(`{"esc":"` + esc.ID + `","text":"hello"}`)})
	if len(f.sent) != 0 {
		t.Fatalf("pre-send check let text into a secret screen: %v", f.sent)
	}
}

func TestDeadAgentRestartsAndKeepsWork(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	os.WriteFile(filepath.Join(a.Cwd, "half.txt"), []byte("half\n"), 0o644)
	r.b.agents[a.Name].exists = false
	r.tick()
	b := r.agentOf("T1")
	if b == nil || b.Name == a.Name {
		t.Fatal("dead agent not restarted")
	}
	log, _ := git(b.Cwd, "log", "--oneline", "-1")
	if !strings.Contains(log, "save uncommitted work") {
		t.Fatalf("work of the dead agent not committed: %s", log)
	}
}

func TestBackendErrorIsNotDeath(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	r.b.err = errors.New("herdr: connection refused")
	for i := 0; i < 6; i++ {
		r.tick()
	}
	r.b.err = nil
	if r.agentOf("T1") != a || len(r.b.starts) != 1 {
		t.Fatalf("a Herdr error was read as a dead agent: starts %v", r.b.starts)
	}
}

func TestIdleWithoutResultIsNotDone(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	f := r.screen("T1", StatusIdle, "● I looked at it.\n> ")
	a.Begun = true
	r.tick() // first idle sighting
	r.advance(2 * time.Minute)
	r.tick()
	if len(f.keys) != 1 || f.keys[0][0] != "Enter" {
		t.Fatalf("first nudge %v, want empty Enter", f.keys)
	}
	r.advance(2 * time.Minute)
	r.tick()
	if len(f.sent) != 1 || !strings.Contains(f.sent[0], "continue") {
		t.Fatalf("second nudge %v, want continue", f.sent)
	}
	for i := 0; i < 4; i++ {
		r.advance(2 * time.Minute)
		r.tick()
	}
	if r.agentOf("T1") == a {
		t.Fatal("an agent that keeps stopping without a result is never restarted")
	}
	if r.e.G.Task("T1").State == TaskDone {
		t.Fatal("idle counted as done")
	}
}

func TestStalledAgentIsNudged(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	f := r.screen("T1", StatusWorking, "● step\n● step 2\n✻ Working (1s)\n>\nctx\nfoot")
	r.tick()
	for i := 0; i < 30; i++ {
		r.advance(time.Minute)
		f.screen = strings.Replace(f.screen, "(1s)", "("+string(rune('a'+i))+"s)", 1)
		r.tick()
	}
	if len(f.keys) == 0 {
		t.Fatal("stall never fired on an unchanged screen with a running timer")
	}
}

func TestNotStartedResendsBrief(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	f := r.screen("T1", StatusIdle, "> ")
	r.tick()
	r.advance(4 * time.Minute)
	r.tick()
	if len(f.sent) != 1 || !strings.Contains(f.sent[0], "Read and follow the brief") {
		t.Fatalf("brief not resent to an agent that never began: %v", f.sent)
	}
}

func TestCapacityContinuesThenSwitchesModel(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	f := r.screen("T1", StatusIdle, "■ Selected model is at capacity. Please try a different model.\n")
	r.tick()
	if len(f.sent) != 1 || f.sent[0] != "continue from where you stopped" {
		t.Fatalf("sent %v", f.sent)
	}
	for i := 0; i < 4; i++ {
		f.screen += " \n"
		r.tick()
	}
	b := r.agentOf("T1")
	if b == nil || b == a || b.Profile == a.Profile {
		t.Fatalf("no model switch after capacity errors: %+v", b)
	}
}

func TestAccountLimitsRotateThenWaitThenResume(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	limit := func() {
		r.screen("T1", StatusIdle, "You've hit your limit · resets in 1h 0m\n")
		r.tick()
	}
	limit()
	if a := r.agentOf("T1"); a.Profile != "alpha" || a.Account != "a2" {
		t.Fatalf("after a1 limit: %s/%s, want alpha/a2", a.Profile, a.Account)
	}
	limit()
	if a := r.agentOf("T1"); a.Profile != "beta" {
		t.Fatalf("after a2 limit: %s, want beta", a.Profile)
	}
	limit()
	if r.e.G.State != GoalWaitingLimit {
		t.Fatalf("state %s, want waiting-limit (never the owner)", r.e.G.State)
	}
	n := len(r.b.starts)
	r.advance(30 * time.Minute)
	r.tick()
	if len(r.b.starts) != n {
		t.Fatal("started an agent while every account is limited")
	}
	r.advance(31 * time.Minute)
	r.tick()
	if r.e.G.State != GoalRunning || len(r.b.starts) != n+1 {
		t.Fatalf("did not resume by itself at the reset: %s, starts %d", r.e.G.State, len(r.b.starts)-n)
	}
}

// ---------- crash and resume ----------

func TestRecoverNeitherLosesNorDoublesAStart(t *testing.T) {
	r := newRig(t, task("T1"))
	started := Action{Kind: "start", Agent: "w-x-1", Start: StartSpec{Name: "w-x-1"}}
	lost := Action{Kind: "start", Agent: "w-y-2", Start: StartSpec{Name: "w-y-2"}}
	r.b.Start(started.Start) // happened before the crash
	r.b.starts = nil
	r.e.G.Intents = []*Intent{{ID: "i1", Action: started}, {ID: "i2", Action: lost}}
	if err := r.e.Recover(); err != nil {
		t.Fatal(err)
	}
	if len(r.b.starts) != 1 || r.b.starts[0] != "w-y-2" {
		t.Fatalf("starts after recovery %v, want only the lost one", r.b.starts)
	}
	if len(r.e.G.Intents) != 0 {
		t.Fatalf("intents left: %d", len(r.e.G.Intents))
	}
}

func TestLedgerSurvivesTornWrites(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	want, _ := json.Marshal(r.e.G)
	os.WriteFile(filepath.Join(r.e.L.Dir, "goal.json.tmp"), []byte(`{"id":"half`), 0o644) // kill mid-Save
	f, _ := os.OpenFile(filepath.Join(r.e.L.Dir, "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(`{"seq":999,"kind":"tor`) // kill mid-Append
	f.Close()
	g, err := r.e.L.Load()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(g)
	if string(got) != string(want) {
		t.Fatal("state changed by a torn write")
	}
	evs, err := r.e.L.Events()
	if err != nil || len(evs) == 0 || evs[len(evs)-1].Seq == 999 {
		t.Fatalf("torn event line not skipped: %v %d", err, len(evs))
	}
}

// ---------- scheduling, review, merge, done ----------

func TestMaxAgentsAndFreeMemoryGate(t *testing.T) {
	r := newRig(t, task("T1"), task("T2"), task("T3"))
	r.ram = 900
	r.tick()
	if len(r.b.starts) != 0 {
		t.Fatal("started an agent with free RAM under the minimum")
	}
	r.ram = 8000
	r.tick()
	if len(r.b.starts) != 2 {
		t.Fatalf("starts %d, want MaxAgents=2", len(r.b.starts))
	}
}

func TestFileReservationsSerializeTasks(t *testing.T) {
	a, b := task("T1"), task("T2")
	a.Files, b.Files = []string{"main.go"}, []string{"main.go"}
	r := newRig(t, a, b)
	r.tick()
	if r.e.G.Task("T2").State != TaskPending || r.e.G.Task("T1").State != TaskRunning {
		t.Fatal("two tasks with the same reserved file ran together")
	}
}

func TestDependenciesAndIndependentReviewToDone(t *testing.T) {
	r := newRig(t, task("T1"), task("T2", "T1"))
	r.tick()
	if r.e.G.Task("T2").State != TaskPending {
		t.Fatal("T2 started before its dependency")
	}
	r.finish("T1", Answer{Status: "done", Summary: "ok"})
	r.tick()
	r.tick() // result -> checks -> review started
	rev := r.agentOf("T1")
	if rev == nil || rev.Role != "review" || r.e.kindOf(rev.Profile) == r.e.kindOf(r.e.G.Task("T1").Writer) {
		t.Fatalf("reviewer %+v is not a different vendor", rev)
	}
	r.finish("T1", Answer{Status: "done", Verdict: "pass", Summary: "good"})
	r.tick()
	if r.e.G.Task("T1").State != TaskDone {
		t.Fatalf("T1 %s", r.e.G.Task("T1").State)
	}
	r.tick()
	r.finish("T2", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.finish("T2", Answer{Status: "done", Verdict: "fail", Summary: "bug", Notes: "fix the bug"})
	r.tick()
	if r.e.G.Task("T2").State != TaskRunning || !strings.Contains(r.e.G.Task("T2").Feedback, "fix the bug") {
		t.Fatalf("rejected work did not go back with the notes: %+v", r.e.G.Task("T2"))
	}
	r.tick()
	r.finish("T2", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.finish("T2", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	r.tick()
	if r.e.G.State != GoalDone {
		t.Fatalf("goal %s (%s)", r.e.G.State, r.e.G.Reason)
	}
}

func TestDoneOnlyAfterTheSupervisorsOwnAcceptance(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Accept = []string{"test -f never-made.txt"}
	r.tick()
	r.finish("T1", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.finish("T1", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	r.tick()
	if r.e.G.State == GoalDone {
		t.Fatal("goal done although its acceptance fails")
	}
}

func TestDoneWithoutCommitIsRejected(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	a := r.agentOf("T1")
	b, _ := json.Marshal(Answer{Msg: a.Msg, Status: "done"})
	os.WriteFile(a.Answer, b, 0o644)
	r.tick()
	if st := r.e.G.Task("T1").State; st != TaskRunning || r.e.G.Task("T1").Attempts != 1 {
		t.Fatalf("a done with no commit was accepted: %s", st)
	}
}

func TestMergeConflictGoesBackToWriterAndRevertKeepsOthers(t *testing.T) {
	r := newRig(t, task("T1"), task("T2"))
	r.tick()
	for _, id := range []string{"T1", "T2"} { // both edit README.md differently
		a := r.agentOf(id)
		os.WriteFile(filepath.Join(a.Cwd, "README.md"), []byte(id+"\n"), 0o644)
	}
	r.finish("T1", Answer{Status: "done"})
	r.finish("T2", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.tick()
	r.finish("T1", Answer{Status: "done", Verdict: "pass"})
	r.finish("T2", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	r.tick() // one merge per tick
	t1, t2 := r.e.G.Task("T1"), r.e.G.Task("T2")
	if t1.State != TaskDone || t2.State != TaskRunning || !strings.Contains(t2.Feedback, "merge conflict") {
		t.Fatalf("T1 %s, T2 %s %q", t1.State, t2.State, t2.Feedback)
	}
	// Revert T1's merge: a later good merge must stay. Add T3 and merge it first.
	r.e.apply(Command{Cmd: "task-cancel", By: "lead", Args: json.RawMessage(`{"id":"T2"}`)})
	t3 := task("T3")
	raw, _ := json.Marshal(t3)
	r.e.apply(Command{Cmd: "task-add", By: "lead", Args: raw})
	r.tick()
	r.finish("T3", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.finish("T3", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	if err := r.e.apply(Command{Cmd: "revert", By: "lead", Args: json.RawMessage(`{"id":"T1"}`)}); err != nil {
		t.Fatal(err)
	}
	main := r.e.W.MainDir()
	if _, err := os.Stat(filepath.Join(main, "T1.txt")); err == nil {
		t.Fatal("T1 still on the goal branch after revert")
	}
	if _, err := os.Stat(filepath.Join(main, "T3.txt")); err != nil {
		t.Fatal("revert of T1 lost the later T3 merge")
	}
}

func TestFailingProjectTestsUndoTheMerge(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Tests = []string{"test ! -f T1.txt"} // T1's change breaks the project's tests
	head, _ := r.e.W.Head()
	r.tick()
	r.finish("T1", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.finish("T1", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	after, _ := r.e.W.Head()
	if after != head || r.e.G.Task("T1").State == TaskDone {
		t.Fatalf("a merge that broke the tests stayed: %s -> %s", short(head), short(after))
	}
}

func TestLeadCannotStopTheGoal(t *testing.T) {
	r := newRig(t, task("T1"))
	if err := r.e.apply(Command{Cmd: "stop", By: "lead"}); err == nil {
		t.Fatal("the lead stopped the goal")
	}
	if err := r.e.apply(Command{Cmd: "stop", By: "owner"}); err != nil || r.e.G.State != GoalStopped {
		t.Fatalf("owner stop: %v %s", err, r.e.G.State)
	}
}

func TestCreateRefusesNonGitFolderAndCostlessProfiles(t *testing.T) {
	probe := Probes{FreeRAMMB: func() int { return 8000 }, FreeDiskMB: func(string) int { return 1 << 20 }, LookPath: func(s string) (string, error) { return s, nil }}
	_, _, err := Create(NewGoal{ID: "g", Text: "x", Project: t.TempDir(), Accept: []string{"true"}, Profiles: testProfiles()}, probe, time.Now())
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("non-git folder: %v", err)
	}
	one := []Profile{testProfiles()[0]}
	if err := Preflight(one, "", probe); err == nil {
		t.Fatal("a goal with no reviewer of another vendor was allowed")
	}
}

func TestNoProgressStops(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	r.screen("T1", StatusWorking, "same")
	r.e.G.Limits.StallAfter = Duration(10 * time.Hour)
	r.advance(46 * time.Minute)
	r.tick()
	if r.e.G.State != GoalStopped {
		t.Fatalf("state %s after 46 min with no event", r.e.G.State)
	}
}

func TestOnlySecretBlockedTasksLeftMeansNeedsOwnerEvenWithALead(t *testing.T) {
	r := newRig(t, task("T1"), task("T2"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.e.G.Limits.MaxAgents = 3
	r.tick()
	r.screen("T2", StatusIdle, "Please enter your API key:")
	r.finish("T1", Answer{Status: "done"})
	r.tick()
	r.tick()
	r.finish("T1", Answer{Status: "done", Verdict: "pass"})
	r.tick()
	r.tick()
	if r.e.G.State != GoalNeedsOwner || r.e.G.Task("T1").State != TaskDone {
		t.Fatalf("state %s (%s), T1 %s", r.e.G.State, r.e.G.Reason, r.e.G.Task("T1").State)
	}
}

func TestLeadHandsOffWhenContextIsLow(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.e.G.Limits.MaxAgents = 3
	r.tick()
	var lead *AgentRun
	for _, a := range r.e.G.Agents {
		if a.Role == "lead" {
			lead = a
		}
	}
	f := r.b.agents[lead.Name]
	f.status, f.screen = StatusIdle, "● planning\nContext left until auto-compact: 3%\n"
	r.tick()
	if len(f.sent) != 1 || !strings.Contains(f.sent[0], "handoff.md") {
		t.Fatalf("lead not asked for a handoff: %v", f.sent)
	}
	for i := 0; i < 5; i++ {
		f.screen += " \n"
		r.tick()
	}
	n := 0
	for _, a := range r.e.G.Agents {
		if a.Role == "lead" && !a.Ended {
			n++
			if a.Name == lead.Name {
				t.Fatal("old lead still running")
			}
		}
	}
	if n != 1 {
		t.Fatalf("%d live leads after handoff, want 1", n)
	}
}

func TestMenuGoesToDeciderAndOnlyItsValidPick(t *testing.T) {
	r := newRig(t, task("T1"))
	menu := " Which format?\n ❯ 1. Plain\n   2. JSON\n Enter to confirm · Esc to cancel\n"
	picked := "2"
	r.e.Decide = func(g *Goal, a *AgentRun, screen string, opts []Option) ([]string, string, error) {
		for _, o := range opts {
			if o.Num == picked {
				return []string{o.Num}, "json fits the task", nil
			}
		}
		return nil, "", errors.New("not on screen")
	}
	r.tick()
	f := r.screen("T1", StatusBlocked, menu)
	r.tick()
	if len(f.keys) != 1 || f.keys[0][0] != "2" {
		t.Fatalf("keys %v, want the decider's pick", f.keys)
	}
	picked = "9" // a pick that is not on the screen is refused; the CLI default is taken and recorded
	f.screen += "\n"
	r.tick()
	if len(f.keys) != 2 || f.keys[1][0] != "Enter" || len(r.e.G.Assumptions) == 0 {
		t.Fatalf("keys %v assumptions %v", f.keys, r.e.G.Assumptions)
	}
}

type failingStart struct{ *fakeBackend }

func (f failingStart) Start(s StartSpec) error {
	f.starts = append(f.starts, s.Name)
	return errors.New("exec: \"codex\": executable file not found")
}

func TestAgentsThatCannotStartAreCapped(t *testing.T) {
	r := newRig(t, task("T1"))
	r.e.G.Lead = "alpha"
	r.e.G.Profiles[0].Roles = append(r.e.G.Profiles[0].Roles, "lead")
	r.e.B = failingStart{r.b}
	for i := 0; i < 60; i++ {
		r.tick()
	}
	if n := len(r.b.starts); n > 25 {
		t.Fatalf("%d start attempts in 60 ticks: no cap", n)
	}
	if r.e.G.Task("T1").State != TaskFailed || r.e.G.State != GoalNeedsOwner {
		t.Fatalf("task %s goal %s", r.e.G.Task("T1").State, r.e.G.State)
	}
}

func TestHealthyAgentKilledManyTimesKeepsRestarting(t *testing.T) {
	r := newRig(t, task("T1"))
	r.tick()
	for i := 0; i < 8; i++ {
		a := r.agentOf("T1")
		a.Begun = true
		r.advance(5 * time.Minute) // it worked for a while, then something killed it
		r.b.agents[a.Name].exists = false
		r.tick()
	}
	if len(r.events("model-switch")) > 0 || r.agentOf("T1") == nil || r.e.G.Task("T1").Attempts != 0 || r.e.G.State != GoalRunning {
		t.Fatalf("outside kills of a healthy agent used up its caps: %+v %s", r.e.G.Task("T1"), r.e.G.State)
	}
}

func TestNeedsOwnerLeavesNoUnwatchedAgents(t *testing.T) {
	r := newRig(t, task("T1"), task("T2"))
	r.tick()
	r.screen("T2", StatusIdle, "Please enter your API key:")
	r.tick()
	r.e.setState(GoalNeedsOwner, "test")
	for _, a := range r.e.G.Agents {
		if !a.Ended && a.State != "needs-owner" {
			t.Fatalf("%s left running unwatched", a.Name)
		}
	}
	if a := r.agentOf("T2"); a == nil || a.Ended {
		t.Fatal("the agent showing the owner's prompt was closed")
	}
}
