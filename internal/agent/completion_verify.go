package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
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
		if len(change.Details) > 0 {
			for _, detail := range change.Details {
				if detail.FilePath != "" && !seen[detail.FilePath] {
					seen[detail.FilePath] = true
					files = append(files, detail.FilePath)
				}
			}
			continue
		}
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

	result := toolslsp.CheckDiagnosticsForFilesWithClient(a.GetLSPClient(), changedFiles)
	if !result.HasErrors {
		return false, ""
	}

	feedback = fmt.Sprintf(`[SYSTEM] Completion verification failed. LSP diagnostics found %d error(s) in modified files:

%s

Please fix these errors before declaring completion. Do NOT skip these issues.`, result.ErrorCount, result.Summary)

	return true, feedback
}

// checkGitDiffEmpty は git diff --stat を実行し、差分が空の場合に警告を返す。
// runCompletionHooks とは独立して呼び出し元でチェックする。
// タスク中のコミットにも対応している。
func (a *Agent) checkGitDiffEmpty() (needsContinue bool, feedback string) {
	out := a.output()

	// タスク中にコミットがあったかチェック
	var currentHash string
	if out, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		currentHash = strings.TrimSpace(string(out))
	}

	if a.taskBaseCommitHash != "" && currentHash != "" && currentHash != a.taskBaseCommitHash {
		// タスク中にコミットが行われている場合は diff empty チェックをスキップ（表示のみ）
		yellow.Fprintln(out, "📊 Changes already committed during this task.")
		diffCmd := exec.Command("git", "diff", a.taskBaseCommitHash, "HEAD", "--stat")
		if output, err := diffCmd.CombinedOutput(); err == nil && len(output) > 0 {
			_, _ = fmt.Fprintln(out, string(output))
		}
		return false, ""
	}

	// tracked changes
	diffCmd := exec.Command("git", "diff", "--stat")
	output, err := diffCmd.CombinedOutput()
	hasDiff := err == nil && strings.TrimSpace(string(output)) != ""

	// untracked files（write_file で新規作成されたファイル用）
	untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	untrackedOut, err2 := untrackedCmd.CombinedOutput()
	hasUntracked := err2 == nil && strings.TrimSpace(string(untrackedOut)) != ""

	if !hasDiff && !hasUntracked {
		yellow.Fprintln(out, "⚠️  WARNING: No changes detected by git diff.")
		return true, "[SYSTEM] WARNING: You declared completion but git diff shows NO changes. Did you actually make the required modifications? Review your plan and ensure all steps are implemented."
	}
	return false, ""
}

// runCompletionHooks は config.yaml の hooks.on_completion に定義された
// シェルコマンドを順番に実行する。いずれかのコマンドが失敗した場合、
// needsContinue=true と AI 向けのフィードバックを返す。
// すべて成功した場合は needsContinue=false を返す。
func (a *Agent) runCompletionHooks(changedFiles []string) (needsContinue bool, feedback string) {
	cfg := a.cfg()
	hooks := cfg.Hooks.OnCompletion
	out := a.output()

	// No user hooks → nothing to verify
	if len(hooks) == 0 {
		return false, ""
	}

	// git diff --stat: hook 失敗時のコンテキスト用
	yellow.Fprintln(out, "📊 Verifying changes with git diff --stat...")
	var diffOutput string
	diffCmd := exec.Command("git", "diff", "--stat")
	if output, err := diffCmd.CombinedOutput(); err == nil {
		diffOutput = string(output)
		if strings.TrimSpace(diffOutput) != "" {
			_, _ = fmt.Fprintln(out, diffOutput)
		}
	}

	// untracked files（write_file で新規作成されたファイル用）
	untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	if untrackedOut, err := untrackedCmd.CombinedOutput(); err == nil {
		untrackedStr := strings.TrimSpace(string(untrackedOut))
		if untrackedStr != "" {
			_, _ = fmt.Fprintf(out, "New files (untracked):\n%s\n", untrackedStr)
			diffOutput += "\nNew files (untracked):\n" + untrackedStr
		}
	}

	// User defined hooks

	timeout := time.Duration(cfg.Hooks.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	// XELYON_CHANGED_FILES 環境変数を設定
	changedFilesEnv := strings.Join(changedFiles, " ")

	for _, cmd := range hooks {
		yellow.Fprintf(out, "🏁 Running completion hook: %s\n", cmd)

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

			red.Fprintf(out, "  Hook failed (exit code %d): %s\n", exitCode, cmd)

			feedback = fmt.Sprintf(`[SYSTEM] Completion verification failed. Hook command %q failed (exit code %d):

%s

[Context] git diff --stat:
%s

Please fix these errors before declaring completion. Do NOT skip these issues.`, cmd, exitCode, outputStr, diffOutput)

			return true, feedback
		}

		green.Fprintf(out, "  Hook passed: %s\n", cmd)
	}

	return false, ""
}

