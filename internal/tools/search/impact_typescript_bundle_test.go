package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactRelatedTests(t *testing.T) {
	tests := []struct {
		name     string
		testPath string
	}{
		{name: "sibling test", testPath: "src/build.test.ts"},
		{name: "sibling spec", testPath: "src/build.spec.ts"},
		{name: "__tests__ directory", testPath: "src/__tests__/build.test.ts"},
		{name: "__tests__ spec", testPath: "src/__tests__/build.spec.ts"},
		{name: "workspace test", testPath: "tests/build.test.ts"},
		{name: "workspace spec", testPath: "tests/build.spec.ts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/build.ts": "export function buildUser(id: string) { return id }\n",
				"src/app.ts":   "import { buildUser } from './build'\nbuildUser('1')\n",
				tt.testPath:    "import { describe } from 'vitest'\ndescribe('build', () => {})\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

			assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			if !recommendedReadsContainFile(artifact.Metadata.Bundle, tt.testPath) {
				t.Fatalf("expected recommended reads to include %s, got %v", tt.testPath, recommendedReadFiles(artifact.Metadata.Bundle))
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactSkipsIgnoredNearbyTest(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":      "export function buildUser(id: string) { return id }\n",
		"src/app.ts":        "import { buildUser } from './build'\nbuildUser('1')\n",
		"src/build.spec.ts": "\nimport { describe } from 'vitest'\ndescribe('build', () => {})\n",
	})
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - src/build.spec.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	artifact := ExecuteSearchCodeArtifactWithConfig(config.DefaultConfig(), nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.spec.ts") {
		t.Fatalf("ignored nearby test should not be recommended, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
	if strings.Contains(artifact.Rendered, "src/build.spec.ts") {
		t.Fatalf("ignored nearby test should not render, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactHonorsProjectIgnoreDefinitions(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":            "export function buildUser(id: string) { return id }\n",
		"src/app.ts":              "import { buildUser } from './build'\nbuildUser('1')\n",
		"generated/build.ts":      "export function buildUser(id: string) { return id }\n",
		"generated/build.test.ts": "import { buildUser } from './build'\nbuildUser('test')\n",
	})
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	artifact := ExecuteSearchCodeArtifactWithConfig(config.DefaultConfig(), nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if artifact.Metadata.Ambiguous {
		t.Fatalf("ignored duplicate definition should not make impact ambiguous, got:\n%s", artifact.Rendered)
	}
	if strings.Contains(artifact.Rendered, "generated/") {
		t.Fatalf("ignored generated TypeScript files should not render, got:\n%s", artifact.Rendered)
	}
	if recommendedReadsContainFile(artifact.Metadata.Bundle, "generated/build.test.ts") {
		t.Fatalf("ignored generated test should not be recommended, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactRecommendedReadOrder(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":      "export function buildUser(id: string) { return id }\n",
		"src/app.ts":        "import { buildUser } from './build'\nbuildUser?.('1')\n",
		"src/build.test.ts": "import { buildUser } from './build'\nbuildUser('test')\n",
		"src/build.spec.ts": "\nimport { describe } from 'vitest'\ndescribe('build', () => {})\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/build.ts")
	assertRecommendedReadAt(t, reads, 1, "callers", "src/app.ts")
	assertRecommendedReadAt(t, reads, 2, "tests", "src/build.test.ts")
	assertRecommendedReadAt(t, reads, 3, "imports", "src/app.ts")
	assertRecommendedReadAt(t, reads, 4, "tests", "src/build.spec.ts")
	if !strings.Contains(artifact.Rendered, "buildUser?.('1')") {
		t.Fatalf("expected optional chaining caller in rendered output, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactTypeRefsFirstForInterfaces(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/types.ts":      "export interface BuildOptions { id: string }\n",
		"src/app.ts":        "import { BuildOptions } from './types'\nconst options = data as BuildOptions\nconst checked = data satisfies BuildOptions\nconst list: BuildOptions[] = []\nconst array: Array<BuildOptions> = []\nclass Builder implements BuildOptions { id = '1' }\n",
		"src/types.test.ts": "import type { BuildOptions } from './types'\nconst input = {} as BuildOptions\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "BuildOptions"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "BuildOptions", "interface")
	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/types.ts")
	assertRecommendedReadAt(t, reads, 1, "type_refs", "src/app.ts")
	assertRecommendedReadAt(t, reads, 2, "tests", "src/types.test.ts")
	if !strings.Contains(artifact.Rendered, "Type References") {
		t.Fatalf("expected type reference section, got:\n%s", artifact.Rendered)
	}
	for _, want := range []string{"as BuildOptions", "satisfies BuildOptions", "BuildOptions[]", "Array<BuildOptions>", "implements BuildOptions"} {
		if !strings.Contains(artifact.Rendered, want) {
			t.Fatalf("expected rendered TypeScript impact to include %q, got:\n%s", want, artifact.Rendered)
		}
	}
}

func TestBuildTypeScriptImpactBundleFromDisplayAndTotalRefsKeepsSummaryTotals(t *testing.T) {
	def := genericSymbolDef{
		Name:      "buildUser",
		Kind:      "function",
		File:      "src/build.ts",
		Line:      1,
		Signature: "function buildUser(id: string) { return id }",
	}
	displayRefs := []genericSymbolRef{{
		File:    "src/app0.ts",
		Line:    1,
		Snippet: "buildUser('0')",
		Class:   codeast.ClassCall,
	}}
	totalRefs := make([]genericSymbolRef, 0, jsFamilyImpactHighNonTestReferenceThreshold)
	for i := 0; i < jsFamilyImpactHighNonTestReferenceThreshold; i++ {
		totalRefs = append(totalRefs, genericSymbolRef{
			File:  filepath.ToSlash(filepath.Join("src", "app"+string(rune('0'+i))+".ts")),
			Line:  1,
			Class: codeast.ClassCall,
		})
	}

	bundle := buildTypeScriptImpactBundleFromDisplayAndTotalRefs("buildUser", def, SearchOptions{}, displayRefs, totalRefs)

	if bundle == nil || bundle.Impact == nil {
		t.Fatal("bundle impact = nil, want structured impact bundle")
	}
	if got := bundle.Impact.RiskLevel; got != impactplan.RiskHigh {
		t.Fatalf("risk = %q, want %q from total refs summary", got, impactplan.RiskHigh)
	}
	callers := symbolBundleSectionByKind(bundle, "callers")
	if callers == nil {
		t.Fatal("callers section = nil, want budgeted evidence section")
	}
	if len(callers.Items) != 1 {
		t.Fatalf("callers items len = %d, want one display evidence item", len(callers.Items))
	}
	if callers.Total != jsFamilyImpactHighNonTestReferenceThreshold || !callers.More {
		t.Fatalf("callers total/more = %d/%v, want total refs summary %d with More", callers.Total, callers.More, jsFamilyImpactHighNonTestReferenceThreshold)
	}
}

func TestClassifyTypeScriptImpactRisk(t *testing.T) {
	tests := []struct {
		name string
		def  genericSymbolDef
		refs typeScriptImpactRefs
		want string
	}{
		{
			name: "exported with direct tests is not high",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "export  function buildUser() {}"},
			refs: typeScriptImpactRefs{
				callers:     genericSymbolRefsForTest("src/app", ".ts", 8),
				directTests: []genericSymbolRef{{File: "src/build.test.ts", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "exported many refs without tests is high",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "export\tfunction buildUser() {}"},
			refs: typeScriptImpactRefs{
				callers: genericSymbolRefsForTest("src/app", ".ts", 8),
			},
			want: impactplan.RiskHigh,
		},
		{
			name: "local few refs with nearby tests is low",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}"},
			refs: typeScriptImpactRefs{
				callers:     genericSymbolRefsForTest("src/app", ".ts", 1),
				nearbyTests: []genericSymbolRef{{File: "src/build.spec.ts", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskLow,
		},
		{
			name: "local declaration exported later with direct tests is medium",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}"},
			refs: typeScriptImpactRefs{
				imports:     []genericSymbolRef{{File: "src/index.ts", Line: 2, Snippet: "export { buildUser }", Class: jsast.ClassExport}},
				directTests: []genericSymbolRef{{File: "src/build.test.ts", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "ast exported definition with direct tests is medium",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}", Exported: true},
			refs: typeScriptImpactRefs{
				directTests: []genericSymbolRef{{File: "src/build.test.ts", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "export default class is exported",
			def:  genericSymbolDef{Name: "UserBuilder", Kind: "class", Signature: "export default class UserBuilder {}"},
			refs: typeScriptImpactRefs{
				callers: genericSymbolRefsForTest("src/app", ".ts", 4),
			},
			want: impactplan.RiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyTypeScriptImpactRisk(tt.def, tt.refs); got != tt.want {
				t.Fatalf("classifyTypeScriptImpactRisk() = %q, want %q", got, tt.want)
			}
		})
	}
}
