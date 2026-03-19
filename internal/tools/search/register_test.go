package search

import "testing"

func TestSearchCodeToolParameters_RemoveTokenBudget(t *testing.T) {
	params := (&SearchCodeTool{}).Parameters()

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", params["properties"])
	}
	if len(properties) != 10 {
		t.Fatalf("expected 10 parameters after removing token_budget, got %d", len(properties))
	}
	if _, exists := properties["token_budget"]; exists {
		t.Fatal("token_budget should not be exposed in the schema")
	}
}
