package agent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// runImplementationPhase は実装フェーズを実行（順次実行）
func (a *Agent) runImplementationPhase(ctx context.Context, p *plan.Plan) error {
	for {
		// 次のステップを取得
		nextID := p.GetNextStep()
		if nextID == -1 {
			break // 全て完了
		}
		step := p.GetStep(nextID)
		if step == nil {
			break
		}

		fmt.Printf("\n%s\n", ui.FormatStepProgress(nextID, len(p.Steps), step.Description, "running"))
		if err := a.executeStepV2(ctx, p, step, nextID-1, 0); err != nil {
			return err
		}
		p.UpdateStatus(nextID, "completed", "")

		// ステップ完了フックを実行
		if hooks := config.GetGlobalConfig().Hooks; len(hooks.OnStepComplete) > 0 {
			if !a.runStepCompleteHooksWithRetry(ctx, nextID, step.Description, "completed") {
				yellow.Printf("⚠️  Step %d hooks failed but proceeding to next step\n", nextID)
			}
		}

		// 全て完了したか確認
		if p.IsCompleted() {
			break
		}
	}

	green.Printf("\n✓ All %d steps completed!\n", len(p.Steps))
	return nil
}

// executeStepV2 は単一ステップを実行（失敗検知・リトライ対応）
func (a *Agent) executeStepV2(ctx context.Context, p *plan.Plan, step *plan.PlanStep, idx int, retryCount int) error {
	maxRetries := config.PlanMaxRetries

	if retryCount > 0 {
		ui.StopGlobalSpinner()
		yellow.Printf("🔄 Retry attempt %d/%d for step %d...\n", retryCount, maxRetries, step.ID)
	}

	// ステップ実行を指示
	stepPrompt := promptplan.BuildStepPrompt(step.ID, step.Description, step.Tools)

	// リトライ時は履歴に追加しない
	if retryCount == 0 {
		a.History = append(a.History, api.Message{Role: "user", Content: stepPrompt})
	}

	// ステップ内のツール実行ループ
	cfg := config.GetGlobalConfig()
	maxStepIterations := cfg.General.ToolLoopLimit
	maxContinues := config.PlanMaxAutoContinues
	continueCount := 0
	var lastFailedResult string
	var lastFailReason string

	// ステップ完了検証用トラッカー
	executedTools := make(map[string]bool) // ステップ内で実行されたツール名
	stepHadWrites := false                 // 書き込み系ツールが実行されたか
	beforeDiffFiles := getGitDiffFiles()   // Level 2 用: ステップ開始時の diff ファイル一覧

	for j := 0; j < maxStepIterations; j++ {
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			ui.StopGlobalSpinner()
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		a.History = append(a.History, api.Message{
			Role:             "assistant",
			Content:          response,
			ReasoningContent: a.getLastReasoningContent(),
		})
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}

		// ツール呼び出しチェック
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// AIが質問している場合、自動続行を試みる
			if isAIQuestion(response) && continueCount < maxContinues {
				continueCount++
				yellow.Printf("⚠️  AI asked a question, auto-continuing (%d/%d)...\n", continueCount, maxContinues)

				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: "[AUTO-CONTINUE] Yes, proceed with the step. Execute the required tools directly without asking for confirmation.",
				})
				continue
			}

			// Level 1: 計画の tools に書き込み系があるのに未実行 → 強制続行
			if missing := checkMissingWriteTools(step.Tools, executedTools); len(missing) > 0 {
				if continueCount < maxContinues {
					continueCount++
					yellow.Printf("⚠️  Step %d: expected %s but never executed (%d/%d)\n",
						step.ID, strings.Join(missing, ", "), continueCount, maxContinues)
					a.History = append(a.History, api.Message{
						Role: "user",
						Content: fmt.Sprintf("[SYSTEM] Step %d requires %s but none were executed. "+
							"Do NOT declare completion. Execute the required tools now.", step.ID, strings.Join(missing, ", ")),
					})
					continue
				}
				// maxContinues 到達 → failed
				yellow.Printf("⚠️  Step %d: required %s never executed after %d attempts. Marking as failed.\n",
					step.ID, strings.Join(missing, ", "), maxContinues)
				return fmt.Errorf("step %d incomplete: required tools [%s] were never executed after %d attempts",
					step.ID, strings.Join(missing, ", "), maxContinues)
			}

			// Level 2: 書き込みツール実行済みだが diff 変化なし → 強制続行
			if stepHadWrites {
				afterDiffFiles := getGitDiffFiles()
				if beforeDiffFiles != nil && afterDiffFiles != nil && diffFilesEqual(beforeDiffFiles, afterDiffFiles) {
					if continueCount < maxContinues {
						continueCount++
						yellow.Printf("⚠️  Step %d: write tools executed but no file changes detected (%d/%d)\n",
							step.ID, continueCount, maxContinues)
						a.History = append(a.History, api.Message{
							Role: "user",
							Content: fmt.Sprintf("[SYSTEM] Step %d executed write tools but git diff shows no new changes. "+
								"The tool may have failed silently. Verify and retry.", step.ID),
						})
						continue
					}
				}
			}

			// ツール呼び出しなし = ステップ完了
			green.Printf("✓ Step %d completed\n", step.ID)
			return nil
		}

		// ツールを実行
		var allResults []string
		for _, toolCall := range toolCalls {
			// Plan 実行中は create_plan を無視（再帰防止）
			if toolCall.Tool == "create_plan" {
				allResults = append(allResults, "[create_plan] Ignored: already executing a plan. Continue with current step.")
				continue
			}

			if a.Stats != nil {
				a.Stats.AddToolExecution(toolCall.Tool)
			}

			result, change := tools.Execute(toolCall)
			allResults = append(allResults, fmt.Sprintf("[%s]\n%s", toolCall.Tool, result))

			// ステップ完了検証用: 実行されたツールを記録
			executedTools[toolCall.Tool] = true
			if tools.IsWriteTool(toolCall.Tool) {
				stepHadWrites = true
			}

			// 失敗パターンをチェック
			if failed, reason := plan.ContainsFailure(result); failed {
				lastFailedResult = result
				lastFailReason = reason
			}

			// 変更履歴を保存
			if change != nil {
				a.changeStack = append(a.changeStack, *change)
				if len(a.changeStack) > config.MaxChangeStack {
					a.changeStack = a.changeStack[1:]
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
			cfg := config.GetGlobalConfig()
			autoRetryMax := cfg.PlanMode.AutoRetry

			// 自動リトライが有効で、まだ上限に達していない場合
			if autoRetryMax > 0 && retryCount < autoRetryMax {
				ui.StopGlobalSpinner()
				red.Printf("❌ Step %d Failed (auto-retry %d/%d)\n", step.ID, retryCount+1, autoRetryMax)
				yellow.Printf("🔄 Retrying...\n")

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
				return a.executeStepV2(ctx, p, step, idx, retryCount+1)
			}

			// 自動リトライが exhausted または無効 → Selector UI で確認
			a.SetStatus(StateWaitingApproval, "Step failed - waiting for action", "ステップ失敗 - アクション待ち", "Choose action", "アクションを選択")
			ui.StopGlobalSpinner()

			action, comment := promptFailureActionWithSelector(step, lastFailedResult, lastFailReason, autoRetryMax)

			switch action {
			case plan.FailureActionRetry:
				if retryCount >= maxRetries {
					red.Printf("⚠️  Max retries (%d) reached for step %d\n", maxRetries, step.ID)
					return fmt.Errorf("step %d failed after %d retries", step.ID, maxRetries)
				}
				// 手動リトライ: リトライカウンターをリセットして新しいシーケンスを開始
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
				return a.executeStepV2(ctx, p, step, idx, 0) // リセット: retryCount=0
			case plan.FailureActionComment:
				if retryCount >= maxRetries {
					red.Printf("⚠️  Max retries (%d) reached for step %d\n", maxRetries, step.ID)
					return fmt.Errorf("step %d failed after %d retries", step.ID, maxRetries)
				}
				// ユーザーの指示付きリトライ: リトライカウンターをリセット
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: fmt.Sprintf(`The previous step FAILED. Here are the user's instructions for fixing it:

%s

Error that occurred:
%s

Please follow these instructions to fix the issue and retry the step.`, comment, lastFailedResult),
				})
				return a.executeStepV2(ctx, p, step, idx, 0) // リセット: retryCount=0
			case plan.FailureActionSkip:
				yellow.Printf("⏭️  Step %d skipped by user\n", step.ID)
				return nil
			case plan.FailureActionAbort:
				red.Printf("🛑 Step %d aborted by user\n", step.ID)
				return fmt.Errorf("step %d aborted by user: %s", step.ID, lastFailReason)
			}
		}

		// 結果を履歴に追加
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Results]\n%s", strings.Join(allResults, "\n\n")),
		})
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

