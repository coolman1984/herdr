package deck

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var keyPattern = regexp.MustCompile(`^(ctrl\+[a-z]|f([1-9]|1[0-2]))$`)

func validKey(key string) bool {
	return keyPattern.MatchString(key) || map[string]bool{"esc": true, "ctrl+c": true, "enter": true, "up": true, "down": true, "tab": true, "y": true, "n": true, "left": true, "right": true, "backspace": true, "space": true, "shift+tab": true}[key]
}
func (a *App) script(ctx context.Context, name string, params map[string]any, timeout time.Duration) (string, error) {
	runner, ok := a.herdr.(ScriptRunner)
	if !ok {
		return "", fmt.Errorf("toolkit script runner unavailable")
	}
	return runner.RunScript(ctx, name, params, timeout)
}
func (a *App) parity(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/api/jobs") {
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		a.mu.Lock()
		list := []Job{}
		for _, j := range a.jobs {
			list = append(list, *j)
		}
		a.mu.Unlock()
		sort.Slice(list, func(i, j int) bool { return list[i].Started.After(list[j].Started) })
		respond(w, 200, map[string]any{"jobs": list})
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/pane/") || strings.HasPrefix(r.URL.Path, "/api/tab/") || strings.HasPrefix(r.URL.Path, "/api/workspace/") {
		commands := map[string]bool{"/api/pane/resize": true, "/api/pane/swap": true, "/api/pane/move": true, "/api/pane/zoom": true, "/api/pane/focus": true, "/api/tab/create": true, "/api/tab/rename": true, "/api/tab/close": true, "/api/workspace/rename": true, "/api/workspace/close": true, "/api/workspace/focus": true}
		if commands[r.URL.Path] {
			var p struct {
				Key       string  `json:"key"`
				With      string  `json:"with"`
				Direction string  `json:"direction"`
				Amount    float64 `json:"amount"`
				Tab       string  `json:"tab_id"`
				Workspace string  `json:"workspace_id"`
				Label     string  `json:"label"`
				Cwd       string  `json:"cwd"`
			}
			if !readBody(w, r, &p) {
				return true
			}
			a.lockedWrite(w, func() (any, error) {
				state, err := a.currentSnapshot(r.Context())
				if err != nil {
					return nil, err
				}
				args := []string{}
				path := r.URL.Path
				paneID := ""
				if p.Key != "" {
					paneID, err = a.paneID(p.Key)
					if err != nil {
						return nil, err
					}
					found := false
					for _, pane := range state.Panes {
						found = found || pane.ID == paneID
					}
					if !found {
						return nil, fmt.Errorf("pane is no longer present")
					}
				}
				hasTab, hasWS := false, false
				for _, tab := range state.Tabs {
					hasTab = hasTab || tab.Tab == p.Tab
				}
				for _, ws := range state.Workspaces {
					hasWS = hasWS || ws.ID == p.Workspace
				}
				if strings.HasPrefix(path, "/api/pane/") && paneID == "" {
					return nil, fmt.Errorf("pane key required")
				}
				switch path {
				case "/api/pane/resize":
					if !map[string]bool{"left": true, "right": true, "up": true, "down": true}[p.Direction] || math.IsNaN(p.Amount) || p.Amount <= 0 || p.Amount > 100 {
						return nil, fmt.Errorf("invalid resize direction or amount")
					}
					args = []string{"pane", "resize", "--pane", paneID, "--direction", p.Direction, "--amount", fmt.Sprint(p.Amount)}
				case "/api/pane/swap":
					other, e := a.paneID(p.With)
					if e != nil {
						return nil, e
					}
					found := false
					for _, pane := range state.Panes {
						found = found || pane.ID == other
					}
					if !found {
						return nil, fmt.Errorf("target pane absent")
					}
					args = []string{"pane", "swap", "--source-pane", paneID, "--target-pane", other}
				case "/api/pane/move":
					if !hasTab {
						return nil, fmt.Errorf("target tab absent")
					}
					args = []string{"pane", "move", paneID, "--tab", p.Tab, "--no-focus"}
				case "/api/pane/zoom":
					args = []string{"pane", "zoom", paneID, "--toggle"}
				case "/api/pane/focus":
					args = []string{"pane", "focus", "--pane", paneID, "--direction", "right"}
					if direct, ok := a.herdr.(*CLI); ok && direct.Socket != nil {
						out, e := direct.Socket.request(r.Context(), "pane.focus", map[string]string{"pane_id": paneID})
						return map[string]any{"ok": e == nil, "message": string(out)}, e
					}
					return nil, fmt.Errorf("exact pane focus requires direct Herdr socket")
				case "/api/tab/create":
					if !hasWS {
						return nil, fmt.Errorf("workspace absent")
					}
					args = []string{"tab", "create", "--workspace", p.Workspace}
					if p.Cwd != "" {
						if !filepath.IsAbs(p.Cwd) {
							return nil, fmt.Errorf("absolute folder required")
						}
						args = append(args, "--cwd", p.Cwd)
					}
				case "/api/tab/rename", "/api/tab/close":
					if !hasTab {
						return nil, fmt.Errorf("tab absent")
					}
					verb := strings.TrimPrefix(path, "/api/tab/")
					args = []string{"tab", verb, p.Tab}
					if verb == "rename" {
						if strings.TrimSpace(p.Label) == "" {
							return nil, fmt.Errorf("label required")
						}
						args = append(args, p.Label)
					}
				case "/api/workspace/rename", "/api/workspace/close", "/api/workspace/focus":
					if !hasWS {
						return nil, fmt.Errorf("workspace absent")
					}
					verb := strings.TrimPrefix(path, "/api/workspace/")
					args = []string{"workspace", verb, p.Workspace}
					if verb == "rename" {
						if strings.TrimSpace(p.Label) == "" {
							return nil, fmt.Errorf("label required")
						}
						args = append(args, p.Label)
					}
				}
				out, e := a.herdr.Call(r.Context(), args...)
				if e != nil {
					return nil, e
				}
				go a.refresh(context.Background(), true)
				return map[string]any{"ok": true, "message": string(out)}, nil
			})
			return true
		}
	}
	switch r.URL.Path {
	case "/api/folders/recent":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		last := ""
		data, err := os.ReadFile(filepath.Join(os.Getenv("APPDATA"), "herdr", "open-folder-last.txt"))
		if err != nil && !os.IsNotExist(err) {
			failHTTP(w, 502, err)
			return true
		}
		last = strings.TrimSpace(string(data))
		folders := []string{}
		seen := map[string]bool{}
		add := func(s string) {
			if s != "" && !seen[strings.ToLower(s)] {
				folders = append(folders, s)
				seen[strings.ToLower(s)] = true
			}
		}
		add(last)
		a.mu.Lock()
		for _, p := range a.state.Panes {
			add(p.Cwd)
		}
		a.mu.Unlock()
		respond(w, 200, map[string]any{"folders": folders, "last": last})
		return true
	case "/api/folders/list":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		path := r.URL.Query().Get("path")
		dirs := []string{}
		if path == "" {
			for letter := 'A'; letter <= 'Z'; letter++ {
				drive := fmt.Sprintf("%c:\\", letter)
				if st, err := os.Stat(drive); err == nil && st.IsDir() {
					dirs = append(dirs, drive)
				}
			}
			respond(w, 200, map[string]any{"path": "", "parent": "", "dirs": dirs})
			return true
		}
		if !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) {
			failHTTP(w, 400, fmt.Errorf("choose an absolute local folder"))
			return true
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			failHTTP(w, 400, err)
			return true
		}
		for _, e := range entries {
			if e.IsDir() && e.Type()&os.ModeSymlink == 0 {
				dirs = append(dirs, e.Name())
				if len(dirs) >= 1000 {
					break
				}
			}
		}
		respond(w, 200, map[string]any{"path": filepath.Clean(path), "parent": filepath.Dir(path), "dirs": dirs})
		return true
	case "/api/workspace/open":
		var p struct {
			Cwd   string `json:"cwd"`
			Agent string `json:"agent"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if !filepath.IsAbs(p.Cwd) || !map[string]bool{"": true, "claude": true, "codex": true, "team": true}[p.Agent] {
				return nil, fmt.Errorf("absolute folder and supported agent required")
			}
			params := map[string]any{"Path": p.Cwd}
			if p.Agent != "" {
				params["Agent"] = p.Agent
			}
			out, err := a.script(r.Context(), "herdr-open-folder.ps1", params, 60*time.Second)
			return map[string]any{"ok": err == nil, "message": out}, err
		})
		return true
	case "/api/columns":
		var p struct {
			Key       string `json:"key"`
			Count     int    `json:"count"`
			Direction string `json:"direction"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			id, err := a.paneID(p.Key)
			if err != nil {
				return nil, err
			}
			if p.Count < 2 || p.Count > 8 || !map[string]bool{"right": true, "down": true}[p.Direction] {
				return nil, fmt.Errorf("2–8 columns/rows and right/down required")
			}
			out, err := a.script(r.Context(), "herdr-columns.ps1", map[string]any{"Count": p.Count, "Direction": p.Direction, "Pane": id}, 30*time.Second)
			go a.refresh(context.Background(), true)
			return map[string]any{"ok": err == nil, "message": out}, err
		})
		return true
	case "/api/layouts":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		presets := []map[string]any{}
		dir := filepath.Join(os.Getenv("APPDATA"), "herdr", "layouts")
		entries, err := os.ReadDir(dir)
		if err != nil && !os.IsNotExist(err) {
			failHTTP(w, 502, err)
			return true
		}
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
				text, e2 := readAllowed(dir, e.Name())
				if e2 != nil {
					failHTTP(w, 502, e2)
					return true
				}
				var plan any
				if e2 = json.Unmarshal([]byte(text), &plan); e2 != nil {
					failHTTP(w, 502, e2)
					return true
				}
				presets = append(presets, map[string]any{"name": strings.TrimSuffix(e.Name(), ".json"), "plan": plan})
			}
		}
		last, err := optionalJSON(filepath.Join(os.Getenv("APPDATA"), "herdr"), "layout-last.json")
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		respond(w, 200, map[string]any{"presets": presets, "last": last})
		return true
	case "/api/layouts/build", "/api/layouts/save":
		var p struct {
			Name string     `json:"name"`
			Plan LayoutPlan `json:"plan"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if err := a.validatePlan(p.Plan); err != nil {
				return nil, err
			}
			if strings.HasSuffix(r.URL.Path, "/save") {
				if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`).MatchString(p.Name) {
					return nil, fmt.Errorf("preset name must be letters, digits, - or _")
				}
				columns := []map[string]string{}
				for _, c := range p.Plan.Columns {
					runs, model, account := "", "", ""
					if c.Type == "new" {
						runs = c.Value
						model = c.Model
						account = c.Account
					}
					columns = append(columns, map[string]string{"folder": c.Folder, "runs": runs, "model": model, "account": account})
				}
				saved := map[string]any{"count": len(columns), "where": p.Plan.Where, "name": p.Plan.Name, "columns": columns}
				data, _ := json.MarshalIndent(saved, "", "  ")
				path := filepath.Join(os.Getenv("APPDATA"), "herdr", "layouts", p.Name+".json")
				if err := atomicWrite(path, data); err != nil {
					return nil, err
				}
				return map[string]any{"ok": true, "message": "Preset saved", "plan": saved}, nil
			}
			path, err := a.tempPlan(p.Plan)
			if err != nil {
				return nil, err
			}
			result, err := a.startJob("herdr-layout.ps1", map[string]any{"PlanFile": path}, 120*time.Second, func() { os.Remove(path) })
			if err != nil {
				os.Remove(path)
			}
			return result, err
		})
		return true
	case "/api/spaces":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		record, err := optionalJSON(filepath.Join(os.Getenv("APPDATA"), "herdr"), "side-by-side.json")
		if err != nil {
			failHTTP(w, 502, err)
		} else {
			respond(w, 200, record)
		}
		return true
	case "/api/spaces/join", "/api/spaces/separate":
		var p struct {
			IDs []string `json:"workspace_ids"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			params := map[string]any{}
			if strings.HasSuffix(r.URL.Path, "/separate") {
				params["Separate"] = true
			} else {
				state, err := a.currentSnapshot(r.Context())
				if err != nil {
					return nil, err
				}
				known := map[string]bool{}
				for _, ws := range state.Workspaces {
					known[ws.ID] = true
				}
				if len(p.IDs) < 2 {
					return nil, fmt.Errorf("select at least two workspaces")
				}
				for _, id := range p.IDs {
					if !known[id] || strings.Contains(id, ",") {
						return nil, fmt.Errorf("unknown workspace")
					}
				}
				params["Spaces"] = strings.Join(p.IDs, ",")
			}
			return a.startJob("herdr-spaces.ps1", params, 120*time.Second, nil)
		})
		return true
	case "/api/stop-all":
		var p struct {
			Confirm string `json:"confirm"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		if p.Confirm != "stop" {
			failHTTP(w, 400, fmt.Errorf("type stop to confirm stopping this Herdr session and every pane"))
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			out, err := a.script(r.Context(), "herdr-stop.ps1", map[string]any{"DeckConfirmedStop": true}, 15*time.Second)
			return map[string]any{"ok": err == nil, "message": out}, err
		})
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/agents/") || strings.HasPrefix(r.URL.Path, "/api/headless") || r.URL.Path == "/api/trial" || r.URL.Path == "/api/sync-skills" {
		return a.toolEndpoint(w, r)
	}
	if r.URL.Path == "/api/docs" && r.URL.Query().Get("name") == "KEYS" {
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		out, err := a.script(r.Context(), "herdr-help.ps1", map[string]any{"List": true}, 15*time.Second)
		if err != nil {
			failHTTP(w, 502, err)
		} else {
			respond(w, 200, map[string]string{"text": out, "name": "KEYS"})
		}
		return true
	}
	return false
}

