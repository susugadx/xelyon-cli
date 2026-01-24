package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
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
