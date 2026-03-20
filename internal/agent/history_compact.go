package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	DefaultMaxLines  = 50
	DefaultHeadLines = 20
	DefaultTailLines = 5
)

// CompactionMetrics は履歴圧縮時に発生した圧縮メトリクスを表す。
type CompactionMetrics struct {
	ErrorCompressions      int
	FailedPairCompressions int
	TruncationCount        int
}

// CompactOldToolResults は送信前に古いツール結果をtruncateした履歴コピーと圧縮メトリクスを返す。
// 元の history は変更しない（セッション保存用に原本を保持）。
//
// ルール:
// - 後続に "assistant" メッセージが存在する tool 結果は「既読」とみなして圧縮対象にする
// - 後続に "assistant" メッセージが存在しない tool 結果は未読として保護する
//
// 失敗ペア圧縮:
//   - 既読 tool result が "Error:" で始まり、直後の assistant が同じツールを再呼び出しする場合、
//     失敗 tool result を 1行サマリーに圧縮
//
// 注意: shallow copy のため、返されたスライスの Content 以外のフィールド（ToolCalls 等）は
// 元の history と共有されている。返り値の Content 以外を変更してはならない。
func CompactOldToolResults(history []api.Message, maxLines, headLines, tailLines int) ([]api.Message, CompactionMetrics) {
	var metrics CompactionMetrics
	if len(history) == 0 {
		return history, metrics
	}

	lastAssistantIdx := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			lastAssistantIdx = i
			break
		}
	}

	if lastAssistantIdx < 0 {
		result := make([]api.Message, len(history))
		copy(result, history)
		return result, metrics
	}

	result := make([]api.Message, len(history))
	copy(result, history)

	for i := 0; i < lastAssistantIdx; i++ {
		if result[i].Role == "tool" {
			if compressed, ok := compressFailedPair(result, i); ok {
				metrics.FailedPairCompressions++
				result[i] = compressed
				continue
			}

			var itemMetrics CompactionMetrics
			result[i], itemMetrics = truncateToolResult(result[i], maxLines, headLines, tailLines)
			metrics.ErrorCompressions += itemMetrics.ErrorCompressions
			metrics.TruncationCount += itemMetrics.TruncationCount
		}
	}

	return result, metrics
}

// compressFailedPair は失敗ペアを検出して圧縮する。
// tool result が "Error:" で始まり、直後の assistant メッセージが同じツールを再呼び出しする場合、
// 1行サマリーに圧縮した Message を返す。
func compressFailedPair(history []api.Message, toolIdx int) (api.Message, bool) {
	content := strings.TrimSpace(history[toolIdx].Content)
	if !strings.HasPrefix(content, "Error:") {
		return api.Message{}, false
	}

	// 直後の assistant メッセージを探す
	assistantIdx := -1
	for j := toolIdx + 1; j < len(history); j++ {
		if history[j].Role == "assistant" {
			assistantIdx = j
			break
		}
		// user が先に来たら失敗ペアではない
		if history[j].Role == "user" {
			return api.Message{}, false
		}
	}
	if assistantIdx < 0 {
		return api.Message{}, false
	}

	// assistant が同じツールを再呼び出ししているかチェック
	failedToolName := history[toolIdx].ToolName
	if failedToolName == "" {
		return api.Message{}, false
	}
	retried := false
	for _, tc := range history[assistantIdx].ToolCalls {
		if tc.Function.Name == failedToolName {
			retried = true
			break
		}
	}
	if !retried {
		return api.Message{}, false
	}

	// 1行サマリーに圧縮
	firstLine := content
	if idx := strings.IndexByte(content, '\n'); idx >= 0 {
		firstLine = content[:idx]
	}
	compressed := history[toolIdx]
	compressed.Content = fmt.Sprintf("[Failed: %s — %s]", failedToolName, firstLine)
	return compressed, true
}

// compressErrorResult はエラーパターンを検出して短い要約に圧縮する。
// 圧縮した場合は (要約, true) を返す。エラーパターンでなければ ("", false)。
func compressErrorResult(content string) (string, bool) {
	trimmed := strings.TrimSpace(content)

	// "No matches found" → 既に短いのでそのまま
	if trimmed == "No matches found" {
		return trimmed, true
	}

	// "Error: pattern not found" / "Error: old_str not found"
	if strings.HasPrefix(trimmed, "Error: pattern not found") ||
		strings.HasPrefix(trimmed, "Error: old_str not found") {
		// 1行目だけ残す
		if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
			return trimmed[:idx], true
		}
		return trimmed, true
	}

	// "Error reading file:" → 1行目だけ
	if strings.HasPrefix(trimmed, "Error reading file:") {
		if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
			return trimmed[:idx], true
		}
		return trimmed, true
	}

	// "Error:" で始まる結果全般 → 1行目だけ
	if strings.HasPrefix(trimmed, "Error:") {
		if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
			return trimmed[:idx], true
		}
		return trimmed, true
	}

	return "", false
}

// truncateToolResult は大きなツール結果を truncate する。
// エラーパターンは行数に関係なく圧縮する。
func truncateToolResult(msg api.Message, maxLines, headLines, tailLines int) (api.Message, CompactionMetrics) {
	var metrics CompactionMetrics
	if isCompactedToolResult(msg.Content) {
		return msg, metrics
	}

	// エラーパターンの圧縮を先に試す
	if compressed, ok := compressErrorResult(msg.Content); ok {
		metrics.ErrorCompressions++
		if compressed != msg.Content {
			result := msg
			result.Content = compressed
			return result, metrics
		}
		return msg, metrics
	}

	lines := strings.Split(strings.TrimRight(msg.Content, "\n"), "\n")
	if len(lines) <= maxLines || headLines+tailLines >= len(lines) {
		return msg, metrics
	}

	// おおよそのトークン数を推定（~4文字/トークン）
	estimatedTokens := len(msg.Content) / 4

	head := strings.Join(lines[:headLines], "\n")
	tail := strings.Join(lines[len(lines)-tailLines:], "\n")
	truncated := fmt.Sprintf("%s\n... (truncated: was %d lines, ~%d tokens)\n%s",
		head, len(lines), estimatedTokens, tail)

	// shallow copy して Content だけ差し替え
	result := msg
	result.Content = truncated
	metrics.TruncationCount++
	return result, metrics
}

func isCompactedToolResult(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "[Cleared ") ||
		strings.HasPrefix(trimmed, "[Old ") ||
		strings.HasPrefix(trimmed, "[Failed:")
}
