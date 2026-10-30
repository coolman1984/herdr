// Package goal is the Herdr goal engine: a durable ledger, a Supervisor that keeps the loop alive,
// an Agent Watchdog, git worktree/merge handling and the herdr-goal command. Standard library only.
// Spec: docs/SPEC-GOAL-ENGINE.md. Behaviour notes: docs/GOAL-ENGINE.md.
package goal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Goal states.
const (
	GoalPlanning     = "planning"
	GoalRunning      = "running"
	GoalVerifying    = "verifying"
	GoalDone         = "done"
	GoalPaused       = "paused"
	GoalNeedsOwner   = "needs-owner"
	GoalFailed       = "failed"
	GoalWaitingLimit = "waiting-limit"
	GoalStopped      = "stopped"
)

// Task states.
const (
	TaskPending   = "pending"   // waiting for dependencies
	TaskRunning   = "running"   // a writer agent owns it
	TaskReview    = "review"    // a reviewer agent owns it
	TaskChecking  = "checking"  // writer finished; the Supervisor runs the task's own checks
	TaskMerging   = "merging"   // approved, waiting for merge + full tests
	TaskDone      = "done"      // merged and full tests green
	TaskFailed    = "failed"    // out of attempts, Architect must re-plan
	TaskBlocked   = "blocked"   // needs the owner (secret, sign-in)
	TaskCancelled = "cancelled" // removed by the Architect
)

type Limits struct {
	MaxDuration     Duration `json:"max_duration"`
	MaxTasks        int      `json:"max_tasks"`
	MaxAttempts     int      `json:"max_attempts"`      // writer attempts per task before it fails
	MaxAgents       int      `json:"max_agents"`        // delegates alive at once (lead excluded)
	MaxProcs        int      `json:"max_procs"`         // lead + delegates + background jobs at once
	AgentRAMMB      int      `json:"agent_ram_mb"`      // RAM one more agent is expected to need
	MinFreeRAMMB    int      `json:"min_free_ram_mb"`   // no new agent below this
	MinFreeDiskMB   int      `json:"min_free_disk_mb"`  // no new agent below this
	NoProgress      Duration `json:"no_progress"`       // no progress event (result, merge, re-plan ...) this long: stop and report
	StartTimeout    Duration `json:"start_timeout"`     // agent must show it began (ack/working/output)
	AckTimeout      Duration `json:"ack_timeout"`       // message must be acknowledged
	StallAfter      Duration `json:"stall_after"`       // unchanged normalized screen while working
	IdleGrace       Duration `json:"idle_grace"`        // idle with no result this long = stopped early
	RetryCap        int      `json:"retry_cap"`         // per problem kind per agent run
	VerifyTimeout   Duration `json:"verify_timeout"`    // each acceptance command
	DefaultLimitFor Duration `json:"default_limit_for"` // account limit without a printed reset time
}

func DefaultLimits() Limits {
	return Limits{
		MaxDuration: Duration(8 * time.Hour), MaxTasks: 40, MaxAttempts: 3, MaxAgents: 2, MaxProcs: 4, AgentRAMMB: 600,
		MinFreeRAMMB: 1536, MinFreeDiskMB: 2048, NoProgress: Duration(45 * time.Minute),
		StartTimeout: Duration(3 * time.Minute), AckTimeout: Duration(3 * time.Minute),
		StallAfter: Duration(20 * time.Minute), IdleGrace: Duration(90 * time.Second), RetryCap: 3,
		VerifyTimeout: Duration(20 * time.Minute), DefaultLimitFor: Duration(time.Hour),
	}
}

// Profile is one way to start an agent: a CLI + model on an account.
type Profile struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // vendor family; a reviewer must differ from the writer
	// Provider names the real subscription (anthropic, openai, google, cline); profiles with the same
	// provider and account share one quota. Empty = Kind.
	Provider string   `json:"provider,omitempty"`
	Command  []string `json:"command"` // argv; {prompt} is replaced by the first prompt when present
	Model    string   `json:"model,omitempty"`
	// Fallback is tried after RetryCap capacity/model errors.
	Fallback string            `json:"fallback,omitempty"`
	Accounts []string          `json:"accounts,omitempty"` // tried in order on account limits
	Env      map[string]string `json:"env,omitempty"`      // {account} is replaced
	Roles    []string          `json:"roles,omitempty"`    // write, review, lead
	// Probe is a tiny real request (e.g. claude -p "Reply OK") run at goal creation: it proves the CLI
	// is signed in and the model answers. A failing probe removes the profile from the goal.
	Probe []string `json:"probe,omitempty"`
}

