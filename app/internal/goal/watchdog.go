package goal

import (
	"fmt"
	"strings"
	"time"
)

// watch is the Agent Watchdog for one agent run: sense, answer, nudge, restart, escalate, record.
func (e *Engine) watch(a *AgentRun) {
	if a.Retries == nil {
		a.Retries = map[string]int{}
	}
	now := e.Now()
	o := e.B.Observe(a.Name)
	if o.Err != nil {
		// Herdr could not be asked: that is not "dead". Count it and look again next tick.
		if e.retry(a, "backend-error") {
			e.event("backend-error", a.Name, a.Task, o.Err.Error())
		} else if a.Retries["backend-error"] == e.G.Limits.RetryCap+1 {
			e.escalate(a, "Herdr does not answer about this agent: "+o.Err.Error(), "")
		}
		return
	}
	a.Retries["backend-error"] = 0
	e.checkAck(a)

	// A result file wins over everything else: the agent finished.
	ans, err := e.readAnswer(a)
	if err != nil {
		e.setAgent(a, "error")
		if e.retry(a, "bad-answer") {
			e.send(a, fmt.Sprintf("Your result file is invalid: %s. Rewrite %s exactly as your brief says.", err, a.Answer))
		} else {
			e.restartOrFail(a, "bad result file: "+err.Error())
		}
		return
	}
	if ans != nil {
		e.finished(a, ans)
		return
	}

	if !o.Exists || o.Status == StatusExited {
		e.setAgent(a, "dead")
		e.event("agent-dead", a.Name, a.Task, "agent process is gone without a result", o.Screen)
		e.restartOrFail(a, "agent died")
		return
	}

	sight := e.P.Classify(o.Screen, now)
	if sight.Kind != "" && sight.Kind != "secret" && a.AnsHash != "" && a.AnsHash == screenHash(o.Screen) {
		// An answer was sent to exactly this screen; give the CLI time to react before answering again.
		if now.Sub(a.AnsAt.Time) < 30*time.Second {
			return
		}
		a.AnsHash = ""
	}
	switch sight.Kind {
	case "secret":
		if a.State != "needs-owner" {
			e.setAgent(a, "needs-owner")
			e.ownerBlock(a, fmt.Sprintf("asks for a secret or sign-in (%q); never answered by the engine", sight.Line), o.Screen)
		}
		return
	case "trust":
		e.setAgent(a, "trust")
		if !insideProject(e.G.Project, a.Cwd) {
			if a.State != "needs-owner" {
				e.setAgent(a, "needs-owner")
				e.ownerBlock(a, "trust prompt for a folder outside the goal's project: "+a.Cwd, o.Screen)
			}
			return
		}
		keys := TrustKeys(sight.Options)
		if keys == nil || !e.retry(a, "trust") {
			e.escalate(a, "cannot answer this trust prompt safely", o.Screen)
			e.restartOrFail(a, "trust prompt not answerable")
			return
		}
		e.keys(a, keys, "trust inside project: "+sight.Line, o.Screen)
		return
	case "popup", "queued":
		e.setAgent(a, sight.Kind)
		if e.retry(a, sight.Kind) {
			e.keys(a, sight.Keys, sight.Name, o.Screen)
		} else {
			e.restartOrFail(a, sight.Name+" keeps coming back")
		}
		return
	case "choice":
		e.setAgent(a, "choice")
		if !e.retry(a, "choice") {
			e.escalate(a, "menu keeps coming back after answers", o.Screen)
			e.restartOrFail(a, "menu loop")
			return
		}
		e.choose(a, o.Screen, sight)
		return
	case "capacity", "model-error":
		e.setAgent(a, "error")
		e.event(sight.Kind, a.Name, a.Task, sight.Line, o.Screen)
		e.G.score(a.Profile).Errors++
		if e.retry(a, sight.Kind) && sight.Kind == "capacity" {
			// LESSONS #98: resuming in the same conversation keeps what it already read.
			e.send(a, "continue from where you stopped")
			return
		}
		e.switchModel(a, sight.Kind+": "+sight.Line)
		return
	case "rate-limited":
		e.setAgent(a, "rate-limited")
		e.accountLimited(a, sight, o.Screen)
		return
	case "context-low":
		if a.Role == "lead" {
			e.handoff(a)
			return
		}
	}

	// No prompt on screen: launch check, progress, early stop, stall.
	h := ProgressHash(o.Screen)
	moved := h != a.LastHash
	if moved {
		if a.LastHash != "" {
			a.Begun = true
		}
		a.LastHash, a.LastMove = h, T(now)
		a.Retries["stall"] = 0
	}
	if o.Status == StatusWorking || o.Status == "running" {
		a.Begun = true
	}
	if !a.Sent && (o.Status == StatusIdle || o.Status == StatusDone) {
		// The CLI took no first prompt on its command line: type the brief once it is ready.
		a.Sent = true
		e.send(a, e.briefLine(a))
		return
	}
	if !a.Begun {
		if now.Sub(a.Started.Time) > e.G.Limits.StartTimeout.D() {
			if e.retry(a, "start") {
				e.event("not-started", a.Name, a.Task, "no ack, no work, no output since start; resending the brief", o.Screen)
				e.send(a, e.briefLine(a))
				a.Started = T(now)
			} else {
				e.restartOrFail(a, "never began its task")
			}
		}
		return
	}
	e.setAgent(a, map[bool]string{true: "working", false: "idle"}[o.Status == StatusWorking])
	switch o.Status {
	case StatusIdle, StatusDone:
		if a.IdleSince.IsZero() {
			a.IdleSince = T(now)
			return
		}
		if now.Sub(a.IdleSince.Time) < e.G.Limits.IdleGrace.D() || a.Role == "lead" {
			return
		}
		// Idle is not done: no result file yet.
		e.setAgent(a, "stopped-early")
		n := a.Retries["nudge"]
		if !e.retry(a, "nudge") {
			e.restartOrFail(a, "stops without a result after nudges")
			return
		}
		if n == 0 {
			e.keys(a, []string{"Enter"}, "stopped early: empty Enter (queued text)", o.Screen)
		} else {
			e.send(a, fmt.Sprintf("continue. You are not done until you write %s as your brief says (message %s).", a.Answer, a.Msg))
		}
		a.IdleSince = T(now)
	default:
		a.IdleSince = Time{}
		if !moved && now.Sub(a.LastMove.Time) > e.G.Limits.StallAfter.D() {
			e.setAgent(a, "stalled")
			if e.retry(a, "stall") {
				e.event("stalled", a.Name, a.Task, fmt.Sprintf("screen unchanged for %s", now.Sub(a.LastMove.Time).Round(time.Second)), o.Screen)
				e.keys(a, []string{"Enter"}, "stalled: Enter", o.Screen)
				a.LastMove = T(now)
			} else {
				e.restartOrFail(a, "stalled after nudges")
			}
		}
	}
}

