package gemini

import (
	"context"
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const geminiApplyPatchDescription = "Use the apply_patch function tool by passing the complete patch text in the patch argument. Do not include markdown fences. The patch must start with *** Begin Patch and end with *** End Patch. One patch may change multiple files."

// GetGeminiToolDefinitions returns all tool definitions for Function Calling API
// ToolRegistry から自動生成
func GetGeminiToolDefinitions() []api.GeminiToolConfig {
	return GetGeminiToolDefinitionsWithContext(context.Background())
}

// GetGeminiToolDefinitionsWithContext は request context の Registry を使って Function Calling API 用のツール定義を返す。
func GetGeminiToolDefinitionsWithContext(ctx context.Context) []api.GeminiToolConfig {
	defs := api.ToolDefinitionsFromContext(ctx)
	declarations := make([]api.GeminiFunctionDeclaration, 0, len(defs))

	for _, def := range defs {
		declarations = append(declarations, api.GeminiFunctionDeclaration{
			Name:        def.Name,
			Description: geminiToolDescription(def),
			Parameters:  convertToGeminiSchema(def.Parameters),
		})
	}

	return []api.GeminiToolConfig{{FunctionDeclarations: declarations}}
}

// convertToGeminiSchema は map[string]interface{} を GeminiParameterSchema に変換
func convertToGeminiSchema(params map[string]interface{}) *api.GeminiParameterSchema {
	return api.ConvertJSONSchemaToGeminiParameterSchema(params)
}

func geminiToolDescription(def api.ToolDefinition) string {
	if def.Name == "apply_patch" {
		return geminiApplyPatchDescription
	}
	return def.Description
}

// GetToolDefinitionNames returns all defined tool names for testing
func GetToolDefinitionNames() []string {
	defs := api.ToolDefinitionsFromContext(context.Background())
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}

// internalToolCall は内部ツール呼び出し形式（JSON出力順序を保証）
type internalToolCall struct {
	Tool             string           `json:"tool"`
	Args             map[string]any   `json:"args"`
	ThoughtSignature string           `json:"thought_signature,omitempty"` // Gemini 3: 思考署名
	ThoughtParts     []map[string]any `json:"thought_parts,omitempty"`     // Gemini 3: thought パート情報
}

// convertFunctionCallToToolJSON converts Gemini FunctionCall to internal tool JSON format
// Returns: {"tool": "read_file", "args": {"path": "/path/to/file"}}
// NOTE: 構造体を使用して "tool" が "args" より前に来ることを保証
// ParseToolCalls は {"tool" パターンで検索するため、この順序が重要
func convertFunctionCallToToolJSON(fc *api.GeminiFunctionCall) string {
	toolCall := internalToolCall{
		Tool:             fc.Name,
		Args:             fc.Args,
		ThoughtSignature: fc.ThoughtSignature,
		ThoughtParts:     fc.ThoughtParts,
	}
	jsonBytes, _ := json.Marshal(toolCall)
	return string(jsonBytes)
}

// convertFunctionCallToDisplayJSON は表示用のツールJSON（signature除外）を返す
// 重複排除キーとしても使用。thought_signature/thought_parts はユーザーに見せない。
func convertFunctionCallToDisplayJSON(fc *api.GeminiFunctionCall) string {
	toolCall := struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}{
		Tool: fc.Name,
		Args: fc.Args,
	}
	jsonBytes, _ := json.Marshal(toolCall)
	return string(jsonBytes)
}

// GetCombinedToolDefinitions は組み込みツール + MCPツールの定義を返す
// 重複するツール名がある場合は最初に登録されたものを優先
func GetCombinedToolDefinitions(mcpTools []api.ToolDefinition) []api.GeminiToolConfig {
	return GetCombinedToolDefinitionsWithContext(context.Background(), mcpTools)
}

// GetCombinedToolDefinitionsWithContext は request context の Registry を使って組み込みツール + MCPツール定義を返す。
func GetCombinedToolDefinitionsWithContext(ctx context.Context, mcpTools []api.ToolDefinition) []api.GeminiToolConfig {
	defs := api.ToolDefinitionsWithAdditional(ctx, mcpTools)
	declarations := make([]api.GeminiFunctionDeclaration, 0, len(defs))
	for _, def := range defs {
		declarations = append(declarations, api.GeminiFunctionDeclaration{
			Name:        def.Name,
			Description: geminiToolDescription(def),
			Parameters:  convertToGeminiSchema(def.Parameters),
		})
	}

	return []api.GeminiToolConfig{{FunctionDeclarations: declarations}}
}