type Task struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Spec       string   `json:"spec"`
	Deps       []string `json:"deps,omitempty"`
	Accept     []string `json:"accept,omitempty"` // task acceptance, run by the Supervisor in the worktree
	Files      []string `json:"files,omitempty"`  // reserved while the task runs
	Tags       []string `json:"tags,omitempty"`   // kind of work: documents, web, code ... (skill choice)
	Skills     []string `json:"skills,omitempty"` // skills given to the last brief, name@hash
	Writer     string   `json:"writer,omitempty"` // profile; empty = chosen by record
	Reviewer   string   `json:"reviewer,omitempty"`
	State      string   `json:"state"`
	Attempts   int      `json:"attempts"`
	Total      int      `json:"total_attempts"` // across re-plans; a budget the lead cannot reset
	Gen        int      `json:"gen"`            // bumped on every change by the lead/owner; stale job results are dropped
	Reviews    int      `json:"reviews"`
	Branch     string   `json:"branch,omitempty"`
	ReviewHead string   `json:"review_head,omitempty"` // the commit the reviewer was given; merged exactly
	Worktree   string   `json:"worktree,omitempty"`
	Agent      string   `json:"agent,omitempty"` // current agent run name
	Feedback   string   `json:"feedback,omitempty"`
	Result     string   `json:"result,omitempty"`
	Merge      string   `json:"merge,omitempty"`    // merge commit on the goal branch
	Reverted   bool     `json:"reverted,omitempty"` // merge reverted by `revert`
	Note       string   `json:"note,omitempty"`     // why failed/blocked
	Proof      []string `json:"proof,omitempty"`    // evidence lines
	Updated    Time     `json:"updated"`
}

// AgentRun is one live (or finished) agent the Supervisor watches.
type AgentRun struct {
	Name      string         `json:"name"`
	Role      string         `json:"role"` // write, review, lead
	Task      string         `json:"task,omitempty"`
	Profile   string         `json:"profile"`
	Account   string         `json:"account,omitempty"`
	Cwd       string         `json:"cwd"`
	Answer    string         `json:"answer"` // result file the agent must write last
	Msg       string         `json:"msg"`    // id of the task message it must acknowledge
	State     string         `json:"state"`  // watchdog state
	Started   Time           `json:"started"`
	Acked     bool           `json:"acked"`
	Sent      bool           `json:"sent"`  // brief delivered (in argv or typed)
	Begun     bool           `json:"begun"` // launch check passed
	LastHash  string         `json:"last_hash,omitempty"`
	LastFP    string         `json:"last_worktree,omitempty"`   // worktree fingerprint: files changing is progress too
	AnsHash   string         `json:"answered_screen,omitempty"` // screen an answer was sent to
	AnsAt     Time           `json:"answered_at"`
	LastMove  Time           `json:"last_move"`  // normalized screen last changed
	IdleSince Time           `json:"idle_since"` // first idle-without-result observation
	Retries   map[string]int `json:"retries,omitempty"`
	Ended     bool           `json:"ended,omitempty"`
}

type Account struct {
	LimitedUntil Time `json:"limited_until"`
}

// Message is a numbered instruction to an agent; it counts as delivered only when acknowledged.
type Message struct {
	ID       string `json:"id"`
	To       string `json:"to"`
	Kind     string `json:"kind"`
	Body     string `json:"body"`
	Sent     Time   `json:"sent"`
	Acked    Time   `json:"acked"`
	Attempts int    `json:"attempts"`
}

// Intent is an action written down before it runs; after a crash it is reconciled, never blindly redone.
type Intent struct {
	ID     string `json:"id"`
	Action Action `json:"action"`
	At     Time   `json:"at"`
}

type Escalation struct {
	ID       string `json:"id"`
	Agent    string `json:"agent,omitempty"`
	Task     string `json:"task,omitempty"`
	Problem  string `json:"problem"`
	Screen   string `json:"screen,omitempty"`
	Answer   string `json:"answer,omitempty"`
	Resolved bool   `json:"resolved"`
	At       Time   `json:"at"`
}

