package tools

import (
	"io"
	"os"
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
	return ParseToolCallsWithRegistry(response, DefaultRegistry, nil)
}

// ParseToolCallsWithRegistry は registry を指定して全てのツール呼び出しを抽出する。
// debugOut にはデバッグログの出力先を渡す。
func ParseToolCallsWithRegistry(response string, registry *Registry, debugOut io.Writer) []*ToolCall {
	options := resolveParseRunOptions(registry, debugOut)
	return parseToolCalls(response, options)
}

type parseRunOptions struct {
	registry        *Registry
	startFinder     jsonToolCallStartFinder
	codeBlockPolicy markdownCodeBlockPolicy
	logger          *parseDebugLogger
}

func resolveParseRunOptions(registry *Registry, debugOut io.Writer) parseRunOptions {
	debugEnabled := os.Getenv("XELYON_DEBUG_PARSE") == "1"
	return parseRunOptions{
		registry:        resolveRegistry(registry),
		startFinder:     newDefaultJSONToolCallStartFinder(),
		codeBlockPolicy: defaultMarkdownCodeBlockPolicy(),
		logger:          newParseDebugLogger(debugEnabled, debugOut),
	}
}

func parseToolCalls(response string, options parseRunOptions) []*ToolCall {
	options.logger.LogParseResponse(response, options.startFinder)
	ctx := newParseToolCallContext(response, options)
	return runParseToolCallPhases(ctx, defaultParseToolCallPhases())
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
