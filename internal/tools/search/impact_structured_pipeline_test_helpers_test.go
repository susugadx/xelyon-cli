package search

import (
	"os"
	"path/filepath"
	"testing"
)

func newStructuredImpactPipelineTestBundle(root string) *SymbolBundle {
	return &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "go",
			Query:       "Run",
			Canonical:   "go|run.go|3|Run",
			DisplayName: "Run",
			Kind:        "function",
			File:        "run.go",
			Line:        3,
			EndLine:     3,
		},
		Definition: SymbolBundleDefinition{
			File:      "run.go",
			Line:      3,
			EndLine:   3,
			Signature: "func Run()",
			Body:      []string{"3: func Run() {}"},
		},
		Impact: &SymbolBundleImpact{
			RiskLevel: "low",
			RecommendedReads: []SymbolBundleItem{
				{Kind: "definition", File: "run.go", Line: 3, Snippet: "func Run()"},
				{Kind: "callers", File: "caller.go", Line: 3, Snippet: "Run()"},
			},
		},
		Debug: SymbolBundleDebug{
			FileRootPath: root,
		},
	}
}

func writeStructuredImpactPipelineTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
