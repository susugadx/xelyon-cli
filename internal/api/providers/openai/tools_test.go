package openai

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools/applypatch"
)

func TestGetCombinedOpenAITools_DeduplicatesMCPTools(t *testing.T) {
	tools := GetCombinedOpenAITools([]api.ToolDefinition{
		{
			Name:        "read_file",
			Description: "duplicate built-in",
			Parameters:  map[string]interface{}{"type": "object"},
		},
		{
			Name:        "custom_tool",
			Description: "custom mcp tool",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	})

	readFileCount := 0
	foundCustom := false
	for _, tool := range tools {
		if tool.Function == nil {
			t.Fatalf("tool.Function should not be nil: %+v", tool)
		}
		if tool.Function.Name == "read_file" {
			readFileCount++
		}
		if tool.Function.Name == "custom_tool" {
			foundCustom = true
		}
	}

	if readFileCount != 1 {
		t.Fatalf("read_file count = %d, want 1", readFileCount)
	}
	if !foundCustom {
		t.Fatal("custom_tool should be included")
	}
}

func TestGetToolDefinitionNames_IncludesBuiltins(t *testing.T) {
	names := GetToolDefinitionNames()
	if len(names) == 0 {
		t.Fatal("GetToolDefinitionNames() should not be empty")
	}

	required := map[string]bool{
		"read_file":  false,
		"write_file": false,
		"bash":       false,
	}
	for _, name := range names {
		if _, ok := required[name]; ok {
			required[name] = true
		}
	}

	for name, found := range required {
		if !found {
			t.Fatalf("required tool %q not found in %v", name, names)
		}
	}
}

func TestGetOpenAIToolDefinitions_ApplyPatchDescriptionUnchanged(t *testing.T) {
	applyPatch := &applypatch.ApplyPatchTool{}
	ctx := api.WithToolDefinitions(context.Background(), []api.ToolDefinition{{
		Name:        applyPatch.Name(),
		Description: applyPatch.Description(),
		Parameters:  applyPatch.Parameters(),
	}})

	tools := GetOpenAIToolDefinitionsWithContext(ctx)
	if len(tools) != 1 || tools[0].Function == nil {
		t.Fatalf("OpenAI tools = %+v, want one function tool", tools)
	}
	desc := tools[0].Function.Description
	if desc != applyPatch.Description() {
		t.Fatalf("OpenAI apply_patch description changed")
	}
	if !strings.Contains(desc, "Use the `apply_patch` shell command to edit files.") {
		t.Fatalf("OpenAI apply_patch description no longer contains the existing shell-command wording")
	}
}
