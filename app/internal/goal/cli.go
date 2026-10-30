package goal

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const usage = `herdr-goal - give a goal, the engine finishes it (docs/GOAL-ENGINE.md)

  new     --project DIR --id ID --goal TEXT --accept CMD... [--test CMD...] --profiles FILE [--plan FILE]
          (--accept = the goal's acceptance, the gate to done; --test = the project's full tests,
           run on the goal branch after every merge, a failing merge is undone)
  run     LEDGER [--backend herdr|proc] [--interval 5s] [--lead-pane ID] [--once]
  keep    LEDGER [--every 15s] [run flags]   restarts a dead runner until the goal ends
  status  LEDGER                     events LEDGER
  env     [--project DIR] [--policy FILE] [--out FILE]   capability report (discover before planning)
  task    add|update|retry|cancel LEDGER --id ID [--title T] [--spec S|--spec-file F] [--deps a,b]
          [--accept CMD]... [--files f,g] [--writer P] [--reviewer P] [--feedback TEXT]
  assume  LEDGER TEXT                decide LEDGER TEXT
  answer  LEDGER ESCALATION [--keys k,k] [--text TEXT]
  pause|resume|stop LEDGER           revert LEDGER TASK
  ack     LEDGER MSG                 (agents: acknowledge a brief)

LEDGER is <project>/.herdr-goal/<id>. Exit 0 ok, 1 error, 2 usage.`

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ";") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func who() string {
	if r := os.Getenv("HERDR_GOAL_ROLE"); r != "" {
		return r
	}
	return "owner"
}

func fail(err error) int { fmt.Fprintln(os.Stderr, "ERROR:", err); return 1 }

// Main is the herdr-goal command.
func Main(args []string) int {
	if len(args) == 0 {
		fmt.Println(usage)
		return 2
	}
	cmd, rest := args[0], args[1:]
	if r := who(); (r == "write" || r == "review") && cmd != "ack" && cmd != "status" && cmd != "relay" {
		return fail(errors.New("delegates may only ack and read status; the lead changes the plan"))
	}
	switch cmd {
	case "relay":
		if len(rest) < 3 || rest[1] != "--" {
			return 2
		}
		return Relay(rest[0], rest[2:])
	case "new":
		return cmdNew(rest)
	case "env":
		fs := flag.NewFlagSet("env", flag.ContinueOnError)
		project := fs.String("project", ".", "")
		policy := fs.String("policy", DevicePolicyPath(), "")
		out := fs.String("out", "", "")
		if fs.Parse(rest) != nil {
			return 2
		}
		r := Discover(*project, *policy)
		b, _ := json.MarshalIndent(r, "", "  ")
		if *out != "" {
			if err := WriteReport(r, *out); err != nil {
				return fail(err)
			}
		}
		fmt.Println(string(b))
		return 0
	case "run":
		return cmdRun(rest)
	case "keep":
		return cmdKeep(rest)
	case "status":
		if len(rest) != 1 {
			return 2
		}
		s, err := Status(rest[0])
		if err != nil {
			return fail(err)
		}
		fmt.Print(s)
		return 0
	case "events":
		if len(rest) != 1 {
			return 2
		}
		evs, err := Ledger{Dir: rest[0]}.Events()
		if err != nil {
			return fail(err)
		}
		for _, e := range evs {
			fmt.Printf("%4d %s %-18s %-14s %-5s %s\n", e.Seq, e.At.Format("15:04:05"), e.Kind, e.Agent, e.Task, oneLine(e.Detail, 160))
		}
		return 0
	case "ack":
		if len(rest) != 2 {
			return 2
		}
		if err := os.WriteFile(filepath.Join(rest[0], "acks", filepath.Base(rest[1])), []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
			return fail(err)
		}
		return 0
	case "task":
		return cmdTask(rest)
	case "assume", "decide":
		if len(rest) < 2 {
			return 2
		}
		return enqueue(rest[0], cmd, map[string]string{"text": strings.Join(rest[1:], " ")})
	case "answer":
		fs := flag.NewFlagSet("answer", flag.ContinueOnError)
		keys := fs.String("keys", "", "keys, comma separated")
		text := fs.String("text", "", "text to type")
		if len(rest) < 2 || fs.Parse(rest[2:]) != nil {
			return 2
		}
		return enqueue(rest[0], "answer", map[string]any{"esc": rest[1], "keys": splitList(*keys), "text": *text})
	case "pause", "resume", "stop":
		if len(rest) != 1 {
			return 2
		}
		return enqueue(rest[0], cmd, map[string]string{})
	case "revert":
		if len(rest) != 2 {
			return 2
		}
		return enqueue(rest[0], "revert", map[string]string{"id": rest[1]})
	}
	fmt.Println(usage)
	return 2
}

