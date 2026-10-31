package goal

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Command is one line of inbox.jsonl. The lead, the owner and the CLI change a running goal only through
// it; the runner applies each line once, in order, and stores the offset with the state in one write.
type Command struct {
	ID   string          `json:"id"`
	Cmd  string          `json:"cmd"`
	By   string          `json:"by"`
	At   Time            `json:"at"`
	Args json.RawMessage `json:"args,omitempty"`
	// Key names the sender's key (not secret); Mac is HMAC-SHA256 of the command under that key. The key
	// itself is never written to the inbox, the ledger, a brief or an event. By is only a claim.
	Key string `json:"key,omitempty"`
	Mac string `json:"mac,omitempty"`
}

func commandMAC(secret string, c Command) string {
	m := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(m, "%s\n%s\n%s\n%s\n", c.ID, c.Cmd, c.At.UTC().Format(time.RFC3339Nano), c.Args)
	return hex.EncodeToString(m.Sum(nil))
}

// keyID is the public name of a key: enough to find its file, useless to forge with.
func keyID(secret string) string { return tokenHash(secret)[:16] }

// keyFile is where the consumer keeps a key's secret: keys/ is 0700, each file 0600.
func keyFile(dir, role, id string) string {
	if role == "owner" {
		return filepath.Join(dir, "keys", "owner")
	}
	return filepath.Join(dir, "keys", role+"-"+id)
}

// newToken makes a random capability token.
func newToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func tokenHash(t string) string {
	h := sha256.Sum256([]byte("herdr-goal:" + t))
	return hex.EncodeToString(h[:])
}

// senderKey is the key this process may sign with: the lead's own key file (HERDR_GOAL_KEY_FILE, given
// only to the lead), else the owner key file. Delegates have none.
func senderKey(dir string) string {
	path := os.Getenv("HERDR_GOAL_KEY_FILE")
	if path == "" {
		if os.Getenv("HERDR_GOAL_ROLE") != "" {
			return ""
		}
		path = filepath.Join(dir, "keys", "owner")
	}
	b, _ := os.ReadFile(path)
	return strings.TrimSpace(string(b))
}

// Enqueue appends a command. Safe while a runner works: appends of one short line are atomic enough on
// both systems, and a torn last line is retried on the next read, never half-applied.
func Enqueue(dir, cmd, by string, args any) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, "goal.json")); err != nil {
		return "", fmt.Errorf("no goal ledger in %s", dir)
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("c%d", time.Now().UnixNano())
	c := Command{ID: id, Cmd: cmd, By: by, At: T(time.Now()), Args: raw}
	if secret := senderKey(dir); secret != "" {
		c.Key, c.Mac = keyID(secret), commandMAC(secret, c)
	}
	line, _ := json.Marshal(c)
	f, err := os.OpenFile(filepath.Join(dir, "inbox.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return "", err
	}
	return id, f.Sync()
}

// verify checks a command's signature against the key the ledger knows by that id. A replaced lead's key
// is removed from the ledger, so its old commands no longer verify.
func (e *Engine) verifyCommand(c Command) string {
	role := e.G.Tokens[c.Key]
	if role == "" || c.Mac == "" {
		return ""
	}
	b, err := os.ReadFile(keyFile(e.L.Dir, role, c.Key))
	if err != nil {
		return ""
	}
	secret := strings.TrimSpace(string(b))
	if keyID(secret) != c.Key || !hmac.Equal([]byte(commandMAC(secret, c)), []byte(c.Mac)) {
		return ""
	}
	return role
}

// seen records a command id; true if it was applied before (a replayed line).
func (g *Goal) seen(id string) bool {
	for _, x := range g.SeenCmds {
		if x == id {
			return true
		}
	}
	g.SeenCmds = append(g.SeenCmds, id)
	if len(g.SeenCmds) > 2000 {
		g.SeenCmds = g.SeenCmds[len(g.SeenCmds)-2000:]
	}
	return false
}

func (e *Engine) applyInbox() error {
	f, err := os.Open(e.L.path("inbox.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(e.G.InboxOff, io.SeekStart); err != nil {
		return err
	}
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return nil // EOF, or a torn line still being written: read again next tick
		}
		e.G.InboxOff += int64(len(line))
		var c Command
		if jerr := json.Unmarshal(bytes.TrimSpace(line), &c); jerr != nil {
			e.event("command-rejected", "", "", "unreadable inbox line: "+jerr.Error())
			continue
		}
		claimed := c.By
		c.By = e.verifyCommand(c) // the role the signature proves; "" when none
		if c.By == "" {
			e.event("command-rejected", "", "", fmt.Sprintf("%s %s claimed by %q: no valid signature", c.ID, c.Cmd, claimed))
			continue
		}
		if e.G.seen(c.ID) {
			e.event("command-rejected", "", "", fmt.Sprintf("%s %s by %s: replayed command id", c.ID, c.Cmd, c.By))
			continue
		}
		if err := e.apply(c); err != nil {
			e.event("command-rejected", "", "", fmt.Sprintf("%s %s by %s: %v", c.ID, c.Cmd, c.By, err))
		} else {
			e.event("command", "", "", fmt.Sprintf("%s %s by %s %s", c.ID, c.Cmd, c.By, oneLine(string(c.Args), 300)))
		}
	}
}

var idRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,31}$`)

// allowed is the command authority at the consumer: who may do what, whatever the sender wrote.
var allowed = map[string][]string{
	"owner": {"*"},
	"plan":  {"task-add"},
	"lead":  {"task-add", "task-update", "task-retry", "task-cancel", "assume", "decide", "answer", "revert", "research"},
}

func authorized(by, cmd string) bool {
	for _, c := range allowed[by] {
		if c == "*" || c == cmd {
			return true
		}
	}
	return false
}

// cycleWith reports a cycle if task id took deps.
func (e *Engine) cycleWith(id string, deps []string) string {
	seen := map[string]bool{}
	var walk func(n string, trail []string) string
	walk = func(n string, trail []string) string {
		if n == id {
			return strings.Join(append(trail, n), " -> ")
		}
		if seen[n] {
			return ""
		}
		seen[n] = true
		t := e.G.Task(n)
		if t == nil {
			return ""
		}
		for _, d := range t.Deps {
			if c := walk(d, append(trail, n)); c != "" {
				return c
			}
		}
		return ""
	}
	for _, d := range deps {
		if c := walk(d, []string{id}); c != "" {
			return c
		}
	}
	return ""
}

func (e *Engine) apply(c Command) error {
	g := e.G
	if !authorized(c.By, c.Cmd) {
		return fmt.Errorf("%q may not run %s", c.By, c.Cmd)
	}
	switch c.Cmd {
	case "task-add":
		var t Task
		if err := json.Unmarshal(c.Args, &t); err != nil {
			return err
		}
		if !idRe.MatchString(t.ID) {
			return fmt.Errorf("task id %q: letters, digits, . _ - only", t.ID)
		}
		if g.Task(t.ID) != nil {
			return fmt.Errorf("task %s exists", t.ID)
		}
		if g.Limits.MaxTasks > 0 && len(g.Tasks) >= g.Limits.MaxTasks {
			return fmt.Errorf("task limit %d reached", g.Limits.MaxTasks)
		}
		if t.Title == "" || t.Spec == "" {
			return errors.New("a task needs a title and a spec")
		}
		for _, d := range t.Deps {
			if g.Task(d) == nil {
				return fmt.Errorf("unknown dependency %s", d)
			}
		}
		for _, p := range []string{t.Writer, t.Reviewer} {
			if p != "" && g.Profile(p) == nil {
				return fmt.Errorf("unknown profile %s", p)
			}
		}
		t.State, t.Updated = TaskPending, T(e.Now())
		t.Attempts, t.Reviews, t.Agent, t.Merge, t.Proof = 0, 0, "", "", nil
		g.Tasks = append(g.Tasks, &t)
	case "task-cancel", "task-retry", "task-update":
		var a struct {
			ID       string    `json:"id"`
			Feedback string    `json:"feedback"`
			Spec     string    `json:"spec"`
			Writer   string    `json:"writer"`
			Accept   []string  `json:"accept"`
			Deps     *[]string `json:"deps"`
		}
		if err := json.Unmarshal(c.Args, &a); err != nil {
			return err
		}
		// Validate everything first: a rejected command has no effect at all.
		t := g.Task(a.ID)
		if t == nil {
			return fmt.Errorf("no task %s", a.ID)
		}
		if t.State == TaskDone || t.State == TaskMerging {
			return fmt.Errorf("task %s is %s; add a new task instead", t.ID, t.State)
		}
		if c.Cmd == "task-retry" && t.Total >= 2 && !g.researchedSince(t.ID, t.LastFail.Time) {
			return fmt.Errorf("task %s failed %d times: record research for it first (`herdr-goal research --task %s ...`: what the failure is, sources checked, the fix chosen)", t.ID, t.Total, t.ID)
		}
		if c.Cmd == "task-retry" && strings.HasPrefix(t.Note, "BUDGET:") && c.By != "owner" {
			return fmt.Errorf("task %s used its whole failure budget; split it into new tasks instead", t.ID)
		}
		if a.Writer != "" && g.Profile(a.Writer) == nil {
			return fmt.Errorf("unknown profile %s", a.Writer)
		}
		if a.Deps != nil {
			for _, d := range *a.Deps {
				if g.Task(d) == nil {
					return fmt.Errorf("unknown dependency %s", d)
				}
			}
			if cyc := e.cycleWith(t.ID, *a.Deps); cyc != "" {
				return fmt.Errorf("dependency cycle: %s", cyc)
			}
		}
		if t.Agent != "" && g.Agents[t.Agent] != nil && !g.Agents[t.Agent].Ended {
			if !e.endAgent(g.Agents[t.Agent], c.Cmd+" by "+c.By) {
				return fmt.Errorf("the agent of %s did not confirm its stop; nothing changed, send the command again", t.ID)
			}
		}
		t.Agent = ""
		e.cancelJobs(":" + t.ID + ":")
		t.Gen++ // results of checks or reviews started before this change are now stale
		switch c.Cmd {
		case "task-cancel":
			t.State = TaskCancelled
			e.unlock(t)
		default:
			if a.Spec != "" {
				t.Spec = a.Spec
			}
			if a.Writer != "" {
				t.Writer = a.Writer
			}
			if a.Accept != nil {
				t.Accept = a.Accept
			}
			if a.Deps != nil {
				t.Deps = *a.Deps
			}
			if a.Feedback != "" {
				t.Feedback = a.Feedback
			}
			t.ReviewHead = ""
			if c.Cmd == "task-retry" || t.State == TaskFailed || t.State == TaskBlocked || t.State == TaskCancelled {
				t.State, t.Attempts, t.Reviews, t.Note = TaskPending, 0, 0, ""
			} else if t.State == TaskChecking || t.State == TaskReview {
				t.State = TaskRunning // changed work is checked and reviewed again
			}
		}
		t.Updated = T(e.Now())
	case "research":
		var r ResearchRecord
		if err := json.Unmarshal(c.Args, &r); err != nil {
			return err
		}
		if strings.TrimSpace(r.Source) == "" || strings.TrimSpace(r.Why) == "" {
			return errors.New("research needs a source (URL, document, or \"offline\") and why it applies")
		}
		r.By, r.At = c.By, T(e.Now())
		g.Research = append(g.Research, r)
		e.event("research", c.By, r.Task, r.Source+" "+r.Version+": "+r.Why)
	case "assume", "decide":
		var a struct{ Text string }
		if err := json.Unmarshal(c.Args, &a); err != nil || a.Text == "" {
			return errors.New("needs text")
		}
		if c.Cmd == "assume" {
			g.Assumptions = append(g.Assumptions, a.Text)
		}
		e.event(c.Cmd, c.By, "", a.Text)
	case "answer":
		var a struct {
			Esc  string   `json:"esc"`
			Keys []string `json:"keys"`
			Text string   `json:"text"`
		}
		if err := json.Unmarshal(c.Args, &a); err != nil {
			return err
		}
		var esc *Escalation
		for _, x := range g.Escalations {
			if x.ID == a.Esc {
				esc = x
			}
		}
		if esc == nil || esc.Resolved {
			return fmt.Errorf("no open escalation %s", a.Esc)
		}
		ag := g.Agents[esc.Agent]
		if strings.HasPrefix(esc.Problem, "OWNER:") && c.By != "owner" {
			return errors.New("this escalation is the owner's (secret, sign-in or outside trust)")
		}
		esc.Answer, esc.Resolved = strings.TrimSpace(a.Text+" "+strings.Join(a.Keys, " ")), true
		if ag != nil && !ag.Ended {
			if len(a.Keys) > 0 {
				e.keys(ag, a.Keys, "escalation "+esc.ID+" answered by "+c.By, "")
			}
			if a.Text != "" {
				e.send(ag, a.Text)
			}
			if ag.State == "needs-owner" {
				ag.State = "working"
				if t := g.Task(ag.Task); t != nil && t.State == TaskBlocked {
					t.State = TaskRunning
				}
			}
		}
	case "pause", "resume", "stop":
		switch c.Cmd {
		case "pause":
			e.cancelJobs("")
			e.stopAll("paused by " + c.By)
			for _, t := range g.Tasks {
				if t.State == TaskRunning || t.State == TaskReview {
					t.Agent = ""
				}
			}
			e.setState(GoalPaused, "paused by "+c.By)
		case "resume":
			if g.State != GoalPaused && g.State != GoalNeedsOwner && g.State != GoalStopped {
				return fmt.Errorf("goal is %s", g.State)
			}
			g.LastEvent = T(e.Now())
			e.setState(GoalRunning, "resumed by "+c.By)
		case "stop":
			e.cancelJobs("")
			e.stopAll("stopped by " + c.By)
			e.setState(GoalStopped, "stopped by "+c.By)
		}
	case "revert":
		var a struct{ ID string }
		if err := json.Unmarshal(c.Args, &a); err != nil {
			return err
		}
		t := g.Task(a.ID)
		if t == nil || t.State != TaskDone || t.Reverted {
			return fmt.Errorf("task %s is not a merged task", a.ID)
		}
		for _, q := range g.Reverts {
			if q == t.ID {
				return fmt.Errorf("revert of %s already queued", t.ID)
			}
		}
		g.Reverts = append(g.Reverts, t.ID) // applied by the scheduler when no merge or check holds the worktree
		if g.State == GoalDone || g.State == GoalVerifying {
			e.setState(GoalRunning, "revert of "+t.ID+" requested; acceptance runs again after it")
		}
	default:
		return fmt.Errorf("unknown command %q", c.Cmd)
	}
	return nil
}

// NewGoal is everything `herdr-goal new` needs.
type NewGoal struct {
	ID        string
	Text      string
	Project   string
	Accept    []string
	Tests     []string
	Profiles  []Profile
	Lead      string
	Limits    Limits
	Tasks     []Task
	Decider   []string
	Research  []ResearchRecord
	Protected []string
}

// Create checks readiness and writes a new ledger. It refuses rather than guess.
func Create(n NewGoal, probe Probes, now time.Time) (*Goal, Ledger, error) {
	if !idRe.MatchString(n.ID) {
		return nil, Ledger{}, fmt.Errorf("goal id %q: letters, digits, . _ - only", n.ID)
	}
	if strings.TrimSpace(n.Text) == "" || len(n.Accept) == 0 {
		return nil, Ledger{}, errors.New("a goal needs a text and at least one acceptance command")
	}
	top, branch, err := CheckRepo(n.Project)
	if err != nil {
		return nil, Ledger{}, err
	}
	if st, _ := git(top, "status", "--porcelain", "--untracked-files=no"); st != "" {
		return nil, Ledger{}, fmt.Errorf("%s has uncommitted changes; commit or stash them first so the goal starts from a known state", top)
	}
	profiles, notes := ProbeProfiles(n.Profiles, 90*time.Second)
	n.Profiles = profiles
	if err := Preflight(n.Profiles, n.Lead, probe); err != nil {
		return nil, Ledger{}, err
	}
	if _, err := probe.LookPath("git"); err != nil {
		return nil, Ledger{}, errors.New("git is not on PATH")
	}
	lim := n.Limits
	if lim.MaxAttempts == 0 {
		lim = DefaultLimits()
	}
	g := &Goal{ID: n.ID, Text: n.Text, Project: top, Base: branch, Branch: "goal/" + n.ID, State: GoalPlanning,
		Accept: n.Accept, Tests: n.Tests, Limits: lim, Profiles: n.Profiles, Lead: n.Lead, Created: T(now), LastEvent: T(now),
		Agents: map[string]*AgentRun{}, Accounts: map[string]*Account{}, Locks: map[string]string{},
		Record: map[string]*Score{}, Decider: n.Decider}
	l := Ledger{Dir: LedgerDir(top, n.ID)}
	owner := newToken()
	g.Tokens = map[string]string{keyID(owner): "owner"}
	if err := l.Init(g); err != nil {
		return nil, l, err
	}
	// The owner key is readable by the owner's account only. Agents run as the same user could still
	// read it: real isolation needs agents under another OS account (docs/GOAL-ENGINE.md).
	os.MkdirAll(filepath.Join(l.Dir, "keys"), 0o700)
	if err := os.WriteFile(filepath.Join(l.Dir, "keys", "owner"), []byte(owner), 0o600); err != nil {
		return nil, l, err
	}
	w := Workspace{Project: top, Ledger: l}
	if err := w.Setup(g); err != nil {
		return nil, l, err
	}
	e := &Engine{L: l, W: w, G: g, Now: func() time.Time { return now }}
	e.event("goal-created", "", "", n.Text)
	for _, note := range notes {
		e.event("profile-probe", "", "", note)
	}
	for _, t := range n.Tasks {
		raw, _ := json.Marshal(t)
		if err := e.apply(Command{ID: "plan", Cmd: "task-add", By: "plan", Args: raw}); err != nil {
			return nil, l, fmt.Errorf("plan task %s: %w", t.ID, err)
		}
		e.event("task-added", "", t.ID, t.Title)
	}
	for _, r := range n.Research {
		r.By, r.At = "owner", T(now)
		g.Research = append(g.Research, r)
	}
	g.Protected = n.Protected
	if len(g.Tasks) > 0 {
		if !g.planResearched() {
			return nil, l, errors.New("a plan needs research first: give --research FILE (sources, versions, licenses, why), or a record with source \"offline\" and the reason")
		}
		e.setState(GoalRunning, "plan given at creation, with research")
	}
	return g, l, e.flush()
}

// ProbeProfiles runs each profile's Probe (a tiny real request) with a timeout and returns the
// profiles that answered plus why the others were dropped. No probe = kept, marked "not probed".
func ProbeProfiles(ps []Profile, timeout time.Duration) ([]Profile, []string) {
	var ok []Profile
	var notes []string
	for _, p := range ps {
		if len(p.Probe) == 0 {
			ok = append(ok, p)
			notes = append(notes, p.Name+": not probed (no probe command)")
			continue
		}
		out, err, timedOut := runBounded(context.Background(), "", timeout, 8<<10, p.Probe[0], p.Probe[1:]...)
		if err != nil || timedOut {
			notes = append(notes, fmt.Sprintf("%s: DROPPED, probe failed (%v): %s", p.Name, err, oneLine(out, 160)))
			continue
		}
		ok = append(ok, p)
		notes = append(notes, p.Name+": probe answered")
	}
	return ok, notes
}

// Preflight: at least one writer and one reviewer of a different vendor must be startable.
func Preflight(ps []Profile, lead string, probe Probes) error {
	var writers, reviewers []Profile
	var missing []string
	for _, p := range ps {
		if len(p.Command) == 0 {
			return fmt.Errorf("profile %s has no command", p.Name)
		}
		if _, err := probe.LookPath(p.Command[0]); err != nil {
			missing = append(missing, p.Name+" ("+p.Command[0]+")")
			continue
		}
		if has(p.Roles, "write") {
			writers = append(writers, p)
		}
		if has(p.Roles, "review") {
			reviewers = append(reviewers, p)
		}
	}
	if len(writers) == 0 {
		return fmt.Errorf("no writer profile can start; missing: %s", strings.Join(missing, ", "))
	}
	for _, w := range writers {
		for _, r := range reviewers {
			if r.Kind != w.Kind {
				if lead != "" {
					found := false
					for _, p := range ps {
						if p.Name == lead {
							found = true
						}
					}
					if !found {
						return fmt.Errorf("lead profile %s is not defined", lead)
					}
				}
				return nil
			}
		}
	}
	return fmt.Errorf("no reviewer of a different vendor than the writers can start (independent review is required); missing: %s", strings.Join(missing, ", "))
}

// Status is the readable summary for `herdr-goal status`.
func Status(dir string) (string, error) {
	l := Ledger{Dir: dir}
	g, err := l.Load()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(RenderPlan(g))
	b.WriteString("\nAgents:\n")
	for _, n := range sortedKeys(g.Agents) {
		a := g.Agents[n]
		if !a.Ended {
			fmt.Fprintf(&b, "- %s %s task=%s profile=%s state=%s acked=%v\n", a.Name, a.Role, a.Task, a.Profile, a.State, a.Acked)
		}
	}
	evs, _ := l.Events()
	if len(evs) > 15 {
		evs = evs[len(evs)-15:]
	}
	b.WriteString("\nRecent events:\n")
	for _, ev := range evs {
		fmt.Fprintf(&b, "%4d %s %-16s %-12s %-5s %s\n", ev.Seq, ev.At.Format("15:04:05"), ev.Kind, ev.Agent, ev.Task, oneLine(ev.Detail, 120))
	}
	return b.String(), nil
}

// planResearched: at least one research record for the plan (Task == "").
func (g *Goal) planResearched() bool {
	for _, r := range g.Research {
		if r.Task == "" {
			return true
		}
	}
	return false
}

// researchedSince: a research record for the task made after its last failure.
func (g *Goal) researchedSince(task string, since time.Time) bool {
	for _, r := range g.Research {
		if r.Task == task && !r.At.Before(since) {
			return true
		}
	}
	return false
}
