package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeAllowsHelperConstructedReceiver(t *testing.T) {
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

func makeRunner() *Agent {
	return &Agent{}
}

func TestRun(t *testing.T) {
	a := makeRunner()
	_ = a.Run()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected helper-constructed receiver test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeAllowsHelperOnlyCall(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}
type Job struct{}

func (a *Agent) Run() error { return nil }
func (j *Job) Run() error { return nil }

func UseAgent(a *Agent) error {
	return a.Run()
}
`,
		"agent_test.go": `package example

import "testing"

func assertRun(t *testing.T, a *Agent) {
	t.Helper()
	_ = a.Run()
}

func TestRun(t *testing.T) {
	a := &Agent{}
	assertRun(t, a)
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 9, Character: 11, EndLine: 9, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected helper-only call test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeFallsBackOnParseError(t *testing.T) {
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
	assertRun(t, &Agent{})
}

func assertRun(t *testing.T, a *Agent) {
	t.Helper()
	_ = a.Run()
}

func broken() {
	if true {
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected parse-failure fallback to retain name-probed TestRun, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeAllowsSplitSelector(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}
type Job struct{}

func (a *Agent) Run() error { return nil }
func (j *Job) Run() error { return nil }

func UseAgent(a *Agent) error {
	return a.Run()
}
`,
		"agent_test.go": `package example

import "testing"

func TestRun(t *testing.T) {
	a := &Agent{}
	_ = a.
		Run()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent.go", Line: 9, Character: 11, EndLine: 9, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected split-selector test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeNormalizesGenericReceiver(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"set.go": `package example

type Set[T comparable] struct{}
type Bag struct{}

func (s *Set[T]) Add(v T) {}
func (b *Bag) Add(v string) {}

func UseSet(s *Set[int]) {
	s.Add(1)
}
`,
		"set_test.go": `package example

import "testing"

func TestAdd(t *testing.T) {
	var s Set[int]
	s.Add(1)
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Set.Add",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "set.go", Line: 9, Character: 4, EndLine: 9, EndChar: 8}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestAdd") {
		t.Fatalf("expected generic receiver method test to be retained, got:\n%s", output)
	}
}
