package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsReflectHelperCall(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"integration/helpers.go": `package integration

import (
	"reflect"
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
	"reflect"
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	run := reflect.ValueOf(assertRun)
	run.Call([]reflect.Value{reflect.ValueOf(t), reflect.ValueOf(agentpkg.New())})
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

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected reflect helper call test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsMapIndexedHelperCall(t *testing.T) {
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
	dispatch := map[string]func(*testing.T, *agentpkg.Agent){
		"run": assertRun,
	}
	dispatch["run"](t, agentpkg.New())
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

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected map-indexed helper call test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsTypeAssertedHelperCall(t *testing.T) {
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
	var run any = assertRun
	run.(func(*testing.T, *agentpkg.Agent))(t, agentpkg.New())
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

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected type-asserted helper call test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsConditionalHelperDispatch(t *testing.T) {
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

func noop(*testing.T, *agentpkg.Agent) {}
`,
		"integration/agent_test.go": `package integration

import (
	"os"
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	run := noop
	if os.Getenv("XELYON_DYNAMIC") == "1" {
		run = noop
	} else {
		run = assertRun
	}
	run(t, agentpkg.New())
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

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected conditional helper dispatch test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsNonConstantReflectHelperCall(t *testing.T) {
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
	"reflect"
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	dispatch := map[string]func(*testing.T, *agentpkg.Agent){
		"run": assertRun,
	}
	run := dispatch["run"]
	value := reflect.ValueOf(run)
	value.Call([]reflect.Value{reflect.ValueOf(t), reflect.ValueOf(agentpkg.New())})
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

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected non-constant reflect helper call test to remain discoverable, got:\n%s", output)
	}
}
