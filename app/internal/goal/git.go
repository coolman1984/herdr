package goal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func init() {
	os.Setenv("GIT_TERMINAL_PROMPT", "0") // git never waits for a password prompt
	os.Setenv("GIT_EDITOR", "true")
}

// gitTimeout bounds every git call: a hung git (lock, prompt, network) must not hang the runner.
var gitTimeout = 3 * time.Minute

func git(dir string, args ...string) (string, error) {
	return gitCtx(context.Background(), dir, args...)
}

func gitCtx(ctx context.Context, dir string, args ...string) (string, error) {
	full := append([]string{"-c", "core.quotepath=false"}, args...)
	out, err, timedOut := runBounded(ctx, dir, gitTimeout, 256<<10, "git", full...)
	s := strings.TrimSpace(out)
	if timedOut {
		return s, fmt.Errorf("git %s: timed out after %s", strings.Join(args, " "), gitTimeout)
	}
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

// autosaveSkip are untracked paths never auto-committed: caches, dependencies, builds, logs and
// anything that looks like a secret. An agent that wants them committed commits them itself.
var autosaveSkip = regexp.MustCompile(`(?i)(^|/)(__pycache__|node_modules|\.venv|venv|\.pytest_cache|\.mypy_cache|\.gradle|target|bin|obj|dist|build|\.next|\.cache|coverage)(/|$)|\.(pyc|pyo|class|o|obj|exe|dll|so|dylib|log|tmp|swp|pem|key|pfx|p12|kdbx)$|(^|/)(\.env(\..*)?|id_rsa.*|id_ed25519.*|credentials.*|secrets?\..*|auth\.json|\.netrc|\.npmrc|\.pypirc)$`)

const autosaveMaxBytes = 5 << 20

// SaveWork commits what an agent left behind, selectively: changes to tracked files, plus new files
// that are not caches, dependencies, builds, secrets or larger than 5 MB. It reports what it skipped.
func (w Workspace) SaveWork(t *Task, who string) (bool, error) {
	if t.Worktree == "" {
		return false, nil
	}
	if _, err := git(t.Worktree, "add", "-u"); err != nil {
		return false, err
	}
	untracked, err := git(t.Worktree, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return false, err
	}
	var add []string
	for _, f := range strings.Split(untracked, "\x00") {
		if f == "" || autosaveSkip.MatchString(f) {
			continue
		}
		if st, err := os.Stat(filepath.Join(t.Worktree, f)); err != nil || st.Size() > autosaveMaxBytes {
			continue
		}
		add = append(add, f)
	}
	for len(add) > 0 {
		n := min(len(add), 100)
		if _, err := git(t.Worktree, append([]string{"add", "--"}, add[:n]...)...); err != nil {
			return false, err
		}
		add = add[n:]
	}
	if staged, _ := git(t.Worktree, "diff", "--cached", "--name-only"); staged == "" {
		return false, nil
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
	return w.MergeCtx(context.Background(), g, t, timeout)
}

// MergeCtx: the integration worktree may hold untracked files from test runs (caches, an offline
// node_modules or venv the project needs). They are kept. Only the untracked files that the incoming
// commit would overwrite - which git refuses to touch - are removed, and only when git names them.
func (w Workspace) MergeCtx(ctx context.Context, g *Goal, t *Task, timeout time.Duration) (sha, output string, err error) {
	dir := w.MainDir()
	pre, err := w.Head()
	if err != nil {
		return "", "", err
	}
	gitCtx(ctx, dir, "reset", "-q", "--hard") // tracked files back to the goal branch; untracked kept
	merge := func() (string, error) {
		return gitCtx(ctx, dir, "-c", "user.name=herdr-goal", "-c", "user.email=herdr-goal@localhost",
			"merge", "--no-ff", "--no-edit", "-m", fmt.Sprintf("herdr-goal %s: merge %s %s", g.ID, t.ID, t.Title), mergeTarget(t))
	}
	out, err := merge()
	if err != nil {
		if blocking := untrackedBlockers(out); len(blocking) > 0 {
			for _, f := range blocking {
				if p, ok := insideDir(dir, f); ok {
					os.Remove(p)
				}
			}
			out, err = merge()
		}
	}
	if err != nil {
		files, _ := gitCtx(ctx, dir, "diff", "--name-only", "--diff-filter=U")
		git(dir, "merge", "--abort")
		git(dir, "reset", "-q", "--hard", pre)
		if files != "" {
			return "", out, fmt.Errorf("%w in %s", ErrConflict, strings.ReplaceAll(files, "\n", ", "))
		}
		return "", out, err
	}
	ok, out := RunChecksCtx(ctx, dir, g.Tests, timeout)
	if !ok || ctx.Err() != nil {
		git(dir, "reset", "-q", "--hard", pre)
		if ctx.Err() != nil {
			return "", out, fmt.Errorf("merge of %s cancelled; goal branch reset to %s", t.ID, short(pre))
		}
		return "", out, fmt.Errorf("the project's tests failed after merging %s; goal branch reset to %s", t.ID, short(pre))
	}
	sha, err = w.Head()
	return sha, out, err
}

// untrackedBlockers reads git's "untracked working tree files would be overwritten" list.
func untrackedBlockers(out string) []string {
	var files []string
	in := false
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.Contains(l, "untracked working tree files would be overwritten"):
			in = true
		case in && strings.HasPrefix(l, "\t"):
			files = append(files, strings.TrimSpace(l))
		case in:
			in = false
		}
	}
	return files
}

