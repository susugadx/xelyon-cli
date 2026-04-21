package tools

import (
	"io"
	"os"

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
	options := resolveParseRunOptions(registry, debugOut)
	return parseToolCalls(response, options)
}

type parseRunOptions struct {
	registry *Registry
	debug    bool
	debugOut io.Writer
}

func resolveParseRunOptions(registry *Registry, debugOut io.Writer) parseRunOptions {
	return parseRunOptions{
		registry: resolveRegistry(registry),
		debug:    os.Getenv("XELYON_DEBUG_PARSE") == "1",
		debugOut: debugOut,
	}
}

func parseToolCalls(response string, options parseRunOptions) []*ToolCall {
	if options.debug {
		logParseResponseDebug(response, options.debugOut)
	}
	codeBlockRanges := findCodeBlockRanges(response)
	results := parseJSONToolCalls(response, codeBlockRanges, options.debug, options.debugOut)

	// XML rescue: JSONで何も見つからなかった場合にXML形式を試す
	if len(results) == 0 {
		return parseXMLToolCalls(response, codeBlockRanges, options.debug, options.registry, options.debugOut)
	}

	return results
}

func resolveRegistry(registry *Registry) *Registry {
	if registry == nil {
		return DefaultRegistry
	}
	return registry
}

// truncateDebug はデバッグ表示用に文字列を切り詰める
func truncateDebug(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
