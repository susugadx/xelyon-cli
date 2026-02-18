package claude

import (
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
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
	defs := tools.DefaultRegistry.GetToolDefinitions()
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

// GetCombinedClaudeTools は組み込みツール + MCPツールを返す
// 重複するツール名がある場合は最初に登録されたものを優先
func GetCombinedClaudeTools(mcpTools []api.ToolDefinition) []ClaudeTool {
	result := GetClaudeToolDefinitions()
	seen := make(map[string]bool)

	// 組み込みツールの名前を記録
	for _, tool := range result {
		seen[tool.Name] = true
	}

	// MCPツール（重複チェック）
	for _, mcp := range mcpTools {
		if seen[mcp.Name] {
			continue
		}
		seen[mcp.Name] = true
		result = append(result, ConvertOpenAIToolToClaude(mcp))
	}

	// 最後のツールに cache_control を設定（プロンプトキャッシュ用ブレークポイント #1）
	cfg := config.GetGlobalConfig()
	if len(result) > 0 && cfg != nil && cfg.PromptCache.Enabled {
		result[len(result)-1].CacheControl = &api.CacheControl{Type: "ephemeral"}
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
	defs := tools.DefaultRegistry.GetToolDefinitions()
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}
