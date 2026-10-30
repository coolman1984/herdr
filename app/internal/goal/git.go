package goal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_EDITOR=true")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		return s, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, oneLine(s, 300))
	}
	return s, nil
}

// CheckRepo refuses a project that is not the top of a git repository: worktrees, merges and
// checkpoints all need git, and the engine never runs `git init` in someone's folder by itself.
func CheckRepo(project string) (top, branch string, err error) {
	top, err = git(project, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", fmt.Errorf("%s is not a git repository; run `git init` there and commit once, then create the goal", project)
	}
	if _, err := git(project, "rev-parse", "--verify", "HEAD"); err != nil {
		return "", "", fmt.Errorf("%s has no commit yet; commit once, then create the goal", project)
	}
	branch, err = git(project, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", "", err
	}
	return top, branch, nil
}

// excludeLedger keeps .herdr-goal/ out of every commit without touching the project's .gitignore.
func excludeLedger(project string) error {
	gd, err := git(project, "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(project, gd)
	}
	p := filepath.Join(gd, "info", "exclude")
	b, _ := os.ReadFile(p)
	if bytes.Contains(b, []byte("/.herdr-goal/")) {
		return nil
	}
	os.MkdirAll(filepath.Dir(p), 0o755)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n/.herdr-goal/\n")
	return err
}

// Workspace is the git side of a goal: the integration worktree on goal/<id>, one worktree per task.
type Workspace struct {
	Project string
	Ledger  Ledger
}

func (w Workspace) MainDir() string            { return filepath.Join(w.Ledger.WorktreeDir(), "_main") }
func (w Workspace) TaskDir(task string) string { return filepath.Join(w.Ledger.WorktreeDir(), task) }

func (w Workspace) Setup(g *Goal) error {
	if err := excludeLedger(w.Project); err != nil {
		return err
	}
	if _, err := os.Stat(w.MainDir()); err == nil {
		return nil
	}
	if _, err := git(w.Project, "rev-parse", "--verify", g.Branch); err == nil {
		_, err = git(w.Project, "worktree", "add", w.MainDir(), g.Branch)
		return err
	}
	_, err := git(w.Project, "worktree", "add", "-b", g.Branch, w.MainDir(), g.Base)
	return err
}

func (w Workspace) Head() (string, error) { return git(w.MainDir(), "rev-parse", "HEAD") }

// TaskWorktree creates (or reuses: a retry keeps the earlier attempt's work) the task's worktree.
func (w Workspace) TaskWorktree(g *Goal, t *Task) error {
	dir := w.TaskDir(t.ID)
	t.Worktree, t.Branch = dir, g.Branch+"-"+t.ID
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil
	}
	if _, err := git(w.Project, "rev-parse", "--verify", t.Branch); err == nil {
		_, err = git(w.Project, "worktree", "add", dir, t.Branch)
		return err
	}
	_, err := git(w.Project, "worktree", "add", "-b", t.Branch, dir, g.Branch)
	return err
}

// SaveWork commits anything an agent left uncommitted, so no work is lost when it dies or forgets.
func (w Workspace) SaveWork(t *Task, who string) (bool, error) {
	if t.Worktree == "" {
		return false, nil
	}
	st, err := git(t.Worktree, "status", "--porcelain")
	if err != nil || st == "" {
		return false, err
	}
	if _, err := git(t.Worktree, "add", "-A"); err != nil {
		return false, err
	}
	_, err = git(t.Worktree, "-c", "user.name=herdr-goal", "-c", "user.email=herdr-goal@localhost",
		"commit", "-q", "--no-verify", "-m", fmt.Sprintf("herdr-goal: save uncommitted work of %s (%s)", who, t.ID))
	return err == nil, err
}

// BranchHead is the task branch's current commit ("" when unknown).
func (w Workspace) BranchHead(t *Task) string {
	if t.Branch == "" {
		return ""
	}
	out, err := git(w.Project, "rev-parse", "--verify", t.Branch)
	if err != nil {
		return ""
	}
	return out
}

