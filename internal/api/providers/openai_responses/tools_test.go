package openairesponses

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestBuildToolDefinitionsWithContextPreservesMCPToolDefinition(t *testing.T) {
	ctx := api.WithToolDefinitions(context.Background(), nil)
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string"},
		},
	}

	tools := BuildToolDefinitionsWithContext(ctx, []api.ToolDefinition{{
		Name:        "mcp_github_server_search_issues",
		Description: "Search GitHub issues",
		Parameters:  params,
	}})

	if len(tools) != 1 {
		t.Fatalf("len(BuildToolDefinitionsWithContext()) = %d, want 1: %#v", len(tools), tools)
	}
	got := tools[0]
	if got.Type != "function" {
		t.Fatalf("Tool.Type = %q, want function", got.Type)
	}
	if got.Name != "mcp_github_server_search_issues" {
		t.Fatalf("Tool.Name = %q, want MCP exported name", got.Name)
	}
	if got.Description != "Search GitHub issues" {
		t.Fatalf("Tool.Description = %q, want MCP description", got.Description)
	}
	if got.Parameters["type"] != "object" {
		t.Fatalf("Tool.Parameters = %#v, want object schema", got.Parameters)
	}
	properties, ok := got.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Tool.Parameters[properties] type = %T, want map[string]interface{}", got.Parameters["properties"])
	}
	if _, ok := properties["query"]; !ok {
		t.Fatalf("Tool.Parameters[properties] = %#v, want query property", properties)
	}
}

func TestBuildToolDefinitionsWithContextSerializesMCPEmptySchema(t *testing.T) {
	tools := BuildToolDefinitionsWithContext(
		api.WithToolDefinitions(context.Background(), nil),
		[]api.ToolDefinition{api.ConvertMCPToolToToolDefinition("mcp_server_ping", "Ping server", []byte(`null`))},
	)

	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1: %#v", len(tools), tools)
	}
	params := tools[0].Parameters
	if params["type"] != "object" {
		t.Fatalf("Parameters = %#v, want object schema", params)
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok || len(props) != 0 {
		t.Fatalf("Parameters[properties] = %#v, want empty map", params["properties"])
	}
	if params["additionalProperties"] != false {
		t.Fatalf("Parameters[additionalProperties] = %#v, want false", params["additionalProperties"])
	}

	raw, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(raw), `"parameters"`) {
		t.Fatalf("serialized tools = %s, want parameters field", raw)
	}
}
