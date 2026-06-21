package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestMCPToolWrapper_ConvertArgsWithSchema(t *testing.T) {
	t.Run("converts scalar types from schema", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			inputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"count":{"type":"integer"},
					"ratio":{"type":"number"},
					"enabled":{"type":"boolean"},
					"name":{"type":"string"}
				}
			}`),
		}

		got := wrapper.convertArgsWithSchema(map[string]string{
			"count":   "42",
			"ratio":   "3.14",
			"enabled": "true",
			"name":    "xelyon",
		})

		if got["count"] != int64(42) {
			t.Fatalf("count = %#v, want int64(42)", got["count"])
		}
		if got["ratio"] != 3.14 {
			t.Fatalf("ratio = %#v, want 3.14", got["ratio"])
		}
		if got["enabled"] != true {
			t.Fatalf("enabled = %#v, want true", got["enabled"])
		}
		if got["name"] != "xelyon" {
			t.Fatalf("name = %#v, want xelyon", got["name"])
		}
	})

	t.Run("falls back to strings when schema or value is invalid", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			inputSchema: json.RawMessage(`{"properties":{"count":{"type":"integer"}}`),
		}

		got := wrapper.convertArgsWithSchema(map[string]string{"count": "not-a-number"})
		if got["count"] != "not-a-number" {
			t.Fatalf("count = %#v, want original string", got["count"])
		}
	})
}

func TestMCPToolWrapper_ValidateArgs(t *testing.T) {
	t.Run("missing required argument returns error", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			toolName: "echo",
			inputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"path":{"type":"string"}
				},
				"required":["path"]
			}`),
		}

		err := wrapper.validateArgs(io.Discard, map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "required argument 'path' is missing") {
			t.Fatalf("validateArgs() error = %v, want missing required argument", err)
		}
	})

	t.Run("invalid schema emits warning and continues", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			toolName:    "echo",
			inputSchema: json.RawMessage(`{invalid json`),
		}
		var out bytes.Buffer

		err := wrapper.validateArgs(&out, map[string]string{"path": "test.txt"})
		if err != nil {
			t.Fatalf("validateArgs() error = %v, want nil", err)
		}
		if !strings.Contains(out.String(), "Failed to parse input schema") {
			t.Fatalf("warning output = %q, want schema parse warning", out.String())
		}
	})

	t.Run("invalid property schema emits warning and continues", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			toolName: "echo",
			inputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"path":"invalid"
				}
			}`),
		}
		var out bytes.Buffer

		err := wrapper.validateArgs(&out, map[string]string{"path": "test.txt"})
		if err != nil {
			t.Fatalf("validateArgs() error = %v, want nil", err)
		}
		if !strings.Contains(out.String(), "Invalid property schema for path") {
			t.Fatalf("warning output = %q, want property warning", out.String())
		}
	})
}

func TestMCPToolWrapper_Run(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	t.Run("returns validation error before confirmation", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			manager:    NewManager(),
			serverName: "missing",
			toolName:   "echo",
			inputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"path":{"type":"string"}},
				"required":["path"]
			}`),
		}

		var stdout bytes.Buffer
		result, fileChange, err := wrapper.Run(newMCPExecutionContext("", &stdout), map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "required argument 'path' is missing") {
			t.Fatalf("Run() error = %v, want validation error", err)
		}
		if fileChange != nil {
			t.Fatalf("fileChange = %#v, want nil", fileChange)
		}
		if !strings.Contains(result, "Validation Error: required argument 'path' is missing") {
			t.Fatalf("result = %q, want validation error message", result)
		}
	})

	t.Run("reject returns without calling tool", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			manager:    NewManager(),
			serverName: "missing",
			toolName:   "echo",
		}

		var stdout bytes.Buffer
		result, fileChange, err := wrapper.Run(newMCPExecutionContext("n\n", &stdout), map[string]string{"name": "tester"})
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
		if fileChange != nil {
			t.Fatalf("fileChange = %#v, want nil", fileChange)
		}
		if result != "User rejected MCP tool execution" {
			t.Fatalf("result = %q, want rejection message", result)
		}
	})

	t.Run("comment returns user feedback", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			manager:    NewManager(),
			serverName: "missing",
			toolName:   "echo",
		}

		var stdout bytes.Buffer
		result, fileChange, err := wrapper.Run(newMCPExecutionContext("c\nneeds a different path\n\n", &stdout), map[string]string{"name": "tester"})
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
		if fileChange != nil {
			t.Fatalf("fileChange = %#v, want nil", fileChange)
		}
		if result != "User provided feedback: needs a different path" {
			t.Fatalf("result = %q, want comment feedback", result)
		}
	})

	t.Run("call tool error is returned after approval", func(t *testing.T) {
		wrapper := &MCPToolWrapper{
			manager:    NewManager(),
			serverName: "missing",
			toolName:   "echo",
		}

		var stdout bytes.Buffer
		result, fileChange, err := wrapper.Run(newMCPExecutionContext("y\n", &stdout), map[string]string{"name": "tester"})
		if err == nil || !strings.Contains(err.Error(), "not connected") {
			t.Fatalf("Run() error = %v, want not connected", err)
		}
		if fileChange != nil {
			t.Fatalf("fileChange = %#v, want nil", fileChange)
		}
		if !strings.Contains(result, "Error: MCP server 'missing' not connected") {
			t.Fatalf("result = %q, want call tool error", result)
		}
	})

	t.Run("success formats tool result after approval", func(t *testing.T) {
		manager, _ := newInMemoryManagerWithTool(t, "echo", func(_ context.Context, _ *sdkmcp.CallToolRequest, input map[string]any) (*sdkmcp.CallToolResult, any, error) {
			name, _ := input["name"].(string)
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{
					&sdkmcp.TextContent{Text: "Hello " + name},
				},
			}, nil, nil
		})
		wrapper := &MCPToolWrapper{
			manager:    manager,
			serverName: "test-server",
			toolName:   "echo",
		}

		var stdout bytes.Buffer
		result, fileChange, err := wrapper.Run(newMCPExecutionContext("y\n", &stdout), map[string]string{"name": "tester"})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if fileChange != nil {
			t.Fatalf("fileChange = %#v, want nil", fileChange)
		}
		if result != "Hello tester\n" {
			t.Fatalf("result = %q, want tool output", result)
		}
	})

	t.Run("deadline exceeded returns timeout error", func(t *testing.T) {
		manager, _ := newInMemoryManagerWithTool(t, "slow", func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ map[string]any) (*sdkmcp.CallToolResult, any, error) {
			<-ctx.Done()
			return nil, nil, ctx.Err()
		})
		wrapper := &MCPToolWrapper{
			manager:     manager,
			serverName:  "test-server",
			toolName:    "slow",
			callTimeout: 20 * time.Millisecond,
		}

		var stdout bytes.Buffer
		result, fileChange, err := wrapper.Run(newMCPExecutionContext("y\n", &stdout), map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("Run() error = %v, want timeout error", err)
		}
		if fileChange != nil {
			t.Fatalf("fileChange = %#v, want nil", fileChange)
		}
		if !strings.Contains(result, "timed out") {
			t.Fatalf("result = %q, want timeout message", result)
		}
	})
}

func newMCPExecutionContext(input string, stdout io.Writer) tools.ExecutionContext {
	runtime := uiruntime.NewRuntime(strings.NewReader(input), stdout, stdout)
	return tools.ExecutionContext{
		Context: context.Background(),
		Stdin:   runtime.Input(),
		Stdout:  runtime.Output(),
		Stderr:  runtime.ErrorOutput(),
		Runtime: runtime,
		Config:  config.DefaultConfig(),
	}
}
