package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsCrossPackageHelperOnlyCall(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func assertRun(t *testing.T, a *agentpkg.Agent) {
	t.Helper()
	_ = a.Run()
}

func TestRun(t *testing.T) {
	a := agentpkg.New()
	assertRun(t, a)
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
		t.Fatalf("expected helper-only cross-package method test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsCrossPackageHelperOnlyCallAcrossFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"integration/helpers_test.go": `package integration

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

func TestRun(t *testing.T) {
	a := agentpkg.New()
	assertRun(t, a)
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
		t.Fatalf("expected helper-only cross-package method test across files to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsDeepCrossPackageHelperChainThroughSupportFile(t *testing.T) {
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
	forwardRun(t, a)
}

func forwardRun(t *testing.T, a *agentpkg.Agent) {
	t.Helper()
	invokeRun(a)
}

func invokeRun(a *agentpkg.Agent) {
	_ = a.Run()
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	a := agentpkg.New()
	assertRun(t, a)
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
		t.Fatalf("expected deep helper-chain method test to remain discoverable, got:\n%s", output)
	}
}
