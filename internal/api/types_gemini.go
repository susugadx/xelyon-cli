package api

import (
	"encoding/json"
	"fmt"
	"os"
)

// このファイルは Gemini Function Calling API で使用される型を定義します。
// MCPToolProvider インターフェースで使用されるため、api パッケージに配置されています。

// GeminiFunctionDeclaration - Gemini API用の関数宣言
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  *GeminiParameterSchema `json:"parameters,omitempty"`
}

// GeminiParameterSchema - パラメータスキーマ
type GeminiParameterSchema struct {
	Type       string                       `json:"type"`
	Properties map[string]GeminiPropertyDef `json:"properties,omitempty"`
	Required   []string                     `json:"required,omitempty"`
}

// GeminiPropertyDef - プロパティ定義
type GeminiPropertyDef struct {
	Type        string             `json:"type"`
	Description string             `json:"description,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
	Items       *GeminiPropertyDef `json:"items,omitempty"` // array型用
}

// GeminiToolConfig - API リクエスト用ツール設定
type GeminiToolConfig struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"function_declarations"`
}

// GeminiFunctionCall - Function Call response from Gemini
type GeminiFunctionCall struct {
	Name             string           `json:"name"`
	Args             map[string]any   `json:"args"`
	ThoughtSignature string           `json:"-"` // JSON出力には含めない（内部転送用）
	ThoughtParts     []map[string]any `json:"-"` // 同ターンの thought パート情報（内部転送用）
}

// convertPropertyDef はJSON Schemaプロパティ定義をGeminiPropertyDefに変換する
// array型の場合はitemsも再帰的に変換する
func convertPropertyDef(propMap map[string]any) GeminiPropertyDef {
	def := GeminiPropertyDef{}
	if t, ok := propMap["type"].(string); ok {
		def.Type = t
	}
	if d, ok := propMap["description"].(string); ok {
		def.Description = d
	}
	// enum対応
	if enumVal, ok := propMap["enum"].([]any); ok {
		for _, e := range enumVal {
			if s, ok := e.(string); ok {
				def.Enum = append(def.Enum, s)
			}
		}
	}
	// array型のitems対応
	if def.Type == "array" {
		if itemsVal, ok := propMap["items"].(map[string]any); ok {
			itemDef := convertPropertyDef(itemsVal)
			// items.type が空の場合はデフォルトで "string" を設定
			// Gemini APIはtypeが必須
			if itemDef.Type == "" {
				itemDef.Type = "string"
			}
			def.Items = &itemDef
		} else {
			// items が指定されていない場合もデフォルトで string 型を設定
			def.Items = &GeminiPropertyDef{Type: "string"}
		}
	}
	return def
}

// ConvertMCPToolToGeminiDeclaration はMCPツールをGemini Function Declaration形式に変換
// MCPのInputSchemaはJSON Schema形式でGeminiと互換性がある
// XELYON_DEBUG_GEMINI=1 でデバッグログを出力
func ConvertMCPToolToGeminiDeclaration(name, description string, inputSchema json.RawMessage) GeminiFunctionDeclaration {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	// スキーマが空またはnullの場合
	if len(inputSchema) == 0 || string(inputSchema) == "null" {
		return GeminiFunctionDeclaration{
			Name:        name,
			Description: description,
			Parameters:  nil,
		}
	}

	var schema map[string]any
	if err := json.Unmarshal(inputSchema, &schema); err != nil {
		// パースエラー時は空のパラメータで続行
		return GeminiFunctionDeclaration{
			Name:        name,
			Description: description,
			Parameters:  nil,
		}
	}

	// propertiesをGeminiPropertyDefに変換
	var geminiParams *GeminiParameterSchema
	if props, ok := schema["properties"].(map[string]any); ok {
		geminiProps := make(map[string]GeminiPropertyDef)
		for propName, propInfo := range props {
			if propMap, ok := propInfo.(map[string]any); ok {
				def := convertPropertyDef(propMap)
				geminiProps[propName] = def

				// デバッグログ: array型のitems情報
				if debug && def.Type == "array" {
					if def.Items != nil {
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Tool %s, property %s: array items.type=%q\n",
							name, propName, def.Items.Type)
					} else {
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Tool %s, property %s: array but items is nil\n",
							name, propName)
					}
				}
			}
		}

		var required []string
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
		}

		geminiParams = &GeminiParameterSchema{
			Type:       "object",
			Properties: geminiProps,
			Required:   required,
		}
	}

	if debug {
		jsonData, _ := json.Marshal(geminiParams)
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Tool %s parameters: %s\n", name, string(jsonData))
	}

	return GeminiFunctionDeclaration{
		Name:        name,
		Description: description,
		Parameters:  geminiParams,
	}
}
