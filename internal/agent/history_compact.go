package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	DefaultMaxLines        = 50
	DefaultHeadLines       = 20
	DefaultTailLines       = 5
	ultraCompactAge        = 10
	phase0ClearAge         = 3
	phase0ClearAgeCautious = 6
	phase0MinContentLen    = 200
)

// CompactionMetrics は履歴圧縮時に発生した圧縮メトリクスを表す。
type CompactionMetrics struct {
	Phase0Clears           int
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
// - 既読 tool 結果は段階的に truncate し、古いほど短くする
//
// 段階的圧縮:
// - 直近 5 回分: headLines/tailLines（デフォルト 20/5）
// - 6 〜 15 回前: headLines=10, tailLines=3
// - 16 回以上前: headLines=5, tailLines=2
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

	toolAge := buildToolAgeMap(history, lastAssistantIdx)
	argsLookup := buildToolArgsLookup(history, lastAssistantIdx)

	result := make([]api.Message, len(history))
	copy(result, history)

	for i := 0; i < lastAssistantIdx; i++ {
		if result[i].Role == "tool" {
			if compressed, ok := compressFailedPair(result, i); ok {
				metrics.FailedPairCompressions++
				result[i] = compressed
				continue
			}

			age := toolAge[i]

			if placeholder, ok := phase0Clear(result[i], age, argsLookup); ok {
				result[i].Content = placeholder
				metrics.Phase0Clears++
				continue
			}

			if age > ultraCompactAge {
				if summary := ultraCompactToolResult(result[i]); summary != "" {
					result[i].Content = summary
					metrics.TruncationCount++
					continue
				}
			}

			h, t := graduatedTruncateByAge(age, headLines, tailLines)
			var itemMetrics CompactionMetrics
			result[i], itemMetrics = truncateToolResult(result[i], maxLines, h, t)
			metrics.ErrorCompressions += itemMetrics.ErrorCompressions
			metrics.TruncationCount += itemMetrics.TruncationCount
		}
	}

	return result, metrics
}

// buildToolAgeMap returns the tool execution distance from lastAssistantIdx for
// each tool result before it. The closest prior tool is age=1.
func buildToolAgeMap(history []api.Message, lastAssistantIdx int) map[int]int {
	ages := make(map[int]int)
	toolCount := 0
	for i := lastAssistantIdx - 1; i >= 0; i-- {
		if history[i].Role == "tool" {
			toolCount++
			ages[i] = toolCount
		}
	}
	return ages
}

type toolCallInfo struct {
	ToolName string
	Args     map[string]string
}

func buildToolArgsLookup(history []api.Message, lastAssistantIdx int) map[string]toolCallInfo {
	lookup := make(map[string]toolCallInfo)
	for i := 0; i <= lastAssistantIdx; i++ {
		if history[i].Role != "assistant" {
			continue
		}
		for _, tc := range history[i].ToolCalls {
			if tc.ID == "" {
				continue
			}
			lookup[tc.ID] = toolCallInfo{
				ToolName: tc.Function.Name,
				Args:     parseToolArgs(tc.Function.Arguments),
			}
		}
	}
	return lookup
}

func parseToolArgs(argsJSON string) map[string]string {
	result := make(map[string]string)
	if argsJSON == "" {
		return result
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &raw); err != nil {
		return result
	}

	for key, val := range raw {
		switch v := val.(type) {
		case string:
			result[key] = v
		case float64:
			result[key] = fmt.Sprintf("%g", v)
		case bool:
			result[key] = fmt.Sprintf("%v", v)
		default:
			if b, err := json.Marshal(v); err == nil {
				result[key] = string(b)
			}
		}
	}
	return result
}

// graduatedTruncateByAge returns truncate parameters based on tool execution age.
func graduatedTruncateByAge(age, defaultHead, defaultTail int) (headLines, tailLines int) {
	switch {
	case age <= 5:
		return defaultHead, defaultTail
	case age <= 15:
		return 10, 3
	default:
		return 5, 2
	}
}

func phase0Clear(msg api.Message, age int, argsLookup map[string]toolCallInfo) (string, bool) {
	content := msg.Content
	if isCompactedToolResult(content) || len(content) < phase0MinContentLen {
		return "", false
	}

	threshold := phase0GetAgeThreshold(msg.ToolName)
	if threshold == 0 || age < threshold {
		return "", false
	}

	var args map[string]string
	if msg.ToolCallID != "" {
		if info, ok := argsLookup[msg.ToolCallID]; ok && (info.ToolName == "" || info.ToolName == msg.ToolName) {
			args = info.Args
		}
	}

	placeholder := phase0BuildPlaceholder(msg, args, content)
	if placeholder == "" {
		return "", false
	}
	return placeholder, true
}