type LayoutColumn struct{ Folder, Type, Value, Model, Account string }
type LayoutPlan struct {
	Where             int
	Name, WorkspaceID string
	Columns           []LayoutColumn
}

func (a *App) validatePlan(p LayoutPlan) error {
	if p.Where < 0 || p.Where > 1 || len(p.Columns) < 1 || len(p.Columns) > 6 {
		return fmt.Errorf("layout needs 1–6 columns and Where 0 or 1")
	}
	for _, c := range p.Columns {
		if !filepath.IsAbs(c.Folder) {
			return fmt.Errorf("column folder must be absolute")
		}
		switch c.Type {
		case "shell":
		case "new":
			if _, ok := a.profiles[c.Value]; !ok {
				return fmt.Errorf("unknown layout profile")
			}
		case "existing":
			if c.Value == "" {
				return fmt.Errorf("existing pane ID required")
			}
		default:
			return fmt.Errorf("unknown column type")
		}
	}
	return nil
}
func (a *App) tempPlan(plan any) (string, error) {
	dir := filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-plans")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(dir, "plan-*.json")
	if err != nil {
		return "", err
	}
	path := file.Name()
	err = json.NewEncoder(file).Encode(plan)
	closeErr := file.Close()
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return path, closeErr
}
func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "deck-write-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func optionalJSON(root, file string) (any, error) {
	text, err := readAllowed(root, file)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var value any
	err = json.Unmarshal([]byte(text), &value)
	return value, err
}
