package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	filter *ToolsFilter,
	replace bool,
) (toolRegistrationSummary, error) {
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return toolRegistrationSummary{}, err
	}

	summary := m.storeServerTools(serverName, session, toolsResult.Tools, filter, replace)
	return summary, nil
}

func (m *Manager) storeServerTools(
	serverName string,
	session *mcp.ClientSession,
	toolDefs []*mcp.Tool,
	filter *ToolsFilter,
	replace bool,
) toolRegistrationSummary {
	if replace {
		m.removeServerTools(serverName)
	}

	summary := toolRegistrationSummary{}
	for _, tool := range toolDefs {
		if tool == nil {
			continue
		}
		if !shouldIncludeTool(tool.Name, filter) {
			summary.skipped++
			continue
		}

		schemaBytes, _ := json.Marshal(tool.InputSchema)
		m.tools = append(m.tools, MCPTool{
			ServerName:  serverName,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schemaBytes,
			Session:     session,
		})
		summary.registered++
	}

	return summary
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
