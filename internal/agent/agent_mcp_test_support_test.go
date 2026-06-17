package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
)

type mockMCPProvider struct {
	name             string
	setMCPToolsCalls int
	lastTools        []api.ToolDefinition
}

var _ api.Provider = (*mockMCPProvider)(nil)
var _ api.MCPProvider = (*mockMCPProvider)(nil)

func (p *mockMCPProvider) Name() string { return p.name }

func (p *mockMCPProvider) ChatWithTools(_ context.Context, _ string, _ []api.Message, _ string) (string, error) {
	return "", nil
}

func (p *mockMCPProvider) SupportsImages() bool { return false }

func (p *mockMCPProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

func (p *mockMCPProvider) IsFunctionCallingEnabled() bool { return true }

func (p *mockMCPProvider) SetMCPEnabled(_ bool) {}

func (p *mockMCPProvider) SetMCPTools(tools []api.ToolDefinition) {
	p.setMCPToolsCalls++
	p.lastTools = tools
}

type mcpSurfaceTestCaller struct{}

func (mcpSurfaceTestCaller) CallTool(context.Context, string, string, map[string]any) (string, error) {
	return "ok", nil
}

func exportedMCPToolNamesForTest(mcpTools []mcp.MCPTool) []string {
	names := make([]string, 0, len(mcpTools))
	for _, tool := range mcpTools {
		names = append(names, mcpnames.ExportedToolName(tool.ServerName, tool.Name))
	}
	return names
}

func toolDefinitionNamesForTest(defs []api.ToolDefinition) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}
