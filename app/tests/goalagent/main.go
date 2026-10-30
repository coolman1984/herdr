// A scripted stand-in for an agent CLI, for goal-engine tests only. It never calls a model or the
// network. It reads the brief the engine gives it, acknowledges, runs the ```sh block of the task spec in
// its worktree, commits, and writes the result file. Fault tokens in the spec make it misbehave once
// per task (marker files in $HERDR_GOAL/faults): FAULT:trust, FAULT:crash, FAULT:idle, FAULT:limit,
// FAULT:capacity, FAULT:secret, FAULT:menu; REVIEW-FAIL-ONCE makes the reviewer reject the first time.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var in = bufio.NewReader(os.Stdin)

// readKey reads one key press of a menu: a digit or Enter.
func readKey() byte {
	for {
		c, err := in.ReadByte()
		if err != nil {
			hang()
		}
		if c == '\n' || c >= '0' && c <= '9' {
			return c
		}
	}
}

// readLine reads up to a newline.
func readLine() (string, bool) {
	var b strings.Builder
	for {
		c, err := in.ReadByte()
		if err != nil {
			return b.String(), false
		}
		if c == '\n' {
			return b.String(), true
		}
		b.WriteByte(c)
	}
}

// hang blocks like a stuck CLI until the process is killed.
func hang() {
	for {
		time.Sleep(time.Hour)
	}
}

func say(f string, a ...any) { fmt.Printf(f+"\n", a...) }

func once(task, fault string) bool {
	dir := filepath.Join(os.Getenv("HERDR_GOAL"), "faults")
	os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, task+"-"+fault+"-"+os.Getenv("HERDR_GOAL_ROLE"))
	if _, err := os.Stat(p); err == nil {
		return false
	}
	os.WriteFile(p, []byte(time.Now().String()), 0o644)
	return true
}

func waitFor(pred func(string) bool) {
	for {
		l, ok := readLine()
		if !ok {
			hang() // stdin closed: hang like a stuck CLI until killed
		}
		if pred(strings.TrimSpace(l)) {
			return
		}
	}
}

