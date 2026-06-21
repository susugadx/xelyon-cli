package search

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFiltersPatternsAndPath(t *testing.T) {
	tests := []struct {
		name       string
		defPath    string
		callerPath string
		opts       func(dir string) SearchOptions
	}{
		{
			name:       "js file filter",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactSearchOptions(dir, "buildUser")
			},
		},
		{
			name:       "basename js glob",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "*.js")
			},
		},
		{
			name:       "double star js glob",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "**/*.js")
			},
		},
		{
			name:       "src double star js glob",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "src/**/*.js")
			},
		},
		{
			name:       "package source double star js glob",
			defPath:    "packages/app/src/build.js",
			callerPath: "packages/app/src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "packages/*/src/**/*.js")
			},
		},
		{
			name:       "direct js path",
			defPath:    "packages/app/src/build.js",
			callerPath: "packages/app/src/app.js",
			opts: func(dir string) SearchOptions {
				return SearchOptions{
					Pattern:       "buildUser",
					Intent:        "impact",
					Path:          filepath.Join(dir, "packages", "app", "src", "build.js"),
					InvocationCWD: dir,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				tt.defPath:                    "export function buildUser(id) { return id }\n",
				tt.callerPath:                 "buildUser('1')\n",
				"packages/app/src/build.ts":   "export function buildUser(id: string) { return id }\n",
				"packages/app/src/view.jsx":   "export function buildUser() { return <></> }\n",
				"packages/other/src/other.js": "export function otherUser() { return 'other' }\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, tt.opts(dir))

			assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			if got := artifact.Metadata.Bundle.Definition.File; got != tt.defPath {
				t.Fatalf("definition file = %q, want %s", got, tt.defPath)
			}
			if strings.Contains(artifact.Rendered, "build.ts") || strings.Contains(artifact.Rendered, "view.jsx") {
				t.Fatalf("JavaScript structured impact should stay on .js scope, got:\n%s", artifact.Rendered)
			}
		})
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactUnsupportedFiltersFallback(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		filePath string
		source   string
	}{
		{name: "javascript filter", fileType: "javascript", filePath: "src/build.js", source: "export function buildUser() { return '1' }\n"},
		{name: "mjs filter", fileType: "mjs", filePath: "src/build.mjs", source: "export function buildUser() { return '1' }\n"},
		{name: "cjs filter", fileType: "cjs", filePath: "src/build.cjs", source: "function buildUser() { return '1' }\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{tt.filePath: tt.source})
			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
				Pattern:  "buildUser",
				Intent:   "impact",
				Path:     dir,
				FileType: tt.fileType,
			})
			if artifact.Metadata.StructuredImpact {
				t.Fatalf("StructuredImpact = true for unsupported JavaScript structured filter %q", tt.fileType)
			}
			if !strings.Contains(artifact.Rendered, tt.filePath) {
				t.Fatalf("expected fallback output to keep searching %s, got:\n%s", tt.filePath, artifact.Rendered)
			}
		})
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactCacheKeepsRecommendedReadsAndScope(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	dir := setupMultiLangDir(t, map[string]string{
		"src/shared.ts":  "export function buildUser() { return 'ts' }\n",
		"src/shared.tsx": "export function buildUser() { return <button /> }\n",
		"src/shared.js":  "export function buildUser() { return 'js' }\n",
		"src/app.ts":     "buildUser()\n",
		"src/App.tsx":    "export function App() { return <buildUser /> }\n",
		"src/app.js":     "buildUser()\n",
	})
	cache := &testSearchCache{data: make(map[string]string)}

	tsFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newTypeScriptImpactSearchOptions(dir, "buildUser"))
	assertTypeScriptStructuredImpactArtifact(t, tsFirst, "buildUser", "function")
	if got := tsFirst.Metadata.Bundle.Definition.File; got != "src/shared.ts" {
		t.Fatalf("ts definition file = %q, want src/shared.ts", got)
	}

	tsxFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newTSXImpactSearchOptions(dir, "buildUser"))
	assertTypeScriptStructuredImpactArtifact(t, tsxFirst, "buildUser", "function")
	if got := tsxFirst.Metadata.Bundle.Definition.File; got != "src/shared.tsx" {
		t.Fatalf("tsx definition file = %q, want src/shared.tsx", got)
	}

	jsFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newJavaScriptImpactSearchOptions(dir, "buildUser"))
	assertJavaScriptStructuredImpactArtifact(t, jsFirst, "buildUser", "function")
	if got := jsFirst.Metadata.Bundle.Definition.File; got != "src/shared.js" {
		t.Fatalf("js definition file = %q, want src/shared.js", got)
	}

	jsSecond := ExecuteSearchCodeArtifactWithConfig(nil, cache, newJavaScriptImpactSearchOptions(dir, "buildUser"))
	assertJavaScriptStructuredImpactArtifact(t, jsSecond, "buildUser", "function")
	if got := jsSecond.Metadata.Bundle.Definition.File; got != "src/shared.js" {
		t.Fatalf("cached js definition file = %q, want src/shared.js", got)
	}
	if len(jsSecond.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("cached JavaScript structured impact lost recommended reads")
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactAmbiguous(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/a.js": "export function buildUser(id) { return id }\n",
		"src/b.js": "export function buildUser(id) { return id }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:  "buildUser",
		Intent:   "impact",
		Path:     dir,
		FileType: "js",
	})

	if !artifact.Metadata.StructuredImpact {
		t.Fatal("StructuredImpact = false, want true for JavaScript ambiguous result")
	}
	if !artifact.Metadata.Ambiguous {
		t.Fatal("Ambiguous = false, want true")
	}
	if artifact.Metadata.Bundle != nil {
		t.Fatalf("Bundle = %+v, want nil for ambiguous JavaScript impact", artifact.Metadata.Bundle)
	}
	if len(artifact.Metadata.AffectedFiles) != 2 {
		t.Fatalf("AffectedFiles = %v, want two JavaScript candidates", artifact.Metadata.AffectedFiles)
	}
	for _, want := range []string{filepath.Join(dir, "src/a.js"), filepath.Join(dir, "src/b.js")} {
		if !slices.Contains(artifact.Metadata.AffectedFiles, want) {
			t.Fatalf("AffectedFiles = %v, want %s", artifact.Metadata.AffectedFiles, want)
		}
	}
	if !strings.Contains(artifact.Rendered, `Multiple definitions found for "buildUser":`) {
		t.Fatalf("expected JavaScript ambiguous output, got:\n%s", artifact.Rendered)
	}
}
