package openairesponses

import (
	"context"
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// BuildToolDefinitionsWithContext は request context の Registry を使って Responses API 用のツール定義を返す。
func BuildToolDefinitionsWithContext(ctx context.Context, mcpTools []api.ToolDefinition) []Tool {
	defs := api.ToolDefinitionsWithAdditional(ctx, mcpTools)
	result := make([]Tool, 0, len(defs))
	for _, def := range defs {
		result = append(result, Tool{
			Type:        "function",
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}
	return result
}

type internalToolCall struct {
	ID   string         `json:"id,omitempty"`
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// ConvertToolCallToToolJSON は OpenAI tool_call を内部 JSON 形式に変換する。
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
