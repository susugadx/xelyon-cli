package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsReturnedCrossPackageHelperCall(t *testing.T) {
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

func runner() func(*testing.T, *agentpkg.Agent) {
	return assertRun
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	a := agentpkg.New()
	runner()(t, a)
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

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected returned helper call test to remain discoverable, got:\n%s", output)
	}
}
func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsTupleReturnHelperCall(t *testing.T) {
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

func loadAgent() (error, *agentpkg.Agent) {
	return nil, agentpkg.New()
}

func assertRun(t *testing.T) {
	t.Helper()
	_, a := loadAgent()
	_ = a.Run()
}

func assertRunVar(t *testing.T) {
	t.Helper()
	var err error
	var a *agentpkg.Agent
	err, a = loadAgent()
	_ = err
	_ = a.Run()
}
`,
		"integration/agent_test.go": `package integration

import "testing"

func TestRun(t *testing.T) {
	assertRun(t)
	assertRunVar(t)
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

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected tuple-return helper call test to remain discoverable, got:\n%s", output)
	}
}
