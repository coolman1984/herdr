package deck

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type Job struct {
	ID      string    `json:"id"`
	OK      bool      `json:"ok"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Output  string    `json:"output"`
	Started time.Time `json:"started"`
	cancel  context.CancelFunc
}

func (a *App) closeJobs() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, job := range a.jobs {
		if job.cancel != nil {
			job.cancel()
		}
	}
}
func (a *App) startJob(script string, params map[string]any, timeout time.Duration, cleanup func()) (map[string]any, error) {
	runner, ok := a.herdr.(ScriptRunner)
	if !ok {
		return nil, fmt.Errorf("script runner unavailable")
	}
	plan, err := a.scriptLaunchPlan(script, params)
	if err != nil {
		return nil, err
	}
	ticket, err := a.admit(context.Background(), plan)
	if err != nil {
		return nil, err
	}
	admitted := false
	defer func() {
		if !admitted {
			ticket.finish("", fmt.Errorf("job not started"))
		}
	}()
	idBytes := make([]byte, 10)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(idBytes)
	a.mu.Lock()
	active := 0
	for key, j := range a.jobs {
		if j.Status == "running" {
			active++
		} else if len(a.jobs) > 24 {
			delete(a.jobs, key)
		}
	}
	if active >= 4 {
		a.mu.Unlock()
		return nil, fmt.Errorf("four toolkit jobs already running; wait for one to finish")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	job := &Job{ID: id, Status: "running", Message: "Started " + script, Started: time.Now(), cancel: cancel}
	a.jobs[id] = job
	a.mu.Unlock()
	admitted = true
	go func() {
		defer cancel()
		if cleanup != nil {
			defer cleanup()
		}
		out, err := runner.RunScript(ctx, script, params, timeout)
		ticket.finish(out, err)
		message := out
		if err != nil {
			message = err.Error()
		}
		if len(message) > 256<<10 {
			message = message[len(message)-(256<<10):]
		}
		if len(out) > 256<<10 {
			out = out[len(out)-(256<<10):]
		}
		a.mu.Lock()
		job.OK = err == nil
		job.Status = "done"
		if err != nil {
			job.Status = "failed"
		}
		job.Message = message
		job.Output = out
		job.cancel = nil
		msg := event("result", job)
		for ch := range a.clients {
			a.queueLocked(ch, msg)
		}
		a.mu.Unlock()
	}()
	return map[string]any{"ok": true, "id": id, "job_id": id, "message": "Started " + script}, nil
}
