package gemini

import (
	"context"
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
)

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
			Description: def.Description,
			Parameters:  convertToGeminiSchema(def.Parameters),
		})
	}

	return []api.GeminiToolConfig{{FunctionDeclarations: declarations}}
}

// convertToGeminiSchema は map[string]interface{} を GeminiParameterSchema に変換
func convertToGeminiSchema(params map[string]interface{}) *api.GeminiParameterSchema {
	if params == nil {
		return nil
	}

	schema := &api.GeminiParameterSchema{
		Type: "object",
	}

	// type を取得
	if t, ok := params["type"].(string); ok {
		schema.Type = t
	}

	// properties を変換
	if props, ok := params["properties"].(map[string]interface{}); ok {
		schema.Properties = make(map[string]api.GeminiPropertyDef)
		for name, propVal := range props {
			if propMap, ok := propVal.(map[string]interface{}); ok {
				propDef := api.GeminiPropertyDef{}

				if t, ok := propMap["type"].(string); ok {
					propDef.Type = t
				}
				if desc, ok := propMap["description"].(string); ok {
					propDef.Description = desc
				}
				// enum を変換
				if enumVal, ok := propMap["enum"]; ok {
					switch e := enumVal.(type) {
					case []string:
						propDef.Enum = e
					case []interface{}:
						enumStrings := make([]string, 0, len(e))
						for _, v := range e {
							if s, ok := v.(string); ok {
								enumStrings = append(enumStrings, s)
							}
						}
						propDef.Enum = enumStrings
					}
				}
				// array 型の場合は items を設定（Gemini API では必須）
				if propDef.Type == "array" {
					if itemsVal, ok := propMap["items"].(map[string]interface{}); ok {
						itemDef := convertPropertyItems(itemsVal)
						propDef.Items = &itemDef
					} else {
						// items が指定されていない場合はデフォルトで string 型
						propDef.Items = &api.GeminiPropertyDef{Type: "string"}
					}
				}

				schema.Properties[name] = propDef
			}
		}
	}

	// required を変換
	if req, ok := params["required"]; ok {
		switch r := req.(type) {
		case []string:
			schema.Required = r
		case []interface{}:
			reqStrings := make([]string, 0, len(r))
			for _, v := range r {
				if s, ok := v.(string); ok {
					reqStrings = append(reqStrings, s)
				}
			}
			schema.Required = reqStrings
		}
	}

	return schema
}

// convertPropertyItems は items の map を GeminiPropertyDef に変換する
func convertPropertyItems(itemsMap map[string]interface{}) api.GeminiPropertyDef {
	itemDef := api.GeminiPropertyDef{}

	if t, ok := itemsMap["type"].(string); ok {
		itemDef.Type = t
	} else {
		// type が指定されていない場合はデフォルトで string
		itemDef.Type = "string"
	}
	if desc, ok := itemsMap["description"].(string); ok {
		itemDef.Description = desc
	}
	// enum 対応
	if enumVal, ok := itemsMap["enum"]; ok {
		switch e := enumVal.(type) {
		case []string:
			itemDef.Enum = e
		case []interface{}:
			enumStrings := make([]string, 0, len(e))
			for _, v := range e {
				if s, ok := v.(string); ok {
					enumStrings = append(enumStrings, s)
				}
			}
			itemDef.Enum = enumStrings
		}
	}

	return itemDef
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
	defs := api.ToolDefinitionsFromContext(ctx)
	declarations := make([]api.GeminiFunctionDeclaration, 0, len(defs)+len(mcpTools))
	seen := make(map[string]bool)

	// 組み込みツール（Registry から生成）
	for _, def := range defs {
		if seen[def.Name] {
			continue
		}
		seen[def.Name] = true
		declarations = append(declarations, api.GeminiFunctionDeclaration{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  convertToGeminiSchema(def.Parameters),
		})
	}

	// MCPツール（ToolDefinitionからGeminiFunctionDeclarationに変換）
	for _, mcp := range mcpTools {
		if seen[mcp.Name] {
			continue
		}
		seen[mcp.Name] = true
		declarations = append(declarations, api.GeminiFunctionDeclaration{
			Name:        mcp.Name,
			Description: mcp.Description,
			Parameters:  convertToGeminiSchema(mcp.Parameters),
		})
	}

	return []api.GeminiToolConfig{{FunctionDeclarations: declarations}}
}
