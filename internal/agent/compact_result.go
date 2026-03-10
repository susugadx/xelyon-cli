package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/dev"
)

const (
	// compactBashMinLen はbash成功時にverifyコマンドを圧縮する最小出力長。
	// この長さ未満の出力はそのまま返す。
	compactBashMinLen = 500

	// compactTailLines は失敗時に残す末尾行数。
	compactTailLines = 30

	// compactCommandMaxLen はサマリー内のコマンド文字列最大長。
	compactCommandMaxLen = 120
)

// compactToolResult はtool実行結果をHistory格納用に圧縮する。
// deduplicateToolResult の後に適用し、重複参照文字列はそのまま返す。
// read_file, search_code は既存の最適化があるため対象外。
func (a *Agent) compactToolResult(toolCall *tools.ToolCall, result string) string {
	// 1. 重複チェック（元の全文で判定する必要がある）
	deduped := a.deduplicateToolResult(toolCall.Tool, result)
	if deduped != result {
		// 重複参照に差し替えられた → 圧縮不要
		return deduped
	}

	// 2. ツール種別に応じた圧縮
	switch toolCall.Tool {
	case "bash":
		command := toolCall.Args["command"]
		return compactBash(command, result)
	default:
		return result
	}
}

// compactBash はbashの実行結果を圧縮する。
func compactBash(command, result string) string {
	if isBashError(result) {
		return tailLines(result, compactTailLines)
	}

	// 成功: 短い出力はそのまま
	if len(result) < compactBashMinLen {
		return result
	}

	// 成功 + 長い出力: verifyコマンドなら1行サマリー
	if dev.IsVerifyCommand(command) {
		return fmt.Sprintf("OK: %s (exit 0)", truncateCommand(command, compactCommandMaxLen))
	}

	return result
}

// isBashError はbash実行結果がエラーかを判定する。
// bash.go の出力フォーマットに基づく:
//   - "Error: ..." (exit ≠ 0)
//   - "Command interrupted." (context cancelled)
func isBashError(result string) bool {
	return strings.HasPrefix(result, "Error:") ||
		strings.HasPrefix(result, "Command interrupted.")
}

// tailLines は文字列の末尾n行を返す。全体がn行以下の場合はそのまま返す。
// 切り詰めた場合は先頭に "[... N lines omitted ...]" を付加。
func tailLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	omitted := len(lines) - n
	return fmt.Sprintf("[... %d lines omitted ...]\n%s", omitted, strings.Join(lines[len(lines)-n:], "\n"))
}

// truncateCommand はコマンド文字列を最大maxLen文字に切り詰める。
func truncateCommand(command string, maxLen int) string {
	if len(command) <= maxLen {
		return command
	}
	return command[:maxLen-3] + "..."
}
