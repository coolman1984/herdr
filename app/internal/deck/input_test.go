package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func inputFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "input", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestInputTextMatchesEveryHerdrFixture(t *testing.T) {
	for _, v := range []struct {
		file, tool, want string
	}{
		{"claude-empty.real.ansi", "claude", ""},
		{"claude-typed.synthetic.ansi", "claude", "please also fix the tests"},
		{"claude-typed-white.synthetic.ansi", "claude", "stop and wait"},
		{"claude-suggestion.synthetic.ansi", "claude", ""},
		{"claude-history-prompt.synthetic.ansi", "claude", ""},
		{"codex-empty.synthetic.ansi", "codex", ""},
		{"codex-typed.synthetic.ansi", "codex", "fix the failing test first"},
		{"shell-prompt.synthetic.ansi", "", ""},
	} {
		if got := inputText(inputFixture(t, v.file), v.tool); got != v.want {
			t.Errorf("%s (%s): got %q, want %q", v.file, v.tool, got, v.want)
		}
	}
	multiline := inputText(inputFixture(t, "claude-multiline.synthetic.ansi"), "claude")
	if !strings.Contains(multiline, "second line typed by the owner") {
		t.Errorf("claude-multiline.synthetic.ansi: got %q", multiline)
	}
}

func TestTypedTextIgnoresDimTextAndKeepsSpaces(t *testing.T) {
	for _, dim := range []string{
		"\x1b[38;2;153;153;153mrefactor app.js\x1b[0m",
		"\x1b[2mask codex\x1b[0m",
		"\x1b[38;5;244mgrey 232-250\x1b[0m",
		"\x1b[90mbright black\x1b[0m",
		"\x1b[0m   \x1b[0m",
	} {
		if got := typedText(dim); got != "" {
			t.Fatalf("%q counted as typed: %q", dim, got)
		}
	}
	if got := typedText("please also fix the tests"); got != "please also fix the tests" {
		t.Fatalf("plain text changed: %q", got)
	}
	if got := typedText("\x1b[2mdim\x1b[22mthen typed"); got != "then typed" {
		t.Fatalf("SGR 22 did not reset dim: %q", got)
	}
	if got := typedText("\x1b[38;2;255;255;255mwhite text\x1b[0m"); got != "white text" {
		t.Fatalf("white text dropped: %q", got)
	}
	if got := typedText("\x1b[90mhint \x1b[37mtyped\x1b[0m"); got != "typed" {
		t.Fatalf("a basic foreground colour did not end the grey run: %q", got)
	}
}

func TestPromptAndPasteRefuseTextInTheInputBox(t *testing.T) {
	f := &stubHerdr{snapshot: agentSnapshot("claude", "coder", "")}
	f.screen = inputFixture(t, "claude-typed.synthetic.ansi")
	a := stubApp(f)
	for _, v := range []struct{ path, body string }{
		{"/api/pane/prompt", `{"key":"lab|w1:p1","text":"also do this"}`},
		{"/api/pane/text", `{"key":"lab|w1:p1","text":"also do this"}`},
		{"/api/action", `{"action":"prompt","pane_id":"w1:p1","text":"also do this"}`},
	} {
		w := request(a, "POST", v.path, v.body, "test-token", "")
		if w.Code != 409 {
			t.Fatalf("%s wrote into a typed input box: %d %s", v.path, w.Code, w.Body.String())
		}
		body := decodeBody(t, w)
		if body["error"] != "input_not_empty" || body["agent"] != "coder" || body["pane_id"] != "w1:p1" {
			t.Fatalf("%s: bad refusal: %v", v.path, body)
		}
		if typed, _ := body["typed"].(string); !strings.Contains(typed, "please also fix the tests") {
			t.Fatalf("%s: typed text missing: %v", v.path, body)
		}
		if message, _ := body["message"].(string); !strings.Contains(message, "has text in its input box") {
			t.Fatalf("%s: message missing: %v", v.path, body)
		}
		if len(f.writes) != 0 {
			t.Fatalf("%s wrote before the input check: %v", v.path, f.writes)
		}
	}
}

func TestEmptyInputBoxStillAcceptsAPromptAndPaste(t *testing.T) {
	f := &stubHerdr{snapshot: agentSnapshot("claude", "coder", "")}
	f.screen = inputFixture(t, "claude-empty.real.ansi")
	a := stubApp(f)
	w := request(a, "POST", "/api/pane/prompt", `{"key":"lab|w1:p1","text":"go"}`, "test-token", "")
	if w.Code != 200 {
		t.Fatalf("empty input box refused: %d %s", w.Code, w.Body.String())
	}
	if len(f.writes) != 1 || f.writes[0] != "agent|prompt|coder|go" {
		t.Fatalf("prompt not delivered once: %v", f.writes)
	}
	if decodeBody(t, w)["dispatch_id"] == nil {
		t.Fatalf("no dispatch id: %s", w.Body.String())
	}
	w = request(a, "POST", "/api/pane/text", `{"key":"lab|w1:p1","text":"x"}`, "test-token", "")
	if w.Code != 200 || len(f.writes) != 2 || f.writes[1] != "pane|send-text|w1:p1|x" {
		t.Fatalf("paste refused or lost: %d %v", w.Code, f.writes)
	}
}

func TestAskRefusesTypedInputBeforeAnyWrite(t *testing.T) {
	f := &stubHerdr{snapshot: agentSnapshot("claude", "coder", "")}
	f.screen = inputFixture(t, "claude-typed.synthetic.ansi")
	a := stubApp(f)
	w := request(a, "POST", "/api/agents/ask", `{"name":"coder","text":"also do this"}`, "test-token", "")
	if w.Code != 409 || decodeBody(t, w)["agent"] != "coder" {
		t.Fatalf("ask ignored a typed input box: %d %s", w.Code, w.Body.String())
	}
	if len(f.writes) != 0 {
		t.Fatalf("ask wrote before the input check: %v", f.writes)
	}
	f.screen = inputFixture(t, "claude-empty.real.ansi")
	if w = request(a, "POST", "/api/agents/ask", `{"name":"coder","text":"also do this"}`, "test-token", ""); w.Code != 200 {
		t.Fatalf("ask with an empty input box failed: %d %s", w.Code, w.Body.String())
	}
}

func TestFanoutChecksEveryNamedAgentFirst(t *testing.T) {
	f := &stubHerdr{
		screens:  map[string]string{"w1:p1": inputFixture(t, "codex-empty.synthetic.ansi"), "w1:p2": inputFixture(t, "codex-typed.synthetic.ansi")},
		snapshot: twoAgentSnapshot("codex", "one", "two"),
	}
	a := stubApp(f)
	w := request(a, "POST", "/api/agents/fanout", `{"names":["one","two"],"text":"also do this"}`, "test-token", "")
	if w.Code != 409 {
		t.Fatalf("fanout ignored the second agent's typed input box: %d %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["agent"] != "two" || body["pane_id"] != "w1:p2" {
		t.Fatalf("wrong refusal: %v", body)
	}
	if len(f.writes) != 0 {
		t.Fatalf("fanout wrote before the input check: %v", f.writes)
	}
	f.screens["w1:p2"] = inputFixture(t, "codex-empty.synthetic.ansi")
	if w = request(a, "POST", "/api/agents/fanout", `{"names":["one","two"],"text":"also do this"}`, "test-token", ""); w.Code != 200 {
		t.Fatalf("fanout with two empty input boxes failed: %d %s", w.Code, w.Body.String())
	}
}
