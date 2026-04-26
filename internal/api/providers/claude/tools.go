package claude

import (
	"context"
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// ClaudeTool は Anthropic Claude API 用のツール定義
// OpenAI とは異なり、フラットな構造で input_schema を使用
type ClaudeTool struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	InputSchema  map[string]interface{} `json:"input_schema"`
	CacheControl *api.CacheControl      `json:"cache_control,omitempty"`
}

// GetClaudeToolDefinitions は組み込みツール定義を Claude 形式で返す
// ToolRegistry から自動生成
func GetClaudeToolDefinitions() []ClaudeTool {
	return GetClaudeToolDefinitionsWithContext(context.Background())
}

// GetClaudeToolDefinitionsWithContext は request context の Registry を使って Claude 形式のツール定義を返す。
func GetClaudeToolDefinitionsWithContext(ctx context.Context) []ClaudeTool {
	defs := api.ToolDefinitionsFromContext(ctx)
	result := make([]ClaudeTool, 0, len(defs))
	for _, def := range defs {
		result = append(result, ClaudeTool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.Parameters, // Parameters → InputSchema
		})
	}
	return result
}

// ConvertOpenAIToolToClaude は OpenAI 形式のツールを Claude 形式に変換する
func ConvertOpenAIToolToClaude(tool api.ToolDefinition) ClaudeTool {
	return ClaudeTool{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: tool.Parameters,
	}
}

// GetCombinedClaudeToolsWithContext は request context の Registry/Config を使って組み込みツール + MCPツールを返す。
func GetCombinedClaudeToolsWithContext(ctx context.Context, mcpTools []api.ToolDefinition) []ClaudeTool {
	defs := api.ToolDefinitionsWithAdditional(ctx, mcpTools)
	result := make([]ClaudeTool, 0, len(defs))
	for _, def := range defs {
		result = append(result, ConvertOpenAIToolToClaude(def))
	}

	// BP#2: ツール定義末尾に cache_control を設定
	if len(result) > 0 {
		cfg := config.FromContext(ctx)
		if cfg != nil && cfg.PromptCache.Enabled {
			result[len(result)-1].CacheControl = api.NewCacheControlWithConfig(cfg)
		}
	}

	return result
}

// internalToolCall は内部ツール呼び出し形式（JSON出力順序を保証）
type internalToolCall struct {
	ID   string         `json:"id,omitempty"` // Claude の tool_use ID
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// ConvertToolUseToToolJSON は Claude の tool_use を内部JSON形式に変換
// Returns: {"id": "toolu_01ABC...", "tool": "read_file", "args": {"path": "/path/to/file"}}
func ConvertToolUseToToolJSON(id, name string, input map[string]interface{}) (string, error) {
	// input の型を map[string]any に変換
	args := make(map[string]any, len(input))
	for k, v := range input {
		args[k] = v
	}

	toolCall := internalToolCall{
		ID:   id,
		Tool: name,
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
