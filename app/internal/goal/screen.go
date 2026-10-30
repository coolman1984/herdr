package goal

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed prompts.json
var defaultPrompts []byte

type Prompt struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Window  int      `json:"window"`
	Footer  bool     `json:"footer"`
	Pattern string   `json:"pattern"`
	Keys    []string `json:"keys,omitempty"`
	re      *regexp.Regexp
}

type Prompts []Prompt

// LoadPrompts reads prompts.json from path, or the built-in copy when path is empty or missing.
func LoadPrompts(path string) (Prompts, error) {
	data := defaultPrompts
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			data = b
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	var f struct {
		Prompts []Prompt `json:"prompts"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("prompts.json: %w", err)
	}
	for i := range f.Prompts {
		re, err := regexp.Compile(f.Prompts[i].Pattern)
		if err != nil {
			return nil, fmt.Errorf("prompts.json %s: %w", f.Prompts[i].Name, err)
		}
		f.Prompts[i].re = re
	}
	return f.Prompts, nil
}

func MustPrompts() Prompts {
	p, err := LoadPrompts("")
	if err != nil {
		panic(err)
	}
	return p
}

// Sight is what a screen shows.
type Sight struct {
	Kind    string // "" = nothing known; else a prompt kind
	Name    string
	Line    string
	Keys    []string
	Options []Option
	Reset   time.Time // rate-limited: when the account frees up, zero if not printed
}

type Option struct {
	Num         string
	Text        string
	Highlighted bool
}

var footerRe = regexp.MustCompile(`(?i)enter to (confirm|select|continue)|press enter|↑/↓|↑↓|arrow keys|esc to (cancel|exit|go back)|navigate`)

func nonEmptyLines(screen string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(screen, "\r\n", "\n"), "\n") {
		l = strings.TrimRight(l, " \t")
		if strings.TrimSpace(trimBox(l)) != "" {
			out = append(out, l)
		}
	}
	return out
}

// trimBox removes TUI frame characters so "│ 1. Yes │" reads as "1. Yes".
func trimBox(l string) string {
	return strings.Trim(strings.TrimSpace(l), "│┃|╭╮╰╯─━ ")
}

func last(lines []string, n int) []string {
	if n <= 0 || n >= len(lines) {
		return lines
	}
	return lines[len(lines)-n:]
}

// Classify reads a screen. Only the bottom of the screen counts: an agent's answer that quotes a prompt
// higher up is not a prompt. Secret kinds come first and win over everything.
func (ps Prompts) Classify(screen string, now time.Time) Sight {
	lines := nonEmptyLines(screen)
	footer := false
	for _, l := range last(lines, 8) {
		if footerRe.MatchString(l) {
			footer = true
		}
	}
	for _, p := range ps {
		if p.Footer && !footer {
			continue
		}
		for _, l := range last(lines, p.Window) {
			if p.re.MatchString(trimBox(l)) {
				s := Sight{Kind: p.Kind, Name: p.Name, Line: trimBox(l), Keys: p.Keys}
				if p.Kind == "trust" || p.Kind == "choice" {
					s.Options = parseOptions(last(lines, 16))
				}
				if p.Kind == "rate-limited" {
					s.Reset = parseReset(strings.Join(last(lines, p.Window), "\n"), now)
				}
				return s
			}
		}
	}
	return Sight{}
}

var optionRe = regexp.MustCompile(`^\s*(❯|›|>|\*)?\s*(\d+)[.)]\s+(.+?)\s*$`)
var arrowOptionRe = regexp.MustCompile(`^\s*(❯|›|>)\s+(.+?)\s*$`)

func parseOptions(lines []string) []Option {
	var opts []Option
	for _, raw := range lines {
		l := trimBox(raw)
		if m := optionRe.FindStringSubmatch(l); m != nil {
			opts = append(opts, Option{Num: m[2], Text: m[3], Highlighted: m[1] != ""})
		}
	}
	if len(opts) > 0 {
		return opts
	}
	for _, raw := range lines {
		if m := arrowOptionRe.FindStringSubmatch(trimBox(raw)); m != nil {
			opts = append(opts, Option{Text: m[2], Highlighted: true})
		}
	}
	return opts
}

var yesRe = regexp.MustCompile(`(?i)^(yes|trust|proceed|continue|allow)|i trust`)
var noRe = regexp.MustCompile(`(?i)\b(no|exit|quit|cancel|don't|do not|deny)\b`)

// TrustKeys picks the "yes, trust" option: its number when the menu is numbered, Enter when the
// highlighted unnumbered option is the yes option. nil = no safe answer (escalate).
func TrustKeys(opts []Option) []string {
	for _, o := range opts {
		if yesRe.MatchString(o.Text) && !noRe.MatchString(strings.ReplaceAll(o.Text, "I trust", "")) {
			if o.Num != "" {
				return []string{o.Num}
			}
			if o.Highlighted {
				return []string{"Enter"}
			}
		}
	}
	return nil
}

var clockRe = regexp.MustCompile(`(?i)(?:resets?|try again|available)\s*(?:at|in)?\s*(\d{1,2})(?::(\d{2}))?\s*(am|pm)?(?:\s*\(([A-Za-z_]+/[A-Za-z_/+-]+|UTC)\))?`)
var inRe = regexp.MustCompile(`(?i)(?:resets?|try again)\s+in\s+(?:(\d+)\s*h(?:ours?)?)?\s*(?:(\d+)\s*m(?:in(?:utes?)?)?)?`)

// parseReset reads "resets 3pm", "try again at 15:05" or "resets in 2h 10m". Local time; a clock time
// already past means tomorrow. Zero when nothing is printed.
func parseReset(text string, now time.Time) time.Time {
	if m := inRe.FindStringSubmatch(text); m != nil && (m[1] != "" || m[2] != "") {
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		return now.Add(time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute)
	}
	m := clockRe.FindStringSubmatch(text)
	if m == nil {
		return time.Time{}
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	switch strings.ToLower(m[3]) {
	case "pm":
		if h < 12 {
			h += 12
		}
	case "am":
		if h == 12 {
			h = 0
		}
	}
	if h > 23 || mi > 59 {
		return time.Time{}
	}
	loc := now.Location()
	if m[4] != "" { // "resets 3pm (Africa/Cairo)": the zone the CLI printed, not the PC's
		if l, err := time.LoadLocation(m[4]); err == nil {
			loc = l
		}
	}
	n := now.In(loc)
	t := time.Date(n.Year(), n.Month(), n.Day(), h, mi, 0, 0, loc)
	if !t.After(now) {
		t = t.Add(24 * time.Hour)
	}
	return t
}

var volatileLine = regexp.MustCompile(`(?i)esc to interrupt|\(\d+s|\d+\s*tokens|thinking|working|elapsed|ctx|context left|⏱`)
var spinnerRe = regexp.MustCompile(`[⠁-⣿✻✶✳✢·•●◐◓◑◒|/\\-]`)
var digitsRe = regexp.MustCompile(`\d+`)

// ProgressHash hashes the screen without what changes while nothing happens: spinner and timer lines,
// digits, and the bottom status block (last 3 lines). Two screens of a stuck agent hash the same.
func ProgressHash(screen string) string {
	lines := nonEmptyLines(screen)
	if len(lines) > 3 {
		lines = lines[:len(lines)-3]
	} else {
		lines = nil
	}
	var b strings.Builder
	for _, l := range lines {
		if volatileLine.MatchString(l) {
			continue
		}
		l = digitsRe.ReplaceAllString(l, "#")
		l = spinnerRe.ReplaceAllString(l, "")
		b.WriteString(strings.TrimSpace(l))
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

func tailText(screen string, n int) string {
	return strings.Join(last(nonEmptyLines(screen), n), "\n")
}
