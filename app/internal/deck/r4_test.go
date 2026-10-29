package deck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func r4post(a *App, path, body string) *httptest.ResponseRecorder {
	return request(a, "POST", path, body, "test-token", "")
}

func TestR4RawOrderedAndReadOnly(t *testing.T) {
	f := &fakeHerdr{status: "blocked", screen: "› existing draft"}
	a := testApp(f)
	for _, text := range []string{"first", " second"} {
		w := r4post(a, "/api/pane/raw", fmt.Sprintf(`{"key":"lab|w1:p1","text":%q,"keys":["enter"]}`, text))
		if w.Code != 200 {
			t.Fatalf("raw %d %s", w.Code, w.Body)
		}
	}
	want := []string{"pane|send-text|w1:p1|first", "pane|send-keys|w1:p1|enter", "pane|send-text|w1:p1| second", "pane|send-keys|w1:p1|enter"}
	if fmt.Sprint(f.writes) != fmt.Sprint(want) {
		t.Fatal(f.writes)
	}
	if w := r4post(a, "/api/pane/raw", `{"key":"lab|w1:p1","text":"must not leak","keys":["enter","not-a-key"]}`); w.Code != 400 || len(f.writes) != 4 {
		t.Fatal("partially sent invalid raw batch", w.Code, f.writes)
	}
	r4post(a, "/api/read-only", `{"enabled":true}`)
	if w := r4post(a, "/api/pane/raw", `{"key":"lab|w1:p1","text":"x"}`); w.Code != 403 || len(f.writes) != 4 {
		t.Fatal(w.Code, f.writes)
	}
}

func TestR4ReservationOwnershipAndOverlap(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	a := testApp(&fakeHerdr{})
	body := `{"task":"tasks/a.md","owner":"one","paths":["app/internal"]}`
	one := decodeBody(t, r4post(a, "/api/reservations", body))
	two := decodeBody(t, r4post(a, "/api/reservations", body))
	if one["id"] != two["id"] {
		t.Error("same ownership changed id")
	}
	other := strings.Replace(body, `"one"`, `"two"`, 1)
	if w := r4post(a, "/api/reservations", other); w.Code != 409 {
		t.Errorf("reassigned: %d %s", w.Code, w.Body)
	}
	shared := strings.TrimSuffix(other, "}") + `,"overlap":true}`
	if w := r4post(a, "/api/reservations", shared); w.Code != 200 {
		t.Fatalf("overlap: %d %s", w.Code, w.Body)
	}
	if len(a.guards.Reservations) != 2 {
		t.Fatal(a.guards.Reservations)
	}
}

func TestR4UnreadableReservationLedger(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	if err := atomicWrite(reservationPath(), []byte("broken")); err != nil {
		t.Fatal(err)
	}
	a := testApp(&fakeHerdr{})
	w := r4post(a, "/api/reservations", `{"task":"t","owner":"o","paths":["app"]}`)
	if w.Code != 503 || decodeBody(t, w)["error"] != "ledger_unavailable" {
		t.Fatal(w.Code, w.Body)
	}
	data, _ := os.ReadFile(reservationPath())
	if string(data) != "broken" {
		t.Fatal("ledger overwritten")
	}
}

func TestR4UnknownInputAndMultiline(t *testing.T) {
	for _, kind := range []string{"claude", "codex"} {
		t.Run(kind, func(t *testing.T) {
			p := Pane{ID: "w1:p1", Agent: kind, Name: "coder"}
			if err := inputNotEmpty("unrecognized terminal", p); err == nil || err.Error() != "input_unknown" {
				t.Errorf("unknown allowed: %v", err)
			}
			screen := "› \n  draft on second line\n\n  ? for shortcuts"
			if kind == "claude" {
				screen = strings.Repeat("─", 40) + "\n❯ \n" + strings.Repeat("\n", 16) + "draft\n" + strings.Repeat("─", 40)
			}
			if err := inputNotEmpty(screen, p); err == nil || err.Error() != "input_not_empty" {
				t.Errorf("draft allowed: %v", err)
			}
		})
	}
}

