package mcptool

import (
	"encoding/json"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"testing"
	"time"
)

func TestRegisterToRegistry(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterToRegistry(registry, testCaller{}, []Definition{{
		ServerName:  "server-a",
		Name:        "tool-one",
		Description: "First tool",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		CallTimeout: 7 * time.Minute,
		Approval:    mcpapproval.ModeAuto,
	}})

	tool := registry.GetTool("mcp_server_a_tool_one")
	if tool == nil {
		t.Fatal("registered MCP tool not found")
	}
	if tool.Description() != "First tool" {
		t.Fatalf("Description() = %q", tool.Description())
	}
	wrapper, ok := tool.(*Wrapper)
	if !ok {
		t.Fatalf("registered tool type = %T, want *Wrapper", tool)
	}
	if wrapper.callTimeoutDuration() != 7*time.Minute {
		t.Fatalf("callTimeoutDuration() = %v, want 7m", wrapper.callTimeoutDuration())
	}
}

func TestWrapperCallTimeoutDurationUsesLongDefault(t *testing.T) {
	wrapper := NewWrapper(WrapperOptions{
		Caller:     testCaller{},
		ServerName: "server",
		ToolName:   "tool",
	})

	if got := wrapper.callTimeoutDuration(); got != 600*time.Second {
		t.Fatalf("callTimeoutDuration() = %v, want 600s", got)
	}
}

func TestRegisterToRegistrySkipsDuplicateExportedNames(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterToRegistry(registry, testCaller{}, []Definition{
		{
			ServerName:  "server-a",
			Name:        "tool.one",
			Description: "First tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			Approval:    mcpapproval.ModeAuto,
		},
		{
			ServerName:  "server_a",
			Name:        "tool_one",
			Description: "Second tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			Approval:    mcpapproval.ModeAuto,
		},
	})

	tool := registry.GetTool("mcp_server_a_tool_one")
	if tool == nil {
		t.Fatal("registered MCP tool not found")
	}
	if tool.Description() != "First tool" {
		t.Fatalf("Description() = %q, want first duplicate to win", tool.Description())
	}
	defs := registry.GetToolDefinitions()
	if len(defs) != 1 {
		t.Fatalf("registered tool definitions = %d, want 1", len(defs))
	}
}

func TestWrapperDefaultsForDescriptionParametersAndEmptyResult(t *testing.T) {
	wrapper := NewWrapper(WrapperOptions{
		ServerName: "github",
		ToolName:   "list_issues",
	})

	if got := wrapper.Description(); got != "MCP tool: list_issues from server github" {
		t.Fatalf("Description() = %q, want default description", got)
	}
	params := wrapper.Parameters()
	if params["type"] != "object" {
		t.Fatalf("Parameters()[type] = %#v, want object", params["type"])
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok || len(props) != 0 {
		t.Fatalf("Parameters()[properties] = %#v, want empty map", params["properties"])
	}
	if params["additionalProperties"] != false {
		t.Fatalf("Parameters()[additionalProperties] = %#v, want false", params["additionalProperties"])
	}
	if got := wrapper.FormatResult(""); got != "Tool executed successfully (no output)" {
		t.Fatalf("FormatResult(\"\") = %q, want empty-result default", got)
	}
}
