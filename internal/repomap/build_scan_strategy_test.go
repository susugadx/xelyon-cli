package repomap

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type stubPatternScanner struct {
	calls   []patternScanCall
	results []map[string][]Symbol
	err     error
}

type stubASTScanner struct {
	supportedPaths map[string]bool
	results        map[string][]Symbol
	errByPath      map[string]error
	calls          []string
}

type patternScanCall struct {
	def     languagePattern
	targets []string
}

func (s *stubASTScanner) supports(path string) bool {
	if s.supportedPaths == nil {
		return false
	}
	return s.supportedPaths[path]
}

func (s *stubASTScanner) scan(absPath string) ([]Symbol, error) {
	s.calls = append(s.calls, absPath)
	if err := s.errByPath[absPath]; err != nil {
		return nil, err
	}
	return append([]Symbol(nil), s.results[absPath]...), nil
}

func (s *stubPatternScanner) scan(def languagePattern, targets []string, _ map[string]map[int]struct{}) (map[string][]Symbol, error) {
	s.calls = append(s.calls, patternScanCall{
		def:     def,
		targets: append([]string(nil), targets...),
	})
	if s.err != nil {
		return nil, s.err
	}
	if len(s.results) == 0 {
		return map[string][]Symbol{}, nil
	}
	next := s.results[0]
	s.results = s.results[1:]
	return next, nil
}

func TestSymbolScanStrategy_CollectPatternFallbackTargets(t *testing.T) {
	strategy := newSymbolScanStrategy(&ProjectMap{})
	states := []fileState{
		{path: "pkg/build.go", supportsSym: true},
		{path: "pkg/build_test.go", supportsSym: true},
		{path: "pkg/task.py", supportsSym: true},
		{path: "pkg/cached.py", supportsSym: true, cached: &CacheFile{}},
		{path: "README.md", supportsSym: false},
		{path: "scripts/run", supportsSym: true},
	}

	resolvedByAST := map[string][]Symbol{
		"pkg/build.go": {{Name: "Build"}},
	}
	got := strategy.collectPatternFallbackTargets(states, resolvedByAST)

	goTargets := got[".go"]
	if len(goTargets) != 1 || goTargets[0] != "pkg/build_test.go" {
		t.Fatalf("go targets = %#v, want [pkg/build_test.go]", goTargets)
	}
	pyTargets := got[".py"]
	if len(pyTargets) != 1 || pyTargets[0] != "pkg/task.py" {
		t.Fatalf("py targets = %#v, want [pkg/task.py]", pyTargets)
	}
	if _, ok := got[".md"]; ok {
		t.Fatalf("markdown target should not be collected: %#v", got[".md"])
	}
}

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

func TestSymbolScanStrategy_ScanWithPatternFallback_UsesPatternScanner(t *testing.T) {
	scanner := &stubPatternScanner{
		results: []map[string][]Symbol{
			{"pkg/fallback.go": {{Name: "Build", Line: 3}}},
			{"pkg/task.py": {{Name: "run_task", Line: 5}}},
		},
	}
	strategy := newSymbolScanStrategyWithPatternScanner([]languagePattern{
		{
			Extensions: []string{".go"},
			Patterns:   []string{`^func `},
		},
		{
			Extensions: []string{".py"},
			Patterns:   []string{`^def `},
		},
	}, scanner)

	results := map[string][]Symbol{}
	targetsByExt := map[string][]string{
		".go": {"pkg/fallback.go"},
		".py": {"pkg/task.py"},
	}
	if err := strategy.scanWithPatternFallback(results, targetsByExt); err != nil {
		t.Fatalf("scanWithPatternFallback() error = %v", err)
	}

	if len(scanner.calls) != 2 {
		t.Fatalf("pattern scanner calls = %d, want 2", len(scanner.calls))
	}
	if len(scanner.calls[0].targets) != 1 || scanner.calls[0].targets[0] != "pkg/fallback.go" {
		t.Fatalf("first call targets = %#v, want [pkg/fallback.go]", scanner.calls[0].targets)
	}
	if len(scanner.calls[1].targets) != 1 || scanner.calls[1].targets[0] != "pkg/task.py" {
		t.Fatalf("second call targets = %#v, want [pkg/task.py]", scanner.calls[1].targets)
	}
	if len(results["pkg/fallback.go"]) != 1 || results["pkg/fallback.go"][0].Name != "Build" {
		t.Fatalf("go fallback symbols = %#v, want Build", results["pkg/fallback.go"])
	}
	if len(results["pkg/task.py"]) != 1 || results["pkg/task.py"][0].Name != "run_task" {
		t.Fatalf("python fallback symbols = %#v, want run_task", results["pkg/task.py"])
	}
}

func TestSymbolScanStrategy_ScanWithPatternFallback_PropagatesScannerError(t *testing.T) {
	wantErr := errors.New("scan failed")
	strategy := newSymbolScanStrategyWithPatternScanner([]languagePattern{
		{
			Extensions: []string{".go"},
			Patterns:   []string{`^func `},
		},
	}, &stubPatternScanner{err: wantErr})

	err := strategy.scanWithPatternFallback(map[string][]Symbol{}, map[string][]string{
		".go": {"pkg/fallback.go"},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("scanWithPatternFallback() error = %v, want %v", err, wantErr)
	}
}

func TestProjectMapPatternScanner_Scan_NilProjectMap(t *testing.T) {
	scanner := newProjectMapPatternScanner(nil)
	_, err := scanner.scan(languagePattern{}, nil, map[string]map[int]struct{}{})
	if err == nil {
		t.Fatal("scanner.scan() error = nil, want non-nil")
	}
}
