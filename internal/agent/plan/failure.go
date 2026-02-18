package plan

import "strings"

// FailureAction は失敗時のユーザー選択
type FailureAction string

const (
	FailureActionRetry   FailureAction = "retry"
	FailureActionComment FailureAction = "comment"
	FailureActionSkip    FailureAction = "skip"
	FailureActionAbort   FailureAction = "abort"
)

// ContainsFailure はツール結果に失敗パターンが含まれるか検出
// 失敗を検出した場合、(true, 理由) を返す
//
// NOTE: "error:" や "Error:" のような汎用パターンは使用しない。
// コード検索結果（例: t.Errorf）やログ出力に含まれる "Error" 文字列で
// 誤検知してしまうため。実際のコマンド失敗を示す具体的なパターンのみ使用。
func ContainsFailure(result string) (bool, string) {
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
		// Read-Before-Write guard errors
		"You must read_file before": "Read-before-write guard: file not read",
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
		"You must read_file before",
		"exit status 1",
	}

	for _, pattern := range priorityPatterns {
		if strings.Contains(result, pattern) {
			return true, failPatterns[pattern]
		}
	}
	return false, ""
}
