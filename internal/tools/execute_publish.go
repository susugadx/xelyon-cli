package tools

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

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
	status := ToolStatusOK
	if isError {
		status = ToolStatusError
	}
	startedAt := execResult.StartedAt
	if startedAt.IsZero() && execResult.Duration > 0 {
		startedAt = time.Now().Add(-execResult.Duration)
	}

	if execCtx.ToolResultCallback != nil {
		// TUIモード: 構造化データをコールバックで送信
		execCtx.ToolResultCallback(ToolResultInfo{
			ToolName:  tc.Tool,
			Args:      CloneToolArgs(tc.Args),
			Result:    result,
			Error:     isError,
			ID:        tc.DisplayID(),
			Status:    status,
			StartedAt: startedAt,
			Duration:  execResult.Duration,
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
