package goal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Agent statuses a backend reports.
const (
	StatusWorking = "working"
	StatusIdle    = "idle"
	StatusBlocked = "blocked"
	StatusDone    = "done"
	StatusExited  = "exited" // process gone
	StatusUnknown = "unknown"
)

type Observation struct {
	Exists bool
	Status string
	Screen string
	Err    error // the backend could not be asked; never read as "dead"
}

type StartSpec struct {
	Name   string
	Cwd    string
	Argv   []string
	Env    map[string]string
	Prompt string // first message, sent after start
}

// Backend runs agents. Herdr panes on the owner's PC; detached processes in tests and headless runs.
type Backend interface {
	Start(s StartSpec) error
	Observe(name string) Observation
	Send(name, text string) error // text then Enter
	Keys(name string, keys ...string) error
	Stop(name string) error
	Live() ([]string, error)
}

// ---------- Herdr panes ----------

// HerdrBackend drives agents through the herdr CLI, one pane per agent, split to the right of the lead
// pane. Pane ids are stored in <dir>/panes.json so a restarted runner finds its agents again.
type HerdrBackend struct {
	Exe      string // absolute herdr path (LESSONS #1)
	LeadPane string // pane to split from; HERDR_PANE_ID when empty
	Dir      string
	Timeout  time.Duration
}

func DefaultHerdrExe() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Herdr", "bin", "herdr.exe")
	}
	if p, err := exec.LookPath("herdr"); err == nil {
		return p
	}
	return "herdr"
}

