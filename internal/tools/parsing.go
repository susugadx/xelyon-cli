package tools

import (
	"encoding/json"
	"fmt"
	"os"
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
	// デバッグモード
	debug := os.Getenv("XELYON_DEBUG_PARSE") == "1"
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] response length: %d\n", len(response))
		// ツールパターンの存在をチェック
		for _, p := range []string{`{"tool"`, `{ "tool"`} {
			if idx := strings.Index(response, p); idx != -1 {
				fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] found pattern %q at index %d\n", p, idx)
				// 周辺100文字を表示
				start := idx
				if start > 50 {
					start = idx - 50
				}
				end := idx + 100
				if end > len(response) {
					end = len(response)
				}
				fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] context: ...%s...\n", response[start:end])
			}
		}
	}
	// Markdownコードブロック（```...```）の位置を記録
	codeBlockRanges := findCodeBlockRanges(response)

	// JSONブロックを探す（複数パターン対応）
	// Function Calling では {"id": "...", "tool": "..."} 形式
	patterns := []string{
		"{\"id\"",     // {"id" (Function Calling)
		"{ \"id\"",    // { "id" (Function Calling)
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
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] skipping: in code block at %d\n", start)
			}
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
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] incomplete JSON: no closing brace found from index %d\n", start)
				// 末尾100文字を表示
				showStart := start
				if len(response)-showStart > 200 {
					showStart = len(response) - 200
				}
				fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] tail: ...%s\n", response[showStart:])
			}
			break
		}

		jsonStr := response[start:end]
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] extracted JSON (%d bytes): %s\n", len(jsonStr), truncateDebug(jsonStr, 200))
		}
		var toolCall ToolCall
		if err := json.Unmarshal([]byte(jsonStr), &toolCall); err != nil {
			// パースエラーの場合はスキップして次を探す
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] JSON parse error: %v\n", err)
			}
			searchFrom = end
			continue
		}

		// tool フィールドが空の場合はスキップ
		if toolCall.Tool == "" {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG ParseToolCalls] skipping: empty tool field\n")
			}
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

// truncateDebug はデバッグ表示用に文字列を切り詰める
func truncateDebug(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