func TestR4BlockedClearsOnFreshScreen(t *testing.T) {
	f := &fakeHerdr{status: "idle", screen: "Trust this folder?\nEnter to confirm · Esc to cancel"}
	a := testApp(f)
	a.refresh(context.Background(), true)
	f.screen = "› "
	a.refresh(context.Background(), true)
	if a.state.Panes[0].Blocked || a.state.Panes[0].Status == "blocked" {
		t.Fatal(a.state.Panes[0])
	}
}

func TestR4OfflineRetainsStalePanes(t *testing.T) {
	f := &fakeHerdr{status: "idle", screen: "› "}
	a := testApp(f)
	a.refresh(context.Background(), true)
	f.offline = true
	a.refresh(context.Background(), true)
	data, _ := json.Marshal(a.state)
	if len(a.state.Panes) != 1 || !strings.Contains(string(data), `"stale":true`) || a.state.Herdr.Running {
		t.Fatal(string(data))
	}
}

func TestR4RetainedRowsAndPatchBounds(t *testing.T) {
	rows := make([]string, 2400)
	for i := range rows {
		rows[i] = fmt.Sprint(i)
	}
	old, _ := parseScreen([]byte(strings.Join(rows, "\n")), 400)
	if len(strings.Split(old, "\n")) != 2000 {
		t.Fatalf("retained %d rows", len(strings.Split(old, "\n")))
	}
	next, _ := parseScreen([]byte(strings.Join(append(rows[23:], "tail"), "\n")), 400)
	current := strings.Split(old, "\n")
	for _, p := range rowChanges(old, next, 3) {
		if p.Start < 0 || p.Delete < 0 || p.Start+p.Delete > len(current) {
			t.Fatalf("bad bounds %+v", p)
		}
		current = append(append(append([]string{}, current[:p.Start]...), p.Lines...), current[p.Start+p.Delete:]...)
	}
	if strings.Join(current, "\n") != next {
		t.Fatal("patch did not reconstruct")
	}
}

func TestR4DiffTailUsesSinglePatch(t *testing.T) {
	old := strings.Repeat("same\n", 1999) + "old"
	next := strings.Repeat("same\n", 1999) + "new"
	patches := rowChanges(old, next, 1)
	if len(patches) != 1 || patches[0].Start != 1999 || patches[0].Delete != 1 {
		t.Fatalf("tail change scanned as scroll: %+v", patches)
	}
}

