package mcp

import (
	"context"
	"encoding/json"
	"fmt"
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
	manager    *Manager
	serverName string
	toolName   string
	desc       string
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

// Run はツールを実行
func (w *MCPToolWrapper) Run(args map[string]string) (string, *tools.FileChange, error) {
	// 引数バリデーション（簡易版）
	if err := w.validateArgs(args); err != nil {
		return fmt.Sprintf("Validation Error: %v", err), nil, err
	}

	// string map を any map に変換
	anyArgs := make(map[string]any)
	for k, v := range args {
		anyArgs[k] = v
	}

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
	if properties, ok := schema["properties"].(map[string]any); ok {
		for propName, propInfo := range properties {
			if propMap, ok := propInfo.(map[string]any); ok {
				if required, ok := propMap["required"].(bool); ok && required {
					if _, hasArg := args[propName]; !hasArg {
						return fmt.Errorf("required argument '%s' is missing", propName)
					}
				}
			}
		}
	}

	return nil
}

// formatResult は結果をフォーマットする
func (w *MCPToolWrapper) formatResult(result string) string {
	// 結果が空の場合
	if result == "" {
		return "Tool executed successfully (no output)"
	}

	// 長い出力の場合は省略
	lines := strings.Split(result, "\n")
	if len(lines) > 20 {
		truncated := strings.Join(lines[:20], "\n")
		return fmt.Sprintf("%s\n... (output truncated, %d more lines)", truncated, len(lines)-20)
	}

	return result
}
