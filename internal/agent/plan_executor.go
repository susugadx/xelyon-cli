package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"golang.org/x/sync/errgroup"
)

// runImplementationPhase は実装フェーズを実行（並列実行対応）
func (a *Agent) runImplementationPhase(ctx context.Context, plan *Plan) error {
	// 依存関係解析器を初期化
	analyzer := NewDependencyAnalyzer(nil) // LSP連携は将来の拡張
	_ = analyzer.Analyze(plan.Steps)       // ファイルアクセスマップを構築

	for {
		// 並列実行可能なステップを取得
		parallelSteps := plan.GetParallelSteps()

		if len(parallelSteps) > 1 {
			// 並列実行前に競合チェック
			conflicts := analyzer.DetectConflicts(parallelSteps, plan.Steps)
			if len(conflicts) > 0 {
				// 競合検出時は直列実行にフォールバック
				yellow.Printf("\n⚠️  Conflict detected, falling back to sequential execution:\n")
				for _, c := range conflicts {
					yellow.Printf("   - %s (files: %v)\n", c.Message, c.Files)
				}
				// 直列実行
				for _, id := range parallelSteps {
					step := plan.GetStep(id)
					if step == nil {
						continue
					}
					cyan.Printf("\n[%d/%d] %s\n", id, len(plan.Steps), step.Description)
					if err := a.executeStepV2(ctx, plan, step, id-1, 0, false); err != nil {
						return err
					}
					plan.UpdateStatus(id, "completed", "")
				}
			} else {
				// 競合なし - 並列実行
				cyan.Printf("\n⚡ Executing %d steps in parallel...\n", len(parallelSteps))
				if err := a.executeStepsParallel(ctx, plan, parallelSteps); err != nil {
					return err
				}
				// 完了したステップをマーク
				for _, id := range parallelSteps {
					plan.UpdateStatus(id, "completed", "")
				}
			}
		} else {
			// 直列実行
			nextID := plan.GetNextStep()
			if nextID == -1 {
				break // 全て完了
			}
			step := plan.GetStep(nextID)
			if step == nil {
				break
			}

			cyan.Printf("\n[%d/%d] %s\n", nextID, len(plan.Steps), step.Description)
			if err := a.executeStepV2(ctx, plan, step, nextID-1, 0, false); err != nil {
				return err
			}
			plan.UpdateStatus(nextID, "completed", "")
		}

		// 全て完了したか確認
		if plan.IsCompleted() {
			break
		}
	}

	green.Printf("\n✓ All %d steps completed!\n", len(plan.Steps))
	return nil
}

// executeStepsParallel は複数ステップを並列実行
func (a *Agent) executeStepsParallel(ctx context.Context, plan *Plan, stepIDs []int) error {
	cfg := config.GetGlobalConfig()
	maxWorkers := cfg.PlanMode.MaxParallelSteps
	if maxWorkers <= 0 {
		maxWorkers = 3
	}

	// セマフォでワーカー数を制限
	sem := make(chan struct{}, maxWorkers)
	g, ctx := errgroup.WithContext(ctx)

	for _, stepID := range stepIDs {
		stepID := stepID // capture
		step := plan.GetStep(stepID)
		if step == nil {
			continue
		}

		g.Go(func() error {
			// セマフォ取得
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return ctx.Err()
			}

			cyan.Printf("\n[Parallel] Step %d: %s\n", step.ID, step.Description)
			return a.executeStepV2(ctx, plan, step, step.ID-1, 0, true)
		})
	}

	return g.Wait()
}