func TestR4PrefsPersistAndLimit(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	a := testApp(&fakeHerdr{})
	w := r4post(a, "/api/prefs", `{"prefs":{"theme":"light","nested":{"a":1}}}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	b := testApp(&fakeHerdr{})
	w = request(b, "GET", "/api/prefs", "", "test-token", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"theme":"light"`) {
		t.Fatal(w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-prefs.json")); err != nil {
		t.Fatal(err)
	}
	w = r4post(a, "/api/prefs", `{"prefs":{"large":"`+strings.Repeat("x", 65536)+`"}}`)
	if w.Code != 413 {
		t.Fatal(w.Code)
	}
}

func TestR4ReadTimeoutIsNotDelivery(t *testing.T) {
	w := httptest.NewRecorder()
	failHTTP(w, 502, fmt.Errorf("read timed out: %w", context.DeadlineExceeded))
	b := decodeBody(t, w)
	if w.Code != 504 || b["error"] != "read_timeout" || b["dispatch_id"] != nil || b["kind"] == "delivery_unknown" {
		t.Fatal(w.Code, b)
	}
}

func TestR4CancelledWriteNeverStarts(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	f := &fakeHerdr{}
	a := testApp(f)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := a.deliver(ctx, "lab|w1:p1", "w1:p1", "hi", "agent", "prompt", "coder", "hi")
	w := httptest.NewRecorder()
	if err == nil {
		t.Fatal("cancelled write sent", f.writes)
	}
	failHTTP(w, 502, err)
	if w.Code != 502 || decodeBody(t, w)["error"] != "not_sent" || len(f.writes) != 0 {
		t.Fatal(w.Code, w.Body, f.writes)
	}
}

func TestR4VisibilityIndependent(t *testing.T) {
	a := testApp(&fakeHerdr{})
	for _, body := range []string{`{"client_id":"c-1","keys":["lab|w1:p1"]}`, `{"client_id":"c-2","keys":["lab|w1:p2"]}`, `{"client_id":"c-1","keys":[]}`} {
		if w := r4post(a, "/api/visible", body); w.Code != 200 {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
	}
	if !a.visible["w1:p2"] || a.visible["w1:p1"] {
		t.Fatal(a.visible)
	}
}

func TestR4SettingsBroadcast(t *testing.T) {
	a := testApp(&fakeHerdr{})
	ch := make(chan []byte, 2)
	a.clients[ch] = true
	r4post(a, "/api/read-only", `{"enabled":true}`)
	select {
	case msg := <-ch:
		if !strings.Contains(string(msg), "event: settings") || !strings.Contains(string(msg), `"read_only":true`) {
			t.Fatal(string(msg))
		}
	default:
		t.Fatal("no settings event")
	}
}

func TestR4Counters(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.refresh(context.Background(), true)
	w := request(a, "GET", "/api/debug/counters", "", "test-token", "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	b := decodeBody(t, w)
	for _, k := range strings.Fields("goroutines handles rss_mb sse_clients sse_events sse_bytes polls poll_errors poll_ms_last poll_ms_max rpc_waiting resyncs jobs_live") {
		if _, ok := b[k]; !ok {
			t.Error("missing", k)
		}
	}
	if b["polls"].(float64) < 1 {
		t.Fatal(b)
	}
}

// The counters and delayed responses are protected because refresh/health can run concurrently.
type r4Herdr struct {
	fakeHerdr
	mu       sync.Mutex
	calls    []string
	delay    bool
	result   string
	snapshot string
	failure  error
}

func (f *r4Herdr) Call(ctx context.Context, args ...string) ([]byte, error) {
	f.mu.Lock()
	f.calls = append(f.calls, strings.Join(args, "|"))
	f.mu.Unlock()
	if f.delay && len(args) > 2 && args[1] == "read" && args[2] == "w1:p1" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if len(args) == 2 && args[0] == "api" {
		f.mu.Lock()
		snapshot := f.snapshot
		f.mu.Unlock()
		if snapshot != "" {
			return []byte(snapshot), nil
		}
		return []byte(twoAgentSnapshot("codex", "one", "two", "three")), nil
	}
	return f.fakeHerdr.Call(ctx, args...)
}
func (f *r4Herdr) Helper(context.Context, map[string]string) (string, error) {
	return f.result, f.failure
}
func (f *r4Herdr) RunScript(context.Context, string, map[string]any, time.Duration) (string, error) {
	return f.result, f.failure
}

func TestR4PollFairnessAndBackoff(t *testing.T) {
	f := &r4Herdr{delay: true}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	a.visible = map[string]bool{"w1:p1": true, "w1:p2": true, "w1:p3": true}
	start := time.Now()
	a.refresh(context.Background(), true)
	if time.Since(start) > 2*time.Second {
		t.Error("one pane exhausted cycle budget")
	}
	a.refresh(context.Background(), false)
	f.mu.Lock()
	calls := strings.Join(f.calls, "\n")
	f.mu.Unlock()
	if !strings.Contains(calls, "pane|read|w1:p2") || !strings.Contains(calls, "pane|read|w1:p3") {
		t.Error("later panes starved", calls)
	}
	if strings.Contains(calls, "agent|list") {
		t.Error("redundant agent list", calls)
	}
	if strings.Count(calls, "pane|read|w1:p1") > 1 {
		t.Error("failed pane immediately retried", calls)
	}
}

func TestR4HealthIncludesDAndNullUnknown(t *testing.T) {
	h, err := systemHealth()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range h["disks"].([]map[string]any) {
		if d["drive"] == "D:" {
			found = true
		}
		if _, ok := d["free_gb"]; !ok {
			t.Error("missing explicit null", d)
		}
	}
	if !found {
		t.Fatal("D: omitted")
	}
}

func TestR4LaunchPendingAndIdentity(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	f := &r4Herdr{result: "STARTED name=worker profile=codex pane=w2:p1"}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	a.root = t.TempDir()
	a.profiles["codex"] = Profile{Kind: "codex"}
	r4post(a, "/api/launch/policy", `{"max_active":4,"min_ram_mb":0,"min_disk_gb":0}`)
	body := `{"profile":"codex","worktree":true}`
	first := r4post(a, "/api/agent/start", body)
	if first.Code != 200 {
		t.Fatal(first.Code, first.Body)
	}
	b := decodeBody(t, first)
	if b["name"] != "worker" || b["pane_id"] != "w2:p1" {
		t.Error("missing confirmed identity", b)
	}
	second := r4post(a, "/api/agent/start", body)
	if second.Code != 409 || decodeBody(t, second)["error"] != "capacity" {
		t.Fatal("pending oversubscribed", second.Code, second.Body)
	}
}

func TestR4EveryLaunchRouteUsesAdmission(t *testing.T) {
	for _, route := range []string{"/api/workspace/open", "/api/layouts/build", "/api/agents/team"} {
		t.Run(route, func(t *testing.T) {
			t.Setenv("APPDATA", t.TempDir())
			f := &r4Herdr{}
			a := testApp(&f.fakeHerdr)
			a.herdr = f
			a.root = t.TempDir()
			a.profiles["codex"] = Profile{Kind: "codex"}
			a.profiles["claude"] = Profile{Kind: "claude", DefaultAccount: "2030"}
			a.lead = LeadIdentity{Tool: "claude", Account: "2030", Known: true}
			r4post(a, "/api/launch/policy", `{"max_active":1,"min_ram_mb":0,"min_disk_gb":0}`)
			body := fmt.Sprintf(`{"cwd":%q,"agent":"codex"}`, a.root)
			if route == "/api/layouts/build" {
				body = fmt.Sprintf(`{"plan":{"Where":0,"Columns":[{"Folder":%q,"Type":"new","Value":"codex"}]}}`, a.root)
			}
			if route == "/api/agents/team" {
				body = fmt.Sprintf(`{"cwd":%q,"preset":"claude-codex"}`, a.root)
			}
			w := r4post(a, route, body)
			if w.Code != 409 {
				t.Fatal("admission bypassed", w.Code, w.Body)
			}
		})
	}
}

func TestR4UnknownLaunchVolumeRefused(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	a := testApp(&fakeHerdr{})
	a.profiles["codex"] = Profile{Kind: "codex"}
	err := a.launchGuard(paneRequest{Profile: "codex", Cwd: `Z:\missing-volume`})
	if err == nil || err.Error() != "disk_unknown" {
		t.Fatal("unknown disk allowed/misreported", err)
	}
}

func TestR4HealthSingleSample(t *testing.T) {
	f := &r4Herdr{}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	a.state.Panes = []Pane{{ID: "w1:p1", Key: "lab|w1:p1"}}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); request(a, "GET", "/api/health", "", "test-token", "") }()
	}
	wg.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, c := range f.calls {
		if strings.Contains(c, "process-info") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("overlapping health samples: %d", count)
	}
}