func (e *Engine) setAgent(a *AgentRun, s string) {
	if a.State != s {
		a.State = s
		e.event("agent-"+s, a.Name, a.Task, "")
	}
}

// retry counts one more try of kind; false when the cap is used up.
func (e *Engine) retry(a *AgentRun, kind string) bool {
	a.Retries[kind]++
	return a.Retries[kind] <= e.G.Limits.RetryCap
}

func (e *Engine) send(a *AgentRun, text string) {
	e.event("send", a.Name, a.Task, text)
	if err := e.do(Action{Kind: "send", Agent: a.Name, Text: text}); err != nil {
		e.event("send-refused", a.Name, a.Task, err.Error())
	}
}

func (e *Engine) keys(a *AgentRun, keys []string, why, screen string) {
	e.event("answer", a.Name, a.Task, fmt.Sprintf("%s -> keys %v", why, keys), screen)
	if screen != "" {
		a.AnsHash, a.AnsAt = screenHash(screen), T(e.Now())
	}
	if err := e.do(Action{Kind: "keys", Agent: a.Name, Keys: keys}); err != nil {
		e.event("send-refused", a.Name, a.Task, err.Error())
	}
}

func (e *Engine) choose(a *AgentRun, screen string, s Sight) {
	if e.Decide != nil {
		keys, why, err := e.Decide(e.G, a, screen, s.Options)
		if err == nil && len(keys) > 0 {
			e.keys(a, keys, "decider: "+why, screen)
			return
		}
		e.event("decider-failed", a.Name, a.Task, fmt.Sprint(err))
	}
	// No decider: the CLI's highlighted default is the routine answer. Recorded as an assumption.
	e.G.Assumptions = append(e.G.Assumptions, fmt.Sprintf("%s: took the default of menu %q", a.Name, s.Line))
	e.keys(a, []string{"Enter"}, "menu default", screen)
}

