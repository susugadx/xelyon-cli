package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