type r4Stream struct {
	header   http.Header
	messages chan []byte
}

func (s *r4Stream) Header() http.Header { return s.header }
func (s *r4Stream) WriteHeader(int)     {}
func (s *r4Stream) Write(b []byte) (int, error) {
	s.messages <- append([]byte{}, b...)
	return len(b), nil
}
func (s *r4Stream) Flush() {}
func r4Connect(t *testing.T, a *App) (*r4Stream, string, func()) {
	t.Helper()
	s := &r4Stream{header: http.Header{}, messages: make(chan []byte, 64)}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	req := httptest.NewRequest("GET", a.origin+"/api/events?t=test-token", nil).WithContext(ctx)
	go func() { defer close(finished); a.ServeHTTP(s, req) }()
	closeStream := func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Error("SSE disconnect stuck")
		}
	}
	t.Cleanup(closeStream)
	var initial []byte
	select {
	case initial = <-s.messages:
	case <-time.After(time.Second):
		t.Fatal("SSE initial event missing")
	}
	if !strings.HasPrefix(string(initial), "event: hello\n") || !strings.Contains(string(initial), "event: settings\n") || !strings.Contains(string(initial), "event: state\n") {
		t.Fatalf("wrong initial sequence: %s", initial)
	}
	line := strings.Split(string(initial), "\n")[1]
	var hello struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &hello); err != nil {
		t.Fatal(err)
	}
	return s, hello.ClientID, closeStream
}