func (e *Engine) escalate(a *AgentRun, problem, screen string) *Escalation {
	esc := &Escalation{ID: fmt.Sprintf("e%d", len(e.G.Escalations)+1), Agent: a.Name, Task: a.Task, Problem: problem, Screen: tailText(screen, 25), At: T(e.Now())}
	e.G.Escalations = append(e.G.Escalations, esc)
	e.event("escalated", a.Name, a.Task, problem, screen)
	return esc
}

// ownerBlock stops the agent's task for the owner (secret, outside trust). Other tasks keep running.
func (e *Engine) ownerBlock(a *AgentRun, problem, screen string) {
	esc := e.escalate(a, "OWNER: "+problem, screen)
	if t := e.G.Task(a.Task); t != nil {
		t.State, t.Note = TaskBlocked, esc.Problem
	}
	if a.Role == "lead" {
		e.setState(GoalNeedsOwner, "the lead "+problem)
	}
}

// restartOrFail restarts a broken agent in the same folder (its work is committed first). After the cap
// it switches profile; with none left the task attempt fails.
func (e *Engine) restartOrFail(a *AgentRun, why string) {
	if t := e.G.Task(a.Task); t != nil && a.Role == "write" {
		if saved, err := e.W.SaveWork(t, a.Name); err != nil {
			e.event("error", a.Name, a.Task, "save work: "+err.Error())
		} else if saved {
			e.event("work-saved", a.Name, a.Task, "committed what the agent left")
		}
	}
	if a.Begun && e.Now().Sub(a.Started.Time) > e.G.Limits.StartTimeout.D() {
		a.Retries["restart"] = 0 // it ran fine for a while: an outside death, not a crash loop
	}
	if !e.retry(a, "restart") {
		e.switchModel(a, why)
		return
	}
	e.endAgent(a, why)
	next := e.startAgent(a.Role, a.Task, a.Profile, a.Account, a.Retries, "Your previous run stopped ("+why+"). Its work is committed in this folder; continue from there.")
	if next != nil {
		e.event("restarted", next.Name, a.Task, why)
	}
}

// switchModel replaces the agent with its profile's fallback, or another profile of the same role.
func (e *Engine) switchModel(a *AgentRun, why string) {
	e.endAgent(a, why)
	p := e.G.Profile(a.Profile)
	alt := ""
	if p != nil && p.Fallback != "" && e.G.Profile(p.Fallback) != nil {
		alt = p.Fallback
	}
	if alt == "" {
		exclude := ""
		if t := e.G.Task(a.Task); t != nil && a.Role == "review" {
			exclude = e.kindOf(t.Writer)
		}
		alt = e.pickProfile(a.Role, exclude, a.Profile)
	}
	if alt == "" {
		e.taskAttemptFailed(a, "no other model to switch to after: "+why)
		return
	}
	e.event("model-switch", a.Name, a.Task, fmt.Sprintf("%s -> %s (%s)", a.Profile, alt, why))
	if t := e.G.Task(a.Task); t != nil {
		if a.Role == "write" {
			t.Writer = alt
		} else if a.Role == "review" {
			t.Reviewer = alt
		}
	}
	e.startAgent(a.Role, a.Task, alt, "", nil, "Another model stopped on this task ("+why+"); its work is committed in this folder.")
}

