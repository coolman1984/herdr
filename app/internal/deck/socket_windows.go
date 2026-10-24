package deck

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Herdr's public API is NDJSON on \\.\pipe\<socket marker path> on Windows.
// Protocol 22 closes ordinary RPC connections after one response. Reuse the client,
// serialize bounded connections, and never spawn polling child processes.
type Socket struct {
	Path        string
	mu          sync.Mutex
	file        *os.File
	reader      *bufio.Reader
	sequence    uint64
	Connections uint64
}

func (s *Socket) Close() { s.mu.Lock(); defer s.mu.Unlock(); s.drop() }
func (s *Socket) drop() {
	if s.file != nil {
		s.file.Close()
		s.file = nil
		s.reader = nil
	}
}
func (s *Socket) request(parent context.Context, method string, params any) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(parent, 6*time.Second)
	defer cancel()
	if s.file == nil {
		name, err := syscall.UTF16PtrFromString(`\\.\pipe\` + s.Path)
		if err != nil {
			return nil, err
		}
		handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OVERLAPPED, 0)
		if err != nil {
			return nil, fmt.Errorf("Herdr socket: %w (start Herdr, or use an account with access to its server)", err)
		}
		s.file = os.NewFile(uintptr(handle), "herdr-api")
		s.reader = bufio.NewReaderSize(s.file, 32<<10)
		s.Connections++
	}
	defer s.drop()
	deadline, _ := ctx.Deadline()
	if err := s.file.SetDeadline(deadline); err != nil {
		s.drop()
		return nil, fmt.Errorf("Herdr socket deadlines unavailable: %w", err)
	}
	// Deadlines cancel I/O in the Go poller; no per-request goroutines are needed.
	s.sequence++
	id := fmt.Sprintf("deck-%d", s.sequence)
	data, err := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	if _, err = s.file.Write(append(data, '\n')); err != nil {
		s.drop()
		return nil, fmt.Errorf("Herdr socket write: %w", err)
	}
	result := []byte{}
	for {
		part, e := s.reader.ReadSlice('\n')
		if len(result)+len(part) > 4<<20 {
			s.drop()
			return nil, fmt.Errorf("Herdr socket response exceeds 4 MB")
		}
		result = append(result, part...)
		if e == bufio.ErrBufferFull {
			continue
		}
		if e != nil {
			s.drop()
			return nil, fmt.Errorf("Herdr socket read: %w", e)
		}
		break
	}
	var response struct {
		ID    string                          `json:"id"`
		Error *struct{ Code, Message string } `json:"error"`
	}
	if err = json.Unmarshal(result, &response); err != nil {
		s.drop()
		return nil, fmt.Errorf("invalid Herdr NDJSON: %w", err)
	}
	if response.ID != id {
		s.drop()
		return nil, fmt.Errorf("Herdr response ID mismatch")
	}
	if response.Error != nil {
		return nil, fmt.Errorf("%s: %s", response.Error.Code, response.Error.Message)
	}
	return result, nil
}
func (s *Socket) call(ctx context.Context, args []string) ([]byte, bool, error) {
	if len(args) == 2 && args[0] == "api" && args[1] == "snapshot" {
		out, err := s.request(ctx, "session.snapshot", map[string]any{})
		return out, true, err
	}
	if len(args) == 2 && args[0] == "agent" && args[1] == "list" {
		out, err := s.request(ctx, "agent.list", map[string]any{})
		return out, true, err
	}
	if len(args) >= 3 && args[0] == "pane" && args[1] == "process-info" {
		out, err := s.request(ctx, "pane.process_info", map[string]any{"pane_id": args[2]})
		return out, true, err
	}
	if len(args) >= 3 && args[0] == "pane" && args[1] == "read" {
		params := map[string]any{"pane_id": args[2], "source": "recent", "format": "text", "strip_ansi": false}
		for i := 3; i+1 < len(args); i += 2 {
			switch args[i] {
			case "--source":
				params["source"] = args[i+1]
			case "--format":
				params["format"] = args[i+1]
			case "--lines":
				n, e := strconv.Atoi(args[i+1])
				if e != nil {
					return nil, true, e
				}
				params["lines"] = n
			}
		}
		out, err := s.request(ctx, "pane.read", params)
		if err != nil {
			return nil, true, err
		}
		var response struct {
			Result struct {
				Read struct {
					Text string `json:"text"`
				} `json:"read"`
				Text string `json:"text"`
			} `json:"result"`
		}
		if err = json.Unmarshal(out, &response); err != nil {
			return nil, true, err
		}
		// Protocol 22 returns result.read.text (validated in the lab integration test).
		return []byte(response.Result.Read.Text), true, nil
	}
	return nil, false, nil
}

var _ io.Reader = (*os.File)(nil)
