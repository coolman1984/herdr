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
 ("tasks.go", "ram >= 0 && ram < l.MinFreeRAMMB", "false", "TestMaxAgentsAndFreeMemoryGate"),
 ("tasks.go", "if owner := e.G.Locks[f]; owner != \"\" && owner != t.ID {", "if owner := e.G.Locks[f]; false && owner != \"\" && owner != t.ID {", "TestFileReservationsSerializeTasks"),
 ("tasks.go", 'r = e.pickProfile("review", e.kindOf(t.Writer), "")', 'r = e.pickProfile("review", "", "")', "TestDependenciesAndIndependentReviewToDone"),
 ("tasks.go", "if !j.ok {\n\t\te.event(\"acceptance-failed\"", "if false {\n\t\te.event(\"acceptance-failed\"", "TestDoneOnlyAfterTheSupervisorsOwnAcceptance"),
 ("tasks.go", "if e.W.Ahead(g, t) == 0 {", "if false {", "TestDoneWithoutCommitIsRejected"),
 ("git.go", "ok, out := RunChecks(dir, g.Tests, timeout)\n\tif !ok {", "ok, out := RunChecks(dir, g.Tests, timeout)\n\tif false && !ok {", "TestFailingProjectTestsUndoTheMerge"),
 ("git.go", "if files != \"\" {", "if false {", "TestMergeConflictGoesBackToWriterAndRevertKeepsOthers"),
 ("commands.go", 'if c.By == "lead" {', "if false {", "TestLeadCannotStopTheGoal"),
 ("engine.go", "if d := g.Limits.NoProgress.D(); d > 0 &&", "if d := g.Limits.NoProgress.D(); d < 0 &&", "TestNoProgressStops"),
 ("commands.go", 'return "", "", fmt.Errorf("%s is not a git repository', 'top = project; err = nil; if false { return "", "", fmt.Errorf("%s is not a git repository', None),
]
killed = 0
for f, old, new, test in M:
    if test is None:
        continue
    path = os.path.join(pkg, f)
    src = open(path, encoding="utf-8").read()
    if src.count(old) != 1:
        print(f"SKIP  {test}: pattern not unique in {f}: {old[:50]!r}"); continue
    open(path, "w", encoding="utf-8").write(src.replace(old, new))
    try:
        r = subprocess.run(["go", "test", "-count=1", "-run", "^" + test + "$", "./internal/goal/"], cwd=app, capture_output=True, text=True, timeout=300)
    finally:
        open(path, "w", encoding="utf-8").write(src)
    ok = r.returncode != 0
    killed += ok
    why = "fails" if ok else "STILL PASSES"
    if ok and ("build failed" in r.stdout or "[setup failed]" in r.stdout):
        why = "BUILD BROKE (not a real kill)"; killed -= 1
    print(f"{'KILL' if why == 'fails' else 'MISS'}  {test:52s} {f}: {old.splitlines()[0][:60]!r} -> {why}")
n = sum(1 for m in M if m[3])
print(f"{killed}/{n} mutations caught")
sys.exit(0 if killed == n else 1)
