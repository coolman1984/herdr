package goal

import (
 "encoding/json"
 "testing"
)

// These regression specifications are intentionally NOT pre-certified.
// Run against cd846676 first, record failures, then fix behavior without weakening them.

func TestIndependentAuditExitedAgentGetsNoInput(t *testing.T) {
 r:=newRig(t,task("T1")); r.tick()
 a:=r.agentOf("T1"); f:=r.b.agents[a.Name]; f.exists=false
 err:=r.e.perform(Action{Kind:"send",Agent:a.Name,Text:"continue"})
 if err==nil || len(f.sent)>0 { t.Fatal("exited agent accepted input; shell must never receive agent instructions") }
}
func TestIndependentAuditBlockedReviewCannotApprove(t *testing.T) {
 r:=newRig(t,task("T1"))
 tt:=r.e.G.Task("T1"); tt.State=TaskReview; tt.Writer="alpha"; tt.Reviewer="beta"
 a:=&AgentRun{Name:"review",Role:"review",Task:tt.ID,Profile:"beta",Retries:map[string]int{}}
 r.e.G.Agents[a.Name]=a; r.b.agents[a.Name]=&fakeAgent{exists:true,status:StatusIdle}
 r.e.finished(a,&Answer{Status:"blocked",Verdict:"pass",Summary:"conflicting result"})
 if tt.State==TaskMerging || tt.State==TaskDone { t.Fatal("non-done review approved work") }
}
func TestIndependentAuditCancelledPrerequisiteIsNotSuccess(t *testing.T) {
 r:=newRig(t,task("T1"),task("T2","T1"))
 r.e.G.Task("T1").State=TaskCancelled
 if r.e.depsDone(r.e.G.Task("T2")) { t.Fatal("cancelled prerequisite unlocked dependent work") }
}
func TestIndependentAuditDirectoryOwnershipIncludesChildren(t *testing.T) {
 r:=newRig(t,task("T1"),task("T2"))
 r.e.G.Locks["src"]="T1"; r.e.G.Task("T2").Files=[]string{"src/service.go"}
 if r.e.locksFree(r.e.G.Task("T2")) { t.Fatal("child path bypassed directory ownership") }
}
func TestIndependentAuditConsumerRejectsDelegateStop(t *testing.T) {
 r:=newRig(t,task("T1"))
 err:=r.e.apply(Command{Cmd:"stop",By:"write",Args:json.RawMessage("{}")})
 if err==nil { t.Fatal("inbox consumer accepted a delegate's owner-only command") }
}
func TestIndependentAuditInvalidUpdateLeavesWorkerRunning(t *testing.T) {
 r:=newRig(t,task("T1")); r.tick()
 a:=r.agentOf("T1")
 err:=r.e.apply(Command{Cmd:"task-update",By:"lead",Args:json.RawMessage("{\"id\":\"T1\",\"writer\":\"missing\"}")})
 if err==nil { t.Fatal("invalid update accepted") }
 if !r.b.agents[a.Name].exists || a.Ended { t.Fatal("rejected update stopped a valid running worker") }
}
