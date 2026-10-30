package goal

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// tailBuf keeps only the last max bytes of a command's output, so a command printing gigabytes cannot
// fill the runner's memory; total counts what was dropped.
type tailBuf struct {
	mu    sync.Mutex
	max   int
	buf   []byte
	total int64
}

func (t *tailBuf) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.total += int64(len(p))
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max {
		t.buf = append([]byte(nil), t.buf[len(t.buf)-t.max:]...)
	}
	return len(p), nil
}

func (t *tailBuf) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.total > int64(len(t.buf)) {
		return fmt.Sprintf("[... %d earlier bytes dropped]\n%s", t.total-int64(len(t.buf)), t.buf)
	}
	return string(t.buf)
}

// runBounded runs one process with a timeout and a cancel context. On either, the whole process tree
// is killed (not just the shell), and output is kept bounded.
func runBounded(ctx context.Context, dir string, timeout time.Duration, maxOut int, name string, args ...string) (string, error, bool) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	hideWindow(cmd)
	newGroup(cmd)
	out := &tailBuf{max: maxOut}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		return "", err, false
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return out.String(), err, false
	case <-ctx.Done():
		killTree(cmd.Process.Pid)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
		}
		return out.String(), ctx.Err(), ctx.Err() == context.DeadlineExceeded
	}
}

// shellArgv runs a check command with the shell this machine has: PowerShell 7 if present, else
// Windows PowerShell (CLAUDE policy: use the PowerShell that exists, do not assume 7); sh elsewhere.
func shellArgv(c string) (string, []string) {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("pwsh"); err == nil {
			return "pwsh", []string{"-NoProfile", "-NonInteractive", "-Command", c}
		}
		return "powershell", []string{"-NoProfile", "-NonInteractive", "-Command", c}
	}
	return "sh", []string{"-c", c}
}