func phase0GetAgeThreshold(toolName string) int {
	switch toolName {
	case "read_file", "search_code", "list_dir", "inspect_symbol":
		return phase0ClearAge
	case "bash", "web_search":
		return phase0ClearAgeCautious
	default:
		return 0
	}
}

func phase0BuildPlaceholder(msg api.Message, args map[string]string, content string) string {
	lines := countContentLines(content)

	switch msg.ToolName {
	case "read_file":
		path := getArg(args, "path", "")
		if path == "" {
			path = extractPathFromContentFallback(content)
		}
		if path == "" {
			path = "unknown"
		}
		return fmt.Sprintf("[Cleared read_file for %s: %d lines. Re-run read_file if needed.]", path, lines)
	case "search_code":
		pattern := getArg(args, "pattern", "")
		summary := extractSearchSummaryLine(content)
		switch {
		case pattern != "" && summary != "":
			return fmt.Sprintf("[Cleared search_code for %q: %s. Re-run search_code if needed.]", pattern, summary)
		case summary != "":
			return fmt.Sprintf("[Cleared search_code result: %s. Re-run search_code if needed.]", summary)
		case pattern != "":
			return fmt.Sprintf("[Cleared search_code for %q: %d lines. Re-run search_code if needed.]", pattern, lines)
		default:
			return fmt.Sprintf("[Cleared search_code result: %d lines. Re-run search_code if needed.]", lines)
		}
	case "list_dir":
		path := getArg(args, "path", ".")
		return fmt.Sprintf("[Cleared list_dir for %s: %d entries. Re-run list_dir if needed.]", path, extractListDirEntryCount(content))
	case "inspect_symbol":
		symbol := getArg(args, "symbol", "")
		filePath := getArg(args, "path", "")
		switch {
		case symbol != "" && filePath != "":
			return fmt.Sprintf("[Cleared inspect_symbol for %q in %s. Re-run inspect_symbol if needed.]", symbol, filePath)
		case symbol != "":
			return fmt.Sprintf("[Cleared inspect_symbol for %q. Re-run inspect_symbol if needed.]", symbol)
		}
		header := extractInspectHeaderFallback(content)
		if header != "" {
			return fmt.Sprintf("[Cleared inspect_symbol: %s. Re-run inspect_symbol if needed.]", header)
		}
		return fmt.Sprintf("[Cleared inspect_symbol result: %d lines. Re-run inspect_symbol if needed.]", lines)
	case "bash":
		command := truncatePlaceholderValue(getArg(args, "command", "..."), 80)
		status := "exit 0"
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "Command interrupted.") {
			status = "error"
		}
		return fmt.Sprintf("[Cleared bash result for %q: %d lines, %s. Re-run if needed.]", command, lines, status)
	case "web_search":
		query := truncatePlaceholderValue(getArg(args, "query", "..."), 80)
		if sources := extractWebSearchSourceCount(content); sources > 0 {
			return fmt.Sprintf("[Cleared web_search for %q: %d sources. Re-run web_search for latest results.]", query, sources)
		}
		return fmt.Sprintf("[Cleared web_search for %q: %d lines. Re-run web_search for latest results.]", query, lines)
	default:
		return ""
	}
}

func getArg(args map[string]string, key, defaultVal string) string {
	if args == nil {
		return defaultVal
	}
	if v, ok := args[key]; ok && v != "" {
		return v
	}
	return defaultVal
}

func truncatePlaceholderValue(s string, max int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func countContentLines(content string) int {
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "\n") + 1
}

func isCompactedToolResult(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "[Cleared ") ||
		strings.HasPrefix(trimmed, "[Old ") ||
		strings.HasPrefix(trimmed, "[Failed:")
}

