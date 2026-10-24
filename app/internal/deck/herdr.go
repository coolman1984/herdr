package deck

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

type Herdr interface {
	Call(context.Context, ...string) ([]byte, error)
	Helper(context.Context, map[string]string) (string, error)
}
type ScriptRunner interface {
	RunScript(context.Context, string, map[string]any, time.Duration) (string, error)
}
type CLI struct {
	Bin, Session, Root string
	Socket             *Socket
}

var ansiPattern = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07))`)

type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len()+n > b.limit {
		return 0, fmt.Errorf("child output exceeded %d bytes", b.limit)
	}
	return b.Buffer.Write(p)
}
func hiddenCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	hideProcess(cmd)
	cmd.WaitDelay = time.Second
	return cmd
}
func runCommand(ctx context.Context, name string, stdin string, args ...string) ([]byte, error) {
	cmd := hiddenCommand(ctx, name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out := &cappedBuffer{limit: 4 << 20}
	fail := &cappedBuffer{limit: 1 << 20}
	cmd.Stdout = out
	cmd.Stderr = fail
	err := cmd.Start()
	if err == nil {
		job, e := containProcess(cmd)
		if e != nil {
			cmd.Process.Kill()
			cmd.Wait()
			return nil, e
		}
		defer closeMutex(job)
		err = cmd.Wait()
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%s timed out or cancelled: %w", filepath.Base(name), ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s %s", filepath.Base(name), err, strings.TrimSpace(fail.String()), strings.TrimSpace(out.String()))
	}
	return out.Bytes(), nil
}
func (c *CLI) Call(parent context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 7*time.Second)
	defer cancel()
	if c.Socket != nil {
		if out, handled, err := c.Socket.call(ctx, args); handled {
			return out, err
		}
	}
	if c.Session != "" {
		args = append([]string{"--session", c.Session}, args...)
	}
	out, err := runCommand(ctx, c.Bin, "", args...)
	if err != nil {
		return nil, err
	}
	var v struct {
		Error *struct{ Code, Message string } `json:"error"`
	}
	if json.Unmarshal(out, &v) == nil && v.Error != nil {
		return nil, fmt.Errorf("%s: %s", v.Error.Code, v.Error.Message)
	}
	return out, nil
}
func encodedPS(text string) string {
	units := utf16.Encode([]rune(text))
	data := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(data[2*i:], u)
	}
	return base64.StdEncoding.EncodeToString(data)
}
func (c *CLI) Helper(parent context.Context, params map[string]string) (string, error) {
	values := map[string]any{}
	for k, v := range params {
		values[k] = v
	}
	return c.RunScript(parent, "herdr-agents.ps1", values, 50*time.Second)
}
func (c *CLI) RunScript(parent context.Context, scriptName string, params map[string]any, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	// The script is fixed. User text travels only as JSON on stdin, with splatted values.
	script := `$ErrorActionPreference='Stop'; $p=[Console]::In.ReadToEnd() | ConvertFrom-Json -AsHashtable; $script=$p['Script']; $p.Remove('Script'); $confirmed=$p['DeckConfirmedStop']; $p.Remove('DeckConfirmedStop'); $handoff=$p['DeckHandoffText']; $p.Remove('DeckHandoffText'); if($confirmed){ if([IO.Path]::GetFileName($script) -ne 'herdr-stop.ps1'){throw 'Invalid stop script'}; $tokens=$null; $parseErrors=$null; $ast=[Management.Automation.Language.Parser]::ParseFile($script,[ref]$tokens,[ref]$parseErrors); if($parseErrors.Count){throw 'Stop script parse failed'}; $f=$ast.Find({param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Stop-HerdrServer'},$true); if(-not $f){throw 'Stop function missing'}; $herdr=Join-Path $env:LOCALAPPDATA 'Programs\Herdr\bin\herdr.exe'; . ([scriptblock]::Create($f.Extent.Text)); Stop-HerdrServer; exit 0 }; & $script @p; $code=$LASTEXITCODE; if($code -eq 0 -and $handoff){Add-Content -LiteralPath $p.TaskFile -Value ("` + "`n## Owner notes`n" + `"+$handoff); Write-Output ('Handoff: '+$p.TaskFile)}; exit $code`
	input := map[string]any{"Script": filepath.Join(c.Root, scriptName)}
	for k, v := range params {
		input[k] = v
	}
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	out, err := runCommand(ctx, "pwsh.exe", string(data), "-NoProfile", "-STA", "-NonInteractive", "-EncodedCommand", encodedPS(script))
	return ansiPattern.ReplaceAllString(string(out), ""), err
}

type Profile struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Command      string   `json:"-"`
	Model        string   `json:"model"`
	ModelFlag    string   `json:"-"`
	Args         []string `json:"-"`
	Effort       string   `json:"effort"`
	EffortArgs   []string `json:"-"`
	Provider     string   `json:"-"`
	ProviderFlag string   `json:"-"`
	Account      *struct {
		Tool string
		Env  map[string]string
		Args []string
	} `json:"-"`
	Accounts []string `json:"accounts"`
}

func readProfiles(root string) (map[string]Profile, error) {
	data, err := os.ReadFile(filepath.Join(root, "agents.json"))
	if err != nil {
		return nil, err
	}
	// Decode a private wire type: launch arguments never go to the browser.
	var v struct {
		Agents map[string]struct {
			Kind, Command, Model, ModelFlag, Effort, Provider, ProviderFlag string
			Args, EffortArgs                                                []string
			Account                                                         *struct {
				Tool string
				Env  map[string]string
				Args []string
			}
		}
	}
	if err = json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	profiles := map[string]Profile{}
	for name, p := range v.Agents {
		profiles[name] = Profile{Name: name, Kind: p.Kind, Command: p.Command, Model: p.Model, ModelFlag: p.ModelFlag, Args: p.Args, Effort: p.Effort, EffortArgs: p.EffortArgs, Provider: p.Provider, ProviderFlag: p.ProviderFlag, Account: p.Account}
	}
	return profiles, nil
}
func fallbackCommand(p Profile, model, effort, account string) (string, error) {
	argv := append([]string{}, p.Args...)
	if model == "" {
		model = p.Model
	}
	if effort == "" {
		effort = p.Effort
	}
	if model != "" && p.ModelFlag != "" {
		argv = append(argv, p.ModelFlag, model)
	}
	if p.Provider != "" && p.ProviderFlag != "" {
		argv = append(argv, p.ProviderFlag, p.Provider)
	}
	for _, a := range p.EffortArgs {
		if effort != "" {
			argv = append(argv, strings.ReplaceAll(a, "{0}", effort))
		}
	}
	env := map[string]string{}
	if account != "" {
		if p.Account == nil || !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`).MatchString(account) {
			return "", fmt.Errorf("unsupported or invalid account")
		}
		dir := filepath.Join(os.Getenv("USERPROFILE"), ".herdr-accounts", p.Account.Tool, account)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			return "", fmt.Errorf("account folder missing: sign in using herdr-agents first")
		}
		for k, v := range p.Account.Env {
			env[k] = strings.ReplaceAll(v, "{dir}", dir)
		}
		for _, a := range p.Account.Args {
			argv = append(argv, strings.ReplaceAll(a, "{dir}", dir))
		}
	}
	payload, _ := json.Marshal(map[string]any{"command": p.Command, "argv": argv, "env": env})
	// Only base64 appears in the shell command; no user-supplied quoting or metacharacters.
	script := `$ErrorActionPreference='Stop'; $p=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('` + base64.StdEncoding.EncodeToString(payload) + `'))|ConvertFrom-Json -AsHashtable; foreach($k in $p.env.Keys){[Environment]::SetEnvironmentVariable($k,$p.env[$k],'Process')}; & $p.command @($p.argv); if($LASTEXITCODE -ne 0){throw "Agent exited with code $LASTEXITCODE"}`
	return "pwsh.exe -NoProfile -EncodedCommand " + encodedPS(script), nil
}
