package deck

import (
	"io"
	"os"
	"syscall"
)

// dialPipe opens Herdr's client socket: on Windows a named pipe \\.\pipe\<marker path>.
func dialPipe(path string) (io.ReadWriteCloser, error) {
	name, err := syscall.UTF16PtrFromString(`\\.\pipe\` + path)
	if err != nil {
		return nil, err
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), "herdr-client"), nil
}
