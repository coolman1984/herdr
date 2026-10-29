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
	"sync/atomic"
	"syscall"
	"time"
)

// Herdr's public API is NDJSON on \\.\pipe\<socket marker path> on Windows.
// Protocol 22 closes ordinary RPC connections after one response. Reuse the client,
// serialize bounded connections, and never spawn polling child processes.
type backgroundRPC struct{}

var errRPCBusy = fmt.Errorf("background RPC deferred for an active request")

type Socket struct {
	Path        string
	once        sync.Once
	gate        chan struct{}
	waiting     atomic.Int64
	file        *os.File
	reader      *bufio.Reader
	sequence    uint64
	Connections uint64
}

func (s *Socket) Close() { _ = s.acquire(context.Background()); defer s.release(); s.drop() }
func (s *Socket) acquire(ctx context.Context) error {
	s.once.Do(func() { s.gate = make(chan struct{}, 1) })
	if err := ctx.Err(); err != nil {
		return err
	}
	if background, _ := ctx.Value(backgroundRPC{}).(bool); background {
		select {
		case s.gate <- struct{}{}:
			return nil
		default:
			return errRPCBusy
		}
	}
	s.waiting.Add(1)
	defer s.waiting.Add(-1)
	select {
	case s.gate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			s.release()
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Socket) release() { <-s.gate }
func (s *Socket) drop() {
	if s.file != nil {
		s.file.Close()
		s.file = nil
		s.reader = nil
	}
}
func (s *Socket) request(parent context.Context, method string, params any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 6*time.Second)
	defer cancel()
	if err := s.acquire(ctx); err != nil {
		return nil, err
	}
	defer s.release()
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
	// Cancellation also interrupts an in-flight read before its deadline. Capture the file
	// and join the callback before dropping it or admitting the next request.
	file := s.file
	stopped := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { file.SetDeadline(time.Now()); close(stopped) })
	defer func() {
		if !stop() {
			<-stopped
		}
	}()
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