// executeStepV2 は単一ステップを実行（失敗検知・リトライ対応）
// parallel=true の場合はスレッドセーフなメソッドを使用し、ユーザー入力が必要な処理はスキップ
func (a *Agent) executeStepV2(ctx context.Context, plan *Plan, step *PlanStep, idx int, retryCount int, parallel bool) error {
	maxRetries := 3

	if retryCount > 0 {
		yellow.Printf("🔄 Retry attempt %d/%d for step %d...\n", retryCount, maxRetries, step.ID)
	}

	// 期待するツールがある場合、明示的に指示
	toolsHint := ""
	if len(step.Tools) > 0 {
		toolsHint = fmt.Sprintf("\n\nYou MUST use the following tools to complete this step: %s\nDo NOT ask for confirmation - execute the tools directly.", strings.Join(step.Tools, ", "))
	}

	// ステップ実行を指示
	stepPrompt := fmt.Sprintf(`Execute step %d of the implementation plan:
%s

IMPORTANT INSTRUCTIONS:
1. Execute this step autonomously without asking questions
2. Use tools directly - do NOT ask "Should I proceed?" or "Do you want me to..."
3. If you need to create/modify files, use write_file or str_replace directly
4. Only stop for SafetyLow operations (delete_file, dangerous bash commands)%s`, step.ID, step.Description, toolsHint)

	// リトライ時は履歴に追加しない
	if retryCount == 0 {
		if parallel {
			a.appendHistory(api.Message{Role: "user", Content: stepPrompt})
		} else {
			a.History = append(a.History, api.Message{Role: "user", Content: stepPrompt})
		}
	}

	// ステップ内のツール実行ループ
	maxStepIterations := 10
	maxContinues := 3
	continueCount := 0
	var lastFailedResult string
	var lastFailReason string

	for j := 0; j < maxStepIterations; j++ {
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		if parallel {
			a.appendHistory(api.Message{Role: "assistant", Content: response})
			a.incrementAssistantMessages()
		} else {
			a.History = append(a.History, api.Message{Role: "assistant", Content: response})
			if a.Stats != nil {
				a.Stats.AssistantMessages++
			}
		}

		// ツール呼び出しチェック
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// AIが質問している場合、自動続行を試みる
			if isAIQuestion(response) && continueCount < maxContinues {
				continueCount++
				yellow.Printf("⚠️  AI asked a question, auto-continuing (%d/%d)...\n", continueCount, maxContinues)

				msg := api.Message{
					Role:    "user",
					Content: "[AUTO-CONTINUE] Yes, proceed with the step. Execute the required tools directly without asking for confirmation.",
				}
				if parallel {
					a.appendHistory(msg)
				} else {
					a.History = append(a.History, msg)
				}
				continue
			}

			// ツール呼び出しなし = ステップ完了
			green.Printf("✓ Step %d completed\n", step.ID)
			return nil
		}

		// ツールを実行
		var allResults []string
		for _, toolCall := range toolCalls {
			if parallel {
				a.incrementToolExecution(toolCall.Tool)
			} else {
				if a.Stats != nil {
					a.Stats.AddToolExecution(toolCall.Tool)
				}
			}

			result, change := tools.Execute(toolCall)
			allResults = append(allResults, fmt.Sprintf("[%s]\n%s", toolCall.Tool, result))

			// 失敗パターンをチェック
			if failed, reason := containsFailure(result); failed {
				lastFailedResult = result
				lastFailReason = reason
			}

			// 変更履歴を保存
			if change != nil {
				if parallel {
					a.appendChange(*change)
				} else {
					a.changeStack = append(a.changeStack, *change)
					if len(a.changeStack) > config.MaxChangeStack {
						a.changeStack = a.changeStack[1:]
					}
				}

				if a.changeStorage != nil && a.session != nil {
					if err := a.changeStorage.AppendChange(a.session.ID, *change); err != nil {
						yellow.Printf("Warning: Failed to persist change: %v\n", err)
					}
				}
			}
		}

		// 失敗検出時の処理
		if lastFailedResult != "" {
			if parallel {
				// 並列実行時はユーザー入力不可、即座にエラーを返す
				return fmt.Errorf("step %d failed during parallel execution: %s", step.ID, lastFailReason)
			}

			a.SetStatus(StateWaitingApproval, "Step failed - waiting for action", "ステップ失敗 - アクション待ち", "Choose r/c/s/a", "r/c/s/a を選択")

			action := promptFailureAction(step, lastFailedResult, lastFailReason)

			switch action {
			case FailureActionRetry:
				if retryCount >= maxRetries {
					red.Printf("⚠️  Max retries (%d) reached for step %d\n", maxRetries, step.ID)
					return fmt.Errorf("step %d failed after %d retries", step.ID, maxRetries)
				}
				// リトライ用プロンプトを追加
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: fmt.Sprintf(`The previous step FAILED with the following error:

%s

Please:
1. Analyze the error carefully
2. Identify the root cause
3. Fix the code or configuration
4. Re-run the step to verify the fix

Do NOT skip this step. The issue must be resolved before proceeding.`, lastFailedResult),
				})
				return a.executeStepV2(ctx, plan, step, idx, retryCount+1, false)
			case FailureActionComment:
				if retryCount >= maxRetries {
					red.Printf("⚠️  Max retries (%d) reached for step %d\n", maxRetries, step.ID)
					return fmt.Errorf("step %d failed after %d retries", step.ID, maxRetries)
				}
				// ユーザーの指示付きリトライ
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: fmt.Sprintf(`The previous step FAILED. Here are the user's instructions for fixing it:

%s

Error that occurred:
%s

Please follow these instructions to fix the issue and retry the step.`, failureComment, lastFailedResult),
				})
				return a.executeStepV2(ctx, plan, step, idx, retryCount+1, false)
			case FailureActionSkip:
				yellow.Printf("⏭️  Step %d skipped by user\n", step.ID)
				return nil
			case FailureActionAbort:
				red.Printf("🛑 Step %d aborted by user\n", step.ID)
				return fmt.Errorf("step %d aborted by user: %s", step.ID, lastFailReason)
			}
		}

		// 結果を履歴に追加
		msg := api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Results]\n%s", strings.Join(allResults, "\n\n")),
		}
		if parallel {
			a.appendHistory(msg)
		} else {
			a.History = append(a.History, msg)
		}
	}

	green.Printf("✓ Step %d completed\n", step.ID)
	return nil
}

// isAIQuestion はAIが質問しているかを判定
func isAIQuestion(response string) bool {
	// ツール呼び出しがある場合は質問とみなさない
	if len(tools.ParseToolCalls(response)) > 0 {
		return false
	}

	questionPatterns := []string{
		"続行しますか", "よろしいですか", "確認してください", "どうしますか",
		"選択してください", "指定してください", "教えてください",
		"Should I", "Do you want", "Would you like", "Shall I",
		"Can you confirm", "Please confirm", "proceed?", "continue?",
	}

	lowered := strings.ToLower(response)
	for _, pattern := range questionPatterns {
		if strings.Contains(lowered, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// confirmPlan は計画の承認確認
func (a *Agent) confirmPlan() (approved bool, feedback string) {
	result := tools.ConfirmInteractive("Approve this plan?")

	switch result.Action {
	case "yes":
		return true, ""
	case "no":
		return false, ""
	case "comment":
		return false, strings.TrimSpace(result.Comment)
	default:
		return false, ""
	}
}
