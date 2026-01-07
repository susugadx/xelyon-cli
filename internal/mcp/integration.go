package mcp

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RegisterToToolRegistry はMCPツールをTool Registryに登録
func (m *Manager) RegisterToToolRegistry(registry *tools.Registry) {
	for _, mcpTool := range m.tools {
		// クロージャで値をキャプチャ
		tool := mcpTool

		// MCPツール用のラッパーを作成
		wrapper := &MCPToolWrapper{
			manager:    m,
			serverName: tool.ServerName,
			toolName:   tool.Name,
			desc:       tool.Description,
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
}

// Name はツール名を返す（mcp_<server>_<tool> 形式）
func (w *MCPToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", w.serverName, w.toolName)
}

// Run はツールを実行
func (w *MCPToolWrapper) Run(args map[string]string) (string, *tools.FileChange, error) {
	// string map を any map に変換
	anyArgs := make(map[string]any)
	for k, v := range args {
		anyArgs[k] = v
	}

	ctx := context.Background()
	result, err := w.manager.CallTool(ctx, w.serverName, w.toolName, anyArgs)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	return result, nil, nil
}
