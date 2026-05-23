package search

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptSymbolFiltersLSPReferencesByPathScope(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "buildUser('src')\n",
		"other/app.ts": "buildUser('other')\n",
	})
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{
			{File: "src/app.ts", Line: 1, Character: 1},
			{File: "other/app.ts", Line: 1, Character: 1},
		},
	}
	opts := SearchOptions{
		Pattern:       "buildUser",
		Mode:          string(SearchModeSymbol),
		Path:          filepath.Join(dir, "src"),
		FileType:      "ts",
		InvocationCWD: dir,
		LSPClient:     lspClient,
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want TypeScript symbol bundle; output:\n%s", artifact.Rendered)
	}
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.ts") {
		t.Fatalf("callers = %+v, want in-scope LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "other/app.ts") || symbolBundleItemsContainSnippet(callers, "other") {
		t.Fatalf("callers = %+v, did not want out-of-scope LSP caller", callers)
	}
}
