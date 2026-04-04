package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestSearchCode_SymbolSingleHit(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: "."})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit, got no matches")
	}
	// symbol auto hit は inspect 形式の結果を返す（lineRangeHint を含まない）
	if strings.Contains(result, lineRangeHint) {
		t.Error("symbol auto result should not contain lineRangeHint")
	}
	if !strings.Contains(result, "NewAgent") {
		t.Error("expected symbol name in result")
	}
}

func TestSearchCode_SymbolMultipleHit(t *testing.T) {
	setupSymbolTestDir(t, "multi.go", symbolTestMultiSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "Build", Path: "."})
	if !strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected multiple symbols result, got: %s", result)
	}
}

func TestSearchCode_SymbolFallbackToText(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "NonExistentXYZ12345", Path: "."})
	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected 'No matches found', got: %s", result)
	}
}

func TestSearchCode_RegexSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// explicit mode=regex → symbol rescue/resolve をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", Mode: string(SearchModeRegex)})
	// text search 結果は lineRangeHint を含む（symbol auto は IsRegex=true で完全スキップ）
	if !strings.Contains(result, lineRangeHint) {
		t.Error("regex search should fall back to text search with lineRangeHint")
	}
}

func TestSearchCode_UnsupportedLangSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// 非対応言語 → symbol auto をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", FileType: "haskell"})
	if !strings.Contains(result, "No matches found") && !strings.Contains(result, lineRangeHint) {
		t.Error("unsupported language should fall back to text search")
	}
}

func TestSearchCode_FilePatternSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// glob パターン指定 → symbol auto をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", FilePattern: "*.go"})
	if !strings.Contains(result, lineRangeHint) {
		t.Error("file pattern should fall back to text search with lineRangeHint")
	}
}

// ── 多言語シンボル解決テスト ──

func TestSearchCode_MultiPatternGoSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// カンマ区切りで Go シンボル2つ → 両方 symbol 解決される
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent,Run", Path: "."})
	if !strings.Contains(result, "Pattern 1/2") {
		t.Error("expected Pattern 1/2 header")
	}
	if !strings.Contains(result, "Pattern 2/2") {
		t.Error("expected Pattern 2/2 header")
	}
	if !strings.Contains(result, "NewAgent") {
		t.Error("expected NewAgent in result")
	}
	if !strings.Contains(result, "Run") {
		t.Error("expected Run in result")
	}
}

func TestGoSymbolResolver_UsesBundleDiagnostics(t *testing.T) {
	setupSymbolTestDir(t, "example.go", `package example

func Run() {}
`)

	resolved := goSymbolResolver{}.Resolve("Run", SearchOptions{
		Path:            ".",
		LSPClient:       &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "example.go", Line: 3, Character: 1, EndLine: 3, EndChar: 5}}},
		LocatorRegistry: nil,
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}
	if resolved.Bundle == nil {
		t.Fatal("expected bundle in go symbol resolution")
	}
	if !strings.Contains(resolved.Output, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in go symbol output, got:\n%s", resolved.Output)
	}
}

func TestGoSymbolResolver_UsesProjectMapSnapshot(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"broken.go": "package example\n\nfunc (\n",
	})

	resolved := goSymbolResolver{}.Resolve("Run", SearchOptions{
		Path: dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "broken.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "go-snapshot-fast-path",
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected snapshot-backed single resolution, got %s", resolved.Status)
	}
	if resolved.Bundle == nil || resolved.Bundle.Definition.File != "broken.go" {
		t.Fatalf("expected snapshot bundle for broken.go, got %+v", resolved.Bundle)
	}
}

func TestSearchCode_MultiPatternGoSymbolPreservesDiagnostics(t *testing.T) {
	setupSymbolTestDir(t, "example.go", `package example

type Agent struct{}

func (a *Agent) Close() error { return nil }

func run(a *Agent) error {
	return a.Close()
}
`)

	result := ExecuteSearchCode(SearchOptions{
		Pattern: "Close,(*Agent).Close,\\.Close\\(\\)",
		Path:    ".",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{{File: "example.go", Line: 5, Character: 1, EndLine: 5, EndChar: 10}},
		},
	})

	if !strings.Contains(result, "Matched patterns:") {
		t.Fatalf("expected multi-pattern bundle output, got:\n%s", result)
	}
	if !strings.Contains(result, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in multi-pattern output, got:\n%s", result)
	}
}