func enqueue(dir, cmd string, args any) int {
	id, err := Enqueue(dir, cmd, who(), args)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("queued %s %s (the runner applies it on its next tick; see `herdr-goal events`)\n", id, cmd)
	return 0
}

func cmdTask(args []string) int {
	if len(args) < 2 {
		return 2
	}
	sub, dir := args[0], args[1]
	fs := flag.NewFlagSet("task", flag.ContinueOnError)
	var t Task
	var accept multi
	fs.StringVar(&t.ID, "id", "", "")
	fs.StringVar(&t.Title, "title", "", "")
	fs.StringVar(&t.Spec, "spec", "", "")
	specFile := fs.String("spec-file", "", "")
	deps := fs.String("deps", "", "")
	files := fs.String("files", "", "")
	fs.StringVar(&t.Writer, "writer", "", "")
	fs.StringVar(&t.Reviewer, "reviewer", "", "")
	tags := fs.String("tags", "", "")
	feedback := fs.String("feedback", "", "")
	fs.Var(&accept, "accept", "")
	if fs.Parse(args[2:]) != nil || t.ID == "" {
		return 2
	}
	if *specFile != "" {
		b, err := os.ReadFile(*specFile)
		if err != nil {
			return fail(err)
		}
		t.Spec = string(b)
	}
	t.Deps, t.Files, t.Accept, t.Tags = splitList(*deps), splitList(*files), accept, splitList(*tags)
	switch sub {
	case "add":
		return enqueue(dir, "task-add", t)
	case "update", "retry", "cancel":
		a := map[string]any{"id": t.ID, "spec": t.Spec, "writer": t.Writer, "feedback": *feedback}
		if len(accept) > 0 {
			a["accept"] = []string(accept)
		}
		return enqueue(dir, "task-"+sub, a)
	}
	return 2
}

// GoalConfig is the --profiles file: agent profiles, the lead, the menu decider and limits.
type GoalConfig struct {
	Profiles []Profile `json:"profiles"`
	Lead     string    `json:"lead"`
	Decider  []string  `json:"decider"`
	Limits   *Limits   `json:"limits"`
}

func cmdNew(args []string) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	project := fs.String("project", "", "")
	id := fs.String("id", "", "")
	text := fs.String("goal", "", "")
	profiles := fs.String("profiles", "", "")
	plan := fs.String("plan", "", "")
	var accept, tests multi
	fs.Var(&accept, "accept", "")
	fs.Var(&tests, "test", "")
	if fs.Parse(args) != nil || *project == "" || *id == "" || *profiles == "" {
		fmt.Println(usage)
		return 2
	}
	var cfg GoalConfig
	b, err := os.ReadFile(*profiles)
	if err != nil {
		return fail(err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return fail(fmt.Errorf("%s: %w", *profiles, err))
	}
	lim := DefaultLimits()
	if cfg.Limits != nil {
		mergeLimits(&lim, *cfg.Limits)
	}
	n := NewGoal{ID: *id, Text: *text, Project: *project, Accept: accept, Tests: tests, Profiles: cfg.Profiles, Lead: cfg.Lead, Limits: lim, Decider: cfg.Decider}
	if *plan != "" {
		var p struct {
			Tasks []Task `json:"tasks"`
		}
		b, err := os.ReadFile(*plan)
		if err != nil {
			return fail(err)
		}
		if err := json.Unmarshal(b, &p); err != nil {
			return fail(fmt.Errorf("%s: %w", *plan, err))
		}
		n.Tasks = p.Tasks
	}
	// Discover first: the measured budget caps the configured one (never raises it).
	capr := Discover(*project, DevicePolicyPath())
	if b := capr.Budget; b.MaxAgents < n.Limits.MaxAgents {
		n.Limits.MaxAgents = b.MaxAgents // 0 = defer agents until the machine has room (never clamped up)
		n.Limits.MaxProcs = min(n.Limits.MaxProcs, max(3, b.MaxProcs))
	}
	g, l, err := Create(n, RealProbes(), time.Now())
	if err != nil {
		return fail(err)
	}
	if err := WriteReport(capr, filepath.Join(l.Dir, "capabilities.json")); err != nil {
		return fail(err)
	}
	fmt.Printf("capabilities: %s; budget %d agents (%s); %d unknowns\n", filepath.Join(l.Dir, "capabilities.json"), g.Limits.MaxAgents, capr.Budget.Reason, len(capr.Unknowns))
	fmt.Printf("goal %s created: %s\nstate %s, %d tasks, branch %s\nrun it: herdr-goal run %s\n", g.ID, l.Dir, g.State, len(g.Tasks), g.Branch, l.Dir)
	return 0
}

