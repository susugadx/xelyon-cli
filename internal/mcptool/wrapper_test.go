package mcptool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type testCaller struct{}

func (testCaller) CallTool(context.Context, string, string, map[string]any) (string, error) {
	return "ok", nil
}

func TestRegisterToRegistry(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterToRegistry(registry, testCaller{}, []Definition{{
		ServerName:  "server-a",
		Name:        "tool-one",
		Description: "First tool",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}})

	tool := registry.GetTool("mcp_server_a_tool_one")
	if tool == nil {
		t.Fatal("registered MCP tool not found")
	}
	if tool.Description() != "First tool" {
		t.Fatalf("Description() = %q", tool.Description())
	}
}
