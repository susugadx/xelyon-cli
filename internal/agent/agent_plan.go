package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RunPlanMode はPlan Modeで計画を生成・承認・実行
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	// 実装前チェック：既存定義の重複を警告
	if warning := CheckBeforeImplementation(userRequest); warning != "" {
		yellow.Println(warning)
		// AIにも警告を伝える（計画生成時に考慮させる）
		userRequest = userRequest + "\n\n[SYSTEM NOTE: " + warning + " Please check existing code before creating new definitions.]"
	}

	// Step 1: 計画生成をAIに依頼
	a.SetStatus(StateRunning, "Generating plan", "計画生成中", "Wait for plan output", "計画の出力を待ってください")
	cyan.Println("\n📋 Generating execution plan...")

	// ユーザーリクエストを履歴に追加
	planMessage := api.Message{
		Role: "user",
		Content: fmt.Sprintf(`USER REQUEST: %s

IMPORTANT: You are in Plan Mode. Do NOT execute any tools yet.

Your task is to:
1. Analyze the user's request above
2. Break it down into a sequence of concrete steps
3. Output ONLY a JSON plan in this exact format (no other text):

{"steps": [
  {"id": 1, "description": "First step description", "tools": ["tool1", "tool2"], "depends_on": [], "parallel": false},
  {"id": 2, "description": "Second step description", "tools": ["tool3"], "depends_on": [1], "parallel": false}
]}

Rules:
- Start with id=1 and increment sequentially
- "description": Brief description of what this step does
- "tools": List of tool names to use (can be empty if no tools needed)
- "depends_on": List of step IDs that must complete before this step (empty array [] if no dependencies)
- "parallel": true if this step can run in parallel with other steps with same dependencies

Example plan:
{"steps": [
  {"id": 1, "description": "Read current Agent struct definition", "tools": ["read_file"], "depends_on": [], "parallel": false},
  {"id": 2, "description": "Add DryRunMode field to Agent struct", "tools": ["str_replace"], "depends_on": [1], "parallel": false},
  {"id": 3, "description": "Implement /dryrun command handler", "tools": ["write_file"], "depends_on": [1], "parallel": false},
  {"id": 4, "description": "Write tests for dry run functionality", "tools": ["write_file"], "depends_on": [2, 3], "parallel": false}
]}

Output the JSON plan now:`, userRequest),
	}
	historyWithRequest := append(a.History, planMessage)

	// AIに計画生成を依頼
	response, err := a.CurrentProvider.ChatWithTools(
		ctx,
		a.SystemPrompt,
		historyWithRequest,
		a.CurrentModel,
	)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	// デバッグ: レスポンスを表示（開発用）
	// fmt.Printf("DEBUG: AI response:\n%s\n", response)

	// Step 2: レスポンスからPlan JSONを抽出
	planJSON, err := ExtractPlanJSON(response)
	if err != nil {
		red.Printf("❌ Failed to extract plan JSON from AI response\n")
		yellow.Printf("AI Response:\n%s\n", response)
		return fmt.Errorf("failed to extract plan JSON: %w (AI may have returned a tool call instead of a plan)", err)
	}

	// Step 3: Planをパース
	plan, err := ParsePlan(planJSON)
	if err != nil {
		red.Printf("❌ Failed to parse plan JSON\n")
		yellow.Printf("Extracted JSON:\n%s\n", planJSON)
		return fmt.Errorf("failed to parse plan: %w", err)
	}

	// 空の計画チェック
	if len(plan.Steps) == 0 {
		red.Println("❌ Generated plan has no steps")
		return fmt.Errorf("empty plan generated")
	}

	// Step 4: 計画を表示
	fmt.Println()
	fmt.Println(FormatPlan(plan))
	fmt.Println()

	// Step 5: ユーザー承認
	a.SetStatus(StateWaitingApproval, "Waiting for plan approval", "計画の承認待ち", "Answer y/n/c", "y/n/c で回答")
	approved, feedback := a.confirmPlan()
	if !approved {
		if feedback != "" {
			yellow.Printf("Plan rejected with feedback: %s\n", feedback)
			// フィードバック付き拒否の場合、再生成を試みる
			return a.RunPlanMode(ctx, userRequest+" (Previous plan feedback: "+feedback+")")
		}
		red.Println("Plan execution cancelled.")
		return nil
	}

	green.Println("✓ Plan approved. Starting execution...")
	a.SetStatus(StateRunning, "Executing plan", "計画を実行中", "Wait for completion", "完了を待ってください")

	// Step 6: 自律実行
	err = a.executePlan(ctx, plan)
	if err != nil {
		a.SetStatus(StateAborted, "Plan execution failed", "計画の実行に失敗", "Review errors and retry", "エラーを確認して再試行")
		return err
	}

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
	return nil
}

