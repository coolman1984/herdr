//go:build windows

package goal

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var errLocked = errors.New("locked by another process")

// lockFile takes an exclusive LockFileEx lock without waiting; Windows releases it when the handle
// closes or the process dies.
func lockFile(f *os.File) error {
	const lockfileExclusive, failImmediately = 0x2, 0x1
	var ol syscall.Overlapped
	r, _, err := kernel32.NewProc("LockFileEx").Call(f.Fd(), lockfileExclusive|failImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&ol)))
	if r == 0 {
		if errno, ok := err.(syscall.Errno); ok && (errno == 33 || errno == 167) { // ERROR_LOCK_VIOLATION, ERROR_LOCK_FAILED
			return errLocked
		}
		return err
	}
	return nil
}
