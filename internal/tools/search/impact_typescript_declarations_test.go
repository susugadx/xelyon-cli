package search

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactPrefersImplementationOverDeclaration(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":   "export function buildUser(id: string) { return id }\n",
		"src/build.d.ts": "export function buildUser(id: string): string\ntype BuildUserFactory = () => buildUser\n",
		"src/app.ts":     "import { buildUser } from './build'\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/build.ts" {
		t.Fatalf("definition file = %q, want src/build.ts", got)
	}
	if artifact.Metadata.Ambiguous {
		t.Fatal("Ambiguous = true, want false when implementation and declaration both define the symbol")
	}
	if strings.Contains(artifact.Rendered, `Multiple definitions found for "buildUser"`) {
		t.Fatalf("declaration candidate should not force ambiguity when implementation exists, got:\n%s", artifact.Rendered)
	}
	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 1, "callers", "src/app.ts")
	assertTypeScriptImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "src/build.d.ts")
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactDirectPathSuppressesPairedDeclarationRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":   "export function buildUser(id: string) { return id }\n",
		"src/build.d.ts": "export function buildUser(id: string): string\ntype BuildUserFactory = () => buildUser\n",
		"src/app.ts":     "import { buildUser } from './build'\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:       "buildUser",
		Intent:        "impact",
		Path:          filepath.Join(dir, "src", "build.ts"),
		InvocationCWD: dir,
	})

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/build.ts" {
		t.Fatalf("definition file = %q, want src/build.ts", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.ts") {
		t.Fatalf("callers = %+v, want workspace caller after direct definition path", callers)
	}
	assertTypeScriptImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "src/build.d.ts")
	if strings.Contains(artifact.Rendered, "src/build.d.ts") {
		t.Fatalf("paired declaration should stay suppressed after widened refs, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactKeepsUnrelatedDeclarationAmbiguous(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/user.ts":       "export function User(id: string) { return id }\n",
		"types/global.d.ts": "export interface User { id: string }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "User"))

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want true; output:\n%s", artifact.Rendered)
	}
	if !artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = false, want true for unrelated declaration definition; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle != nil {
		t.Fatalf("Bundle = %+v, want nil for ambiguous unrelated declaration", artifact.Metadata.Bundle)
	}
	for _, want := range []string{"src/user.ts", "types/global.d.ts"} {
		if !strings.Contains(artifact.Rendered, want) {
			t.Fatalf("expected ambiguous output to include %s, got:\n%s", want, artifact.Rendered)
		}
	}
	for _, want := range []string{filepath.Join(dir, "src/user.ts"), filepath.Join(dir, "types/global.d.ts")} {
		if !slices.Contains(artifact.Metadata.AffectedFiles, want) {
			t.Fatalf("AffectedFiles = %v, want %s", artifact.Metadata.AffectedFiles, want)
		}
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactDeclarationOnly(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/types.d.ts":      "export interface BuildOptions { id: string }\n",
		"src/app.ts":          "import type { BuildOptions } from './types'\nconst options = data as BuildOptions\nconst raw = BuildOptions\n",
		"src/types.d.test.ts": "describe('types declarations', () => {})\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "BuildOptions"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "BuildOptions", "interface")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/types.d.ts" {
		t.Fatalf("definition file = %q, want src/types.d.ts", got)
	}
	if source := artifact.Metadata.Bundle.Debug.Source; !strings.Contains(source, "declaration") {
		t.Fatalf("Debug.Source = %q, want declaration source", source)
	}
	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/types.d.ts")
	assertRecommendedReadAt(t, reads, 1, "type_refs", "src/app.ts")
	assertRecommendedReadAt(t, reads, 2, "imports", "src/app.ts")
	assertRecommendedReadAt(t, reads, 3, "references", "src/app.ts")
	if recommendedReadsContainFile(artifact.Metadata.Bundle, "src/types.d.test.ts") {
		t.Fatalf("declaration impact should not add nearby tests, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactDeclareForms(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		definition string
		wantKind   string
	}{
		{
			name:       "export declare function",
			symbol:     "buildUser",
			definition: "export declare function buildUser(id: string): string;\n",
			wantKind:   "function",
		},
		{
			name:       "declare function",
			symbol:     "buildUser",
			definition: "declare function buildUser(id: string): string;\n",
			wantKind:   "function",
		},
		{
			name:       "export declare class",
			symbol:     "UserBuilder",
			definition: "export declare class UserBuilder {}\n",
			wantKind:   "class",
		},
		{
			name:       "declare interface",
			symbol:     "BuildOptions",
			definition: "declare interface BuildOptions { id: string }\n",
			wantKind:   "interface",
		},
		{
			name:       "export declare type",
			symbol:     "BuildOptions",
			definition: "export declare type BuildOptions = { id: string }\n",
			wantKind:   "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/types.d.ts": tt.definition,
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, tt.symbol))

			assertTypeScriptStructuredImpactArtifact(t, artifact, tt.symbol, tt.wantKind)
			if got := artifact.Metadata.Bundle.Definition.File; got != "src/types.d.ts" {
				t.Fatalf("definition file = %q, want src/types.d.ts", got)
			}
			if source := artifact.Metadata.Bundle.Debug.Source; !strings.Contains(source, "declaration") {
				t.Fatalf("Debug.Source = %q, want declaration source", source)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactDeclarationFilePattern(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":     "export interface BuildOptions { id: string }\n",
		"src/types.d.ts":   "export interface BuildOptions { id: string }\n",
		"src/ambient.d.ts": "export interface OtherOptions { id: string }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactFilePatternSearchOptions(dir, "BuildOptions", "**/*.d.ts"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "BuildOptions", "interface")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/types.d.ts" {
		t.Fatalf("definition file = %q, want src/types.d.ts", got)
	}
}
