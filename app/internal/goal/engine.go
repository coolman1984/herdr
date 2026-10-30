package goal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Probes are the machine checks; replaced by fakes in tests.
type Probes struct {
	FreeRAMMB  func() int
	FreeDiskMB func(string) int
	LookPath   func(string) (string, error)
}

func RealProbes() Probes { return Probes{FreeRAMMB, FreeDiskMB, exec.LookPath} }

// Decider picks an option for a routine menu (a CLI question that is not trust or a secret).
// It returns the keys to send. A nil Decider means: take the CLI's highlighted default and record it.
type Decider func(g *Goal, agent *AgentRun, screen string, opts []Option) ([]string, string, error)

// Engine is the Supervisor: it never thinks about the goal, it keeps the loop alive and honest.
type Engine struct {
	L         Ledger
	W         Workspace
	B         Backend
	P         Prompts
	Now       func() time.Time
	Probe     Probes
	Decide    Decider
	Q         *Quotas // account quotas; nil = every account unknown (low trust)
	Skills    *SkillCatalog
	Caps      *CapReport // the machine's capability report (skills and budgets use it)
	Sync      bool       // run checks and merges inside the tick (tests); else in the background
	Gen       int        // this runner's generation; a newer one on disk means this runner must stop writing
	G         *Goal
	Log       func(string)
	mu        sync.Mutex
	jobs      map[string]*job
	decisions map[string]*decision
	pending   []Event
}

type job struct {
	cancelled bool
	cancel    context.CancelFunc
	main      bool
	done      bool
	ok        bool
	out       string
	sha       string
	err       error
}

func Open(dir string, b Backend) (*Engine, error) {
	l := Ledger{Dir: dir}
	g, err := l.Load()
	if err != nil {
		return nil, err
	}
	p, err := LoadPrompts(filepath.Join(dir, "prompts.json"))
	if err != nil {
		return nil, err
	}
	return &Engine{L: l, W: Workspace{Project: g.Project, Ledger: l}, B: b, P: p, Now: time.Now,
		Probe: RealProbes(), G: g, jobs: map[string]*job{}}, nil
}

func (e *Engine) event(kind, agent, task, detail string, screen ...string) {
	e.G.Seq++
	ev := Event{Seq: e.G.Seq, At: T(e.Now()), Kind: kind, Agent: agent, Task: task, Detail: detail}
	if len(screen) > 0 && screen[0] != "" {
		ev.Screen = tailText(screen[0], 25)
	}
	e.pending = append(e.pending, ev)
	if progressKinds[kind] {
		e.G.LastEvent = T(e.Now())
	}
	if e.Log != nil {
		e.Log(fmt.Sprintf("%s %-14s %-10s %-6s %s", e.Now().Format("15:04:05"), kind, agent, task, oneLine(detail, 140)))
	}
}

// progressKinds are the events that move the product forward. Restarts, nudges, deaths and retries
// are activity, not progress: a goal stuck in a restart loop must still hit the no-progress limit.
var progressKinds = map[string]bool{
	"goal-created": true, "task-added": true, "result": true, "task-checking": true, "task-review": true,
	"task-approved": true, "task-done": true, "acceptance-passed": true, "reverted": true, "command": true,
	"goal-running": true, "runner-start": false,
}

// ErrSuperseded: a newer runner generation owns the ledger; this one must stop.
var ErrSuperseded = errors.New("a newer runner owns this goal; stopping")

// flush writes events first, then the state: an event can at worst be duplicated, never lost.
func (e *Engine) flush() error {
	if e.Gen > 0 {
		if disk, err := e.L.Load(); err == nil && disk.RunnerGen > e.Gen {
			e.pending = nil
			return ErrSuperseded
		}
	}
	if err := e.L.Append(e.pending...); err != nil {
		return err
	}
	e.pending = nil
	return e.L.Save(e.G)
}

// do runs one outward action under an intent record: written before, removed after.
func (e *Engine) do(a Action) error {
	in := &Intent{ID: fmt.Sprintf("i%d", e.G.Seq+1), Action: a, At: T(e.Now())}
	e.G.Seq++
	e.G.Intents = append(e.G.Intents, in)
	if err := e.flush(); err != nil {
		return err
	}
	err := e.perform(a)
	for i, x := range e.G.Intents {
		if x == in {
			e.G.Intents = append(e.G.Intents[:i], e.G.Intents[i+1:]...)
			break
		}
	}
	return errors.Join(err, e.flush())
}

