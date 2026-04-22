package repomap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSymbolScanStrategy_ScanWithAST_SkipsCachedAndUnsupported(t *testing.T) {
	strategy := newSymbolScanStrategy(&ProjectMap{})
	root := t.TempDir()

	goPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(goPath, []byte("package main\n\nfunc Build() error {\n\treturn nil\n}\n"), 0644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	states := []fileState{
		{
			path:        "main.go",
			absPath:     goPath,
			supportsSym: true,
		},
		{
			path:        "cached.go",
			absPath:     goPath,
			supportsSym: true,
			cached:      &CacheFile{},
		},
		{
			path:        "pkg/tasks.py",
			absPath:     goPath,
			supportsSym: true,
		},
	}

	got := strategy.scanWithAST(states)
	if _, ok := got["main.go"]; !ok {
		t.Fatal("main.go should be scanned by AST")
	}
	if _, ok := got["cached.go"]; ok {
		t.Fatal("cached.go should be skipped when cache is reusable")
	}
	if _, ok := got["pkg/tasks.py"]; ok {
		t.Fatal("non-Go file should be skipped by AST scanner")
	}
}

func TestSymbolScanStrategy_ScanWithAST_UsesInjectedASTScanner(t *testing.T) {
	astScanner := &stubASTScanner{
		supportedPaths: map[string]bool{
			"pkg/custom.foo": true,
		},
		results: map[string][]Symbol{
			"/tmp/custom.foo": {
				{Name: "Custom", Kind: "function", Line: 7},
			},
		},
	}
	strategy := newSymbolScanStrategyWithScanners(defaultPatterns, astScanner, &stubPatternScanner{})

	got := strategy.scanWithAST([]fileState{
		{
			path:        "pkg/custom.foo",
			absPath:     "/tmp/custom.foo",
			supportsSym: true,
		},
		{
			path:        "pkg/skip.foo",
			absPath:     "/tmp/skip.foo",
			supportsSym: true,
		},
	})

	if len(got["pkg/custom.foo"]) != 1 || got["pkg/custom.foo"][0].Name != "Custom" {
		t.Fatalf("custom symbols = %#v, want one Custom symbol", got["pkg/custom.foo"])
	}
	if _, ok := got["pkg/skip.foo"]; ok {
		t.Fatalf("unsupported path should be skipped: %#v", got["pkg/skip.foo"])
	}
	if len(astScanner.calls) != 1 || astScanner.calls[0] != "/tmp/custom.foo" {
		t.Fatalf("ast scanner calls = %#v, want [/tmp/custom.foo]", astScanner.calls)
	}
}
