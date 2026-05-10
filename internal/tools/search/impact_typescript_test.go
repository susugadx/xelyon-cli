package search

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactShapes(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		definition string
		wantKind   string
	}{
		{
			name:       "export function",
			symbol:     "buildUser",
			definition: "export function buildUser(id: string) { return id }\n",
			wantKind:   "function",
		},
		{
			name:       "export const arrow",
			symbol:     "buildUser",
			definition: "export const buildUser = (id: string) => id\n",
			wantKind:   "function",
		},
		{
			name:       "export typed const arrow",
			symbol:     "buildUser",
			definition: "type Builder = (id: string) => string\nexport const buildUser: Builder = (id: string) => id\n",
			wantKind:   "function",
		},
		{
			name:       "export inline typed const arrow",
			symbol:     "buildUser",
			definition: "export const buildUser: (id: string) => string = (id) => id\n",
			wantKind:   "function",
		},
		{
			name:       "export interface",
			symbol:     "BuildOptions",
			definition: "export interface BuildOptions { id: string }\n",
			wantKind:   "interface",
		},
		{
			name:       "non-export type alias",
			symbol:     "BuildOptions",
			definition: "type BuildOptions = { id: string }\n",
			wantKind:   "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/build.ts": tt.definition,
				"src/app.ts":   "import { buildUser } from './build'\nconst id: BuildOptions = { id: '1' }\nbuildUser?.('1')\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
				Pattern:       tt.symbol,
				Intent:        "impact",
				Path:          dir,
				FileType:      "ts",
				InvocationCWD: dir,
			})

			assertTypeScriptStructuredImpactArtifact(t, artifact, tt.symbol, tt.wantKind)
			if got := artifact.Metadata.Bundle.Impact.RecommendedReads[0].Kind; got != "definition" {
				t.Fatalf("first recommended read kind = %q, want definition", got)
			}
			if got := artifact.Metadata.Bundle.Impact.RecommendedReads[0].File; got != "src/build.ts" {
				t.Fatalf("definition recommended file = %q, want src/build.ts", got)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactIncludesFallbackExpansion(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":      "export function buildUser(id: string) { return id }\n",
		"src/build_impl.ts": "export function buildUserImpl(id: string) { return buildUser(id) }\n",
		"src/build.test.ts": "function TestBuildUser() { expect(1).toBe(1) }\n",
	})

	opts := SearchOptions{
		Pattern:       "buildUser",
		Intent:        "impact",
		Path:          dir,
		FileType:      "ts",
		InvocationCWD: dir,
	}
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.MultiPattern {
		t.Fatal("MultiPattern = false, want true when TypeScript structured impact keeps impact expansion")
	}
	for _, want := range []string{"buildUserImpl", "TestBuildUser", "src/build_impl.ts", "src/build.test.ts"} {
		if !strings.Contains(artifact.Rendered, want) {
			t.Fatalf("expected structured impact expansion output to contain %q, got:\n%s", want, artifact.Rendered)
		}
	}
	for _, want := range []string{filepath.Join(dir, "src/build_impl.ts"), filepath.Join(dir, "src/build.test.ts")} {
		if !slices.Contains(artifact.Metadata.AffectedFiles, want) {
			t.Fatalf("AffectedFiles = %v, want %s", artifact.Metadata.AffectedFiles, want)
		}
	}

	rendered := ExecuteSearchCodeWithConfig(nil, nil, opts)
	for _, want := range []string{"buildUserImpl", "TestBuildUser"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected executeImpactSearch output to contain %q, got:\n%s", want, rendered)
		}
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactFilePattern(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":  "export function buildUser(id: string) { return id }\n",
		"src/build.tsx": "export function buildUser(id: string) { return <>{id}</> }\n",
		"src/app.ts":    "buildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:     "buildUser",
		Intent:      "impact",
		Path:        dir,
		FilePattern: "*.ts",
	})

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if strings.Contains(artifact.Rendered, "build.tsx") {
		t.Fatalf("structured TypeScript impact should stay on *.ts files, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactRelatedTests(t *testing.T) {
	tests := []struct {
		name     string
		testPath string
	}{
		{name: "sibling test", testPath: "src/build.test.ts"},
		{name: "__tests__ directory", testPath: "src/__tests__/build.test.ts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/build.ts": "export function buildUser(id: string) { return id }\n",
				"src/app.ts":   "import { buildUser } from './build'\nbuildUser('1')\n",
				tt.testPath:    "import { buildUser } from '../build'\nbuildUser('test')\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
				Pattern:       "buildUser",
				Intent:        "impact",
				Path:          dir,
				FileType:      "ts",
				InvocationCWD: dir,
			})

			assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			if !recommendedReadsContainFile(artifact.Metadata.Bundle, tt.testPath) {
				t.Fatalf("expected recommended reads to include %s, got %v", tt.testPath, recommendedReadFiles(artifact.Metadata.Bundle))
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactPathArtifact(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern: "buildUser",
		Intent:  "impact",
		Path:    filepath.Join(dir, "src", "build.ts"),
	})

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if len(artifact.Metadata.AffectedFiles) == 0 {
		t.Fatal("AffectedFiles should not be empty for TypeScript structured impact")
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactCacheKeepsRecommendedReads(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser } from './build'\nbuildUser('1')\n",
	})
	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:       "buildUser",
		Intent:        "impact",
		Path:          dir,
		FileType:      "ts",
		InvocationCWD: dir,
	}

	first := ExecuteSearchCodeArtifactWithConfig(nil, cache, opts)
	assertTypeScriptStructuredImpactArtifact(t, first, "buildUser", "function")
	second := ExecuteSearchCodeArtifactWithConfig(nil, cache, opts)
	assertTypeScriptStructuredImpactArtifact(t, second, "buildUser", "function")

	if len(second.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("cached TypeScript structured impact lost recommended reads")
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactAmbiguous(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/a.ts": "export function buildUser(id: string) { return id }\n",
		"src/b.ts": "export function buildUser(id: string) { return id }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:  "buildUser",
		Intent:   "impact",
		Path:     dir,
		FileType: "ts",
	})

	if !artifact.Metadata.StructuredImpact {
		t.Fatal("StructuredImpact = false, want true for TypeScript ambiguous result")
	}
	if !artifact.Metadata.Ambiguous {
		t.Fatal("Ambiguous = false, want true")
	}
	if artifact.Metadata.Bundle != nil {
		t.Fatalf("Bundle = %+v, want nil for ambiguous TypeScript impact", artifact.Metadata.Bundle)
	}
	if len(artifact.Metadata.AffectedFiles) != 2 {
		t.Fatalf("AffectedFiles = %v, want two TypeScript candidates", artifact.Metadata.AffectedFiles)
	}
	if !strings.Contains(artifact.Rendered, `Multiple definitions found for "buildUser":`) {
		t.Fatalf("expected TypeScript ambiguous output, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactUnsupportedFiltersFallback(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		filePath string
		source   string
	}{
		{name: "tsx filter", fileType: "tsx", filePath: "src/widget.tsx", source: "export function buildUser() { return <></> }\n"},
		{name: "js filter", fileType: "js", filePath: "src/build.js", source: "export function buildUser() { return '1' }\n"},
		{name: "jsx filter", fileType: "jsx", filePath: "src/widget.jsx", source: "export function buildUser() { return <></> }\n"},
		{name: "javascript filter", fileType: "javascript", filePath: "src/build.js", source: "export function buildUser() { return '1' }\n"},
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
				t.Fatalf("StructuredImpact = true for unsupported TypeScript structured filter %q", tt.fileType)
			}
			if !strings.Contains(artifact.Rendered, tt.filePath) {
				t.Fatalf("expected fallback output to keep searching %s, got:\n%s", tt.filePath, artifact.Rendered)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptFilterFallbackStillIncludesTSX(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/widget.tsx": "export function buildUser() { return <></> }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:  "buildUser",
		Intent:   "impact",
		Path:     dir,
		FileType: "typescript",
	})

	if artifact.Metadata.StructuredImpact {
		t.Fatal("StructuredImpact = true, want fallback when only .tsx matches file_filter=typescript")
	}
	if !strings.Contains(artifact.Rendered, "src/widget.tsx") {
		t.Fatalf("expected fallback file_filter=typescript to retain .tsx contract, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptFilterFallbackIncludesTSXCallers(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/view.tsx": "import { buildUser } from './build'\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:  "buildUser",
		Intent:   "impact",
		Path:     dir,
		FileType: "typescript",
	})

	if artifact.Metadata.StructuredImpact {
		t.Fatal("StructuredImpact = true, want fallback so file_filter=typescript keeps .ts and .tsx scope")
	}
	for _, want := range []string{"src/build.ts", "src/view.tsx"} {
		if !strings.Contains(artifact.Rendered, want) {
			t.Fatalf("expected fallback file_filter=typescript to include %s, got:\n%s", want, artifact.Rendered)
		}
	}
}

