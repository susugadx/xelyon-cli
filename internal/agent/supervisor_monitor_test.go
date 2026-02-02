package agent

import (
	"context"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

type noopWorkerPool struct{}

func (n *noopWorkerPool) SubmitStep(step *plan.PlanStep, confirmLevel string) {}

func TestSupervisor_Monitor_RecordsCompletedAndDetectsDuplicate(t *testing.T) {
	s := &Supervisor{
		maxWorkers:      2,
		confirmLevel:    "auto",
		sharedContext:   NewSharedContext(),
		retryCount:      make(map[int]int),
		completedByStep: make(map[int]WorkerMessage),
		failedByStep:    make(map[int]WorkerMessage),
	}

	// workerPool は SubmitStep だけ使うので最低限のダミーを差し込む
	// （監視ループの重複完了時の介入で呼ばれる）
	// NOTE: Supervisor.workerPool は *WorkerPool 型のため、nil を避けて実行経路を制限する
	// 重複完了の分岐で step が nil の場合 SubmitStep は呼ばれないので、ここでは plan 側に step を用意しない。

	p := &plan.Plan{Steps: []plan.PlanStep{{ID: 1, Description: "s1"}}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	go func() {
		close(ready)
		s.runMonitor(ctx, p)
	}()
	<-ready

	s.sharedContext.MarkStepCompleted(1)
	s.sharedContext.Publish(WorkerMessage{FromWorker: 1, Topic: "step_completed", StepID: 1, Content: "done"})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := s.completedByStep[1]; ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := s.completedByStep[1]; !ok {
		t.Fatal("expected completedByStep to be updated")
	}

	// 重複完了を publish して競合検出分岐に入ること（マップは初回のまま保持される）
	s.sharedContext.Publish(WorkerMessage{FromWorker: 2, Topic: "step_completed", StepID: 1, Content: "done-again"})

	// 競合検出自体はログ出力なので、ここでは状態が壊れないことを確認
	time.Sleep(50 * time.Millisecond)
	got := s.completedByStep[1]
	if got.FromWorker != 1 {
		t.Fatalf("expected first completion to remain recorded, got worker %d", got.FromWorker)
	}
}

func TestSupervisor_Monitor_RecordsFailed(t *testing.T) {
	s := &Supervisor{
		maxWorkers:      2,
		confirmLevel:    "auto",
		sharedContext:   NewSharedContext(),
		retryCount:      make(map[int]int),
		completedByStep: make(map[int]WorkerMessage),
		failedByStep:    make(map[int]WorkerMessage),
	}

	p := &plan.Plan{Steps: []plan.PlanStep{{ID: 1, Description: "s1"}}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	go func() {
		close(ready)
		s.runMonitor(ctx, p)
	}()
	<-ready

	s.sharedContext.Publish(WorkerMessage{FromWorker: 1, Topic: "step_failed", StepID: 1, Content: "boom"})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := s.failedByStep[1]; ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := s.failedByStep[1]; !ok {
		t.Fatal("expected failedByStep to be updated")
	}
}
