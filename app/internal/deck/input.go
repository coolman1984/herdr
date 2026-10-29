package deck

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	promptGlyph = "\u276f"
	codexGlyph  = "\u203a"
	ruleGlyph   = '\u2500'
	noBreak     = "\u00a0"
)

// inputText returns the text the owner has typed into an agent's input box, or "" when the box is
// empty or shows only dim suggestion text. tool is the pane's agent kind ("claude", "codex", ...).
func inputText(screen, tool string) string {
	lines := strings.Split(screen, "\n")
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	if tool == "codex" {
		if region, ok := codexRegion(lines); ok {
			return typedText(region)
		}
		return ""
	}
	if region, ok := claudeRegion(lines); ok {
		return typedText(region)
	}
	return ""
}

// inputNotEmpty refuses a write when the owner has already typed into the agent's input box.
func inputNotEmpty(screen string, pane Pane) error {
	if pane.Agent == "" && pane.Name == "" {
		return nil
	}
	typed := inputText(screen, pane.Agent)
	if typed == "" {
		return nil
	}
	name := pane.Name
	if name == "" {
		name = pane.ID
	}
	return &statusError{Status: 409, Msg: "input_not_empty", Fields: map[string]any{"error": "input_not_empty", "agent": name, "pane_id": pane.ID, "typed": firstRunes(typed, 60), "message": name + " has text in its input box - not sent"}}
}

// checkInput reads a pane screen and refuses when the owner has already typed in its input box.
func (a *App) checkInput(ctx context.Context, pane Pane) error {
	if pane.Agent == "" && pane.Name == "" {
		return nil
	}
	screen, err := a.herdr.Call(ctx, "pane", "read", pane.ID, "--source", "visible", "--lines", "60", "--format", "ansi")
	if err != nil {
		return err
	}
	return inputNotEmpty(string(screen), pane)
}

func firstRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}

// claudeRegion is the input box of a Claude pane: the prompt line after its glyph, plus every line
// up to the rule below it. A history prompt has no rule above it and is not a box.
func claudeRegion(lines []string) (string, bool) {
	for i := len(lines) - 1; i >= 0; i-- {
		at := strings.Index(lines[i], promptGlyph)
		if at < 0 || !ruleAbove(lines, i) {
			continue
		}
		lower := ruleBelow(lines, i, 12)
		if lower < 0 {
			continue
		}
		region := lines[i][at+len(promptGlyph):]
		switch {
		case strings.HasPrefix(region, " "):
			region = region[1:]
		case strings.HasPrefix(region, noBreak):
			region = region[len(noBreak):]
		}
		if lower > i+1 {
			region += "\n" + strings.Join(lines[i+1:lower], "\n")
		}
		return region, true
	}
	return "", false
}

// codexRegion is the text after the last "> " prompt line in the last ten lines.
func codexRegion(lines []string) (string, bool) {
	tail := lines
	if len(tail) > 10 {
		tail = tail[len(tail)-10:]
	}
	for i := len(tail) - 1; i >= 0; i-- {
		if !strings.HasPrefix(strings.TrimLeft(stripANSI(tail[i]), " "), codexGlyph) {
			continue
		}
		at := strings.Index(tail[i], codexGlyph)
		if at < 0 {
			continue
		}
		return strings.TrimPrefix(tail[i][at+len(codexGlyph):], " "), true
	}
	return "", false
}

func ruleAbove(lines []string, at int) bool {
	for i := at - 1; i >= 0; i-- {
		if strings.TrimSpace(stripANSI(lines[i])) == "" {
			continue
		}
		return ruleLine(lines[i])
	}
	return false
}

func ruleBelow(lines []string, at, within int) int {
	for i := at + 1; i < len(lines) && i <= at+within; i++ {
		if ruleLine(lines[i]) {
			return i
		}
	}
	return -1
}

func ruleLine(line string) bool {
	clean := strings.TrimSpace(stripANSI(line))
	if len([]rune(clean)) < 20 {
		return false
	}
	for _, r := range clean {
		if r != ruleGlyph {
			return false
		}
	}
	return true
}

// typedText keeps every visible rune that is not dim; the rest is placeholder or suggestion text.
func typedText(region string) string {
	state := sgrState{}
	out := []rune{}
	for i := 0; i < len(region); {
		if region[i] == 0x1b {
			next, params, ok := sgrAt(region, i)
			if ok {
				state.apply(params)
			}
			i = next
			continue
		}
		r, size := utf8.DecodeRuneInString(region[i:])
		i += size
		if !state.dim() {
			out = append(out, r)
		}
	}
	return firstRunes(strings.TrimSpace(string(out)), 200)
}

func stripANSI(text string) string { return ansiPattern.ReplaceAllString(text, "") }

type sgrState struct {
	faint bool
	grey  bool
}

func (s *sgrState) dim() bool { return s.faint || s.grey }

func (s *sgrState) apply(params []string) {
	for i := 0; i < len(params); i++ {
		switch params[i] {
		case "", "0":
			s.faint, s.grey = false, false
		case "2":
			s.faint = true
		case "22":
			s.faint = false
		case "39":
			s.grey = false
		case "90":
			s.grey = true
		case "30", "31", "32", "33", "34", "35", "36", "37", "91", "92", "93", "94", "95", "96", "97":
			s.grey = false
		case "38":
			if i+2 < len(params) && params[i+1] == "5" {
				n, err := strconv.Atoi(params[i+2])
				s.grey = err == nil && (n == 8 || (n >= 232 && n <= 250))
				i += 2
			} else if i+4 < len(params) && params[i+1] == "2" {
				red, e1 := strconv.Atoi(params[i+2])
				green, e2 := strconv.Atoi(params[i+3])
				blue, e3 := strconv.Atoi(params[i+4])
				if e1 == nil && e2 == nil && e3 == nil {
					s.grey = max(red, green, blue)-min(red, green, blue) <= 12 && max(red, green, blue) <= 170
				}
				i += 4
			}
		}
	}
}

// sgrAt reads one escape sequence; it reports whether it carried SGR parameters.
func sgrAt(text string, at int) (int, []string, bool) {
	i := at + 1
	if i >= len(text) || text[i] != '[' {
		return i, nil, false
	}
	i++
	start := i
	for i < len(text) && text[i] >= 0x20 && text[i] <= 0x3f {
		i++
	}
	params := []string{}
	if i > start {
		params = strings.Split(text[start:i], ";")
	}
	for i < len(text) && text[i] >= 0x20 && text[i] <= 0x2f {
		i++
	}
	if i >= len(text) {
		return i, nil, false
	}
	final := text[i]
	i++
	if final != 'm' {
		return i, nil, false
	}
	if len(params) == 0 {
		params = []string{""}
	}
	return i, params, true
}
