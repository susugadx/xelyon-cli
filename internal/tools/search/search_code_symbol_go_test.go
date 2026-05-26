package search

import (
	"path/filepath"
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

func TestSearchCode_GoSymbolFallbackReferencesStayScopedToSearchPath(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"app/agent.go": `package app

type Agent struct{}

func (a *Agent) Close() error { return nil }

func run(a *Agent) error {
	return a.Close()
}
`,
		"other/agent.go": `package other

type Agent struct{}

func (a *Agent) Close() error { return nil }

func unrelated(a *Agent) error {
	return a.Close()
}
`,
	})
	withWorkingDirForSearchTest(t, root)

	result := ExecuteSearchCode(SearchOptions{Pattern: "Close", Path: "app", FileType: "go"})
	if !strings.Contains(result, "app/agent.go") {
		t.Fatalf("expected scoped app symbol output, got:\n%s", result)
	}
	if strings.Contains(result, "other/agent.go") {
		t.Fatalf("expected fallback references to stay scoped to app, got:\n%s", result)
	}
}

func TestSearchCode_GoSymbolDirectoryScopeFiltersLSPReferences(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"app/target.go": `package app

func Target() string { return "target" }
`,
		"app/use.go": `package app

func UseTarget() string {
	return Target()
}
`,
		"other/use.go": `package other

func UseOtherTarget() string {
	return Target()
}
`,
	})
	withWorkingDirForSearchTest(t, root)

	result := ExecuteSearchCode(SearchOptions{
		Pattern:       "Target",
		Path:          "app",
		FileType:      "go",
		InvocationCWD: root,
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "app/use.go", Line: 4, Character: 9, EndLine: 4, EndChar: 15},
				{File: "other/use.go", Line: 4, Character: 9, EndLine: 4, EndChar: 15},
			},
		},
	})
	if !strings.Contains(result, "Note: resolved via gopls.") {
		t.Fatalf("expected scoped LSP references to be used, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers (1)") || !strings.Contains(result, "app/use.go:4") {
		t.Fatalf("expected directory-scoped LSP caller, got:\n%s", result)
	}
	if strings.Contains(result, "other/use.go") {
		t.Fatalf("expected out-of-scope LSP caller to be filtered, got:\n%s", result)
	}
}

func TestSearchCode_GoSymbolDirectFileScopeFiltersLSPReferencesToPackageDir(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"pkg/target.go": `package pkg

func Target() string { return "target" }
`,
		"pkg/use.go": `package pkg

func UseTarget() string {
	return Target()
}
`,
		"pkg/sub/use.go": `package sub

func UseSubTarget() string {
	return Target()
}
`,
		"other/use.go": `package other

func UseOtherTarget() string {
	return Target()
}
`,
	})
	withWorkingDirForSearchTest(t, root)

	result := ExecuteSearchCode(SearchOptions{
		Pattern:       "Target",
		Path:          filepath.Join(root, "pkg", "target.go"),
		FileType:      "go",
		InvocationCWD: root,
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "pkg/use.go", Line: 4, Character: 9, EndLine: 4, EndChar: 15},
				{File: "pkg/sub/use.go", Line: 4, Character: 9, EndLine: 4, EndChar: 15},
				{File: "other/use.go", Line: 4, Character: 9, EndLine: 4, EndChar: 15},
			},
		},
	})
	if !strings.Contains(result, "Note: resolved via gopls.") {
		t.Fatalf("expected scoped LSP references to be used, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers (1)") || !strings.Contains(result, "pkg/use.go:4") {
		t.Fatalf("expected direct-file LSP caller from same package dir, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/sub/use.go") || strings.Contains(result, "other/use.go") {
		t.Fatalf("expected out-of-package LSP callers to be filtered, got:\n%s", result)
	}
}

func TestSearchCode_GoSymbolFallbackReferencesResolveRelativePathFromInvocationCWD(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"workspace/app/target.go": `package app

func Target() string { return "target" }

func UseTarget() string {
	return Target()
}
`,
	})
	withWorkingDirForSearchTest(t, root)

	result := ExecuteSearchCode(SearchOptions{
		Pattern:  "Target",
		Path:     "app",
		FileType: "go",
		ProjectMap: &repomap.ProjectMap{
			RootPath: root,
			Files: []*repomap.FileEntry{
				{
					Path: "workspace/app/target.go",
					Symbols: []repomap.Symbol{
						{Name: "Target", Kind: "function", Line: 3, EndLine: 3, Signature: "func Target() string", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: root,
		ProjectMapStateKey: "go-symbol-fallback-relative-invocation-cwd",
		InvocationCWD:      filepath.Join(root, "workspace"),
	})
	if !strings.Contains(result, "workspace/app/target.go") {
		t.Fatalf("expected snapshot-backed symbol under invocation cwd, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers (1)") || !strings.Contains(result, "workspace/app/target.go:6") {
		t.Fatalf("expected fallback references to resolve relative path from invocation cwd, got:\n%s", result)
	}
}

func TestSearchCode_GoSymbolASTFallbackRelativePathFromInvocationCWDIncludesDefinitionBody(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"workspace/app/target.go": `package app

func Target() string {
	return "target"
}

func UseTarget() string {
	return Target()
}
`,
	})
	withWorkingDirForSearchTest(t, root)

	result := ExecuteSearchCode(SearchOptions{
		Pattern:       "Target",
		Path:          "app",
		FileType:      "go",
		InvocationCWD: filepath.Join(root, "workspace"),
	})
	if !strings.Contains(result, "in app/target.go") || strings.Contains(result, "workspace/app/target.go") {
		t.Fatalf("expected AST fallback output path to stay invocation-cwd relative, got:\n%s", result)
	}
	if !strings.Contains(result, `return "target"`) {
		t.Fatalf("expected AST fallback definition body, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers (1)") || !strings.Contains(result, "app/target.go:8") {
		t.Fatalf("expected AST fallback caller path to stay invocation-cwd relative, got:\n%s", result)
	}
}

func TestSearchCode_GoSymbolDirectFileFallbackReferencesIncludeSiblingCallers(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"pkg/target.go": `package pkg

func Target() string { return "target" }
`,
		"pkg/use.go": `package pkg

func UseTarget() string {
	return Target()
}
`,
		"pkg/sub/use.go": `package sub

func UseTargetFromSubpackage() string {
	return Target()
}
`,
		"other/target.go": `package other

func Target() string { return "other" }
`,
		"other/use.go": `package other

func UseOtherTarget() string {
	return Target()
}
`,
	})
	withWorkingDirForSearchTest(t, root)

	result := ExecuteSearchCode(SearchOptions{
		Pattern:  "Target",
		Path:     filepath.Join(root, "pkg", "target.go"),
		FileType: "go",
	})
	if !strings.Contains(result, "pkg/target.go") {
		t.Fatalf("expected direct-file definition output, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers (1)") || !strings.Contains(result, "pkg/use.go:4") {
		t.Fatalf("expected direct-file fallback references to include sibling caller, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/sub/use.go") {
		t.Fatalf("expected direct-file fallback references to exclude subpackage caller, got:\n%s", result)
	}
	if strings.Contains(result, "other/use.go") {
		t.Fatalf("expected direct-file fallback references to exclude other package caller, got:\n%s", result)
	}
}