func run(dir, script string) (string, error) {
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var (
	briefRe  = regexp.MustCompile(`brief (\S+\.md) \(message (\S+)\)`)
	ackRe    = regexp.MustCompile(`create the file (\S+)`)
	answerRe = regexp.MustCompile(`write (\S+\.json) as JSON`)
	folderRe = regexp.MustCompile(`(?:Work only in this folder: |In )(\S+) (?:\(git|run)`)
	branchRe = regexp.MustCompile("git diff (\\S+)\\.\\.\\.HEAD")
	shRe     = regexp.MustCompile("(?s)```sh\n(.*?)```")
	fixRe    = regexp.MustCompile("(?s)```resolve\n(.*?)```")
	mergeRe  = regexp.MustCompile(`Merge (goal/\S+) into your branch`)
	taskRe   = regexp.MustCompile(`# (?:Review )?[Tt]ask (\S+):`)
)

func main() {
	persona := os.Getenv("STUB_PERSONA")
	say("%s agent (scripted test stand-in) ready, account %q", persona, os.Getenv("STUB_ACCOUNT"))
	say("> ")
	first := ""
	if len(os.Args) > 1 {
		first = strings.Join(os.Args[1:], " ")
	}
	for !briefRe.MatchString(first) {
		l, ok := readLine()
		if !ok {
			hang()
		}
		first = l
	}
	m := briefRe.FindStringSubmatch(first)
	b, err := os.ReadFile(m[1])
	if err != nil {
		say("cannot read brief: %v", err)
		os.Exit(2)
	}
	brief := string(b)
	role := os.Getenv("HERDR_GOAL_ROLE")
	task := ""
	if t := taskRe.FindStringSubmatch(brief); t != nil {
		task = t[1]
	}
	if role == "lead" {
		os.WriteFile(ackRe.FindStringSubmatch(brief)[1], []byte("ack"), 0o644)
		say("lead: plan given at creation; watching.")
		retried := map[string]bool{}
		for {
			l, ok := readLine()
			if !ok {
				hang()
			}
			say("lead got: %s", l)
			if !strings.Contains(l, "Next step needed") {
				continue
			}
			// Stand-in for the Architect's re-plan: retry each failed task once with a note.
			var g struct {
				Tasks []struct{ ID, State string }
			}
			b, _ := os.ReadFile(filepath.Join(os.Getenv("HERDR_GOAL"), "goal.json"))
			json.Unmarshal(b, &g)
			for _, t := range g.Tasks {
				if t.State == "failed" && !retried[t.ID] {
					retried[t.ID] = true
					out, err := exec.Command("herdr-goal", "task", "retry", os.Getenv("HERDR_GOAL"), "--id", t.ID, "--feedback", "lead: one more try after a failure").CombinedOutput()
					say("lead retried %s: %s %v", t.ID, out, err)
				}
			}
		}
	}
	dir := folderRe.FindStringSubmatch(brief)[1]
	if strings.Contains(brief, "FAULT:trust") && once(task, "trust") {
		say("────────────────────────────────────────")
		say(" Do you trust the files in this folder?")
		say(" %s", dir)
		say("   1. Yes, proceed")
		say(" ❯ 2. No, exit")
		say(" Enter to confirm · Esc to exit")
		for readKey() != '1' {
		}
		say("trusted.")
	}
	if strings.Contains(brief, "FAULT:secret") && once(task, "secret") {
		say("Please enter your API key:")
		hang()
	}
	if strings.Contains(brief, "FAULT:menu") && once(task, "menu") {
		say(" How should I format the output?")
		say(" ❯ 1. Plain text")
		say("   2. JSON")
		say(" Enter to confirm · Esc to cancel")
		say("chose %c", readKey())
	}
	os.WriteFile(ackRe.FindStringSubmatch(brief)[1], []byte("ack"), 0o644)
	say("● acknowledged %s, task %s, role %s", m[2], task, role)
	for i := 1; i <= 3; i++ {
		say("● working step %d", i)
		time.Sleep(700 * time.Millisecond)
	}
	if role == "write" {
		if strings.Contains(brief, "FAULT:crash") && once(task, "crash") {
			os.WriteFile(filepath.Join(dir, "half-done.txt"), []byte("work in progress\n"), 0o644)
			say("panic: simulated crash")
			os.Exit(3)
		}
		if strings.Contains(brief, "FAULT:limit") && once(task, "limit") {
			say("You've hit your limit · resets in 0h 1m")
			hang()
		}
		if strings.Contains(brief, "FAULT:capacity") && once(task, "capacity") {
			say("■ Selected model is at capacity. Please try a different model.")
			waitFor(func(l string) bool { return strings.HasPrefix(l, "continue") })
		}
		if strings.Contains(brief, "FAULT:idle") && once(task, "idle") {
			say("I have looked at the task.")
			waitFor(func(l string) bool { return strings.HasPrefix(l, "continue") })
		}
	}
	ans := map[string]string{"msg": m[2], "status": "done"}
	switch role {
	case "write":
		notes := ""
		if mm := mergeRe.FindStringSubmatch(brief); mm != nil {
			out, err := run(dir, "git merge --no-edit "+mm[1])
			say("%s", out)
			if err != nil {
				fix := fixRe.FindStringSubmatch(brief)
				if fix == nil {
					ans["status"], ans["summary"] = "failed", "merge conflict and no resolve recipe"
					break
				}
				if out, err := run(dir, fix[1]+"\ngit add -A && git -c user.name=stub -c user.email=stub@x commit -q --no-edit"); err != nil {
					say("resolve failed: %s", out)
				}
				notes = "resolved merge conflict"
			}
		}
		if s := shRe.FindStringSubmatch(brief); s != nil {
			out, err := run(dir, s[1])
			say("%s", out)
			if err != nil {
				ans["status"], ans["summary"] = "failed", "script failed: "+err.Error()
				break
			}
		}
		out, _ := run(dir, "git add -A && git -c user.name=stub-"+persona+" -c user.email=stub@x commit -q -m 'task "+task+" by "+persona+"' || true")
		say("%s", out)
		ans["summary"], ans["notes"] = "did task "+task+" as "+persona, notes
	case "review":
		branch := branchRe.FindStringSubmatch(brief)[1]
		diff, _ := run(dir, "git diff --stat "+branch+"...HEAD")
		if strings.Contains(brief, "REVIEW-FAIL-ONCE") && once(task, "review") {
			ans["verdict"], ans["summary"], ans["notes"] = "fail", "missing edge case", "handle the empty input case too"
		} else if strings.TrimSpace(diff) == "" {
			ans["verdict"], ans["summary"] = "fail", "no change to review"
		} else {
			ans["verdict"], ans["summary"] = "pass", "reviewed by "+persona+": "+strings.TrimSpace(strings.Split(diff, "\n")[0])
		}
	}
	out, _ := json.Marshal(ans)
	os.WriteFile(answerRe.FindStringSubmatch(brief)[1]+".tmp", out, 0o644)
	os.Rename(answerRe.FindStringSubmatch(brief)[1]+".tmp", answerRe.FindStringSubmatch(brief)[1])
	say("● done; result written")
	say("> ")
	hang()
}
