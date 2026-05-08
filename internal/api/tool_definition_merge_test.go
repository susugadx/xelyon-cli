package api

import (
	"context"
	"testing"
)

func TestToolDefinitionsWithAdditionalMergesAndDeduplicates(t *testing.T) {
	ctx := WithToolDefinitions(context.Background(), []ToolDefinition{{
		Name:        "read_file",
		Description: "builtin",
		Parameters:  map[string]interface{}{"type": "object"},
	}})

	got := ToolDefinitionsWithAdditional(ctx, []ToolDefinition{
		{
			Name:        "read_file",
			Description: "duplicate additional",
			Parameters:  map[string]interface{}{"type": "object"},
		},
		{
			Name:        "mcp_lookup",
			Description: "additional",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	})

	if len(got) != 2 {
		t.Fatalf("len(ToolDefinitionsWithAdditional()) = %d, want 2: %#v", len(got), got)
	}
	if got[0].Name != "read_file" || got[1].Name != "mcp_lookup" {
		t.Fatalf("ToolDefinitionsWithAdditional() = %#v, want builtin then additional", got)
	}
}

func TestToolDefinitionsWithAdditionalCanDisableAdditionalTools(t *testing.T) {
	ctx := WithToolDefinitions(context.Background(), []ToolDefinition{{
		Name:        "read_file",
		Description: "builtin",
		Parameters:  map[string]interface{}{"type": "object"},
	}})
	ctx = WithAdditionalToolDefinitionsDisabled(ctx)

	got := ToolDefinitionsWithAdditional(ctx, []ToolDefinition{{
		Name:        "mcp_lookup",
		Description: "additional",
		Parameters:  map[string]interface{}{"type": "object"},
	}})

	if len(got) != 1 {
		t.Fatalf("len(ToolDefinitionsWithAdditional()) = %d, want base-only result: %#v", len(got), got)
	}
	if got[0].Name != "read_file" {
		t.Fatalf("ToolDefinitionsWithAdditional()[0].Name = %q, want read_file", got[0].Name)
	}
}
