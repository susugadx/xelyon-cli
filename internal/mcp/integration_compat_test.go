package mcp

import (
	"encoding/json"
	"io"
	"time"

	"github.com/susugadx/xelyon-cli/internal/mcptool"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type MCPToolWrapper struct {
	manager     mcptool.ToolCaller
	serverName  string
	toolName    string
	desc        string
	inputSchema json.RawMessage
	callTimeout time.Duration
}

func (w *MCPToolWrapper) wrapper() *mcptool.Wrapper {
	return mcptool.NewWrapper(mcptool.WrapperOptions{
		Caller:      w.manager,
		ServerName:  w.serverName,
		ToolName:    w.toolName,
		Description: w.desc,
		InputSchema: w.inputSchema,
		CallTimeout: w.callTimeout,
	})
}

func (w *MCPToolWrapper) Name() string {
	return w.wrapper().Name()
}

func (w *MCPToolWrapper) Description() string {
	return w.wrapper().Description()
}

func (w *MCPToolWrapper) Parameters() map[string]interface{} {
	return w.wrapper().Parameters()
}

func (w *MCPToolWrapper) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	return w.wrapper().Run(execCtx, args)
}

func (w *MCPToolWrapper) convertArgsWithSchema(args map[string]string) map[string]any {
	return w.wrapper().ConvertArgsWithSchema(args)
}

func (w *MCPToolWrapper) validateArgs(out io.Writer, args map[string]string) error {
	return w.wrapper().ValidateArgs(out, args)
}

func (w *MCPToolWrapper) formatResult(result string) string {
	return w.wrapper().FormatResult(result)
}

func (m *Manager) RegisterToToolRegistry(registry *tools.Registry) {
	mcptool.RegisterToRegistry(registry, m, mcpDefinitionsForTest(m.tools))
}

func mcpDefinitionsForTest(tools []MCPTool) []mcptool.Definition {
	defs := make([]mcptool.Definition, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, mcptool.Definition{
			ServerName:  tool.ServerName,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
			CallTimeout: tool.CallTimeout,
		})
	}
	return defs
}
