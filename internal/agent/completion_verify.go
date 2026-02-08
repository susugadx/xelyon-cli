package agent

import (
	"fmt"
	"strings"

	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
)

// 英語完了宣言パターン（case-insensitive で検索）
var completionPatternsEnglish = []string{
	"done",
	"completed",
	"finished",
	"all done",
	"that's it",
	"task complete",
	"all set",
	"changes are complete",
	"implementation is complete",
}

// 日本語完了宣言パターン（そのまま検索）
var completionPatternsJapanese = []string{
	"完了",
	"修正しました",
	"以上です",
	"実装しました",
	"対応しました",
	"修正完了",
	"作業は以上",
	"変更は以上",
}

// containsCompletionDeclaration はAIレスポンスに完了宣言パターンが含まれるかを検出する。
// 英語パターンはcase-insensitive、日本語パターンはそのまま検出。
func containsCompletionDeclaration(response string) bool {
	lowered := strings.ToLower(response)

	for _, pattern := range completionPatternsEnglish {
		if strings.Contains(lowered, pattern) {
			return true
		}
	}

	for _, pattern := range completionPatternsJapanese {
		if strings.Contains(response, pattern) {
			return true
		}
	}

	return false
}

// getTaskChangedFiles は現在タスクで変更されたファイルの一覧を返す（重複排除済み）。
// a.changeStack[a.taskChangeOffset:] から一意なファイルパスを抽出する。
func (a *Agent) getTaskChangedFiles() []string {
	taskChanges := a.changeStack[a.taskChangeOffset:]
	if len(taskChanges) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(taskChanges))
	var files []string
	for _, change := range taskChanges {
		if change.FilePath != "" && !seen[change.FilePath] {
			seen[change.FilePath] = true
			files = append(files, change.FilePath)
		}
	}
	return files
}

// verifyCompletionWithDiagnostics は完了宣言時にLSP診断を実行し、
// エラーがあればフィードバックメッセージを返す。
// needsContinue=true の場合、呼び出し元はループを continue すべき。
// LSP未起動時は何もせず needsContinue=false を返す（graceful degradation）。
func (a *Agent) verifyCompletionWithDiagnostics(response string) (needsContinue bool, feedback string) {
	if !containsCompletionDeclaration(response) {
		return false, ""
	}

	changedFiles := a.getTaskChangedFiles()
	if len(changedFiles) == 0 {
		return false, ""
	}

	result := toolslsp.CheckDiagnosticsForFiles(changedFiles)
	if !result.HasErrors {
		return false, ""
	}

	feedback = fmt.Sprintf(`[SYSTEM] Completion verification failed. LSP diagnostics found %d error(s) in modified files:

%s

Please fix these errors before declaring completion. Do NOT skip these issues.`, result.ErrorCount, result.Summary)

	return true, feedback
}
