package openai

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
)

// GetOpenAIToolDefinitions は組み込みツール定義を OpenAI 形式で返す
// ToolRegistry から自動生成
func GetOpenAIToolDefinitions() []api.OpenAITool {
	return GetOpenAIToolDefinitionsWithContext(context.Background())
}

// GetOpenAIToolDefinitionsWithContext は request context の Registry を使って OpenAI 形式のツール定義を返す。
func GetOpenAIToolDefinitionsWithContext(ctx context.Context) []api.OpenAITool {
	defs := api.ToolDefinitionsFromContext(ctx)
	result := make([]api.OpenAITool, 0, len(defs))
	for _, def := range defs {
		result = append(result, api.OpenAITool{
			Type: "function",
			Function: &api.ToolDefinition{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  def.Parameters,
			},
		})
	}
	return result
}

// GetCombinedOpenAITools は組み込みツール + MCPツールを返す
// 重複するツール名がある場合は最初に登録されたものを優先
func GetCombinedOpenAITools(mcpTools []api.ToolDefinition) []api.OpenAITool {
	return GetCombinedOpenAIToolsWithContext(context.Background(), mcpTools)
}

// GetCombinedOpenAIToolsWithContext は request context の Registry を使って組み込みツール + MCPツールを返す。
func GetCombinedOpenAIToolsWithContext(ctx context.Context, mcpTools []api.ToolDefinition) []api.OpenAITool {
	defs := api.ToolDefinitionsWithAdditional(ctx, mcpTools)
	result := make([]api.OpenAITool, 0, len(defs))
	for _, def := range defs {
		defCopy := def
		result = append(result, api.OpenAITool{
			Type:     "function",
			Function: &defCopy,
		})
	}
	return result
}

// ConvertToolCallToToolJSON は OpenAI-compatible tool_call を内部 JSON 形式に変換します。
func ConvertToolCallToToolJSON(tc *api.OpenAIToolCall) (string, error) {
	return openairesponses.ConvertToolCallToToolJSON(tc)
}

// GetToolDefinitionNames は定義済みツール名一覧を返す（テスト用）
func GetToolDefinitionNames() []string {
	defs := api.ToolDefinitionsFromContext(context.Background())
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}
