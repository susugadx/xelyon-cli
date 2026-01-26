package openai

import (
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// GetOpenAIToolDefinitions は組み込みツール定義を OpenAI 形式で返す
// ToolRegistry から自動生成
func GetOpenAIToolDefinitions() []api.OpenAITool {
	defs := tools.DefaultRegistry.GetToolDefinitions()
	result := make([]api.OpenAITool, 0, len(defs))
	for _, def := range defs {
		result = append(result, api.OpenAITool{
			Type: "function",
			Function: &api.OpenAIToolFunction{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  def.Parameters,
			},
		})
	}
	return result
}

// GetCombinedOpenAITools は組み込みツール + MCPツールを返す
func GetCombinedOpenAITools(mcpTools []api.OpenAIToolFunction) []api.OpenAITool {
	result := GetOpenAIToolDefinitions()
	for _, mcp := range mcpTools {
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
func GetResponsesToolDefinitions(mcpTools []api.OpenAIToolFunction) []ResponsesTool {
	defs := tools.DefaultRegistry.GetToolDefinitions()
	result := make([]ResponsesTool, 0, len(defs)+len(mcpTools))

	// 組み込みツール（Registry から生成）
	for _, def := range defs {
		result = append(result, ResponsesTool{
			Type:        "function",
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}

	// MCPツール
	for _, mcp := range mcpTools {
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
	defs := tools.DefaultRegistry.GetToolDefinitions()
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}
