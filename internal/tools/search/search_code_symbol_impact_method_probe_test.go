package search

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoUsesBaseNameForMethodTestProbe(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		files     map[string]string
		refs      []navigation.LSPLocation
		wantTest  string
		notWanted string
	}{
		{
			name:    "value receiver",
			pattern: "Config.Build",
			files: map[string]string{
				"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
				"config_test.go": `package example

import "testing"

func TestBuild(t *testing.T) {
	var c Config
	_ = c.Build()
}
`,
			},
			refs: []navigation.LSPLocation{
				{File: "config.go", Line: 8, Character: 11, EndLine: 8, EndChar: 16},
			},
			wantTest:  "TestBuild",
			notWanted: "TestConfig.Build",
		},
		{
			name:    "pointer receiver",
			pattern: "(*Agent).Run",
			files: map[string]string{
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
			},
			refs: []navigation.LSPLocation{
				{File: "agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14},
			},
			wantTest:  "TestRun",
			notWanted: "Test(*Agent).Run",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, tc.files)
			output := ExecuteSearchCode(SearchOptions{
				Pattern:   tc.pattern,
				Intent:    "impact",
				Path:      dir,
				FileType:  "go",
				LSPClient: &mockGoSymbolLSPClient{refs: tc.refs},
			})

			if !strings.Contains(output, "Related Tests") || !strings.Contains(output, tc.wantTest) {
				t.Fatalf("expected structured method impact probe to find %s, got:\n%s", tc.wantTest, output)
			}
			if strings.Contains(output, tc.notWanted) {
				t.Fatalf("expected probe to use base symbol name instead of %q, got:\n%s", tc.notWanted, output)
			}
		})
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeDoesNotCrossPackages(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

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
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected structured method probe to avoid unrelated cross-package TestRun matches, got:\n%s", output)
	}
	if strings.Contains(output, "job/job_test.go") {
		t.Fatalf("expected unrelated cross-package test file to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeFindsCrossPackageTests(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func (a *Agent) Run() error { return nil }

func UseAgent(a *Agent) error {
	return a.Run()
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	var a agentpkg.Agent
	_ = a.Run()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected cross-package method test probe to retain integration test, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeFindsCrossPackageTestsFromSubdirInvocation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func (a *Agent) Run() error { return nil }

func UseAgent(a *Agent) error {
	return a.Run()
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	var a agentpkg.Agent
	_ = a.Run()
}
`,
	})
	subdir := filepath.Join(dir, "agent")

	output := ExecuteSearchCode(SearchOptions{
		Pattern:            "(*Agent).Run",
		Intent:             "impact",
		Path:               dir,
		FileType:           "go",
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		LSPClient:          &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected cross-package method test probe to survive subdir invocation, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeFindsCrossPackageConstructorTests(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Run() error { return nil }

func UseAgent(a *Agent) error {
	return a.Run()
}
`,
		"integration/agent_test.go": `package integration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	a := agentpkg.New()
	_ = a.Run()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 8, Character: 11, EndLine: 8, EndChar: 14}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected constructor-based cross-package method test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeScansAllCrossPackageCandidates(t *testing.T) {
	files := map[string]string{
		"agent/agent.go": `package agent

type Agent struct{}

func (a *Agent) Run() error { return nil }
`,
		"zzzintegration/agent_test.go": `package zzzintegration

import (
	"testing"

	agentpkg "example/agent"
)

func TestRun(t *testing.T) {
	var a agentpkg.Agent
	_ = a.Run()
}
`,
	}
	for i := 0; i < 25; i++ {
		files[fmt.Sprintf("noise%02d/agent_test.go", i)] = fmt.Sprintf(`package noise%02d

import "testing"

func TestRun(t *testing.T) {
	t.Helper()
}
`, i)
	}

	dir := setupMultiLangDir(t, files)
	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "(*Agent).Run",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "agent/agent.go", Line: 4, Character: 17, EndLine: 4, EndChar: 20}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "zzzintegration/agent_test.go") || !strings.Contains(output, "TestRun") {
		t.Fatalf("expected structured method probe to scan past early unrelated TestRun candidates, got:\n%s", output)
	}
}
