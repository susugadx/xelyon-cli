package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
