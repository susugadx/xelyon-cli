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
//
// 自律実行（依存解決で自動開始）を行う場合は、Worker 自身が SharedContext の Pub/Sub を購読し、
// 依存関係が満たされたステップを自動で実行する。
// 重複実行を防ぐため、inFlightSteps（実行中）を Worker 内で管理する。
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

	// 自律実行用状態
	inFlightSteps map[int]bool
	knownSteps    map[int]*plan.PlanStep

	// 実行関数（テスト用に差し替え可能）
	executeStepFunc func(ctx context.Context, step *plan.PlanStep, confirmLevel string) WorkerResult
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
	w := &Worker{
		id:            id,
		provider:      provider,
		model:         model,
		sharedContext: sharedCtx,
		agent:         agent,
		stepTimeout:   stepTimeout,
		commands:      commands,
		results:       results,
		status:        WorkerIdle,
		inFlightSteps: make(map[int]bool),
		knownSteps:    make(map[int]*plan.PlanStep),
	}
	w.executeStepFunc = w.executeStep
	return w
}

// RegisterSteps は Worker の自律実行対象ステップ一覧を登録する。
//
// 自律ループは RegisterSteps で渡されたステップ集合を対象に、
// SharedContext の完了通知（step_completed）をトリガーに依存関係を解決し、
// 実行可能になったステップを自動実行する。
func (w *Worker) RegisterSteps(steps []*plan.PlanStep) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, s := range steps {
		if s == nil {
			continue
		}
		w.knownSteps[s.ID] = s
	}
}

// tryStartStepIfReady は依存関係が満たされたステップを自動開始する。
//
// 重複実行を避けるため、inFlightSteps と SharedContext の完了状態でガードする。
func (w *Worker) tryStartStepIfReady(ctx context.Context, step *plan.PlanStep) {
	if step == nil {
		return
	}

	w.mu.Lock()
	// 既に完了している/実行中ならスキップ
	if w.sharedContext.IsStepCompleted(step.ID) || w.inFlightSteps[step.ID] {
		w.mu.Unlock()
		return
	}
	// 依存が未完了なら開始しない
	for _, depID := range step.DependsOn {
		if !w.sharedContext.IsStepCompleted(depID) {
			w.mu.Unlock()
			return
		}
	}
	w.inFlightSteps[step.ID] = true
	w.mu.Unlock()

	// 自律実行: 直接 executeStep を呼んで結果を results へ返す
	go func() {
		// confirmLevel は Supervisor と同等の設定値を使う
		result := w.executeStepFunc(ctx, step, config.GetGlobalConfig().PlanMode.ConfirmLevel)
		w.mu.Lock()
		delete(w.inFlightSteps, step.ID)
		w.mu.Unlock()
		w.results <- result
	}()
}

