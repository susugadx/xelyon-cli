package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsReceiverPrecision(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"config_test.go": `package example

import "testing"

func TestBuild(t *testing.T) {
	var b Builder
	_ = b.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 9, Character: 11, EndLine: 9, EndChar: 16}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected method probe to avoid same-package wrong-receiver TestBuild matches, got:\n%s", output)
	}
	if strings.Contains(output, "config_test.go") {
		t.Fatalf("expected wrong-receiver same-package test file to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesUnknownSelectorChain(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"config_test.go": `package example

import "testing"

type Outer struct {
	B Builder
}

func TestBuild(t *testing.T) {
	var outer Outer
	_ = outer.B.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 9, Character: 11, EndLine: 9, EndChar: 16}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected unknown selector-chain wrong receiver TestBuild to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsEmbeddedFieldSelectorChain(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"config_test.go": `package example

import "testing"

type Outer struct {
	Config
}

func TestBuild(t *testing.T) {
	var outer Outer
	_ = outer.Config.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 9, Character: 11, EndLine: 9, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected embedded-field selector chain test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsLowercaseWrapperSelectorChain(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"config_test.go": `package example

import "testing"

type Service struct {
	client Config
}

func TestBuild(t *testing.T) {
	var svc Service
	_ = svc.client.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 9, Character: 11, EndLine: 9, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected lowercase wrapper selector chain test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesTestOnlyFakeReceiver(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"config_test.go": `package example

import "testing"

type FakeBuilder struct{}

func (FakeBuilder) Build() string { return "" }

func TestBuild(t *testing.T) {
	var fake FakeBuilder
	_ = fake.Build()
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

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected test-only fake receiver TestBuild to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeKeepsPromotedOrInterfaceCalls(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}
type Wrapper struct{ Config }
type Buildable interface{ Build() string }

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"config_test.go": `package example

import "testing"

func TestBuild(t *testing.T) {
	var w Wrapper
	var iface Buildable = w
	_ = iface.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 11, Character: 11, EndLine: 11, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected promoted/interface call test to remain discoverable, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeHandlesMultipleCallsPerLine(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}
type Builder struct{}

func (Config) Build() string { return "" }
func (Builder) Build() string { return "" }

func UseConfig(c Config) string {
	return c.Build()
}
`,
		"config_test.go": `package example

import "testing"

func TestBuild(t *testing.T) {
	var b Builder
	var c Config
	_, _ = b.Build(), c.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 9, Character: 11, EndLine: 9, EndChar: 16}}},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestBuild") {
		t.Fatalf("expected multi-call single-line test to remain discoverable, got:\n%s", output)
	}
}
