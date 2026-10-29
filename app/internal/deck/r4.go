package deck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// commandError records whether a child process could have submitted a write.
type commandError struct {
	Started bool
	Err     error
}

func (e *commandError) Error() string { return e.Err.Error() }
func (e *commandError) Unwrap() error { return e.Err }
func notSent(err error) error {
	return &statusError{Status: 502, Msg: "not_sent", Fields: map[string]any{"message": err.Error()}}
}
func readFailure(err error) error {
	if err == nil {
		return nil
	}
	var timeout interface{ Timeout() bool }
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
		return &statusError{Status: 504, Msg: "read_timeout", Fields: map[string]any{"message": err.Error()}}
	}
	return err
}

type debugCounters struct {
	Events, Bytes, Polls, Errors, Resyncs uint64
	LastMS, MaxMS                         float64
}

func (a *App) r4Endpoint(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/api/pane/raw":
		var p paneRequest
		if !readBody(w, r, &p) {
			return true
		}
		done := a.rawTurn(p.Key)
		defer done()
		a.lockedWrite(w, func() (any, error) {
			if err := r.Context().Err(); err != nil {
				return nil, notSent(err)
			}
			if (p.Text == "" && len(p.Keys) == 0) || len(p.Text) > 64<<10 || strings.ContainsRune(p.Text, 0) || len(p.Keys) > 32 {
				return nil, &statusError{Status: 400, Msg: "invalid raw input"}
			}
			keys := make([]string, len(p.Keys))
			for i, key := range p.Keys {
				keys[i] = normalizeKey(key)
				if !validKey(keys[i]) {
					return nil, &statusError{Status: 400, Msg: "unsupported key; nothing sent"}
				}
			}
			id, err := a.paneID(p.Key)
			if err != nil {
				return nil, err
			}
			state, err := a.currentSnapshot(r.Context())
			if err != nil {
				return nil, err
			}
			found := false
			for _, pane := range state.Panes {
				found = found || pane.ID == id
			}
			if !found {
				return nil, fmt.Errorf("pane no longer exists")
			}
			if p.Text != "" {
				if _, err = a.herdr.Call(r.Context(), "pane", "send-text", id, p.Text); err != nil {
					return nil, err
				}
			}
			for _, key := range keys {
				if _, err = a.herdr.Call(r.Context(), "pane", "send-keys", id, key); err != nil {
					return nil, err
				}
			}
			return map[string]bool{"ok": true}, nil
		})
		return true
	case "/api/prefs":
		path := filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-prefs.json")
		if r.Method == "GET" {
			a.prefsMu.Lock()
			defer a.prefsMu.Unlock()
			data, err := os.ReadFile(path)
			prefs := map[string]any{}
			if err == nil {
				err = json.Unmarshal(data, &prefs)
			} else if os.IsNotExist(err) {
				err = nil
			}
			if err != nil {
				failHTTP(w, 503, fmt.Errorf("preferences unavailable: %w", err))
				return true
			}
			if prefs == nil {
				prefs = map[string]any{}
			}
			respond(w, 200, map[string]any{"prefs": prefs})
			return true
		}
		data, err := io.ReadAll(io.LimitReader(r.Body, (64<<10)+1))
		if len(data) > 64<<10 {
			failHTTP(w, 413, fmt.Errorf("preferences exceed 64 KiB"))
			return true
		}
		var p struct {
			Prefs map[string]any `json:"prefs"`
		}
		if err == nil {
			err = json.Unmarshal(data, &p)
		}
		if err != nil || p.Prefs == nil {
			failHTTP(w, 400, fmt.Errorf("prefs must be a JSON object"))
			return true
		}
		a.prefsMu.Lock()
		defer a.prefsMu.Unlock()
		data, _ = json.Marshal(p.Prefs)
		if err = atomicWrite(path, data); err != nil {
			failHTTP(w, 503, err)
			return true
		}
		respond(w, 200, map[string]bool{"ok": true})
		return true
	case "/api/debug/counters":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		a.mu.Lock()
		c := a.counters
		clients := len(a.clients)
		jobs := 0
		for _, j := range a.jobs {
			if j.Status == "running" {
				jobs++
			}
		}
		a.mu.Unlock()
		waiting := int64(0)
		if cli, ok := a.herdr.(*CLI); ok && cli.Socket != nil {
			waiting = cli.Socket.waiting.Load()
		}
		var rss any
		if bytes, err := processMemory(uint32(os.Getpid())); err == nil {
			rss = float64(bytes) / (1 << 20)
		}
		respond(w, 200, map[string]any{"goroutines": runtime.NumGoroutine(), "handles": processHandles(), "rss_mb": rss, "sse_clients": clients, "sse_events": c.Events, "sse_bytes": c.Bytes, "polls": c.Polls, "poll_errors": c.Errors, "poll_ms_last": c.LastMS, "poll_ms_max": c.MaxMS, "rpc_waiting": waiting, "resyncs": c.Resyncs, "jobs_live": jobs})
		return true
	}
	return false
}

