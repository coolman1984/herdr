package deck

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var user32 = syscall.NewLazyDLL("user32.dll")

func hideProcess(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

// Closing this job ends only this child tree, including grandchildren on timeout.
func containProcess(cmd *exec.Cmd) (syscall.Handle, error) {
	job, _, e := kernel32.NewProc("CreateJobObjectW").Call(0, 0)
	if job == 0 {
		return 0, fmt.Errorf("create child job: %w", e)
	}
	type basicLimits struct {
		ProcessTime, JobTime int64
		Flags                uint32
		Minimum, Maximum     uintptr
		Active               uint32
		Affinity             uintptr
		Priority, Scheduling uint32
	}
	type extendedLimits struct {
		Basic                                                      basicLimits
		IO                                                         [6]uint64
		ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory uintptr
	}
	limits := extendedLimits{}
	limits.Basic.Flags = 0x2000
	ok, _, e := kernel32.NewProc("SetInformationJobObject").Call(job, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits))
	if ok == 0 {
		syscall.CloseHandle(syscall.Handle(job))
		return 0, fmt.Errorf("limit child job: %w", e)
	}
	handle, err := syscall.OpenProcess(0x0100|0x0001, false, uint32(cmd.Process.Pid))
	if err != nil {
		syscall.CloseHandle(syscall.Handle(job))
		return 0, err
	}
	defer syscall.CloseHandle(handle)
	ok, _, e = kernel32.NewProc("AssignProcessToJobObject").Call(job, uintptr(handle))
	if ok == 0 {
		syscall.CloseHandle(syscall.Handle(job))
		return 0, fmt.Errorf("contain child process: %w", e)
	}
	return syscall.Handle(job), nil
}
func messageBox(message string) {
	m, _ := syscall.UTF16PtrFromString(message)
	t, _ := syscall.UTF16PtrFromString("Herdr Deck")
	user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x10)
}
func instanceMutex(name string) (syscall.Handle, bool, error) {
	ptr, _ := syscall.UTF16PtrFromString(`Local\HerdrDeck-` + name)
	handle, _, err := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(ptr)))
	if handle == 0 {
		return 0, false, fmt.Errorf("single-instance mutex: %w", err)
	}
	return syscall.Handle(handle), err == syscall.Errno(183), nil
}
func closeMutex(h syscall.Handle) { syscall.CloseHandle(h) }
func findEdge() (string, error) {
	roots := []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")}
	for _, root := range roots {
		if root != "" {
			path := filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe")
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}
	// Standard library syscall registry API, no helper process or external module.
	sub, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\App Paths\msedge.exe`)
	for _, root := range []syscall.Handle{syscall.HKEY_CURRENT_USER, syscall.HKEY_LOCAL_MACHINE} {
		var key syscall.Handle
		if syscall.RegOpenKeyEx(root, sub, 0, syscall.KEY_READ, &key) != nil {
			continue
		}
		buf := make([]uint16, 2048)
		size := uint32(len(buf) * 2)
		var kind uint32
		err := syscall.RegQueryValueEx(key, nil, nil, &kind, (*byte)(unsafe.Pointer(&buf[0])), &size)
		syscall.RegCloseKey(key)
		if err == nil {
			path := strings.Trim(syscall.UTF16ToString(buf), `"`)
			if _, err = os.Stat(path); err == nil {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("Microsoft Edge is missing. Install Edge to open Herdr Deck.")
}
func openWindow(edge, url, profile string) error {
	cmd := exec.Command(edge, "--app="+url, "--user-data-dir="+profile, "--window-size=1400,900", "--no-first-run")
	hideProcess(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
func systemHealth() (map[string]any, error) {
	type memoryStatus struct {
		Length, Load                                                                          uint32
		TotalPhys, AvailPhys, TotalPage, AvailPage, TotalVirtual, AvailVirtual, AvailExtended uint64
	}
	status := memoryStatus{}
	status.Length = uint32(unsafe.Sizeof(status))
	ok, _, err := kernel32.NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return nil, fmt.Errorf("memory status: %w", err)
	}
	disks := []map[string]any{}
	for _, drive := range []string{"C:", "G:", "I:"} {
		ptr, _ := syscall.UTF16PtrFromString(drive + `\`)
		var free, total, available uint64
		ok, _, e := kernel32.NewProc("GetDiskFreeSpaceExW").Call(uintptr(unsafe.Pointer(ptr)), uintptr(unsafe.Pointer(&available)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
		row := map[string]any{"drive": drive}
		if ok == 0 {
			row["error"] = e.Error()
		} else {
			row["free_gb"] = float64(available) / (1 << 30)
		}
		disks = append(disks, row)
	}
	return map[string]any{"ram_used_pct": status.Load, "ram_total_gb": float64(status.TotalPhys) / (1 << 30), "ram_available_mb": float64(status.AvailPhys) / (1 << 20), "disks": disks}, nil
}
func processMemory(pid uint32) (uint64, error) {
	handle, err := syscall.OpenProcess(0x0400|0x0010, false, pid)
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(handle)
	type counters struct {
		Size, PageFault                                                                                            uint32
		PeakWorking, Working, QuotaPeakPaged, QuotaPaged, QuotaPeakNonPaged, QuotaNonPaged, Pagefile, PeakPagefile uintptr
	}
	value := counters{}
	value.Size = uint32(unsafe.Sizeof(value))
	ok, _, e := syscall.NewLazyDLL("psapi.dll").NewProc("GetProcessMemoryInfo").Call(uintptr(handle), uintptr(unsafe.Pointer(&value)), uintptr(value.Size))
	if ok == 0 {
		return 0, e
	}
	return uint64(value.Working), nil
}
func processHandles() uint32 {
	var count uint32
	proc, _, _ := kernel32.NewProc("GetCurrentProcess").Call()
	kernel32.NewProc("GetProcessHandleCount").Call(proc, uintptr(unsafe.Pointer(&count)))
	return count
}
