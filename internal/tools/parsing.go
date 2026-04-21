package tools

import (
	"io"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ParseToolCall はレスポンスからツール呼び出しを抽出（最初の1つのみ - 後方互換）
func ParseToolCall(response string) *ToolCall {
	calls := ParseToolCalls(response)
	if len(calls) == 0 {
		return nil
	}
	return calls[0]
}

// ParseToolCalls はレスポンスから全てのツール呼び出しを抽出
// Markdownコードブロック内のJSONは除外する
func ParseToolCalls(response string) []*ToolCall {
	return ParseToolCallsWithRegistry(response, DefaultRegistry, ui.DefaultRuntime().ErrorOutput())
}

// ParseToolCallsWithRegistry は registry を指定して全てのツール呼び出しを抽出する。
// debugOut にはデバッグログの出力先を渡す。
func ParseToolCallsWithRegistry(response string, registry *Registry, debugOut io.Writer) []*ToolCall {
	registry = resolveRegistry(registry)

	debug := os.Getenv("XELYON_DEBUG_PARSE") == "1"
	if debug {
		logParseResponseDebug(response, debugOut)
	}
	codeBlockRanges := findCodeBlockRanges(response)
	results := parseJSONToolCalls(response, codeBlockRanges, debug, debugOut)

	// XML rescue: JSONで何も見つからなかった場合にXML形式を試す
	if len(results) == 0 {
		return parseXMLToolCalls(response, codeBlockRanges, debug, registry, debugOut)
	}

	return results
}

func resolveRegistry(registry *Registry) *Registry {
	if registry == nil {
		return DefaultRegistry
	}
	return registry
}

// findCodeBlockRanges はMarkdownコードブロックの範囲を返す
func findCodeBlockRanges(text string) [][2]int {
	var ranges [][2]int
	idx := 0

	for idx < len(text) {
		// ``` の開始を探す
		start := strings.Index(text[idx:], "```")
		if start == -1 {
			break
		}
		start += idx

		// 対応する ``` の終了を探す（開始の次の行から）
		endSearch := start + 3
		// 言語指定がある場合は改行まで読み飛ばす
		newline := strings.Index(text[endSearch:], "\n")
		if newline != -1 {
			endSearch += newline + 1
		}

		end := strings.Index(text[endSearch:], "```")
		if end == -1 {
			// 閉じていない場合は残り全部をコードブロックとみなす
			ranges = append(ranges, [2]int{start, len(text)})
			break
		}
		end += endSearch + 3

		ranges = append(ranges, [2]int{start, end})
		idx = end
	}

	return ranges
}

// isInCodeBlock は指定位置がコードブロック内かどうかを返す
func isInCodeBlock(pos int, ranges [][2]int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}

// truncateDebug はデバッグ表示用に文字列を切り詰める
func truncateDebug(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
