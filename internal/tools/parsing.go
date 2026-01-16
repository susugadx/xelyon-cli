package tools

import (
	"encoding/json"
	"strings"
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
	// Markdownコードブロック（```...```）の位置を記録
	codeBlockRanges := findCodeBlockRanges(response)

	// JSONブロックを探す（複数パターン対応）
	patterns := []string{
		"{\"tool\"",   // {"tool"
		"{ \"tool\"",  // { "tool"
		"{\"tool\":",  // {"tool":
		"{ \"tool\":", // { "tool":
	}

	var results []*ToolCall
	searchFrom := 0

	for searchFrom < len(response) {
		// 次のツール呼び出し候補を探す
		start := -1
		for _, pattern := range patterns {
			idx := strings.Index(response[searchFrom:], pattern)
			if idx != -1 {
				absIdx := searchFrom + idx
				if start == -1 || absIdx < start {
					start = absIdx
				}
			}
		}
		if start == -1 {
			break
		}

		// コードブロック内の場合はスキップ
		if isInCodeBlock(start, codeBlockRanges) {
			searchFrom = start + 1
			continue
		}

		// 対応する閉じ括弧を探す
		depth := 0
		end := -1
		for i := start; i < len(response); i++ {
			if response[i] == '{' {
				depth++
			} else if response[i] == '}' {
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}

		if end == -1 {
			break
		}

		jsonStr := response[start:end]
		var toolCall ToolCall
		if err := json.Unmarshal([]byte(jsonStr), &toolCall); err != nil {
			// パースエラーの場合はスキップして次を探す
			searchFrom = end
			continue
		}

		// tool フィールドが空の場合はスキップ
		if toolCall.Tool == "" {
			searchFrom = end
			continue
		}

		// RawArgs（any型）をArgs（string型）に変換
		toolCall.NormalizeArgs()

		results = append(results, &toolCall)
		searchFrom = end
	}

	return results
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
