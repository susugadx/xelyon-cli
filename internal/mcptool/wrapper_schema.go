package mcptool

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// convertArgsWithSchema はスキーマに基づいて引数の型を変換する
func (w *Wrapper) convertArgsWithSchema(args map[string]string) map[string]any {
	anyArgs := make(map[string]any)

	// スキーマが空の場合は文字列のまま返す
	if len(w.inputSchema) == 0 || string(w.inputSchema) == "null" {
		for k, v := range args {
			anyArgs[k] = v
		}
		return anyArgs
	}

	// JSONスキーマをパース
	var schema map[string]any
	if err := json.Unmarshal(w.inputSchema, &schema); err != nil {
		// パースエラーの場合は文字列のまま返す
		for k, v := range args {
			anyArgs[k] = v
		}
		return anyArgs
	}

	// プロパティ情報を取得
	properties, _ := schema["properties"].(map[string]any)

	for k, v := range args {
		converted := false
		// スキーマに型情報があれば変換
		if properties != nil {
			if propInfo, ok := properties[k].(map[string]any); ok {
				if propType, ok := propInfo["type"].(string); ok {
					switch propType {
					case "integer":
						if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
							anyArgs[k] = intVal
							converted = true
						}
					case "number":
						if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
							anyArgs[k] = floatVal
							converted = true
						}
					case "boolean":
						if boolVal, err := strconv.ParseBool(v); err == nil {
							anyArgs[k] = boolVal
							converted = true
						}
					case "array", "object":
						if structured, err := parseStructuredArg(propType, k, v); err == nil {
							anyArgs[k] = structured
							converted = true
						}
					}
				}
			}
		}
		// 変換できない場合は文字列のまま
		if !converted {
			anyArgs[k] = v
		}
	}

	return anyArgs
}

// ConvertArgsWithSchema は schema に基づく引数変換を実行する。
func (w *Wrapper) ConvertArgsWithSchema(args map[string]string) map[string]any {
	return w.convertArgsWithSchema(args)
}

func parseStructuredArg(propType, argName, rawValue string) (any, error) {
	var decoded any
	if err := json.Unmarshal([]byte(rawValue), &decoded); err != nil {
		return nil, fmt.Errorf("argument '%s' must be valid JSON %s: %w", argName, propType, err)
	}

	switch propType {
	case "array":
		arrayValue, ok := decoded.([]any)
		if !ok {
			return nil, fmt.Errorf("argument '%s' must be a JSON array", argName)
		}
		return arrayValue, nil
	case "object":
		objectValue, ok := decoded.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("argument '%s' must be a JSON object", argName)
		}
		return objectValue, nil
	default:
		return rawValue, nil
	}
}
