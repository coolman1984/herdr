package deck

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

type App struct {
	herdr                           Herdr
	token, origin, root             string
	profiles                        map[string]Profile
	files                           http.Handler
	reopen                          func() error
	mu                              sync.Mutex
	refreshMu                       sync.Mutex
	actionMu                        sync.Mutex
	state                           State
	clients                         map[chan []byte]bool
	visible                         map[string]bool
	active, readOnly, seen          bool
	started, lastClient, metadataAt time.Time
	jobs                            map[string]*Job
	lastRead, lastChange            map[string]time.Time
	debug                           bool
	guards                          *GuardState
}

func newApp(h Herdr, session, token, origin, root string, profiles map[string]Profile) *App {
	return &App{herdr: h, token: token, origin: origin, root: root, profiles: profiles, state: State{Session: session, Panes: []Pane{}, Workspaces: []Nav{}, Tabs: []Nav{}}, clients: map[chan []byte]bool{}, visible: map[string]bool{}, active: true, started: time.Now(), lastClient: time.Now(), jobs: map[string]*Job{}}
}
func (a *App) shouldExit(now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.clients) > 0 {
		return false
	}
	if !a.seen {
		return now.Sub(a.started) > 15*time.Second
	}
	return now.Sub(a.lastClient) > 3*time.Second
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func failHTTP(w http.ResponseWriter, status int, err error) {
	kind, next := recovery(err.Error())
	respond(w, status, map[string]string{"error": err.Error(), "kind": kind, "next_step": next, "cause": "unknown unless stated in the observed error"})
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; object-src 'none'; frame-ancestors 'none'; base-uri 'none'")
	token := r.URL.Query().Get("t")
	if token == "" {
		token = r.Header.Get("X-Deck-Token")
	}
	if token == "" {
		token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if token == "" {
		if cookie, err := r.Cookie("deck-token"); err == nil {
			token = cookie.Value
		}
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) != 1 || r.Host != strings.TrimPrefix(a.origin, "http://") {
		failHTTP(w, 403, fmt.Errorf("Deck access token or host is invalid"))
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != a.origin {
		failHTTP(w, 403, fmt.Errorf("cross-origin requests are not allowed"))
		return
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		failHTTP(w, 403, fmt.Errorf("cross-site requests are not allowed"))
		return
	}
	if r.URL.Path == "/" && r.URL.Query().Get("t") == a.token {
		http.SetCookie(w, &http.Cookie{Name: "deck-token", Value: a.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	}
	if r.Method != "GET" && r.Method != "POST" {
		failHTTP(w, 405, fmt.Errorf("method not allowed"))
		return
	}
	if r.Method == "POST" && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		failHTTP(w, 415, fmt.Errorf("JSON body required"))
		return
	}
	switch r.URL.Path {
	case "/debug/pprof/", "/debug/pprof/goroutine", "/debug/pprof/heap", "/debug/pprof/profile":
		if !a.debug {
			failHTTP(w, 404, fmt.Errorf("debug profiling is disabled"))
			return
		}
		if r.URL.Path == "/debug/pprof/profile" {
			pprof.Profile(w, r)
		} else {
			pprof.Index(w, r)
		}
	case "/api/state":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return
		}
		a.mu.Lock()
		state := a.state
		a.mu.Unlock()
		respond(w, 200, state)
	case "/api/events":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return
		}
		a.events(w, r)
	case "/api/visibility":
		var v struct {
			Visible []string `json:"visible"`
			Active  bool     `json:"active"`
		}
		if !readBody(w, r, &v) {
			return
		}
		if len(v.Visible) > 256 {
			failHTTP(w, 400, fmt.Errorf("too many visible panes"))
			return
		}
		a.mu.Lock()
		a.active = v.Active
		a.visible = map[string]bool{}
		for _, id := range v.Visible {
			a.visible[id] = true
		}
		a.mu.Unlock()
		respond(w, 200, map[string]bool{"ok": true})
	case "/api/read-only":
		var v struct {
			Enabled bool `json:"enabled"`
		}
		if !readBody(w, r, &v) {
			return
		}
		a.actionMu.Lock()
		a.mu.Lock()
		a.readOnly = v.Enabled
		a.mu.Unlock()
		a.actionMu.Unlock()
		respond(w, 200, map[string]bool{"enabled": v.Enabled})
	case "/api/action":
		var v Action
		if !readBody(w, r, &v) {
			return
		}
		a.actionMu.Lock()
		defer a.actionMu.Unlock()
		a.mu.Lock()
		readOnly := a.readOnly
		a.mu.Unlock()
		if readOnly {
			failHTTP(w, 403, fmt.Errorf("read-only mode is enabled"))
			return
		}
		out, err := a.action(r.Context(), v)
		if err != nil {
			status := 502
			if strings.HasPrefix(err.Error(), "blocked:") {
				status = 409
			}
			failHTTP(w, status, err)
			return
		}
		respond(w, 200, map[string]any{"ok": true, "output": out, "message": out})
		go a.refresh(context.Background(), true)
	case "/api/profiles":
		list := []Profile{}
		for _, p := range a.profiles {
			list = append(list, p)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		respond(w, 200, map[string]any{"profiles": list})
	case "/api/usage", "/api/accounts":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return
		}
		out, err := a.herdr.Helper(r.Context(), map[string]string{"Command": strings.TrimPrefix(r.URL.Path, "/api/")})
		if err != nil {
			failHTTP(w, 502, err)
			return
		}
		respond(w, 200, map[string]string{"output": out, "text": out})
	case "/api/reopen":
		var v struct{}
		if !readBody(w, r, &v) {
			return
		}
		if err := a.reopen(); err != nil {
			failHTTP(w, 502, err)
			return
		}
		respond(w, 200, map[string]bool{"ok": true})
	default:
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if a.extensions(w, r) {
				return
			}
			failHTTP(w, 404, fmt.Errorf("unknown Deck endpoint"))
			return
		}
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return
		}
		if a.files == nil {
			failHTTP(w, 404, fmt.Errorf("web assets unavailable"))
			return
		}
		a.files.ServeHTTP(w, r)
	}
}
func readBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != "POST" {
		failHTTP(w, 405, fmt.Errorf("POST required"))
		return false
	}
	reader := http.MaxBytesReader(w, r.Body, 128<<10)
	d := json.NewDecoder(reader)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		failHTTP(w, 400, fmt.Errorf("invalid JSON: %w", err))
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		failHTTP(w, 400, fmt.Errorf("only one JSON value allowed"))
		return false
	}
	return true
}
func event(name string, v any) []byte {
	data, _ := json.Marshal(v)
	return []byte("event: " + name + "\ndata: " + string(data) + "\n\n")
}
func (a *App) events(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		failHTTP(w, 500, fmt.Errorf("streaming unavailable"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	ch := make(chan []byte, 2)
	a.mu.Lock()
	if len(a.clients) >= 4 {
		a.mu.Unlock()
		failHTTP(w, 503, fmt.Errorf("four Deck clients already connected"))
		return
	}
	a.clients[ch] = true
	a.seen = true
	a.lastClient = time.Now()
	initial := event("state", a.state)
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.clients, ch); a.lastClient = time.Now(); a.mu.Unlock() }()
	if _, err := w.Write(initial); err != nil {
		return
	}
	f.Flush()
	go a.refresh(r.Context(), true)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			http.NewResponseController(w).SetWriteDeadline(time.Now().Add(3 * time.Second))
			if _, err := w.Write(msg); err != nil {
				return
			}
			f.Flush()
		case <-ticker.C:
			http.NewResponseController(w).SetWriteDeadline(time.Now().Add(3 * time.Second))
			if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}
			f.Flush()
		}
	}
}
func (a *App) poll(ctx context.Context) {
	ticker := time.NewTicker(280 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			active := a.active && len(a.clients) > 0
			a.mu.Unlock()
			if active {
				a.refresh(ctx, false)
			}
		}
	}
}
func (a *App) refresh(parent context.Context, force bool) {
	if !a.refreshMu.TryLock() {
		return
	}
	defer a.refreshMu.Unlock()
	ctx, cancel := context.WithTimeout(parent, 7*time.Second)
	defer cancel()
	a.mu.Lock()
	old := a.state
	meta := force || time.Since(a.metadataAt) > 2*time.Second
	visible := map[string]bool{}
	for k, v := range a.visible {
		visible[k] = v
	}
	a.mu.Unlock()
	next := old
	if meta {
		raw, err := a.herdr.Call(ctx, "api", "snapshot")
		if err == nil {
			next, err = parseSnapshot(raw, old.Session)
		}
		if err != nil {
			next = State{Session: old.Session, Panes: []Pane{}, Workspaces: []Nav{}, Tabs: []Nav{}, Error: err.Error()}
			next.Herdr.Error = err.Error()
			a.publish(old, next)
			return
		}
		// Snapshot already carries agents; list checks the dedicated surface and catches divergent errors.
		raw, err = a.herdr.Call(ctx, "agent", "list")
		if err != nil {
			next.Error = err.Error()
		} else {
			var list struct {
				Result struct {
					Agents []Pane `json:"agents"`
				} `json:"result"`
			}
			if err = json.Unmarshal(raw, &list); err != nil {
				next.Error = "invalid agent list: " + err.Error()
			} else {
				for i := range next.Panes {
					for _, agent := range list.Result.Agents {
						if agent.ID == next.Panes[i].ID {
							next.Panes[i].Name = agent.Name
							next.Panes[i].Status = agent.Status
						}
					}
				}
			}
		}
		a.mu.Lock()
		a.metadataAt = time.Now()
		a.mu.Unlock()
	} else {
		next.Panes = append([]Pane{}, old.Panes...)
	}
	cache := map[string]Pane{}
	if a.lastRead == nil {
		a.lastRead = map[string]time.Time{}
		a.lastChange = map[string]time.Time{}
	}
	for _, p := range old.Panes {
		cache[p.Key] = p
	}
	for i := range next.Panes {
		p := &next.Panes[i]
		previous := cache[p.Key]
		p.Screen = previous.Screen
		p.ScreenVersion = previous.ScreenVersion
		p.Error = previous.Error
		p.Blocked = p.Status == "blocked" || blockedScreen(p.Screen)
		if p.Blocked {
			p.Status = "blocked"
		}
		interval := 7 * time.Second
		if visible[p.ID] {
			interval = 1400 * time.Millisecond
		}
		if p.Focused && time.Since(a.lastChange[p.Key]) < 2*time.Second {
			interval = 280 * time.Millisecond
		}
		if !force && time.Since(a.lastRead[p.Key]) < interval {
			continue
		}
		rows := p.Scroll.Rows
		if rows < 1 {
			rows = 40
		}
		if rows > 500 {
			rows = 500
		}
		raw, err := a.herdr.Call(ctx, "pane", "read", p.ID, "--source", "recent", "--lines", fmt.Sprint(rows+2000), "--format", "ansi")
		if err == nil {
			p.Screen, err = parseScreen(raw, rows)
			if p.Screen != previous.Screen {
				p.ScreenVersion++
				a.lastChange[p.Key] = time.Now()
			}
			a.lastRead[p.Key] = time.Now()
		}
		if err != nil {
			p.Error = err.Error()
		} else {
			p.Error = ""
			if blockedScreen(p.Screen) {
				p.Status = "blocked"
			}
			p.Blocked = p.Status == "blocked"
		}
	}
	alive := map[string]bool{}
	for _, p := range next.Panes {
		alive[p.Key] = true
	}
	for key := range a.lastRead {
		if !alive[key] {
			delete(a.lastRead, key)
			delete(a.lastChange, key)
		}
	}
	a.publish(old, next)
}
func (a *App) publish(old, next State) {
	next.Profiles = []Profile{}
	for _, p := range a.profiles {
		next.Profiles = append(next.Profiles, p)
	}
	sort.Slice(next.Profiles, func(i, j int) bool { return next.Profiles[i].Name < next.Profiles[j].Name })
	diff := map[string]any{"panes": []Pane{}, "removed": []string{}, "error": next.Error}
	prior := map[string]Pane{}
	for _, p := range old.Panes {
		prior[p.Key] = p
	}
	changed := []PaneUpdate{}
	for i, p := range next.Panes {
		if !reflect.DeepEqual(p, prior[p.Key]) {
			update := PaneUpdate{Pane: p}
			previous, exists := prior[p.Key]
			if !exists {
				screen := p.Screen
				update.Screen = &screen
			} else if p.Screen != previous.Screen {
				patches := rowChanges(previous.Screen, p.Screen, previous.ScreenVersion)
				for j := range patches {
					item := update
					item.Patch = &patches[j]
					item.ScreenVersion = patches[j].Version
					changed = append(changed, item)
				}
				next.Panes[i].ScreenVersion = patches[len(patches)-1].Version
				delete(prior, p.Key)
				continue
			}
			changed = append(changed, update)
		}
		delete(prior, p.Key)
	}
	removed := []string{}
	for k := range prior {
		removed = append(removed, k)
	}
	sort.Strings(removed)
	diff["panes"] = changed
	diff["removed"] = removed
	if !reflect.DeepEqual(old.Workspaces, next.Workspaces) {
		diff["workspaces"] = next.Workspaces
	}
	if !reflect.DeepEqual(old.Tabs, next.Tabs) {
		diff["tabs"] = next.Tabs
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = next
	if reflect.DeepEqual(old, next) {
		return
	}
	msg := event("panes", diff)
	if !reflect.DeepEqual(old.Workspaces, next.Workspaces) || !reflect.DeepEqual(old.Tabs, next.Tabs) || !reflect.DeepEqual(old.Profiles, next.Profiles) || !reflect.DeepEqual(old.Layouts, next.Layouts) || old.FocusedPane != next.FocusedPane {
		msg = event("state", next)
	}
	if !reflect.DeepEqual(old.Herdr, next.Herdr) {
		msg = append(msg, event("herdr", next.Herdr)...)
	}
	for ch := range a.clients {
		select {
		case ch <- msg:
		default:
			for len(ch) > 0 {
				select {
				case <-ch:
				default:
				}
			}
			select {
			case ch <- event("state", next):
			default:
			}
		}
	}
}

type Action struct {
	Action    string `json:"action"`
	Pane      string `json:"pane_id"`
	Text      string `json:"text"`
	Key       string `json:"key"`
	Profile   string `json:"profile"`
	Model     string `json:"model"`
	Effort    string `json:"effort"`
	Account   string `json:"account"`
	Cwd       string `json:"cwd"`
	Confirm   bool   `json:"confirm"`
	Direction string `json:"direction"`
	Worktree  string `json:"worktree"`
}

func (a *App) action(ctx context.Context, v Action) (string, error) {
	if strings.ContainsRune(v.Text, 0) || len(v.Text) > 64<<10 {
		return "", fmt.Errorf("text too long or contains NUL")
	}
	call := func(args ...string) (string, error) { out, err := a.herdr.Call(ctx, args...); return string(out), err }
	if v.Action == "reload" {
		return call("server", "reload-config")
	}
	if v.Action == "launch" {
		cmd := exec.Command(filepath.Join(a.root, "launcher", "HerdrLauncher.exe"))
		hideProcess(cmd)
		if err := cmd.Start(); err != nil {
			return "", err
		}
		go cmd.Wait()
		return "Herdr launcher started", nil
	}
	raw, err := a.herdr.Call(ctx, "api", "snapshot")
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	session := a.state.Session
	a.mu.Unlock()
	state, err := parseSnapshot(raw, session)
	if err != nil {
		return "", err
	}
	var pane *Pane
	for i := range state.Panes {
		if state.Panes[i].ID == v.Pane {
			pane = &state.Panes[i]
			break
		}
	}
	if v.Pane != "" && pane == nil {
		return "", fmt.Errorf("pane no longer exists in this session")
	}
	if v.Action == "shell" || v.Action == "start" {
		if v.Action == "start" {
			if err := a.launchGuard(paneRequest{Profile: v.Profile, Model: v.Model, Effort: v.Effort, Account: v.Account, Cwd: v.Cwd}); err != nil {
				return "", err
			}
		}
		if v.Action == "start" && (v.Profile == "next" || v.Worktree != "") {
			params := map[string]string{"Command": "start", "Target": v.Profile}
			if v.Worktree != "" {
				params["Worktree"] = v.Worktree
			}
			if v.Cwd != "" {
				params["Cwd"] = v.Cwd
			}
			if v.Model != "" {
				params["Model"] = v.Model
			}
			if v.Effort != "" {
				params["Effort"] = v.Effort
			}
			if v.Account != "" {
				params["Account"] = v.Account
			}
			return a.herdr.Helper(ctx, params)
		}
		if v.Action == "start" {
			if _, ok := a.profiles[v.Profile]; !ok {
				return "", fmt.Errorf("unknown agent profile")
			}
		}
		if v.Cwd == "" {
			if pane != nil {
				v.Cwd = pane.Cwd
			} else {
				v.Cwd = a.root
			}
		}
		if !filepath.IsAbs(v.Cwd) {
			return "", fmt.Errorf("folder must be absolute")
		}
		var created []byte
		if v.Direction == "" {
			v.Direction = "right"
		}
		if v.Direction != "right" && v.Direction != "down" {
			return "", fmt.Errorf("invalid split direction")
		}
		if pane != nil {
			created, err = a.herdr.Call(ctx, "pane", "split", pane.ID, "--direction", v.Direction, "--cwd", v.Cwd, "--no-focus")
		} else {
			created, err = a.herdr.Call(ctx, "workspace", "create", "--cwd", v.Cwd, "--label", "Deck", "--no-focus")
		}
		if err != nil {
			return "", err
		}
		if v.Action == "shell" {
			return string(created), nil
		}
		var response struct {
			Result struct {
				Pane Pane `json:"pane"`
				Root Pane `json:"root_pane"`
			} `json:"result"`
		}
		if err = json.Unmarshal(created, &response); err != nil {
			return "", err
		}
		id := response.Result.Pane.ID
		if id == "" {
			id = response.Result.Root.ID
		}
		if id == "" {
			return "", fmt.Errorf("creation returned no pane ID")
		}
		params := map[string]string{"Command": "start", "Target": v.Profile, "Pane": id, "Cwd": v.Cwd}
		if v.Model != "" {
			params["Model"] = v.Model
		}
		if v.Effort != "" {
			params["Effort"] = v.Effort
		}
		if v.Account != "" {
			params["Account"] = v.Account
		}
		output, startErr := a.herdr.Helper(ctx, params)
		if startErr != nil {
			if !strings.Contains(startErr.Error(), "agent_pane_busy") {
				return "", fmt.Errorf("start failed in %s: %w", id, startErr)
			}
			command, e := fallbackCommand(a.profiles[v.Profile], v.Model, v.Effort, v.Account)
			if e != nil {
				return "", e
			}
			out, e := call("pane", "run", id, command)
			if e != nil {
				return "", e
			}
			return "Busy-pane fallback submitted to " + id + ". Verify the detected agent before prompting.\n" + out, nil
		}
		return output, nil
	}
	if pane == nil {
		return "", fmt.Errorf("select a current pane")
	}
	switch v.Action {
	case "prompt":
		if strings.TrimSpace(v.Text) == "" {
			return "", fmt.Errorf("prompt is empty")
		}
		screen, err := a.herdr.Call(ctx, "pane", "read", pane.ID, "--source", "visible", "--lines", "200", "--format", "text")
		if err != nil {
			return "", err
		}
		if pane.Status == "blocked" || blockedScreen(string(screen)) {
			return "", fmt.Errorf("blocked: this pane needs your explicit key or paste action; no prompt sent")
		}
		if pane.Agent != "" || pane.Name != "" {
			target := pane.Name
			if target == "" {
				target = pane.ID
			}
			return a.deliver(ctx, pane.ID, "agent", "prompt", target, v.Text)
		}
		return a.deliver(ctx, pane.ID, "pane", "run", pane.ID, v.Text)
	case "paste":
		return call("pane", "send-text", pane.ID, v.Text)
	case "keys":
		valid := map[string]bool{"esc": true, "ctrl+c": true, "enter": true, "up": true, "down": true, "tab": true, "y": true, "n": true, "left": true, "right": true, "backspace": true}
		if !valid[v.Key] {
			return "", fmt.Errorf("unsupported key")
		}
		return call("pane", "send-keys", pane.ID, v.Key)
	case "close":
		if !v.Confirm {
			return "", fmt.Errorf("closing a pane requires confirmation")
		}
		return call("pane", "close", pane.ID)
	case "rename":
		if strings.TrimSpace(v.Text) == "" || len(v.Text) > 160 {
			return "", fmt.Errorf("name must be 1–160 bytes")
		}
		return call("pane", "rename", pane.ID, v.Text)
	case "focus":
		if pane.Name != "" {
			return call("agent", "focus", pane.Name)
		}
		return call("tab", "focus", pane.Tab)
	case "stop":
		if !v.Confirm {
			return "", fmt.Errorf("stopping an agent requires confirmation")
		}
		if pane.Name == "" {
			return "", fmt.Errorf("this pane has no managed agent name")
		}
		return a.herdr.Helper(ctx, map[string]string{"Command": "stop", "Target": pane.Name})
	default:
		return "", fmt.Errorf("unsupported action")
	}
}
