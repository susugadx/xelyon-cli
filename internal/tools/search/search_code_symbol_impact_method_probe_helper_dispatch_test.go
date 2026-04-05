package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsMethodizedCrossPackageHelperCall(t *testing.T) {
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

type helperSuite struct{}

func (helperSuite) assertRun(t *testing.T, a *agentpkg.Agent) {
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
	helperSuite{}.assertRun(t, a)
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
		t.Fatalf("expected methodized helper call test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsFunctionValueAliasCrossPackageHelperCall(t *testing.T) {
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

func TestRun(t *testing.T) {
	run := assertRun
	a := agentpkg.New()
	run(t, a)
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
		t.Fatalf("expected function-value alias helper call test to remain discoverable, got:\n%s", output)
	}
}

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

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsFunctionAdapterCrossPackageCall(t *testing.T) {
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

func invoke(run func(*testing.T, *agentpkg.Agent), t *testing.T, a *agentpkg.Agent) {
	run(t, a)
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	a := agentpkg.New()
	invoke(assertRun, t, a)
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
		t.Fatalf("expected function adapter cross-package helper call to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsInterfaceAdapterCrossPackageCall(t *testing.T) {
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

type runner interface {
	Run(*testing.T, *agentpkg.Agent)
}

type helperSuite struct{}

func (helperSuite) Run(t *testing.T, a *agentpkg.Agent) {
	t.Helper()
	_ = a.Run()
}

func invoke(r runner, t *testing.T, a *agentpkg.Agent) {
	r.Run(t, a)
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	a := agentpkg.New()
	invoke(helperSuite{}, t, a)
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
		t.Fatalf("expected interface adapter test to remain discoverable, got:\n%s", output)
	}
}
