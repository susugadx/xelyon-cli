package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesReflectMethodByNameOnForeignReceiver(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"otherpkg/runner.go": `package otherpkg

type Runner struct{}

func New() *Runner { return &Runner{} }

func (r *Runner) Run() error { return nil }
`,
		"integration/agent_test.go": `package integration

import (
	"os"
	"reflect"
	"testing"

	otherpkg "example/otherpkg"
)

func TestRun(t *testing.T) {
	r := otherpkg.New()
	name := os.Getenv("XELYON_METHOD")
	reflect.ValueOf(r).MethodByName(name).Call(nil)
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

	if strings.Contains(output, "Related Tests") || strings.Contains(output, "integration/agent_test.go") {
		t.Fatalf("expected foreign reflect.MethodByName receiver to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesPluginLookupWithForeignReceiver(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }
`,
		"otherpkg/runner.go": `package otherpkg

type Runner struct{}

func New() *Runner { return &Runner{} }

func (r *Runner) Run() error { return nil }
`,
		"integration/agent_test.go": `package integration

import (
	"os"
	"plugin"
	"testing"

	otherpkg "example/otherpkg"
)

func TestRun(t *testing.T) {
	p, _ := plugin.Open(os.Getenv("XELYON_PLUGIN"))
	sym, _ := p.Lookup(os.Getenv("XELYON_SYMBOL"))
	run := sym.(func(*testing.T, *otherpkg.Runner))
	run(t, otherpkg.New())
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

	if strings.Contains(output, "Related Tests") || strings.Contains(output, "integration/agent_test.go") {
		t.Fatalf("expected plugin lookup using foreign receiver to be excluded, got:\n%s", output)
	}
}
