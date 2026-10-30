package goal

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// schedule moves tasks forward: start writers and reviewers, run checks and merges.
func (e *Engine) schedule() {
	g := e.G
	e.runReverts()
	merging := false
	for _, t := range g.Tasks {
		switch t.State {
		case TaskPending:
			if b := e.brokenDep(t); b != "" {
				if note := "waiting for a re-plan: dependency " + b; t.Note != note {
					t.Note = note
					e.event("dependency-broken", "", t.ID, note)
				}
				continue
			}
			if !e.depsDone(t) || !e.locksFree(t) {
				continue
			}
			if ok, why := e.canStart(); !ok {
				e.waitNote(why)
				return
			}
			if err := e.W.TaskWorktree(g, t); err != nil {
				e.event("error", "", t.ID, "worktree: "+err.Error())
				continue
			}
			w := t.Writer
			if w == "" || e.unusable(w) {
				w = e.pickProfile("write", "", "")
			}
			if w == "" {
				e.noProfile(t, "write")
				continue
			}
			if t.Writer != w {
				e.event("choose", "", t.ID, fmt.Sprintf("writer %s (record %+v)", w, *g.score(w)))
			}
			t.Writer = w
			for _, f := range t.Files {
				g.Locks[f] = t.ID
			}
			t.State = TaskRunning
			e.event("task-running", "", t.ID, "attempt "+fmt.Sprint(t.Attempts+1))
			e.startAgent("write", t.ID, w, "", nil, "")
		case TaskRunning, TaskReview:
			if t.Agent != "" && g.Agents[t.Agent] != nil && !g.Agents[t.Agent].Ended {
				continue
			}
			if ok, why := e.canStart(); !ok {
				e.waitNote(why)
				return
			}
			if e.jobsBusy(":"+t.ID+":") > 0 {
				continue // an old check/merge of this task is still stopping: no writer on top of it
			}
			if t.State == TaskRunning {
				e.startAgent("write", t.ID, t.Writer, "", nil, "Continue this task; earlier work is committed in this folder.")
				continue
			}
			r := t.Reviewer
			if r == "" || e.unusable(r) || e.kindOf(r) == e.kindOf(t.Writer) {
				r = e.pickProfile("review", e.kindOf(t.Writer), "")
			}
			if r == "" {
				e.noProfile(t, "review (different vendor from "+t.Writer+")")
				continue
			}
			t.Reviewer = r
			t.ReviewHead = e.W.BranchHead(t)
			e.event("choose", "", t.ID, "reviewer "+r+" for writer "+t.Writer+" on "+short(t.ReviewHead))
			e.startAgent("review", t.ID, r, "", nil, "")
		case TaskChecking:
			e.runCheck(t)
		case TaskMerging:
			if !merging {
				merging = true
				e.runMerge(t)
			}
		}
	}
}

// depsDone: only a merged prerequisite counts. A cancelled or failed one is not success; brokenDep
// names it so the lead can re-plan (task-update --deps).
func (e *Engine) depsDone(t *Task) bool {
	for _, d := range t.Deps {
		dt := e.G.Task(d)
		if dt == nil || dt.State != TaskDone {
			return false
		}
	}
	return true
}

func (e *Engine) brokenDep(t *Task) string {
	for _, d := range t.Deps {
		dt := e.G.Task(d)
		if dt == nil {
			return d + " (missing)"
		}
		if dt.State == TaskCancelled || dt.State == TaskFailed || dt.State == TaskBlocked {
			return d + " (" + dt.State + ")"
		}
		if b := e.brokenDep(dt); b != "" {
			return b
		}
	}
	return ""
}

