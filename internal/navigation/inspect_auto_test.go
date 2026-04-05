package navigation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestInspectSymbolAuto_Empty(t *testing.T) {
	output, status := InspectSymbolAuto("", "", nil, nil)
	if status != SymbolAutoNone {
		t.Fatalf("expected SymbolAutoNone, got %s", status)
	}
	if output != "" {
		t.Errorf("expected empty output, got: %s", output)
	}
}

func TestInspectSymbolAuto_NotFound(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	setupTestGoFile(t, "example.go", testGoSource)

	output, status := InspectSymbolAuto("NonExistentXYZ12345", "", nil, nil)
	if status != SymbolAutoNone {
		t.Fatalf("expected SymbolAutoNone, got %s", status)
	}
	if output != "" {
		t.Errorf("expected empty output, got: %s", output)
	}
}

func TestInspectSymbolAuto_SingleCandidate(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	setupTestGoFile(t, "example.go", testGoSource)

	output, status := InspectSymbolAuto("Run", "", nil, nil)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if !strings.Contains(output, "Run") {
		t.Error("expected output to contain symbol name")
	}
	if !strings.Contains(output, "func") {
		t.Error("expected output to contain definition kind")
	}
}

func TestInspectSymbolAuto_MultipleCandidates(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	setupTestGoFile(t, "example.go", testGoSource)

	output, status := InspectSymbolAuto("Build", "", nil, nil)
	if status != SymbolAutoMultiple {
		t.Fatalf("expected SymbolAutoMultiple, got %s", status)
	}
	if !strings.Contains(output, "Multiple symbols matched") {
		t.Error("expected 'Multiple symbols matched' in output")
	}
}

func TestResolveInspectSymbolAuto_UsesSnapshotCacheWhenProjectMapBecomesNil(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"cached.go": "package example\n\nfunc Run() {}\n",
	})

	opts := InspectSymbolAutoOptions{
		Budget: SummaryBudget,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "cached.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run() {}", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "snapshot-cache-state",
	}

	result, _, status := ResolveInspectSymbolAuto("Run", filepath.Join(dir, "cached.go"), opts)
	if status != SymbolAutoSingle || result.Symbol == nil {
		t.Fatalf("expected initial snapshot-backed resolution, got status=%s result=%+v", status, result)
	}

	if err := os.WriteFile(filepath.Join(dir, "cached.go"), []byte("package example\n\nfunc (\n"), 0o644); err != nil {
		t.Fatalf("failed to break source file: %v", err)
	}

	opts.ProjectMap = nil
	result, _, status = ResolveInspectSymbolAuto("Run", filepath.Join(dir, "cached.go"), opts)
	if status != SymbolAutoSingle || result.Symbol == nil {
		t.Fatalf("expected cached snapshot-backed resolution after ProjectMap removal, got status=%s result=%+v", status, result)
	}
	if result.Symbol.File != "cached.go" {
		t.Fatalf("unexpected cached snapshot symbol: %+v", result.Symbol)
	}
}

