package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsImportedInterfaceCalls(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
		"otherpkg/buildable.go": `package otherpkg

type Buildable interface{ Build() string }
`,
		"integration/config_test.go": `package integration

import (
	"testing"

	examplepkg "example"
	otherpkg "example/otherpkg"
)

type Wrapper struct{ examplepkg.Config }

func TestBuild(t *testing.T) {
	var b otherpkg.Buildable = Wrapper{Config: examplepkg.Config{}}
	_ = b.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 5, Character: 11, EndLine: 5, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/config_test.go") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected imported interface-based integration test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsImportedInterfaceCallsThroughHelperChain(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }
`,
		"otherpkg/buildable.go": `package otherpkg

type Buildable interface{ Build() string }
`,
		"integration/helpers.go": `package integration

import (
	"testing"

	otherpkg "example/otherpkg"
)

func assertBuild(t *testing.T, b otherpkg.Buildable) {
	t.Helper()
	forwardBuild(b)
}

func forwardBuild(b otherpkg.Buildable) {
	_ = b.Build()
}
`,
		"integration/config_test.go": `package integration

import (
	"testing"

	examplepkg "example"
	otherpkg "example/otherpkg"
)

type Wrapper struct{ examplepkg.Config }

func TestBuild(t *testing.T) {
	var b otherpkg.Buildable = Wrapper{Config: examplepkg.Config{}}
	assertBuild(t, b)
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 6, Character: 11, EndLine: 6, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/config_test.go") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected imported interface helper-chain test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsExternalQualifiedInterfaceCalls(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"

	sdk "github.com/acme/sdk"
)

func TestBuild(t *testing.T) {
	var b sdk.Builder
	_ = b.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 7, Character: 11, EndLine: 7, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "config_test.go") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected external qualified interface receiver to remain ambiguous and discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsExternalQualifiedInterfaceCallsWithLocalSuffixDir(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"go.mod": `module example

go 1.25.0
`,
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"

	sdk "github.com/acme/sdk"
)

func TestBuild(t *testing.T) {
	var b sdk.Builder
	_ = b.Build()
}
`,
		"sdk/builder.go": `package sdk

type Builder struct{}

func (Builder) Build() string { return "" }
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 7, Character: 11, EndLine: 7, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "config_test.go") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected external qualified receiver to remain ambiguous even with local suffix dir, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsImportedWrapperCalls(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
		"otherpkg/wrapper.go": `package otherpkg

import examplepkg "example"

type Wrapper struct{ examplepkg.Config }
`,
		"integration/config_test.go": `package integration

import (
	"testing"

	otherpkg "example/otherpkg"
)

func TestBuild(t *testing.T) {
	var w otherpkg.Wrapper
	_ = w.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 4, Character: 11, EndLine: 4, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "integration/config_test.go") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected imported wrapper-based integration test to remain discoverable, got:\n%s", output)
	}
}
