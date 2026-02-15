package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
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

// runCompletionHooks は config.yaml の hooks.on_completion に定義された
// シェルコマンドを順番に実行する。いずれかのコマンドが失敗した場合、
// needsContinue=true と AI 向けのフィードバックを返す。
// すべて成功した場合は needsContinue=false を返す。
func (a *Agent) runCompletionHooks(changedFiles []string) (needsContinue bool, feedback string) {
	cfg := config.GetGlobalConfig()
	hooks := cfg.Hooks.OnCompletion
	if len(hooks) == 0 {
		return false, ""
	}

	timeout := time.Duration(cfg.Hooks.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	// XELYON_CHANGED_FILES 環境変数を設定
	changedFilesEnv := strings.Join(changedFiles, " ")

	for _, cmd := range hooks {
		yellow.Printf("🏁 Running completion hook: %s\n", cmd)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		proc := exec.CommandContext(ctx, "bash", "-c", cmd)

		// 作業ディレクトリを設定
		if cwd, err := os.Getwd(); err == nil {
			proc.Dir = cwd
		}

		// 環境変数を設定
		proc.Env = append(os.Environ(), "XELYON_CHANGED_FILES="+changedFilesEnv)

		// stdout と stderr を結合して取得
		output, err := proc.CombinedOutput()
		cancel()

		if err != nil {
			// 出力を最大2000文字に切り詰め
			outputStr := string(output)
			if len(outputStr) > 2000 {
				outputStr = outputStr[:2000] + "\n... (truncated)"
			}

			// 終了コードを取得
			exitCode := -1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}

			red.Printf("  Hook failed (exit code %d): %s\n", exitCode, cmd)

			feedback = fmt.Sprintf(`[SYSTEM] Completion verification failed. Hook command %q failed (exit code %d):

%s

Please fix these errors before declaring completion. Do NOT skip these issues.`, cmd, exitCode, outputStr)

			return true, feedback
		}

		green.Printf("  Hook passed: %s\n", cmd)
	}

	return false, ""
}

// runCompletionHooksWithRetry は completion hooks を最大 MaxRetry 回実行する。
// フック失敗時は AI にフィードバックして修正を試み、再実行する。
// Plan mode での使用を想定（ループ型の runNormalMode では直接カウンターを使用）。
// 戻り値: hooks がすべてパスした場合 true、max_retry 到達で打ち切った場合 false。
func (a *Agent) runCompletionHooksWithRetry(ctx context.Context) bool {
	cfg := config.GetGlobalConfig()
	if len(cfg.Hooks.OnCompletion) == 0 {
		return true
	}

	changedFiles := a.getTaskChangedFiles()
	if len(changedFiles) == 0 {
		return true
	}

	maxRetry := cfg.Hooks.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 3
	}

	for attempt := 1; attempt <= maxRetry; attempt++ {
		needsContinue, feedback := a.runCompletionHooks(changedFiles)
		if !needsContinue {
			return true // all hooks passed
		}

		if attempt >= maxRetry {
			yellow.Printf("⚠️  Hook retry limit reached (%d/%d). Proceeding with completion.\n", attempt, maxRetry)
			return false
		}

		// AI にフィードバックして修正を試みる
		yellow.Printf("⚠️  Completion hook failed (%d/%d). Asking AI to fix...\n", attempt, maxRetry)
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: feedback,
		})

		response, err := a.CurrentProvider.ChatWithTools(ctx, a.SystemPrompt, a.History, a.CurrentModel)
		if err != nil {
			yellow.Printf("⚠️  AI fix attempt failed: %v\n", err)
			return false
		}

		a.History = append(a.History, api.Message{
			Role:             "assistant",
			Content:          response,
			ReasoningContent: a.getLastReasoningContent(),
		})

		// ツール呼び出しがあれば実行
		toolCalls := tools.ParseToolCalls(response)
		for _, tc := range toolCalls {
			a.executeToolCallWithResult(response, tc)
		}

		// 変更ファイルを更新（修正で新しいファイルが変わる可能性）
		changedFiles = a.getTaskChangedFiles()
		if len(changedFiles) == 0 {
			return true
		}
	}

	return false
}
