//go:build !windows

package goal

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func hideWindow(cmd *exec.Cmd) {}

// detach starts the process in its own session so it outlives the runner.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

// newGroup puts the command in its own process group so killTree reaches its children.
func newGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func killTree(pid int) { syscall.Kill(-pid, syscall.SIGKILL); syscall.Kill(pid, syscall.SIGKILL) }

// processAlive: signal 0 succeeds for a zombie too (killed, not yet reaped - common in containers whose
// init does not reap), so on Linux the state in /proc decides.
func processAlive(pid int) bool {
	if pid <= 0 || syscall.Kill(pid, 0) != nil {
		return false
	}
	if b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		if i := strings.LastIndexByte(string(b), ')'); i > 0 && i+2 < len(b) && b[i+2] == 'Z' {
			return false
		}
	}
	return true
}

// FreeRAMMB reads MemAvailable from /proc/meminfo; -1 when unknown.
func FreeRAMMB() int {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return -1
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "MemAvailable:") {
			f := strings.Fields(l)
			if len(f) >= 2 {
				kb, _ := strconv.Atoi(f[1])
				return kb / 1024
			}
		}
	}
	return -1
}

func FreeDiskMB(path string) int {
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return -1
	}
	return int(st.Bavail * uint64(st.Bsize) / (1 << 20))
}