// Action is one outward step. Kinds: start, send, keys, stop.
type Action struct {
	Kind  string    `json:"kind"`
	Agent string    `json:"agent"`
	Text  string    `json:"text,omitempty"`
	Keys  []string  `json:"keys,omitempty"`
	Start StartSpec `json:"start,omitempty"`
}

// perform checks the recipient's screen right before every send: nothing is ever typed into a pane that
// shows a secret or sign-in request, whatever path asked for it.
func (e *Engine) perform(a Action) error {
	switch a.Kind {
	case "start":
		return e.B.Start(a.Start)
	case "stop":
		return e.B.Stop(a.Agent)
	case "send", "keys":
		if ag := e.G.Agents[a.Agent]; ag == nil || ag.Ended {
			return fmt.Errorf("refused to type into %s: no live agent run of this goal owns it", a.Agent)
		}
		o := e.B.Observe(a.Agent)
		if o.Err != nil {
			return o.Err
		}
		if !o.Exists || o.Status == StatusExited {
			// The CLI is gone; whatever is left (a shell) must never get agent text as commands.
			return fmt.Errorf("refused to type into %s: the agent has exited", a.Agent)
		}
		if s := e.P.Classify(o.Screen, e.Now()); s.Kind == "secret" {
			return fmt.Errorf("refused to type into %s: it shows a secret request (%q)", a.Agent, s.Line)
		}
		if a.Kind == "send" {
			return e.B.Send(a.Agent, a.Text)
		}
		return e.B.Keys(a.Agent, a.Keys...)
	}
	return fmt.Errorf("unknown action %q", a.Kind)
}

// Recover runs once when a runner takes over a ledger: every intent left by a crash is reconciled with
// what is really there, so an action is neither lost nor doubled.
func (e *Engine) Recover() error {
	left := e.G.Intents
	e.G.Intents = nil
	for _, in := range left {
		a := in.Action
		switch a.Kind {
		case "start":
			if o := e.B.Observe(a.Agent); o.Exists {
				e.event("reconciled", a.Agent, "", "start had happened before the crash; agent kept")
				continue
			}
			e.event("reconciled", a.Agent, "", "start had not happened; starting now")
			if err := e.do(a); err != nil {
				e.event("error", a.Agent, "", err.Error())
			}
		case "stop":
			e.B.Stop(a.Agent)
			e.event("reconciled", a.Agent, "", "stop repeated (idempotent)")
		default:
			// A send or key press is not repeated blindly: the next tick looks at the screen and the
			// message acknowledgements and decides again.
			e.event("reconciled", a.Agent, "", a.Kind+" dropped; the watchdog re-decides from the screen")
		}
	}
	// A merge interrupted by the crash: put the goal branch back to a clean state.
	if _, err := os.Stat(e.W.MainDir()); err == nil {
		if _, err := git(e.W.MainDir(), "rev-parse", "-q", "--verify", "MERGE_HEAD"); err == nil {
			git(e.W.MainDir(), "merge", "--abort")
			e.event("reconciled", "", "", "aborted a merge left by the crash")
		}
		git(e.W.MainDir(), "reset", "-q", "--hard")
	}
	if e.G.Verify != nil && !e.G.Verify.Done {
		e.G.Verify = nil // its process died with the runner; run it again
	}
	// Agents the ledger does not know (started outside an intent) are not ours: never touched.
	return e.flush()
}

// ---------- the tick ----------

func (e *Engine) Tick() error {
	g := e.G
	if err := e.applyInbox(); err != nil {
		e.event("error", "", "", "inbox: "+err.Error())
	}
	switch g.State {
	case GoalDone, GoalFailed, GoalStopped, GoalPaused:
		return e.flush()
	case GoalWaitingLimit:
		if e.Now().Before(g.Waiting.Time) {
			return e.flush()
		}
		e.setState(GoalRunning, "account limit reset reached; resuming")
	case GoalPlanning:
		if len(g.Tasks) > 0 {
			e.setState(GoalRunning, "plan has tasks")
		}
	}
	if d := g.Limits.MaxDuration.D(); d > 0 && e.Now().Sub(g.Created.Time) > d {
		e.stopAll("time limit")
		e.setState(GoalFailed, fmt.Sprintf("time limit %s reached", d))
		return e.flush()
	}
	if d := g.Limits.NoProgress.D(); d > 0 && !g.LastEvent.IsZero() && e.Now().Sub(g.LastEvent.Time) > d {
		e.stopAll("no progress")
		e.setState(GoalStopped, fmt.Sprintf("no progress for %s; stopped, work kept", d))
		return e.flush()
	}
	for _, name := range sortedKeys(g.Agents) {
		if a := g.Agents[name]; !a.Ended {
			e.watch(a)
		}
	}
	if g.State == GoalWaitingLimit {
		return e.flush()
	}
	e.lead() // the lead takes its share of the process budget before delegates
	e.schedule()
	e.verify()
	return e.flush()
}