func extractPathFromContentFallback(content string) string {
	lines := strings.Split(content, "\n")
	limit := len(lines)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		for _, prefix := range []string{"File:", "Path:", "Read file:", "Reading file:", "📄"} {
			if strings.HasPrefix(line, prefix) {
				line = strings.TrimSpace(strings.TrimPrefix(line, prefix))
				break
			}
		}
		if candidate := sanitizePathCandidate(line); isLikelyPath(candidate) {
			return candidate
		}
		if idx := strings.Index(line, ": "); idx >= 0 {
			if candidate := sanitizePathCandidate(line[idx+2:]); isLikelyPath(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func sanitizePathCandidate(s string) string {
	return strings.Trim(strings.TrimSpace(s), "`\"'<>[]()")
}

func isLikelyPath(candidate string) bool {
	if candidate == "" {
		return false
	}
	lower := strings.ToLower(candidate)
	if strings.Contains(candidate, "/") || strings.Contains(candidate, "\\") {
		return true
	}
	for _, suffix := range []string{
		".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rb", ".java", ".rs", ".c", ".cc", ".cpp",
		".h", ".hpp", ".json", ".yaml", ".yml", ".toml", ".md", ".txt", ".sh",
	} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func extractSearchSummaryLine(content string) string {
	for _, line := range strings.SplitN(content, "\n", 5) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Found ") && strings.Contains(trimmed, "match") {
			return trimmed
		}
	}
	return ""
}

func extractInspectHeaderFallback(content string) string {
	for _, line := range strings.SplitN(content, "\n", 3) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "──") {
			return trimmed
		}
	}
	return ""
}

func extractListDirEntryCount(content string) int {
	for _, line := range strings.SplitN(content, "\n", 4) {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "summary:") {
			continue
		}
		dirs := -1
		files := -1
		for _, part := range strings.Split(trimmed, ",") {
			part = strings.TrimSpace(part)
			switch {
			case strings.HasPrefix(part, "dirs="):
				if _, err := fmt.Sscanf(part, "dirs=%d", &dirs); err != nil {
					dirs = -1
				}
			case strings.HasPrefix(part, "files="):
				if _, err := fmt.Sscanf(part, "files=%d", &files); err != nil {
					files = -1
				}
			}
		}
		if dirs >= 0 && files >= 0 {
			return dirs + files
		}
	}
	return countContentLines(content)
}

func extractWebSearchSourceCount(content string) int {
	count := 0
	inSources := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "Sources:" {
			inSources = true
			continue
		}
		if !inSources {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			count++
			continue
		}
		if len(trimmed) > 2 && trimmed[0] >= '1' && trimmed[0] <= '9' {
			if dotIdx := strings.Index(trimmed, ". "); dotIdx > 0 && dotIdx <= 3 {
				count++
			}
		}
	}
	return count
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

// ToolResultContentRatio はHistory内のtool resultコンテンツが全体に占める割合を返す。
// len(Content) ベースの簡易推定。
func ToolResultContentRatio(history []api.Message) float64 {
	total := 0
	toolTotal := 0
	for _, msg := range history {
		n := len(msg.Content)
		total += n
		if msg.Role == "tool" {
			toolTotal += n
		}
	}
	if total == 0 {
		return 0
	}
	return float64(toolTotal) / float64(total)
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

// ultraCompactToolResult は非常に古い tool result を 1行サマリーに圧縮する。
// ToolName からツール種別を判定し、Content の先頭行から要点を抽出する。
// 圧縮できない場合は空文字を返す（呼び出し元は通常の graduated truncation にフォールバック）。
func ultraCompactToolResult(msg api.Message) string {
	content := strings.TrimSpace(msg.Content)
	if content == "" || isCompactedToolResult(content) {
		return ""
	}

	lines := strings.Split(content, "\n")
	lineCount := len(lines)
	// 既に短い結果は ultra-compact 不要
	if lineCount <= 7 {
		return ""
	}

	// ツール名に応じた 1行サマリーを生成
	switch msg.ToolName {
	case "read_file":
		// first line often has path info or content start
		return fmt.Sprintf("[Old read_file result: %d lines — use read_file again if needed]", lineCount)
	case "search_code":
		// first line is typically "Found N matches in M files..."
		firstLine := lines[0]
		if strings.HasPrefix(firstLine, "Found ") || strings.HasPrefix(firstLine, "No matches") {
			return fmt.Sprintf("[Old search_code result: %s]", firstLine)
		}
		return fmt.Sprintf("[Old search_code result: %d lines — search again if needed]", lineCount)
	case "inspect_symbol":
		return fmt.Sprintf("[Old inspect_symbol result: %d lines — inspect again if needed]", lineCount)
	case "list_dir":
		return fmt.Sprintf("[Old list_dir result: %d lines — list again if needed]", lineCount)
	case "bash":
		return fmt.Sprintf("[Old bash result: %d lines]", lineCount)
	default:
		// ToolName が空（text-based パス）または未知のツール → フォールバック
		if msg.ToolName == "" {
			return ""
		}
		return fmt.Sprintf("[Old %s result: %d lines]", msg.ToolName, lineCount)
	}
}
