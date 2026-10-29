package deck

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) toolEndpoint(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/headless/routes" {
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		respond(w, 200, map[string]any{"routes": []map[string]any{}, "note": "No server-search-only research route has been verified for the Deck; research stays unavailable."})
		return true
	}
	command := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	reads := map[string]bool{"tools": true, "status": true, "read": true, "diff": true, "history": true, "collect": true, "watch": true, "accounts": true, "usage": true}
	if reads[command] {
		if r.Method != "GET" {
			failHTTP(w, 405, fmt.Errorf("GET required"))
			return true
		}
		params := map[string]any{"Command": command}
		if name := r.URL.Query().Get("name"); name != "" {
			params["Target"] = name
		}
		if command == "read" || command == "diff" {
			if params["Target"] == nil {
				failHTTP(w, 400, fmt.Errorf("agent name required"))
				return true
			}
		}
		if command == "watch" {
			params["Once"] = true
		}
		if command == "collect" {
			names := strings.Split(r.URL.Query().Get("names"), ",")
			if len(names) == 0 || names[0] == "" {
				failHTTP(w, 400, fmt.Errorf("names query required"))
				return true
			}
			params["Agents"] = names
			a.lockedWrite(w, func() (any, error) { return a.startJob("herdr-agents.ps1", params, 15*time.Minute, nil) })
			return true
		}
		out, err := a.script(r.Context(), "herdr-agents.ps1", params, 20*time.Second)
		if err != nil {
			failHTTP(w, 502, err)
		} else {
			respond(w, 200, map[string]string{"output": out, "text": out})
		}
		return true
	}
	supported := map[string]bool{"/api/agents/ask": true, "/api/agents/fanout": true, "/api/agents/team": true, "/api/agents/handoff": true, "/api/headless": true, "/api/trial": true, "/api/sync-skills": true}
	if !supported[r.URL.Path] {
		return false
	}
	var p struct {
		Name       string   `json:"name"`
		Text       string   `json:"text"`
		Fresh      bool     `json:"fresh"`
		Names      []string `json:"names"`
		Preset     string   `json:"preset"`
		Cwd        string   `json:"cwd"`
		Profile    string   `json:"profile"`
		Model      string   `json:"model"`
		Effort     string   `json:"effort"`
		Account    string   `json:"account"`
		Purpose    string   `json:"purpose"`
		Run        string   `json:"run"`
		TrialRoot  string   `json:"trial_root"`
		TimeoutSec int      `json:"timeout_sec"`
		TimeoutMin int      `json:"timeout_min"`
	}
	if !readBody(w, r, &p) {
		return true
	}
	a.lockedWrite(w, func() (any, error) {
		if len(p.Text) > 64<<10 || strings.ContainsRune(p.Text, 0) {
			return nil, fmt.Errorf("invalid task text")
		}
		script := "herdr-agents.ps1"
		params := map[string]any{}
		timeout := 15 * time.Minute
		var cleanup func()
		switch r.URL.Path {
		case "/api/agents/ask":
			if p.Name == "" || strings.TrimSpace(p.Text) == "" {
				return nil, fmt.Errorf("agent name and text required")
			}
			state, err := a.currentSnapshot(r.Context())
			if err != nil {
				return nil, err
			}
			found := false
			for _, pane := range state.Panes {
				if pane.Name == p.Name {
					found = true
					if pane.Status == "blocked" {
						return nil, fmt.Errorf("blocked: agent needs the owner; no ask submitted")
					}
					if err := a.checkInput(r.Context(), pane); err != nil {
						return nil, err
					}
				}
			}
			if !found {
				return nil, fmt.Errorf("agent absent")
			}
			params = map[string]any{"Command": "ask", "Target": p.Name, "Task": p.Text}
			if p.Fresh {
				params["Fresh"] = true
			}
		case "/api/agents/fanout":
			if len(p.Names) < 1 || len(p.Names) > 8 || strings.TrimSpace(p.Text) == "" {
				return nil, fmt.Errorf("select 1–8 agents and a task")
			}
			state, err := a.currentSnapshot(r.Context())
			if err != nil {
				return nil, err
			}
			for _, name := range p.Names {
				for _, pane := range state.Panes {
					if pane.Name == name {
						if err := a.checkInput(r.Context(), pane); err != nil {
							return nil, err
						}
					}
				}
			}
			params = map[string]any{"Command": "fanout", "Task": p.Text, "Agents": p.Names}
		case "/api/agents/team":
			if !filepath.IsAbs(p.Cwd) {
				return nil, fmt.Errorf("absolute team folder required")
			}
			profiles := []string{"claude", "codex"}
			if p.Preset == "claude-codex-gemini" {
				profiles = append(profiles, "gemini")
			} else if p.Preset != "claude-codex" {
				return nil, fmt.Errorf("unknown team preset")
			}
			agents := []map[string]string{}
			for _, name := range profiles {
				if _, ok := a.profiles[name]; !ok {
					return nil, fmt.Errorf("team profile unavailable")
				}
				account := ""
				if a.profiles[name].Kind == "claude" {
					account = p.Account
				}
				agents = append(agents, map[string]string{"name": fmt.Sprintf("deck-%s-%d", name, time.Now().UnixMilli()), "agent": name, "cwd": p.Cwd, "task": p.Text, "account": account})
			}
			path, err := a.tempPlan(map[string]any{"agents": agents})
			if err != nil {
				return nil, err
			}
			params = map[string]any{"Command": "team", "PlanFile": path}
			cleanup = func() { os.Remove(path) }
		case "/api/agents/handoff":
			dir := filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-handoffs")
			if err := os.MkdirAll(dir, 0700); err != nil {
				return nil, err
			}
			path := filepath.Join(dir, fmt.Sprintf("handoff-%d.md", time.Now().UnixNano()))
			params = map[string]any{"Command": "handoff", "TaskFile": path, "DeckHandoffText": p.Text}
		case "/api/headless":
			if p.Purpose == "research" {
				return nil, &statusError{Status: 409, Msg: "research route unavailable: no verified server-search route"}
			}
			profile, ok := a.profiles[p.Profile]
			if !ok || profile.Kind != "cline" {
				return nil, fmt.Errorf("headless requires a Cline profile")
			}
			if strings.TrimSpace(p.Text) == "" {
				return nil, fmt.Errorf("task text required")
			}
			seconds := p.TimeoutSec
			if seconds == 0 {
				seconds = 1800
			}
			if seconds < 1 || seconds > 86400 {
				return nil, fmt.Errorf("timeout_sec must be 1–86400")
			}
			timeout = time.Duration(seconds+35) * time.Second
			script = "herdr-headless.ps1"
			params = map[string]any{"AgentProfile": p.Profile, "Task": p.Text, "TimeoutSec": seconds}
		case "/api/trial":
			if _, ok := a.profiles[p.Profile]; !ok {
				return nil, fmt.Errorf("unknown trial profile")
			}
			minutes := p.TimeoutMin
			if minutes == 0 {
				minutes = 60
			}
			if minutes < 1 || minutes > 1440 {
				return nil, fmt.Errorf("timeout_min must be 1–1440")
			}
			timeout = time.Duration(minutes+1) * time.Minute
			script = "herdr-trial.ps1"
			params = map[string]any{"AgentProfile": p.Profile, "TimeoutMin": minutes}
			if p.Run != "" {
				params["Run"] = p.Run
			}
			if p.TrialRoot != "" {
				if !filepath.IsAbs(p.TrialRoot) {
					return nil, fmt.Errorf("absolute trial_root required")
				}
				params["TrialRoot"] = p.TrialRoot
			}
		case "/api/sync-skills":
			script = "sync-skills.ps1"
			timeout = 60 * time.Second
		}
		if script == "herdr-headless.ps1" || script == "herdr-trial.ps1" {
			if p.Model != "" {
				params["Model"] = p.Model
			}
			if p.Effort != "" {
				params["Effort"] = p.Effort
			}
			if script == "herdr-headless.ps1" {
				if p.Cwd != "" {
					if !filepath.IsAbs(p.Cwd) {
						return nil, fmt.Errorf("absolute folder required")
					}
					params["Cwd"] = p.Cwd
				}
				if p.Account != "" {
					params["Account"] = p.Account
				}
			}
		}
		result, err := a.startJob(script, params, timeout, cleanup)
		if err != nil && cleanup != nil {
			cleanup()
		}
		return result, err
	})
	return true
}