// confirmPlan は計画の承認確認
func (a *Agent) confirmPlan() (approved bool, feedback string) {
	result := tools.ConfirmInteractive("Approve this plan? [y/n/c(omment)]:")

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

// executePlan は承認された計画を実行
func (a *Agent) executePlan(ctx context.Context, plan *Plan) error {
	for {
		// 次に実行すべきステップを取得
		nextStepID := plan.GetNextStep()
		if nextStepID == -1 {
			// すべて完了 or ブロック中
			if plan.IsCompleted() {
				// 実行統計を表示
				a.showPlanSummary(plan)
				return nil
			}
			if plan.HasFailed() {
				red.Println("\n✗ Plan execution failed.")
				a.showPlanSummary(plan)
				return fmt.Errorf("plan execution failed")
			}
			// ブロック中（依存関係未解決）
			yellow.Println("\n⚠ Plan execution blocked. Waiting for dependencies...")
			return fmt.Errorf("plan execution blocked")
		}

		step := plan.GetStep(nextStepID)
		if step == nil {
			return fmt.Errorf("step %d not found", nextStepID)
		}

		// 並列実行可能なステップを取得
		parallelSteps := plan.GetParallelSteps()
		if len(parallelSteps) > 1 {
			// 並列実行
			if err := a.executeParallelSteps(ctx, plan, parallelSteps); err != nil {
				return err
			}
		} else {
			// 順次実行
			if err := a.executeStep(ctx, plan, step); err != nil {
				return err
			}
		}
	}
}

// showPlanSummary は計画実行のサマリーを表示
func (a *Agent) showPlanSummary(plan *Plan) {
	fmt.Println()
	cyan.Println("📊 Plan Execution Summary:")
	cyan.Println(strings.Repeat("─", 50))

	totalTools := 0
	completedSteps := 0
	failedSteps := 0
	warningSteps := 0

	for _, step := range plan.Steps {
		statusIcon := "✓"
		statusColor := green

		switch step.Status {
		case "completed":
			completedSteps++
			if strings.Contains(step.Result, "Warning") {
				statusIcon = "⚠"
				statusColor = yellow
				warningSteps++
			}
		case "failed":
			statusIcon = "✗"
			statusColor = red
			failedSteps++
		default:
			statusIcon = "○"
			statusColor = yellow
		}

		statusColor.Printf("  %s Step %d: %s\n", statusIcon, step.ID, step.Description)
		if step.ToolsExecuted > 0 {
			fmt.Printf("      Tools executed: %d\n", step.ToolsExecuted)
		}
		totalTools += step.ToolsExecuted
	}

	cyan.Println(strings.Repeat("─", 50))

	// 最終結果
	if failedSteps == 0 && warningSteps == 0 {
		green.Printf("✓ All %d steps completed successfully!\n", len(plan.Steps))
	} else if failedSteps == 0 {
		yellow.Printf("⚠ %d steps completed with %d warnings\n", completedSteps, warningSteps)
	} else {
		red.Printf("✗ %d steps failed, %d completed\n", failedSteps, completedSteps)
	}

	fmt.Printf("  Total tools executed: %d\n", totalTools)
}

// containsFailure はツール結果に失敗パターンが含まれるか検出
// 失敗を検出した場合、(true, 理由) を返す
func containsFailure(result string) (bool, string) {
	failPatterns := map[string]string{
		"--- FAIL:":      "Go test failed",
		"FAIL\t":         "Go test failed",
		"exit status 1":  "Command failed with exit code 1",
		"exit status":    "Command failed",
		"panic:":         "Panic detected",
		"fatal error:":   "Fatal error",
		"AssertionError": "Assertion failed",
		"FAILED":         "Test failed",
		"npm ERR!":       "npm error",
		"SyntaxError":    "Syntax error",
		"TypeError":      "Type error",
		"ReferenceError": "Reference error",
		"compile error":  "Compilation error",
		"build failed":   "Build failed",
		"error: ":        "Error occurred",
		"Error: ":        "Error occurred",
	}

	// パターンの優先度順にチェック（より具体的なものを先に）
	priorityPatterns := []string{
		"--- FAIL:",
		"FAIL\t",
		"panic:",
		"fatal error:",
		"build failed",
		"compile error",
		"AssertionError",
		"npm ERR!",
		"SyntaxError",
		"TypeError",
		"ReferenceError",
		"FAILED",
		"exit status 1",
		"exit status",
		"error: ",
		"Error: ",
	}

	for _, pattern := range priorityPatterns {
		if strings.Contains(result, pattern) {
			return true, failPatterns[pattern]
		}
	}
	return false, ""
}

// FailureAction は失敗時のユーザー選択
type FailureAction string

const (
	FailureActionRetry FailureAction = "retry"
	FailureActionSkip  FailureAction = "skip"
	FailureActionAbort FailureAction = "abort"
)

// promptFailureAction は失敗時のユーザー選択UIを表示
func promptFailureAction(step *PlanStep, result string, reason string) FailureAction {
	fmt.Println()
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	red.Printf("❌ Step %d Failed: %s\n", step.ID, step.Description)
	red.Printf("   Reason: %s\n", reason)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// エラー内容を表示（最大20行）
	lines := strings.Split(result, "\n")
	maxShow := 20
	if len(lines) > maxShow {
		fmt.Println()
		yellow.Println("Error output (last 20 lines):")
		for _, line := range lines[len(lines)-maxShow:] {
			fmt.Printf("  %s\n", line)
		}
		fmt.Printf("  ... (%d more lines above)\n", len(lines)-maxShow)
	} else {
		fmt.Println()
		yellow.Println("Error output:")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	}

	fmt.Println()
	yellow.Println("Options:")
	yellow.Println("  [r]etry  - AI will analyze the error and try to fix it")
	yellow.Println("  [s]kip   - Mark as skipped and continue to next step")
	yellow.Println("  [a]bort  - Stop plan execution")
	fmt.Println()

	// 入力を受け付け
	result2 := tools.ConfirmInteractive("Choose action [r/s/a]:")

	switch result2.Action {
	case "yes":
		// y を押した場合は retry として扱う
		return FailureActionRetry
	case "no":
		// n を押した場合は abort として扱う
		return FailureActionAbort
	case "comment":
		// コメントがある場合は AIに伝えてretry
		if strings.TrimSpace(result2.Comment) != "" {
			yellow.Printf("📝 Comment received: %s\n", result2.Comment)
			yellow.Println("   (Comment will be passed to AI for retry)")
		}
		return FailureActionRetry
	default:
		return FailureActionAbort
	}
}

// retryStepWithFix はAIにエラーを伝えて修正を依頼し再実行
func (a *Agent) retryStepWithFix(ctx context.Context, plan *Plan, step *PlanStep, errorResult string, userComment string, retryCount int) error {
	maxRetries := 3

	if retryCount >= maxRetries {
		red.Printf("⚠️  Max retries (%d) reached for step %d\n", maxRetries, step.ID)
		plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Max retries reached after %d attempts", maxRetries))
		return fmt.Errorf("step %d failed after %d retries", step.ID, maxRetries)
	}

	yellow.Printf("\n🔄 Retry attempt %d/%d for step %d...\n", retryCount+1, maxRetries, step.ID)

	// エラー出力を適切なサイズにトリム
	trimmedError := errorResult
	lines := strings.Split(errorResult, "\n")
	if len(lines) > 50 {
		// 最初の10行と最後の40行を保持
		trimmedLines := append(lines[:10], "... (truncated) ...")
		trimmedLines = append(trimmedLines, lines[len(lines)-40:]...)
		trimmedError = strings.Join(trimmedLines, "\n")
	}

	// リトライ用のプロンプト構築
	retryPrompt := fmt.Sprintf(`The previous step FAILED with the following error:

%s

Please:
1. Analyze the error carefully
2. Identify the root cause
3. Fix the code or configuration
4. Re-run the step to verify the fix

Do NOT skip this step. The issue must be resolved before proceeding.`, trimmedError)

	// ユーザーコメントがあれば追加
	if userComment != "" {
		retryPrompt += fmt.Sprintf("\n\nUser feedback: %s", userComment)
	}

	a.History = append(a.History, api.Message{Role: "user", Content: retryPrompt})

	// ステータスを retrying に更新
	plan.UpdateStatus(step.ID, "running", fmt.Sprintf("Retry %d/%d", retryCount+1, maxRetries))

	// 再実行（内部でretryCountをインクリメント）
	return a.executeStepWithRetry(ctx, plan, step, retryCount+1)
}

// isAIQuestion はAIの応答が質問/確認を含むか検知
func isAIQuestion(response string) bool {
	// ツール呼び出しがある場合は質問とみなさない
	if len(tools.ParseToolCalls(response)) > 0 {
		return false
	}

	// 質問パターンの検出
	questionPatterns := []string{
		"続行しますか",
		"よろしいですか",
		"確認してください",
		"どうしますか",
		"選択してください",
		"指定してください",
		"教えてください",
		"Should I",
		"Do you want",
		"Would you like",
		"Shall I",
		"Can you confirm",
		"Please confirm",
		"proceed?",
		"continue?",
	}

	lowered := strings.ToLower(response)
	for _, pattern := range questionPatterns {
		if strings.Contains(lowered, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// executeStep は単一ステップを実行（リトライなし）
func (a *Agent) executeStep(ctx context.Context, plan *Plan, step *PlanStep) error {
	return a.executeStepWithRetry(ctx, plan, step, 0)
}

// executeStepWithRetry は単一ステップを実行（リトライカウント付き）
func (a *Agent) executeStepWithRetry(ctx context.Context, plan *Plan, step *PlanStep, retryCount int) error {
	if retryCount == 0 {
		cyan.Printf("\n[%d/%d] %s\n", step.ID, len(plan.Steps), step.Description)
	}

	// ステータスを実行中に更新
	plan.UpdateStatus(step.ID, "running", "")

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

	// リトライ時は履歴に追加しない（retryStepWithFix で追加済み）
	if retryCount == 0 {
		a.History = append(a.History, api.Message{Role: "user", Content: stepPrompt})
	}

	response, err := a.CurrentProvider.ChatWithTools(
		ctx,
		a.SystemPrompt,
		a.History,
		a.CurrentModel,
	)
	if err != nil {
		plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Error: %v", err))
		return fmt.Errorf("step %d failed: %w", step.ID, err)
	}

	// レスポンスを履歴に追加
	a.History = append(a.History, api.Message{Role: "assistant", Content: response})

	// 統計情報更新
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}

	// ツール呼び出しを処理
	maxToolCalls := 10 // 無限ループ防止
	maxContinues := 3  // 質問への自動続行回数制限
	continueCount := 0

	// 失敗検出用の変数
	var lastFailedResult string
	var lastFailReason string

	for i := 0; i < maxToolCalls; i++ {
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// ツール呼び出しなし

			// AIが質問している場合、自動続行を試みる
			if isAIQuestion(response) && continueCount < maxContinues {
				continueCount++
				yellow.Printf("⚠️  AI asked a question, auto-continuing (%d/%d)...\n", continueCount, maxContinues)

				// 続行を指示
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: "[AUTO-CONTINUE] Yes, proceed with the step. Execute the required tools directly without asking for confirmation.",
				})

				// 次のAI応答を取得
				response, err = a.CurrentProvider.ChatWithTools(
					ctx,
					a.SystemPrompt,
					a.History,
					a.CurrentModel,
				)
				if err != nil {
					plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Error: %v", err))
					return fmt.Errorf("step %d failed: %w", step.ID, err)
				}

				a.History = append(a.History, api.Message{Role: "assistant", Content: response})
				if a.Stats != nil {
					a.Stats.AssistantMessages++
				}
				continue
			}

			// ステップ完了
			break
		}

		// レスポンスから説明部分とツール呼び出しを分離
		explanation, _ := extractExplanationAndTool(response)

		// 説明部分を先に表示
		if explanation != "" {
			cyan.Println("\n💭 AI Explanation:")
			fmt.Println(explanation)
			fmt.Println()
		}

		// 複数ツールを順次実行
		var allResults []string
		for idx, toolCall := range toolCalls {
			// 複数ツールの場合は番号を表示
			if len(toolCalls) > 1 {
				cyan.Printf("🔧 Tool %d/%d: %s\n", idx+1, len(toolCalls), toolCall.Tool)
			}

			// 統計情報更新: ツール実行回数
			if a.Stats != nil {
				a.Stats.AddToolExecution(toolCall.Tool)
			}

			// ツール実行数をインクリメント
			plan.IncrementToolsExecuted(step.ID)

			// ツール実行
			result, change := tools.Execute(toolCall)
			allResults = append(allResults, fmt.Sprintf("[%s]\n%s", toolCall.Tool, result))

			// 失敗パターンをチェック
			if failed, reason := containsFailure(result); failed {
				lastFailedResult = result
				lastFailReason = reason
			}

			// 変更履歴を保存
			if change != nil {
				a.changeStack = append(a.changeStack, *change)
				if len(a.changeStack) > config.MaxChangeStack {
					a.changeStack = a.changeStack[1:]
				}

				// 永続的変更履歴に保存
				if a.changeStorage != nil && a.session != nil {
					if err := a.changeStorage.AppendChange(a.session.ID, *change); err != nil {
						yellow.Printf("Warning: Failed to persist change: %v\n", err)
					}
				}
			}
		}

		// 失敗検出時の処理
		if lastFailedResult != "" {
			// ステータスを更新して承認待ちに
			a.SetStatus(StateWaitingApproval, "Step failed - waiting for action", "ステップ失敗 - アクション待ち", "Choose r/s/a", "r/s/a を選択")

			action := promptFailureAction(step, lastFailedResult, lastFailReason)

			switch action {
			case FailureActionRetry:
				// AIにエラーを伝えて修正を依頼
				return a.retryStepWithFix(ctx, plan, step, lastFailedResult, "", retryCount)
			case FailureActionSkip:
				yellow.Printf("⏭️  Step %d skipped by user\n", step.ID)
				plan.UpdateStatus(step.ID, "skipped", fmt.Sprintf("Skipped: %s", lastFailReason))
				return nil
			case FailureActionAbort:
				red.Printf("🛑 Step %d aborted by user\n", step.ID)
				plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Aborted: %s", lastFailReason))
				return fmt.Errorf("step %d aborted by user: %s", step.ID, lastFailReason)
			}
		}

		// 結果を履歴に追加
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Results]\n%s", strings.Join(allResults, "\n\n")),
		})

		// 次のAI応答を取得
		response, err = a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Error: %v", err))
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		// レスポンスを履歴に追加
		a.History = append(a.History, api.Message{Role: "assistant", Content: response})

		// 統計情報更新
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}
	}

	// 完了判定の厳密化
	toolsExecuted := plan.GetToolsExecuted(step.ID)
	expectedTools := len(step.Tools)

	if expectedTools > 0 && toolsExecuted == 0 {
		// 期待されたツールが1つも実行されなかった
		yellow.Printf("⚠️  Warning: Step %d expected %d tools but executed %d\n", step.ID, expectedTools, toolsExecuted)
		plan.UpdateStatus(step.ID, "completed", fmt.Sprintf("Warning: No tools executed (expected: %d)", expectedTools))
	} else {
		plan.UpdateStatus(step.ID, "completed", fmt.Sprintf("Success (tools: %d)", toolsExecuted))
	}

	green.Printf("✓ Step %d completed (tools executed: %d)\n", step.ID, toolsExecuted)

	return nil
}

// executeParallelSteps は複数ステップを並列実行
func (a *Agent) executeParallelSteps(ctx context.Context, plan *Plan, stepIDs []int) error {
	cyan.Printf("\n🔀 Executing %d steps in parallel...\n", len(stepIDs))

	// 並列実行用のゴルーチンを起動
	// TODO: internal/agent/parallel.go で実装
	for _, stepID := range stepIDs {
		step := plan.GetStep(stepID)
		if step == nil {
			continue
		}
		if err := a.executeStep(ctx, plan, step); err != nil {
			return err
		}
	}

	return nil
}
