package search

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

type mockJSFamilyLSPClient struct {
	refs          []navigation.LSPLocation
	err           error
	requestedFile string
	requestedLine int
	requestedChar int
}

func (m *mockJSFamilyLSPClient) FindReferences(_ context.Context, filePath string, line, character int, _ bool) ([]navigation.LSPLocation, error) {
	m.requestedFile = filePath
	m.requestedLine = line
	m.requestedChar = character
	if m.err != nil {
		return nil, m.err
	}
	return m.refs, nil
}

func (m *mockJSFamilyLSPClient) GotoDefinition(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func (m *mockJSFamilyLSPClient) GotoImplementation(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactUsesLSPReferences(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser } from './build'\nbuildUser('semantic')\n",
		"src/noise.ts": "buildUser('fallback-only')\n",
	})
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.ts", Line: 2, Character: 1}},
	}
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	if lspClient.requestedLine != 1 || lspClient.requestedChar <= 1 {
		t.Fatalf("LSP request position = line %d char %d, want definition identifier position", lspClient.requestedLine, lspClient.requestedChar)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "buildUser('semantic')") {
		t.Fatalf("callers = %+v, want semantic LSP caller", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "fallback-only") {
		t.Fatalf("callers = %+v, did not want ripgrep fallback caller when LSP resolved refs", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactClassifiesLSPAliasReference(t *testing.T) {
	appLine := "const result = createUser('semantic')"
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser as createUser } from './build'\n" + appLine + "\n",
		"src/noise.ts": "buildUser('fallback-only')\n",
	})
	start, end := testLSPRangeForSearchToken(appLine, "createUser")
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.ts", Line: 2, Character: start, EndLine: 2, EndChar: end}},
	}
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "createUser('semantic')") {
		t.Fatalf("callers = %+v, want alias caller from LSP range", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "fallback-only") {
		t.Fatalf("callers = %+v, did not want ripgrep fallback caller when LSP resolved refs", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFiltersLSPReferencesByFileType(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\n",
		"src/app.js":   "buildUser('js')\n",
		"src/app.jsx":  "buildUser('jsx')\n",
	})
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{
			{File: "src/app.js", Line: 1, Character: 1},
			{File: "src/app.jsx", Line: 1, Character: 1},
		},
	}
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") {
		t.Fatalf("callers = %+v, want .js LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/app.jsx") || symbolBundleItemsContainSnippet(callers, "jsx") {
		t.Fatalf("callers = %+v, did not want .jsx caller for file_filter=js", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFiltersLSPReferencesByPathScope(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js":  "function buildUser(id) { return id }\n",
		"src/app.js":    "buildUser('src')\n",
		"vendor/app.js": "buildUser('vendor')\n",
	})
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{
			{File: "src/app.js", Line: 1, Character: 1},
			{File: "vendor/app.js", Line: 1, Character: 1},
		},
	}
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.Path = filepath.Join(dir, "src")
	opts.InvocationCWD = dir
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") {
		t.Fatalf("callers = %+v, want in-scope LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "vendor/app.js") || symbolBundleItemsContainSnippet(callers, "vendor") {
		t.Fatalf("callers = %+v, did not want out-of-scope LSP caller", callers)
	}
}

func TestJSFamilyRefFromLSPLocationRejectsIgnoredPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/generated.js": "buildUser('generated')\n",
	})
	opts := SearchOptions{
		Path:          dir,
		FileType:      "js",
		InvocationCWD: dir,
		ignoreMatcher: pathmatch.NewMatcher([]string{"src/generated.js"}),
	}

	_, ok := jsFamilyRefFromLSPLocation("buildUser", navigation.LSPLocation{
		File:      "src/generated.js",
		Line:      1,
		Character: 1,
	}, opts)

	if ok {
		t.Fatal("jsFamilyRefFromLSPLocation accepted ignored path, want reject")
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFallsBackWhenLSPHasNoRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\n",
		"src/app.js":   "buildUser('fallback')\n",
	})
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = true, want false for empty LSP result fallback")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "buildUser('fallback')") {
		t.Fatalf("callers = %+v, want fallback AST caller", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFallsBackWhenLSPFails(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\n",
		"src/app.js":   "buildUser('fallback')\n",
	})
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{err: errors.New("lsp unavailable")}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = true, want false for LSP error fallback")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "buildUser('fallback')") {
		t.Fatalf("callers = %+v, want fallback AST caller", callers)
	}
}
