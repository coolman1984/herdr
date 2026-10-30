package goal

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var pickRe = regexp.MustCompile(`(?m)^\s*PICK\s+(\d+)\s*$`)

// CommandDecider runs a one-shot model (e.g. `claude -p` on another account) with the goal, the agent's
// task and its screen, and sends the option it picks. Its answer must be one line "PICK <n>" naming an
// option that is on the screen; anything else is refused and the CLI default is taken instead.
func CommandDecider(argv []string) Decider {
	return func(g *Goal, a *AgentRun, screen string, opts []Option) ([]string, string, error) {
		if len(opts) == 0 {
			return nil, "", fmt.Errorf("menu has no numbered options")
		}
		var b strings.Builder
		fmt.Fprintf(&b, "An unattended coding agent (%s, task %s) shows a menu. Goal: %s\n\nScreen:\n%s\n\n", a.Profile, a.Task, g.Text, tailText(screen, 30))
		b.WriteString("Pick the option that keeps the agent working on its task safely: never one that signs in, enters a secret, pushes, publishes or deletes outside the project. Answer with exactly one line: PICK <number>\n")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Stdin = strings.NewReader(b.String())
		hideWindow(cmd)
		out, err := cmd.Output()
		if err != nil {
			return nil, "", fmt.Errorf("decider: %v", err)
		}
		m := pickRe.FindStringSubmatch(string(out))
		if m == nil {
			return nil, "", fmt.Errorf("decider gave no PICK line: %s", oneLine(string(out), 120))
		}
		for _, o := range opts {
			if o.Num == m[1] {
				return []string{o.Num}, "picked " + o.Num + " " + o.Text, nil
			}
		}
		return nil, "", fmt.Errorf("decider picked %s, not on the screen", m[1])
	}
}
