package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeSkipsIgnoredCrossPackageHelpers(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Run() error { return nil }
`,
		"integration/agent_test.go": `package integration

import "testing"

func TestRun(t *testing.T) {
	assertRun(t)
}
`,
		"integration/helpers_test.go": `package integration

import (
	"testing"

	examplepkg "example"
)

func assertRun(t *testing.T) {
	t.Helper()
	var a examplepkg.Agent
	_ = a.Run()
}
`,
	})

	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - integration/helpers_test.go\n"), 0o644); err != nil {
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
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 4, Character: 17, EndLine: 4, EndChar: 20}}},
	})

	if strings.Contains(output, "Related Tests") || strings.Contains(output, "integration/agent_test.go") {
		t.Fatalf("expected ignored cross-package helper to suppress related test discovery, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeDoesNotTreatMethodCallAsPackageHelper(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Run() error { return nil }
`,
		"integration/agent_test.go": `package integration

import "testing"

type service struct{}

func (service) assertRun(t *testing.T) {}

func TestRun(t *testing.T) {
	var svc service
	svc.assertRun(t)
}
`,
		"integration/helpers_test.go": `package integration

import (
	"testing"

	examplepkg "example"
)

func assertRun(t *testing.T) {
	t.Helper()
	var a examplepkg.Agent
	_ = a.Run()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 4, Character: 17, EndLine: 4, EndChar: 20}}},
	})

	if strings.Contains(output, "Related Tests") || strings.Contains(output, "integration/agent_test.go") {
		t.Fatalf("expected method call svc.assertRun() to not trigger package helper fallback, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeIgnoresImportedTestOnlyHelperFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"testsupport/helpers.go": `package testsupport

import "testing"

func Setup(t *testing.T) {
	t.Helper()
}
`,
		"testsupport/setup_test.go": `package testsupport_test

import (
	"testing"

	agentpkg "example/agent"
)

func Setup(t *testing.T) {
	t.Helper()
	a := agentpkg.New()
	_ = a.Run()
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	"example/testsupport"
)

func TestRun(t *testing.T) {
	testsupport.Setup(t)
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 5, Character: 17, EndLine: 5, EndChar: 20}}},
	})

	if strings.Contains(output, "Related Tests") || strings.Contains(output, "integration/agent_test.go") {
		t.Fatalf("expected imported helper graph to ignore test-only helper files, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesBrokenCrossPackageTestCandidate(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func (a *Agent) Run() error { return nil }
`,
		"integration/agent_test.go": `package integration

import "testing"

func TestRun(t *testing.T) {
	if true {
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 4, Character: 17, EndLine: 4, EndChar: 20}}},
	})

	if strings.Contains(output, "Related Tests") || strings.Contains(output, "integration/agent_test.go") {
		t.Fatalf("expected broken cross-package TestRun candidate to be excluded, got:\n%s", output)
	}
}
