package openai

import (
	"context"
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
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
	result := GetOpenAIToolDefinitionsWithContext(ctx)
	seen := make(map[string]bool)

	// 組み込みツールの名前を記録
	for _, tool := range result {
		if tool.Function != nil {
			seen[tool.Function.Name] = true
		}
	}

	// MCPツール（重複チェック）
	for _, mcp := range mcpTools {
		if seen[mcp.Name] {
			continue
		}
		seen[mcp.Name] = true
		mcpCopy := mcp // ループ変数のコピー
		result = append(result, api.OpenAITool{
			Type:     "function",
			Function: &mcpCopy,
		})
	}
	return result
}

// GetResponsesToolDefinitions は Responses API 用のツール定義を返す
// Responses API は Chat Completions と異なりフラットな形式
// 重複するツール名がある場合は最初に登録されたものを優先
func GetResponsesToolDefinitions(mcpTools []api.ToolDefinition) []ResponsesTool {
	return GetResponsesToolDefinitionsWithContext(context.Background(), mcpTools)
}

// GetResponsesToolDefinitionsWithContext は request context の Registry を使って Responses API 用のツール定義を返す。
func GetResponsesToolDefinitionsWithContext(ctx context.Context, mcpTools []api.ToolDefinition) []ResponsesTool {
	defs := api.ToolDefinitionsFromContext(ctx)
	result := make([]ResponsesTool, 0, len(defs)+len(mcpTools))
	seen := make(map[string]bool)

	// 組み込みツール（Registry から生成）
	for _, def := range defs {
		if seen[def.Name] {
			continue
		}
		seen[def.Name] = true
		result = append(result, ResponsesTool{
			Type:        "function",
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}

	// MCPツール（重複チェック）
	for _, mcp := range mcpTools {
		if seen[mcp.Name] {
			continue
		}
		seen[mcp.Name] = true
		result = append(result, ResponsesTool{
			Type:        "function",
			Name:        mcp.Name,
			Description: mcp.Description,
			Parameters:  mcp.Parameters,
		})
	}

	return result
}

// internalToolCall は内部ツール呼び出し形式（JSON出力順序を保証）
type internalToolCall struct {
	ID   string         `json:"id,omitempty"` // Function Calling 用の tool_call_id
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// ConvertToolCallToToolJSON は OpenAI tool_call を内部JSON形式に変換
// Returns: {"id": "call_xxx", "tool": "read_file", "args": {"path": "/path/to/file"}}
func ConvertToolCallToToolJSON(tc *api.OpenAIToolCall) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return "", err
	}
	toolCall := internalToolCall{
		ID:   tc.ID,
		Tool: tc.Function.Name,
		Args: args,
	}
	jsonBytes, err := json.Marshal(toolCall)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
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
