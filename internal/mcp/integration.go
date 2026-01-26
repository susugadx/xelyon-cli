package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RegisterToToolRegistry はMCPツールをTool Registryに登録
func (m *Manager) RegisterToToolRegistry(registry *tools.Registry) {
	for _, mcpTool := range m.tools {
		// クロージャで値をキャプチャ
		tool := mcpTool

		// MCPツール用のラッパーを作成
		wrapper := &MCPToolWrapper{
			manager:     m,
			serverName:  tool.ServerName,
			toolName:    tool.Name,
			desc:        tool.Description,
			inputSchema: tool.InputSchema,
		}

		registry.Register(wrapper)
	}
}

// MCPToolWrapper はMCPツールをTool interfaceにラップ
type MCPToolWrapper struct {
	manager     *Manager
	serverName  string
	toolName    string
	desc        string
	inputSchema json.RawMessage // JSONスキーマ情報
}

// Name はツール名を返す（mcp_<server>_<tool> 形式、特殊文字を置換）
func (w *MCPToolWrapper) Name() string {
	// 特殊文字をアンダースコアに置換
	safeServer := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, w.serverName)

	safeTool := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, w.toolName)

	return fmt.Sprintf("mcp_%s_%s", safeServer, safeTool)
}

// Description はツールの説明を返す
func (w *MCPToolWrapper) Description() string {
	if w.desc != "" {
		return w.desc
	}
	return fmt.Sprintf("MCP tool: %s from server %s", w.toolName, w.serverName)
}

// Parameters はツールのパラメータ定義を返す
func (w *MCPToolWrapper) Parameters() map[string]interface{} {
	// inputSchemaをそのままmap[string]interface{}に変換
	if len(w.inputSchema) == 0 || string(w.inputSchema) == "null" {
		return map[string]interface{}{
			"type":                 "object",
			"properties":           map[string]interface{}{},
			"additionalProperties": false,
		}
	}

	var params map[string]interface{}
	if err := json.Unmarshal(w.inputSchema, &params); err != nil {
		return map[string]interface{}{
			"type":                 "object",
			"properties":           map[string]interface{}{},
			"additionalProperties": false,
		}
	}
	return params
}

// Run はツールを実行
func (w *MCPToolWrapper) Run(args map[string]string) (string, *tools.FileChange, error) {
	// 引数バリデーション（簡易版）
	if err := w.validateArgs(args); err != nil {
		return fmt.Sprintf("Validation Error: %v", err), nil, err
	}

	// スキーマに基づいて型変換（string → number/integer/boolean）
	anyArgs := w.convertArgsWithSchema(args)

	// タイムアウト付きコンテキスト（30秒）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := w.manager.CallTool(ctx, w.serverName, w.toolName, anyArgs)
	if err != nil {
		// タイムアウトエラーの場合
		if ctx.Err() == context.DeadlineExceeded {
			return "Error: Tool execution timed out after 30 seconds", nil, fmt.Errorf("tool execution timed out")
		}
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	// 結果をフォーマット
	formattedResult := w.formatResult(result)
	return formattedResult, nil, nil
}

// convertArgsWithSchema はスキーマに基づいて引数の型を変換する
func (w *MCPToolWrapper) convertArgsWithSchema(args map[string]string) map[string]any {
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

// validateArgs は引数を検証する（簡易版）
func (w *MCPToolWrapper) validateArgs(args map[string]string) error {
	// 空のスキーマの場合はスキップ
	if len(w.inputSchema) == 0 || string(w.inputSchema) == "null" {
		return nil
	}

	// JSONスキーマをパース（簡易実装）
	var schema map[string]any
	if err := json.Unmarshal(w.inputSchema, &schema); err != nil {
		// パースエラーは警告のみ
		fmt.Printf("⚠️  Failed to parse input schema for tool %s: %v\n", w.toolName, err)
		return nil
	}

	// 必須パラメータのチェック（簡易実装）
	if properties, ok := schema["properties"].(map[string]any); ok && properties != nil {
		for propName, propInfo := range properties {
			propMap, ok := propInfo.(map[string]any)
			if !ok || propMap == nil {
				// プロパティスキーマが不正な場合は警告してスキップ
				fmt.Printf("⚠️  Warning: Invalid property schema for %s in tool %s\n", propName, w.toolName)
				continue
			}
			if required, ok := propMap["required"].(bool); ok && required {
				if _, hasArg := args[propName]; !hasArg {
					return fmt.Errorf("required argument '%s' is missing", propName)
				}
			}
		}
	}

	return nil
}

// formatResult は結果をフォーマットする
// NOTE: 出力の切り詰めはtoken_guard.goで一元管理するため、ここでは行わない
func (w *MCPToolWrapper) formatResult(result string) string {
	// 結果が空の場合
	if result == "" {
		return "Tool executed successfully (no output)"
	}

	return result
}
