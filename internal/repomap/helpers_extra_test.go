package repomap

import (
	"strings"
	"testing"
)

func TestPatternLangAndExportHelpers(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "main.go", want: "go"},
		{path: "src/app.tsx", want: "js"},
		{path: "lib/tasks.py", want: "py"},
		{path: "src/lib.rs", want: "rs"},
		{path: "pkg/UserService.java", want: "java"},
		{path: "tool.sh", want: "sh"},
		{path: "README.md", want: ""},
	}
	for _, tt := range tests {
		if got := patternLangForPath(tt.path); got != tt.want {
			t.Fatalf("patternLangForPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}

	if !isExportedName("Builder") {
		t.Fatal("isExportedName(\"Builder\") = false, want true")
	}
	if isExportedName("builder") {
		t.Fatal("isExportedName(\"builder\") = true, want false")
	}
	if isExportedName("") {
		t.Fatal("isExportedName(\"\") = true, want false")
	}
}

func TestExtractSignatureMetadataForLang_RespectsLanguageFilter(t *testing.T) {
	if name, kind, ok := ExtractSignatureMetadataForLang("export function buildMap() {}", "js"); !ok || name != "buildMap" || kind != "function" {
		t.Fatalf("ExtractSignatureMetadataForLang(js) = (%q, %q, %v), want buildMap/function/true", name, kind, ok)
	}
	if _, _, ok := ExtractSignatureMetadataForLang("export function buildMap() {}", "go"); ok {
		t.Fatal("ExtractSignatureMetadataForLang(go) unexpectedly matched JS signature")
	}
	if name, kind, ok := ExtractSignatureMetadata("type Builder struct{}"); !ok || name != "Builder" || kind != "struct" {
		t.Fatalf("ExtractSignatureMetadata() = (%q, %q, %v), want Builder/struct/true", name, kind, ok)
	}
}

func TestProjectMapCountsAndFallbacks(t *testing.T) {
	pm := &ProjectMap{
		MaxTokens: 200,
		Files: []*FileEntry{
			{Path: "pkg/service.go", Symbols: []Symbol{{Name: "Build"}, {Name: "Run"}}},
			{Path: "pkg/service_test.go", Symbols: []Symbol{{Name: "TestBuild"}}},
		},
	}
	if got := pm.GetSymbolCount(); got != 3 {
		t.Fatalf("GetSymbolCount() = %d, want 3", got)
	}
	if got := pm.GetFileCount(); got != 2 {
		t.Fatalf("GetFileCount() = %d, want 2", got)
	}
	if got := (*ProjectMap)(nil).GetSymbolCount(); got != 0 {
		t.Fatalf("nil GetSymbolCount() = %d, want 0", got)
	}
	if got := (*ProjectMap)(nil).GetFileCount(); got != 0 {
		t.Fatalf("nil GetFileCount() = %d, want 0", got)
	}

	fallback := pm.generateManifestFallback(2, 1)
	if !strings.Contains(fallback, "Project map omitted to stay within budget") {
		t.Fatalf("generateManifestFallback() = %q, want budget message", fallback)
	}

	tiny := &ProjectMap{MaxTokens: 1}
	if got := tiny.generateManifestFallback(10, 3); got != "" {
		t.Fatalf("generateManifestFallback() = %q, want empty string when even fallback exceeds budget", got)
	}
}

func TestRenderAndSortHelpers(t *testing.T) {
	results := map[string][]Symbol{
		"pkg/service.go": {
			{Name: "Run", Signature: "func Run()", Line: 20},
			{Name: "Build", Signature: "func Build()", Line: 10},
			{Name: "BuildAlias", Signature: "func BuildAlias()", Line: 10, EndLine: 12},
		},
	}
	sortSymbolsByLocation(results)
	got := results["pkg/service.go"]
	if got[0].Name != "Build" || got[1].Name != "BuildAlias" || got[2].Name != "Run" {
		t.Fatalf("sortSymbolsByLocation() order = %+v", got)
	}

	var b strings.Builder
	writeRenderedSymbol(&b, "  ", Symbol{
		Line:      10,
		EndLine:   12,
		Signature: "func Build(\n\tctx context.Context,\n) error",
	})
	rendered := b.String()
	if !strings.Contains(rendered, "10-12: func Build(") {
		t.Fatalf("writeRenderedSymbol() missing location header, got %q", rendered)
	}
	if !strings.Contains(rendered, "\tctx context.Context,") {
		t.Fatalf("writeRenderedSymbol() missing multiline continuation, got %q", rendered)
	}

	if got := directoryDepth("main.go"); got != 0 {
		t.Fatalf("directoryDepth(main.go) = %d, want 0", got)
	}
	if got := directoryDepth("internal/agent/runner.go"); got != 2 {
		t.Fatalf("directoryDepth(internal/agent/runner.go) = %d, want 2", got)
	}
}

func TestGenerate_HandlesNilEmptyAndBudgetReduction(t *testing.T) {
	if got := (*ProjectMap)(nil).Generate(); got != "" {
		t.Fatalf("nil Generate() = %q, want empty string", got)
	}

	empty := &ProjectMap{}
	if got := empty.Generate(); got != "" {
		t.Fatalf("empty Generate() = %q, want empty string", got)
	}

	pm := &ProjectMap{
		MaxTokens: 18,
		Files: []*FileEntry{
			{Path: "pkg/service.go", LineCount: 40, Symbols: []Symbol{{Name: "Build", Line: 10, Signature: "func Build()"}}},
			{Path: "pkg/service_test.go", LineCount: 20, Symbols: []Symbol{{Name: "TestBuild", Line: 5, Signature: "func TestBuild()"}}},
			{Path: "internal/deep/nested.go", LineCount: 10, Symbols: []Symbol{{Name: "Nested", Line: 3, Signature: "func Nested()"}}},
		},
	}

	output := pm.Generate()
	if output == "" {
		t.Fatal("Generate() returned empty output unexpectedly")
	}
	if !strings.Contains(output, "truncated to fit token budget") {
		t.Fatalf("Generate() should report truncation under small budget, got:\n%s", output)
	}
	if strings.Contains(output, "func TestBuild()") {
		t.Fatalf("Generate() should drop test symbols before omitting files, got:\n%s", output)
	}
}
