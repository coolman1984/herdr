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

func killTree(pid int) { syscall.Kill(-pid, syscall.SIGKILL); syscall.Kill(pid, syscall.SIGKILL) }

func processAlive(pid int) bool { return pid > 0 && syscall.Kill(pid, 0) == nil }

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
