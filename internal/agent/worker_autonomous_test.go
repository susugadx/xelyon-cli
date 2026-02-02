package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestWorker_Autonomous_StartsWhenDependenciesCompleted(t *testing.T) {
	sharedCtx := NewSharedContext()
	commands := make(chan WorkerCommand, 10)
	results := make(chan WorkerResult, 10)

	w := NewWorker(0, nil, "test", sharedCtx, nil, 600, commands, results)

	dep := &plan.PlanStep{ID: 1, Description: "dep"}
	next := &plan.PlanStep{ID: 2, Description: "next", DependsOn: []int{1}}
	w.RegisterSteps([]*plan.PlanStep{dep, next})

	var mu sync.Mutex
	started := make([]int, 0, 1)

	w.executeStepFunc = func(ctx context.Context, step *plan.PlanStep, confirmLevel string) WorkerResult {
		mu.Lock()
		started = append(started, step.ID)
		mu.Unlock()
		return WorkerResult{WorkerID: 0, StepID: step.ID, Success: true, Output: "ok"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	go func() {
		close(ready)
		w.Run(ctx)
	}()
	<-ready
	sharedCtx.MarkStepCompleted(1)
	sharedCtx.Publish(WorkerMessage{FromWorker: 99, Topic: "step_completed", StepID: 1, Content: "done"})

	select {
	case res := <-results:
		if !res.Success || res.StepID != 2 {
			t.Fatalf("unexpected result: %+v", res)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for autonomous step execution")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(started) != 1 || started[0] != 2 {
		t.Fatalf("expected step 2 to start once, got: %+v", started)
	}
}

func TestWorker_Autonomous_DoesNotStartWhenDependenciesIncomplete(t *testing.T) {
	sharedCtx := NewSharedContext()
	commands := make(chan WorkerCommand, 10)
	results := make(chan WorkerResult, 10)

	w := NewWorker(0, nil, "test", sharedCtx, nil, 600, commands, results)

	dep := &plan.PlanStep{ID: 1, Description: "dep"}
	next := &plan.PlanStep{ID: 2, Description: "next", DependsOn: []int{1}}
	w.RegisterSteps([]*plan.PlanStep{dep, next})

	startedCh := make(chan int, 1)
	w.executeStepFunc = func(ctx context.Context, step *plan.PlanStep, confirmLevel string) WorkerResult {
		startedCh <- step.ID
		return WorkerResult{WorkerID: 0, StepID: step.ID, Success: true, Output: "ok"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	go func() {
		close(ready)
		w.Run(ctx)
	}()
	<-ready

	// 依存未完了の状態で完了通知だけ流す
	sharedCtx.Publish(WorkerMessage{FromWorker: 99, Topic: "step_completed", StepID: 999, Content: "done"})

	select {
	case got := <-startedCh:
		t.Fatalf("expected no step to start, but started step %d", got)
	case <-time.After(300 * time.Millisecond):
		// OK: 何も開始されない
	}

	select {
	case res := <-results:
		t.Fatalf("expected no results, but got: %+v", res)
	case <-time.After(100 * time.Millisecond):
		// OK
	}
}