// mergeLimits takes every non-zero field of x.
func mergeLimits(dst *Limits, x Limits) {
	var m map[string]any
	b, _ := json.Marshal(x)
	json.Unmarshal(b, &m)
	cur, _ := json.Marshal(dst)
	var c map[string]any
	json.Unmarshal(cur, &c)
	for k, v := range m {
		if v != "0s" && v != float64(0) && v != "" {
			c[k] = v
		}
	}
	b, _ = json.Marshal(c)
	json.Unmarshal(b, dst)
}

type runnerLock struct {
	Pid  int    `json:"pid"`
	Beat string `json:"beat"`
}

// takeLock makes the runner single with an OS lock on runner.lck held for the runner's whole life:
// no read-then-write race, no expiry while a busy runner lives, and released by the OS when it dies.
// runner.lock keeps pid + heartbeat for people and `status` only.
func takeLock(dir string) (func(), func(), error) {
	f, err := os.OpenFile(filepath.Join(dir, "runner.lck"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, nil, err
	}
	if err := lockFile(f); err != nil {
		f.Close()
		if errors.Is(err, errLocked) {
			return nil, nil, errors.New("another runner holds this goal (runner.lck is locked)")
		}
		return nil, nil, err
	}
	p := filepath.Join(dir, "runner.lock")
	beat := func() {
		b, _ := json.Marshal(runnerLock{Pid: os.Getpid(), Beat: time.Now().Format(time.RFC3339)})
		writeAtomic(p, b)
	}
	beat()
	return beat, func() { f.Close() }, nil
}

func cmdRun(args []string) int {
	if len(args) < 1 {
		return 2
	}
	dir := args[0]
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	backend := fs.String("backend", "proc", "herdr or proc")
	interval := fs.Duration("interval", 5*time.Second, "")
	leadPane := fs.String("lead-pane", "", "")
	once := fs.Bool("once", false, "")
	quiet := fs.Bool("quiet", false, "")
	if fs.Parse(args[1:]) != nil {
		return 2
	}
	dir, _ = filepath.Abs(dir)
	beat, release, err := takeLock(dir)
	if err != nil {
		return fail(err)
	}
	defer release()
	var b Backend
	switch *backend {
	case "herdr":
		b = &HerdrBackend{Exe: DefaultHerdrExe(), LeadPane: *leadPane, Dir: dir}
	case "proc":
		exe, err := os.Executable()
		if err != nil {
			return fail(err)
		}
		b = &ProcBackend{Dir: filepath.Join(dir, "proc"), Relay: []string{exe, "relay"}}
	default:
		return 2
	}
	e, err := Open(dir, b)
	if err != nil {
		return fail(err)
	}
	if !*quiet {
		e.Log = func(s string) { fmt.Println(s) }
	}
	e.Q = DefaultQuotas()
	if c, err := os.ReadFile(filepath.Join(dir, "capabilities.json")); err == nil {
		var r CapReport
		if json.Unmarshal(c, &r) == nil {
			e.Caps = &r
		}
	}
	if cat := os.Getenv("HERDR_SKILLS_CATALOG"); cat != "" {
		if sc, err := LoadSkillCatalog(cat); err == nil {
			e.Skills = sc
		} else {
			fmt.Fprintln(os.Stderr, "skills catalog:", err)
		}
	}
	if len(e.G.Decider) > 0 {
		e.Decide = CommandDecider(e.G.Decider)
	}
	e.G.RunnerGen++
	e.Gen = e.G.RunnerGen
	if err := e.Recover(); err != nil {
		return fail(err)
	}
	e.event("runner-start", "", "", "pid "+strconv.Itoa(os.Getpid())+" backend "+*backend)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	for {
		beat()
		if err := e.Tick(); err != nil {
			fmt.Fprintln(os.Stderr, "tick:", err)
			if errors.Is(err, ErrSuperseded) {
				return 1
			}
		}
		switch e.G.State {
		case GoalDone, GoalFailed, GoalStopped, GoalNeedsOwner:
			e.flush()
			fmt.Printf("goal %s: %s - %s\n", e.G.ID, e.G.State, e.G.Reason)
			os.Remove(filepath.Join(dir, "runner.lock"))
			if e.G.State == GoalDone {
				return 0
			}
			return 1
		}
		if *once {
			return 0
		}
		select {
		case <-sig:
			e.event("runner-stop", "", "", "interrupted; agents keep running, a new runner takes over")
			e.flush()
			os.Remove(filepath.Join(dir, "runner.lock"))
			return 1
		case <-time.After(*interval):
		}
	}
}

// runnerAlive reports whether a runner holds the goal's OS lock (pid reuse cannot fool it).
func runnerAlive(dir string) bool {
	f, err := os.OpenFile(filepath.Join(dir, "runner.lck"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return false
	}
	defer f.Close()
	return errors.Is(lockFile(f), errLocked)
}

func terminal(state string) bool {
	switch state {
	case GoalDone, GoalFailed, GoalStopped, GoalNeedsOwner:
		return true
	}
	return false
}

// cmdKeep is the runner's keeper: whenever no runner holds the goal, it starts a detached one (output
// in logs/runner.log). It exits when the goal ends. Killing the keeper leaves the runner running; on
// Windows a scheduled task that starts `keep` at logon makes the pair self-healing.
func cmdKeep(args []string) int {
	if len(args) < 1 {
		return 2
	}
	dir, _ := filepath.Abs(args[0])
	fs := flag.NewFlagSet("keep", flag.ContinueOnError)
	every := fs.Duration("every", 15*time.Second, "")
	var runArgs []string
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--every" && i+1 < len(rest) {
			fs.Parse(rest[i : i+2])
			i++
			continue
		}
		runArgs = append(runArgs, rest[i])
	}
	exe, err := os.Executable()
	if err != nil {
		return fail(err)
	}
	for {
		g, err := Ledger{Dir: dir}.Load()
		if err != nil {
			return fail(err)
		}
		if terminal(g.State) {
			fmt.Printf("goal %s: %s - %s\n", g.ID, g.State, g.Reason)
			if g.State == GoalDone {
				return 0
			}
			return 1
		}
		if !runnerAlive(dir) {
			logf, err := os.OpenFile(filepath.Join(dir, "logs", "runner.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return fail(err)
			}
			cmd := exec.Command(exe, append([]string{"run", dir, "--quiet"}, runArgs...)...)
			cmd.Stdout, cmd.Stderr = logf, logf
			detach(cmd)
			if err := cmd.Start(); err != nil {
				logf.Close()
				return fail(err)
			}
			fmt.Printf("%s keep: started runner %d\n", time.Now().Format("15:04:05"), cmd.Process.Pid)
			go cmd.Wait()
			logf.Close()
		}
		time.Sleep(*every)
	}
}

// DevicePolicyPath is the owner's per-device policy file (profile company_nasca / personal, packages).
func DevicePolicyPath() string {
	if v := os.Getenv("HERDR_DEVICE_POLICY"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "herdr", "device-policy.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "herdr", "device-policy.json")
}