func TestR4ClientScreensAndDisconnect(t *testing.T) {
	a := testApp(&fakeHerdr{status: "idle"})
	one, id1, closeOne := r4Connect(t, a)
	two, id2, _ := r4Connect(t, a)
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	for id, key := range map[string]string{id1: "lab|w1:p1", id2: "lab|w1:p2"} {
		w := r4post(a, "/api/visible", fmt.Sprintf(`{"client_id":%q,"keys":[%q]}`, id, key))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
	}
	for len(one.messages) > 0 {
		<-one.messages
	}
	for len(two.messages) > 0 {
		<-two.messages
	}
	a.mu.Lock()
	old := a.state
	a.mu.Unlock()
	next := old
	next.Panes = []Pane{{ID: "w1:p1", Key: "lab|w1:p1", Screen: "only-one"}, {ID: "w1:p2", Key: "lab|w1:p2", Screen: "only-two"}}
	a.publish(old, next)
	for _, v := range []struct {
		s               *r4Stream
		want, forbidden string
	}{{one, "only-one", "only-two"}, {two, "only-two", "only-one"}} {
		deadline := time.After(time.Second)
		found := false
		for !found {
			select {
			case msg := <-v.s.messages:
				if strings.Contains(string(msg), v.forbidden) {
					t.Fatal("foreign screen leaked", string(msg))
				}
				found = strings.Contains(string(msg), v.want)
			case <-deadline:
				t.Fatal("screen update missing")
			}
		}
	}
	closeOne()
	a.mu.Lock()
	retained := a.visible["w1:p1"]
	other := a.visible["w1:p2"]
	a.mu.Unlock()
	if retained || !other {
		t.Fatal("closed client interest retained or other lost")
	}
}

func TestR4HiddenClientMetadataHeartbeat(t *testing.T) {
	f := &r4Herdr{}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	_, id, _ := r4Connect(t, a)
	if w := r4post(a, "/api/visible", fmt.Sprintf(`{"client_id":%q,"keys":[]}`, id)); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	// Wait for initial refresh, then age metadata and run the real poll loop.
	a.refreshMu.Lock()
	a.mu.Lock()
	a.metadataAt = time.Now().Add(-3 * time.Second)
	a.mu.Unlock()
	f.mu.Lock()
	f.calls = nil
	f.mu.Unlock()
	a.refreshMu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { a.poll(ctx); close(done) }()
	time.Sleep(600 * time.Millisecond)
	cancel()
	<-done
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := strings.Join(f.calls, "\n")
	if !strings.Contains(calls, "api|snapshot") || strings.Contains(calls, "pane|read") {
		t.Fatal("hidden heartbeat/screen reads wrong", calls)
	}
}

func TestR4MissingExecutableNotSent(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	a := testApp(&fakeHerdr{})
	a.herdr = &CLI{Bin: filepath.Join(t.TempDir(), "absent.exe")}
	_, _, err := a.deliver(context.Background(), "lab|w1:p1", "w1:p1", "hello", "agent", "prompt", "coder", "hello")
	w := httptest.NewRecorder()
	failHTTP(w, 502, err)
	b := decodeBody(t, w)
	if w.Code != 502 || b["error"] != "not_sent" || b["dispatch_id"] != nil {
		t.Fatal(w.Code, b)
	}
}

func TestR4LaunchFailureReleasesPending(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	f := &r4Herdr{failure: fmt.Errorf("start failed")}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	a.root = t.TempDir()
	a.profiles["codex"] = Profile{Kind: "codex"}
	r4post(a, "/api/launch/policy", `{"max_active":4,"min_ram_mb":0,"min_disk_gb":0}`)
	body := `{"profile":"codex","worktree":true}`
	if w := r4post(a, "/api/agent/start", body); w.Code != 502 {
		t.Fatal(w.Code, w.Body)
	}
	f.failure = nil
	if w := r4post(a, "/api/agent/start", body); w.Code != 200 {
		t.Fatal("failed launch leaked a slot", w.Code, w.Body)
	}
	if w := r4post(a, "/api/agent/start", body); w.Code != 409 {
		t.Fatal("successful launch lost pending slot", w.Code, w.Body)
	}
}

