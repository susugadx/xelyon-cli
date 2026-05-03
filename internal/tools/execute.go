package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	applyToolCallDefaults(tc)
	result, change := executeToolCallCore(execCtx, tc)
	result = normalizeToolExecutionOutput(result)

	return result, change, IsErrorResult(result)
}

func applyToolCallDefaults(tc *ToolCall) {
	if tc == nil {
		return
	}
	if tc.Args == nil {
		tc.Args = make(map[string]string)
	}
	if tc.Args["path"] != "" {
		return
	}
	switch tc.Tool {
	case "list_dir", "git_add":
		tc.Args["path"] = "."
	}
}

func executeToolCallCore(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	result, change := execCtx.EffectiveRegistry().ExecuteWithContext(execCtx, tc)
	invalidateToolCache(execCtx, tc, change)
	return result, change
}

func normalizeToolExecutionOutput(result string) string {
	if strings.TrimSpace(result) == "" {
		return "(no output)"
	}
	return result
}

// IsErrorResult はツール結果が失敗メッセージかどうかを判定する。
func IsErrorResult(result string) bool {
	return strings.HasPrefix(strings.TrimSpace(result), "Error:")
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
