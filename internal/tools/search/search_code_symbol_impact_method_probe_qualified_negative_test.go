package search

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesDotImportedForeignReceiver(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
		"otherpkg/builder.go": `package otherpkg

type Builder struct{}

func (Builder) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"
	. "example/otherpkg"
)

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
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 5, Character: 11, EndLine: 5, EndChar: 16}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected dot-imported foreign receiver to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesQualifiedImportedReceiverWithoutLocalAlternative(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"

	otherpkg "example/otherpkg"
)

func TestBuild(t *testing.T) {
	var b otherpkg.Builder
	_ = b.Build()
}
`,
		"otherpkg/builder.go": `package otherpkg

type Builder struct{}

func (Builder) Build() string { return "" }
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 5, Character: 11, EndLine: 5, EndChar: 16}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected qualified imported receiver without local alternative to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesQualifiedImportedReceiverWithProjectMap(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"go.mod": "module example\n\ngo 1.25.0\n",
		"config.go": `package example

type Config struct{}
type Builder interface {
	Build() string
}

func (Config) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"

	otherpkg "example/otherpkg"
)

func TestBuild(t *testing.T) {
	var b otherpkg.Builder
	_ = b.Build()
}
`,
		"otherpkg/builder.go": `package otherpkg

type Builder struct{}

func (Builder) Build() string { return "" }
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:  "Config.Build",
		Intent:   "impact",
		Path:     dir,
		FileType: "go",
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "config.go",
					Symbols: []repomap.Symbol{
						{Name: "Config", Kind: "struct", Line: 3, EndLine: 3, Signature: "type Config struct{}", Exported: true},
						{Name: "Builder", Kind: "interface", Line: 4, EndLine: 6, Signature: "type Builder interface { Build() string }", Exported: true},
						{Name: "Build", Kind: "method", Line: 8, EndLine: 8, Signature: "func (Config) Build() string", Exported: true},
					},
				},
				{
					Path: "otherpkg/builder.go",
					Symbols: []repomap.Symbol{
						{Name: "Builder", Kind: "struct", Line: 3, EndLine: 3, Signature: "type Builder struct{}", Exported: true},
						{Name: "Build", Kind: "method", Line: 5, EndLine: 5, Signature: "func (Builder) Build() string", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "qualified-import-projectmap",
		LSPClient:          &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 8, Character: 11, EndLine: 8, EndChar: 16}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected qualified imported receiver with project map to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesVersionedLocalImportByPackageName(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
		"sdk/v2/builder.go": `package sdk

type Builder struct{}

func (Builder) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"

	"example/sdk/v2"
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
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 5, Character: 11, EndLine: 5, EndChar: 16}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected versioned local import by package name to be excluded, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesQualifiedImportedReceiverFromSubdirSearch(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config/config.go": `package config

type Config struct{}

func (Config) Build() string { return "" }
`,
		"otherpkg/builder.go": `package otherpkg

type Builder struct{}

func (Builder) Build() string { return "" }
`,
		"config/config_test.go": `package config

import (
	"testing"

	otherpkg "example/otherpkg"
)

func TestBuild(t *testing.T) {
	var b otherpkg.Builder
	_ = b.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      filepath.Join(dir, "config"),
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config/config.go", Line: 5, Character: 11, EndLine: 5, EndChar: 16}}},
	})

	if strings.Contains(output, "Related Tests") {
		t.Fatalf("expected qualified imported Builder.Build to be excluded in subdir-scoped search, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoMethodTestProbeExcludesQualifiedImportedReceiverWithLocalAlternative(t *testing.T) {
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

import (
	"testing"

	otherpkg "example/otherpkg"
)

func TestBuild(t *testing.T) {
	var b otherpkg.Builder
	_ = b.Build()
}
`,
		"otherpkg/builder.go": `package otherpkg

type Builder struct{}

func (Builder) Build() string { return "" }
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
		t.Fatalf("expected qualified imported Builder.Build to be excluded when local Builder alternative exists, got:\n%s", output)
	}
}