func TestR4CodexTallDraftWithBlankLines(t *testing.T) {
	screen := "› \n\n" + strings.Repeat("  typed continuation\n", 15) + "\n  GPT-6.1-Sol medium · HERDR · Context 81% left\n  ? for shortcuts"
	if err := inputNotEmpty(screen, Pane{Agent: "codex", Name: "coder"}); err == nil || err.Error() != "input_not_empty" {
		t.Fatal(err)
	}
}

func TestR4InheritedInputStyle(t *testing.T) {
	if got := inputText("\x1b[2m› placeholder\x1b[0m\n\n  ? for shortcuts", "codex"); got != "" {
		t.Fatal("lost style preceding prompt", got)
	}
}

func TestR4SocketCancelledBeforeAdmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	socket := &Socket{Path: "deck-r4-nonexistent-test-socket"}
	_, err := socket.request(ctx, "session.snapshot", map[string]any{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request attempted transport: %v", err)
	}
	// Hold transport admission without opening a pipe, then cancel a queued RPC.
	// The interface keeps this test buildable against the old Socket implementation.
	gate, ok := any(socket).(interface {
		acquire(context.Context) error
		release()
	})
	if !ok {
		t.Fatal("socket has no cancellation-aware admission")
	}
	if err := gate.acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer gate.release()
	waiting, stop := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer stop()
	start := time.Now()
	_, err = socket.request(waiting, "session.snapshot", map[string]any{})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatal("queued RPC did not cancel", err)
	}
}

func TestR4WorktreeVolumeFloor(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("USERPROFILE", `Z:\deck-r4-missing-volume`)
	a := testApp(&fakeHerdr{})
	a.root = t.TempDir()
	a.profiles["codex"] = Profile{Kind: "codex"}
	if err := a.launchGuard(paneRequest{Profile: "codex", Cwd: a.root, Worktree: true}); err == nil || err.Error() != "disk_unknown" {
		t.Fatal("worktree volume not checked", err)
	}
}