type Goal struct {
	ID          string               `json:"id"`
	Text        string               `json:"text"`
	Project     string               `json:"project"`
	Base        string               `json:"base"`   // branch the goal started from
	Branch      string               `json:"branch"` // integration branch goal/<id>
	State       string               `json:"state"`
	Reason      string               `json:"reason,omitempty"`
	Accept      []string             `json:"accept"` // the goal's acceptance: the gate to done
	Tests       []string             `json:"tests"`  // the project's full tests: run after every merge
	Limits      Limits               `json:"limits"`
	Profiles    []Profile            `json:"profiles"`
	Lead        string               `json:"lead,omitempty"` // lead profile; empty = no Architect pane
	Created     Time                 `json:"created"`
	Seq         int                  `json:"seq"`
	Tasks       []*Task              `json:"tasks"`
	Agents      map[string]*AgentRun `json:"agents"`
	Accounts    map[string]*Account  `json:"accounts"`
	Messages    []*Message           `json:"messages"`
	Intents     []*Intent            `json:"intents"`
	Escalations []*Escalation        `json:"escalations"`
	Locks       map[string]string    `json:"locks"` // file -> task
	Assumptions []string             `json:"assumptions,omitempty"`
	Record      map[string]*Score    `json:"record"` // per profile: lessons used for agent choice
	Verify      *VerifyRun           `json:"verify,omitempty"`
	LastEvent   Time                 `json:"last_event"`
	Waiting     Time                 `json:"waiting_until"`
	Reverts     []string             `json:"reverts,omitempty"` // queued; run when the integration worktree is free
	RunnerGen   int                  `json:"runner_gen"`        // bumped by each runner that takes over
	Tokens      map[string]string    `json:"tokens"`            // token hash -> role (owner, lead)
	InboxOff    int64                `json:"inbox_offset"`      // commands applied from inbox.jsonl
	LeadSeen    int                  `json:"lead_seen"`         // event seq the lead was last nudged at
	LeadStarts  int                  `json:"lead_starts"`
	LeadNews    string               `json:"lead_news"` // what the lead was last told about
	Decider     []string             `json:"decider,omitempty"`
}

type Score struct {
	Done     int `json:"done"`
	Failed   int `json:"failed"`
	Rejected int `json:"rejected"`
	Errors   int `json:"errors"`
}

type VerifyRun struct {
	Head    string `json:"head"`
	Started Time   `json:"started"`
	Done    bool   `json:"done"`
	Ok      bool   `json:"ok"`
	Output  string `json:"output"`
}

type Event struct {
	Seq    int    `json:"seq"`
	At     Time   `json:"at"`
	Kind   string `json:"kind"`
	Agent  string `json:"agent,omitempty"`
	Task   string `json:"task,omitempty"`
	Detail string `json:"detail,omitempty"`
	Screen string `json:"screen,omitempty"`
}

// Ledger is the goal folder. All state is in goal.json (written atomically), history in events.jsonl.
type Ledger struct{ Dir string }

func LedgerDir(project, id string) string { return filepath.Join(project, ".herdr-goal", id) }

func (l Ledger) path(name string) string { return filepath.Join(l.Dir, name) }
func (l Ledger) AnswerDir() string       { return l.path("answers") }
func (l Ledger) AckDir() string          { return l.path("acks") }
func (l Ledger) WorktreeDir() string     { return l.path("wt") }

