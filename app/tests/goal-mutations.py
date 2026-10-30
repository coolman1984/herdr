#!/usr/bin/env python3
"""Goal engine mutation check: break one rule at a time, its test must fail. Restores every file.
   python3 app/tests/goal-mutations.py      (from anywhere; needs go)"""
import os, subprocess, sys
app = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
pkg = os.path.join(app, "internal", "goal")
M = [  # file, old, new, test
 ("screen.go", "if p.Footer && !footer {", "if p.Footer && !footer && len(lines) < 0 {", "TestTrustNeedsMenuFooter"),
 ("screen.go", "if yesRe.MatchString(o.Text) &&", "if !yesRe.MatchString(o.Text) &&", "TestTrustKeysPickYesWhenNoIsHighlighted"),
 ("screen.go", "func ProgressHash(screen string) string {\n", "func ProgressHash(screen string) string {\n\treturn fmt.Sprint(len(screen), screen)\n", "TestProgressHashIgnoresTimerAndSpinner"),
 ("engine.go", 'if s := e.P.Classify(o.Screen, e.Now()); s.Kind == "secret" {', 'if s := e.P.Classify(o.Screen, e.Now()); s.Kind == "never" {', "TestNothingIsTypedIntoASecretScreen"),
 ("commands.go", 'if strings.HasPrefix(esc.Problem, "OWNER:") && c.By != "owner" {', "if false {", "TestNothingIsTypedIntoASecretScreen"),
 ("engine.go", "return err == nil && rel != \"..\"", "return err == nil || rel != \"..\"", "TestTrustOutsideProjectGoesToOwner"),
 ("watchdog.go", 'e.restartOrFail(a, "agent died")', "", "TestDeadAgentRestartsAndKeepsWork"),
 ("watchdog.go", "if o.Err != nil {\n\t\t// Herdr", "if false {\n\t\t// Herdr", "TestBackendErrorIsNotDeath"),
 ("watchdog.go", 'if now.Sub(a.IdleSince.Time) < e.G.Limits.IdleGrace.D() || a.Role == "lead" {', "if true {", "TestIdleWithoutResultIsNotDone"),
 ("watchdog.go", "if !moved && now.Sub(a.LastMove.Time) > e.G.Limits.StallAfter.D() {", "if false {", "TestStalledAgentIsNudged"),
 ("watchdog.go", "if now.Sub(a.Started.Time) > e.G.Limits.StartTimeout.D() {", "if false {", "TestNotStartedResendsBrief"),
 ("watchdog.go", 'if e.retry(a, sight.Kind) && sight.Kind == "capacity" {', "if true {", "TestCapacityContinuesThenSwitchesModel"),
 ("watchdog.go", "if prof, acct := e.freeAccount(a.Profile); prof != \"\" {", "if prof, acct := e.freeAccount(a.Profile); false && prof != \"\" {", "TestAccountLimitsRotateThenWaitThenResume"),
 ("engine.go", "if e.Now().Before(g.Waiting.Time) {", "if true {", "TestAccountLimitsRotateThenWaitThenResume"),
 ("engine.go", "if o := e.B.Observe(a.Agent); o.Exists {", "if false {", "TestRecoverNeitherLosesNorDoublesAStart"),
 ("tasks.go", "ram >= 0 && ram < l.MinFreeRAMMB+l.AgentRAMMB", "false && ram < 0", "TestMaxAgentsAndFreeMemoryGate"),
 ("tasks.go", "if owner != t.ID && pathsOverlap(f, held) {", "if false && owner != t.ID && pathsOverlap(f, held) {", "TestFileReservationsSerializeTasks"),
 ("tasks.go", 'r = e.pickProfile("review", e.kindOf(t.Writer), "")', 'r = e.pickProfile("review", "", "")', "TestDependenciesAndIndependentReviewToDone"),
 ("tasks.go", "if !j.ok {\n\t\te.event(\"acceptance-failed\"", "if false {\n\t\te.event(\"acceptance-failed\"", "TestDoneOnlyAfterTheSupervisorsOwnAcceptance"),
 ("tasks.go", "if e.W.Ahead(g, t) == 0 {", "if false {", "TestDoneWithoutCommitIsRejected"),
 ("git.go", "if !ok || ctx.Err() != nil {\n\t\tgit(dir, \"reset\"", "if false && !ok || ctx.Err() != nil {\n\t\tgit(dir, \"reset\"", "TestFailingProjectTestsUndoTheMerge"),
 ("git.go", "if files != \"\" {", "if false {", "TestMergeConflictGoesBackToWriterAndRevertKeepsOthers"),
 ("commands.go", "if !authorized(c.By, c.Cmd) {", "if false {", "TestLeadCannotStopTheGoal"),
 ("commands.go", "if !authorized(c.By, c.Cmd) {", "if false {", "TestIndependentAuditConsumerRejectsDelegateStop"),
 ("engine.go", "if !o.Exists || o.Status == StatusExited {\n\t\t\t// The CLI is gone", "if false {\n\t\t\t// The CLI is gone", "TestIndependentAuditExitedAgentGetsNoInput"),
 ("engine.go", "if ag := e.G.Agents[a.Agent]; ag == nil || ag.Ended {", "if false {", None),
 ("tasks.go", 'if ans.Status != "done" || ans.Verdict != "pass" && ans.Verdict != "fail" {', "if false {", "TestIndependentAuditBlockedReviewCannotApprove"),
 ("tasks.go", "if dt == nil || dt.State != TaskDone {", "if dt == nil || dt.State != TaskDone && dt.State != TaskCancelled {", "TestIndependentAuditCancelledPrerequisiteIsNotSuccess"),
 ("tasks.go", "return a == b || a == \"\" || b == \"\" || strings.HasPrefix(b, a+\"/\") || strings.HasPrefix(a, b+\"/\")", "return a == b", "TestIndependentAuditDirectoryOwnershipIncludesChildren"),
 ("commands.go", "if a.Writer != \"\" && g.Profile(a.Writer) == nil {\n\t\t\treturn fmt.Errorf(\"unknown profile %s\", a.Writer)\n\t\t}\n\t\tif a.Deps", "if a.Deps", "TestIndependentAuditInvalidUpdateLeavesWorkerRunning"),
 ("commands.go", "c.By = e.G.Tokens[tokenHash(c.Token)]", "c.By = claimed", "TestInboxRoleComesFromTokenNotClaim"),
 ("engine.go", "if disk, err := e.L.Load(); err == nil && disk.RunnerGen > e.Gen {", "if disk, err := e.L.Load(); err == nil && disk.RunnerGen > e.Gen+100 {", "TestOldRunnerGenerationCannotWrite"),
 ("tasks.go", "if o.main && !o.done && k != key {", "if o.main && !o.done && k != key && false {", "TestIntegrationWorktreeHasOneOwner"),
 ("runproc.go", "killTree(cmd.Process.Pid)", "cmd.Process.Kill()", "TestCancelAndTimeoutKillTheProcessTree"),
 ("tasks.go", "if now := e.W.BranchHead(t); now != j.sha {", "if now := e.W.BranchHead(t); false && now != j.sha {", "TestCheckOfAnOlderCommitIsDropped"),
 ("engine.go", "if progressKinds[kind] {", "if true {", "TestRestartLoopIsNotProgress"),
 ("tasks.go", 'g.LeadNews = ""   // a new lead has been told nothing yet', "", "TestNewLeadIsToldAboutOpenProblems"),
 ("git.go", "if f == \"\" || autosaveSkip.MatchString(f) {", "if f == \"\" {", "TestAutosaveSkipsCachesSecretsAndHugeFiles"),
 ("tasks.go", "if t.Total >= 3*g.Limits.MaxAttempts {", "if false {", "TestFailureBudgetSurvivesReplans"),
 ("watchdog.go", "if !insideProject(e.G.Project, asked) {", "if !insideProject(e.G.Project, a.Cwd) {", "TestTrustUsesTheFolderOnScreen"),
 ("watchdog.go", "if riskyOption.MatchString(o.Text) {", "if false {", "TestRiskyMenuIsNotAnsweredByDefault"),
 ("engine.go", "if o := e.B.Observe(a.Name); o.Err == nil && o.Exists && o.Status != StatusExited {", "if o := e.B.Observe(a.Name); false && o.Err == nil {", "TestUnconfirmedStopKeepsOwnership"),
 ("watchdog.go", "if e.liveAgents(\"\") == 0 {\n\t\te.setState(GoalWaitingLimit", "if true {\n\t\te.stopAll(\"x\")\n\t\te.setState(GoalWaitingLimit", "TestAccountLimitDoesNotStopHealthyAgents"),
 ("tasks.go", "if procs := e.liveAgents(\"\") + e.liveAgents(\"lead\") + e.runningJobs(); l.MaxProcs > 0 && procs >= l.MaxProcs {", "if procs := e.liveAgents(\"\"); l.MaxProcs > 0 && procs >= l.MaxProcs {", "TestProcessBudgetCountsLeadAndJobs"),
 ("watchdog.go", "if fp := worktreeFP(a.Cwd); fp != \"\" && fp != a.LastFP {", "if fp := worktreeFP(a.Cwd); false && fp != a.LastFP {", "TestWorktreeChangesAreProgress"),
 ("engine.go", "if d := g.Limits.NoProgress.D(); d > 0 &&", "if d := g.Limits.NoProgress.D(); d < 0 &&", "TestNoProgressStops"),
 ("commands.go", 'return "", "", fmt.Errorf("%s is not a git repository', 'top = project; err = nil; if false { return "", "", fmt.Errorf("%s is not a git repository', None),
]
killed = 0
for f, old, new, test in M:
    if test is None:
        continue
    files, olds, news = (f, old, new) if isinstance(f, list) else ([f], [old], [new])
    saved, ok_pat = {}, True
    for ff, o in zip(files, olds):
        path = os.path.join(pkg, ff)
        src = saved.get(path) or open(path, encoding="utf-8").read()
        saved[path] = src
        if src.count(o) != 1:
            print(f"SKIP  {test}: pattern not unique in {ff}: {o[:50]!r}"); ok_pat = False
    if not ok_pat:
        continue
    try:
        for ff, o, nw in zip(files, olds, news):
            path = os.path.join(pkg, ff)
            open(path, "w", encoding="utf-8").write(saved[path].replace(o, nw))
        r = subprocess.run(["go", "test", "-count=1", "-run", "^" + test + "$", "./internal/goal/"], cwd=app, capture_output=True, text=True, timeout=300)
    finally:
        for path, src in saved.items():
            open(path, "w", encoding="utf-8").write(src)
    f, old = files[0], olds[0]
    ok = r.returncode != 0
    killed += ok
    why = "fails" if ok else "STILL PASSES"
    if ok and ("build failed" in r.stdout or "[setup failed]" in r.stdout):
        why = "BUILD BROKE (not a real kill)"; killed -= 1
    print(f"{'KILL' if why == 'fails' else 'MISS'}  {test:52s} {f}: {old.splitlines()[0][:60]!r} -> {why}")
n = sum(1 for m in M if m[3])
print(f"{killed}/{n} mutations caught")
sys.exit(0 if killed == n else 1)
