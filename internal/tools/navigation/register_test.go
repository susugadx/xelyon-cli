package navigation

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestInspectSymbolTool_Schema(t *testing.T) {
	tool := &InspectSymbolTool{}

	if tool.Name() != "inspect_symbol" {
		t.Errorf("expected name 'inspect_symbol', got %q", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
	if !strings.Contains(desc, "Internal") {
		t.Error("description should indicate internal-only usage")
	}

	params := tool.Parameters()
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}
	if len(props) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(props))
	}

	if _, ok := props["symbol"]; !ok {
		t.Error("expected 'symbol' property")
	}
	if _, ok := props["path"]; !ok {
		t.Error("expected 'path' property")
	}
}

func TestInspectSymbolTool_ExplicitRegistration(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTools(registry)

	if !registry.HasTool("inspect_symbol") {
		t.Error("expected inspect_symbol to be registered via RegisterTools")
	}
}

func TestInspectSymbolTool_Run_EmptySymbol(t *testing.T) {
	tool := &InspectSymbolTool{}
	result, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"symbol": "",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if change != nil {
		t.Error("expected nil change")
	}
	if result == "" {
		t.Error("expected non-empty error result")
	}
}

func TestInspectSymbolTool_NotInDefaultRegistry(t *testing.T) {
	// inspect_symbol は公開ツールとして廃止済み — DefaultRegistry に登録されていないこと
	if tools.DefaultRegistry.HasTool("inspect_symbol") {
		t.Error("inspect_symbol should NOT be in DefaultRegistry (integrated into search_code)")
	}
}
