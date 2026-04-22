package repomap

import (
	"strings"
	"testing"
)

func TestProjectMapCounts(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			{Path: "pkg/service.go", Symbols: []Symbol{{Name: "Build"}, {Name: "Run"}}},
			{Path: "pkg/service_test.go", Symbols: []Symbol{{Name: "TestBuild"}}},
		},
	}

	if got := pm.GetSymbolCount(); got != 3 {
		t.Fatalf("GetSymbolCount() = %d, want 3", got)
	}
	if got := pm.GetFileCount(); got != 2 {
		t.Fatalf("GetFileCount() = %d, want 2", got)
	}
	if got := (*ProjectMap)(nil).GetSymbolCount(); got != 0 {
		t.Fatalf("nil GetSymbolCount() = %d, want 0", got)
	}
	if got := (*ProjectMap)(nil).GetFileCount(); got != 0 {
		t.Fatalf("nil GetFileCount() = %d, want 0", got)
	}
}

func TestGenerateManifestFallback_BudgetAware(t *testing.T) {
	pm := &ProjectMap{MaxTokens: 200}
	fallback := pm.generateManifestFallback(2, 1)
	if !strings.Contains(fallback, "Project map omitted to stay within budget") {
		t.Fatalf("generateManifestFallback() = %q, want budget message", fallback)
	}

	tiny := &ProjectMap{MaxTokens: 1}
	if got := tiny.generateManifestFallback(10, 3); got != "" {
		t.Fatalf("generateManifestFallback() = %q, want empty string when even fallback exceeds budget", got)
	}
}
