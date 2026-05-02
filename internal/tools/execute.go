package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// printToolArgs はツールの引数を簡潔に表示する（Execute/PreviewToolCallで共通使用）
func printToolArgs(w io.Writer, tc *ToolCall) {
	if tc != nil {
		if printer, ok := toolArgPreviewPrinters[tc.Tool]; ok {
			printer(w, tc)
			_, _ = fmt.Fprintln(w)
			return
		}
	}
	printGenericToolArgs(w, tc)
	_, _ = fmt.Fprintln(w)
}

func formatReadFilePreviewArg(args map[string]string) string {
	paths := previewReadFilePaths(args)
	switch len(paths) {
	case 0:
		return "Files: (none)"
	case 1:
		return "File: " + paths[0]
	default:
		return fmt.Sprintf("Files: %d", len(paths))
	}
}

func previewReadFilePaths(args map[string]string) []string {
	if rawPaths := strings.TrimSpace(args["paths"]); rawPaths != "" {
		var paths []string
		if err := json.Unmarshal([]byte(rawPaths), &paths); err == nil {
			return paths
		}
	}
	if path := strings.TrimSpace(args["path"]); path != "" {
		return []string{path}
	}
	return nil
}

// ExecutionResult はツール実行結果と、結果表示に必要な実行メタデータを保持する。
type ExecutionResult struct {
	Result   string
	Change   *FileChange
	Duration time.Duration
	Error    bool
}

// ExecuteWithContext は実行コンテキスト付きでツールを実行し、結果を公開する。
func ExecuteWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	execResult := ExecuteUnpublishedWithContext(execCtx, tc)
	PublishResultWithContext(execCtx, tc, execResult)
	return execResult.Result, execResult.Change
}

// ExecuteUnpublishedWithContext はツールを実行し、wrapper 層の結果表示は行わない。
func ExecuteUnpublishedWithContext(execCtx ExecutionContext, tc *ToolCall) ExecutionResult {
	execCtx = normalizeExecutionContext(execCtx)
	startTime := time.Now()
	result, change, isError := executeCoreWithContext(execCtx, tc)
	elapsed := time.Since(startTime)

	return ExecutionResult{
		Result:   result,
		Change:   change,
		Duration: elapsed,
		Error:    isError,
	}
}

// ExecuteQuietWithContext は実行コンテキスト付きでツールを実行するが、wrapper 出力を抑制する。
func ExecuteQuietWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	execResult := ExecuteQuietUnpublishedWithContext(execCtx, tc)
	return execResult.Result, execResult.Change
}

// ExecuteQuietUnpublishedWithContext は quiet mode でツールを実行し、wrapper 層の結果表示は行わない。
func ExecuteQuietUnpublishedWithContext(execCtx ExecutionContext, tc *ToolCall) ExecutionResult {
	execCtx = normalizeExecutionContext(execCtx)
	restoreQuiet := common.PushQuietMode()
	defer restoreQuiet()
	return ExecuteUnpublishedWithContext(execCtx, tc)
}

// PublishResultWithContext はツール実行済みの結果を wrapper/TUI 層へ公開する。
func PublishResultWithContext(execCtx ExecutionContext, tc *ToolCall, execResult ExecutionResult) {
	execCtx = normalizeExecutionContext(execCtx)

	// ツール実行完了後、結果表示前にスピナーを停止してクリア
	// （wait_agent のような長時間ブロックツールで表示が混ざるのを防ぐ）
	if execCtx.Runtime != nil {
		execCtx.Runtime.StopSpinner()
	}

	result := execResult.Result
	isError := execResult.Error || IsErrorResult(result)

	if execCtx.ToolResultCallback != nil {
		// TUIモード: 構造化データをコールバックで送信
		execCtx.ToolResultCallback(ToolResultInfo{
			ToolName: tc.Tool,
			Args:     tc.Args,
			Result:   result,
			Error:    isError,
			Duration: execResult.Duration,
		})
	} else {
		// 従来モード: stdoutにテキスト出力
		_, _ = fmt.Fprintln(execCtx.Stdout, ui.FormatToolLine(ui.ToolDisplayInfo{
			ToolName: tc.Tool,
			Args:     tc.Args,
			Result:   result,
			Error:    isError,
		}))

		// ツール出力の折りたたみ表示（bashはストリーミング表示済みなので除外）
		if !isStreamingTool(tc.Tool) && shouldShowCollapsedOutput(result) {
			displayCollapsedOutput(execCtx.Stdout, result, execCtx.EffectiveConfig())
		}
	}
}

func executeCoreWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange, bool) {
	if err := execCtx.EffectiveContext().Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return "Error: context cancelled", nil, true
		}
		return fmt.Sprintf("Error: %v", err), nil, true
	}

	// デフォルト値の設定（Registry実行前）
	// list_dir, git_addでpathが空の場合"."を設定
	if tc.Args["path"] == "" {
		switch tc.Tool {
		case "list_dir", "git_add":
			tc.Args["path"] = "."
		}
	}

	// Registry経由でツール実行
	result, change := execCtx.EffectiveRegistry().ExecuteWithContext(execCtx, tc)

	// ファイル変更系ツールの場合、キャッシュを無効化
	invalidateToolCache(execCtx, tc)

	// ツール出力が空の場合は補完
	if strings.TrimSpace(result) == "" {
		result = "(no output)"
	}

	return result, change, IsErrorResult(result)
}

// IsErrorResult はツール結果が失敗メッセージかどうかを判定する。
func IsErrorResult(result string) bool {
	return strings.HasPrefix(strings.TrimSpace(result), "Error:")
}