func normalizeKey(key string) string {
	aliases := map[string]string{"Escape": "esc", "C-c": "ctrl+c", "Enter": "enter", "Up": "up", "Down": "down", "Tab": "tab", "Left": "left", "Right": "right", "Backspace": "backspace"}
	if value, ok := aliases[key]; ok {
		return value
	}
	return key
}

// a.mu protects subscriptions, clients and outgoing queues.
func (a *App) setVisibleLocked(client string, ids []string) {
	if client == "" {
		client = "legacy"
	}
	if a.interests == nil {
		a.interests = map[string]map[string]bool{}
	}
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	a.interests[client] = set
	a.unionVisibleLocked()
}
func (a *App) unionVisibleLocked() {
	a.visible = map[string]bool{}
	for _, set := range a.interests {
		for id := range set {
			a.visible[id] = true
		}
	}
}
func (a *App) interestedLocked(ch chan []byte, p Pane) bool {
	id := a.clientIDs[ch]
	if id == "" {
		return true
	} // internal legacy subscribers
	set, ok := a.interests[id]
	if !ok {
		set = a.interests["legacy"]
	}
	return set[p.ID] || set[p.Key]
}
func (a *App) clientStateLocked(ch chan []byte, s State) State {
	s.Panes = append([]Pane{}, s.Panes...)
	for i := range s.Panes {
		if !a.interestedLocked(ch, s.Panes[i]) {
			s.Panes[i].Screen = ""
		}
	}
	return s
}
func (a *App) queueLocked(ch chan []byte, msg []byte) {
	select {
	case ch <- msg:
	default:
		lostResult := strings.Contains(string(msg), "event: result\n")
	drain:
		for {
			select {
			case queued := <-ch:
				lostResult = lostResult || strings.Contains(string(queued), "event: result\n")
			default:
				break drain
			}
		}
		a.counters.Resyncs++
		if lostResult {
			// A state snapshot cannot repair a lost job result. Close only this
			// stream so its reconnect fetches the authoritative /api/jobs ledger.
			delete(a.clients, ch)
			delete(a.interests, a.clientIDs[ch])
			delete(a.clientIDs, ch)
			a.unionVisibleLocked()
			close(ch)
			return
		}
		// Include settings on every overflow resync: a slow client must not miss Safe mode.
		ch <- append(event("settings", map[string]bool{"read_only": a.readOnly}), event("state", a.clientStateLocked(ch, a.state))...)
	}
}
func (a *App) countSSE(msg []byte) {
	a.mu.Lock()
	a.counters.Bytes += uint64(len(msg))
	a.counters.Events += uint64(strings.Count(string(msg), "event: "))
	a.mu.Unlock()
}
func (a *App) finishPoll(start time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.counters.Polls++
	a.counters.LastMS = float64(time.Since(start).Microseconds()) / 1000
	a.counters.MaxMS = max(a.counters.MaxMS, a.counters.LastMS)
}

// Ticket chains establish FIFO order for requests entering the raw handler for a
// pane. The browser queue supplies ordering across network transit.
func (a *App) rawTurn(key string) func() {
	a.mu.Lock()
	if a.rawTails == nil {
		a.rawTails = map[string]chan struct{}{}
	}
	previous := a.rawTails[key]
	next := make(chan struct{})
	a.rawTails[key] = next
	a.mu.Unlock()
	if previous != nil {
		<-previous
	}
	return func() {
		a.mu.Lock()
		if a.rawTails[key] == next {
			delete(a.rawTails, key)
		}
		close(next)
		a.mu.Unlock()
	}
}
