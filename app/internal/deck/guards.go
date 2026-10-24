package deck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Reservation struct {
	Owner   string    `json:"owner"`
	Repo    string    `json:"repo"`
	Paths   []string  `json:"paths"`
	Created time.Time `json:"created"`
}
type Delivery struct {
	ID    string    `json:"id"`
	Pane  string    `json:"pane"`
	State string    `json:"state"`
	Error string    `json:"error"`
	At    time.Time `json:"at"`
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

type GuardState struct {
	Reservations []Reservation
	Deliveries   []Delivery
	Scopes       map[string]ScopeBaseline
	Policy       LaunchPolicy
	Queue        []*LaunchEntry
}
type LaunchEntry struct {
	ID      string      `json:"id"`
	Status  string      `json:"status"`
	Reason  string      `json:"reason"`
	Request paneRequest `json:"request"`
	At      time.Time   `json:"at"`
}

func (a *App) initGuardsLocked() {
	if a.guards == nil {
		a.guards = &GuardState{Reservations: []Reservation{}, Deliveries: []Delivery{}, Scopes: map[string]ScopeBaseline{}, Policy: LaunchPolicy{MaxActive: 4, MinRAMMB: 512, MinDiskGB: 0.5}, Queue: []*LaunchEntry{}}
	}
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
	if strings.Contains(lower, "delivery unknown") || strings.Contains(lower, "timed out") || strings.Contains(lower, "stalled") {
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
func (a *App) deliver(ctx context.Context, pane string, args ...string) (string, error) {
	id := fmt.Sprintf("dispatch-%d", time.Now().UnixNano())
	record := Delivery{ID: id, Pane: pane, State: "sending", At: time.Now()}
	out, err := a.herdr.Call(ctx, args...)
	record.State = "accepted"
	if err != nil {
		record.State = "delivery unknown"
		record.Error = err.Error()
		for _, code := range []string{"exit status 2", "agent_not_ready", "agent_blocked", "invalid_request"} {
			if strings.Contains(err.Error(), code) {
				record.State = "rejected"
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
		if record.State == "rejected" {
			return "", fmt.Errorf("dispatch %s rejected before submission: %w", id, err)
		}
		return "", fmt.Errorf("delivery unknown (%s): %w; inspect state/output before any explicit resend", id, err)
	}
	return "Dispatch " + id + " accepted (submission acknowledged; completion not verified).\n" + string(out), nil
}
func (a *App) launchGuard(p paneRequest) error {
	if p.Purpose == "research" {
		return fmt.Errorf("research route unavailable: no server-search-only adapter has been independently verified in this Deck; pane/browser routes are refused")
	}
	profile := a.profiles[p.Profile]
	if profile.Kind == "claude" {
		if !a.lead.Known || p.Account == "" || p.Account == a.lead.Account {
			return fmt.Errorf("account guard: pick another account; lead identity is unknown, empty, or matches the selected account")
		}
	}
	if p.Helper && profile.Kind == "claude" {
		if p.Account == "" || p.LeadAccount == "" {
			return fmt.Errorf("helper account identity unknown: choose an explicit helper account and lead account")
		}
		if p.Account == p.LeadAccount {
			return fmt.Errorf("Claude helpers must use an account different from the lead")
		}
		if p.Account != "1984" && p.Account != "2040" {
			return fmt.Errorf("Claude helper must use the owner's other account 1984 or 2040")
		}
	}
	a.mu.Lock()
	a.initGuardsLocked()
	policy := a.guards.Policy
	panes := append([]Pane{}, a.state.Panes...)
	a.mu.Unlock()
	active := 0
	for _, pane := range panes {
		if pane.Agent != "" {
			active++
		}
	}
	if active >= policy.MaxActive {
		return fmt.Errorf("held: %d active agents reach the chosen limit %d", active, policy.MaxActive)
	}
	health, err := systemHealth()
	if err != nil {
		return fmt.Errorf("launch resource check: %w", err)
	}
	available, _ := health["ram_available_mb"].(float64)
	if available < policy.MinRAMMB {
		return fmt.Errorf("held: free RAM %.0f MB is below %.0f MB floor", available, policy.MinRAMMB)
	}
	drive := strings.ToUpper(filepath.VolumeName(p.Cwd))
	if drive == "" {
		drive = strings.ToUpper(filepath.VolumeName(a.root))
	}
	if disks, ok := health["disks"].([]map[string]any); ok {
		for _, disk := range disks {
			if disk["drive"] == drive {
				free, known := disk["free_gb"].(float64)
				if !known {
					return fmt.Errorf("held: free disk space unavailable")
				}
				if free < policy.MinDiskGB {
					return fmt.Errorf("held: disk free %.2f GB is below %.2f GB floor", free, policy.MinDiskGB)
				}
			}
		}
	}
	return nil
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
	if path == "/api/deliveries" || path == "/api/launch/policy" || path == "/api/launch/queue" || path == "/api/reservations" {
		if r.Method == "GET" {
			a.mu.Lock()
			a.initGuardsLocked()
			var result any
			switch path {
			case "/api/deliveries":
				result = map[string]any{"deliveries": append([]Delivery{}, a.guards.Deliveries...)}
			case "/api/launch/policy":
				result = a.guards.Policy
			case "/api/launch/queue":
				result = map[string]any{"queue": append([]*LaunchEntry{}, a.guards.Queue...)}
			case "/api/reservations":
				result = map[string]any{"reservations": append([]Reservation{}, a.guards.Reservations...)}
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
			if err := a.launchGuard(request); err != nil {
				return nil, err
			}
			id := ""
			if request.Key != "" {
				var err error
				id, err = a.paneID(request.Key)
				if err != nil {
					return nil, err
				}
			}
			out, err := a.action(r.Context(), Action{Action: "start", Pane: id, Profile: request.Profile, Model: request.Model, Effort: request.Effort, Account: request.Account, Cwd: request.Cwd})
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
			Owner   string   `json:"owner"`
			Repo    string   `json:"repo"`
			Paths   []string `json:"paths"`
			Release bool     `json:"release"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if p.Owner == "" || len(p.Owner) > 120 {
				return nil, fmt.Errorf("owner required")
			}
			normalized := []string{}
			if !p.Release {
				if len(p.Paths) == 0 || len(p.Paths) > 128 {
					return nil, fmt.Errorf("reserve 1–128 paths")
				}
				for _, path := range p.Paths {
					n, e := normalizeOwnership(p.Repo, path)
					if e != nil {
						return nil, e
					}
					normalized = append(normalized, n)
				}
			}
			a.mu.Lock()
			defer a.mu.Unlock()
			a.initGuardsLocked()
			next := []Reservation{}
			for _, old := range a.guards.Reservations {
				if old.Owner == p.Owner {
					continue
				}
				if !p.Release {
					for _, x := range normalized {
						for _, y := range old.Paths {
							if overlaps(x, y) {
								return nil, fmt.Errorf("path overlaps reservation by %s: %s", old.Owner, x)
							}
						}
					}
				}
				next = append(next, old)
			}
			if !p.Release {
				next = append(next, Reservation{Owner: p.Owner, Repo: p.Repo, Paths: normalized, Created: time.Now()})
			}
			if len(next) > 64 {
				return nil, fmt.Errorf("reservation limit reached")
			}
			data, _ := json.Marshal(next)
			if err := atomicWrite(filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-state", "reservations.json"), data); err != nil {
				return nil, err
			}
			a.guards.Reservations = next
			return map[string]any{"ok": true, "reservations": next}, nil
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