// addPendingLSPFile は str_replace 成功後に対象ファイルを遅延診断バッファへ追加する。
// 重複ファイルは追加しない（連続 str_replace で同一ファイルを複数回編集した場合も1エントリ）。
func (a *Agent) addPendingLSPFile(path string) {
	if path == "" {
		return
	}
	for _, f := range a.pendingLSPFiles {
		if f == path {
			return
		}
	}
	a.pendingLSPFiles = append(a.pendingLSPFiles, path)
}

// flushLSPDiagnostics はバッファ内の全ファイルに対して LSP 診断を実行し、
// 結果文字列を返してバッファをクリアする。
// エラーがなければ空文字を返す。LSP 未起動時も空文字を返す（graceful degradation）。
func (a *Agent) flushLSPDiagnostics() string {
	if len(a.pendingLSPFiles) == 0 {
		return ""
	}
	files := a.pendingLSPFiles
	a.pendingLSPFiles = nil

	result := toolslsp.CheckDiagnosticsForFilesWithClient(a.GetLSPClient(), files)
	if result.Summary == "" {
		return ""
	}
	return "\n\n⚠️ LSP Diagnostics (deferred):\n" + result.Summary
}

// runCompletionHooksWithRetry は completion hooks を最大 MaxRetry 回実行する。
// フック失敗時は AI にフィードバックして修正を試み、再実行する。
// Plan mode での使用を想定（ループ型の runNormalMode では直接カウンターを使用）。
// 戻り値: hooks がすべてパスした場合 true、max_retry 到達で打ち切った場合 false。
func (a *Agent) runCompletionHooksWithRetry(ctx context.Context) bool {
	cfg := a.cfg()
	out := a.output()

	// No hooks configured → nothing to verify
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
			yellow.Fprintf(out, "⚠️  Hook retry limit reached (%d/%d). Proceeding with completion.\n", attempt, maxRetry)
			return false
		}

		// AI にフィードバックして修正を試みる
		yellow.Fprintf(out, "⚠️  Completion hook failed (%d/%d). Asking AI to fix...\n", attempt, maxRetry)
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: feedback,
		})

		response, err := a.CurrentProvider.ChatWithTools(a.requestContext(ctx), a.SystemPrompt, a.History, a.CurrentModel)
		if err != nil {
			yellow.Fprintf(out, "⚠️  AI fix attempt failed: %v\n", err)
			return false
		}

		// ツール呼び出しをパースして履歴管理
		toolCalls := a.parseToolCalls(response)

		// FC rescue: テキストから抽出された toolCall にダミー ID を注入
		for i, tc := range toolCalls {
			if tc.ID == "" {
				toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
			}
		}

		if len(toolCalls) > 0 {
			execToolCalls := toolCalls

			// バッチで assistant メッセージを追加し、ツールを実行
			a.addToolCallsToHistory(response, execToolCalls)
			for _, tc := range execToolCalls {
				a.executeToolOnly(tc)
			}
		} else {
			// テキストのみの応答
			a.History = append(a.History, api.Message{
				Role:             "assistant",
				Content:          response,
				ReasoningContent: a.getLastReasoningContent(),
			})
		}

		// 変更ファイルを更新（修正で新しいファイルが変わる可能性）
		changedFiles = a.getTaskChangedFiles()
		if len(changedFiles) == 0 {
			return true
		}
	}

	return false
}
