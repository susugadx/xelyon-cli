package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CallTool はMCPツールを呼び出す（リトライ付き）
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (string, error) {
	session, ok := m.sessions[serverName]
	if !ok {
		return "", fmt.Errorf("MCP server '%s' not connected", serverName)
	}

	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}

	result, err := m.callToolWithRetry(ctx, session, toolName, params)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("%s", toolResultErrorMessage(result))
	}

	return toolResultText(result), nil
}
