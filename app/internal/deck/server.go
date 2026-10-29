package deck

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

type App struct {
	herdr                           Herdr
	sizer                           *Sizer // nil: Deck does not size Herdr's tabs (tests, HERDR_BIN)
	token, origin, root             string
	sessionName, herdrBin           string
	profiles                        map[string]Profile
	files                           http.Handler
	reopen                          func() error
	startDetached                   func(name string, args ...string) error
	admissionMu                     sync.Mutex
	pending                         []*pendingLaunch
	launchSequence                  uint64
	rawTails                        map[string]chan struct{}
	launchFolders                   map[string]bool
	prefsMu                         sync.Mutex
	healthMu                        sync.Mutex
	healthAt                        time.Time
	healthCache                     map[string]any
	interests                       map[string]map[string]bool
	clientIDs                       map[chan []byte]string
	clientSequence                  uint64
	counters                        debugCounters
	readCursor                      int
	failedReads                     map[string]int
	mu                              sync.Mutex
	refreshMu                       sync.Mutex
	actionMu                        sync.Mutex
	state                           State
	clients                         map[chan []byte]bool
	visible                         map[string]bool
	readOnly, seen                  bool
	started, lastClient, metadataAt time.Time
	jobs                            map[string]*Job
	lastRead, lastChange            map[string]time.Time
	debug                           bool
	lead                            LeadIdentity
	guards                          *GuardState
	lastDispatch                    string
}

