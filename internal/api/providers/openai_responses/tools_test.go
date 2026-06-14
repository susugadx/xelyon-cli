package openairesponses

import (
	"context"
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