// accountLimited records the account's reset time and moves the agent to the next free account.
// All accounts limited: the goal waits and resumes by itself. Never the owner (owner's decision).
func (e *Engine) accountLimited(a *AgentRun, s Sight, screen string) {
	until := s.Reset
	if until.IsZero() {
		until = e.Now().Add(e.G.Limits.DefaultLimitFor.D())
	}
	key := a.Profile + "/" + a.Account
	acc := e.G.Accounts[key]
	if acc == nil {
		acc = &Account{}
		e.G.Accounts[key] = acc
	}
	acc.LimitedUntil = T(until)
	e.event("account-limited", a.Name, a.Task, fmt.Sprintf("%s limited until %s (%q)", key, until.Format(time.RFC3339), s.Line), screen)
	e.endAgent(a, "account limit")
	if prof, acct := e.freeAccount(a.Profile); prof != "" {
		e.startAgent(a.Role, a.Task, prof, acct, nil, "The previous account hit its limit. Work so far is committed in this folder; continue.")
		return
	}
	if alt := e.pickProfile(a.Role, e.reviewExclude(a), a.Profile); alt != "" {
		if _, acct := e.freeAccount(alt); acct != "" || len(e.G.Profile(alt).Accounts) == 0 {
			e.startAgent(a.Role, a.Task, alt, acct, nil, "The previous model's account hit its limit. Work so far is committed in this folder; continue.")
			return
		}
	}
	// Nothing free: wait for the earliest reset.
	earliest := until
	for _, ac := range e.G.Accounts {
		if ac.LimitedUntil.After(e.Now()) && ac.LimitedUntil.Before(earliest) {
			earliest = ac.LimitedUntil.Time
		}
	}
	if t := e.G.Task(a.Task); t != nil && t.State == TaskRunning || t != nil && t.State == TaskReview {
		t.Agent = "" // picked up again after the wait
	}
	e.G.Waiting = T(earliest)
	e.stopAll("all accounts limited")
	e.setState(GoalWaitingLimit, "all accounts limited; resuming at "+earliest.Format(time.RFC3339))
}

func (e *Engine) reviewExclude(a *AgentRun) string {
	if t := e.G.Task(a.Task); t != nil && a.Role == "review" {
		return e.kindOf(t.Writer)
	}
	return ""
}

func (e *Engine) accountFree(profile, account string) bool {
	acc := e.G.Accounts[profile+"/"+account]
	return acc == nil || !acc.LimitedUntil.After(e.Now())
}

// freeAccount returns the profile and its first account that is not limited ("" account when the
// profile has no account list and is not limited).
func (e *Engine) freeAccount(profile string) (string, string) {
	p := e.G.Profile(profile)
	if p == nil {
		return "", ""
	}
	if len(p.Accounts) == 0 {
		if e.accountFree(profile, "") {
			return profile, ""
		}
		return "", ""
	}
	for _, acct := range p.Accounts {
		if e.accountFree(profile, acct) {
			return profile, acct
		}
	}
	return "", ""
}

// handoff: the lead's context is nearly full. It writes handoff.md, then a fresh lead continues from it.
func (e *Engine) handoff(a *AgentRun) {
	if a.Retries["handoff"] == 0 {
		a.Retries["handoff"]++
		e.send(a, fmt.Sprintf("Your context is nearly full. Write everything the next lead needs to %s now (decisions, open questions, next step), then stop.", e.L.path("handoff.md")))
		return
	}
	if !e.retry(a, "handoff-wait") {
		e.endAgent(a, "handoff")
		e.startAgent("lead", "", a.Profile, a.Account, nil, "You replace a lead whose context was full. Read "+e.L.path("handoff.md")+" first.")
	}
}

func (e *Engine) kindOf(profile string) string {
	if p := e.G.Profile(profile); p != nil {
		return p.Kind
	}
	return ""
}

func screenHash(s string) string { return ProgressHash(s+"\n\n\n") + fmt.Sprint(len(s)) }

func plural(n int, s string) string {
	if n == 1 {
		return "1 " + s
	}
	return fmt.Sprintf("%d %ss", n, s)
}

var _ = strings.TrimSpace
