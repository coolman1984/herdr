package deck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Reservation struct {
	Shared  bool      `json:"shared"`
	ID      string    `json:"id"`
	Task    string    `json:"task"`
	Owner   string    `json:"owner"`
	Paths   []string  `json:"paths"`
	Created time.Time `json:"created"`
}
type Delivery struct {
	ID       string    `json:"id"`
	Dispatch string    `json:"dispatch_id"`
	Key      string    `json:"key"`
	Pane     string    `json:"pane"`
	TextSHA  string    `json:"text_sha"`
	State    string    `json:"state"`
	Error    string    `json:"error"`
	At       time.Time `json:"at"`
}
type ScopeBaseline struct {
	Repo    string
	Allowed []string
	Files   map[string]string
}
type LaunchPolicy struct {
	MaxActive int     `json:"max_active"`
	MinRAMMB  float64 `json:"min_ram_mb"`
	MinDiskGB float64 `json:"min_disk_gb"`
}
type LeadIdentity struct {
	Tool    string `json:"tool"`
	Account string `json:"account"`
	Known   bool   `json:"known"`
}

func leadIdentity(profiles map[string]Profile) LeadIdentity {
	result := LeadIdentity{Tool: "claude"}
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		if p, ok := profiles["claude"]; ok && p.DefaultAccount != "" {
			result.Account = p.DefaultAccount
			result.Known = true
		}
		return result
	}
	expected := filepath.Join(os.Getenv("USERPROFILE"), ".herdr-accounts", "claude")
	if strings.EqualFold(filepath.Clean(filepath.Dir(dir)), filepath.Clean(expected)) {
		result.Account = filepath.Base(dir)
		result.Known = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`).MatchString(result.Account)
	}
	return result
}

// normalizeAccount maps the main sign-in aliases onto the profile's default account folder.
func normalizeAccount(p Profile, account string) string {
	normalized := strings.ToLower(strings.TrimSpace(account))
	if normalized == "" || normalized == "main" || normalized == "default" {
		return p.DefaultAccount
	}
	return normalized
}

// mainAccountAlias drops the account the main sign-in already uses: it has no account folder.
func mainAccountAlias(p Profile, account string) string {
	if normalized := normalizeAccount(p, account); normalized == p.DefaultAccount {
		return ""
	} else {
		return normalized
	}
}

// pickNextAccount reads the usage table and returns the least-used account that is not the lead's.
func pickNextAccount(usage, lead string) (string, error) {
	if lead == "" {
		return "", &statusError{Status: 409, Msg: "account guard: the lead account is unknown, so no next account can be chosen"}
	}
	row := regexp.MustCompile(`^(\S+?)\*?\s+(\d+)%\s+.+?\s(\d+)%\s`)
	best := ""
	bestFive, bestWeek := 0, 0
	for _, line := range strings.Split(usage, "\n") {
		match := row.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if match == nil {
			continue
		}
		five, fiveErr := strconv.Atoi(match[2])
		week, weekErr := strconv.Atoi(match[3])
		if fiveErr != nil || weekErr != nil {
			continue
		}
		name := match[1]
		if name == lead || five >= 95 || week >= 95 {
			continue
		}
		if best == "" || five < bestFive || (five == bestFive && week < bestWeek) || (five == bestFive && week == bestWeek && name < best) {
			best, bestFive, bestWeek = name, five, week
		}
	}
	if best == "" {
		return "", &statusError{Status: 409, Msg: "no Claude account available besides the lead"}
	}
	return best, nil
}

type GuardState struct {
	Reservations []Reservation
	Deliveries   []Delivery
	Scopes       map[string]ScopeBaseline
	Policy       LaunchPolicy
	Queue        []*LaunchEntry
	LoadError    string
}
type LaunchEntry struct {
	ID      string      `json:"id"`
	Status  string      `json:"status"`
	Reason  string      `json:"reason"`
	Request paneRequest `json:"request"`
	At      time.Time   `json:"at"`
}

func (a *App) initGuardsLocked() {
	if a.guards != nil {
		return
	}
	guards := &GuardState{Reservations: []Reservation{}, Deliveries: []Delivery{}, Scopes: map[string]ScopeBaseline{}, Policy: LaunchPolicy{MaxActive: 4, MinRAMMB: 512, MinDiskGB: 0.5}, Queue: []*LaunchEntry{}}
	data, err := os.ReadFile(reservationPath())
	if err == nil {
		if e := json.Unmarshal(data, &guards.Reservations); e != nil {
			guards.Reservations = []Reservation{}
			guards.LoadError = "reservations file is invalid: " + e.Error()
		}
	} else if !os.IsNotExist(err) {
		guards.LoadError = "reservations file could not be read: " + err.Error()
	}
	if guards.Reservations == nil {
		guards.Reservations = []Reservation{}
	}
	a.guards = guards
}
func reservationPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-reservations.json")
}
func normalizeOwnership(repo, path string) (string, error) {
	if !filepath.IsAbs(repo) || strings.HasPrefix(repo, `\\`) {
		return "", fmt.Errorf("absolute local repository required")
	}
	path = strings.TrimSuffix(strings.ReplaceAll(path, "/", `\`), `\**`)
	path = strings.TrimSuffix(path, `\`)
	if path == "" || strings.ContainsAny(path, "*?\x00") {
		return "", fmt.Errorf("reserve concrete file paths or directory/** scopes")
	}
	full := path
	if !filepath.IsAbs(path) {
		full = filepath.Join(repo, path)
	}
	full = filepath.Clean(full)
	rel, err := filepath.Rel(repo, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, `..\`) {
		return "", fmt.Errorf("owned path leaves repository")
	}
	return strings.ToLower(full), nil
}
func overlaps(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+`\`) || strings.HasPrefix(b, a+`\`)
}
func normalizeReservationPath(root, path string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(path, `\`, "/"))
	if strings.ContainsRune(normalized, 0) {
		return "", fmt.Errorf("reservation path contains a NUL byte")
	}
	normalized = strings.TrimPrefix(normalized, "./")
	if strings.HasSuffix(normalized, "/**") {
		normalized = strings.TrimSuffix(normalized, "/**")
	} else if strings.ContainsAny(normalized, "*?") {
		return "", fmt.Errorf("reserve concrete file paths or directory/** scopes")
	}
	normalized = strings.TrimSuffix(normalized, "/")
	if normalized == "" {
		return "", fmt.Errorf("reservation path is empty")
	}
	if filepath.IsAbs(normalized) || strings.HasPrefix(normalized, "/") {
		relative, err := filepath.Rel(root, filepath.FromSlash(normalized))
		if err != nil {
			return "", fmt.Errorf("reservation path is not under the toolkit root")
		}
		normalized = filepath.ToSlash(relative)
	}
	normalized = filepath.ToSlash(filepath.Clean(filepath.FromSlash(normalized)))
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("reservation path leaves the toolkit root")
	}
	return strings.ToLower(normalized), nil
}
func reservationOverlaps(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
func readiness(text string) map[string]any {
	lower := strings.ToLower(text)
	warnings := []string{}
	checks := []struct {
		name  string
		words []string
	}{{"allowed files", []string{"allowed files", "files it may write", "exact scope", "stay in scope"}}, {"answer file or answer format", []string{"answer file", "answer format", "report.md", "section 7"}}, {"completion checks", []string{"definition of done", "checks:", "verification", "verify", "tests must"}}, {"stop conditions", []string{"blocked", "stop only", "stop conditions", "stop when"}}}
	for _, check := range checks {
		found := false
		for _, word := range check.words {
			found = found || strings.Contains(lower, word)
		}
		if !found {
			warnings = append(warnings, "Missing explicit "+check.name)
		}
	}
	return map[string]any{"ready": len(warnings) == 0, "warnings": warnings, "method": "heading/phrase check only; no instructions invented", "proof": "not verified"}
}
func recovery(message string) (string, string) {
	lower := strings.ToLower(message)
	if strings.Contains(lower, "blocked") || strings.Contains(lower, "sign in") || strings.Contains(lower, "trust this") {
		return "owner_required", "Open the pane, read the question, and decide yourself."
	}
	if strings.Contains(lower, "delivery unknown") {
		return "delivery_unknown", "Inspect the pane and delivery record before choosing an explicit resend."
	}
	if strings.Contains(lower, "server_not_running") || strings.Contains(lower, "socket") || strings.Contains(lower, "access is denied") {
		return "transport", "Check Herdr is running and that this Windows account can read its session."
	}
	if strings.Contains(lower, "start failed") || strings.Contains(lower, "not ready") || strings.Contains(lower, "missing") {
		return "startup", "Read the observed error and check the selected tool/profile before starting again."
	}
	return "agent_or_request_error", "Read the exact error and pane output; the cause is not verified."
}
func (a *App) deliver(ctx context.Context, key, pane, text string, args ...string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", notSent(err)
	}
	id := fmt.Sprintf("dispatch-%d", time.Now().UnixNano())
	sum := sha256.Sum256([]byte(text))
	record := Delivery{ID: id, Dispatch: id, Key: key, Pane: pane, TextSHA: hex.EncodeToString(sum[:]), State: "sent", At: time.Now()}
	out, err := a.herdr.Call(ctx, args...)
	if err != nil {
		var command *commandError
		if errors.As(err, &command) && !command.Started {
			return "", "", notSent(err)
		}
		record.State = "unknown"
		record.Error = err.Error()
		for _, code := range []string{"exit status 2", "agent_not_ready", "agent_blocked", "invalid_request"} {
			if strings.Contains(err.Error(), code) {
				record.State = "failed"
			}
		}
	}
	a.mu.Lock()
	a.initGuardsLocked()
	a.guards.Deliveries = append(a.guards.Deliveries, record)
	if len(a.guards.Deliveries) > 100 {
		a.guards.Deliveries = a.guards.Deliveries[len(a.guards.Deliveries)-100:]
	}
	a.mu.Unlock()
	if err != nil {
		if record.State == "failed" {
			return "", id, &statusError{Status: 502, Msg: fmt.Sprintf("dispatch %s rejected before submission: %v", id, err), Fields: map[string]any{"dispatch_id": id}}
		}
		return "", id, &statusError{Status: 504, Msg: "delivery unknown", Fields: map[string]any{"dispatch_id": id, "detail": err.Error()}}
	}
	return "Dispatch " + id + " accepted (submission acknowledged; completion not verified).\n" + string(out), id, nil
}
func (a *App) launchGuard(p paneRequest) error {
	if p.Purpose == "research" {
		return &statusError{Status: 409, Msg: "research route unavailable: no server-search-only adapter has been independently verified in this Deck; pane/browser routes are refused"}
	}
	profile := a.profiles[p.Profile]
	if profile.Kind == "claude" {
		account := normalizeAccount(profile, p.Account)
		if !a.lead.Known || account == "" || account == a.lead.Account {
			return &statusError{Status: 409, Msg: "account guard: pick another account (the lead uses " + leadName(a.lead) + ")", Fields: map[string]any{"lead_account": a.lead.Account}}
		}
	}
	if p.Helper && profile.Kind == "claude" {
		if p.Account == "" || p.LeadAccount == "" {
			return &statusError{Status: 409, Msg: "helper account identity unknown: choose an explicit helper account and lead account"}
		}
		if p.Account == p.LeadAccount {
			return &statusError{Status: 409, Msg: "Claude helpers must use an account different from the lead"}
		}
		if p.Account != "1984" && p.Account != "2040" {
			return &statusError{Status: 409, Msg: "Claude helper must use the owner's other account 1984 or 2040"}
		}
	}
	a.mu.Lock()
	a.initGuardsLocked()
	policy := a.guards.Policy
	a.mu.Unlock()
	launchFolder := a.launchFolder(p)
	folders := []string{launchFolder, a.root}
	if p.Floor != nil {
		for drive := range p.Floor.DiskFreeGB {
			folders = append(folders, drive+`\`)
		}
	}
	health, err := systemHealthVolumes(folders)
	if err != nil {
		return fmt.Errorf("launch resource check: %w", err)
	}
	available, _ := health["ram_available_mb"].(float64)
	if available < policy.MinRAMMB {
		return &statusError{Status: 409, Msg: fmt.Sprintf("held: free RAM %.0f MB is below %.0f MB floor", available, policy.MinRAMMB)}
	}
	drive := strings.ToUpper(filepath.VolumeName(launchFolder))
	if drive == "" {
		drive = strings.ToUpper(filepath.VolumeName(a.root))
	}
	knownDisks := map[string]float64{}
	if disks, ok := health["disks"].([]map[string]any); ok {
		for _, disk := range disks {
			name, _ := disk["drive"].(string)
			if free, ok := disk["free_gb"].(float64); ok {
				knownDisks[strings.ToUpper(name)] = free
			}
		}
	}
	if policy.MinDiskGB > 0 {
		free, known := knownDisks[drive]
		if !known {
			return &statusError{Status: 409, Msg: "disk_unknown", Fields: map[string]any{"drive": drive, "message": "launch folder volume capacity is unknown"}}
		}
		if free < policy.MinDiskGB {
			return &statusError{Status: 409, Msg: fmt.Sprintf("held: disk free %.2f GB is below %.2f GB floor", free, policy.MinDiskGB)}
		}
	}

	if p.Floor != nil {
		short := p.Floor.RAMFreeMB > 0 && available < p.Floor.RAMFreeMB
		free := map[string]float64{}
		known := map[string]float64{}
		if disks, ok := health["disks"].([]map[string]any); ok {
			for _, disk := range disks {
				name, _ := disk["drive"].(string)
				if value, ok := disk["free_gb"].(float64); ok {
					known[strings.ToUpper(name)] = value
				}
			}
		}
		for name, want := range p.Floor.DiskFreeGB {
			value, ok := known[strings.ToUpper(name)]
			if !ok {
				if want > 0 {
					return &statusError{Status: 409, Msg: "disk_unknown", Fields: map[string]any{"drive": strings.ToUpper(name), "message": "requested volume capacity is unknown"}}
				}
				continue
			}
			free[strings.ToUpper(name)] = value
			if value < want {
				short = true
			}
		}
		if short {
			return &statusError{Status: 409, Msg: "held: below the requested floor", Fields: map[string]any{"ram_free_mb": available, "disk_free_gb": free}}
		}
	}
	return nil
}
func leadName(lead LeadIdentity) string {
	if lead.Account == "" {
		return "unknown"
	}
	return lead.Account
}
func (a *App) guardEndpoint(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path == "/api/tasks/validate" {
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		file := r.URL.Query().Get("file")
		relative := strings.TrimPrefix(file, "tasks/")
		if !strings.HasPrefix(file, "tasks/") || strings.ContainsAny(relative, `/\`) || filepath.Ext(relative) != ".md" {
			failHTTP(w, 400, fmt.Errorf("tasks/*.md only"))
			return true
		}
		text, err := readAllowed(filepath.Join(a.root, "tasks"), relative)
		if err != nil {
			failHTTP(w, 400, err)
			return true
		}
		result := readiness(text)
		result["file"] = file
		respond(w, 200, result)
		return true
	}
	if path == "/api/deliveries" || path == "/api/dispatches" || path == "/api/launch/policy" || path == "/api/launch/queue" || path == "/api/reservations" {
		if r.Method == "GET" {
			key := r.URL.Query().Get("key")
			a.mu.Lock()
			a.initGuardsLocked()
			var result any
			switch path {
			case "/api/deliveries":
				result = map[string]any{"deliveries": append([]Delivery{}, a.guards.Deliveries...)}
			case "/api/dispatches":
				list := []Delivery{}
				for i := len(a.guards.Deliveries) - 1; i >= 0 && len(list) < 20; i-- {
					if key != "" && a.guards.Deliveries[i].Key != key {
						continue
					}
					list = append(list, a.guards.Deliveries[i])
				}
				result = map[string]any{"dispatches": list}
			case "/api/launch/policy":
				result = a.guards.Policy
			case "/api/launch/queue":
				result = map[string]any{"queue": append([]*LaunchEntry{}, a.guards.Queue...)}
			case "/api/reservations":
				result = map[string]any{"reservations": append([]Reservation{}, a.guards.Reservations...), "load_error": a.guards.LoadError}
			}
			a.mu.Unlock()
			respond(w, 200, result)
			return true
		}
	}
	switch path {
	case "/api/launch/policy":
		var p LaunchPolicy
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if p.MaxActive < 1 || p.MaxActive > 16 || p.MinRAMMB < 0 || p.MinRAMMB > 131072 || p.MinDiskGB < 0 || p.MinDiskGB > 10000 {
				return nil, fmt.Errorf("invalid launch policy")
			}
			a.mu.Lock()
			a.initGuardsLocked()
			a.guards.Policy = p
			a.mu.Unlock()
			return map[string]any{"ok": true, "policy": p}, nil
		})
		return true
	case "/api/launch/queue":
		var p paneRequest
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if _, ok := a.profiles[p.Profile]; !ok && p.Profile != "next" {
				return nil, fmt.Errorf("unknown launch profile")
			}
			a.mu.Lock()
			a.initGuardsLocked()
			if len(a.guards.Queue) >= 32 {
				a.mu.Unlock()
				return nil, fmt.Errorf("launch queue is full")
			}
			entry := &LaunchEntry{ID: fmt.Sprintf("launch-%d", time.Now().UnixNano()), Status: "held", Request: p, At: time.Now(), Reason: "Waiting for an explicit wave release"}
			a.guards.Queue = append(a.guards.Queue, entry)
			a.mu.Unlock()
			return map[string]any{"ok": true, "id": entry.ID, "status": "held"}, nil
		})
		return true
	case "/api/launch/release":
		var p struct {
			ID string `json:"id"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			a.mu.Lock()
			a.initGuardsLocked()
			var entry *LaunchEntry
			for _, e := range a.guards.Queue {
				if e.ID == p.ID {
					entry = e
					break
				}
			}
			if entry == nil || entry.Status != "held" {
				a.mu.Unlock()
				return nil, fmt.Errorf("held launch not found; failed starts are never replayed")
			}
			request := entry.Request
			a.mu.Unlock()

			id := ""
			if request.Key != "" {
				var err error
				id, err = a.paneID(request.Key)
				if err != nil {
					return nil, err
				}
			}
			out, err := a.action(r.Context(), Action{Action: "start", Pane: id, Profile: request.Profile, Model: request.Model, Effort: request.Effort, Account: request.Account, Cwd: request.Cwd, StartRequest: &request})
			a.mu.Lock()
			entry.Status = "started"
			entry.Reason = out
			if err != nil {
				entry.Status = "failed"
				entry.Reason = err.Error()
			}
			a.mu.Unlock()
			return map[string]any{"ok": err == nil, "message": out, "id": p.ID}, err
		})
		return true
	case "/api/reservations":
		var p struct {
			Task    string   `json:"task"`
			Owner   string   `json:"owner"`
			Paths   []string `json:"paths"`
			Overlap bool     `json:"overlap"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if p.Task == "" || len(p.Task) > 200 {
				return nil, &statusError{Status: 400, Msg: "reservation task must be 1-200 bytes"}
			}
			if p.Owner == "" || len(p.Owner) > 120 {
				return nil, &statusError{Status: 400, Msg: "reservation owner must be 1-120 bytes"}
			}
			if len(p.Paths) == 0 || len(p.Paths) > 128 {
				return nil, &statusError{Status: 400, Msg: "reserve 1-128 paths"}
			}
			normalized := []string{}
			for _, path := range p.Paths {
				n, e := normalizeReservationPath(a.root, path)
				if e != nil {
					return nil, &statusError{Status: 400, Msg: e.Error()}
				}
				normalized = append(normalized, n)
			}
			a.mu.Lock()
			defer a.mu.Unlock()
			a.initGuardsLocked()
			if a.guards.LoadError != "" {
				return nil, &statusError{Status: 503, Msg: "ledger_unavailable", Fields: map[string]any{"message": a.guards.LoadError}}
			}
			conflicts := []map[string]string{}
			kept := []Reservation{}
			for _, old := range a.guards.Reservations {
				if old.Task == p.Task && old.Owner == p.Owner {
					return map[string]any{"ok": true, "id": old.ID, "reservations": a.guards.Reservations}, nil
				}
				for _, x := range normalized {
					for _, y := range old.Paths {
						if reservationOverlaps(x, y) {
							conflicts = append(conflicts, map[string]string{"path": x, "task": old.Task, "owner": old.Owner})
						}
					}
				}
				kept = append(kept, old)
			}
			if len(conflicts) > 0 && !p.Overlap {
				return nil, &statusError{Status: 409, Msg: "reservation conflict", Fields: map[string]any{"conflicts": conflicts}}
			}
			if len(kept) >= 200 {
				return nil, &statusError{Status: 409, Msg: "reservation limit reached"}
			}
			entry := Reservation{Shared: p.Overlap, ID: fmt.Sprintf("res-%d", time.Now().UnixNano()), Task: p.Task, Owner: p.Owner, Paths: normalized, Created: time.Now()}
			kept = append(kept, entry)
			data, _ := json.Marshal(kept)
			if err := atomicWrite(reservationPath(), data); err != nil {
				return nil, err
			}
			a.guards.Reservations = kept
			return map[string]any{"ok": true, "id": entry.ID, "reservations": kept}, nil
		})
		return true
	case "/api/reservations/release":
		var p struct {
			ID string `json:"id"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.initGuardsLocked()
			if a.guards.LoadError != "" {
				return nil, &statusError{Status: 503, Msg: "ledger_unavailable", Fields: map[string]any{"message": a.guards.LoadError}}
			}
			kept := []Reservation{}
			found := false
			for _, old := range a.guards.Reservations {
				if old.ID == p.ID {
					found = true
					continue
				}
				kept = append(kept, old)
			}
			if !found {
				return nil, &statusError{Status: 404, Msg: "reservation not found"}
			}
			data, _ := json.Marshal(kept)
			if err := atomicWrite(reservationPath(), data); err != nil {
				return nil, err
			}
			a.guards.Reservations = kept
			return map[string]any{"ok": true}, nil
		})
		return true
	case "/api/proof/audit":
		var p struct {
			Text     string   `json:"text"`
			Official []string `json:"official_domains"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		if len(p.Text) > 256<<10 {
			failHTTP(w, 400, fmt.Errorf("report exceeds 256 KiB"))
			return true
		}
		respond(w, 200, auditProof(p.Text, p.Official))
		return true
	case "/api/scope/baseline":
		var p struct {
			ID      string   `json:"id"`
			Repo    string   `json:"repo"`
			Allowed []string `json:"allowed"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if p.ID == "" || len(p.ID) > 120 || len(p.Allowed) > 128 {
				return nil, fmt.Errorf("scope id and at most 128 allowed paths required")
			}
			allowed := []string{}
			for _, path := range p.Allowed {
				n, e := normalizeOwnership(p.Repo, path)
				if e != nil {
					return nil, e
				}
				allowed = append(allowed, n)
			}
			files, err := scopeFiles(r.Context(), p.Repo)
			if err != nil {
				return nil, err
			}
			a.mu.Lock()
			a.initGuardsLocked()
			if len(a.guards.Scopes) >= 32 {
				a.mu.Unlock()
				return nil, fmt.Errorf("scope baseline limit reached")
			}
			a.guards.Scopes[p.ID] = ScopeBaseline{Repo: p.Repo, Allowed: allowed, Files: files}
			a.mu.Unlock()
			return map[string]any{"ok": true, "id": p.ID, "baseline_changed_paths": len(files)}, nil
		})
		return true
	case "/api/scope/check":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		a.mu.Lock()
		a.initGuardsLocked()
		baseline, ok := a.guards.Scopes[r.URL.Query().Get("id")]
		a.mu.Unlock()
		if !ok {
			failHTTP(w, 404, fmt.Errorf("scope baseline missing"))
			return true
		}
		files, err := scopeFiles(r.Context(), baseline.Repo)
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		unexpected := []string{}
		changed := []string{}
		for path, hash := range files {
			if baseline.Files[path] == hash {
				continue
			}
			changed = append(changed, path)
			allowed := false
			for _, scope := range baseline.Allowed {
				allowed = allowed || overlaps(path, scope)
			}
			if !allowed {
				unexpected = append(unexpected, path)
			}
		}
		sort.Strings(unexpected)
		sort.Strings(changed)
		respond(w, 200, map[string]any{"changed": changed, "unexpected": unexpected, "warning": "Changes can belong to concurrent agents; no blame or revert is inferred."})
		return true
	}
	return false
}
func scopeFiles(ctx context.Context, repo string) (map[string]string, error) {
	if !filepath.IsAbs(repo) || strings.HasPrefix(repo, `\\`) {
		return nil, fmt.Errorf("absolute local repo required")
	}
	raw, err := runCommand(ctx, "git.exe", "", "--no-optional-locks", "-C", repo, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	parts := strings.Split(string(raw), "\x00")
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if len(part) < 4 {
			continue
		}
		path := part[3:]
		full, e := normalizeOwnership(repo, path)
		if e != nil {
			return nil, e
		}
		fingerprint := part[:2]
		name := strings.ToLower(filepath.Base(full))
		if name == "auth.json" || name == ".credentials.json" || strings.Contains(name, "credential") || strings.HasSuffix(name, ".key") || strings.HasSuffix(name, ".env") {
			fingerprint += "protected:not read"
		} else {
			info, e := os.Lstat(full)
			if e == nil {
				if !info.Mode().IsRegular() || info.Size() > 1<<20 {
					fingerprint += "large/linked:not verified"
				} else {
					relative, e := filepath.Rel(repo, full)
					if e != nil {
						return nil, e
					}
					text, e := readAllowed(repo, relative)
					if e != nil {
						return nil, e
					}
					hash := sha256.Sum256([]byte(text))
					fingerprint += hex.EncodeToString(hash[:])
				}
			} else if !os.IsNotExist(e) {
				return nil, e
			}
		}
		result[full] = fingerprint
		if len(result) > 512 {
			return nil, fmt.Errorf("scope comparison exceeds 512 changed paths")
		}
		if part[0] == 'R' || part[0] == 'C' {
			i++
		}
	}
	return result, nil
}
func auditProof(text string, official []string) map[string]any {
	slots := map[string]string{}
	lower := strings.ToLower(text)
	for _, name := range []string{"checks run", "old-code failure", "mutation", "independent review", "isolated live test"} {
		slots[name] = "not verified"
		if strings.Contains(lower, name) {
			slots[name] = "mentioned; not verified"
		}
	}
	citations := []map[string]any{}
	seen := map[string]bool{}
	for _, raw := range regexp.MustCompile(`https?://[^\s<>\)\]\"`+"`"+`]+`).FindAllString(text, 100) {
		if seen[raw] {
			continue
		}
		seen[raw] = true
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		isOfficial := false
		for _, domain := range official {
			host := strings.ToLower(parsed.Hostname())
			domain = strings.ToLower(strings.TrimSpace(domain))
			if domain != "" && (host == domain || strings.HasPrefix(host, "www.") && strings.TrimPrefix(host, "www.") == domain) {
				isOfficial = true
			}
		}
		citations = append(citations, map[string]any{"url": raw, "official_domain": isOfficial, "claim_status": "not verified"})
	}
	return map[string]any{"proof": slots, "citations": citations, "missing_citations": len(citations) == 0, "note": "URL presence or a claimed test is not proof. No URLs were opened."}
}
