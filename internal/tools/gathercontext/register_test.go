package gathercontext

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRegisterToolsAndParameters(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTools(registry)

	registered := registry.GetTool("gather_context")
	if registered == nil {
		t.Fatal("expected gather_context to be registered")
	}

	params := registered.Parameters()
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}
	if len(props) != 3 {
		t.Fatalf("expected 3 public parameters, got %d", len(props))
	}
	for _, name := range []string{"query", "path", "file_filter"} {
		if _, ok := props[name]; !ok {
			t.Fatalf("expected %q parameter", name)
		}
	}
	for _, hidden := range []string{"mode", "intent", "detail", "targets", "paths", "strategy", "prefetch_count", "read_limit", "risk"} {
		if _, ok := props[hidden]; ok {
			t.Fatalf("unexpected low-level parameter %q in public schema", hidden)
		}
	}

	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "query" {
		t.Fatalf("required = %#v, want [query]", params["required"])
	}
	if additional, ok := params["additionalProperties"].(bool); !ok || additional {
		t.Fatalf("additionalProperties = %#v, want false", params["additionalProperties"])
	}
}
