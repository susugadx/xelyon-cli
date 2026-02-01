package agent

import (
	"context"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestNewWorkerPool(t *testing.T) {
	sharedCtx := NewSharedContext()

	pool := NewWorkerPool(3, nil, "test-model", sharedCtx, nil, 600)

	if pool == nil {
		t.Fatal("NewWorkerPool returned nil")
	}
	if pool.maxWorkers != 3 {
		t.Errorf("maxWorkers = %d, want 3", pool.maxWorkers)
	}
	if len(pool.workers) != 3 {
		t.Errorf("workers count = %d, want 3", len(pool.workers))
	}
	if pool.sharedContext != sharedCtx {
		t.Error("sharedContext not set correctly")
	}
	if pool.multiProgress == nil {
		t.Error("multiProgress should be initialized")
	}
	if pool.commands == nil {
		t.Error("commands channel should be initialized")
	}
	if pool.results == nil {
		t.Error("results channel should be initialized")
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(2, nil, "test", sharedCtx, nil, 600)

	ctx := context.Background()
	pool.Start(ctx)

	// Worker が起動していることを確認するため少し待機
	time.Sleep(10 * time.Millisecond)

	// 停止
	pool.Stop()

	// 停止後に再度 Stop を呼んでもパニックしないことを確認
	pool.Stop()
}

func TestWorkerPool_SubmitStep(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(1, nil, "test", sharedCtx, nil, 600)

	step := &plan.PlanStep{ID: 1, Description: "Test step"}

	// SubmitStep がブロックしないことを確認
	done := make(chan bool)
	go func() {
		pool.SubmitStep(step, "dangerous")
		done <- true
	}()

	select {
	case <-done:
		// 成功
	case <-time.After(100 * time.Millisecond):
		t.Error("SubmitStep blocked unexpectedly")
	}

	// チャンネルからコマンドを取り出して確認
	cmd := <-pool.commands
	if cmd.Type != CmdExecuteStep {
		t.Errorf("command type = %d, want CmdExecuteStep", cmd.Type)
	}
	if cmd.StepID != 1 {
		t.Errorf("command StepID = %d, want 1", cmd.StepID)
	}
	if cmd.ConfirmLevel != "dangerous" {
		t.Errorf("command ConfirmLevel = %s, want 'dangerous'", cmd.ConfirmLevel)
	}
}

func TestWorkerPool_SubmitInvestigation(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(1, nil, "test", sharedCtx, nil, 600)

	// SubmitInvestigation がブロックしないことを確認
	done := make(chan bool)
	go func() {
		pool.SubmitInvestigation("Test query")
		done <- true
	}()

	select {
	case <-done:
		// 成功
	case <-time.After(100 * time.Millisecond):
		t.Error("SubmitInvestigation blocked unexpectedly")
	}

	// チャンネルからコマンドを取り出して確認
	cmd := <-pool.commands
	if cmd.Type != CmdInvestigate {
		t.Errorf("command type = %d, want CmdInvestigate", cmd.Type)
	}
	if cmd.Query != "Test query" {
		t.Errorf("command Query = %s, want 'Test query'", cmd.Query)
	}
}

func TestWorkerPool_CollectResults(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(2, nil, "test", sharedCtx, nil, 600)

	// 結果をチャンネルに送信
	go func() {
		pool.results <- WorkerResult{WorkerID: 0, Success: true, StepID: 1}
		pool.results <- WorkerResult{WorkerID: 1, Success: true, StepID: 2}
	}()

	// 結果を収集
	results := pool.CollectResults(2)

	if len(results) != 2 {
		t.Errorf("results count = %d, want 2", len(results))
	}
}

func TestWorkerPool_GetActiveWorkerCount(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(3, nil, "test", sharedCtx, nil, 600)

	// 初期状態は全員 Idle
	count := pool.GetActiveWorkerCount()
	if count != 0 {
		t.Errorf("initial active count = %d, want 0", count)
	}

	// 1つの Worker を Running に設定
	pool.workers[0].mu.Lock()
	pool.workers[0].status = WorkerRunning
	pool.workers[0].mu.Unlock()

	count = pool.GetActiveWorkerCount()
	if count != 1 {
		t.Errorf("active count = %d, want 1", count)
	}

	// 2つ目も Running に設定
	pool.workers[1].mu.Lock()
	pool.workers[1].status = WorkerRunning
	pool.workers[1].mu.Unlock()

	count = pool.GetActiveWorkerCount()
	if count != 2 {
		t.Errorf("active count = %d, want 2", count)
	}
}

func TestWorkerPool_GetWorkerStatuses(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(3, nil, "test", sharedCtx, nil, 600)

	statuses := pool.GetWorkerStatuses()

	if len(statuses) != 3 {
		t.Errorf("statuses count = %d, want 3", len(statuses))
	}

	// 全員 Idle であることを確認
	for i, status := range statuses {
		if status != WorkerIdle {
			t.Errorf("worker %d status = %v, want WorkerIdle", i, status)
		}
	}

	// 1つの Worker を Running に設定
	pool.workers[1].mu.Lock()
	pool.workers[1].status = WorkerRunning
	pool.workers[1].mu.Unlock()

	statuses = pool.GetWorkerStatuses()
	if statuses[0] != WorkerIdle {
		t.Errorf("worker 0 status = %v, want WorkerIdle", statuses[0])
	}
	if statuses[1] != WorkerRunning {
		t.Errorf("worker 1 status = %v, want WorkerRunning", statuses[1])
	}
	if statuses[2] != WorkerIdle {
		t.Errorf("worker 2 status = %v, want WorkerIdle", statuses[2])
	}
}

func TestWorkerPool_ChannelBufferSize(t *testing.T) {
	sharedCtx := NewSharedContext()

	// maxWorkers = 5 の場合、バッファサイズは 5*2 = 10
	pool := NewWorkerPool(5, nil, "test", sharedCtx, nil, 600)

	// バッファサイズを確認するため、複数のコマンドを送信
	for i := 0; i < 10; i++ {
		select {
		case pool.commands <- WorkerCommand{Type: CmdStop}:
			// 成功
		case <-time.After(10 * time.Millisecond):
			if i < 10 {
				t.Errorf("commands channel blocked at %d, expected buffer size 10", i)
			}
		}
	}
}

func TestWorkerPool_MultipleWorkers(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(5, nil, "test", sharedCtx, nil, 600)

	// 5つの Worker が作成されていることを確認
	if len(pool.workers) != 5 {
		t.Errorf("workers count = %d, want 5", len(pool.workers))
	}

	// 各 Worker の ID を確認
	for i, w := range pool.workers {
		if w.id != i {
			t.Errorf("worker %d id = %d, want %d", i, w.id, i)
		}
	}
}

func TestWorkerPool_SharedContextPropagation(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(3, nil, "test", sharedCtx, nil, 600)

	// 全 Worker が同じ SharedContext を持っていることを確認
	for i, w := range pool.workers {
		if w.sharedContext != sharedCtx {
			t.Errorf("worker %d has different sharedContext", i)
		}
	}
}

func TestWorkerPool_ConcurrentStartStop(t *testing.T) {
	sharedCtx := NewSharedContext()
	pool := NewWorkerPool(3, nil, "test", sharedCtx, nil, 600)

	// 複数回の Start/Stop がパニックしないことを確認
	for i := 0; i < 5; i++ {
		ctx := context.Background()
		pool.Start(ctx)
		time.Sleep(5 * time.Millisecond)
		pool.Stop()
	}
}
