package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"golang.org/x/sync/errgroup"
)

// ParallelExecutor は複数ステップの並列実行を管理
type ParallelExecutor struct {
	agent *Agent
	plan  *Plan
	mu    sync.Mutex // 共有リソースの保護
}

// NewParallelExecutor は新しいParallelExecutorを作成
func NewParallelExecutor(agent *Agent, plan *Plan) *ParallelExecutor {
	return &ParallelExecutor{
		agent: agent,
		plan:  plan,
	}
}

// ExecuteSteps は複数のステップを並列実行
func (e *ParallelExecutor) ExecuteSteps(ctx context.Context, stepIDs []int) error {
	if len(stepIDs) == 0 {
		return nil
	}

	// 1つだけの場合は通常実行
	if len(stepIDs) == 1 {
		step := e.plan.GetStep(stepIDs[0])
		if step == nil {
			return fmt.Errorf("step %d not found", stepIDs[0])
		}
		return e.agent.executeStep(ctx, e.plan, step)
	}

	// 複数の場合は並列実行
	cyan.Printf("\n🔀 Executing %d steps in parallel...\n", len(stepIDs))

	// errgroupで並列実行とエラーハンドリング
	g, gctx := errgroup.WithContext(ctx)

	// 各ステップを並列実行
	for _, stepID := range stepIDs {
		stepID := stepID // クロージャー用にコピー
		step := e.plan.GetStep(stepID)
		if step == nil {
			continue
		}

		g.Go(func() error {
			// ステップ実行
			cyan.Printf("[Step %d] Starting: %s\n", step.ID, step.Description)

			// ステータスを実行中に更新（排他制御）
			e.mu.Lock()
			e.plan.UpdateStatus(step.ID, "running", "")
			e.mu.Unlock()

			// 実行
			err := e.executeStepParallel(gctx, step)

			e.mu.Lock()
			defer e.mu.Unlock()

			if err != nil {
				e.plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Error: %v", err))
				red.Printf("[Step %d] Failed: %v\n", step.ID, err)
				return fmt.Errorf("step %d failed: %w", step.ID, err)
			}

			e.plan.UpdateStatus(step.ID, "completed", "Success")
			green.Printf("[Step %d] Completed: %s\n", step.ID, step.Description)
			return nil
		})
	}

	// すべてのステップの完了を待つ
	if err := g.Wait(); err != nil {
		return err
	}

	green.Println("✓ All parallel steps completed successfully!")
	return nil
}

// executeStepParallel は単一ステップを並列実行用に実行
func (e *ParallelExecutor) executeStepParallel(ctx context.Context, step *PlanStep) error {
	// 期待するツールがある場合、明示的に指示
	toolsHint := ""
	if len(step.Tools) > 0 {
		toolsHint = fmt.Sprintf("\n\nYou MUST use the following tools to complete this step: %s\nDo NOT ask for confirmation - execute the tools directly.", strings.Join(step.Tools, ", "))
	}

	// AIにステップ実行を依頼
	stepPrompt := fmt.Sprintf(`Execute this step from the plan:
Step %d: %s

IMPORTANT INSTRUCTIONS:
1. Execute this step autonomously without asking questions
2. Use tools directly - do NOT ask "Should I proceed?" or "Do you want me to..."
3. If you need to create/modify files, use write_file or str_replace directly
4. Only stop for SafetyLow operations (delete_file, dangerous bash commands)%s`,
		step.ID,
		step.Description,
		toolsHint,
	)

	// 履歴に追加（排他制御）
	e.mu.Lock()
	e.agent.History = append(e.agent.History, api.Message{Role: "user", Content: stepPrompt})
	history := make([]api.Message, len(e.agent.History))
	copy(history, e.agent.History)
	e.mu.Unlock()

	response, err := e.agent.CurrentProvider.ChatWithTools(
		ctx,
		e.agent.SystemPrompt,
		history,
		e.agent.CurrentModel,
	)
	if err != nil {
		return fmt.Errorf("step %d failed: %w", step.ID, err)
	}

	// レスポンスを履歴に追加（排他制御）
	e.mu.Lock()
	e.agent.History = append(e.agent.History, api.Message{Role: "assistant", Content: response})
	if e.agent.Stats != nil {
		e.agent.Stats.AssistantMessages++
	}
	e.mu.Unlock()

	// ツール呼び出しを処理
	maxToolCalls := 10 // 無限ループ防止

	for i := 0; i < maxToolCalls; i++ {
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			break
		}

		// 複数ツールを順次実行
		var allResults []string
		for _, toolCall := range toolCalls {
			// 統計情報更新: ツール実行回数（排他制御）
			e.mu.Lock()
			if e.agent.Stats != nil {
				e.agent.Stats.AddToolExecution(toolCall.Tool)
			}
			e.plan.IncrementToolsExecuted(step.ID)
			e.mu.Unlock()

			// ツール実行
			result, change := tools.Execute(toolCall)
			allResults = append(allResults, fmt.Sprintf("[%s]\n%s", toolCall.Tool, result))

			// 変更履歴を保存（排他制御）
			if change != nil {
				e.mu.Lock()
				e.agent.changeStack = append(e.agent.changeStack, *change)
				if len(e.agent.changeStack) > config.MaxChangeStack {
					e.agent.changeStack = e.agent.changeStack[1:]
				}
				if e.agent.changeStorage != nil && e.agent.session != nil {
					if err := e.agent.changeStorage.AppendChange(e.agent.session.ID, *change); err != nil {
						yellow.Printf("Warning: Failed to persist change: %v\n", err)
					}
				}
				e.mu.Unlock()
			}
		}

		// 結果を履歴に追加（排他制御）
		e.mu.Lock()
		e.agent.History = append(e.agent.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Results]\n%s", strings.Join(allResults, "\n\n")),
		})
		history = make([]api.Message, len(e.agent.History))
		copy(history, e.agent.History)
		e.mu.Unlock()

		// 次のAI応答を取得
		response, err = e.agent.CurrentProvider.ChatWithTools(
			ctx,
			e.agent.SystemPrompt,
			history,
			e.agent.CurrentModel,
		)
		if err != nil {
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		// レスポンスを履歴に追加（排他制御）
		e.mu.Lock()
		e.agent.History = append(e.agent.History, api.Message{Role: "assistant", Content: response})
		if e.agent.Stats != nil {
			e.agent.Stats.AssistantMessages++
		}
		e.mu.Unlock()
	}

	return nil
}
