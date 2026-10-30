//go:build !windows

package goal

import (
	"errors"
	"os"
	"syscall"
)

var errLocked = errors.New("locked by another process")

// lockFile takes an exclusive OS lock without waiting; it lasts until the file is closed or the
// process dies (no stale lock after a kill -9).
func lockFile(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return errLocked
		}
		return err
	}
	return nil
}
