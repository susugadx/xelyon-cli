package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"golang.org/x/sync/errgroup"
)

// RunPlanMode は Claude Code 風の Plan Mode を実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 実装ツール(SafetyMedium/Low)の前に計画を生成・承認
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	// 実装前チェック：既存定義の重複を警告
	if warning := CheckBeforeImplementation(userRequest); warning != "" {
		yellow.Println(warning)
		userRequest = userRequest + "\n\n[SYSTEM NOTE: " + warning + " Please check existing code before creating new definitions.]"
	}

	// Step 1: 調査フェーズ（SafetyHighツールを自由に実行）
	cyan.Println("\n🔍 Investigation phase - researching the codebase...")
	a.SetStatus(StateRunning, "Investigating", "調査中", "Wait for investigation", "調査完了を待ってください")

	// ユーザーリクエストを履歴に追加
	investigationPrompt := fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

IMPORTANT RULES:
1. First, investigate the codebase to understand what needs to be done
2. You CAN use read-only tools freely: read_file, search_file, search_code, list_dir, git_status, git_log, git_diff, lint, test, web_search
3. Do NOT use any modification tools yet (write_file, str_replace, delete_file, bash, etc.)
4. When you have gathered enough information, output a plan for the implementation

After investigation, output your plan in this JSON format:
{"plan": {
  "summary": "Brief summary of what will be done",
  "steps": [
    {"id": 1, "description": "Step description", "tools": ["tool1"]},
    {"id": 2, "description": "Step description", "tools": ["tool2"]}
  ]
}}

