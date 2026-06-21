package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsImportedHelperPackageChain(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"testsupport/helpers.go": `package testsupport

import (
	"testing"

	agentpkg "example/agent"
)

func AssertRun(t *testing.T, a *agentpkg.Agent) {
	t.Helper()
	forwardRun(a)
}

func forwardRun(a *agentpkg.Agent) {
	_ = a.Run()
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
	"example/testsupport"
)

func TestRun(t *testing.T) {
	a := agentpkg.New()
	testsupport.AssertRun(t, a)
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
		t.Fatalf("expected imported helper package chain test to remain discoverable, got:\n%s", output)
	}
}
func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsImportedHelperPackageChainOnParseError(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"testsupport/helpers.go": `package testsupport

import (
	"testing"

	agentpkg "example/agent"
)

func AssertRun(t *testing.T, a *agentpkg.Agent) {
	t.Helper()
	forwardRun(a)
}

func forwardRun(a *agentpkg.Agent) {
	_ = a.Run()
}

func broken() {
	if true {
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
	"example/testsupport"
)

func TestRun(t *testing.T) {
	a := agentpkg.New()
	testsupport.AssertRun(t, a)
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
		t.Fatalf("expected imported helper package parse-failure fallback to retain integration test, got:\n%s", output)
	}
}
