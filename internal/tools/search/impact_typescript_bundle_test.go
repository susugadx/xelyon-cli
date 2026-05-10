package search

import (
	"strings"
	"testing"
)

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

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

			assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			if !recommendedReadsContainFile(artifact.Metadata.Bundle, tt.testPath) {
				t.Fatalf("expected recommended reads to include %s, got %v", tt.testPath, recommendedReadFiles(artifact.Metadata.Bundle))
			}
		})
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
				callers:     typeScriptGenericRefs("src/app", 8),
				directTests: []genericSymbolRef{{File: "src/build.test.ts", Line: 1, IsTest: true}},
			},
			want: goImpactRiskMedium,
		},
		{
			name: "exported many refs without tests is high",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "export\tfunction buildUser() {}"},
			refs: typeScriptImpactRefs{
				callers: typeScriptGenericRefs("src/app", 8),
			},
			want: goImpactRiskHigh,
		},
		{
			name: "local few refs with nearby tests is low",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}"},
			refs: typeScriptImpactRefs{
				callers:     typeScriptGenericRefs("src/app", 1),
				nearbyTests: []genericSymbolRef{{File: "src/build.spec.ts", Line: 1, IsTest: true}},
			},
			want: goImpactRiskLow,
		},
		{
			name: "export default class is exported",
			def:  genericSymbolDef{Name: "UserBuilder", Kind: "class", Signature: "export default class UserBuilder {}"},
			refs: typeScriptImpactRefs{
				callers: typeScriptGenericRefs("src/app", 4),
			},
			want: goImpactRiskHigh,
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
