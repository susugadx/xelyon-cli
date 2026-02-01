package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// Worker は並列実行用のワーカー
// UI 操作なし、Supervisor への報告のみ
type Worker struct {
	id            int
	provider      api.Provider
	model         string
	sharedContext *SharedContext
	agent         *Agent
	stepTimeout   int // タイムアウト（秒）

	// Channel 通信
	commands chan WorkerCommand
	results  chan WorkerResult

	// 状態
	status      WorkerStatus
	currentStep *plan.PlanStep
	mu          sync.Mutex
}

// NewWorker は新しい Worker を作成
func NewWorker(
	id int,
	provider api.Provider,
	model string,
	sharedCtx *SharedContext,
	agent *Agent,
	stepTimeout int,
	commands chan WorkerCommand,
	results chan WorkerResult,
) *Worker {
	return &Worker{
		id:            id,
		provider:      provider,
		model:         model,
		sharedContext: sharedCtx,
		agent:         agent,
		stepTimeout:   stepTimeout,
		commands:      commands,
		results:       results,
		status:        WorkerIdle,
	}
}

// Run は Worker のメインループ（goroutine で実行）
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-w.commands:
			result := w.executeCommand(ctx, cmd)
			w.results <- result
		}
	}
}

// executeCommand はコマンドを実行
func (w *Worker) executeCommand(ctx context.Context, cmd WorkerCommand) WorkerResult {
	w.mu.Lock()
	w.status = WorkerRunning
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.status = WorkerIdle
		w.currentStep = nil
		w.mu.Unlock()
	}()

	switch cmd.Type {
	case CmdExecuteStep:
		w.mu.Lock()
		w.currentStep = cmd.Step
		w.mu.Unlock()
		return w.executeStep(ctx, cmd.Step, cmd.ConfirmLevel)
	case CmdInvestigate:
		return w.executeInvestigation(ctx, cmd.Query)
	case CmdStop:
		return WorkerResult{WorkerID: w.id, Success: true}
	}
	return WorkerResult{WorkerID: w.id, Error: errors.New("unknown command")}
}

// executeStep はステップを1回だけ実行（リトライは Supervisor が判断）
func (w *Worker) executeStep(ctx context.Context, step *plan.PlanStep, confirmLevel string) WorkerResult {
	startTime := time.Now()

	// タイムアウト付きコンテキスト
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(w.stepTimeout)*time.Second)
	defer cancel()

	// SharedContext から依存ステップの結果を取得
	contextInfo := w.sharedContext.GetContextForStep(step)

	// 履歴構築
	history := w.buildWorkerHistory(step, contextInfo)

	// ツール実行ループ（最大50回）
	var toolsExecuted []string
	maxIterations := config.PlanMaxIterations

	for i := 0; i < maxIterations; i++ {
		// タイムアウトチェック
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return WorkerResult{
				WorkerID:      w.id,
				StepID:        step.ID,
				Success:       false,
				Error:         &StepTimeout{StepID: step.ID, Timeout: w.stepTimeout},
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
			}
		}

		response, err := w.provider.ChatWithTools(timeoutCtx, w.agent.SystemPrompt, history, w.model)
		if err != nil {
			// API エラー → 即座に報告
			return WorkerResult{
				WorkerID:      w.id,
				StepID:        step.ID,
				Success:       false,
				Error:         err,
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
			}
		}

		history = append(history, api.Message{Role: "assistant", Content: response})

		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// ツール呼び出しなし = ステップ完了
			w.sharedContext.AddStepResult(step.ID, response)
			return WorkerResult{
				WorkerID:      w.id,
				StepID:        step.ID,
				Success:       true,
				Output:        response,
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
			}
		}

		// ツール実行
		var allResults []string
		for _, tc := range toolCalls {
			// confirm_level チェック
			if ShouldConfirmTool(tc.Tool, confirmLevel) {
				// 並列時は確認不可 → エスカレーション要求
				return WorkerResult{
					WorkerID:         w.id,
					StepID:           step.ID,
					Success:          false,
					NeedsEscalation:  true,
					EscalationReason: fmt.Sprintf("tool %s requires confirmation", tc.Tool),
					ToolsExecuted:    toolsExecuted,
					Duration:         time.Since(startTime),
				}
			}

			result, change := tools.Execute(tc)
			toolsExecuted = append(toolsExecuted, tc.Tool)
			allResults = append(allResults, fmt.Sprintf("[%s]\n%s", tc.Tool, result))

			// 変更を SharedContext に記録
			if change != nil {
				w.sharedContext.AddFileChange(step.ID, *change)
			}

			// 失敗検出 → 即座に報告（リトライは Supervisor が判断）
			if failed, reason := plan.ContainsFailure(result); failed {
				return WorkerResult{
					WorkerID:      w.id,
					StepID:        step.ID,
					Success:       false,
					Error:         &StepFailure{StepID: step.ID, Reason: reason},
					ToolsExecuted: toolsExecuted,
					Duration:      time.Since(startTime),
				}
			}
		}

		// 結果を履歴に追加
		history = append(history, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Results]\n%s", strings.Join(allResults, "\n\n")),
		})
	}

	// イテレーション上限到達
	return WorkerResult{
		WorkerID:      w.id,
		StepID:        step.ID,
		Success:       false,
		Error:         errors.New("max iterations reached"),
		ToolsExecuted: toolsExecuted,
		Duration:      time.Since(startTime),
	}
}