// invalidateToolCache はファイル変更系ツール実行後にキャッシュを無効化
func invalidateToolCache(execCtx ExecutionContext, tc *ToolCall) {
	cache := execCtx.EffectiveToolCache()
	if cache == nil {
		return
	}

	switch tc.Tool {
	// ファイル内容を変更するツール → ファイルキャッシュ＆検索キャッシュ無効化
	case "apply_patch":
		cache.Clear()

	case "write_file", "str_replace", "format", "lint":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				cache.InvalidateFile(absPath)
				cache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// ファイルを削除するツール → ファイル＆ディレクトリ＆検索キャッシュ無効化
	case "delete_file":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				cache.InvalidateFile(absPath)
				cache.InvalidateDir(filepath.Dir(absPath))
				cache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// コピーはコピー先のディレクトリキャッシュ＆検索キャッシュを無効化
	case "copy_file":
		if dest := tc.Args["dest"]; dest != "" {
			if absPath, err := filepath.Abs(dest); err == nil {
				cache.InvalidateDir(filepath.Dir(absPath))
				cache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// ディレクトリ作成は検索結果に影響しないため検索キャッシュはクリアしない
	case "create_dir":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				cache.InvalidateDir(filepath.Dir(absPath))
			}
		}

	// git checkout でファイルが復元される可能性
	case "git_checkout":
		// 全キャッシュクリア（どのファイルが変更されるか分からない）
		cache.Clear()

	// bash: read-only コマンドならキャッシュを保持、それ以外は全クリア
	case "bash":
		if cmd := tc.Args["command"]; !isBashReadOnly(cmd) {
			cache.Clear()
		}
	}
}

// readOnlyCommands は bash 実行後にキャッシュクリアが不要なコマンドのプレフィックス。
// ファイルを変更しないコマンドのみ（go mod tidy 等は除外）。
// NOTE: auto-approve 判定とは独立。discovery 系もキャッシュ判定では read-only 扱い。
var readOnlyCommands = []string{
	"ls", "cat", "pwd", "echo", "which",
	"head", "tail", "wc", "grep", "rg", "find",
	"sed -n", "diff", "file", "du", "stat",
	"md5sum", "sha256sum",
	"git status", "git log", "git diff", "git branch",
	"git ls-files", "git show", "git remote",
	"go version", "go test", "go vet", "go env",
	"node -v", "npm -v", "npm list",
	"python --version", "pip list",
	"env", "printenv", "date", "uname", "whoami", "id",
	"tree", "less", "more", "sort", "uniq", "cut", "tr",
}

// isBashReadOnly はコマンドが read-only（キャッシュクリア不要）かどうかを判定する。
// パイプ、リダイレクト、連結演算子を含む場合は安全側に倒して false を返す。
func isBashReadOnly(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return true
	}

	// パイプ・リダイレクト・連結を含む場合は unsafe
	for _, ch := range []string{"|", ">", ">>", "&&", ";"} {
		if strings.Contains(trimmed, ch) {
			return false
		}
	}

	// 先頭コマンドが read-only リストにマッチするか
	for _, prefix := range readOnlyCommands {
		if trimmed == prefix {
			return true
		}
		if strings.HasPrefix(trimmed, prefix) && len(trimmed) > len(prefix) {
			next := trimmed[len(prefix)]
			if next == ' ' || next == '\t' {
				return true
			}
		}
	}

	return false
}

// IsReadOnlyBashCommand は bash コマンドが read-only かどうかを返す。
func IsReadOnlyBashCommand(command string) bool {
	return isBashReadOnly(command)
}

// PreviewToolCallWithWriter は指定 writer にツール情報を表示する（実行はしない）。
func PreviewToolCallWithWriter(w io.Writer, tc *ToolCall) {
	if w == nil {
		w = ui.DefaultRuntime().Output()
	}
	color.New(color.FgCyan).Fprintf(w, "🔧 Tool: %s (Dry Run)\n", tc.Tool)
	printToolArgs(w, tc)
}

// PreviewToolCall displays tool information without executing it
func PreviewToolCall(tc *ToolCall) {
	PreviewToolCallWithWriter(ui.DefaultRuntime().Output(), tc)
}

// isStreamingTool はストリーミング出力を行うツールか判定
// これらのツールは実行中にリアルタイム出力するため、折りたたみ表示は不要
func isStreamingTool(toolName string) bool {
	switch toolName {
	case "bash":
		// bashはストリーミング設定時にリアルタイム出力
		return true
	default:
		return false
	}
}

func shouldShowCollapsedOutput(output string) bool {
	return strings.Contains(output, "\n")
}

// displayCollapsedOutput はツール出力を折りたたみ表示
func displayCollapsedOutput(w io.Writer, output string, cfg *config.Config) {
	// エラー出力や短い成功メッセージはそのまま表示
	if strings.HasPrefix(output, "Error:") ||
		strings.HasPrefix(output, "Successfully") ||
		strings.HasPrefix(output, "Cancelled") ||
		strings.HasPrefix(output, "No ") {
		// 1行メッセージはそのまま
		if !strings.Contains(output, "\n") {
			color.New(color.Faint).Fprintf(w, "⎿  %s\n", output)
			return
		}
	}

	// 折りたたみ表示
	formatted := ui.FormatToolOutput(output, ui.GetMaxVisibleLinesWithConfig(cfg))
	_, _ = fmt.Fprint(w, formatted)
}
