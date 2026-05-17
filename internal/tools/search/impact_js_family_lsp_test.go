package search

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

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
	if !strings.Contains(artifact.Rendered, "Note: resolved via TypeScript/JavaScript LSP.") {
		t.Fatalf("expected JS family LSP source note, got:\n%s", artifact.Rendered)
	}
	if strings.Contains(artifact.Rendered, "resolved via gopls") {
		t.Fatalf("TypeScript LSP output must not claim gopls, got:\n%s", artifact.Rendered)
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

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactUsesLSPWorkspaceRefsFromDirectPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.ts": "export function buildUser(id: string) { return id }\n",
		"packages/other/src/app.ts": "import { buildUser } from '../../app/src/build'\nbuildUser('semantic workspace')\n",
	})
	subdir := filepath.Join(dir, "packages", "app")
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "../other/src/app.ts", Line: 2, Character: 1}},
	}
	opts := SearchOptions{
		Pattern:            "buildUser",
		Intent:             "impact",
		Path:               filepath.Join(dir, "packages", "app", "src", "build.ts"),
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		LSPClient:          lspClient,
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/other/src/app.ts")
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
	if !strings.Contains(artifact.Rendered, "Note: resolved via TypeScript/JavaScript LSP.") {
		t.Fatalf("expected JS family LSP source note, got:\n%s", artifact.Rendered)
	}
	if strings.Contains(artifact.Rendered, "resolved via gopls") {
		t.Fatalf("TypeScript LSP output must not claim gopls, got:\n%s", artifact.Rendered)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "createUser('semantic')") {
		t.Fatalf("callers = %+v, want alias caller from LSP range", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "fallback-only") {
		t.Fatalf("callers = %+v, did not want ripgrep fallback caller when LSP resolved refs", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactUsesLSPWorkspaceRefsFromDirectPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.js": "export function buildUser(id) { return id }\n",
		"packages/other/src/app.js": "import { buildUser } from '../../app/src/build.js'\nbuildUser('semantic workspace')\n",
	})
	subdir := filepath.Join(dir, "packages", "app")
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "../other/src/app.js", Line: 2, Character: 1}},
	}
	opts := SearchOptions{
		Pattern:            "buildUser",
		Intent:             "impact",
		Path:               filepath.Join(dir, "packages", "app", "src", "build.js"),
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		LSPClient:          lspClient,
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/other/src/app.js")
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactFiltersLSPReferencesByPathScope(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":  "function buildUser(id: string) { return id }\n",
		"src/app.ts":    "buildUser('src')\n",
		"other/app.ts":  "buildUser('other')\n",
		"vendor/app.ts": "buildUser('vendor')\n",
	})
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{
			{File: "src/app.ts", Line: 1, Character: 1},
			{File: "other/app.ts", Line: 1, Character: 1},
			{File: "vendor/app.ts", Line: 1, Character: 1},
		},
	}
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
	opts.Path = filepath.Join(dir, "src")
	opts.InvocationCWD = dir
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.ts") {
		t.Fatalf("callers = %+v, want in-scope LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "other/app.ts") || symbolBundleItemsContainSnippet(callers, "other") {
		t.Fatalf("callers = %+v, did not want sibling out-of-scope LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "vendor/app.ts") || symbolBundleItemsContainSnippet(callers, "vendor") {
		t.Fatalf("callers = %+v, did not want vendor out-of-scope LSP caller", callers)
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
	if !strings.Contains(artifact.Rendered, "Note: resolved via TypeScript/JavaScript LSP.") {
		t.Fatalf("expected JS family LSP source note, got:\n%s", artifact.Rendered)
	}
	if strings.Contains(artifact.Rendered, "resolved via gopls") {
		t.Fatalf("JavaScript LSP output must not claim gopls, got:\n%s", artifact.Rendered)
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
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") {
		t.Fatalf("callers = %+v, want in-scope LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "vendor/app.js") || symbolBundleItemsContainSnippet(callers, "vendor") {
		t.Fatalf("callers = %+v, did not want out-of-scope LSP caller", callers)
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

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFallsBackWhenLSPEvidenceCannotLoad(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\n",
		"src/app.js":   "buildUser('fallback')\n",
	})
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/missing.js", Line: 1, Character: 1}},
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = true, want false when LSP refs have no loadable evidence")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "buildUser('fallback')") {
		t.Fatalf("callers = %+v, want fallback AST caller", callers)
	}
}
