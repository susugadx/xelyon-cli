package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var toolCallJSONStartPatterns = []string{
	"{\"id\"",     // {"id" (Function Calling)
	"{ \"id\"",    // { "id" (Function Calling)
	"{\"tool\"",   // {"tool"
	"{ \"tool\"",  // { "tool"
	"{\"tool\":",  // {"tool":
	"{ \"tool\":", // { "tool":
}

func logParseResponseDebug(response string, debugOut io.Writer) {
	fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] response length: %d\n", len(response))
	for _, p := range []string{`{"tool"`, `{ "tool"`} {
		if idx := strings.Index(response, p); idx != -1 {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] found pattern %q at index %d\n", p, idx)
			start := idx
			if start > 50 {
				start = idx - 50
			}
			end := idx + 100
			if end > len(response) {
				end = len(response)
			}
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] context: ...%s...\n", response[start:end])
		}
	}
}

func parseJSONToolCalls(response string, codeBlockRanges [][2]int, debug bool, debugOut io.Writer) []*ToolCall {
	var results []*ToolCall
	searchFrom := 0

	for searchFrom < len(response) {
		start := findNextToolCallJSONStart(response, searchFrom)
		if start == -1 {
			break
		}

		if shouldSkipCodeBlockJSON(start, codeBlockRanges, debug, debugOut) {
			searchFrom = start + 1
			continue
		}

		jsonStr, end, ok := extractJSONObject(response, start, debug, debugOut)
		if !ok {
			break
		}

		if toolCall, ok := decodeToolCallJSON(jsonStr, debug, debugOut); ok {
			results = append(results, toolCall)
		}
		searchFrom = end
	}

	return results
}

func findNextToolCallJSONStart(response string, searchFrom int) int {
	start := -1
	for _, pattern := range toolCallJSONStartPatterns {
		idx := strings.Index(response[searchFrom:], pattern)
		if idx == -1 {
			continue
		}
		absIdx := searchFrom + idx
		if start == -1 || absIdx < start {
			start = absIdx
		}
	}
	return start
}

func shouldSkipCodeBlockJSON(start int, codeBlockRanges [][2]int, debug bool, debugOut io.Writer) bool {
	if !isInCodeBlock(start, codeBlockRanges) {
		return false
	}
	if debug {
		fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] skipping: in code block at %d\n", start)
	}
	return true
}

func extractJSONObject(response string, start int, debug bool, debugOut io.Writer) (string, int, bool) {
	end := findJSONObjectEnd(response, start)
	if end == -1 {
		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] incomplete JSON: no closing brace found from index %d\n", start)
			showStart := start
			if len(response)-showStart > 200 {
				showStart = len(response) - 200
			}
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] tail: ...%s\n", response[showStart:])
		}
		return "", 0, false
	}

	jsonStr := response[start:end]
	if debug {
		fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] extracted JSON (%d bytes): %s\n", len(jsonStr), truncateDebug(jsonStr, 200))
	}
	return jsonStr, end, true
}

func findJSONObjectEnd(response string, start int) int {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(response); i++ {
		ch := response[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return -1
}

func decodeToolCallJSON(jsonStr string, debug bool, debugOut io.Writer) (*ToolCall, bool) {
	var toolCall ToolCall
	if !unmarshalToolCallJSONWithRepair(jsonStr, &toolCall, debug, debugOut) {
		return nil, false
	}

	if toolCall.Tool == "" {
		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] skipping: empty tool field\n")
		}
		return nil, false
	}

	toolCall.NormalizeArgs()
	return &toolCall, true
}

func unmarshalToolCallJSONWithRepair(jsonStr string, toolCall *ToolCall, debug bool, debugOut io.Writer) bool {
	if err := json.Unmarshal([]byte(jsonStr), toolCall); err == nil {
		return true
	} else {
		repaired := repairJSONStringValues(jsonStr)
		if repaired != jsonStr {
			if err2 := json.Unmarshal([]byte(repaired), toolCall); err2 == nil {
				if debug {
					fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] JSON repaired: fixed raw control characters in string values\n")
				}
				return true
			}
		}

		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] JSON parse error: %v\n", err)
		}
		return false
	}
}

// repairJSONStringValues は JSON 文字列値内の生制御文字をエスケープする。
// LLM が FC rescue テキストで出力する malformed JSON を修復する。
// 正常な JSON はそのまま返す。既にエスケープ済みの \\n, \\t, \\" 等は二重エスケープしない。
//
// 修復対象:
//   - 生改行 (0x0A) → \n
//   - 生キャリッジリターン (0x0D) → \r
//   - 生タブ (0x09) → \t
//   - その他の制御文字 (0x00-0x1F) → \uXXXX
func repairJSONStringValues(jsonStr string) string {
	var buf strings.Builder
	buf.Grow(len(jsonStr) + 64)

	inString := false
	escaped := false

	for i := 0; i < len(jsonStr); i++ {
		ch := jsonStr[i]

		if escaped {
			buf.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			buf.WriteByte(ch)
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			buf.WriteByte(ch)
			continue
		}

		if inString {
			switch ch {
			case '\n':
				buf.WriteString(`\n`)
			case '\r':
				buf.WriteString(`\r`)
			case '\t':
				buf.WriteString(`\t`)
			default:
				if ch < 0x20 {
					fmt.Fprintf(&buf, `\u%04x`, ch)
				} else {
					buf.WriteByte(ch)
				}
			}
		} else {
			buf.WriteByte(ch)
		}
	}

	return buf.String()
}