func TestR4HealthIncludesPaneVolume(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.state.Panes = []Pane{{ID: "w1:p1", Cwd: `Z:\deck-r4-missing-volume`}}
	w := request(a, "GET", "/api/health", "", "test-token", "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	b := decodeBody(t, w)
	found := false
	for _, v := range b["disks"].([]any) {
		row := v.(map[string]any)
		if row["drive"] == "Z:" {
			found = true
			if value, ok := row["free_gb"]; !ok || value != nil {
				t.Fatal("unknown volume not null", row)
			}
		}
	}
	if !found {
		t.Fatal("pane volume missing", b["disks"])
	}
}

func TestR4TeamReservesExpandedPlan(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	f := &r4Herdr{}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	a.root = t.TempDir()
	a.profiles["codex"] = Profile{Kind: "codex"}
	a.profiles["claude"] = Profile{Kind: "claude", DefaultAccount: "2030"}
	a.lead = LeadIdentity{Tool: "claude", Account: "2030", Known: true}
	r4post(a, "/api/launch/policy", `{"max_active":4,"min_ram_mb":0,"min_disk_gb":0}`)
	w := r4post(a, "/api/agents/team", fmt.Sprintf(`{"cwd":%q,"preset":"claude-codex","account":"2040"}`, a.root))
	if w.Code != 409 || decodeBody(t, w)["error"] != "capacity" {
		t.Fatal("team not expanded before admission", w.Code, w.Body)
	}
}

func TestR4LayoutChecksEveryAccount(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	f := &r4Herdr{}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	a.root = t.TempDir()
	a.profiles["codex"] = Profile{Kind: "codex"}
	a.profiles["claude"] = Profile{Kind: "claude", DefaultAccount: "2030"}
	a.lead = LeadIdentity{Tool: "claude", Account: "2030", Known: true}
	r4post(a, "/api/launch/policy", `{"max_active":16,"min_ram_mb":0,"min_disk_gb":0}`)
	w := r4post(a, "/api/layouts/build", fmt.Sprintf(`{"plan":{"Where":0,"Columns":[{"Folder":%q,"Type":"new","Value":"codex"},{"Folder":%q,"Type":"new","Value":"claude"}]}}`, a.root, a.root))
	if w.Code != 409 || !strings.Contains(w.Body.String(), "account guard") {
		t.Fatal("second agent bypassed account guard", w.Code, w.Body)
	}
}

func TestR4FailedScreenRetainsLastGoodRows(t *testing.T) {
	f := &fakeHerdr{status: "idle", screen: "last good screen"}
	a := testApp(f)
	a.refresh(context.Background(), true)
	f.screen = `{"result":{}}`
	a.refresh(context.Background(), true)
	if a.state.Panes[0].Screen != "last good screen" || a.state.Panes[0].Error == "" {
		t.Fatal("failed read erased cached screen", a.state.Panes[0])
	}
}

func TestR4SnapshotConfirmsPendingLaunch(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	f := &r4Herdr{}
	a := testApp(&f.fakeHerdr)
	a.herdr = f
	a.root = t.TempDir()
	a.profiles["codex"] = Profile{Kind: "codex"}
	r4post(a, "/api/launch/policy", `{"max_active":4,"min_ram_mb":0,"min_disk_gb":0}`)
	start := func() error {
		_, err := a.action(context.Background(), Action{Action: "start", Profile: "codex", Worktree: "test"})
		return err
	}
	if err := start(); err != nil {
		t.Fatal(err)
	}
	if err := start(); err == nil || err.Error() != "capacity" {
		t.Fatal("missing pending admission", err)
	}
	f.mu.Lock()
	f.snapshot = twoAgentSnapshot("codex", "one", "two", "three", "new")
	f.mu.Unlock()
	a.refresh(context.Background(), true)
	// After confirmation the agent exits. Its old pending slot must not survive.
	f.mu.Lock()
	f.snapshot = twoAgentSnapshot("codex", "one", "two", "three")
	f.mu.Unlock()
	a.refresh(context.Background(), true)
	if err := start(); err != nil {
		t.Fatal("confirmed pending slot leaked", err)
	}
}

func TestR4PendingLaunchExpiresAt120Seconds(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	root := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		f := &r4Herdr{}
		a := testApp(&f.fakeHerdr)
		a.herdr = f
		a.root = root
		a.profiles["codex"] = Profile{Kind: "codex"}
		r4post(a, "/api/launch/policy", `{"max_active":4,"min_ram_mb":0,"min_disk_gb":0}`)
		start := func() error {
			_, err := a.action(context.Background(), Action{Action: "start", Profile: "codex", Worktree: "test"})
			return err
		}
		if err := start(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(119 * time.Second)
		if err := start(); err == nil || err.Error() != "capacity" {
			t.Fatal("pending slot expired early", err)
		}
		time.Sleep(time.Second)
		if err := start(); err != nil {
			t.Fatal("pending slot did not expire at 120 seconds", err)
		}
	})
}

func TestR4ResultOverflowReconnectsForLedger(t *testing.T) {
	a := testApp(&fakeHerdr{})
	ch := make(chan []byte, 2)
	a.clients[ch] = true
	job := &Job{ID: "completed", Status: "done", OK: true, Output: "verified result"}
	a.jobs[job.ID] = job
	ch <- event("result", job)
	ch <- event("state", a.state)
	old := a.state
	next := old
	next.Panes = []Pane{{ID: "w1:p1", Key: "lab|w1:p1", Screen: "new output"}}
	a.publish(old, next)
	if a.clients[ch] {
		t.Fatal("overflow lost a result without forcing reconciliation")
	}
	w := request(a, "GET", "/api/jobs", "", "test-token", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"done"`) || !strings.Contains(w.Body.String(), "verified result") {
		t.Fatal("authoritative result lost", w.Code, w.Body)
	}
}