func assertTypeScriptStructuredImpactArtifact(t *testing.T, artifact SearchExecutionArtifact, symbol string, kind string) {
	t.Helper()
	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want true; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = true, want false; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want TypeScript structured bundle; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle.Identity.Language != "typescript" {
		t.Fatalf("bundle language = %q, want typescript", artifact.Metadata.Bundle.Identity.Language)
	}
	if artifact.Metadata.Bundle.Identity.DisplayName != symbol {
		t.Fatalf("bundle display name = %q, want %q", artifact.Metadata.Bundle.Identity.DisplayName, symbol)
	}
	if artifact.Metadata.Bundle.Identity.Kind != kind {
		t.Fatalf("bundle kind = %q, want %q", artifact.Metadata.Bundle.Identity.Kind, kind)
	}
	if artifact.Metadata.Bundle.Impact == nil {
		t.Fatal("Bundle.Impact = nil, want TypeScript impact metadata")
	}
	if len(artifact.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("RecommendedReads is empty, want definition read")
	}
}

func recommendedReadsContainFile(bundle *SymbolBundle, file string) bool {
	if bundle == nil || bundle.Impact == nil {
		return false
	}
	for _, item := range bundle.Impact.RecommendedReads {
		if item.File == file {
			return true
		}
	}
	return false
}