func (h *HerdrBackend) run(args ...string) (string, error) {
	t := h.Timeout
	if t == 0 {
		t = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()
	cmd := exec.CommandContext(ctx, h.Exe, args...)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("herdr %s: %v: %s", strings.Join(args[:min(2, len(args))], " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func (h *HerdrBackend) panes() map[string]string {
	m, err := h.loadPanes()
	if err != nil {
		return map[string]string{}
	}
	return m
}

// loadPanes fails loudly on a damaged panes.json: reading it as empty would orphan live panes.
func (h *HerdrBackend) loadPanes() (map[string]string, error) {
	m := map[string]string{}
	b, err := os.ReadFile(filepath.Join(h.Dir, "panes.json"))
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("panes.json is damaged (%v); not starting agents until it is repaired", err)
	}
	return m, nil
}

func (h *HerdrBackend) savePanes(m map[string]string) error {
	b, _ := json.MarshalIndent(m, "", "  ")
	return writeAtomic(filepath.Join(h.Dir, "panes.json"), b)
}

func (h *HerdrBackend) Start(s StartSpec) error {
	if _, err := h.loadPanes(); err != nil {
		return err
	}
	for _, a := range s.Argv {
		if strings.ContainsAny(a, "\r\n\x00\x1b") {
			return fmt.Errorf("argument for %s holds a newline or control character; refused (it would be typed into the pane)", s.Name)
		}
	}
	lead := h.LeadPane
	if lead == "" {
		lead = os.Getenv("HERDR_PANE_ID")
	}
	if lead == "" {
		return errors.New("no lead pane to split from (set HERDR_PANE_ID or --lead-pane)")
	}
	out, err := h.run("pane", "split", lead, "--direction", "right", "--cwd", s.Cwd, "--no-focus")
	if err != nil {
		return err
	}
	var r struct {
		Result struct {
			Pane struct {
				ID string `json:"pane_id"`
			} `json:"pane"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil || r.Result.Pane.ID == "" {
		return fmt.Errorf("herdr pane split gave no pane id: %s", oneLine(out, 200))
	}
	m := h.panes()
	m[s.Name] = r.Result.Pane.ID
	if err := h.savePanes(m); err != nil {
		return err
	}
	// `pane run` is the start path that works (LESSONS #99); `agent start` fails with agent_pane_busy.
	if _, err := h.run("pane", "run", r.Result.Pane.ID, shellLine(s.Env, s.Argv)); err != nil {
		return err
	}
	return nil
}

func (h *HerdrBackend) pane(name string) (string, error) {
	p := h.panes()[name]
	if p == "" {
		return "", fmt.Errorf("no pane recorded for agent %s", name)
	}
	return p, nil
}

func (h *HerdrBackend) Observe(name string) Observation {
	p, err := h.pane(name)
	if err != nil {
		return Observation{Exists: false, Status: StatusExited}
	}
	out, err := h.run("pane", "get", p)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "pane_not_found") {
			return Observation{Exists: false, Status: StatusExited}
		}
		return Observation{Err: err}
	}
	var r struct {
		Result struct {
			Pane struct {
				Status string `json:"agent_status"`
				Agent  string `json:"agent"`
			} `json:"pane"`
		} `json:"result"`
	}
	json.Unmarshal([]byte(out), &r)
	screen, err := h.run("pane", "read", p, "--source", "visible")
	if err != nil {
		return Observation{Err: err}
	}
	st := r.Result.Pane.Status
	if r.Result.Pane.Agent == "" && st == "" {
		st = StatusExited // shell left in the pane: the agent CLI ended (LESSONS #91)
	}
	if st == "" {
		st = StatusUnknown
	}
	return Observation{Exists: true, Status: st, Screen: screen}
}

func (h *HerdrBackend) Send(name, text string) error {
	p, err := h.pane(name)
	if err != nil {
		return err
	}
	if _, err := h.run("pane", "send-text", p, text); err != nil {
		return err
	}
	_, err = h.run("pane", "send-keys", p, "Enter")
	return err
}

func (h *HerdrBackend) Keys(name string, keys ...string) error {
	p, err := h.pane(name)
	if err != nil {
		return err
	}
	_, err = h.run(append([]string{"pane", "send-keys", p}, keys...)...)
	return err
}

func (h *HerdrBackend) Stop(name string) error {
	p, err := h.pane(name)
	if err != nil {
		return nil
	}
	if _, err := h.run("pane", "close", p); err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	m := h.panes()
	delete(m, name)
	return h.savePanes(m)
}

func (h *HerdrBackend) Live() ([]string, error) {
	var live []string
	for _, name := range sortedKeys(h.panes()) {
		if o := h.Observe(name); o.Err != nil {
			return nil, o.Err
		} else if o.Exists && o.Status != StatusExited {
			live = append(live, name)
		}
	}
	return live, nil
}

func shellLine(env map[string]string, argv []string) string {
	q := func(s string) string {
		if s != "" && !strings.ContainsAny(s, " \t\"'$`&|;<>()") {
			return s
		}
		if runtime.GOOS == "windows" {
			return "'" + strings.ReplaceAll(s, "'", "''") + "'"
		}
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
	var parts []string
	for _, k := range sortedKeys(env) {
		if runtime.GOOS == "windows" {
			parts = append(parts, fmt.Sprintf("$env:%s=%s;", k, q(env[k])))
		} else {
			parts = append(parts, k+"="+q(env[k]))
		}
	}
	if runtime.GOOS == "windows" && len(argv) > 0 {
		parts = append(parts, "&")
	}
	for _, a := range argv {
		parts = append(parts, q(a))
	}
	return strings.Join(parts, " ")
}

// ---------- detached processes ----------

// ProcBackend runs each agent as a detached `herdr-goal relay` process: the relay feeds <dir>/<name>/inbox
// into the agent's stdin and appends its output to screen.log, so agents survive a runner restart and the
// runner reads them like panes. Used for tests, headless CLIs and machines without Herdr.
type ProcBackend struct {
	Dir   string
	Relay []string // argv that starts a relay: <relay...> <agentdir> -- <argv...>
}

func (p *ProcBackend) adir(name string) string { return filepath.Join(p.Dir, name) }

func (p *ProcBackend) Start(s StartSpec) error {
	d := p.adir(s.Name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	for _, f := range []string{"inbox", "screen.log", "exit", "heartbeat", "pid"} {
		os.Remove(filepath.Join(d, f))
	}
	if err := os.WriteFile(filepath.Join(d, "inbox"), nil, 0o644); err != nil {
		return err
	}
	args := append(append(append([]string{}, p.Relay[1:]...), d, "--"), s.Argv...)
	cmd := exec.Command(p.Relay[0], args...)
	cmd.Dir = s.Cwd
	cmd.Env = os.Environ()
	for _, k := range sortedKeys(s.Env) {
		cmd.Env = append(cmd.Env, k+"="+s.Env[k])
	}
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	cmd.Process.Release()
	return os.WriteFile(filepath.Join(d, "pid"), []byte(strconv.Itoa(pid)), 0o644)
}

func (p *ProcBackend) Observe(name string) Observation {
	d := p.adir(name)
	if _, err := os.Stat(filepath.Join(d, "pid")); err != nil {
		return Observation{Exists: false, Status: StatusExited}
	}
	screen := lastScreen(readTail(filepath.Join(d, "screen.log"), 16<<10))
	if _, err := os.Stat(filepath.Join(d, "exit")); err == nil {
		return Observation{Exists: false, Status: StatusExited, Screen: screen}
	}
	hb, err := os.Stat(filepath.Join(d, "heartbeat"))
	if err != nil {
		pidInfo, _ := os.Stat(filepath.Join(d, "pid"))
		if pidInfo != nil && time.Since(pidInfo.ModTime()) < 10*time.Second {
			return Observation{Exists: true, Status: StatusWorking, Screen: screen}
		}
		return Observation{Exists: false, Status: StatusExited, Screen: screen}
	}
	if time.Since(hb.ModTime()) > 6*time.Second {
		return Observation{Exists: false, Status: StatusExited, Screen: screen}
	}
	st := StatusIdle
	if li, err := os.Stat(filepath.Join(d, "screen.log")); err == nil && time.Since(li.ModTime()) < 3*time.Second {
		st = StatusWorking
	}
	if b, err := os.ReadFile(filepath.Join(d, "state")); err == nil { // agent-reported state (hook-like)
		if s := strings.TrimSpace(string(b)); s != "" {
			st = s
		}
	}
	return Observation{Exists: true, Status: st, Screen: screen}
}

// inputMark is written to screen.log by the relay each time it delivers input. Like a terminal that
// redraws after a key, the screen an observer sees is only what the agent printed since its last input.
const inputMark = "\x1e⏎ "

func lastScreen(log string) string {
	if i := strings.LastIndex(log, inputMark); i >= 0 {
		return log[i+len(inputMark):]
	}
	return log
}

func readTail(path string, n int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.Size() > n {
		f.Seek(st.Size()-n, io.SeekStart)
	}
	b, _ := io.ReadAll(f)
	return string(b)
}

var keyBytes = map[string]string{"Enter": "\n", "Escape": "\x1b", "Up": "\x1b[A", "Down": "\x1b[B", "Tab": "\t"}

func (p *ProcBackend) inbox(name, data string) error {
	f, err := os.OpenFile(filepath.Join(p.adir(name), "inbox"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("agent %s has no inbox: %w", name, err)
	}
	defer f.Close()
	_, err = f.WriteString(data)
	return err
}

func (p *ProcBackend) Send(name, text string) error {
	return p.inbox(name, strings.ReplaceAll(text, "\n", " ")+"\n")
}

func (p *ProcBackend) Keys(name string, keys ...string) error {
	var b strings.Builder
	for _, k := range keys {
		if v, ok := keyBytes[k]; ok {
			b.WriteString(v)
		} else {
			b.WriteString(k)
		}
	}
	return p.inbox(name, b.String())
}

func (p *ProcBackend) Stop(name string) error {
	b, err := os.ReadFile(filepath.Join(p.adir(name), "pid"))
	if err != nil {
		return nil
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if pid > 0 {
		killTree(pid)
	}
	os.WriteFile(filepath.Join(p.adir(name), "exit"), []byte("stopped"), 0o644)
	return nil
}

func (p *ProcBackend) Live() ([]string, error) {
	entries, err := os.ReadDir(p.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var live []string
	for _, e := range entries {
		if e.IsDir() && p.Observe(e.Name()).Exists {
			live = append(live, e.Name())
		}
	}
	return live, nil
}

// Relay is the body of `herdr-goal relay <dir> -- argv...`: inbox -> stdin, stdout+stderr -> screen.log,
// a heartbeat every second, "exit" with the code at the end.
func Relay(dir string, argv []string) int {
	logf, err := os.OpenFile(filepath.Join(dir, "screen.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer logf.Close()
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = logf, logf
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return 2
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(logf, "relay: cannot start %s: %v\n", argv[0], err)
		os.WriteFile(filepath.Join(dir, "exit"), []byte("start-failed"), 0o644)
		return 2
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var offset int64
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	beat := time.Time{}
	for {
		select {
		case err := <-done:
			code := 0
			if err != nil {
				code = 1
				var ee *exec.ExitError
				if errors.As(err, &ee) {
					code = ee.ExitCode()
				}
			}
			os.WriteFile(filepath.Join(dir, "exit"), []byte(strconv.Itoa(code)), 0o644)
			return 0
		case <-tick.C:
			if time.Since(beat) >= time.Second {
				os.WriteFile(filepath.Join(dir, "heartbeat"), []byte(time.Now().Format(time.RFC3339)), 0o644)
				beat = time.Now()
			}
			if f, err := os.Open(filepath.Join(dir, "inbox")); err == nil {
				f.Seek(offset, io.SeekStart)
				b, _ := io.ReadAll(f)
				f.Close()
				if len(b) > 0 {
					offset += int64(len(b))
					fmt.Fprintf(logf, "%s%q\n", inputMark, oneLine(string(b), 60))
					stdin.Write(bytes.Clone(b))
				}
			}
		}
	}
}
