package navigation

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func TestSymbolQueryMatches_MethodReceiverQualified(t *testing.T) {
	symbol := ast.Symbol{
		Name:      "Build",
		Kind:      ast.SymbolMethod,
		Signature: "func (c *Config) Build() string",
	}

	if !symbolQueryMatches(parseSymbolQuery("Build"), symbol) {
		t.Fatal("unqualified method query should still match by name")
	}
	if !symbolQueryMatches(parseSymbolQuery("Config.Build"), symbol) {
		t.Fatal("receiver-qualified query should match pointer receiver by base type")
	}
	if !symbolQueryMatches(parseSymbolQuery("(*Config).Build"), symbol) {
		t.Fatal("pointer receiver query should match pointer receiver")
	}
	if symbolQueryMatches(parseSymbolQuery("Agent.Build"), symbol) {
		t.Fatal("different receiver type must not match")
	}
}
