package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
	"golang.org/x/sync/errgroup"
)

// runImplementationPhase は実装フェーズを実行（並列実行対応）
func (a *Agent) runImplementationPhase(ctx context.Context, p *plan.Plan) error {
	// 依存関係解析器を初期化
	analyzer := plan.NewDependencyAnalyzer(a.lspClient) // LSP有効時は参照ベースの依存関係検出
	_ = analyzer.Analyze(p.Steps)               // ファイルアクセスマップを構築

	for {
		// 並列実行可能なステップを取得
		parallelSteps := p.GetParallelSteps()

		if len(parallelSteps) > 1 {
			// 並列実行前に競合チェック
			conflicts := analyzer.DetectConflicts(parallelSteps, p.Steps)
			if len(conflicts) > 0 {
				// 競合検出時は直列実行にフォールバック
				yellow.Printf("\n⚠️  Conflict detected, falling back to sequential execution:\n")
				for _, c := range conflicts {
					yellow.Printf("   - %s (files: %v)\n", c.Message, c.Files)
				}
				// 直列実行
				for _, id := range parallelSteps {
					step := p.GetStep(id)
					if step == nil {
						continue
					}
					cyan.Printf("\n[%d/%d] %s\n", id, len(p.Steps), step.Description)
					if err := a.executeStepV2(ctx, p, step, id-1, 0, false); err != nil {
						return err
					}
					p.UpdateStatus(id, "completed", "")
				}
			} else {
				// 競合なし - 並列実行
				cyan.Printf("\n⚡ Executing %d steps in parallel...\n", len(parallelSteps))
				if err := a.executeStepsParallel(ctx, p, parallelSteps); err != nil {
					return err
				}
				// 完了したステップをマーク
				for _, id := range parallelSteps {
					p.UpdateStatus(id, "completed", "")
				}
			}
		} else {
			// 直列実行
			nextID := p.GetNextStep()
			if nextID == -1 {
				break // 全て完了
			}
			step := p.GetStep(nextID)
			if step == nil {
				break
			}

			cyan.Printf("\n[%d/%d] %s\n", nextID, len(p.Steps), step.Description)
			if err := a.executeStepV2(ctx, p, step, nextID-1, 0, false); err != nil {
				return err
			}
			p.UpdateStatus(nextID, "completed", "")
		}

		// 全て完了したか確認
		if p.IsCompleted() {
			break
		}
	}

	green.Printf("\n✓ All %d steps completed!\n", len(p.Steps))
	return nil
}

// executeStepsParallel は複数ステップを並列実行（進捗表示付き）
func (a *Agent) executeStepsParallel(ctx context.Context, p *plan.Plan, stepIDs []int) error {
	cfg := config.GetGlobalConfig()
	maxWorkers := cfg.PlanMode.MaxParallelSteps
	if maxWorkers <= 0 {
		maxWorkers = config.DefaultParallelWorkers
	}

	// MultiProgress を初期化
	mp := ui.NewMultiProgress()
	for _, stepID := range stepIDs {
		step := p.GetStep(stepID)
		if step != nil {
			mp.AddTask(stepID, step.Description)
		}
	}

	// バックグラウンドで進捗表示を開始
	mp.StartRendering()
	defer mp.StopRendering()

	// セマフォでワーカー数を制限
	sem := make(chan struct{}, maxWorkers)
	g, ctx := errgroup.WithContext(ctx)

	for _, stepID := range stepIDs {
		stepID := stepID // capture
		step := p.GetStep(stepID)
		if step == nil {
			continue
		}

		g.Go(func() error {
			// セマフォ取得
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mp.Fail(stepID, "cancelled")
				return ctx.Err()
			}

			// ステップ開始
			mp.Start(stepID)

			// ステップ実行
			err := a.executeStepV2(ctx, p, step, step.ID-1, 0, true)

			if err != nil {
				mp.Fail(stepID, err.Error())
				return err
			}

			// ステップ完了
			mp.Done(stepID)
			return nil
		})
	}

	return g.Wait()
}

// executeStepV2 は単一ステップを実行（失敗検知・リトライ対応）
// parallel=true の場合はスレッドセーフなメソッドを使用し、ユーザー入力が必要な処理はスキップ
func (a *Agent) executeStepV2(ctx context.Context, p *plan.Plan, step *plan.PlanStep, idx int, retryCount int, parallel bool) error {
	maxRetries := config.PlanMaxRetries

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
	maxStepIterations := config.PlanMaxIterations
	maxContinues := config.PlanMaxAutoContinues
	continueCount := 0
	var lastFailedResult string
	var lastFailReason string

	for j := 0; j < maxStepIterations; j++ {
		// 並列実行時はスナップショットを使用してレースコンディションを防止
		var history []api.Message
		if parallel {
			history = a.getHistorySnapshot()
		} else {
			history = a.History
		}
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			history,
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
			if failed, reason := plan.ContainsFailure(result); failed {
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

			action, comment := promptFailureAction(step, lastFailedResult, lastFailReason)

			switch action {
			case plan.FailureActionRetry:
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
				return a.executeStepV2(ctx, p, step, idx, retryCount+1, false)
			case plan.FailureActionComment:
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

Please follow these instructions to fix the issue and retry the step.`, comment, lastFailedResult),
				})
				return a.executeStepV2(ctx, p, step, idx, retryCount+1, false)
			case plan.FailureActionSkip:
				yellow.Printf("⏭️  Step %d skipped by user\n", step.ID)
				return nil
			case plan.FailureActionAbort:
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
