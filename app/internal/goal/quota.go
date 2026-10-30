package goal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Quota policy (docs/COMPANY-ENVIRONMENT-POLICY.md): an account gets new work only while every limit it
// reports is under 90% used. Unknown is never read as 0%: an account with no reading is "unknown" and gets
// at most one agent at a time (low trust). Limits belong to the real account (provider + account id), so
// several profiles on one subscription share one budget.

const QuotaThreshold = 90.0

// Quota states.
const (
	QuotaKnown       = "known"
	QuotaUnknown     = "unknown"     // no reading could be made
	QuotaUnsupported = "unsupported" // the provider exposes no limits we can read
	QuotaStale       = "stale"       // a reading older than QuotaMaxAge
)

var QuotaMaxAge = 30 * time.Minute

type Limit struct {
	Kind     string    `json:"kind"`               // five_hour, weekly, monthly_credits, ...
	UsedPct  *float64  `json:"used_pct,omitempty"` // nil when the source gives no percentage
	Left     *float64  `json:"left,omitempty"`     // remaining in Unit, when that is what the source gives
	Unit     string    `json:"unit,omitempty"`
	ResetsAt time.Time `json:"resets_at,omitempty"`
}

type QuotaReading struct {
	Provider string    `json:"provider"`
	Account  string    `json:"account"` // non-secret account name
	State    string    `json:"state"`
	Limits   []Limit   `json:"limits,omitempty"`
	Observed time.Time `json:"observed"` // when the source wrote it
	Source   string    `json:"source"`
	Note     string    `json:"note,omitempty"`
}

// Verdict is what the scheduler needs from a reading.
type Verdict struct {
	Eligible  bool
	LowTrust  bool      // unknown/unsupported/stale: at most one agent on this account
	Until     time.Time // when an over-threshold limit resets (zero if unknown)
	Reason    string
	Breaching string // the limit at or over 90%
}

// Judge applies the 90% rule to one reading at time now.
func Judge(r QuotaReading, now time.Time) Verdict {
	switch r.State {
	case QuotaUnknown, QuotaUnsupported:
		return Verdict{Eligible: true, LowTrust: true, Reason: r.State + ": " + r.Note}
	}
	if r.State == QuotaStale || now.Sub(r.Observed) > QuotaMaxAge {
		v := Verdict{Eligible: true, LowTrust: true, Reason: fmt.Sprintf("stale reading from %s", r.Observed.Format(time.RFC3339))}
		// A stale reading that already shows a limit reached still blocks until its reset.
		for _, l := range r.Limits {
			if l.UsedPct != nil && *l.UsedPct >= QuotaThreshold && l.ResetsAt.After(now) {
				return Verdict{Until: l.ResetsAt, Reason: v.Reason + " at " + l.Kind, Breaching: l.Kind}
			}
		}
		return v
	}
	v := Verdict{Eligible: true}
	for _, l := range r.Limits {
		if l.UsedPct == nil {
			continue
		}
		used := *l.UsedPct
		if !l.ResetsAt.IsZero() && !l.ResetsAt.After(now) {
			used = 0 // the window has reset since the reading
		}
		if used >= QuotaThreshold {
			v.Eligible = false
			v.Breaching = l.Kind
			v.Reason = fmt.Sprintf("%s at %.1f%% (>= %.0f%%)", l.Kind, used, QuotaThreshold)
			if l.ResetsAt.After(v.Until) {
				v.Until = l.ResetsAt
			}
		}
	}
	if len(r.Limits) == 0 {
		return Verdict{Eligible: true, LowTrust: true, Reason: "reading has no limits"}
	}
	return v
}

// QuotaSource reads one provider's accounts.
type QuotaSource interface {
	Read(account string) QuotaReading
}

// ---------- Claude Code: the status-line files ----------

// ClaudeStatusFiles reads %APPDATA%\herdr\claude-cost\*.json, written by the owner's status line on each
// refresh (claude-limits.ps1 reads the same files): rate_limits.five_hour / seven_day with
// used_percentage and resets_at, and config_dir naming the account. No sign-in file is opened.
type ClaudeStatusFiles struct {
	Dir         string // default: %APPDATA%/herdr/claude-cost
	AccountsDir string // default: ~/.herdr-accounts/claude
}

