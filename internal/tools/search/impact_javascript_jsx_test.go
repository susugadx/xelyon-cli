package search

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

func newJSXImpactSearchOptions(dir string, symbol string) SearchOptions {
	opts := newJavaScriptImpactSearchOptions(dir, symbol)
	opts.FileType = "jsx"
	return opts
}

func TestExecuteSearchCodeArtifactWithConfig_JSXStructuredImpactComponentCallers(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.jsx":      "export function Button() { return <button /> }\n",
		"src/App.jsx":         "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
		"src/Button.test.jsx": "import { Button } from './Button'\nit('renders', () => <Button />)\n",
		"src/Button.spec.jsx": "describe('component', () => {})\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJSXImpactSearchOptions(dir, "Button"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/Button.jsx" {
		t.Fatalf("definition file = %q, want src/Button.jsx", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "<Button />") || !symbolBundleItemsContainFile(callers, "src/App.jsx") {
		t.Fatalf("callers = %+v, want JSX usage in src/App.jsx", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/Button.test.jsx") {
		t.Fatalf("direct JSX test should be separated from callers, got %+v", callers)
	}
	assertRecommendedReadAt(t, artifact.Metadata.Bundle.Impact.RecommendedReads, 0, "definition", "src/Button.jsx")
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/Button.test.jsx") {
		t.Fatalf("expected direct JSX test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
	if !symbolBundleItemsContainFile(symbolBundleSectionItems(artifact.Metadata.Bundle, "tests"), "src/Button.spec.jsx") {
		t.Fatalf("expected nearby JSX test in related tests, got %+v", symbolBundleSectionItems(artifact.Metadata.Bundle, "tests"))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JSXStructuredImpactDefaultWrappedComponent(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.jsx": "import { memo } from 'react'\nexport default memo(function DefaultButton() { return <button /> })\n",
		"src/App.jsx":    "import DefaultButton from './Button'\nexport function App() { return <DefaultButton /> }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJSXImpactSearchOptions(dir, "DefaultButton"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "DefaultButton", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/Button.jsx" {
		t.Fatalf("definition file = %q, want src/Button.jsx", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "<DefaultButton />") {
		t.Fatalf("callers = %+v, want JSX usage caller", callers)
	}
}

func TestStructuredJavaScriptImpactSearchContextAllowsJSX(t *testing.T) {
	opts := SearchOptions{
		Pattern:  "Button",
		Intent:   "impact",
		Path:     ".",
		FileType: "jsx",
	}

	ctx, scope, ok := newStructuredJavaScriptImpactSearchContext(opts)

	if !ok {
		t.Fatal("newStructuredJavaScriptImpactSearchContext() rejected jsx impact opts")
	}
	if ctx.Route.Language != "js" {
		t.Fatalf("route language = %q, want js", ctx.Route.Language)
	}
	if scope.Definition.FileType != "jsx" || scope.Evidence.FileType != "jsx" {
		t.Fatalf("scope file type = (%q, %q), want jsx", scope.Definition.FileType, scope.Evidence.FileType)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JSXStructuredImpactFiltersPatternsAndPath(t *testing.T) {
	tests := []struct {
		name string
		opts func(dir string) SearchOptions
	}{
		{
			name: "jsx file filter",
			opts: func(dir string) SearchOptions {
				return newJSXImpactSearchOptions(dir, "Button")
			},
		},
		{
			name: "basename jsx glob",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "Button", "*.jsx")
			},
		},
		{
			name: "double star jsx glob",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "Button", "**/*.jsx")
			},
		},
		{
			name: "src double star jsx glob",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "Button", "src/**/*.jsx")
			},
		},
		{
			name: "direct jsx path",
			opts: func(dir string) SearchOptions {
				return SearchOptions{
					Pattern:       "Button",
					Intent:        "impact",
					Path:          filepath.Join(dir, "src", "Button.jsx"),
					InvocationCWD: dir,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/Button.jsx": "export function Button() { return <button /> }\n",
				"src/App.jsx":    "export function App() { return <Button /> }\n",
				"src/Button.js":  "export function Button() { return 'js' }\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, tt.opts(dir))

			assertJavaScriptStructuredImpactArtifact(t, artifact, "Button", "function")
			if got := artifact.Metadata.Bundle.Definition.File; got != "src/Button.jsx" {
				t.Fatalf("definition file = %q, want src/Button.jsx", got)
			}
			if strings.Contains(artifact.Rendered, "src/Button.js ") {
				t.Fatalf("JSX structured impact should stay on .jsx scope, got:\n%s", artifact.Rendered)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JSXStructuredImpactCacheKeepsScope(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	dir := setupMultiLangDir(t, map[string]string{
		"src/shared.js":  "export function buildUser() { return 'js' }\n",
		"src/shared.jsx": "export function buildUser() { return <button /> }\n",
		"src/app.js":     "buildUser()\n",
		"src/App.jsx":    "export function App() { return <buildUser /> }\n",
	})
	cache := &testSearchCache{data: make(map[string]string)}

	jsFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newJavaScriptImpactSearchOptions(dir, "buildUser"))
	assertJavaScriptStructuredImpactArtifact(t, jsFirst, "buildUser", "function")
	if got := jsFirst.Metadata.Bundle.Definition.File; got != "src/shared.js" {
		t.Fatalf("js definition file = %q, want src/shared.js", got)
	}

	jsxFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newJSXImpactSearchOptions(dir, "buildUser"))
	assertJavaScriptStructuredImpactArtifact(t, jsxFirst, "buildUser", "function")
	if got := jsxFirst.Metadata.Bundle.Definition.File; got != "src/shared.jsx" {
		t.Fatalf("jsx definition file = %q, want src/shared.jsx", got)
	}

	jsxSecond := ExecuteSearchCodeArtifactWithConfig(nil, cache, newJSXImpactSearchOptions(dir, "buildUser"))
	assertJavaScriptStructuredImpactArtifact(t, jsxSecond, "buildUser", "function")
	if got := jsxSecond.Metadata.Bundle.Definition.File; got != "src/shared.jsx" {
		t.Fatalf("cached jsx definition file = %q, want src/shared.jsx", got)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JSXStructuredImpactFiltersAdapterConvertedRefsByFileType(t *testing.T) {
	jsxLine := "export function App() { return <Button /> }"
	jsLine := "Button('js')"
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.jsx": "export function Button() { return <button /> }\n",
		"src/App.jsx":    jsxLine + "\n",
		"src/app.js":     jsLine + "\n",
	})
	opts := newJSXImpactSearchOptions(dir, "Button")
	opts.LSPClient = &mockJSFamilyRawLSPClient{
		rootDir: dir,
		refs: []lsp.Location{
			rawJSFamilyLSPLocationForToken(filepath.Join(dir, "src", "App.jsx"), 1, jsxLine, "Button"),
			rawJSFamilyLSPLocationForToken(filepath.Join(dir, "src", "app.js"), 1, jsLine, "Button"),
		},
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/App.jsx") || !symbolBundleItemsContainSnippet(callers, "<Button />") {
		t.Fatalf("callers = %+v, want JSX LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/app.js") || symbolBundleItemsContainSnippet(callers, "Button('js')") {
		t.Fatalf("callers = %+v, did not want .js caller for file_filter=jsx", callers)
	}
}