// checkMissingWriteTools は step.Tools に含まれる書き込み系ツールのうち
// 実際に実行されなかったものを返す。
func checkMissingWriteTools(stepTools []string, executedTools map[string]bool) []string {
	var missing []string
	for _, t := range stepTools {
		if tools.IsWriteTool(t) && !executedTools[t] {
			missing = append(missing, t)
		}
	}
	return missing
}

// getGitDiffFiles は git diff --name-only HEAD の結果をファイル名セットとして返す。
// HEAD と比較することで staged + unstaged 両方の変更を検出する。
// git が使えない場合は nil を返す（Level 2 スキップ用）。
func getGitDiffFiles() map[string]bool {
	out, err := exec.Command("git", "diff", "--name-only", "HEAD").CombinedOutput()
	if err != nil {
		return nil
	}
	files := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files[line] = true
		}
	}

	// untracked files も検出（write_file で新規作成されたファイル用）
	untrackedOut, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").CombinedOutput()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(untrackedOut)), "\n") {
			if line != "" {
				files[line] = true
			}
		}
	}

	return files
}

// diffFilesEqual は2つの diff ファイルセットが同一かを比較する。
func diffFilesEqual(before, after map[string]bool) bool {
	if len(before) != len(after) {
		return false
	}
	for f := range before {
		if !after[f] {
			return false
		}
	}
	return true
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
