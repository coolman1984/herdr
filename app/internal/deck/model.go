package deck

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Pane struct {
	ID        string            `json:"pane_id"`
	Key       string            `json:"key"`
	Workspace string            `json:"workspace_id"`
	Tab       string            `json:"tab_id"`
	Cwd       string            `json:"cwd"`
	Label     string            `json:"label"`
	Name      string            `json:"name"`
	Agent     string            `json:"agent"`
	Title     string            `json:"title"`
	Status    string            `json:"agent_status"`
	Tokens    map[string]string `json:"tokens"`
	Scroll    struct {
		Rows int `json:"viewport_rows"`
	} `json:"scroll"`
	Screen        string `json:"screen"`
	Stale         bool   `json:"stale"`
	SourceStatus  string `json:"-"`
	Error         string `json:"error"`
	Blocked       bool   `json:"blocked"`
	Revision      uint64 `json:"revision"`
	Focused       bool   `json:"focused"`
	ScreenVersion uint64 `json:"screen_version"`
}
type Nav struct {
	ID        string            `json:"workspace_id"`
	Tab       string            `json:"tab_id"`
	Workspace string            `json:"parent_workspace"`
	Label     string            `json:"label"`
	Status    string            `json:"agent_status"`
	Focused   bool              `json:"focused"`
	Tokens    map[string]string `json:"tokens,omitempty"`
}
type State struct {
	Session    string `json:"session"`
	Panes      []Pane `json:"panes"`
	Workspaces []Nav  `json:"workspaces"`
	Tabs       []Nav  `json:"tabs"`
	Error      string `json:"error"`
	Herdr      struct {
		Running bool   `json:"running"`
		Error   string `json:"error"`
	} `json:"herdr"`
	Profiles    []Profile       `json:"profiles"`
	Layouts     json.RawMessage `json:"layouts"`
	FocusedPane string          `json:"focused_pane_id"`
	ReadOnly    bool            `json:"read_only"`
	Lead        LeadIdentity    `json:"lead"`
	Root        string          `json:"root"` // the toolkit folder; the page's default folder
}

func parseSnapshot(raw []byte, session string) (State, error) {
	var envelope struct {
		Result struct {
			Snapshot *struct {
				Panes       *[]Pane         `json:"panes"`
				Workspaces  []Nav           `json:"workspaces"`
				Tabs        []Nav           `json:"tabs"`
				Agents      []Pane          `json:"agents"`
				Layouts     json.RawMessage `json:"layouts"`
				FocusedPane string          `json:"focused_pane_id"`
			} `json:"snapshot"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return State{}, fmt.Errorf("invalid Herdr snapshot: %w", err)
	}
	snapshot := envelope.Result.Snapshot
	if snapshot == nil || snapshot.Panes == nil {
		return State{}, fmt.Errorf("Herdr response has no snapshot panes")
	}
	s := State{Session: session, Panes: *snapshot.Panes, Workspaces: snapshot.Workspaces, Tabs: snapshot.Tabs, Layouts: snapshot.Layouts, FocusedPane: snapshot.FocusedPane}
	s.Herdr.Running = true
	for i := range s.Tabs {
		s.Tabs[i].Workspace = s.Tabs[i].ID
	}
	for i := range s.Panes {
		p := &s.Panes[i]
		p.Key = session + "|" + p.ID
		if p.ID == "" {
			return State{}, fmt.Errorf("Herdr returned an empty pane ID")
		}
		p.Blocked = p.Status == "blocked"
		if p.Tokens == nil {
			p.Tokens = map[string]string{}
		}
		for _, a := range snapshot.Agents {
			if a.ID == p.ID {
				p.Name = a.Name
				if a.Agent != "" {
					p.Agent = a.Agent
				}
				if a.Status != "" {
					p.Status = a.Status
				}
				for k, v := range a.Tokens {
					p.Tokens[k] = v
				}
			}
		}
	}
	for i := range s.Panes {
		s.Panes[i].SourceStatus = s.Panes[i].Status
	}
	return s, nil
}
func parseScreen(raw []byte, rows int) (string, error) {
	text := string(raw)
	if len(text) > 512<<10 {
		text = text[len(text)-(512<<10):]
	}
	// pane read emits terminal text on this CLI, while test stubs may return an API envelope.
	if strings.HasPrefix(strings.TrimSpace(text), `{"result":`) || strings.HasPrefix(strings.TrimSpace(text), `{"error":`) {
		var v struct {
			Result struct {
				Output *string `json:"output"`
			} `json:"result"`
		}
		if err := json.Unmarshal(raw, &v); err != nil || v.Result.Output == nil {
			return "", fmt.Errorf("invalid Herdr screen response")
		}
		text = *v.Result.Output
	}
	if rows < 1 {
		rows = 40
	}
	if rows > 500 {
		rows = 500
	}
	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	if len(lines) > 2000 {
		lines = lines[len(lines)-2000:]
	}
	return strings.Join(lines, "\n"), nil
}
func blockedScreen(screen string) bool {
	// Match only the live footer; prose mentioning trust must not block an agent.
	clean := ansiPattern.ReplaceAllString(screen, "")
	lines := []string{}
	for _, l := range strings.Split(strings.ToLower(clean), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	tail := lines
	if len(tail) > 6 {
		tail = tail[len(tail)-6:]
	}
	recent := strings.Join(tail, "\n")
	all := strings.Join(lines, "\n")
	for _, v := range []string{"not logged in", "not signed in", "please sign in", "please log in", "run /login"} {
		if strings.Contains(recent, v) {
			return true
		}
	}
	footer := false
	for _, v := range []string{"enter to confirm", "enter confirm", "navigate", "esc to"} {
		footer = footer || strings.Contains(recent, v)
	}
	if footer {
		for _, v := range []string{"do you trust", "trust this folder", "use my browser", "browser tools", "select a model", "choose a model", "pick a model"} {
			if strings.Contains(all, v) {
				return true
			}
		}
	}
	return false
}