func (c ClaudeStatusFiles) Read(account string) QuotaReading {
	dir := c.Dir
	if dir == "" {
		dir = filepath.Join(os.Getenv("APPDATA"), "herdr", "claude-cost")
	}
	r := QuotaReading{Provider: "anthropic", Account: account, State: QuotaUnknown, Source: dir}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(files) == 0 {
		r.Note = "no status-line files (statusline.js not writing, or never run)"
		return r
	}
	var bestT time.Time
	var best struct {
		ConfigDir  string `json:"config_dir"`
		RateLimits map[string]struct {
			Used     *float64 `json:"used_percentage"`
			ResetsAt int64    `json:"resets_at"`
		} `json:"rate_limits"`
	}
	for _, f := range files {
		st, err := os.Stat(f)
		if err != nil || !st.ModTime().After(bestT) {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var d struct {
			ConfigDir  string `json:"config_dir"`
			RateLimits map[string]struct {
				Used     *float64 `json:"used_percentage"`
				ResetsAt int64    `json:"resets_at"`
			} `json:"rate_limits"`
		}
		if json.Unmarshal(b, &d) != nil || d.RateLimits == nil || claudeAccountOf(d.ConfigDir) != account {
			continue
		}
		best, bestT = d, st.ModTime()
	}
	if bestT.IsZero() {
		r.Note = "no status-line reading for this account"
		return r
	}
	r.State, r.Observed = QuotaKnown, bestT
	names := map[string]string{"five_hour": "five_hour", "seven_day": "weekly"}
	for k, v := range best.RateLimits {
		l := Limit{Kind: k, UsedPct: v.Used}
		if n, ok := names[k]; ok {
			l.Kind = n
		}
		if v.ResetsAt > 0 {
			l.ResetsAt = time.Unix(v.ResetsAt, 0)
		}
		r.Limits = append(r.Limits, l)
	}
	sort.Slice(r.Limits, func(i, j int) bool { return r.Limits[i].Kind < r.Limits[j].Kind })
	return r
}

// claudeAccountOf: "" config_dir = the main sign-in ("main"), else the folder name.
func claudeAccountOf(configDir string) string {
	d := strings.TrimRight(strings.ReplaceAll(configDir, "\\", "/"), "/")
	if d == "" {
		return "main"
	}
	return d[strings.LastIndex(d, "/")+1:]
}

// ---------- Codex: rate limits in the session logs ----------

// CodexSessions reads the newest token_count event of <CODEX_HOME>/sessions/**/rollout-*.jsonl, where
// Codex records rate_limits.primary/secondary {used_percent, window_minutes, resets_in_seconds|resets_at}.
// (Field names from Codex CLI session logs; not verified on the owner's version - tests use a sample.)
type CodexSessions struct {
	Home func(account string) string // CODEX_HOME of an account
}

func (c CodexSessions) Read(account string) QuotaReading {
	home := c.Home(account)
	r := QuotaReading{Provider: "openai", Account: account, State: QuotaUnknown, Source: filepath.Join(home, "sessions")}
	var newest string
	var newestT time.Time
	filepath.WalkDir(filepath.Join(home, "sessions"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(d.Name(), ".jsonl") {
			if fi, err := d.Info(); err == nil && fi.ModTime().After(newestT) {
				newest, newestT = p, fi.ModTime()
			}
		}
		return nil
	})
	if newest == "" {
		r.Note = "no Codex session log"
		return r
	}
	f, err := os.Open(newest)
	if err != nil {
		r.Note = err.Error()
		return r
	}
	defer f.Close()
	type window struct {
		Used    *float64 `json:"used_percent"`
		Minutes int      `json:"window_minutes"`
		InSec   int64    `json:"resets_in_seconds"`
		At      int64    `json:"resets_at"`
	}
	var last map[string]*window
	var lastAt time.Time
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if !strings.Contains(string(line), "rate_limits") {
			continue
		}
		var ev struct {
			Timestamp time.Time `json:"timestamp"`
			Payload   struct {
				RateLimits map[string]*window `json:"rate_limits"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &ev) == nil && ev.Payload.RateLimits != nil {
			last, lastAt = ev.Payload.RateLimits, ev.Timestamp
		}
	}
	if last == nil {
		r.Note = "session log has no rate_limits event"
		return r
	}
	if lastAt.IsZero() {
		lastAt = newestT
	}
	r.State, r.Observed = QuotaKnown, lastAt
	for name, w := range last {
		if w == nil {
			continue
		}
		kind := name
		switch w.Minutes {
		case 300:
			kind = "five_hour"
		case 10080:
			kind = "weekly"
		}
		l := Limit{Kind: kind, UsedPct: w.Used}
		if w.At > 0 {
			l.ResetsAt = time.Unix(w.At, 0)
		} else if w.InSec > 0 {
			l.ResetsAt = lastAt.Add(time.Duration(w.InSec) * time.Second)
		}
		r.Limits = append(r.Limits, l)
	}
	sort.Slice(r.Limits, func(i, j int) bool { return r.Limits[i].Kind < r.Limits[j].Kind })
	return r
}

// Unsupported is for providers that expose nothing we can read (Antigravity, Cline providers today).
type Unsupported struct{ Provider, Why string }

func (u Unsupported) Read(account string) QuotaReading {
	return QuotaReading{Provider: u.Provider, Account: account, State: QuotaUnsupported, Note: u.Why}
}

// CommandSource runs a command that prints a QuotaReading as JSON ({account} is replaced). Bounded.
type CommandSource struct {
	Provider string
	Argv     []string
}

func (c CommandSource) Read(account string) QuotaReading {
	r := QuotaReading{Provider: c.Provider, Account: account, State: QuotaUnknown, Source: strings.Join(c.Argv, " ")}
	argv := make([]string, len(c.Argv))
	for i, a := range c.Argv {
		argv[i] = strings.ReplaceAll(a, "{account}", account)
	}
	out, err, _ := runBounded(context.Background(), "", 30*time.Second, 64<<10, argv[0], argv[1:]...)
	if err != nil {
		r.Note = "quota command failed: " + oneLine(out, 160)
		return r
	}
	var got QuotaReading
	if json.Unmarshal([]byte(strings.TrimSpace(out)), &got) != nil {
		r.Note = "quota command printed no reading"
		return r
	}
	got.Provider, got.Account = c.Provider, account
	if got.State == "" {
		got.State = QuotaKnown
	}
	return got
}

// Quotas caches readings per real account (provider/account) for a short time.
type Quotas struct {
	mu      sync.Mutex
	Sources map[string]QuotaSource // provider -> source
	TTL     time.Duration
	cache   map[string]QuotaReading
	at      map[string]time.Time
}

func (q *Quotas) Reading(provider, account string, now time.Time) QuotaReading {
	q.mu.Lock()
	defer q.mu.Unlock()
	key := provider + "/" + account
	if r, ok := q.cache[key]; ok && now.Sub(q.at[key]) < q.TTL {
		return r
	}
	src := q.Sources[provider]
	var r QuotaReading
	if src == nil {
		r = QuotaReading{Provider: provider, Account: account, State: QuotaUnknown, Note: "no quota source configured for " + provider}
	} else {
		r = src.Read(account)
	}
	if q.cache == nil {
		q.cache, q.at = map[string]QuotaReading{}, map[string]time.Time{}
	}
	q.cache[key], q.at[key] = r, now
	return r
}

// DefaultQuotas wires the sources this project knows.
func DefaultQuotas() *Quotas {
	home, _ := os.UserHomeDir()
	return &Quotas{TTL: 2 * time.Minute, Sources: map[string]QuotaSource{
		"anthropic": ClaudeStatusFiles{},
		"openai": CodexSessions{Home: func(acc string) string {
			if acc == "" || acc == "default" || acc == "main" {
				if h := os.Getenv("CODEX_HOME"); h != "" {
					return h
				}
				return filepath.Join(home, ".codex")
			}
			return filepath.Join(home, ".herdr-accounts", "codex", acc)
		}},
		"google": Unsupported{"google", "Antigravity shows no usage limits to read"},
		"cline":  Unsupported{"cline", "Cline providers expose no common limit reading"},
	}}
}