// normPath makes reservations comparable: forward slashes, no ./ or trailing slash, and on Windows
// case-insensitive.
func normPath(p string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	p = path.Clean("/" + p)[1:]
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

// pathsOverlap: the same path, or one is a folder holding the other.
func pathsOverlap(a, b string) bool {
	a, b = normPath(a), normPath(b)
	return a == b || a == "" || b == "" || strings.HasPrefix(b, a+"/") || strings.HasPrefix(a, b+"/")
}

func (e *Engine) locksFree(t *Task) bool {
	for _, f := range t.Files {
		for held, owner := range e.G.Locks {
			if owner != t.ID && pathsOverlap(f, held) {
				return false
			}
		}
	}
	return true
}

func (e *Engine) unlock(t *Task) {
	for f, owner := range e.G.Locks {
		if owner == t.ID {
			delete(e.G.Locks, f)
		}
	}
}

var lastWait = map[string]string{}

func (e *Engine) waitNote(why string) {
	if lastWait[e.L.Dir] != why {
		lastWait[e.L.Dir] = why
		e.event("waiting-resources", "", "", why)
	}
}

// canStart is the readiness check before every new agent: slots, free memory, free disk.
func (e *Engine) canStart() (bool, string) {
	l := e.G.Limits
	// MaxAgents 0 is a measured "no agent fits": it defers work; a negative value means no cap.
	if n := e.liveAgents(""); l.MaxAgents >= 0 && n >= l.MaxAgents {
		return false, fmt.Sprintf("%d agents running (max %d)", n, l.MaxAgents)
	}
	// The process budget counts everything the goal runs: lead, delegates and background jobs
	// (checks, merges, acceptance, decider).
	if procs := e.liveAgents("") + e.liveAgents("lead") + e.runningJobs(); l.MaxProcs > 0 && procs >= l.MaxProcs {
		return false, fmt.Sprintf("%d goal processes running (budget %d: lead, agents and jobs)", procs, l.MaxProcs)
	}
	if ram := e.Probe.FreeRAMMB(); ram >= 0 && ram < l.MinFreeRAMMB+l.AgentRAMMB {
		return false, fmt.Sprintf("free RAM %d MB < %d MB reserve + %d MB for one more agent", ram, l.MinFreeRAMMB, l.AgentRAMMB)
	}
	if disk := e.Probe.FreeDiskMB(e.G.Project); disk >= 0 && disk < l.MinFreeDiskMB {
		return false, fmt.Sprintf("free disk %d MB < %d MB", disk, l.MinFreeDiskMB)
	}
	lastWait[e.L.Dir] = ""
	return true, ""
}

func (e *Engine) unusable(profile string) bool {
	p := e.G.Profile(profile)
	if p == nil || len(p.Command) == 0 {
		return true
	}
	if _, err := e.Probe.LookPath(p.Command[0]); err != nil {
		return true
	}
	prof, _ := e.freeAccount(profile)
	return prof == ""
}

// pickProfile chooses by role and by this goal's record (lessons): fewer failures and rejections first.
func (e *Engine) pickProfile(role, excludeKind, excludeProfile string) string {
	var cands []string
	for _, p := range e.G.Profiles {
		if !has(p.Roles, role) || p.Name == excludeProfile || excludeKind != "" && p.Kind == excludeKind || e.unusable(p.Name) {
			continue
		}
		cands = append(cands, p.Name)
	}
	rank := func(n string) int {
		s := e.G.score(n)
		return 2*s.Done - 3*s.Failed - 2*s.Rejected - s.Errors
	}
	sort.SliceStable(cands, func(i, j int) bool { return rank(cands[i]) > rank(cands[j]) })
	if len(cands) == 0 {
		return ""
	}
	return cands[0]
}

func has(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func (e *Engine) noProfile(t *Task, role string) {
	if t.Note != "no usable profile for "+role {
		t.Note = "no usable profile for " + role
		e.event("no-profile", "", t.ID, t.Note+" (CLI missing, or all its accounts limited)")
	}
}

// startAgent records the agent and its brief, then starts it under an intent.
func (e *Engine) startAgent(role, task, profile, account string, retries map[string]int, note string) *AgentRun {
	g := e.G
	p := g.Profile(profile)
	if p == nil {
		e.event("error", "", task, "unknown profile "+profile)
		return nil
	}
	if account == "" {
		_, account = e.freeAccount(profile)
	}
	g.Seq++
	name := fmt.Sprintf("%s-%s-%d", map[string]string{"write": "w", "review": "r", "lead": "lead"}[role], task, g.Seq)
	name = strings.ReplaceAll(name, "--", "-")
	msg := fmt.Sprintf("m%d", g.Seq)
	cwd := e.W.MainDir()
	t := g.Task(task)
	if t != nil {
		cwd = t.Worktree
	}
	a := &AgentRun{Name: name, Role: role, Task: task, Profile: profile, Account: account, Cwd: cwd, Msg: msg,
		State: "starting", Started: T(e.Now()), LastMove: T(e.Now()), Retries: map[string]int{}}
	for k, v := range retries {
		if k == "restart" {
			a.Retries[k] = v // restarts are capped across runs; the other counters start fresh
		}
	}
	if role != "lead" {
		a.Answer = filepath.Join(e.L.AnswerDir(), name+".json")
		os.Remove(a.Answer)
	}
	brief := e.brief(a, t, note)
	bdir := e.L.path("briefs")
	os.MkdirAll(bdir, 0o755)
	bpath := filepath.Join(bdir, name+".md")
	if err := writeAtomic(bpath, []byte(brief)); err != nil {
		e.event("error", name, task, "brief: "+err.Error())
		return nil
	}
	line := e.briefLine(a)
	argv := make([]string, 0, len(p.Command))
	for _, c := range p.Command {
		if strings.Contains(c, "{prompt}") {
			a.Sent = true
		}
		argv = append(argv, strings.NewReplacer("{prompt}", line, "{model}", p.Model, "{account}", account, "{brief}", bpath).Replace(c))
	}
	home, _ := os.UserHomeDir()
	env := map[string]string{"HERDR_GOAL": e.L.Dir, "HERDR_GOAL_ROLE": role, "HERDR_GOAL_AGENT": name}
	if role == "lead" {
		g.LeadNews = ""   // a new lead has been told nothing yet
		tok := newToken() // this lead's own key; replaced leads lose theirs
		for h, r := range g.Tokens {
			if r == "lead" {
				delete(g.Tokens, h)
				os.Remove(keyFile(e.L.Dir, "lead", h))
			}
		}
		id := keyID(tok)
		os.MkdirAll(filepath.Join(e.L.Dir, "keys"), 0o700)
		if err := os.WriteFile(keyFile(e.L.Dir, "lead", id), []byte(tok), 0o600); err != nil {
			e.event("error", name, "", "lead key: "+err.Error())
			return nil
		}
		g.Tokens[id] = "lead"
		env["HERDR_GOAL_KEY_FILE"] = keyFile(e.L.Dir, "lead", id) // a path, never the key itself
	}
	for k, v := range p.Env {
		env[k] = strings.NewReplacer("{account}", account, "{home}", home).Replace(v)
	}
	g.Messages = append(g.Messages, &Message{ID: msg, To: name, Kind: role, Body: bpath, Sent: T(e.Now()), Attempts: 1})
	g.Agents[name] = a
	if t != nil {
		t.Agent = name
	}
	e.event("agent-start", name, task, fmt.Sprintf("%s profile %s account %q in %s", role, profile, account, cwd))
	if err := e.do(Action{Kind: "start", Agent: name, Start: StartSpec{Name: name, Cwd: cwd, Argv: argv, Env: env}}); err != nil {
		e.event("start-failed", name, task, err.Error())
		a.Ended, a.State = true, "start-failed"
		g.score(profile).Errors++
		if t != nil {
			t.Agent = ""
			e.taskAttemptFailed(a, "agent could not start: "+err.Error())
		}
	}
	return a
}

func (e *Engine) briefLine(a *AgentRun) string {
	return fmt.Sprintf("Read and follow the brief %s (message %s). No human is watching: decide, act, record.", filepath.Join(e.L.path("briefs"), a.Name+".md"), a.Msg)
}

func (e *Engine) brief(a *AgentRun, t *Task, note string) string {
	g := e.G
	var b strings.Builder
	ack := e.ackPath(a.Msg)
	switch a.Role {
	case "lead":
		fmt.Fprintf(&b, "# You lead goal %s (message %s)\n\n%s\n\nProject: %s. Ledger: %s.\n\n", g.ID, a.Msg, g.Text, g.Project, e.L.Dir)
		fmt.Fprintf(&b, "First create the file %s (acknowledgement).\n\n", ack)
		b.WriteString("Follow the skill herdr-architect. No human is coming: decide and record with `herdr-goal decide`/`assume`.\n")
		fmt.Fprintf(&b, "See the state with `herdr-goal status %s`. Change the plan only with `herdr-goal task ...` commands; never edit ledger files.\n", e.L.Dir)
		b.WriteString("You cannot mark the goal done or stop it: the Supervisor runs the acceptance itself.\n")
	case "write":
		fmt.Fprintf(&b, "# Task %s: %s\n\nGoal %s, message %s. You are a delegate of the Herdr goal engine; no human is watching.\n\n", t.ID, t.Title, g.ID, a.Msg)
		fmt.Fprintf(&b, "1. First create the file %s (any content): it acknowledges this brief.\n", ack)
		fmt.Fprintf(&b, "2. Work only in this folder: %s (git branch %s). Commit your work with git; uncommitted work is committed for you if you stop.\n", t.Worktree, t.Branch)
		fmt.Fprintf(&b, "3. The task:\n\n%s\n\n", t.Spec)
		if len(t.Accept) > 0 {
			fmt.Fprintf(&b, "4. The engine itself runs these checks in your folder when you finish; they must pass:\n")
			for _, c := range t.Accept {
				fmt.Fprintf(&b, "   - `%s`\n", c)
			}
		}
		if t.Feedback != "" {
			fmt.Fprintf(&b, "\nFeedback from the last attempt or review (fix this):\n\n%s\n", t.Feedback)
		}
	case "review":
		fmt.Fprintf(&b, "# Review task %s: %s\n\nGoal %s, message %s. You review work written by another model (%s). No human is watching.\n\n", t.ID, t.Title, g.ID, a.Msg, t.Writer)
		fmt.Fprintf(&b, "1. First create the file %s (any content).\n", ack)
		fmt.Fprintf(&b, "2. In %s run `git diff %s...HEAD` and read the change. Do NOT change any file.\n", t.Worktree, g.Branch)
		fmt.Fprintf(&b, "3. The task it had to do:\n\n%s\n\n", t.Spec)
		if len(t.Accept) > 0 {
			fmt.Fprintf(&b, "Its checks (already green): %s\n", strings.Join(t.Accept, "; "))
		}
		b.WriteString("Verdict \"pass\" only if the change does the task correctly; else \"fail\" with concrete notes the writer can act on.\n")
	}
	if t != nil && e.Skills != nil && a.Role != "lead" {
		tools, profile := map[string]Fact{}, "unknown"
		if e.Caps != nil {
			tools, profile = e.Caps.Tools, e.Caps.Policy.Profile
		}
		chosen, skipped := e.Skills.Select(t.Tags, tools, profile)
		t.Skills = nil
		if len(chosen) > 0 {
			b.WriteString("\nSkills for this task (read each before you start; they are tested for this project):\n")
			for _, s := range chosen {
				fmt.Fprintf(&b, "- %s: %s\n", s.Name, s.Path)
				t.Skills = append(t.Skills, s.Name+"@"+s.Hash)
			}
		}
		if len(chosen)+len(skipped) > 0 {
			e.event("skills", a.Name, t.ID, fmt.Sprintf("chosen %v; left out %v", t.Skills, skipped))
		}
	}
	if note != "" {
		fmt.Fprintf(&b, "\nNote: %s\n", note)
	}
	if a.Answer != "" {
		fmt.Fprintf(&b, "\nLast step, only when finished: write %s as JSON\n", a.Answer)
		if a.Role == "review" {
			fmt.Fprintf(&b, "{\"msg\":\"%s\",\"status\":\"done\",\"verdict\":\"pass|fail\",\"summary\":\"...\",\"notes\":\"...\"}\n", a.Msg)
		} else {
			fmt.Fprintf(&b, "{\"msg\":\"%s\",\"status\":\"done|blocked|failed\",\"summary\":\"...\",\"notes\":\"assumptions, doubts\"}\n", a.Msg)
		}
		b.WriteString("\"blocked\" only for a sign-in or secret you must not handle. Never type secrets, never push, never edit outside your folder.\n")
	}
	return b.String()
}

// finished handles a result file.
func (e *Engine) finished(a *AgentRun, ans *Answer) {
	g := e.G
	if !a.Ended {
		e.event("result", a.Name, a.Task, fmt.Sprintf("%s %s: %s", ans.Status, ans.Verdict, oneLine(ans.Summary, 200)))
		if !e.endAgent(a, "finished") {
			// Still alive: it keeps the task. Nothing is checked, reviewed or merged until its stop is
			// confirmed (watch retries the stop and comes back here).
			return
		}
	}
	t := g.Task(a.Task)
	if t == nil {
		return
	}
	t.Agent = ""
	switch a.Role {
	case "write":
		switch ans.Status {
		case "done":
			if saved, _ := e.W.SaveWork(t, a.Name); saved {
				e.event("work-saved", a.Name, t.ID, "committed what the agent left uncommitted")
			}
			if e.W.Ahead(g, t) == 0 {
				e.taskAttemptFailed(a, "reported done but made no commit on "+t.Branch)
				return
			}
			t.Result = ans.Summary
			t.State = TaskChecking
			e.event("task-checking", a.Name, t.ID, "running the task's own checks")
		case "blocked":
			e.ownerBlock(a, "agent reports blocked: "+ans.Summary+" "+ans.Notes, "")
			e.unlock(t)
		default:
			e.taskAttemptFailed(a, "agent reports failure: "+ans.Summary+" "+ans.Notes)
		}
	case "review":
		if ans.Status != "done" || ans.Verdict != "pass" && ans.Verdict != "fail" {
			// blocked/failed/unclear: not a verdict on the work. Another reviewer looks at it.
			e.event("review-invalid", a.Name, t.ID, fmt.Sprintf("status %q verdict %q is not a review result", ans.Status, ans.Verdict))
			e.taskAttemptFailed(a, "reviewer gave no valid verdict: "+ans.Summary)
			return
		}
		if head := e.W.BranchHead(t); t.ReviewHead != "" && head != t.ReviewHead {
			e.event("review-stale", a.Name, t.ID, "branch moved from "+short(t.ReviewHead)+" to "+short(head)+" after review began; reviewing again")
			t.ReviewHead, t.Reviewer = "", ""
			return
		}
		if ans.Verdict == "pass" {
			t.Proof = append(t.Proof, fmt.Sprintf("review pass by %s (%s): %s", a.Profile, e.kindOf(a.Profile), oneLine(ans.Summary, 200)))
			t.State = TaskMerging
			e.event("task-approved", a.Name, t.ID, ans.Summary)
			return
		}
		t.Reviews++
		g.score(t.Writer).Rejected++
		t.Feedback = "Reviewer " + a.Profile + " rejected the change: " + ans.Summary + "\n" + ans.Notes
		if t.Reviews > g.Limits.MaxAttempts {
			e.failTask(t, "rejected by review too often: "+ans.Summary)
			return
		}
		t.State = TaskRunning
		e.event("task-rejected", a.Name, t.ID, ans.Summary+" "+ans.Notes)
	}
}

// taskAttemptFailed: retry with the feedback; the last attempt goes to a different vendor; then failed.
func (e *Engine) taskAttemptFailed(a *AgentRun, why string) {
	g := e.G
	if !a.Ended && !e.endAgent(a, why) {
		return // not confirmed stopped: it keeps the task; watch retries the stop
	}
	t := g.Task(a.Task)
	if t == nil {
		if a.Role == "lead" {
			e.setState(GoalNeedsOwner, "no lead can run: "+why)
		}
		return
	}
	t.Agent = ""
	if a.Role == "review" {
		t.Reviews++
		t.Reviewer = ""
		if t.Reviews > g.Limits.MaxAttempts {
			e.failTask(t, "no reviewer could finish: "+why)
		}
		return
	}
	t.Attempts++
	t.Total++
	g.score(a.Profile).Failed++
	t.Feedback = why
	if t.Total >= 3*g.Limits.MaxAttempts {
		e.failTask(t, fmt.Sprintf("failure budget used up (%d attempts across re-plans): %s", t.Total, why))
		t.Note = "BUDGET: " + t.Note
		return
	}
	if t.Attempts >= g.Limits.MaxAttempts {
		e.failTask(t, why)
		return
	}
	if t.Attempts == g.Limits.MaxAttempts-1 {
		if alt := e.pickProfile("write", e.kindOf(t.Writer), ""); alt != "" {
			e.event("model-switch", "", t.ID, "last attempt goes to another vendor: "+alt)
			t.Writer = alt
		}
	}
	t.State = TaskRunning
	e.event("task-retry", "", t.ID, why)
}

func (e *Engine) failTask(t *Task, why string) {
	t.State, t.Note = TaskFailed, why
	e.unlock(t)
	e.event("task-failed", "", t.ID, why)
}

// startJob runs fn once per key in the background (inline in tests) and returns the finished job,
// or nil while it runs. main jobs own the integration worktree: only one runs at a time (merge, verify,
// revert). Every job has a cancel; cancelling kills its process tree.
func (e *Engine) startJob(key string, main bool, fn func(ctx context.Context, j *job)) *job {
	e.mu.Lock()
	for k, o := range e.jobs { // forget cancelled jobs once they have really finished
		if o.cancelled && o.done {
			delete(e.jobs, k)
		}
	}
	j := e.jobs[key]
	if j != nil && j.cancelled {
		e.mu.Unlock()
		return nil // still winding down; a new job with this key starts after it has finished
	}
	if j == nil {
		if main {
			for k, o := range e.jobs {
				if o.main && !o.done && k != key {
					e.mu.Unlock()
					return nil // the integration worktree is busy; try on a later tick
				}
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		j = &job{cancel: cancel, main: main}
		e.jobs[key] = j
		e.mu.Unlock()
		if e.Sync {
			fn(ctx, j)
			j.done = true
		} else {
			go func() {
				fn(ctx, j)
				e.mu.Lock()
				j.done = true
				e.mu.Unlock()
			}()
		}
		e.mu.Lock()
	}
	defer e.mu.Unlock()
	if !j.done {
		return nil
	}
	delete(e.jobs, key)
	j.cancel()
	return j
}

// cancelJobs cancels running jobs whose key contains match ("" = all) and forgets them: a result that
// arrives later is never applied.
// cancelJobs cancels jobs whose key contains match ("" = all). A cancelled job stays owned and counted
// (worktree owner, process budget) until it has really finished; its result is never applied.
func (e *Engine) cancelJobs(match string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for k, j := range e.jobs {
		if match == "" || strings.Contains(k, match) {
			j.cancel()
			j.cancelled = true
		}
	}
}

// jobsBusy counts unfinished jobs (cancelled ones included) whose key contains match.
func (e *Engine) jobsBusy(match string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for k, j := range e.jobs {
		if !j.done && (match == "" || strings.Contains(k, match)) {
			n++
		}
	}
	return n
}

func (e *Engine) runningJobs() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for _, j := range e.jobs {
		if !j.done {
			n++
		}
	}
	return n
}

func jobKey(kind string, t *Task) string { return fmt.Sprintf("%s:%s:%d", kind, t.ID, t.Gen) }

func (e *Engine) runCheck(t *Task) {
	dir, cmds, to := t.Worktree, t.Accept, e.G.Limits.VerifyTimeout.D()
	head := e.W.BranchHead(t)
	j := e.startJob(jobKey("check", t), false, func(ctx context.Context, j *job) {
		j.ok, j.out = RunChecksCtx(ctx, dir, cmds, to)
		j.sha = head
	})
	if j == nil {
		return
	}
	if now := e.W.BranchHead(t); now != j.sha {
		e.event("stale-result", "", t.ID, "checks ran on "+short(j.sha)+" but the branch is at "+short(now)+"; checking again")
		return
	}
	if j.ok {
		t.Proof = append(t.Proof, "task checks green on "+short(j.sha)+": "+strings.Join(t.Accept, "; "))
		t.State = TaskReview
		e.event("task-review", "", t.ID, "checks green; to review")
		return
	}
	e.event("task-check-failed", "", t.ID, j.out)
	a := &AgentRun{Name: "", Role: "write", Task: t.ID, Profile: t.Writer, Ended: true}
	e.taskAttemptFailed(a, "the engine's run of your checks failed:\n"+j.out)
}

func (e *Engine) runMerge(t *Task) {
	g := e.G
	w, to := e.W, g.Limits.VerifyTimeout.D()
	tc := *t
	j := e.startJob(jobKey("merge", t), true, func(ctx context.Context, j *job) { j.sha, j.out, j.err = w.MergeCtx(ctx, g, &tc, to) })
	if j == nil {
		return
	}
	if j.err == nil {
		t.Merge, t.State = j.sha, TaskDone
		t.Proof = append(t.Proof, "merged "+short(tc.ReviewHead)+" as "+short(j.sha)+"; project tests green")
		g.score(t.Writer).Done++
		e.unlock(t)
		e.event("task-done", "", t.ID, "merged "+short(j.sha))
		e.W.RemoveTask(t)
		return
	}
	e.event("merge-failed", "", t.ID, j.err.Error())
	t.Reviews = 0
	a := &AgentRun{Role: "write", Task: t.ID, Profile: t.Writer, Ended: true}
	e.taskAttemptFailed(a, fmt.Sprintf("Merging your branch into %s failed: %v\nMerge %s into your branch, fix it, commit, then finish again.\n%s", g.Branch, j.err, g.Branch, j.out))
}

// runReverts applies queued reverts when the integration worktree is free.
func (e *Engine) runReverts() {
	g := e.G
	if len(g.Reverts) == 0 {
		return
	}
	t := g.Task(g.Reverts[0])
	if t == nil || t.State != TaskDone || t.Reverted {
		g.Reverts = g.Reverts[1:]
		return
	}
	w := e.W
	tc := *t
	j := e.startJob("revert:"+t.ID, true, func(ctx context.Context, j *job) { j.err = w.Revert(&tc) })
	if j == nil {
		return
	}
	g.Reverts = g.Reverts[1:]
	if j.err != nil {
		e.event("revert-failed", "", t.ID, j.err.Error())
		return
	}
	t.Reverted, t.State = true, TaskCancelled
	g.Verify = nil
	e.event("reverted", "", t.ID, "merge "+short(t.Merge)+" reverted; later merges kept")
}

// verify is the gate to done: the Supervisor runs the goal's acceptance itself on the goal branch.
func (e *Engine) verify() {
	g := e.G
	if len(g.Tasks) == 0 {
		return
	}
	open, failed, blocked := 0, 0, 0
	for _, t := range g.Tasks {
		switch t.State {
		case TaskDone, TaskCancelled:
		case TaskFailed:
			failed++
		case TaskBlocked:
			blocked++
		case TaskPending:
			if e.brokenDep(t) != "" {
				failed++ // cannot run until the lead re-plans it
			} else {
				open++
			}
		default:
			open++
		}
	}
	if open > 0 {
		return
	}
	if failed > 0 && g.Lead != "" {
		return // the lead re-plans failed tasks
	}
	if failed+blocked > 0 {
		// Only the owner can move on: secrets, sign-ins, or two models failed with no lead to re-plan.
		e.setState(GoalNeedsOwner, fmt.Sprintf("%d task(s) need the owner (blocked %d, failed %d); all other work is done and kept", failed+blocked, blocked, failed))
		return
	}
	head, err := e.W.Head()
	if err != nil {
		e.event("error", "", "", err.Error())
		return
	}
	if g.State != GoalVerifying {
		e.setState(GoalVerifying, "all tasks merged; running acceptance on "+short(head))
	}
	dir, cmds, to := e.W.MainDir(), g.Accept, g.Limits.VerifyTimeout.D()
	if g.Verify == nil || g.Verify.Head != head {
		g.Verify = &VerifyRun{Head: head, Started: T(e.Now())}
	}
	j := e.startJob("verify:"+head, true, func(ctx context.Context, j *job) { j.ok, j.out = RunChecksCtx(ctx, dir, cmds, to) })
	if j == nil {
		return
	}
	g.Verify.Done, g.Verify.Ok, g.Verify.Output = true, j.ok, j.out
	for _, t := range g.Tasks {
		if t.State == TaskDone && !reviewed(t) {
			e.setState(GoalFailed, "task "+t.ID+" has no independent review pass")
			return
		}
	}
	if !j.ok {
		e.event("acceptance-failed", "", "", j.out)
		if g.Lead != "" {
			e.setState(GoalRunning, "acceptance failed; the lead re-plans")
			return
		}
		e.setState(GoalFailed, "acceptance failed: "+oneLine(j.out, 300))
		return
	}
	e.event("acceptance-passed", "", "", j.out)
	e.stopAll("goal done")
	e.setState(GoalDone, fmt.Sprintf("acceptance green on %s at %s", short(head), e.Now().Format(time.RFC3339)))
}

func reviewed(t *Task) bool {
	for _, p := range t.Proof {
		if strings.HasPrefix(p, "review pass by ") {
			return true
		}
	}
	return false
}

// lead keeps the Architect alive and tells it when something needs a decision.
func (e *Engine) lead() {
	g := e.G
	if g.Lead == "" {
		return
	}
	var cur *AgentRun
	for _, a := range g.Agents {
		if a.Role == "lead" && !a.Ended {
			cur = a
		}
	}
	if cur == nil {
		g.LeadStarts++
		if g.LeadStarts > 3*g.Limits.RetryCap+1 {
			e.setState(GoalNeedsOwner, fmt.Sprintf("the lead was started %d times and keeps ending; all work is kept", g.LeadStarts-1))
			return
		}
		e.startAgent("lead", "", g.Lead, "", nil, "")
		g.LeadSeen = g.Seq
		return
	}
	var news []string
	for _, x := range g.Escalations {
		if !x.Resolved && !strings.HasPrefix(x.Problem, "OWNER:") {
			news = append(news, "esc "+x.ID)
		}
	}
	for _, t := range g.Tasks {
		if t.State == TaskFailed {
			news = append(news, "failed "+t.ID+fmt.Sprint(t.Updated.Unix()))
		}
	}
	if len(g.Tasks) == 0 {
		news = append(news, "no plan")
	}
	if g.Verify != nil && g.Verify.Done && !g.Verify.Ok {
		news = append(news, "acceptance failed "+g.Verify.Head)
	}
	key := strings.Join(news, ",")
	if key != "" && key != g.LeadNews && cur.Begun && (cur.State == "idle" || cur.State == "stopped-early") {
		g.LeadSeen, g.LeadNews = g.Seq, key
		e.send(cur, fmt.Sprintf("Next step needed. Run `herdr-goal status %s`, decide, act with herdr-goal commands, record why.", e.L.Dir))
	}
}
