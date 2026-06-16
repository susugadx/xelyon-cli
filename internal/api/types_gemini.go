package api

import (
	"encoding/json"
	"fmt"
	"io"
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
	Type        string                       `json:"type"`
	Description string                       `json:"description,omitempty"`
	Enum        []string                     `json:"enum,omitempty"`
	Items       *GeminiPropertyDef           `json:"items,omitempty"` // array型用
	Properties  map[string]GeminiPropertyDef `json:"properties,omitempty"`
	Required    []string                     `json:"required,omitempty"`
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

// ConvertJSONSchemaToGeminiParameterSchema はJSON SchemaをGemini Function Calling用スキーマへ変換する。
func ConvertJSONSchemaToGeminiParameterSchema(schema map[string]any) *GeminiParameterSchema {
	if schema == nil {
		return nil
	}

	params := &GeminiParameterSchema{Type: "object"}
	if t, ok := schema["type"].(string); ok && t != "" {
		params.Type = t
	}
	if props := convertGeminiProperties(schema["properties"]); len(props) > 0 {
		params.Properties = props
	}
	if required := convertGeminiRequired(schema["required"]); len(required) > 0 {
		params.Required = required
	}
	return params
}

// ConvertJSONSchemaPropertyToGeminiDef はJSON Schemaプロパティ定義をGeminiPropertyDefに変換する。
func ConvertJSONSchemaPropertyToGeminiDef(propMap map[string]any) GeminiPropertyDef {
	def := GeminiPropertyDef{}
	if t, ok := propMap["type"].(string); ok {
		def.Type = t
	}
	if d, ok := propMap["description"].(string); ok {
		def.Description = d
	}
	if enum := convertGeminiEnum(propMap["enum"]); len(enum) > 0 {
		def.Enum = enum
	}
	if props := convertGeminiProperties(propMap["properties"]); len(props) > 0 {
		def.Properties = props
	}
	if required := convertGeminiRequired(propMap["required"]); len(required) > 0 {
		def.Required = required
	}
	if def.Type == "" && len(def.Properties) > 0 {
		def.Type = "object"
	}
	if def.Type == "array" {
		if itemsVal, ok := propMap["items"].(map[string]any); ok {
			itemDef := ConvertJSONSchemaPropertyToGeminiDef(itemsVal)
			if itemDef.Type == "" {
				itemDef.Type = "string"
			}
			def.Items = &itemDef
		} else {
			def.Items = &GeminiPropertyDef{Type: "string"}
		}
	}
	return def
}

// convertPropertyDef は既存テスト互換用の内部エイリアス。
func convertPropertyDef(propMap map[string]any) GeminiPropertyDef {
	return ConvertJSONSchemaPropertyToGeminiDef(propMap)
}

func convertGeminiEnum(value any) []string {
	switch enum := value.(type) {
	case []string:
		return append([]string(nil), enum...)
	case []any:
		values := make([]string, 0, len(enum))
		for _, e := range enum {
			if s, ok := e.(string); ok {
				values = append(values, s)
			}
		}
		return values
	default:
		return nil
	}
}

func convertGeminiProperties(value any) map[string]GeminiPropertyDef {
	props, ok := value.(map[string]any)
	if !ok || len(props) == 0 {
		return nil
	}
	geminiProps := make(map[string]GeminiPropertyDef, len(props))
	for propName, propInfo := range props {
		if propMap, ok := propInfo.(map[string]any); ok {
			geminiProps[propName] = ConvertJSONSchemaPropertyToGeminiDef(propMap)
		}
	}
	return geminiProps
}

func convertGeminiRequired(value any) []string {
	switch req := value.(type) {
	case []string:
		return append([]string(nil), req...)
	case []any:
		required := make([]string, 0, len(req))
		for _, r := range req {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
		return required
	default:
		return nil
	}
}

// ConvertMCPToolToGeminiDeclaration はMCPツールをGemini Function Declaration形式に変換
// MCPのInputSchemaはJSON Schema形式でGeminiと互換性がある
// XELYON_DEBUG_GEMINI=1 でデバッグログを debugOut に出力
func ConvertMCPToolToGeminiDeclaration(name, description string, inputSchema json.RawMessage, debugOut io.Writer) GeminiFunctionDeclaration {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	schema := MCPInputSchemaParameters(inputSchema)

	geminiParams := ConvertJSONSchemaToGeminiParameterSchema(schema)
	if props, ok := schema["properties"].(map[string]any); ok {
		for propName := range props {
			def := geminiParams.Properties[propName]

			// デバッグログ: array型のitems情報
			if debug && def.Type == "array" {
				if def.Items != nil {
					fmt.Fprintf(debugOut, "[DEBUG Gemini] Tool %s, property %s: array items.type=%q\n",
						name, propName, def.Items.Type)
				} else {
					fmt.Fprintf(debugOut, "[DEBUG Gemini] Tool %s, property %s: array but items is nil\n",
						name, propName)
				}
			}
		}
	}

	if debug {
		jsonData, _ := json.Marshal(geminiParams)
		fmt.Fprintf(debugOut, "[DEBUG Gemini] Tool %s parameters: %s\n", name, string(jsonData))
	}

	return GeminiFunctionDeclaration{
		Name:        name,
		Description: description,
		Parameters:  geminiParams,
	}
}
