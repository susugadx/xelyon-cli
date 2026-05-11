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
			name:       "export generic const arrow",
			symbol:     "buildUser",
			definition: "export const buildUser = <T>(id: T): T => id\n",
			wantKind:   "function",
		},
		{
			name:       "export async generic const arrow",
			symbol:     "buildUser",
			definition: "export const buildUser = async <T extends string>(id: T) => id\n",
			wantKind:   "function",
		},
		{
			name:       "export single-param const arrow",
			symbol:     "buildUser",
			definition: "export const buildUser = id => id\n",
			wantKind:   "function",
		},
		{
			name:       "export async single-param const arrow",
			symbol:     "buildUser",
			definition: "export const buildUser = async id => id\n",
			wantKind:   "function",
		},
		{
			name:       "export default function",
			symbol:     "buildOrg",
			definition: "export default function buildOrg() { return 'org' }\n",
			wantKind:   "function",
		},
		{
			name:       "export default class",
			symbol:     "UserBuilder",
			definition: "export default class UserBuilder {}\n",
			wantKind:   "class",
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

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, tt.symbol))

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

	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
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

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactFilePatterns(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		defPath    string
		callerPath string
	}{
		{name: "basename glob", pattern: "*.ts", defPath: "src/build.ts", callerPath: "src/app.ts"},
		{name: "double star glob", pattern: "**/*.ts", defPath: "src/build.ts", callerPath: "src/app.ts"},
		{name: "src double star glob", pattern: "src/**/*.ts", defPath: "src/build.ts", callerPath: "src/app.ts"},
		{name: "package source double star glob", pattern: "packages/*/src/**/*.ts", defPath: "packages/app/src/build.ts", callerPath: "packages/app/src/app.ts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				tt.defPath:      "export function buildUser(id: string) { return id }\n",
				tt.callerPath:   "buildUser('1')\n",
				"src/build.tsx": "export function buildUser(id: string) { return <>{id}</> }\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactFilePatternSearchOptions(dir, "buildUser", tt.pattern))

			assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			if strings.Contains(artifact.Rendered, "build.tsx") {
				t.Fatalf("structured TypeScript impact should stay on .ts-only pattern %q, got:\n%s", tt.pattern, artifact.Rendered)
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
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")

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
		{name: "js filter", fileType: "js", filePath: "src/build.js", source: "export function buildUser() { return '1' }\n"},
		{name: "jsx filter", fileType: "jsx", filePath: "src/widget.jsx", source: "export function buildUser() { return <></> }\n"},
		{name: "mjs filter", fileType: "mjs", filePath: "src/build.mjs", source: "export function buildUser() { return '1' }\n"},
		{name: "cjs filter", fileType: "cjs", filePath: "src/build.cjs", source: "function buildUser() { return '1' }\n"},
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