func (e *Engine) setState(s, reason string) {
	if e.G.State == s && e.G.Reason == reason {
		return
	}
	e.G.State, e.G.Reason = s, reason
	e.event("goal-"+s, "", "", reason)
	if s == GoalNeedsOwner {
		// The runner ends here: nothing may keep running unwatched. Only an agent showing the owner's
		// prompt (secret, sign-in, outside trust) stays, so the owner can answer it in its pane.
		for _, name := range sortedKeys(e.G.Agents) {
			if a := e.G.Agents[name]; !a.Ended && a.State != "needs-owner" {
				e.endAgent(a, "goal needs the owner")
			}
		}
	}
}

func (e *Engine) stopAll(why string) {
	for _, name := range sortedKeys(e.G.Agents) {
		if a := e.G.Agents[name]; !a.Ended {
			e.endAgent(a, why)
		}
	}
}

// endAgent stops an agent and confirms it is gone. If the stop cannot be confirmed the run stays owned
// and counted (state stop-pending) and the watchdog retries the stop; it is never reported as stopped.
func (e *Engine) endAgent(a *AgentRun, why string) bool {
	err := e.do(Action{Kind: "stop", Agent: a.Name})
	if err == nil {
		if o := e.B.Observe(a.Name); o.Err == nil && o.Exists && o.Status != StatusExited {
			err = errors.New("still running after the stop")
		} else if o.Err != nil {
			err = o.Err
		}
	}
	if err != nil {
		a.State = "stop-pending"
		e.event("stop-unconfirmed", a.Name, a.Task, why+": "+err.Error())
		return false
	}
	a.Ended = true
	a.State = "ended"
	e.event("agent-stopped", a.Name, a.Task, why)
	return true
}

func (e *Engine) liveAgents(role string) int {
	n := 0
	for _, a := range e.G.Agents {
		if !a.Ended && (role == "" && a.Role != "lead" || a.Role == role) {
			n++
		}
	}
	return n
}

// ---------- answers and acknowledgements ----------

// Answer is the result file every agent writes last.
type Answer struct {
	Msg     string `json:"msg"`
	Status  string `json:"status"`  // done, blocked, failed
	Verdict string `json:"verdict"` // reviewers: pass or fail
	Summary string `json:"summary"`
	Notes   string `json:"notes"`
}

func (e *Engine) readAnswer(a *AgentRun) (*Answer, error) {
	b, err := os.ReadFile(a.Answer)
	if err != nil {
		return nil, nil
	}
	var ans Answer
	if err := json.Unmarshal(b, &ans); err != nil {
		return nil, fmt.Errorf("answer file %s is not valid JSON: %v", a.Answer, err)
	}
	if ans.Msg != a.Msg {
		return nil, fmt.Errorf("answer file %s answers message %q, expected %q", a.Answer, ans.Msg, a.Msg)
	}
	return &ans, nil
}

func (e *Engine) ackPath(msg string) string { return filepath.Join(e.L.AckDir(), msg) }

func (e *Engine) checkAck(a *AgentRun) {
	if a.Acked {
		return
	}
	if _, err := os.Stat(e.ackPath(a.Msg)); err == nil {
		a.Acked, a.Begun = true, true
		for _, m := range e.G.Messages {
			if m.ID == a.Msg {
				m.Acked = T(e.Now())
			}
		}
		e.event("acked", a.Name, a.Task, "message "+a.Msg+" acknowledged")
	}
}

// insideProject is the trust gate: the agent's folder, with links resolved, must be inside the project.
func insideProject(project, dir string) bool {
	p, err1 := filepath.EvalSymlinks(project)
	d, err2 := filepath.EvalSymlinks(dir)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(p, d)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
