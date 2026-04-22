package repomap

import "testing"

func TestNewSymbolScanStrategyWithPatternDefinitions_UsesInjectedDefinitions(t *testing.T) {
	defs := []languagePattern{
		{
			Extensions: []string{".go"},
			Patterns:   []string{`^func `},
		},
	}

	strategy := newSymbolScanStrategyWithPatternDefinitions(&ProjectMap{}, defs)
	if len(strategy.patternDefinitions) != 1 {
		t.Fatalf("pattern definition length = %d, want 1", len(strategy.patternDefinitions))
	}
	if strategy.patternDefinitions[0].Patterns[0] != `^func ` {
		t.Fatalf("first pattern = %q, want %q", strategy.patternDefinitions[0].Patterns[0], `^func `)
	}
	if strategy.astScanner == nil {
		t.Fatal("ast scanner should be configured")
	}
	if strategy.patternScanner == nil {
		t.Fatal("pattern scanner should be configured")
	}
}
