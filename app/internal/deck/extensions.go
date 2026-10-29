package deck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

type paneRequest struct {
	Key         string   `json:"key"`
	Text        string   `json:"text"`
	Label       string   `json:"label"`
	Keys        []string `json:"keys"`
	Name        string   `json:"name"`
	Tab         string   `json:"tab_id"`
	Cwd         string   `json:"cwd"`
	Profile     string   `json:"profile"`
	Model       string   `json:"model"`
	Effort      string   `json:"effort"`
	Account     string   `json:"account"`
	Direction   string   `json:"direction"`
	Worktree    any      `json:"worktree"`
	Helper      bool     `json:"helper"`
	LeadAccount string   `json:"lead_account"`
	Purpose     string   `json:"purpose"`
	Route       string   `json:"route"`
	Floor       *struct {
		RAMFreeMB  float64            `json:"ram_free_mb"`
		DiskFreeGB map[string]float64 `json:"disk_free_gb"`
	} `json:"floor"`
}

func (a *App) paneID(key string) (string, error) {
	a.mu.Lock()
	session := a.state.Session
	a.mu.Unlock()
	prefix := session + "|"
	if !strings.HasPrefix(key, prefix) || len(key) <= len(prefix) {
		return "", fmt.Errorf("pane key belongs to another session or is invalid")
	}
	return strings.TrimPrefix(key, prefix), nil
}
func (a *App) lockedWrite(w http.ResponseWriter, fn func() (any, error)) {
	a.actionMu.Lock()
	defer a.actionMu.Unlock()
	a.mu.Lock()
	locked := a.readOnly
	a.mu.Unlock()
	if locked {
		failHTTP(w, 403, fmt.Errorf("read-only mode is enabled"))
		return
	}
	result, err := fn()
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
	respond(w, 200, result)
}
func (a *App) currentSnapshot(ctx context.Context) (State, error) {
	raw, err := a.herdr.Call(ctx, "api", "snapshot")
	if err != nil {
		return State{}, err
	}
	a.mu.Lock()
	session := a.state.Session
	a.mu.Unlock()
	return parseSnapshot(raw, session)
}
func (a *App) extensions(w http.ResponseWriter, r *http.Request) bool {
	if a.guardEndpoint(w, r) {
		return true
	}
	if a.parity(w, r) {
		return true
	}
	aliases := map[string]string{"/api/pane/prompt": "prompt", "/api/pane/keys": "keys", "/api/pane/text": "paste", "/api/pane/rename": "rename", "/api/pane/close": "close", "/api/pane/split": "shell", "/api/shell": "shell", "/api/agent/start": "start", "/api/agent/stop": "stop", "/api/agent/focus": "focus", "/api/herdr/start": "launch", "/api/herdr/reload-config": "reload", "/api/tab/focus": "tab-focus"}
	if action, ok := aliases[r.URL.Path]; ok {
		var p paneRequest
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if action == "start" {
				if err := a.launchGuard(p); err != nil {
					return nil, err
				}
			}
			v := Action{Action: action, Text: p.Text, Cwd: p.Cwd, Profile: p.Profile, Model: p.Model, Effort: p.Effort, Account: p.Account, Confirm: true, Direction: p.Direction}
			if p.Worktree != nil {
				switch value := p.Worktree.(type) {
				case bool:
					if value {
						v.Worktree = fmt.Sprintf("deck-%d", time.Now().UnixNano())
					}
				case string:
					v.Worktree = value
				default:
					return nil, fmt.Errorf("worktree must be a bool or branch name")
				}
			}
			if action == "rename" {
				v.Text = p.Label
			}
			if p.Key != "" {
				id, err := a.paneID(p.Key)
				if err != nil {
					return nil, err
				}
				v.Pane = id
			}
			if p.Name != "" || action == "tab-focus" {
				state, err := a.currentSnapshot(r.Context())
				if err != nil {
					return nil, err
				}
				found := false
				for _, pane := range state.Panes {
					if (action == "tab-focus" && pane.Tab == p.Tab) || (p.Name != "" && pane.Name == p.Name) {
						v.Pane = pane.ID
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("requested agent/tab no longer exists")
				}
				if action == "tab-focus" {
					v.Action = "focus"
				}
			}
			var result string
			if action == "keys" {
				if len(p.Keys) < 1 || len(p.Keys) > 32 {
					return nil, fmt.Errorf("supply 1–32 supported keys")
				}
				normalized := []string{}
				names := map[string]string{"Escape": "esc", "C-c": "ctrl+c", "Enter": "enter", "Up": "up", "Down": "down", "Tab": "tab", "Left": "left", "Right": "right", "Backspace": "backspace"}
				for _, key := range p.Keys {
					if k, ok := names[key]; ok {
						key = k
					}
					if !validKey(key) {
						return nil, fmt.Errorf("unsupported key %q; no keys sent", key)
					}
					normalized = append(normalized, key)
				}
				// Validate the pane before sending the entire validated key batch in one CLI call.
				state, err := a.currentSnapshot(r.Context())
				if err != nil {
					return nil, err
				}
				exists := false
				for _, pane := range state.Panes {
					exists = exists || pane.ID == v.Pane
				}
				if !exists {
					return nil, fmt.Errorf("pane no longer exists")
				}
				args := append([]string{"pane", "send-keys", v.Pane}, normalized...)
				out, err := a.herdr.Call(r.Context(), args...)
				if err != nil {
					return nil, err
				}
				result = string(out)
			} else {
				var err error
				result, err = a.action(r.Context(), v)
				if err != nil {
					return nil, err
				}
			}
			a.mu.Lock()
			dispatch := a.lastDispatch
			a.mu.Unlock()
			go a.refresh(context.Background(), true)
			response := map[string]any{"ok": true, "message": result, "output": result}
			if dispatch != "" {
				response["dispatch_id"] = dispatch
			}
			return response, nil
		})
		return true
	}
	switch r.URL.Path {
	case "/api/visible":
		var p struct {
			Keys []string `json:"keys"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		if len(p.Keys) > 256 {
			failHTTP(w, 400, fmt.Errorf("too many pane keys"))
			return true
		}
		ids := []string{}
		for _, key := range p.Keys {
			id, err := a.paneID(key)
			if err != nil {
				failHTTP(w, 400, err)
				return true
			}
			ids = append(ids, id)
		}
		a.mu.Lock()
		a.active = len(ids) > 0
		a.visible = map[string]bool{}
		for _, id := range ids {
			a.visible[id] = true
		}
		a.mu.Unlock()
		respond(w, 200, map[string]bool{"ok": true})
		return true
	case "/api/tasks", "/api/tasks/read", "/api/runs", "/api/runs/read", "/api/docs":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		result, err := a.readExtension(r)
		if err != nil {
			failHTTP(w, 400, err)
		} else {
			respond(w, 200, result)
		}
		return true
	case "/api/health":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		result, err := systemHealth()
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		a.mu.Lock()
		result["panes"] = len(a.state.Panes)
		panes := append([]Pane{}, a.state.Panes...)
		a.mu.Unlock()
		memory := []map[string]any{}
		for _, p := range panes {
			raw, e := a.herdr.Call(r.Context(), "pane", "process-info", p.ID)
			row := map[string]any{"key": p.Key}
			if e != nil {
				row["error"] = e.Error()
			} else {
				var v struct {
					Result struct {
						Info struct {
							Shell      uint32 `json:"shell_pid"`
							Foreground []struct {
								PID uint32 `json:"pid"`
							} `json:"foreground_processes"`
						} `json:"process_info"`
					} `json:"result"`
				}
				if e = json.Unmarshal(raw, &v); e != nil {
					row["error"] = e.Error()
				} else {
					seen := map[uint32]bool{}
					var total uint64
					pids := []uint32{v.Result.Info.Shell}
					for _, proc := range v.Result.Info.Foreground {
						pids = append(pids, proc.PID)
					}
					for _, pid := range pids {
						if pid == 0 || seen[pid] {
							continue
						}
						seen[pid] = true
						bytes, err := processMemory(pid)
						if err != nil {
							row["error"] = err.Error()
							continue
						}
						total += bytes
					}
					row["mb"] = float64(total) / (1 << 20)
				}
			}
			memory = append(memory, row)
		}
		result["panes_mem"] = memory
		result["goroutines"] = runtime.NumGoroutine()
		result["handles"] = processHandles()
		if bytes, e := processMemory(uint32(os.Getpid())); e == nil {
			result["backend_rss_mb"] = float64(bytes) / (1 << 20)
		} else {
			result["backend_memory_error"] = e.Error()
		}
		respond(w, 200, result)
		return true
	case "/api/panes/close-finished":
		var p struct {
			Keys []string `json:"keys"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			if len(p.Keys) == 0 || len(p.Keys) > 128 {
				return nil, fmt.Errorf("select 1–128 finished panes")
			}
			state, err := a.currentSnapshot(r.Context())
			if err != nil {
				return nil, err
			}
			ids := []string{}
			seen := map[string]bool{}
			for _, key := range p.Keys {
				id, e := a.paneID(key)
				if e != nil {
					return nil, e
				}
				if seen[id] {
					return nil, fmt.Errorf("duplicate pane")
				}
				seen[id] = true
				valid := false
				for _, pane := range state.Panes {
					if pane.ID == id {
						valid = pane.Status == "idle" || pane.Status == "done"
					}
				}
				if !valid {
					return nil, fmt.Errorf("pane %s is absent or no longer idle/finished; nothing closed", id)
				}
				ids = append(ids, id)
			}
			closed := []string{}
			for _, id := range ids {
				if _, e := a.herdr.Call(r.Context(), "pane", "close", id); e != nil {
					return nil, fmt.Errorf("closed %v; failed %s: %w", closed, id, e)
				}
				closed = append(closed, id)
			}
			go a.refresh(context.Background(), true)
			return map[string]any{"ok": true, "closed": closed}, nil
		})
		return true
	case "/api/signin":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		respond(w, 200, a.signin(r.Context()))
		return true
	case "/api/signin/open":
		var p struct {
			Tool    string `json:"tool"`
			Account string `json:"account"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			allowed := map[string]string{"claude": "claude", "codex": "codex", "cline": "cline", "grok": "grok", "agy": "agy", "cursor": "cursor"}
			profile, ok := allowed[p.Tool]
			if !ok {
				return nil, fmt.Errorf("unsupported sign-in tool")
			}
			if p.Account != "" && !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`).MatchString(p.Account) {
				return nil, fmt.Errorf("invalid account")
			}
			params := map[string]string{"Command": "login", "Target": profile}
			if p.Account != "" {
				params["Account"] = p.Account
			}
			out, err := a.herdr.Helper(r.Context(), params)
			if err != nil {
				return nil, err
			}
			return map[string]any{"ok": true, "message": out}, nil
		})
		return true
	case "/api/git":
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		id, err := a.paneID(r.URL.Query().Get("key"))
		if err != nil {
			failHTTP(w, 400, err)
			return true
		}
		state, err := a.currentSnapshot(r.Context())
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		cwd := ""
		for _, p := range state.Panes {
			if p.ID == id {
				cwd = p.Cwd
			}
		}
		if cwd == "" {
			failHTTP(w, 404, fmt.Errorf("pane folder unavailable"))
			return true
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		status, err := runCommand(ctx, "git.exe", "", "--no-optional-locks", "-C", cwd, "status", "--short")
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		stat, err := runCommand(ctx, "git.exe", "", "--no-optional-locks", "-C", cwd, "diff", "--no-ext-diff", "--stat")
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		top, err := runCommand(ctx, "git.exe", "", "--no-optional-locks", "-C", cwd, "rev-parse", "--show-toplevel")
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		porcelain, err := runCommand(ctx, "git.exe", "", "--no-optional-locks", "-C", cwd, "status", "--porcelain=v1", "-z", "--untracked-files=all")
		if err != nil {
			failHTTP(w, 502, err)
			return true
		}
		respond(w, 200, map[string]any{"status": string(status), "stat": string(stat), "text": string(status) + "\n" + string(stat), "root": filepath.ToSlash(strings.TrimSpace(string(top))), "paths": parsePorcelainZ(porcelain)})
		return true
	case "/api/check":
		var p struct{}
		if !readBody(w, r, &p) {
			return true
		}
		// The fixed check script is read-only. A failing check is a result, never a fake success.
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		script := `$p=[Console]::In.ReadToEnd()|ConvertFrom-Json; & $p.script 2>&1|ForEach-Object{"$_"}; exit $LASTEXITCODE`
		payload, _ := json.Marshal(map[string]string{"script": filepath.Join(a.root, "check.ps1")})
		out, err := runCommand(ctx, "pwsh.exe", string(payload), "-NoProfile", "-EncodedCommand", encodedPS(script))
		text := string(out)
		if err != nil {
			text = err.Error()
		}
		text = ansiPattern.ReplaceAllString(text, "")
		respond(w, 200, map[string]any{"ok": err == nil, "lines": strings.Split(strings.TrimSpace(text), "\n"), "text": text})
		return true
	case "/api/lessons/append":
		var p struct {
			Text string `json:"text"`
		}
		if !readBody(w, r, &p) {
			return true
		}
		a.lockedWrite(w, func() (any, error) {
			text := strings.TrimSpace(p.Text)
			if text == "" || len(text) > 4000 || strings.ContainsRune(text, 0) {
				return nil, fmt.Errorf("lesson must be 1–4000 bytes")
			}
			path := filepath.Join(a.root, "LESSONS.md")
			old, err := readAllowed(a.root, "LESSONS.md")
			if err != nil {
				return nil, err
			}
			matches := regexp.MustCompile(`(?m)^(\d+)\. `).FindAllStringSubmatch(old, -1)
			number := 1
			for _, m := range matches {
				var n int
				fmt.Sscan(m[1], &n)
				if n >= number {
					number = n + 1
				}
			}
			appendText := fmt.Sprintf("\n%d. %s\n", number, strings.ReplaceAll(text, "\n", "\n    "))
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				return nil, err
			}
			n, e := file.WriteString(appendText)
			closeErr := file.Close()
			if e != nil {
				return nil, e
			}
			if closeErr != nil {
				return nil, closeErr
			}
			if n != len(appendText) {
				return nil, io.ErrShortWrite
			}
			readBack, e := readAllowed(a.root, "LESSONS.md")
			if e != nil {
				return nil, e
			}
			if !strings.HasSuffix(readBack, appendText) {
				return nil, fmt.Errorf("lesson append could not be confirmed")
			}
			return map[string]any{"ok": true, "number": number, "message": "Lesson appended"}, nil
		})
		return true
	}
	return false
}