// executeInvestigation は調査を実行
func (w *Worker) executeInvestigation(ctx context.Context, query string) WorkerResult {
	startTime := time.Now()

	// タイムアウト付きコンテキスト
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(w.stepTimeout)*time.Second)
	defer cancel()

	// 調査用の履歴を構築
	history := []api.Message{
		{Role: "user", Content: fmt.Sprintf("Investigation query: %s\n\nUse read-only tools (read_file, search_code, search_file, list_dir) to investigate.", query)},
	}

	// ツール実行ループ（最大20回）
	var toolsExecuted []string
	maxIterations := 20

	for i := 0; i < maxIterations; i++ {
		// タイムアウトチェック
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return WorkerResult{
				WorkerID:      w.id,
				Success:       false,
				Error:         &StepTimeout{StepID: 0, Timeout: w.stepTimeout},
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
				Query:         query,
			}
		}

		response, err := w.provider.ChatWithTools(timeoutCtx, w.agent.SystemPrompt, history, w.model)
		if err != nil {
			return WorkerResult{
				WorkerID:      w.id,
				Success:       false,
				Error:         err,
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
				Query:         query,
			}
		}

		history = append(history, api.Message{Role: "assistant", Content: response})

		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// 調査完了
			w.sharedContext.AddInvestigationResult(query, response)
			return WorkerResult{
				WorkerID:      w.id,
				Success:       true,
				Output:        response,
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
				Query:         query,
			}
		}

		// 読み取り専用ツールのみ実行
		var allResults []string
		for _, tc := range toolCalls {
			// SafetyHigh のみ許可
			if tools.GetToolSafety(tc.Tool) != tools.SafetyHigh {
				// 読み取り以外のツールはスキップ
				allResults = append(allResults, fmt.Sprintf("[%s] Skipped: only read-only tools allowed in investigation", tc.Tool))
				continue
			}

			result, _ := tools.Execute(tc)
			toolsExecuted = append(toolsExecuted, tc.Tool)
			allResults = append(allResults, fmt.Sprintf("[%s]\n%s", tc.Tool, result))
		}

		history = append(history, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Results]\n%s", strings.Join(allResults, "\n\n")),
		})
	}

	// イテレーション上限到達
	return WorkerResult{
		WorkerID:      w.id,
		Success:       false,
		Error:         errors.New("max iterations reached in investigation"),
		ToolsExecuted: toolsExecuted,
		Duration:      time.Since(startTime),
		Query:         query,
	}
}

// buildWorkerHistory は Worker 用の履歴を構築
func (w *Worker) buildWorkerHistory(step *plan.PlanStep, contextInfo string) []api.Message {
	// ステップ実行プロンプトを構築
	stepPrompt := promptplan.BuildStepPrompt(step.ID, step.Description, step.Tools)

	// 依存ステップのコンテキストがある場合は追加
	var userContent string
	if contextInfo != "" {
		userContent = fmt.Sprintf("Previous step results:\n%s\n\n%s", contextInfo, stepPrompt)
	} else {
		userContent = stepPrompt
	}

	return []api.Message{
		{Role: "user", Content: userContent},
	}
}

// GetStatus は Worker の状態を取得
func (w *Worker) GetStatus() WorkerStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

// GetCurrentStep は現在実行中のステップを取得
func (w *Worker) GetCurrentStep() *plan.PlanStep {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentStep
}
