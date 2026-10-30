//go:build windows

package goal

import (
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

const createNoWindow = 0x08000000

// Nothing may flash a console window (CLAUDE.md section 3).
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | 0x00000200 | 0x00000008}
}

func killTree(pid int) {
	cmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	hideWindow(cmd)
	cmd.Run()
}

func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(0x1000, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	return syscall.GetExitCodeProcess(h, &code) == nil && code == 259
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func FreeRAMMB() int {
	type memStatus struct {
		Length, MemoryLoad                               uint32
		TotalPhys, AvailPhys, TotalPage, AvailPage       uint64
		TotalVirtual, AvailVirtual, AvailExtendedVirtual uint64
	}
	m := memStatus{Length: uint32(unsafe.Sizeof(memStatus{}))}
	if r, _, _ := kernel32.NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return -1
	}
	return int(m.AvailPhys >> 20)
}

func FreeDiskMB(path string) int {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return -1
	}
	var free uint64
	if r, _, _ := kernel32.NewProc("GetDiskFreeSpaceExW").Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&free)), 0, 0); r == 0 {
		return -1
	}
	return int(free >> 20)
}
