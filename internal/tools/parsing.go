package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
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

var toolCallJSONStartPatterns = []string{
	"{\"id\"",     // {"id" (Function Calling)
	"{ \"id\"",    // { "id" (Function Calling)
	"{\"tool\"",   // {"tool"
	"{ \"tool\"",  // { "tool"
	"{\"tool\":",  // {"tool":
	"{ \"tool\":", // { "tool":
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

// xmlOpenTagPattern は <tag_name> 形式の開始タグを検出する正規表現
var xmlOpenTagPattern = regexp.MustCompile(`<([a-zA-Z_][\w-]*)>`)

// parseXMLToolCalls はXML形式のツール呼び出しをパースする
// Kimi K2 等がFC失敗時に出力する XML 形式を rescue する
func parseXMLToolCalls(response string, codeBlockRanges [][2]int, debug bool, registry *Registry, debugOut io.Writer) []*ToolCall {
	var results []*ToolCall
	searchFrom := 0

	for searchFrom < len(response) {
		// 開始タグを探す
		loc := xmlOpenTagPattern.FindStringIndex(response[searchFrom:])
		if loc == nil {
			break
		}
		absStart := searchFrom + loc[0]
		tagEnd := searchFrom + loc[1]

		// タグ名を抽出
		tagName := xmlOpenTagPattern.FindStringSubmatch(response[searchFrom:])[1]

		// 対応する閉じタグを探す
		closeTag := "</" + tagName + ">"
		closeIdx := strings.Index(response[tagEnd:], closeTag)
		if closeIdx == -1 {
			searchFrom = tagEnd
			continue
		}
		absCloseStart := tagEnd + closeIdx
		fullEnd := absCloseStart + len(closeTag)

		innerContent := response[tagEnd:absCloseStart]

		// 次の検索位置を更新
		searchFrom = fullEnd

		// コードブロック内はスキップ
		if isInCodeBlock(absStart, codeBlockRanges) {
			if debug {
				fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: skipping %q in code block\n", tagName)
			}
			continue
		}

		// 指定 Registry に登録されているツール名のみ許可
		if !registry.HasTool(tagName) {
			if debug {
				fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: skipping unknown tool %q\n", tagName)
			}
			continue
		}

		// 内部コンテンツからパラメータを抽出
		args := parseXMLParams(innerContent)

		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: tool=%s, args=%v\n", tagName, args)
		}

		tc := &ToolCall{
			Tool: tagName,
			Args: args,
		}
		results = append(results, tc)
	}

	return results
}

func resolveRegistry(registry *Registry) *Registry {
	if registry == nil {
		return DefaultRegistry
	}
	return registry
}

// parseXMLParams は XML 内部コンテンツからパラメータを抽出する
// パターン1: <args><param>value</param>...</args> （args ラッパーあり）
// パターン2: <param>value</param>... （args ラッパーなし）
// パターン3: {"key": "value"} （JSON 形式）
func parseXMLParams(content string) map[string]string {
	args := make(map[string]string)

	// <args>...</args> ラッパーがある場合は中身を取り出す
	if argsStart := strings.Index(content, "<args>"); argsStart != -1 {
		if argsEnd := strings.Index(content, "</args>"); argsEnd != -1 && argsEnd > argsStart {
			content = content[argsStart+len("<args>") : argsEnd]
		}
	}

	// <param>value</param> を手動パースで抽出（バックリファレンス不要）
	searchFrom := 0
	for searchFrom < len(content) {
		// 開始タグを探す
		loc := xmlOpenTagPattern.FindStringIndex(content[searchFrom:])
		if loc == nil {
			break
		}
		tagStart := searchFrom + loc[1]
		tagName := xmlOpenTagPattern.FindStringSubmatch(content[searchFrom:])[1]

		// "args" タグ自体はスキップ
		if tagName == "args" {
			searchFrom = tagStart
			continue
		}

		// 対応する閉じタグを探す
		closeTag := "</" + tagName + ">"
		closeIdx := strings.Index(content[tagStart:], closeTag)
		if closeIdx == -1 {
			searchFrom = tagStart
			continue
		}

		value := content[tagStart : tagStart+closeIdx]
		args[tagName] = value
		searchFrom = tagStart + closeIdx + len(closeTag)
	}

	// XML パラメータが見つからなかった場合、内容を JSON としてパース
	if len(args) == 0 {
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "{") {
			var jsonArgs map[string]interface{}
			if err := json.Unmarshal([]byte(trimmed), &jsonArgs); err == nil {
				for k, v := range jsonArgs {
					switch val := v.(type) {
					case string:
						args[k] = val
					default:
						if b, err := json.Marshal(v); err == nil {
							args[k] = string(b)
						}
					}
				}
			}
		}
	}

	return args
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