// Run は Worker のメインループ（goroutine で実行）
func (w *Worker) Run(ctx context.Context) {
	// Pub/Sub: ステップ完了/失敗通知を監視して、自律的に次ステップを起動する。
	stepCompletedCh := w.sharedContext.Subscribe("step_completed", 100)
	stepFailedCh := w.sharedContext.Subscribe("step_failed", 100)

	for {
		select {
		case <-ctx.Done():
			return

		case <-stepCompletedCh:
			// 完了通知を受けたら、全ステップを走査して実行可能になったものを起動
			w.mu.Lock()
			steps := make([]*plan.PlanStep, 0, len(w.knownSteps))
			for _, s := range w.knownSteps {
				steps = append(steps, s)
			}
			w.mu.Unlock()
			for _, s := range steps {
				w.tryStartStepIfReady(ctx, s)
			}

		case <-stepFailedCh:
			// 失敗通知自体では自動リトライしない（Supervisor が判断）。
			// ここではループ継続のみ。

		case cmd := <-w.commands:
			// Supervisor からの明示コマンドは従来どおり処理
			if cmd.Type == CmdExecuteStep && cmd.Step != nil {
				w.mu.Lock()
				w.knownSteps[cmd.Step.ID] = cmd.Step
				w.mu.Unlock()
			}
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

	// Worker は Thinking を無効化
	ctx = api.WithThinkingDisabled(ctx)

	// タイムアウト付きコンテキスト
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(w.stepTimeout)*time.Second)
	defer cancel()

	// SharedContext から依存ステップの結果を取得
	contextInfo := w.sharedContext.GetContextForStep(step)

	// 履歴構築
	history := w.buildWorkerHistory(step, contextInfo)

	// ツール実行ループ（最大80回）
	var toolsExecuted []string
	cfg := config.GetGlobalConfig()
	maxIterations := cfg.General.ToolLoopLimit

	for i := 0; i < maxIterations; i++ {
		// タイムアウトチェック
		if timeoutCtx.Err() == context.DeadlineExceeded {
			err := &StepTimeout{StepID: step.ID, Timeout: w.stepTimeout}
			w.sharedContext.Publish(WorkerMessage{
				FromWorker: w.id,
				Topic:      "step_failed",
				Content:    err.Error(),
				StepID:     step.ID,
			})
			return WorkerResult{
				WorkerID:      w.id,
				StepID:        step.ID,
				Success:       false,
				Error:         err,
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
			}
		}

		response, err := w.provider.ChatWithTools(timeoutCtx, w.agent.SystemPrompt, history, w.model)
		if err != nil {
			// API エラー → 即座に報告
			w.sharedContext.Publish(WorkerMessage{
				FromWorker: w.id,
				Topic:      "step_failed",
				Content:    err.Error(),
				StepID:     step.ID,
			})
			return WorkerResult{
				WorkerID:      w.id,
				StepID:        step.ID,
				Success:       false,
				Error:         err,
				ToolsExecuted: toolsExecuted,
				Duration:      time.Since(startTime),
			}
		}

		history = append(history, api.Message{
			Role:             "assistant",
			Content:          response,
			ReasoningContent: api.GetReasoningContent(w.provider),
		})

		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// ツール呼び出しなし = ステップ完了
			w.sharedContext.AddStepResult(step.ID, response)
			w.sharedContext.MarkStepCompleted(step.ID)
			w.sharedContext.Publish(WorkerMessage{
				FromWorker: w.id,
				Topic:      "step_completed",
				Content:    response,
				StepID:     step.ID,
			})
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
				// 並列時は確認不可 → SkipRetry で即ユーザー確認へ
				w.sharedContext.Publish(WorkerMessage{
					FromWorker: w.id,
					Topic:      "step_failed",
					Content:    fmt.Sprintf("tool %s requires user confirmation", tc.Tool),
					StepID:     step.ID,
				})
				return WorkerResult{
					WorkerID:      w.id,
					StepID:        step.ID,
					Success:       false,
					SkipRetry:     true,
					Error:         fmt.Errorf("tool %s requires user confirmation", tc.Tool),
					ToolsExecuted: toolsExecuted,
					Duration:      time.Since(startTime),
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
				err := &StepFailure{StepID: step.ID, Reason: reason}
				w.sharedContext.Publish(WorkerMessage{
					FromWorker: w.id,
					Topic:      "step_failed",
					Content:    err.Error(),
					StepID:     step.ID,
				})
				return WorkerResult{
					WorkerID:      w.id,
					StepID:        step.ID,
					Success:       false,
					Error:         err,
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
	err := errors.New("max iterations reached")
	w.sharedContext.Publish(WorkerMessage{
		FromWorker: w.id,
		Topic:      "step_failed",
		Content:    err.Error(),
		StepID:     step.ID,
	})
	return WorkerResult{
		WorkerID:      w.id,
		StepID:        step.ID,
		Success:       false,
		Error:         err,
		ToolsExecuted: toolsExecuted,
		Duration:      time.Since(startTime),
	}
}

// executeInvestigation は調査を実行
func (w *Worker) executeInvestigation(ctx context.Context, query string) WorkerResult {
	startTime := time.Now()

	// Worker は Thinking を無効化
	ctx = api.WithThinkingDisabled(ctx)

	// タイムアウト付きコンテキスト
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(w.stepTimeout)*time.Second)
	defer cancel()

	// 調査用の履歴を構築
	prompt := promptplan.BuildInvestigationExecutionPrompt(query)
	history := []api.Message{
		{Role: "user", Content: prompt},
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

		history = append(history, api.Message{
			Role:             "assistant",
			Content:          response,
			ReasoningContent: api.GetReasoningContent(w.provider),
		})

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
	// BuildStepExecutionPrompt は context を含むプロンプトを生成
	stepPrompt := promptplan.BuildStepExecutionPrompt(step, contextInfo)

	return []api.Message{
		{Role: "user", Content: stepPrompt},
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
