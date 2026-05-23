package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeDoesNotCrossPackagesFromRootFilePath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Run() error { return nil }

func UseAgent(a *Agent) error {
	return a.Run()
}
`,
		"job/job_test.go": `package job

import "testing"

func TestRun(t *testing.T) {
	t.Fatal("unrelated")
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      filepath.Join(dir, "agent.go"),
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected root-package file-scoped method probe to avoid unrelated subpackage TestRun matches, got:\n%s", output)
	}
	if strings.Contains(output, "job/job_test.go") {
		t.Fatalf("expected unrelated subpackage test file to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeUsesPackageFromFileScopedPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Run() error { return nil }

func UseAgent(a *Agent) error {
	return a.Run()
}
`,
		"agent_test.go": `package example

import "testing"

func TestRun(t *testing.T) {
	var a Agent
	_ = a.Run()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      filepath.Join(dir, "agent.go"),
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "agent_test.go") {
		t.Fatalf("expected file-scoped definition to still include same-package impact tests, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoFunctionTestProbeStaysInPackageFromFileScopedPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"app/build.go": `package app

func Build() string { return "" }

func UseBuild() string {
	return Build()
}
`,
		"app/build_test.go": `package app

import "testing"

func TestBuild(t *testing.T) {
	_ = Build()
}
`,
		"other/build_test.go": `package other

import "testing"

func TestBuild(t *testing.T) {
	t.Fatal("unrelated same-name test")
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:       "Build",
		Intent:        "impact",
		Path:          filepath.Join(dir, "app", "build.go"),
		FileType:      "go",
		InvocationCWD: dir,
		LSPClient:     &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "app/build.go", Line: 6, Character: 9, EndLine: 6, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "app/build_test.go") {
		t.Fatalf("expected file-scoped function probe to include same-package TestBuild, got:\n%s", output)
	}
	if strings.Contains(output, "other/build_test.go") {
		t.Fatalf("expected name-only function probe to exclude sibling same-name TestBuild, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeSkipsHiddenAndIgnoredFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"visible_test.go": `package example

import "testing"

func TestBuild(t *testing.T) {
	var c Config
	_ = c.Build()
}
`,
		".hidden_test.go": `package example

import "testing"

func TestBuild(t *testing.T) {
	var c Config
	_ = c.Build()
}
`,
		"generated/ignored_test.go": `package example

import "testing"

func TestBuild(t *testing.T) {
	var c Config
	_ = c.Build()
}
`,
	})

	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	output := ExecuteSearchCodeWithConfig(config.DefaultConfig(), nil, SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 7, Character: 11, EndLine: 7, EndChar: 16}}},
	})

	if !strings.Contains(output, "visible_test.go") {
		t.Fatalf("expected visible method test to remain, got:\n%s", output)
	}
	if strings.Contains(output, ".hidden_test.go") {
		t.Fatalf("expected hidden test file to be excluded, got:\n%s", output)
	}
	if strings.Contains(output, "generated/ignored_test.go") {
		t.Fatalf("expected ignored test file to be excluded, got:\n%s", output)
	}
}
