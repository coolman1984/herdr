package deck

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// stubHerdr answers a fixed snapshot and screens, records writes like fakeHerdr, and runs scripts.
type stubHerdr struct {
	fakeHerdr
	snapshot string
	screens  map[string]string
}

func (f *stubHerdr) Call(ctx context.Context, args ...string) ([]byte, error) {
	if f.snapshot != "" && strings.Join(args, " ") == "api snapshot" {
		return []byte(f.snapshot), nil
	}
	if len(args) > 1 && args[0] == "pane" && args[1] == "read" {
		if f.screens != nil {
			return []byte(f.screens[args[2]]), nil
		}
		return []byte(f.screen), nil
	}
	if len(args) > 1 && args[0] == "agent" && args[1] == "list" {
		return []byte(`{"result":{"agents":[]}}`), nil
	}
	f.writes = append(f.writes, strings.Join(args, "|"))
	return []byte(`{"result":{"ok":true}}`), nil
}

func (f *stubHerdr) RunScript(context.Context, string, map[string]any, time.Duration) (string, error) {
	return "stub script output", nil
}

func stubApp(f *stubHerdr) *App {
	return newApp(f, "lab", "test-token", "http://127.0.0.1:9876", "I:\\herdr", map[string]Profile{})
}

func agentSnapshot(agent, name, cwd string) string {
	return fmt.Sprintf(`{"result":{"snapshot":{"workspaces":[],"tabs":[],"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1","workspace_id":"w1","cwd":%q,"agent":%q,"agent_status":"idle"}],"agents":[{"pane_id":"w1:p1","name":%q,"agent":%q,"agent_status":"idle"}]}}}`, cwd, agent, name, agent)
}

func twoAgentSnapshot(agent string, names ...string) string {
	panes, agents := []string{}, []string{}
	for i, name := range names {
		id := fmt.Sprintf("w1:p%d", i+1)
		panes = append(panes, fmt.Sprintf(`{"pane_id":%q,"tab_id":"w1:t1","workspace_id":"w1","agent":%q,"agent_status":"idle"}`, id, agent))
		agents = append(agents, fmt.Sprintf(`{"pane_id":%q,"name":%q,"agent":%q,"agent_status":"idle"}`, id, name, agent))
	}
	return fmt.Sprintf(`{"result":{"snapshot":{"workspaces":[],"tabs":[],"panes":[%s],"agents":[%s]}}}`, strings.Join(panes, ","), strings.Join(agents, ","))
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body %q: %v", w.Body.String(), err)
	}
	return body
}

func TestStatusErrorCarriesItsOwnStatusAndFields(t *testing.T) {
	w := httptest.NewRecorder()
	failHTTP(w, 502, &statusError{Status: 409, Msg: "held: nope", Fields: map[string]any{"error": "overridden", "extra": "yes"}})
	if w.Code != 409 {
		t.Fatalf("status ignored: %d", w.Code)
	}
	body := decodeBody(t, w)
	if body["error"] != "overridden" || body["extra"] != "yes" || body["kind"] == "" || body["next_step"] == "" {
		t.Fatalf("fields not merged: %v", body)
	}
	a := testApp(&fakeHerdr{})
	for _, v := range []struct {
		err  error
		want int
	}{{&statusError{Status: 409, Msg: "input_not_empty"}, 409}, {fmt.Errorf("blocked: needs the owner"), 409}, {fmt.Errorf("boom"), 502}} {
		w = httptest.NewRecorder()
		a.lockedWrite(w, func() (any, error) { return nil, v.err })
		if w.Code != v.want {
			t.Fatalf("%v -> %d, want %d", v.err, w.Code, v.want)
		}
	}
}

