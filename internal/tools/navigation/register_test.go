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

	params := tool.Parameters()
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}

	if _, ok := props["symbol"]; !ok {
		t.Error("expected 'symbol' property")
	}
	if _, ok := props["path"]; !ok {
		t.Error("expected 'path' property")
	}
	if _, ok := props["mode"]; !ok {
		t.Error("expected 'mode' property")
	}
	if symbolProp, ok := props["symbol"].(map[string]interface{}); ok {
		desc, _ := symbolProp["description"].(string)
		if !strings.Contains(desc, "Config.Build") {
			t.Errorf("expected receiver-qualified example in symbol description, got %q", desc)
		}
	}
	if modeProp, ok := props["mode"].(map[string]interface{}); ok {
		enumVals, ok := modeProp["enum"].([]string)
		if !ok || len(enumVals) != 2 || enumVals[0] != "summary" || enumVals[1] != "full" {
			t.Errorf("expected mode enum [summary full], got %#v", modeProp["enum"])
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required array")
	}
	if len(required) != 1 || required[0] != "symbol" {
		t.Errorf("expected required=[symbol], got %v", required)
	}
}

func TestInspectSymbolTool_Registration(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTools(registry)

	if !registry.HasTool("inspect_symbol") {
		t.Error("expected inspect_symbol to be registered")
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

func TestInspectSymbolTool_DefaultRegistration(t *testing.T) {
	// init() により DefaultRegistry に登録されている
	if !tools.DefaultRegistry.HasTool("inspect_symbol") {
		t.Error("expected inspect_symbol in DefaultRegistry")
	}
}
