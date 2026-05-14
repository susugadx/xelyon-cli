package search

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type mockJSFamilyRawLSPClient struct {
	rootDir string
	refs    []lsp.Location
	err     error
}

func (m *mockJSFamilyRawLSPClient) FindReferences(context.Context, string, int, int, bool) ([]navigation.LSPLocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return navigation.ProtocolLocationsToLSPLocations(m.refs, m.rootDir), nil
}

func (m *mockJSFamilyRawLSPClient) GotoDefinition(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func (m *mockJSFamilyRawLSPClient) GotoImplementation(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func TestExecuteSearchCodeArtifactWithConfig_JSFamilyStructuredImpactUsesAdapterConvertedWorkspaceRefsFromDirectPath(t *testing.T) {
	tests := []struct {
		name           string
		extension      string
		definition     string
		assertArtifact func(*testing.T, SearchExecutionArtifact)
	}{
		{
			name:       "typescript",
			extension:  "ts",
			definition: "export function buildUser(id: string) { return id }\n",
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
		{
			name:       "javascript",
			extension:  "js",
			definition: "export function buildUser(id) { return id }\n",
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callLine := "buildUser('semantic workspace')"
			dir := setupMultiLangDir(t, map[string]string{
				"packages/app/src/build." + tt.extension:   tt.definition,
				"packages/other/src/app." + tt.extension:   "import { buildUser } from '../../app/src/build'\n" + callLine + "\n",
				"packages/other/src/noise." + tt.extension: "buildUser('fallback-only')\n",
			})
			subdir := filepath.Join(dir, "packages", "app")
			opts := SearchOptions{
				Pattern:            "buildUser",
				Intent:             "impact",
				Path:               filepath.Join(dir, "packages", "app", "src", "build."+tt.extension),
				ProjectMapRootPath: dir,
				InvocationCWD:      subdir,
				LSPClient: &mockJSFamilyRawLSPClient{
					rootDir: subdir,
					refs: []lsp.Location{
						rawJSFamilyLSPLocationForToken(filepath.Join(dir, "packages", "other", "src", "app."+tt.extension), 2, callLine, "buildUser"),
					},
				},
			}

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

			tt.assertArtifact(t, artifact)
			if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
				t.Fatal("ResolvedViaLSP = false, want true")
			}
			assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/other/src/app."+tt.extension)
			assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "packages/other/src/noise."+tt.extension)
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JSFamilyStructuredImpactFiltersAdapterConvertedRefsByDirectoryScope(t *testing.T) {
	tests := []struct {
		name           string
		extension      string
		definition     string
		newOptions     func(string) SearchOptions
		assertArtifact func(*testing.T, SearchExecutionArtifact)
	}{
		{
			name:       "typescript",
			extension:  "ts",
			definition: "export function buildUser(id: string) { return id }\n",
			newOptions: func(dir string) SearchOptions {
				opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
				opts.Path = filepath.Join(dir, "src")
				return opts
			},
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
		{
			name:       "javascript",
			extension:  "js",
			definition: "export function buildUser(id) { return id }\n",
			newOptions: func(dir string) SearchOptions {
				opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
				opts.Path = filepath.Join(dir, "src")
				return opts
			},
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcLine := "buildUser('src')"
			otherLine := "buildUser('other')"
			dir := setupMultiLangDir(t, map[string]string{
				"src/build." + tt.extension: tt.definition,
				"src/app." + tt.extension:   srcLine + "\n",
				"other/app." + tt.extension: otherLine + "\n",
			})
			opts := tt.newOptions(dir)
			opts.InvocationCWD = dir
			opts.ProjectMapRootPath = dir
			opts.LSPClient = &mockJSFamilyRawLSPClient{
				rootDir: dir,
				refs: []lsp.Location{
					rawJSFamilyLSPLocationForToken(filepath.Join(dir, "src", "app."+tt.extension), 1, srcLine, "buildUser"),
					rawJSFamilyLSPLocationForToken(filepath.Join(dir, "other", "app."+tt.extension), 1, otherLine, "buildUser"),
				},
			}

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

			tt.assertArtifact(t, artifact)
			if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
				t.Fatal("ResolvedViaLSP = false, want true")
			}
			assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "src/app."+tt.extension)
			assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "other/app."+tt.extension)
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFiltersAdapterConvertedRefsByFileType(t *testing.T) {
	jsLine := "buildUser('js')"
	jsxLine := "buildUser('jsx')"
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js":   jsLine + "\n",
		"src/app.jsx":  jsxLine + "\n",
	})
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyRawLSPClient{
		rootDir: dir,
		refs: []lsp.Location{
			rawJSFamilyLSPLocationForToken(filepath.Join(dir, "src", "app.js"), 1, jsLine, "buildUser"),
			rawJSFamilyLSPLocationForToken(filepath.Join(dir, "src", "app.jsx"), 1, jsxLine, "buildUser"),
		},
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "src/app.js")
	assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "src/app.jsx")
}

func TestExecuteSearchCodeArtifactWithConfig_JSFamilyStructuredImpactFiltersAdapterConvertedRefsByScopedGlob(t *testing.T) {
	tests := []struct {
		name           string
		extension      string
		definition     string
		newOptions     func(string) SearchOptions
		assertArtifact func(*testing.T, SearchExecutionArtifact)
	}{
		{
			name:       "typescript",
			extension:  "ts",
			definition: "export function buildUser(id: string) { return id }\n",
			newOptions: func(dir string) SearchOptions {
				return newTypeScriptImpactFilePatternSearchOptions(dir, "buildUser", "packages/app/src/**/*.ts")
			},
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
		{
			name:       "javascript",
			extension:  "js",
			definition: "export function buildUser(id) { return id }\n",
			newOptions: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "packages/app/src/**/*.js")
			},
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appLine := "buildUser('in scope')"
			siblingLine := "buildUser('sibling caller')"
			siblingTestLine := "buildUser('sibling test')"
			dir := setupMultiLangDir(t, map[string]string{
				"packages/app/src/build." + tt.extension:        tt.definition,
				"packages/app/src/app." + tt.extension:          appLine + "\n",
				"packages/other/src/app." + tt.extension:        siblingLine + "\n",
				"packages/other/src/build.test." + tt.extension: siblingTestLine + "\n",
			})
			opts := tt.newOptions(dir)
			opts.LSPClient = &mockJSFamilyRawLSPClient{
				rootDir: dir,
				refs: []lsp.Location{
					rawJSFamilyLSPLocationForToken(filepath.Join(dir, "packages", "app", "src", "app."+tt.extension), 1, appLine, "buildUser"),
					rawJSFamilyLSPLocationForToken(filepath.Join(dir, "packages", "other", "src", "app."+tt.extension), 1, siblingLine, "buildUser"),
					rawJSFamilyLSPLocationForToken(filepath.Join(dir, "packages", "other", "src", "build.test."+tt.extension), 1, siblingTestLine, "buildUser"),
				},
			}

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

			tt.assertArtifact(t, artifact)
			if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
				t.Fatal("ResolvedViaLSP = false, want true")
			}
			assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/app/src/app."+tt.extension)
			assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "packages/other/src/app."+tt.extension, "packages/other/src/build.test."+tt.extension)
		})
	}
}

func rawJSFamilyLSPLocationForToken(file string, line int, lineText string, token string) lsp.Location {
	start, end := testLSPRangeForSearchToken(lineText, token)
	return rawJSFamilyLSPLocation(file, line, start, end)
}

func rawJSFamilyLSPLocation(file string, line int, startCharacter int, endCharacter int) lsp.Location {
	return lsp.Location{
		URI: lsp.FileToURI(file),
		Range: lsp.Range{
			Start: lsp.Position{Line: line - 1, Character: startCharacter - 1},
			End:   lsp.Position{Line: line - 1, Character: endCharacter - 1},
		},
	}
}