// Ahead counts the task branch's commits that are not on the goal branch yet.
func (w Workspace) Ahead(g *Goal, t *Task) int {
	out, err := git(w.Project, "rev-list", "--count", g.Branch+".."+t.Branch)
	if err != nil {
		return 0
	}
	var n int
	fmt.Sscan(out, &n)
	return n
}

// Diff is what a reviewer reviews.
func (w Workspace) Diff(g *Goal, t *Task) string {
	out, _ := git(w.Project, "diff", "--stat", g.Branch+"..."+t.Branch)
	return out
}

var ErrConflict = errors.New("merge conflict")

// Merge merges the task branch into the goal branch and runs the project's full tests there. Any
// failure puts the goal branch back where it was, so a bad change never stays and good ones are kept.
func (w Workspace) Merge(g *Goal, t *Task, timeout time.Duration) (sha, output string, err error) {
	dir := w.MainDir()
	pre, err := w.Head()
	if err != nil {
		return "", "", err
	}
	// The integration worktree is the engine's own: test runs leave files there (caches, builds) that
	// would block a merge. Start every merge from a clean tree.
	git(dir, "reset", "-q", "--hard")
	git(dir, "clean", "-q", "-fdx")
	if out, err := git(dir, "-c", "user.name=herdr-goal", "-c", "user.email=herdr-goal@localhost",
		"merge", "--no-ff", "--no-edit", "-m", fmt.Sprintf("herdr-goal %s: merge %s %s", g.ID, t.ID, t.Title), mergeTarget(t)); err != nil {
		files, _ := git(dir, "diff", "--name-only", "--diff-filter=U")
		git(dir, "merge", "--abort")
		git(dir, "reset", "--hard", pre)
		if files != "" {
			return "", out, fmt.Errorf("%w in %s", ErrConflict, strings.ReplaceAll(files, "\n", ", "))
		}
		return "", out, err
	}
	ok, out := RunChecks(dir, g.Tests, timeout)
	if !ok {
		git(dir, "reset", "--hard", pre)
		return "", out, fmt.Errorf("the project's tests failed after merging %s; goal branch reset to %s", t.ID, short(pre))
	}
	sha, err = w.Head()
	return sha, out, err
}

// mergeTarget is the exact commit the reviewer approved, so later pushes to the branch are not merged
// unreviewed.
func mergeTarget(t *Task) string {
	if t.ReviewHead != "" {
		return t.ReviewHead
	}
	return t.Branch
}

// Revert undoes one merged task on the goal branch with a new commit; later good merges stay.
func (w Workspace) Revert(t *Task) error {
	if t.Merge == "" {
		return fmt.Errorf("task %s was never merged", t.ID)
	}
	_, err := git(w.MainDir(), "-c", "user.name=herdr-goal", "-c", "user.email=herdr-goal@localhost",
		"revert", "-m", "1", "--no-edit", t.Merge)
	return err
}

func (w Workspace) RemoveTask(t *Task) {
	if t.Worktree != "" {
		git(w.Project, "worktree", "remove", "--force", t.Worktree)
	}
}

// RunChecks runs commands in dir, stops at the first failure, returns the combined output tail.
func RunChecks(dir string, cmds []string, timeout time.Duration) (bool, string) {
	var out strings.Builder
	for _, c := range cmds {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "pwsh", "-NoProfile", "-Command", c)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", c)
		}
		cmd.Dir = dir
		hideWindow(cmd)
		b, err := cmd.CombinedOutput()
		timedOut := ctx.Err() == context.DeadlineExceeded
		cancel()
		fmt.Fprintf(&out, "$ %s\n%s", c, b)
		if timedOut {
			fmt.Fprintf(&out, "TIMEOUT after %s\n", timeout)
			return false, tailString(out.String(), 4000)
		}
		if err != nil {
			fmt.Fprintf(&out, "FAILED: %v\n", err)
			return false, tailString(out.String(), 4000)
		}
	}
	return true, tailString(out.String(), 4000)
}

func tailString(s string, n int) string {
	if len(s) > n {
		return "..." + s[len(s)-n:]
	}
	return s
}

func short(sha string) string {
	if len(sha) > 10 {
		return sha[:10]
	}
	return sha
}