func TestReservationLifecycleConflictsAndReload(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	a := testApp(&fakeHerdr{})
	w := request(a, "POST", "/api/reservations", `{"task":"tasks/a.md","owner":"codex-1","paths":["app\\WEB\\App.js"]}`, "test-token", "")
	if w.Code != 200 {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	id, _ := body["id"].(string)
	if id == "" || body["ok"] != true {
		t.Fatalf("bad create envelope: %v", body)
	}
	created, _ := json.Marshal(body["reservations"])
	if !strings.Contains(string(created), `"app/web/app.js"`) {
		t.Fatalf("path not normalized: %s", created)
	}
	w = request(a, "POST", "/api/reservations", `{"task":"tasks/b.md","owner":"cline","paths":["app/web/app.js"]}`, "test-token", "")
	if w.Code != 409 {
		t.Fatalf("conflict not refused: %d", w.Code)
	}
	body = decodeBody(t, w)
	conflicts, _ := body["conflicts"].([]any)
	if body["error"] != "reservation conflict" || len(conflicts) != 1 {
		t.Fatalf("bad conflict body: %v", body)
	}
	pair, _ := conflicts[0].(map[string]any)
	if pair["path"] != "app/web/app.js" || pair["task"] != "tasks/a.md" || pair["owner"] != "codex-1" {
		t.Fatalf("bad conflict pair: %v", pair)
	}
	if w = request(a, "POST", "/api/reservations", `{"task":"tasks/c.md","owner":"agy","paths":["app/web/**"]}`, "test-token", ""); w.Code != 409 {
		t.Fatalf("subtree overlap not refused: %d", w.Code)
	}
	if w = request(a, "POST", "/api/reservations", `{"task":"tasks/d.md","owner":"agy","paths":["../outside"]}`, "test-token", ""); w.Code != 400 {
		t.Fatalf("traversal accepted: %d", w.Code)
	}
	if w = request(a, "POST", "/api/reservations", `{"task":"tasks/d.md","owner":"agy","paths":["app/*.js"]}`, "test-token", ""); w.Code != 400 {
		t.Fatalf("wildcard accepted: %d", w.Code)
	}
	w = request(a, "POST", "/api/reservations", `{"task":"tasks/a.md","owner":"codex-1","paths":["app/web/index.html"]}`, "test-token", "")
	if w.Code != 200 {
		t.Fatalf("same-task replace failed: %d %s", w.Code, w.Body.String())
	}
	body = decodeBody(t, w)
	if kept, _ := body["reservations"].([]any); len(kept) != 1 {
		t.Fatalf("same task did not replace: %v", body)
	}
	id, _ = body["id"].(string)
	b := testApp(&fakeHerdr{})
	w = request(b, "GET", "/api/reservations", "", "test-token", "")
	body = decodeBody(t, w)
	reloaded, _ := body["reservations"].([]any)
	if len(reloaded) != 1 || body["load_error"] != "" {
		t.Fatalf("reload lost the reservation: %v", body)
	}
	entry, _ := reloaded[0].(map[string]any)
	if entry["id"] != id || entry["task"] != "tasks/a.md" {
		t.Fatalf("reloaded entry differs: %v", entry)
	}
	if w = request(b, "POST", "/api/reservations/release", `{"id":"`+id+`"}`, "test-token", ""); w.Code != 200 {
		t.Fatalf("release failed: %d %s", w.Code, w.Body.String())
	}
	w = request(b, "POST", "/api/reservations/release", `{"id":"`+id+`"}`, "test-token", "")
	if w.Code != 404 || decodeBody(t, w)["error"] != "reservation not found" {
		t.Fatalf("unknown release not 404: %d %s", w.Code, w.Body.String())
	}
}

func TestNormalizeReservationPathNormalizesAndRefuses(t *testing.T) {
	root := `I:\herdr`
	for _, v := range []struct{ in, want string }{
		{`app\WEB\App.js`, "app/web/app.js"},
		{"./app/web/app.js", "app/web/app.js"},
		{"app/web/**", "app/web"},
		{"app/web/", "app/web"},
		{`I:\herdr\app\web\app.js`, "app/web/app.js"},
	} {
		got, err := normalizeReservationPath(root, v.in)
		if err != nil || got != v.want {
			t.Fatalf("%q -> %q, %v (want %q)", v.in, got, err, v.want)
		}
	}
	for _, in := range []string{"", "..", "../outside", "app/../../outside", "app/*.js", "app/web/**/x", `I:\other\app.js`} {
		if got, err := normalizeReservationPath(root, in); err == nil {
			t.Fatalf("%q accepted as %q", in, got)
		}
	}
	if !reservationOverlaps("app/web", "app/web/app.js") || !reservationOverlaps("app/web/app.js", "app/web") {
		t.Fatal("subtree overlap missed")
	}
	if reservationOverlaps("app/web", "app/webbing/app.js") {
		t.Fatal("prefix false positive")
	}
}

func TestClaudeAccountGuardRefusesTheLeadAndTheMainAlias(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.profiles["claude"] = Profile{Kind: "claude", DefaultAccount: "2030"}
	a.lead = LeadIdentity{Tool: "claude", Account: "2030", Known: true}
	for _, account := range []string{"", "main", "MAIN", "default", "2030"} {
		err := a.launchGuard(paneRequest{Profile: "claude", Account: account, Cwd: `C:\`})
		var detail *statusError
		if !errors.As(err, &detail) || detail.Status != 409 {
			t.Fatalf("%q admitted: %v", account, err)
		}
		if detail.Fields["lead_account"] != "2030" {
			t.Fatalf("lead account missing: %+v", detail)
		}
	}
	if err := a.launchGuard(paneRequest{Profile: "claude", Account: "1984", Cwd: `C:\`}); err != nil {
		t.Fatalf("another account refused: %v", err)
	}
	a.lead = LeadIdentity{Tool: "claude", Known: false}
	if err := a.launchGuard(paneRequest{Profile: "claude", Account: "1984", Cwd: `C:\`}); err == nil {
		t.Fatal("unknown lead admitted a Claude launch")
	}
}

func TestAccountAliasesMapOntoTheDefaultAccount(t *testing.T) {
	p := Profile{DefaultAccount: "2030"}
	for _, in := range []string{"", "main", "MAIN", " default "} {
		if got := normalizeAccount(p, in); got != "2030" {
			t.Fatalf("%q -> %q", in, got)
		}
	}
	if normalizeAccount(p, " 1984 ") != "1984" {
		t.Fatal("account not trimmed and lower-cased")
	}
	if mainAccountAlias(p, "main") != "" || mainAccountAlias(p, "2030") != "" {
		t.Fatal("the main sign-in must not pass an account folder")
	}
	if mainAccountAlias(p, "1984") != "1984" {
		t.Fatal("another account was dropped")
	}
}

const nextUsage = `account  5h     5h resets    weekly  week resets  session cost (updated today) data age
2030*    16%    Mon 11:50    11%     Fri 13:00    $6.72                        0 min
1984     0%     Sun 17:00    4%      Fri 13:00    $0.00                        1188 min
2040     0%     Fri 21:00    90%     Tue 07:00    $0.00                        3798 min
(* = the main sign-in; LIMIT = at or above 95% of the 5-hour or weekly limit)
NEXT: 1984`

func TestPickNextAccountSkipsTheLeadAndExhaustedAccounts(t *testing.T) {
	for _, v := range []struct{ lead, want string }{{"2030", "1984"}, {"1984", "2040"}, {"2040", "1984"}} {
		got, err := pickNextAccount(nextUsage, v.lead)
		if err != nil || got != v.want {
			t.Fatalf("lead %s -> %q, %v (want %q)", v.lead, got, err, v.want)
		}
	}
	busy := "2030*    97%    Mon 11:50    97%     Fri 13:00    $6.72    0 min\n1984     95%    Sun 17:00    95%     Fri 13:00    $0.00    1188 min\n"
	if _, err := pickNextAccount(busy, "2030"); err == nil {
		t.Fatal("an exhausted account set admitted a launch")
	}
	one := "1984     95%    Sun 17:00    4%      Fri 13:00    $0.00    1188 min\n2040     1%     Fri 21:00    90%     Tue 07:00    $0.00    3798 min\n"
	if got, err := pickNextAccount(one, "2030"); err != nil || got != "2040" {
		t.Fatalf("limit row not skipped: %q %v", got, err)
	}
	if _, err := pickNextAccount(nextUsage, ""); err == nil {
		t.Fatal("unknown lead admitted a next account")
	}
}

func TestHealthAddsRamFreeAlias(t *testing.T) {
	a := testApp(&fakeHerdr{})
	w := request(a, "GET", "/api/health", "", "test-token", "")
	if w.Code != 200 {
		t.Fatalf("health failed: %d", w.Code)
	}
	body := decodeBody(t, w)
	for _, field := range []string{"ram_used_pct", "ram_total_gb", "ram_available_mb", "ram_free_mb", "disks"} {
		if body[field] == nil {
			t.Fatalf("health lost %s: %v", field, body)
		}
	}
	if body["ram_free_mb"] != body["ram_available_mb"] {
		t.Fatalf("ram_free_mb differs: %v vs %v", body["ram_free_mb"], body["ram_available_mb"])
	}
}

func TestLaunchFloorRefusesBelowTheRequestedFloor(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.profiles["codex"] = Profile{Kind: "codex"}
	for _, v := range []struct{ body, field string }{
		{`{"profile":"codex","cwd":"C:\\","floor":{"ram_free_mb":1000000}}`, "ram_free_mb"},
		{`{"profile":"codex","cwd":"C:\\","floor":{"disk_free_gb":{"C:":1000000}}}`, "disk_free_gb"},
	} {
		var p paneRequest
		if err := json.Unmarshal([]byte(v.body), &p); err != nil {
			t.Fatalf("floor payload rejected: %v", err)
		}
		var detail *statusError
		if err := a.launchGuard(p); !errors.As(err, &detail) || detail.Status != 409 {
			t.Fatalf("%s -> %v", v.body, err)
		}
		if detail.Msg != "held: below the requested floor" || detail.Fields[v.field] == nil {
			t.Fatalf("bad floor refusal: %+v", detail)
		}
	}
	var p paneRequest
	if err := json.Unmarshal([]byte(`{"profile":"codex","cwd":"C:\\","floor":{"ram_free_mb":1}}`), &p); err != nil {
		t.Fatal(err)
	}
	if err := a.launchGuard(p); err != nil {
		t.Fatalf("a floor above the free space must pass: %v", err)
	}
}

func TestHeadlessRoutesStayEmptyAndResearchIsRefused(t *testing.T) {
	a := testApp(&fakeHerdr{})
	w := request(a, "GET", "/api/headless/routes", "", "test-token", "")
	if w.Code != 200 {
		t.Fatalf("routes failed: %d %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	routes, ok := body["routes"].([]any)
	note, _ := body["note"].(string)
	if !ok || len(routes) != 0 || !strings.Contains(note, "unavailable") {
		t.Fatalf("research routes claimed: %v", body)
	}
	w = request(a, "POST", "/api/headless", `{"profile":"cline-ds","text":"x","purpose":"research"}`, "test-token", "")
	if w.Code != 409 || decodeBody(t, w)["error"] != "research route unavailable: no verified server-search route" {
		t.Fatalf("research headless not refused: %d %s", w.Code, w.Body.String())
	}
}

// deliveryHerdr fails agent prompts on demand and counts the calls.
type deliveryHerdr struct {
	fakeHerdr
	fail  error
	calls int
}

func (f *deliveryHerdr) Call(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) > 1 && args[0] == "agent" && args[1] == "prompt" {
		f.calls++
		if f.fail != nil {
			return nil, f.fail
		}
		f.writes = append(f.writes, strings.Join(args, "|"))
		return []byte(`{"result":{"ok":true}}`), nil
	}
	return f.fakeHerdr.Call(ctx, args...)
}

func deliveryApp(f *deliveryHerdr) *App {
	return newApp(f, "lab", "test-token", "http://127.0.0.1:9876", "I:\\herdr", map[string]Profile{})
}

func TestPromptDispatchIDAndDispatchLedger(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	f := &deliveryHerdr{}
	a := deliveryApp(f)
	w := request(a, "POST", "/api/pane/prompt", `{"key":"lab|w1:p1","text":"do it"}`, "test-token", "")
	if w.Code != 200 {
		t.Fatalf("prompt failed: %d %s", w.Code, w.Body.String())
	}
	dispatch, _ := decodeBody(t, w)["dispatch_id"].(string)
	if dispatch == "" || f.calls != 1 {
		t.Fatalf("missing dispatch id or extra calls: %s %d", dispatch, f.calls)
	}
	w = request(a, "GET", "/api/dispatches?key=lab%7Cw1:p1", "", "test-token", "")
	list, _ := decodeBody(t, w)["dispatches"].([]any)
	if len(list) != 1 {
		t.Fatalf("dispatch ledger missing: %v", w.Body.String())
	}
	entry, _ := list[0].(map[string]any)
	if entry["state"] != "sent" || entry["dispatch_id"] != dispatch || entry["key"] != "lab|w1:p1" || entry["text_sha"] == "" {
		t.Fatalf("bad dispatch record: %v", entry)
	}
	w = request(a, "GET", "/api/deliveries", "", "test-token", "")
	if w.Code != 200 {
		t.Fatalf("/api/deliveries broke: %d %s", w.Code, w.Body.String())
	}
	if len(decodeBody(t, w)["deliveries"].([]any)) != 1 {
		t.Fatal("/api/deliveries lost the record")
	}
}

func TestUncertainDeliveryIs504AndRejectedDeliveryIs502(t *testing.T) {
	for _, v := range []struct {
		fail  error
		want  int
		state string
	}{
		{fmt.Errorf("context deadline exceeded"), 504, "unknown"},
		{fmt.Errorf("agent_not_ready: nothing to prompt"), 502, "failed"},
	} {
		t.Setenv("APPDATA", t.TempDir())
		f := &deliveryHerdr{fail: v.fail}
		a := deliveryApp(f)
		w := request(a, "POST", "/api/pane/prompt", `{"key":"lab|w1:p1","text":"do it"}`, "test-token", "")
		if w.Code != v.want {
			t.Fatalf("%v -> %d, want %d (%s)", v.fail, w.Code, v.want, w.Body.String())
		}
		body := decodeBody(t, w)
		if body["dispatch_id"] == nil || f.calls != 1 {
			t.Fatalf("no dispatch id or a retry: %v %d", body, f.calls)
		}
		w = request(a, "GET", "/api/dispatches", "", "test-token", "")
		list, _ := decodeBody(t, w)["dispatches"].([]any)
		if len(list) != 1 {
			t.Fatalf("%v -> %v", v.fail, list)
		}
		entry, _ := list[0].(map[string]any)
		if entry["state"] != v.state {
			t.Fatalf("%v -> %v", v.fail, entry)
		}
	}
}

func TestParsePorcelainZReadsRenamesAndSpaces(t *testing.T) {
	raw := " M app/web/app.js\x00R  new name.txt\x00old name.txt\x00?? app\\web\\new.js\x00"
	got := parsePorcelainZ([]byte(raw))
	want := []map[string]string{{"path": "app/web/app.js", "code": " M"}, {"path": "new name.txt", "code": "R "}, {"path": "app/web/new.js", "code": "??"}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i]["path"] != want[i]["path"] || got[i]["code"] != want[i]["code"] {
			t.Fatalf("%d: %v, want %v", i, got[i], want[i])
		}
	}
	if len(parsePorcelainZ(nil)) != 0 {
		t.Fatal("empty porcelain is not empty")
	}
}

func TestGitEndpointReportsRootAndPorcelainPaths(t *testing.T) {
	git, err := exec.LookPath("git.exe")
	if err != nil {
		t.Skip("git.exe is unavailable")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		out, err := exec.Command(git, append([]string{"-C", repo}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "deck@test.invalid")
	run("config", "user.name", "Deck Test")
	if err := os.WriteFile(filepath.Join(repo, "old name.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-qm", "one")
	run("mv", "old name.txt", "new name.txt")
	f := &stubHerdr{snapshot: agentSnapshot("codex", "coder", repo)}
	w := request(stubApp(f), "GET", "/api/git?key=lab%7Cw1:p1", "", "test-token", "")
	if w.Code != 200 {
		t.Fatalf("git failed: %d %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	for _, field := range []string{"status", "stat", "text"} {
		if body[field] == nil {
			t.Fatalf("git lost %s: %v", field, body)
		}
	}
	root, _ := body["root"].(string)
	if !strings.EqualFold(root, filepath.ToSlash(repo)) || strings.Contains(root, `\`) {
		t.Fatalf("bad root %q for %q", root, repo)
	}
	paths, _ := body["paths"].([]any)
	if len(paths) != 1 {
		t.Fatalf("bad paths: %v", body["paths"])
	}
	entry, _ := paths[0].(map[string]any)
	if entry["path"] != "new name.txt" || entry["code"] != "R " {
		t.Fatalf("bad path entry: %v", entry)
	}
}

func TestStaticHandlerCachesFontsAndAssets(t *testing.T) {
	files := fstest.MapFS{
		"fonts/Alexandria-latin.woff2": {Data: []byte("font")},
		"assets/app.css":               {Data: []byte("css")},
		"app.js":                       {Data: []byte("page")},
	}
	handler := staticHandler(files)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:9876/fonts/Alexandria-latin.woff2", nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "font/woff2" || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("font not cached: %d %v", w.Code, w.Header())
	}
	tag := w.Header().Get("ETag")
	if tag == "" {
		t.Fatal("font has no ETag")
	}
	req := httptest.NewRequest("GET", "http://127.0.0.1:9876/fonts/Alexandria-latin.woff2", nil)
	req.Header.Set("If-None-Match", tag)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 304 {
		t.Fatalf("matching ETag not 304: %d", w.Code)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:9876/assets/app.css", nil))
	if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset not cached: %d %v", w.Code, w.Header())
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:9876/fonts/missing.woff2", nil))
	if w.Code != 404 {
		t.Fatalf("missing font is not 404: %d", w.Code)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:9876/app.js", nil))
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("the page must be revalidated on every load: %d %v", w.Code, w.Header())
	}
}
func TestStaticHandlerGzipsPageCodeAndRevalidates(t *testing.T) {
	code := strings.Repeat("export const x = 1;\n", 200)
	handler := staticHandler(fstest.MapFS{"app.js": {Data: []byte(code)}, "index.html": {Data: []byte("<!doctype html>")}})
	req := httptest.NewRequest("GET", "http://127.0.0.1:9876/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 || w.Header().Get("Content-Encoding") != "gzip" || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("app.js not gzipped: %d %v", w.Code, w.Header())
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(zr)
	if err != nil || string(plain) != code || w.Body.Len() >= len(code) {
		t.Fatalf("gzip body wrong: err=%v plain=%d gz=%d", err, len(plain), w.Body.Len())
	}
	tag := w.Header().Get("ETag")
	req = httptest.NewRequest("GET", "http://127.0.0.1:9876/app.js", nil)
	req.Header.Set("If-None-Match", tag)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if tag == "" || w.Code != 304 {
		t.Fatalf("revalidation not 304: tag=%q code=%d", tag, w.Code)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:9876/", nil))
	if w.Code != 200 || w.Body.String() != "<!doctype html>" || w.Header().Get("Content-Encoding") != "" {
		t.Fatalf("index without gzip: %d %q %v", w.Code, w.Body.String(), w.Header())
	}
}

func TestHerdrStartTargetsTheNamedSession(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.sessionName, a.herdrBin = "lab", `I:\tools\herdr.exe`
	var calls [][]string
	a.startDetached = func(name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}
	w := request(a, "POST", "/api/herdr/start", `{}`, "test-token", "")
	if w.Code != 200 {
		t.Fatalf("start failed: %d %s", w.Code, w.Body.String())
	}
	if len(calls) != 1 || strings.Join(calls[0], " ") != `I:\tools\herdr.exe --session lab server` {
		t.Fatalf("wrong start: %v", calls)
	}
	if message, _ := decodeBody(t, w)["message"].(string); !strings.Contains(message, "lab") {
		t.Fatalf("message does not name the session: %q", message)
	}
	b := testApp(&fakeHerdr{})
	var launched []string
	b.startDetached = func(name string, args ...string) error {
		launched = append(launched, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	if w = request(b, "POST", "/api/herdr/start", `{}`, "test-token", ""); w.Code != 200 {
		t.Fatalf("launcher start failed: %d %s", w.Code, w.Body.String())
	}
	if len(launched) != 1 || !strings.HasSuffix(launched[0], `launcher\HerdrLauncher.exe`) {
		t.Fatalf("the default session must keep the launcher: %v", launched)
	}
}
func TestStateReportsTheToolkitRoot(t *testing.T) {
	a := testApp(&fakeHerdr{})
	w := request(a, "GET", "/api/state", "", "test-token", "")
	var st struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil || w.Code != 200 || st.Root != `I:\herdr` {
		t.Fatalf("state root: %d %q err=%v body=%s", w.Code, st.Root, err, w.Body.String())
	}
}
func TestDeckWindowSkipsTheSystemProxy(t *testing.T) {
	args := windowArgs("http://127.0.0.1:9876/?t=x", `C:\p`)
	if !strings.Contains(strings.Join(args, " "), "--no-proxy-server") || args[0] != "--app=http://127.0.0.1:9876/?t=x" {
		t.Fatalf("window args: %v", args)
	}
}