func newApp(h Herdr, session, token, origin, root string, profiles map[string]Profile) *App {
	lead := leadIdentity(profiles)
	return &App{launchFolders: map[string]bool{}, herdr: h, token: token, origin: origin, root: root, profiles: profiles, lead: lead, startDetached: startHiddenProcess, state: State{Session: session, Panes: []Pane{}, Workspaces: []Nav{}, Tabs: []Nav{}, Lead: lead, Root: root}, clients: map[chan []byte]bool{}, visible: map[string]bool{}, started: time.Now(), lastClient: time.Now(), jobs: map[string]*Job{}}
}
func startHiddenProcess(name string, args ...string) error {
	cmd := hiddenCommand(context.Background(), name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
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

// statusError carries an HTTP status and extra JSON fields to the client.
type statusError struct {
	Status int
	Fields map[string]any // merged into the error body; "error" is set from Msg
	Msg    string
}

func (e *statusError) Error() string { return e.Msg }
func failHTTP(w http.ResponseWriter, status int, err error) {
	var detail *statusError
	if !errors.As(err, &detail) {
		err = readFailure(err)
	}
	if errors.As(err, &detail) {
		status = detail.Status
	}
	kind, next := recovery(err.Error())
	body := map[string]any{"error": err.Error(), "kind": kind, "next_step": next, "cause": "unknown unless stated in the observed error"}
	if detail != nil {
		for key, value := range detail.Fields {
			body[key] = value
		}
	}
	respond(w, status, body)
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
		if !v.Active {
			v.Visible = nil
		}
		a.setVisibleLocked("legacy", v.Visible)
		a.mu.Unlock()
		respond(w, 200, map[string]bool{"ok": true})
	case "/api/size":
		// The page's card area in terminal cells; Herdr sizes the panes from it (sizer.go).
		var v struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		}
		if !readBody(w, r, &v) {
			return
		}
		if v.Cols < 20 || v.Cols > 1000 || v.Rows < 5 || v.Rows > 500 {
			failHTTP(w, 400, fmt.Errorf("size %dx%d is outside 20x5..1000x500", v.Cols, v.Rows))
			return
		}
		a.mu.Lock()
		readOnly := a.readOnly
		a.mu.Unlock()
		if a.sizer == nil || readOnly {
			respond(w, 200, SizerStatus{Error: map[bool]string{true: "read-only mode: Deck leaves Herdr's size alone", false: "sizing is off in this Deck"}[readOnly]})
			return
		}
		respond(w, 200, a.sizer.Set(uint16(v.Cols), uint16(v.Rows)))
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
		a.state.ReadOnly = v.Enabled
		for ch := range a.clients {
			a.queueLocked(ch, event("settings", map[string]bool{"read_only": v.Enabled}))
		}
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
			var detail *statusError
			if errors.As(err, &detail) {
				status = detail.Status
			} else if strings.HasPrefix(err.Error(), "blocked:") {
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
	a.clientSequence++
	id := fmt.Sprintf("c-%d", a.clientSequence)
	if a.clientIDs == nil {
		a.clientIDs = map[chan []byte]string{}
	}
	a.clientIDs[ch] = id
	a.seen = true
	a.lastClient = time.Now()
	initial := append(event("hello", map[string]string{"client_id": id}), event("settings", map[string]bool{"read_only": a.readOnly})...)
	initial = append(initial, event("state", a.clientStateLocked(ch, a.state))...)
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.clients, ch)
		delete(a.clientIDs, ch)
		delete(a.interests, id)
		if len(a.clients) == 0 {
			delete(a.interests, "legacy")
		}
		a.unionVisibleLocked()
		a.lastClient = time.Now()
		a.mu.Unlock()
	}()
	if _, err := w.Write(initial); err != nil {
		return
	}
	a.countSSE(initial)
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
			a.countSSE(msg)
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
			active := len(a.clients) > 0
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
	defer a.finishPoll(time.Now())
	ctx, cancel := context.WithTimeout(context.WithValue(parent, backgroundRPC{}, true), 750*time.Millisecond)
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
		if errors.Is(err, errRPCBusy) {
			return
		}
		if err == nil {
			next, err = parseSnapshot(raw, old.Session)
		}
		if err != nil {
			next = old
			next.Panes = append([]Pane{}, old.Panes...)
			for i := range next.Panes {
				next.Panes[i].Stale = true
			}
			next.Error = err.Error()
			next.Herdr.Running = false
			a.mu.Lock()
			a.counters.Errors++
			a.metadataAt = time.Now()
			a.mu.Unlock()
			next.Herdr.Error = err.Error()
			a.publish(old, next)
			return
		}

		a.mu.Lock()
		a.metadataAt = time.Now()
		a.reconcilePendingLocked(next)
		a.mu.Unlock()
	} else {
		next.Panes = append([]Pane{}, old.Panes...)
	}
	cache := map[string]Pane{}
	if a.lastRead == nil {
		a.lastRead = map[string]time.Time{}
		a.lastChange = map[string]time.Time{}
		a.failedReads = map[string]int{}
	}
	for _, p := range old.Panes {
		cache[p.Key] = p
	}
	// Copy cached content before the bounded round-robin read pass.
	for i := range next.Panes {
		p := &next.Panes[i]
		previous := cache[p.Key]
		p.Screen = previous.Screen
		p.ScreenVersion = previous.ScreenVersion
		p.Error = previous.Error
		p.Blocked = p.SourceStatus == "blocked" || blockedScreen(p.Screen)
		if p.Blocked {
			p.Status = "blocked"
		} else {
			p.Status = p.SourceStatus
		}
	}
	// At most four reads, each with its own deadline; a slow pane cannot consume the metadata budget.
	reads := 0
	n := len(next.Panes)
	start := 0
	if n > 0 {
		start = a.readCursor % n
	}
	for offset := 0; offset < n && reads < 4; offset++ {
		i := (start + offset) % n
		p := &next.Panes[i]
		previous := cache[p.Key]
		if !visible[p.ID] && !visible[p.Key] {
			continue
		}
		interval := 1400 * time.Millisecond
		if p.Focused && time.Since(a.lastChange[p.Key]) < 2*time.Second {
			interval = 280 * time.Millisecond
		}
		if failures := a.failedReads[p.Key]; failures > 0 {
			interval = time.Duration(1<<min(failures, 5)) * time.Second
		}
		if (!force || a.failedReads[p.Key] > 0) && time.Since(a.lastRead[p.Key]) < interval {
			continue
		}
		readCtx, readCancel := context.WithTimeout(context.WithValue(parent, backgroundRPC{}, true), 350*time.Millisecond)
		raw, err := a.herdr.Call(readCtx, "pane", "read", p.ID, "--source", "recent", "--lines", "2000", "--format", "ansi")
		readCancel()
		if errors.Is(err, errRPCBusy) {
			continue
		}
		reads++
		a.readCursor = (i + 1) % n
		a.lastRead[p.Key] = time.Now()
		if err == nil {
			var screen string
			screen, err = parseScreen(raw, p.Scroll.Rows)
			if err == nil {
				p.Screen = screen
			}
		}
		if err != nil {
			p.Error = err.Error()
			a.failedReads[p.Key]++
			a.mu.Lock()
			a.counters.Errors++
			a.mu.Unlock()
			continue
		}
		a.failedReads[p.Key] = 0
		p.Error = ""
		if p.Screen != previous.Screen {
			p.ScreenVersion++
			a.lastChange[p.Key] = time.Now()
		}
		p.Blocked = p.SourceStatus == "blocked" || blockedScreen(p.Screen)
		p.Status = p.SourceStatus
		if p.Blocked {
			p.Status = "blocked"
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
			delete(a.failedReads, key)
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
				before, after := strings.Split(previous.Screen, "\n"), strings.Split(p.Screen, "\n")
				patches := rowChangesRows(before, after, previous.ScreenVersion)
				if !validPatches(patches, len(before), len(after), previous.ScreenVersion) {
					screen := p.Screen
					update.Screen = &screen
					update.ScreenVersion = previous.ScreenVersion + 1
					next.Panes[i].ScreenVersion = update.ScreenVersion
					changed = append(changed, update)
					delete(prior, p.Key)
					continue
				}
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
	next.ReadOnly = a.readOnly
	next.Lead = a.lead
	next.Root = a.root
	a.state = next
	if reflect.DeepEqual(old, next) {
		return
	}
	full := !reflect.DeepEqual(old.Workspaces, next.Workspaces) || !reflect.DeepEqual(old.Tabs, next.Tabs) || !reflect.DeepEqual(old.Profiles, next.Profiles) || !reflect.DeepEqual(old.Layouts, next.Layouts) || old.FocusedPane != next.FocusedPane
	for ch := range a.clients {
		filtered := make([]PaneUpdate, 0, len(changed))
		for _, update := range changed {
			if !a.interestedLocked(ch, update.Pane) {
				update.Screen = nil
				update.Patch = nil
				update.Pane.Screen = ""
			}
			filtered = append(filtered, update)
		}
		diff["panes"] = filtered
		msg := event("panes", diff)
		if full {
			msg = event("state", a.clientStateLocked(ch, next))
		}
		if !reflect.DeepEqual(old.Herdr, next.Herdr) {
			msg = append(msg, event("herdr", next.Herdr)...)
		}
		a.queueLocked(ch, msg)
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
	// Expect is what the sender saw: "shell" for a shell card; anything else means an agent.
	Expect       string       `json:"expect"`
	StartRequest *paneRequest `json:"-"`
}

func (a *App) action(ctx context.Context, v Action) (result string, resultErr error) {
	a.mu.Lock()
	a.lastDispatch = ""
	a.mu.Unlock()
	if strings.ContainsRune(v.Text, 0) || len(v.Text) > 64<<10 {
		return "", fmt.Errorf("text too long or contains NUL")
	}
	call := func(args ...string) (string, error) { out, err := a.herdr.Call(ctx, args...); return string(out), err }
	if v.Action == "reload" {
		return call("server", "reload-config")
	}
	if v.Action == "launch" {
		if a.sessionName != "" {
			if err := a.startDetached(a.herdrBin, "--session", a.sessionName, "server"); err != nil {
				return "", err
			}
			return "Herdr server for session " + a.sessionName + " started", nil
		}
		if err := a.startDetached(filepath.Join(a.root, "launcher", "HerdrLauncher.exe")); err != nil {
			return "", err
		}
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
	nextClaude := false
	if v.Action == "start" && v.Profile == "next" {
		usage, err := a.herdr.Helper(ctx, map[string]string{"Command": "usage"})
		if err != nil {
			return "", fmt.Errorf("account usage lookup: %w", err)
		}
		account, err := pickNextAccount(usage, a.lead.Account)
		if err != nil {
			return "", err
		}
		v.Profile, v.Account, nextClaude = "claude", account, true
	}
	if v.Action == "shell" || v.Action == "start" {
		if v.Action == "start" {
			req := paneRequest{Profile: v.Profile, Model: v.Model, Effort: v.Effort, Account: v.Account, Cwd: v.Cwd, Worktree: v.Worktree}
			if v.StartRequest != nil {
				req = *v.StartRequest
				req.Profile = v.Profile
				req.Account = v.Account
			}
			if req.Cwd == "" && pane != nil {
				req.Cwd = pane.Cwd
			}
			if req.Cwd == "" {
				req.Cwd = a.root
			}
			v.Cwd = req.Cwd
			ticket, e := a.admit(ctx, []paneRequest{req})
			if e != nil {
				return "", e
			}
			defer func() { ticket.finish(result, resultErr) }()
		}

		if v.Action == "start" && (nextClaude || v.Worktree != "") {
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
			if account := mainAccountAlias(a.profiles[v.Profile], v.Account); account != "" {
				params["Account"] = account
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
		if account := mainAccountAlias(a.profiles[v.Profile], v.Account); account != "" {
			params["Account"] = account
		}
		output, startErr := a.herdr.Helper(ctx, params)
		if startErr != nil {
			if !strings.Contains(startErr.Error(), "agent_pane_busy") {
				return "", fmt.Errorf("start failed in %s: %w", id, startErr)
			}
			command, e := fallbackCommand(a.profiles[v.Profile], v.Model, v.Effort, mainAccountAlias(a.profiles[v.Profile], v.Account))
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
		// The agent has exited and only its shell is left: agent text must never run as a command.
		// Checked before the screen: an old dialog still on screen is not the reason.
		noAgent := pane.Agent == "" && pane.Name == ""
		if noAgent && v.Expect != "shell" {
			return "", &statusError{Status: 409, Msg: "agent_exited", Fields: map[string]any{"error": "agent_exited", "pane_id": pane.ID, "message": "The agent in " + pane.ID + " has exited; only a shell is left - not sent"}}
		}
		screen, err := a.herdr.Call(ctx, "pane", "read", pane.ID, "--source", "visible", "--lines", "60", "--format", "ansi")
		if err != nil {
			return "", err
		}
		if pane.Status == "blocked" || blockedScreen(string(screen)) {
			return "", fmt.Errorf("blocked: this pane needs your explicit key or paste action; no prompt sent")
		}
		if err := inputNotEmpty(string(screen), *pane); err != nil {
			return "", err
		}
		args := []string{"agent", "prompt", pane.Name, v.Text}
		if noAgent {
			args = []string{"pane", "run", pane.ID, v.Text}
		} else if pane.Name == "" {
			args[2] = pane.ID
		}
		out, dispatch, err := a.deliver(ctx, session+"|"+pane.ID, pane.ID, v.Text, args...)
		a.mu.Lock()
		a.lastDispatch = dispatch
		a.mu.Unlock()
		return out, err
	case "paste":
		screen, err := a.herdr.Call(ctx, "pane", "read", pane.ID, "--source", "visible", "--lines", "60", "--format", "ansi")
		if err != nil {
			return "", err
		}
		if err := inputNotEmpty(string(screen), *pane); err != nil {
			return "", err
		}
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
