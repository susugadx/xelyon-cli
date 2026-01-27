package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// promptFailureAction は失敗時のユーザー選択UIを表示
// 戻り値: (アクション, コメント文字列) - コメントアクション時のみ第2引数が非空
func promptFailureAction(step *plan.PlanStep, result string, reason string) (plan.FailureAction, string) {
	fmt.Println()
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	red.Printf("❌ Step %d Failed: %s\n", step.ID, step.Description)
	red.Printf("   Reason: %s\n", reason)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// エラー内容を表示
	lines := strings.Split(result, "\n")
	maxShow := config.ErrorOutputMaxLines
	if len(lines) > maxShow {
		fmt.Println()
		yellow.Printf("Error output (last %d lines):\n", maxShow)
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
// 戻り値: (アクション, コメント文字列) - コメントアクション時のみ第2引数が非空
func promptFailureActionInput() (plan.FailureAction, string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		yellow.Print("Choose action [r/c/s/a]: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			// EOF時はabortを返して終了
			return plan.FailureActionAbort, ""
		}
		response = strings.ToLower(strings.TrimSpace(response))

		// 空入力は無視してリトライ
		if response == "" {
			continue
		}

		switch response {
		case "r", "retry":
			return plan.FailureActionRetry, ""
		case "c", "comment":
			comment, _ := common.ReadMultiLineComment(reader)
			return plan.FailureActionComment, comment
		case "s", "skip":
			return plan.FailureActionSkip, ""
		case "a", "abort":
			return plan.FailureActionAbort, ""
		default:
			yellow.Println("Invalid input. Please enter r/c/s/a.")
		}
	}
}

// promptFailureActionWithSelector は失敗時の Selector UI を表示
// autoRetryMax が > 0 の場合、自動リトライが exhausted されたことを表示
// 戻り値: (アクション, コメント文字列) - コメントアクション時のみ第2引数が非空
func promptFailureActionWithSelector(step *plan.PlanStep, result string, reason string, autoRetryMax int) (plan.FailureAction, string) {
	fmt.Println()
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 自動リトライが有効だった場合のメッセージ
	if autoRetryMax > 0 {
		red.Printf("❌ Step %d Failed (%d retries exhausted)\n", step.ID, autoRetryMax)
	} else {
		red.Printf("❌ Step %d Failed: %s\n", step.ID, step.Description)
	}
	red.Printf("   Reason: %s\n", reason)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// エラー内容を表示
	lines := strings.Split(result, "\n")
	maxShow := config.ErrorOutputMaxLines
	if len(lines) > maxShow {
		fmt.Println()
		yellow.Printf("Error output (last %d lines):\n", maxShow)
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
	cyan.Println("? What do you want to do?")
	fmt.Println()

	// Selector 風のオプション表示
	options := []ui.SelectOption{
		{Label: "Retry", Description: "Try again with different approach", Value: "retry"},
		{Label: "Comment", Description: "Give instructions", Value: "comment"},
		{Label: "Skip", Description: "Continue to next step", Value: "skip"},
		{Label: "Abort", Description: "Stop execution", Value: "abort"},
	}

	return runFailureSelectorUI(options)
}

// runFailureSelectorUI は失敗時の選択UIを実行
func runFailureSelectorUI(options []ui.SelectOption) (plan.FailureAction, string) {
	// オプションを表示
	for i, opt := range options {
		marker := "  "
		if i == 0 {
			marker = "▶ "
		}
		fmt.Printf("  %s%d. %-8s - %s\n", marker, i+1, opt.Label, opt.Description)
	}

	// ヒント表示
	fmt.Println()
	dim.Printf("  (Enter=Retry, 2/c=Comment, 3/s=Skip, 4/a=Abort)\n")
	fmt.Println()

	// 入力を受け付け
	reader := bufio.NewReader(os.Stdin)
	for {
		cyan.Print("Choice [1]: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return plan.FailureActionAbort, ""
		}
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "", "1", "r", "retry":
			green.Println("✓ Retry")
			return plan.FailureActionRetry, ""
		case "2", "c", "comment":
			green.Println("✓ Comment")
			comment, _ := common.ReadMultiLineComment(reader)
			return plan.FailureActionComment, comment
		case "3", "s", "skip":
			green.Println("✓ Skip")
			return plan.FailureActionSkip, ""
		case "4", "a", "abort":
			green.Println("✓ Abort")
			return plan.FailureActionAbort, ""
		default:
			yellow.Println("Invalid input. Please enter 1/2/3/4 or r/c/s/a.")
		}
	}
}
