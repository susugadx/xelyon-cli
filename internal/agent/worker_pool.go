package agent

import (
	"context"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// WorkerPool は Worker を管理するプール
type WorkerPool struct {
	workers       []*Worker
	maxWorkers    int
	sharedContext *SharedContext
	multiProgress *ui.MultiProgress

	// Channel
	commands chan WorkerCommand
	results  chan WorkerResult

	// 制御
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewWorkerPool は新しい WorkerPool を作成
func NewWorkerPool(
	maxWorkers int,
	provider api.Provider,
	model string,
	sharedCtx *SharedContext,
	agent *Agent,
	stepTimeout int,
) *WorkerPool {
	pool := &WorkerPool{
		maxWorkers:    maxWorkers,
		sharedContext: sharedCtx,
		multiProgress: ui.NewMultiProgress(),
		commands:      make(chan WorkerCommand, maxWorkers*2),
		results:       make(chan WorkerResult, maxWorkers*2),
	}

	// Worker 作成
	for i := 0; i < maxWorkers; i++ {
		w := NewWorker(i, provider, model, sharedCtx, agent, stepTimeout, pool.commands, pool.results)
		pool.workers = append(pool.workers, w)
	}

	return pool
}

// Start は Worker を起動
func (wp *WorkerPool) Start(ctx context.Context) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.ctx, wp.cancel = context.WithCancel(ctx)
	for _, w := range wp.workers {
		wp.wg.Add(1)
		go func(worker *Worker) {
			defer wp.wg.Done()
			worker.Run(wp.ctx)
		}(w)
	}
}

// Stop は Worker を停止
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	if wp.cancel != nil {
		wp.cancel()
	}
	wp.mu.Unlock()

	wp.wg.Wait()
}

// SubmitStep はステップ実行をサブミット
func (wp *WorkerPool) SubmitStep(step *plan.PlanStep, confirmLevel string) {
	wp.commands <- WorkerCommand{
		Type:         CmdExecuteStep,
		StepID:       step.ID,
		Step:         step,
		ConfirmLevel: confirmLevel,
	}
}

// SubmitInvestigation は調査をサブミット
func (wp *WorkerPool) SubmitInvestigation(query string) {
	wp.commands <- WorkerCommand{
		Type:  CmdInvestigate,
		Query: query,
	}
}

// CollectResults は指定数の結果を収集
func (wp *WorkerPool) CollectResults(expected int) []WorkerResult {
	results := make([]WorkerResult, 0, expected)
	for i := 0; i < expected; i++ {
		result := <-wp.results
		results = append(results, result)
	}
	return results
}

// ExecuteStepsWithUI はステップを UI 付きで並列実行
func (wp *WorkerPool) ExecuteStepsWithUI(ctx context.Context, steps []*plan.PlanStep, confirmLevel string) []WorkerResult {
	// MultiProgress 初期化
	wp.multiProgress.Clear()
	for _, step := range steps {
		wp.multiProgress.AddTask(step.ID, step.Description)
	}
	wp.multiProgress.StartRendering()
	defer wp.multiProgress.StopRendering()

	// ステップをサブミット
	for _, step := range steps {
		wp.multiProgress.Start(step.ID) // Running 状態
		wp.SubmitStep(step, confirmLevel)
	}

	// 結果を収集して UI 更新
	results := make([]WorkerResult, 0, len(steps))
	for i := 0; i < len(steps); i++ {
		result := <-wp.results
		results = append(results, result)

		// UI 更新
		if result.Success {
			wp.multiProgress.Done(result.StepID)
		} else {
			errMsg := ""
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			wp.multiProgress.Fail(result.StepID, errMsg)
		}

		// SharedContext 更新
		if result.Success {
			wp.sharedContext.MarkStepCompleted(result.StepID)
		}
	}

	return results
}

// ExecuteInvestigationsWithUI は調査を UI 付きで並列実行
func (wp *WorkerPool) ExecuteInvestigationsWithUI(ctx context.Context, queries []string) []WorkerResult {
	// MultiProgress 初期化
	wp.multiProgress.Clear()

	// クエリと taskID のマッピングを作成
	queryToTaskID := make(map[string]int)
	for i, query := range queries {
		// 調査クエリを短縮して表示
		displayQuery := query
		if len(displayQuery) > 40 {
			displayQuery = displayQuery[:37] + "..."
		}
		wp.multiProgress.AddTask(i+1, "Query "+displayQuery)
		queryToTaskID[query] = i + 1
	}
	wp.multiProgress.StartRendering()
	defer wp.multiProgress.StopRendering()

	// 調査をサブミット
	for i := range queries {
		wp.multiProgress.Start(i + 1) // Running 状態
		wp.SubmitInvestigation(queries[i])
	}

	// 結果を収集して UI 更新
	results := make([]WorkerResult, 0, len(queries))
	for i := 0; i < len(queries); i++ {
		result := <-wp.results
		results = append(results, result)

		// UI 更新（Query から taskID を取得）
		taskID, ok := queryToTaskID[result.Query]
		if !ok {
			// フォールバック: 収集順で更新
			taskID = i + 1
		}
		if result.Success {
			wp.multiProgress.Done(taskID)
		} else {
			errMsg := ""
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			wp.multiProgress.Fail(taskID, errMsg)
		}
	}

	return results
}

// GetActiveWorkerCount はアクティブな Worker 数を取得
func (wp *WorkerPool) GetActiveWorkerCount() int {
	count := 0
	for _, w := range wp.workers {
		if w.GetStatus() == WorkerRunning {
			count++
		}
	}
	return count
}

// GetWorkerStatuses は全 Worker のステータスを取得
func (wp *WorkerPool) GetWorkerStatuses() []WorkerStatus {
	statuses := make([]WorkerStatus, len(wp.workers))
	for i, w := range wp.workers {
		statuses[i] = w.GetStatus()
	}
	return statuses
}