func (l Ledger) Init(g *Goal) error {
	for _, d := range []string{l.Dir, l.AnswerDir(), l.AckDir(), l.WorktreeDir(), l.path("logs")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(l.path("goal.json")); err == nil {
		return fmt.Errorf("goal %s already exists in %s", g.ID, l.Dir)
	}
	return l.Save(g)
}

// Save writes goal.json through a synced temp file and a rename: a kill at any point leaves the old or
// the new file, never half of one.
func (l Ledger) Save(g *Goal) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(l.path("goal.json"), data); err != nil {
		return err
	}
	return writeAtomic(l.path("plan.md"), []byte(RenderPlan(g)))
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func (l Ledger) Load() (*Goal, error) {
	data, err := os.ReadFile(l.path("goal.json"))
	if err != nil {
		return nil, fmt.Errorf("no goal ledger in %s: %w", l.Dir, err)
	}
	g := &Goal{}
	if err := json.Unmarshal(data, g); err != nil {
		return nil, fmt.Errorf("goal.json in %s is damaged: %w", l.Dir, err)
	}
	if g.Agents == nil {
		g.Agents = map[string]*AgentRun{}
	}
	if g.Accounts == nil {
		g.Accounts = map[string]*Account{}
	}
	if g.Locks == nil {
		g.Locks = map[string]string{}
	}
	if g.Record == nil {
		g.Record = map[string]*Score{}
	}
	if g.Tokens == nil {
		g.Tokens = map[string]string{}
	}
	return g, nil
}

// Append adds events to events.jsonl and fsyncs. Event numbers come from the goal, so a crash
// between Append and Save can only leave events whose numbers are reused; Events() keeps the last.
func (l Ledger) Append(evs ...Event) error {
	if len(evs) == 0 {
		return nil
	}
	f, err := os.OpenFile(l.path("events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	for _, e := range evs {
		b, _ := json.Marshal(e)
		buf.Write(b)
		buf.WriteByte('\n')
	}
	if _, err = f.Write(buf.Bytes()); err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// Events reads events.jsonl; a torn last line (kill mid-write) is skipped.
func (l Ledger) Events() ([]Event, error) {
	f, err := os.Open(l.path("events.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// A crash between Append and Save can leave events whose numbers the next runner reuses; the later
	// line wins. A damaged line anywhere (not only the last) is skipped and counted.
	var out []Event
	index := map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 8<<20)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) != nil || e.Seq == 0 {
			continue
		}
		if i, ok := index[e.Seq]; ok {
			out[i] = e
			continue
		}
		index[e.Seq] = len(out)
		out = append(out, e)
	}
	return out, sc.Err()
}

func (g *Goal) Task(id string) *Task {
	for _, t := range g.Tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (g *Goal) Profile(name string) *Profile {
	for i := range g.Profiles {
		if g.Profiles[i].Name == name {
			return &g.Profiles[i]
		}
	}
	return nil
}

func (g *Goal) score(profile string) *Score {
	s := g.Record[profile]
	if s == nil {
		s = &Score{}
		g.Record[profile] = s
	}
	return s
}

// RenderPlan is plan.md: a readable view of the ledger, regenerated on every save.
func RenderPlan(g *Goal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Goal %s\n\nState: **%s**", g.ID, g.State)
	if g.Reason != "" {
		fmt.Fprintf(&b, " (%s)", g.Reason)
	}
	fmt.Fprintf(&b, "\n\n%s\n\nProject: `%s`, branch `%s` from `%s`.\n\nAcceptance (run by the Supervisor):\n", g.Text, g.Project, g.Branch, g.Base)
	for _, a := range g.Accept {
		fmt.Fprintf(&b, "- `%s`\n", a)
	}
	b.WriteString("\n| Task | State | Deps | Writer | Attempts | Note |\n|---|---|---|---|---|---|\n")
	for _, t := range g.Tasks {
		fmt.Fprintf(&b, "| %s %s | %s | %s | %s | %d | %s |\n", t.ID, t.Title, t.State, strings.Join(t.Deps, ","), t.Writer, t.Attempts, oneLine(t.Note, 80))
	}
	if len(g.Assumptions) > 0 {
		b.WriteString("\nAssumptions:\n")
		for _, a := range g.Assumptions {
			fmt.Fprintf(&b, "- %s\n", a)
		}
	}
	var open []string
	for _, e := range g.Escalations {
		if !e.Resolved {
			open = append(open, fmt.Sprintf("- %s %s %s: %s", e.ID, e.Agent, e.Task, e.Problem))
		}
	}
	if len(open) > 0 {
		b.WriteString("\nOpen escalations:\n" + strings.Join(open, "\n") + "\n")
	}
	return b.String()
}

func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		s = s[:n] + "..."
	}
	return s
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Duration and Time marshal readably and keep zero values small.
type Duration time.Duration

func (d Duration) D() time.Duration { return time.Duration(d) }
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		var n int64
		if err2 := json.Unmarshal(b, &n); err2 != nil {
			return err
		}
		*d = Duration(n)
		return nil
	}
	v, err := time.ParseDuration(s)
	*d = Duration(v)
	return err
}

type Time struct{ time.Time }

func T(t time.Time) Time { return Time{t} }
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return json.Marshal(t.UTC().Format(time.RFC3339Nano))
}
func (t *Time) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	v, err := time.Parse(time.RFC3339Nano, s)
	t.Time = v
	return err
}