func TestLoadGoSymbolSnapshot_ReusesCachedEntryBeforeRebuild(t *testing.T) {
	dir := t.TempDir()
	cacheKey := goSymbolSnapshotCacheKey(dir, "reuse-state")
	want := &goSymbolSnapshot{
		RootPath: dir,
		StateKey: "reuse-state",
		ByName: map[string][]goSymbolSnapshotEntry{
			"Run": {{Name: "Run", File: "cached.go", Line: 3}},
		},
	}
	storeGoSymbolSnapshot(cacheKey, want)
	t.Cleanup(clearGoSymbolSnapshotCache)

	got := loadGoSymbolSnapshot(GoSymbolRuntime{
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "other.go",
					Symbols: []repomap.Symbol{
						{Name: "Other", Kind: "function", Line: 5, Signature: "func Other()"},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "reuse-state",
	})
	if got != want {
		t.Fatalf("expected cached snapshot pointer reuse, got %+v want %+v", got, want)
	}
}

func TestGoSymbolSnapshotCacheClearedBySearchCacheHook(t *testing.T) {
	dir := t.TempDir()
	storeGoSymbolSnapshot(goSymbolSnapshotCacheKey(dir, "clear-state"), &goSymbolSnapshot{
		RootPath: dir,
		StateKey: "clear-state",
		ByName:   map[string][]goSymbolSnapshotEntry{"Run": {{Name: "Run"}}},
	})

	if got := lookupGoSymbolSnapshot(goSymbolSnapshotCacheKey(dir, "clear-state")); got == nil {
		t.Fatal("expected snapshot cache entry before clear")
	}

	tools.NotifySearchCacheCleared()

	if got := lookupGoSymbolSnapshot(goSymbolSnapshotCacheKey(dir, "clear-state")); got != nil {
		t.Fatalf("expected snapshot cache entry to be cleared, got %+v", got)
	}
}

func TestNormalizeInspectResultPaths_PrefersProjectMapRootPath(t *testing.T) {
	parentDir := t.TempDir()
	projectRoot := filepath.Join(parentDir, "repo")
	if err := os.MkdirAll(filepath.Join(projectRoot, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "pkg", "run.go"), []byte("package pkg\n\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shared", "run_test.go"), []byte("package shared\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := InspectResult{
		Symbol: &SymbolCandidate{
			File:     "pkg/run.go",
			Line:     3,
			RootPath: parentDir,
		},
		Refs: []Reference{
			{File: filepath.Join(projectRoot, "pkg", "run.go"), Line: 3},
		},
		Tests: []TestRef{
			{File: filepath.Join(projectRoot, "shared", "run_test.go"), Line: 1},
		},
	}

	normalizeInspectResultPaths(&result, GoSymbolRuntime{
		ProjectMapRootPath: projectRoot,
		InvocationCWD:      filepath.Join(projectRoot, "pkg"),
	})

	if result.Symbol.RootPath != projectRoot {
		t.Fatalf("expected symbol root path %s, got %s", projectRoot, result.Symbol.RootPath)
	}
	if got := result.Refs[0].File; got != "pkg/run.go" {
		t.Fatalf("expected ref path pkg/run.go, got %s", got)
	}
	if got := result.Tests[0].File; got != "shared/run_test.go" {
		t.Fatalf("expected test path shared/run_test.go, got %s", got)
	}
}

func TestNormalizeInspectResultPaths_RecoversProcessCWDRelativeTestPath(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "pkg", "run.go"), []byte("package pkg\n\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(projectRoot, "shared", "run_test.go")
	if err := os.WriteFile(testFile, []byte("package shared\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	processRelativePath := toRelativePath(testFile)
	if filepath.IsAbs(processRelativePath) {
		t.Fatalf("expected process-relative test path, got absolute %s", processRelativePath)
	}

	result := InspectResult{
		Symbol: &SymbolCandidate{
			File:     "pkg/run.go",
			Line:     3,
			RootPath: projectRoot,
		},
		Tests: []TestRef{
			{File: processRelativePath, Line: 1},
		},
	}

	normalizeInspectResultPaths(&result, GoSymbolRuntime{
		ProjectMapRootPath: projectRoot,
		InvocationCWD:      filepath.Join(projectRoot, "pkg"),
	})

	if got := result.Tests[0].File; got != "shared/run_test.go" {
		t.Fatalf("expected recovered test path shared/run_test.go, got %s", got)
	}
}

type mockNavigationLSPClient struct {
	refs     []LSPLocation
	refsErr  error
	impls    []LSPLocation
	implsErr error
}

func (m *mockNavigationLSPClient) FindReferences(context.Context, string, int, int, bool) ([]LSPLocation, error) {
	return m.refs, m.refsErr
}

func (m *mockNavigationLSPClient) GotoDefinition(context.Context, string, int, int) ([]LSPLocation, error) {
	return nil, nil
}

func (m *mockNavigationLSPClient) GotoImplementation(context.Context, string, int, int) ([]LSPLocation, error) {
	return m.impls, m.implsErr
}

func TestInspectSymbolAuto_UsesLSPReferences(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"run.go": `package example

func Run() {
}
`,
		"caller.go": `package example

func main() {
	Run()
}
`,
	})

	client := &mockNavigationLSPClient{
		refs: []LSPLocation{
			{File: "caller.go", Line: 4, Character: 1, EndLine: 4, EndChar: 5},
		},
	}

	output, status := InspectSymbolAuto("Run", "", nil, client)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if !strings.Contains(output, "Callers (1)") {
		t.Fatalf("expected callers section from LSP result, got: %s", output)
	}
	if !strings.Contains(output, "resolved via gopls") {
		t.Fatalf("expected gopls suffix, got: %s", output)
	}
}

func TestInspectSymbolAuto_LSPFallbackOnError(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	client := &mockNavigationLSPClient{refsErr: errors.New("boom")}

	output, status := InspectSymbolAuto("Run", "", nil, client)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if strings.Contains(output, "resolved via gopls") {
		t.Fatalf("expected fallback output without gopls suffix, got: %s", output)
	}
	if !strings.Contains(output, "func Run") {
		t.Fatalf("expected fallback output to still inspect the symbol, got: %s", output)
	}
}

func TestInspectSymbolAuto_UsesLSPImplementations(t *testing.T) {
	setupTestGoFile(t, "example.go", `package example

type Builder interface {
	Build() string
}

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`)

	client := &mockNavigationLSPClient{
		refs: []LSPLocation{
			{File: "example.go", Line: 4, Character: 1, EndLine: 4, EndChar: 6},
		},
		impls: []LSPLocation{
			{File: "example.go", Line: 7, Character: 1, EndLine: 7, EndChar: 11},
		},
	}

	output, status := InspectSymbolAuto("Builder", "", nil, client)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if !strings.Contains(output, "Implementations (1)") {
		t.Fatalf("expected implementations section, got: %s", output)
	}
	if !strings.Contains(output, "resolved via gopls") {
		t.Fatalf("expected gopls suffix, got: %s", output)
	}
}