// insideDir resolves rel under dir and refuses anything that escapes it.
func insideDir(dir, rel string) (string, bool) {
	p := filepath.Join(dir, filepath.FromSlash(rel))
	r, err := filepath.Rel(dir, p)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", false
	}
	return p, true
}

// Snapshot makes a detached worktree at the task's reviewed commit for the reviewer.
func (w Workspace) Snapshot(t *Task) (string, error) {
	dir := filepath.Join(w.Ledger.WorktreeDir(), "review-"+t.ID)
	git(w.Project, "worktree", "remove", "--force", dir)
	if _, err := git(w.Project, "worktree", "add", "--detach", dir, t.ReviewHead); err != nil {
		return "", err
	}
	return dir, nil
}

// ProtectedTouched lists the protected paths the task's reviewed commit changes.
func (w Workspace) ProtectedTouched(g *Goal, t *Task) []string {
	if len(g.Protected) == 0 {
		return nil
	}
	out, err := git(w.Project, "diff", "--name-only", g.Branch+"..."+mergeTarget(t))
	if err != nil {
		return []string{"(could not list the change: " + err.Error() + ")"}
	}
	var hit []string
	for _, f := range strings.Split(out, "\n") {
		for _, p := range g.Protected {
			if f != "" && pathsOverlap(p, f) {
				hit = append(hit, f)
			}
		}
	}
	return hit
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
	git(w.MainDir(), "reset", "-q", "--hard")
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
	return RunChecksCtx(context.Background(), dir, cmds, timeout)
}

// RunChecksCtx runs commands in dir, stops at the first failure or on cancel, kills the whole process
// tree on timeout or cancel, and keeps bounded output.
func RunChecksCtx(ctx context.Context, dir string, cmds []string, timeout time.Duration) (bool, string) {
	var out strings.Builder
	for _, c := range cmds {
		name, args := shellArgv(c)
		b, err, timedOut := runBounded(ctx, dir, timeout, 64<<10, name, args...)
		fmt.Fprintf(&out, "$ %s\n%s", c, b)
		if timedOut {
			fmt.Fprintf(&out, "TIMEOUT after %s (process tree killed)\n", timeout)
			return false, tailString(out.String(), 4000)
		}
		if ctx.Err() != nil {
			fmt.Fprintf(&out, "CANCELLED (process tree killed)\n")
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
