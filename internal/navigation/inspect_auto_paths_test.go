package navigation

import (
	"os"
	"path/filepath"
	"testing"
)

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