Start your investigation now. Use read_file, search_code, list_dir etc. to understand the codebase.`, userRequest)

	a.History = append(a.History, api.Message{Role: "user", Content: investigationPrompt})

	// 調査フェーズ: SafetyHighツールを実行
	planJSON, err := a.runInvestigationPhase(ctx)
	if err != nil {
		return err
	}

	// 計画が空の場合（調査のみで完了、または実装不要）
	if planJSON == "" {
		green.Println("\n✓ Investigation complete. No implementation needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	// Step 2: 計画をパースして表示
	plan, err := ParsePlan(planJSON)
	if err != nil {
		red.Printf("❌ Failed to parse plan: %v\n", err)
		yellow.Printf("Plan JSON:\n%s\n", planJSON)
		return err
	}

	if len(plan.Steps) == 0 {
		green.Println("\n✓ Investigation complete. No implementation steps needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	// 計画を表示
	fmt.Println()
	cyan.Println("📋 Implementation Plan:")
	cyan.Println(strings.Repeat("─", 50))
	if plan.Summary != "" {
		fmt.Printf("Summary: %s\n\n", plan.Summary)
	}
	for _, step := range plan.Steps {
		fmt.Printf("  %d. %s\n", step.ID, step.Description)
		if len(step.Tools) > 0 {
			fmt.Printf("     Tools: %s\n", strings.Join(step.Tools, ", "))
		}
	}
	cyan.Println(strings.Repeat("─", 50))
	fmt.Println()

	// Step 3: ユーザー承認
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

	green.Println("✓ Plan approved. Starting implementation...")
	a.SetStatus(StateRunning, "Implementing", "実装中", "Wait for completion", "完了を待ってください")

	// Step 4: 実装フェーズ
	err = a.runImplementationPhase(ctx, plan)
	if err != nil {
		a.SetStatus(StateAborted, "Implementation failed", "実装に失敗", "Review errors and retry", "エラーを確認して再試行")
		return err
	}

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
	return nil
}

// runInvestigationPhase は調査フェーズを実行
// SafetyHighツールのみを実行し、計画JSONを返す
func (a *Agent) runInvestigationPhase(ctx context.Context) (string, error) {
	maxIterations := config.MaxToolIterations
	var lastToolCall *tools.ToolCall
	var sameCallCount int
	var lastToolCallsHash string // 並列実行時のループ検知用
	var sameSetCount int         // 並列実行時の同一セットカウント

	for i := 0; i < maxIterations; i++ {
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			return "", fmt.Errorf("investigation failed: %w", err)
		}

		a.History = append(a.History, api.Message{Role: "assistant", Content: response})
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}

		// 計画JSONを検出
		if planJSON := ExtractPlanJSON(response); planJSON != "" {
			return planJSON, nil
		}

		// デバッグモード: レスポンスの診断情報を出力
		if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] response length: %d\n", len(response))
			// ツールパターンの存在チェック
			if idx := strings.Index(response, `{"tool"`); idx != -1 {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] found {\"tool\" at index %d\n", idx)
			}
			if idx := strings.Index(response, `{ "tool"`); idx != -1 {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] found { \"tool\" at index %d\n", idx)
			}
			// コードブロックの数をチェック
			codeBlockOpens := strings.Count(response, "```")
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] code block markers (```): %d (odd = unclosed)\n", codeBlockOpens)
			// 閉じていないJSONの可能性をチェック
			openBraces := strings.Count(response, "{")
			closeBraces := strings.Count(response, "}")
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] braces: { = %d, } = %d (mismatch = incomplete JSON)\n", openBraces, closeBraces)
		}

		// ツール呼び出しチェック
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// デバッグ: なぜ0件なのか追加診断
			if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] ParseToolCalls returned 0 tools\n")
				// ツールパターンがあるのに0件の場合、原因を調査
				if strings.Contains(response, `{"tool"`) || strings.Contains(response, `{ "tool"`) {
					fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] WARNING: tool pattern exists but not parsed!\n")
					// 最後の200文字を表示
					if len(response) > 200 {
						fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] tail: ...%s\n", response[len(response)-200:])
					}
				}
			}
			// ツール呼び出しも計画もない場合は終了
			// AIが調査を終えて単に説明しているか、質問している場合
			fmt.Println(response)
			return "", nil
		}

		// ツールを分類
		var safetyHighTools []*tools.ToolCall
		var otherTools []*tools.ToolCall

		for _, toolCall := range toolCalls {
			safety := tools.GetToolSafety(toolCall.Tool)
			if safety == tools.SafetyHigh {
				safetyHighTools = append(safetyHighTools, toolCall)
			} else {
				otherTools = append(otherTools, toolCall)
			}
		}

		// SafetyMedium/Low がある場合は計画生成を要求（従来通り）
		if len(otherTools) > 0 {
			tc := otherTools[0]
			cyan.Printf("\n⚡ Implementation tool detected: %s\n", tc.Tool)
			cyan.Println("   Requesting implementation plan...")

			a.History = append(a.History, api.Message{
				Role: "user",
				Content: fmt.Sprintf(`[SYSTEM] You tried to use a modification tool (%s) during the investigation phase.

Before using modification tools, you must output an implementation plan.
Output your plan now in this JSON format:

{"plan": {
  "summary": "Brief summary of what will be done",
  "steps": [
    {"id": 1, "description": "Step description", "tools": ["tool1"]},
    {"id": 2, "description": "Step description", "tools": ["tool2"]}
  ]
}}`, tc.Tool),
			})
			continue
		}

		// SafetyHigh ツールがない場合（ツールなし）= 調査完了
		if len(safetyHighTools) == 0 {
			fmt.Println(response)
			return "", nil
		}

		// SafetyHigh ツールを実行
		var allResults []string
		if len(safetyHighTools) > 1 {
			// 並列実行時のループ検知
			currentHash := hashToolCalls(safetyHighTools)
			if currentHash == lastToolCallsHash {
				sameSetCount++
				if sameSetCount >= 3 {
					return "", fmt.Errorf("tool loop detected: same tool set repeated %d times", sameSetCount)
				}
			} else {
				sameSetCount = 0
			}
			lastToolCallsHash = currentHash

			// 並列実行
			cyan.Printf("⚡ Executing %d investigation tools in parallel...\n", len(safetyHighTools))
			allResults = a.executeInvestigationToolsParallel(ctx, safetyHighTools)
		} else {
			// 単一ツールは直列実行
			tc := safetyHighTools[0]

			// ループ検知
			if a.shouldAbortToolLoop(tc, lastToolCall, &sameCallCount) {
				return "", fmt.Errorf("tool loop detected during investigation")
			}
			lastToolCall = tc

			if a.Stats != nil {
				a.Stats.AddToolExecution(tc.Tool)
			}
			result, _ := tools.Execute(tc)
			allResults = []string{fmt.Sprintf("[Tool Result for %s]\n%s", tc.Tool, result)}
		}

		// 結果を履歴に追加
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: strings.Join(allResults, "\n\n"),
		})
	}

	yellow.Printf("⚠️  調査フェーズが%d回のツール実行に達しました。続けて指示してください。\n", maxIterations)
	return "", nil
}

// runImplementationPhase は実装フェーズを実行（並列実行対応）
func (a *Agent) runImplementationPhase(ctx context.Context, plan *Plan) error {
	for {
		// 並列実行可能なステップを取得
		parallelSteps := plan.GetParallelSteps()

		if len(parallelSteps) > 1 {
			// 並列実行
			cyan.Printf("\n⚡ Executing %d steps in parallel...\n", len(parallelSteps))
			if err := a.executeStepsParallel(ctx, plan, parallelSteps); err != nil {
				return err
			}
			// 完了したステップをマーク
			for _, id := range parallelSteps {
				plan.UpdateStatus(id, "completed", "")
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

// hashToolCalls は toolCalls セットのハッシュを生成（ループ検知用）
func hashToolCalls(toolCalls []*tools.ToolCall) string {
	var parts []string
	for _, tc := range toolCalls {
		argsStr := fmt.Sprintf("%v", tc.Args)
		parts = append(parts, tc.Tool+":"+argsStr)
	}
	sort.Strings(parts) // 順序に依存しないようソート
	return strings.Join(parts, "|")
}

// executeInvestigationToolsParallel は SafetyHigh ツールを並列実行
// 結果は toolCall 順で返す（順序保持）
func (a *Agent) executeInvestigationToolsParallel(ctx context.Context, toolCalls []*tools.ToolCall) []string {
	cfg := config.GetGlobalConfig()
	maxWorkers := cfg.PlanMode.MaxParallelSteps
	if maxWorkers <= 0 {
		maxWorkers = 3
	}

	results := make([]string, len(toolCalls))
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, toolCall *tools.ToolCall) {
			defer wg.Done()

			// セマフォ取得
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[idx] = fmt.Sprintf("[%s]\nError: context cancelled", toolCall.Tool)
				return
			}

			// Stats 更新（スレッドセーフ）
			a.incrementToolExecution(toolCall.Tool)

			// ツール実行
			result, _ := tools.Execute(toolCall)
			results[idx] = fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result)
		}(i, tc)
	}

	wg.Wait()
	return results
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


// containsFailure はツール結果に失敗パターンが含まれるか検出
// 失敗を検出した場合、(true, 理由) を返す
//
// NOTE: "error:" や "Error:" のような汎用パターンは使用しない。
// コード検索結果（例: t.Errorf）やログ出力に含まれる "Error" 文字列で
// 誤検知してしまうため。実際のコマンド失敗を示す具体的なパターンのみ使用。
func containsFailure(result string) (bool, string) {
	failPatterns := map[string]string{
		// Go test failures
		"--- FAIL:": "Go test failed",
		"FAIL\t":    "Go test failed",
		// Command failures (exit code)
		"exit status 1": "Command failed with exit code 1",
		// Panics and fatal errors
		"panic:":       "Panic detected",
		"fatal error:": "Fatal error",
		// Build/compile errors
		"compile error":  "Compilation error",
		"build failed":   "Build failed",
		"cannot find":    "Build error",
		"undefined:":     "Undefined symbol",
		"undeclared":     "Undeclared identifier",
		"does not exist": "File or module not found",
		// npm/node errors
		"npm ERR!": "npm error",
		// JavaScript runtime errors (specific patterns)
		"SyntaxError:":    "Syntax error",
		"TypeError:":      "Type error",
		"ReferenceError:": "Reference error",
		// Python errors
		"Traceback (most recent call last):": "Python exception",
		"AssertionError:":                    "Assertion failed",
		// Rust errors
		"error[E": "Rust compilation error",
	}

	// パターンの優先度順にチェック（より具体的なものを先に）
	priorityPatterns := []string{
		"--- FAIL:",
		"FAIL\t",
		"panic:",
		"fatal error:",
		"Traceback (most recent call last):",
		"error[E",
		"compile error",
		"build failed",
		"undefined:",
		"undeclared",
		"cannot find",
		"does not exist",
		"npm ERR!",
		"SyntaxError:",
		"TypeError:",
		"ReferenceError:",
		"AssertionError:",
		"exit status 1",
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
	FailureActionRetry   FailureAction = "retry"
	FailureActionComment FailureAction = "comment"
	FailureActionSkip    FailureAction = "skip"
	FailureActionAbort   FailureAction = "abort"
)

// failureComment はコメントアクション時のユーザー入力を保持
var failureComment string

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
	yellow.Println("  [r]etry   - AI will analyze the error and try to fix it")
	yellow.Println("  [c]omment - Give instructions for how to fix it")
	yellow.Println("  [s]kip    - Mark as skipped and continue to next step")
	yellow.Println("  [a]bort   - Stop plan execution")
	fmt.Println()

	// r/c/s/a 専用の入力を受け付け
	return promptFailureActionInput()
}

// promptFailureActionInput は r/c/s/a 専用の入力プロンプト
func promptFailureActionInput() FailureAction {
	reader := bufio.NewReader(os.Stdin)

	for {
		yellow.Print("Choose action [r/c/s/a]: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			// EOF時はabortを返して終了
			return FailureActionAbort
		}
		response = strings.ToLower(strings.TrimSpace(response))

		// 空入力は無視してリトライ
		if response == "" {
			continue
		}

		switch response {
		case "r", "retry":
			return FailureActionRetry
		case "c", "comment":
			failureComment, _ = tools.ReadMultiLineComment(reader)
			return FailureActionComment
		case "s", "skip":
			return FailureActionSkip
		case "a", "abort":
			return FailureActionAbort
		default:
			yellow.Println("Invalid input. Please enter r/c/s/a.")
		}
	}
}
