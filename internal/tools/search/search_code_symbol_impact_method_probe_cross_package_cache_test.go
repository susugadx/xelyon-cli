package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsMultipleCrossPackageTestsFromSamePackage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"integration/helpers.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func assertRun(t *testing.T, a *agentpkg.Agent) {
	t.Helper()
	_ = a.Run()
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRunPrimary(t *testing.T) {
	assertRun(t, agentpkg.New())
}

func TestRunSecondary(t *testing.T) {
	assertRun(t, agentpkg.New())
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 7, Character: 11, EndLine: 7, EndChar: 14}}},
	})

	for _, want := range []string{"Related Tests", "TestRunPrimary", "TestRunSecondary", "integration/agent_test.go"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected cross-package method probe output to contain %q, got:\n%s", want, output)
		}
	}
}