// parsePorcelainZ reads `git status --porcelain=v1 -z`: one "XY path" per NUL field, with the new
// name first and the original name in the following field for renames and copies.
func parsePorcelainZ(raw []byte) []map[string]string {
	paths := []map[string]string{}
	fields := strings.Split(string(raw), "\x00")
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if len(field) < 4 {
			continue
		}
		if field[0] == 'R' || field[0] == 'C' {
			i++
		}
		paths = append(paths, map[string]string{"path": filepath.ToSlash(field[3:]), "code": field[:2]})
	}
	return paths
}
func readAllowed(root, relative string) (string, error) {
	if filepath.IsAbs(relative) || relative == "" {
		return "", fmt.Errorf("invalid relative file")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file traversal is forbidden")
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(root, clean))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("linked file leaves the allowed folder")
	}
	file, err := os.Open(target)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return "", fmt.Errorf("file is not a regular document under 1 MB")
	}
	data, err := io.ReadAll(io.LimitReader(file, 1<<20+1))
	if err != nil {
		return "", err
	}
	if len(data) > 1<<20 {
		return "", fmt.Errorf("document grew beyond 1 MB")
	}
	return string(data), nil
}
func (a *App) readExtension(r *http.Request) (any, error) {
	path := r.URL.Path
	if path == "/api/docs" {
		docs := map[string]string{"HISTORY": "HISTORY.md", "LESSONS": "LESSONS.md", "AGENT-ROSTER": "docs/AGENT-ROSTER.md", "TEAM-PLAYBOOK": "docs/TEAM-PLAYBOOK.md", "GUIDE": "docs/GUIDE.md", "herdr-team": "skills/herdr-team/SKILL.md", "herdr-workspace": "skills/herdr-workspace/SKILL.md"}
		name := r.URL.Query().Get("name")
		file, ok := docs[name]
		if !ok {
			return nil, fmt.Errorf("unknown document")
		}
		text, err := readAllowed(a.root, file)
		return map[string]string{"name": name, "file": file, "text": text}, err
	}
	runs := strings.HasPrefix(path, "/api/runs")
	root := filepath.Join(a.root, "tasks")
	prefix := "tasks/"
	field := "tasks"
	if runs {
		root = filepath.Join(os.Getenv("APPDATA"), "herdr", "agent-runs")
		prefix = ""
		field = "runs"
	}
	if strings.HasSuffix(path, "/read") {
		file := r.URL.Query().Get("file")
		relative := strings.TrimPrefix(file, prefix)
		if (!runs && (!strings.HasPrefix(file, prefix) || strings.ContainsAny(relative, `/\`))) || strings.ToLower(filepath.Ext(relative)) != ".md" {
			return nil, fmt.Errorf("only listed Markdown documents are readable")
		}
		text, err := readAllowed(root, relative)
		return map[string]string{"file": file, "text": text}, err
	}
	list := []map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if !runs && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		info, e := entry.Info()
		if e != nil {
			return e
		}
		title := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		list = append(list, map[string]string{"file": prefix + filepath.ToSlash(rel), "title": title, "mtime": info.ModTime().Format(time.RFC3339)})
		return nil
	})
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i]["mtime"] > list[j]["mtime"] })
	if len(list) > 500 {
		list = list[:500]
	}
	return map[string]any{field: list}, nil
}
func (a *App) signin(ctx context.Context) map[string]any {
	rows := []map[string]any{}
	for _, tool := range []string{"claude", "codex", "cline", "grok", "agy", "cursor"} {
		var signed any
		detail := "not verified: no supported read-only sign-in probe; use the tool's login pane"
		argv := []string{}
		switch tool {
		case "claude":
			argv = []string{"auth", "status"}
		case "codex":
			argv = []string{"login", "status"}
		case "cursor":
			argv = []string{"status"}
		}
		if len(argv) > 0 {
			payload, _ := json.Marshal(map[string]any{"command": map[string]string{"cursor": "cursor-agent"}[tool], "args": argv})
			if tool != "cursor" {
				payload, _ = json.Marshal(map[string]any{"command": tool, "args": argv})
			}
			script := `$p=[Console]::In.ReadToEnd()|ConvertFrom-Json; & $p.command @($p.args); exit $LASTEXITCODE`
			probe, cancel := context.WithTimeout(ctx, 5*time.Second)
			out, err := runCommand(probe, "pwsh.exe", string(payload), "-NoProfile", "-NonInteractive", "-EncodedCommand", encodedPS(script))
			cancel()
			if err == nil {
				signed = true
				detail = "Read-only status command succeeded"
				if tool == "claude" {
					var v struct {
						LoggedIn bool `json:"loggedIn"`
					}
					if json.Unmarshal(out, &v) == nil {
						signed = v.LoggedIn
					}
				}
			} else {
				detail = "not verified: status command failed or timed out (sign in in the tool's pane)"
			}
		}
		rows = append(rows, map[string]any{"tool": tool, "account": "", "signed_in": signed, "detail": detail})
	}
	return map[string]any{"tools": rows}
}
