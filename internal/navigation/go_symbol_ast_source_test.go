package navigation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInspectSymbolAuto_ASTFallbackAbsoluteHintUsesInvocationCWDRoot(t *testing.T) {
	repoRoot, workspace := setupASTFallbackInvocationWorkspace(t)
	withNavigationWorkingDir(t, repoRoot)

	result, output, status := ResolveInspectSymbolAuto("Target", filepath.Join(workspace, "app"), InspectSymbolAutoOptions{
		Budget:        SummaryBudget,
		InvocationCWD: workspace,
	})
	if status != SymbolAutoSingle {
		t.Fatalf("status = %s, want single; output:\n%s", status, output)
	}
	if result.Symbol == nil {
		t.Fatalf("expected symbol result")
	}
	if result.Symbol.RootPath != workspace {
		t.Fatalf("RootPath = %q, want %q", result.Symbol.RootPath, workspace)
	}
	if result.Symbol.File != "app/target.go" {
		t.Fatalf("File = %q, want app/target.go", result.Symbol.File)
	}
	if !strings.Contains(output, "in app/target.go") || strings.Contains(output, "workspace/app/target.go") {
		t.Fatalf("expected invocation-cwd relative output path, got:\n%s", output)
	}
	if !strings.Contains(output, `return "target"`) {
		t.Fatalf("expected definition body from invocation-cwd file, got:\n%s", output)
	}
	if len(result.Callers) != 1 {
		t.Fatalf("callers = %+v, want one invocation-cwd relative caller", result.Callers)
	}
	if result.Callers[0].File != "app/target.go" {
		t.Fatalf("caller file = %q, want app/target.go", result.Callers[0].File)
	}
}

func TestResolveInspectSymbolAuto_ASTFallbackRelativeHintUsesInvocationCWD(t *testing.T) {
	repoRoot, workspace := setupASTFallbackInvocationWorkspace(t)
	withNavigationWorkingDir(t, repoRoot)

	result, output, status := ResolveInspectSymbolAuto("Target", "app", InspectSymbolAutoOptions{
		Budget:        SummaryBudget,
		InvocationCWD: workspace,
	})
	if status != SymbolAutoSingle {
		t.Fatalf("status = %s, want single; output:\n%s", status, output)
	}
	if result.Symbol == nil || result.Symbol.File != "app/target.go" {
		t.Fatalf("expected app/target.go candidate, got %+v", result.Symbol)
	}
	if !strings.Contains(output, `return "target"`) {
		t.Fatalf("expected definition body from relative invocation-cwd path, got:\n%s", output)
	}
}

func setupASTFallbackInvocationWorkspace(t *testing.T) (string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	workspace := filepath.Join(repoRoot, "workspace")
	appDir := filepath.Join(workspace, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := `package app

func Target() string {
	return "target"
}

func UseTarget() string {
	return Target()
}
`
	if err := os.WriteFile(filepath.Join(appDir, "target.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return repoRoot, workspace
}

func withNavigationWorkingDir(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
}
