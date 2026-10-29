package deck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeHerdr struct {
	status, screen string
	offline        bool
	shell          bool // the agent has exited: the pane is a bare shell
	writes         []string
}

func (f *fakeHerdr) Call(ctx context.Context, args ...string) ([]byte, error) {
	if f.offline {
		return nil, fmt.Errorf("server_not_running")
	}
	if strings.Join(args, " ") == "api snapshot" && f.shell {
		return []byte(`{"result":{"snapshot":{"workspaces":[],"tabs":[],"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1","workspace_id":"w1"}],"agents":[]}}}`), nil
	}
	if strings.Join(args, " ") == "api snapshot" {
		raw := fmt.Sprintf(`{"result":{"snapshot":{"workspaces":[],"tabs":[],"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1","workspace_id":"w1","agent":"codex","agent_status":%q}],"agents":[{"pane_id":"w1:p1","name":"coder","agent":"codex","agent_status":%q}]}}}`, f.status, f.status)
		return []byte(raw), nil
	}
	if len(args) > 1 && args[0] == "agent" && args[1] == "list" {
		return []byte(fmt.Sprintf(`{"result":{"agents":[{"pane_id":"w1:p1","name":"coder","agent_status":%q}]}}`, f.status)), nil
	}
	if len(args) > 1 && args[0] == "pane" && args[1] == "read" {
		return []byte(f.screen), nil
	}
	f.writes = append(f.writes, strings.Join(args, "|"))
	return []byte(`{"result":{"ok":true}}`), nil
}
func (f *fakeHerdr) Helper(context.Context, map[string]string) (string, error) {
	return "helper result", nil
}
func testApp(f *fakeHerdr) *App {
	a := newApp(f, "lab", "test-token", "http://127.0.0.1:9876", "I:\\herdr", map[string]Profile{})
	// The fake page shows its one pane; production visibility comes from /api/visible.
	a.visible["w1:p1"] = true
	return a
}
func request(a *App, method, path, body, token, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://127.0.0.1:9876"+path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("X-Deck-Token", token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	return w
}
func TestHTTPRejectsMissingTokenForeignOriginAndHost(t *testing.T) {
	a := testApp(&fakeHerdr{})
	for _, v := range []struct{ token, origin string }{{"", ""}, {"wrong", ""}, {"test-token", "https://bad.example"}} {
		if w := request(a, "GET", "/api/state", "", v.token, v.origin); w.Code != 403 {
			t.Fatalf("accepted %+v: %d", v, w.Code)
		}
	}
	req := httptest.NewRequest("GET", "http://evil.example/api/state?t=test-token", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("host not checked")
	}
}
func TestCookieBootstrapAndAssetAuthentication(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.files = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("asset")) })
	w := request(a, "GET", "/?t=test-token", "", "", "")
	if len(w.Result().Cookies()) != 1 {
		t.Fatal("no token cookie")
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe cookie")
	}
	req := httptest.NewRequest("GET", "http://127.0.0.1:9876/app.js", nil)
	req.AddCookie(cookie)
	out := httptest.NewRecorder()
	a.ServeHTTP(out, req)
	if out.Code != 200 || out.Body.String() != "asset" {
		t.Fatal("module cookie rejected")
	}
}
func TestRefreshDisplaysBlockedAndRecoversFromOffline(t *testing.T) {
	f := &fakeHerdr{status: "idle", screen: "Trust this folder?\nEnter to confirm · Esc to cancel"}
	a := testApp(f)
	a.refresh(context.Background(), true)
	if len(a.state.Panes) != 1 || !a.state.Panes[0].Blocked || a.state.Panes[0].Status != "blocked" {
		t.Fatalf("missing guard: %+v", a.state)
	}
	if len(f.writes) != 0 {
		t.Fatal("polling answered a dialog")
	}
	f.offline = true
	a.refresh(context.Background(), true)
	if a.state.Herdr.Running || len(a.state.Panes) != 1 || a.state.Error == "" {
		t.Fatal("offline was hidden")
	}
	f.offline = false
	f.screen = "ready"
	a.refresh(context.Background(), true)
	if !a.state.Herdr.Running || a.state.Error != "" || len(a.state.Panes) != 1 {
		t.Fatal("did not recover")
	}
}
func TestPromptGuardRejectsBlockedBeforeAnyWrite(t *testing.T) {
	f := &fakeHerdr{status: "blocked", screen: "approval"}
	a := testApp(f)
	w := request(a, "POST", "/api/action", `{"action":"prompt","pane_id":"w1:p1","text":"do it"}`, "test-token", "")
	if w.Code != 409 || len(f.writes) != 0 {
		t.Fatalf("approval answered: %d %+v", w.Code, f.writes)
	}
	f.status = "idle"
	f.screen = "Please sign in"
	w = request(a, "POST", "/api/action", `{"action":"prompt","pane_id":"w1:p1","text":"do it"}`, "test-token", "")
	if w.Code != 409 || len(f.writes) != 0 {
		t.Fatal("signin answered")
	}
}
func TestPromptToExitedAgentNeverRunsInTheShell(t *testing.T) {
	f := &fakeHerdr{shell: true, screen: "PS D:\\WORK> "}
	a := testApp(f)
	w := request(a, "POST", "/api/pane/prompt", `{"key":"lab|w1:p1","text":"make text reflow [see screenshot]"}`, "test-token", "")
	if w.Code != 409 || !strings.Contains(w.Body.String(), "agent_exited") || len(f.writes) != 0 {
		t.Fatalf("agent text reached the shell: %d %s %+v", w.Code, w.Body.String(), f.writes)
	}
	w = request(a, "POST", "/api/pane/prompt", `{"key":"lab|w1:p1","text":"git status","expect":"shell"}`, "test-token", "")
	if w.Code != 200 || len(f.writes) != 1 || f.writes[0] != "pane|run|w1:p1|git status" {
		t.Fatalf("shell card command not run: %d %s %+v", w.Code, w.Body.String(), f.writes)
	}
	// The agent's last dialog is still on screen: the reason is still "agent exited", not "blocked".
	f.screen = "Do you trust this folder?\nenter to confirm  esc to cancel\nPS D:\\WORK> "
	w = request(a, "POST", "/api/pane/prompt", `{"key":"lab|w1:p1","text":"hello"}`, "test-token", "")
	if w.Code != 409 || !strings.Contains(w.Body.String(), "agent_exited") || len(f.writes) != 1 {
		t.Fatalf("old dialog hid the exited agent: %d %s %+v", w.Code, w.Body.String(), f.writes)
	}
}
func TestPromptPreservesQuotesNewlinesAndMetacharacters(t *testing.T) {
	f := &fakeHerdr{status: "idle", screen: "› "}
	a := testApp(f)
	text := "hello \"world\"\n$(never execute); C:\\folder\\"
	_, err := a.action(context.Background(), Action{Action: "prompt", Pane: "w1:p1", Text: text})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.writes) != 1 || f.writes[0] != "agent|prompt|coder|"+text {
		t.Fatalf("prompt changed: %+v", f.writes)
	}
}
func TestBlockedScreenStaysBlockedOnMetadataOnlyPoll(t *testing.T) {
	f := &fakeHerdr{status: "idle", screen: "Trust this folder?\nEnter to confirm · Esc to cancel"}
	a := testApp(f)
	a.refresh(context.Background(), true)
	if !a.state.Panes[0].Blocked {
		t.Fatal("setup missed dialog")
	}
	f.status = "unknown"
	a.metadataAt = time.Now().Add(-3 * time.Second)
	a.lastRead["lab|w1:p1"] = time.Now()
	a.refresh(context.Background(), false)
	if !a.state.Panes[0].Blocked || a.state.Panes[0].Status != "blocked" {
		t.Fatal("metadata-only poll cleared a still-visible dialog")
	}
}
func TestReadOnlyDisablesMutationsAndKeysRequireKnownPane(t *testing.T) {
	f := &fakeHerdr{status: "idle"}
	a := testApp(f)
	request(a, "POST", "/api/read-only", `{"enabled":true}`, "test-token", "")
	w := request(a, "POST", "/api/action", `{"action":"keys","pane_id":"w1:p1","key":"enter"}`, "test-token", "")
	if w.Code != 403 || len(f.writes) != 0 {
		t.Fatal("read-only ignored")
	}
	if _, err := a.action(context.Background(), Action{Action: "keys", Pane: "other-session|w1:p1", Key: "enter"}); err == nil {
		t.Fatal("foreign pane accepted")
	}
	if _, err := a.action(context.Background(), Action{Action: "keys", Pane: "w1:p1", Key: "ctrl+b; stop"}); err == nil {
		t.Fatal("invalid key accepted")
	}
}
func TestDiffOnlyContainsChangedCardsAndRemovedKeys(t *testing.T) {
	a := testApp(&fakeHerdr{})
	ch := make(chan []byte, 16)
	a.clients[ch] = true
	old := a.state
	next := old
	next.Panes = []Pane{{ID: "w1:p1", Key: "lab|w1:p1", Screen: "one"}}
	a.publish(old, next)
	var last []byte
	for len(ch) > 0 {
		last = <-ch
	}
	if !strings.Contains(string(last), `"screen":"one"`) {
		t.Fatalf("missing screen: %s", last)
	}
	old = a.state
	next = old
	next.Panes = []Pane{}
	a.publish(old, next)
	last = <-ch
	if !strings.Contains(string(last), `"removed":["lab|w1:p1"]`) {
		t.Fatalf("missing deletion: %s", last)
	}
	old = a.state
	a.publish(old, old)
	if len(ch) != 0 {
		t.Fatal("unchanged state emitted event")
	}
}
func TestLifecycleGraceAndDisconnect(t *testing.T) {
	a := testApp(&fakeHerdr{})
	if a.shouldExit(a.started.Add(5 * time.Second)) {
		t.Fatal("startup too short")
	}
	if !a.shouldExit(a.started.Add(16 * time.Second)) {
		t.Fatal("startup never ends")
	}
	a.seen = true
	a.lastClient = time.Now()
	if a.shouldExit(a.lastClient.Add(2 * time.Second)) {
		t.Fatal("reconnect grace missing")
	}
	if !a.shouldExit(a.lastClient.Add(4 * time.Second)) {
		t.Fatal("disconnect did not exit")
	}
}
func TestDedicatedPaneRouteValidatesSessionAndReadOnly(t *testing.T) {
	f := &fakeHerdr{status: "idle"}
	a := testApp(f)
	w := request(a, "POST", "/api/pane/text", `{"key":"wrong|w1:p1","text":"x"}`, "test-token", "")
	if w.Code == 200 || len(f.writes) != 0 {
		t.Fatal("cross-session key accepted")
	}
	w = request(a, "POST", "/api/pane/keys", `{"key":"lab|w1:p1","keys":["Escape"]}`, "test-token", "")
	if w.Code != 200 || len(f.writes) != 1 {
		t.Fatalf("dedicated keys failed: %d %s", w.Code, w.Body.String())
	}
	var out map[string]any
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || out["ok"] != true {
		t.Fatal("bad result envelope")
	}
	request(a, "POST", "/api/read-only", `{"enabled":true}`, "test-token", "")
	w = request(a, "POST", "/api/pane/text", `{"key":"lab|w1:p1","text":"x"}`, "test-token", "")
	if w.Code != 403 {
		t.Fatal("alias bypassed read-only")
	}
}
