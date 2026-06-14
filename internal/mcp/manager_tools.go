package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
)

// shouldIncludeTool はツールがフィルタを通過するか判定
// - include が設定されている場合: include に含まれるツールのみ通過（ホワイトリスト）
// - include 未設定で exclude が設定されている場合: exclude に含まれないツールが通過（ブラックリスト）
// - どちらも未設定: 全ツール通過
func shouldIncludeTool(toolName string, filter *ToolsFilter) bool {
	if filter == nil {
		return true
	}

	if len(filter.Include) > 0 {
		for _, name := range filter.Include {
			if name == toolName {
				return true
			}
		}
		return false
	}

	if len(filter.Exclude) > 0 {
		for _, name := range filter.Exclude {
			if name == toolName {
				return false
			}
		}
	}

	return true
}

func (m *Manager) refreshServerTools(
	ctx context.Context,
	serverName string,
	session *mcp.ClientSession,
	serverConfig ServerConfig,
) ([]MCPTool, toolRegistrationSummary, error) {
	listCtx, cancel := mcpServerOperationContext(ctx, serverConfig.startupTimeoutDuration())
	defer cancel()

	toolsResult, err := session.ListTools(listCtx, nil)
	if err != nil {
		return nil, toolRegistrationSummary{}, err
	}

	serverTools, summary := m.buildServerTools(serverName, session, toolsResult.Tools, serverConfig.Tools, serverConfig.toolTimeoutDuration())
	return serverTools, summary, nil
}

func (m *Manager) buildServerTools(
	serverName string,
	session *mcp.ClientSession,
	toolDefs []*mcp.Tool,
	filter *ToolsFilter,
	callTimeout time.Duration,
) ([]MCPTool, toolRegistrationSummary) {
	seenExportedNames := m.existingExportedToolNames(serverName)
	serverTools := make([]MCPTool, 0, len(toolDefs))
	summary := toolRegistrationSummary{}
	for _, tool := range toolDefs {
		if tool == nil {
			continue
		}
		if !shouldIncludeTool(tool.Name, filter) {
			summary.skipped++
			continue
		}

		exportedName := mcpnames.ExportedToolName(serverName, tool.Name)
		if seenExportedNames[exportedName] {
			fmt.Fprintf(m.out(), "⚠️  MCP tool '%s' from server '%s' skipped: exported name %q already registered\n", tool.Name, serverName, exportedName)
			summary.skipped++
			continue
		}
		seenExportedNames[exportedName] = true

		schemaBytes, _ := json.Marshal(tool.InputSchema)
		serverTools = append(serverTools, MCPTool{
			ServerName:  serverName,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schemaBytes,
			Session:     session,
			CallTimeout: callTimeout,
		})
		summary.registered++
	}

	return serverTools, summary
}

func (m *Manager) existingExportedToolNames(excludeServerName string) map[string]bool {
	seen := make(map[string]bool, len(m.tools))
	for _, tool := range m.tools {
		if tool.ServerName == excludeServerName {
			continue
		}
		seen[mcpnames.ExportedToolName(tool.ServerName, tool.Name)] = true
	}
	return seen
}

func (m *Manager) replaceServerTools(serverName string, serverTools []MCPTool) {
	m.removeServerTools(serverName)
	m.tools = append(m.tools, serverTools...)
}

func (m *Manager) removeServerTools(serverName string) {
	filteredTools := m.tools[:0]
	for _, tool := range m.tools {
		if tool.ServerName != serverName {
			filteredTools = append(filteredTools, tool)
		}
	}
	m.tools = filteredTools
}

// GetTools は利用可能なMCPツール一覧を返す
func (m *Manager) GetTools() []MCPTool {
	return m.tools
}
