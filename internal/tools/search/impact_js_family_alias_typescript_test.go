package search

import "testing"

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactFindsNamedImportAliasCallUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id) { return id }\n",
		"src/app.ts":   "import { buildUser as createUser } from './build'\nexport const user = createUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.ts") || !symbolBundleItemsContainSnippet(callers, "createUser('1')") {
		t.Fatalf("callers = %+v, want named import alias call usage", callers)
	}
}
func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactFindsTypeOnlyImportAliasTypeUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/types.ts": "export interface BuildOptions { id: string }\n",
		"src/app.ts": "import type { BuildOptions as Options } from './types'\n" +
			"export type AppOptions = Options\n" +
			"export const input = {} as Options\n",
		"src/types.test.ts": "import type { BuildOptions as Options } from './types'\n" +
			"const input = {} as Options\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "BuildOptions"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "BuildOptions", "interface")
	typeRefs := symbolBundleSectionItems(artifact.Metadata.Bundle, "type_refs")
	if !symbolBundleItemsContainFile(typeRefs, "src/app.ts") || !symbolBundleItemsContainSnippet(typeRefs, "AppOptions = Options") {
		t.Fatalf("typeRefs = %+v, want type-only import alias type usage", typeRefs)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/types.test.ts") {
		t.Fatalf("expected type-only alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
