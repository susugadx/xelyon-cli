package search

import "testing"

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsDefaultImportAliasWithoutSymbolText(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js":      "export default function buildUser(id) { return id }\n",
		"src/app.js":        "import makeUser from './build.js'\nexport const user = makeUser('1')\n",
		"src/build.test.js": "import makeUser from './build.js'\nit('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want default import alias call without symbol text", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS default alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsNamedDefaultImportAlias(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export default function buildUser(id) { return id }\n",
		"src/app.js": "import { default as makeUser } from './build.js'\n" +
			"export const user = makeUser('1')\n",
		"src/build.spec.js": "import { default as makeUser } from './build.js'\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want named default import alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "default as makeUser") {
		t.Fatalf("imports = %+v, want named default import alias evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.spec.js") {
		t.Fatalf("expected direct JS named default alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js": "const { buildUser: makeUser } = require('./build')\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const { buildUser: makeUser } = require('./build')\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS require alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "buildUser: makeUser") {
		t.Fatalf("imports = %+v, want CommonJS require alias evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSDefaultRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "module.exports = function buildUser(id) { return id }\n",
		"src/app.js": "const makeUser = require('./build')\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const makeUser = require('./build')\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS default require alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "const makeUser = require") {
		t.Fatalf("imports = %+v, want CommonJS default require evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS default require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSSeparateDefaultRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\nmodule.exports = buildUser\n",
		"src/app.js": "const makeUser = require('./build')\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const makeUser = require('./build')\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS separate default require alias call", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS separate default require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSMemberRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js": "const makeUser = require('./build').buildUser\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const makeUser = require('./build').buildUser\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS member require alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "require('./build').buildUser") {
		t.Fatalf("imports = %+v, want CommonJS member require evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS member require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsMultilineNamedDefaultImportAlias(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export default function buildUser(id) { return id }\n",
		"src/app.js": "import {\n" +
			"  default as makeUser,\n" +
			"} from './build.js'\n" +
			"export const user = makeUser('1')\n",
		"src/build.spec.js": "import {\n" +
			"  default as makeUser,\n" +
			"} from './build.js'\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want multiline named default import alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "import {") {
		t.Fatalf("imports = %+v, want multiline named default import evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.spec.js") {
		t.Fatalf("expected direct JS multiline named default alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactSkipsNestedRequireDestructuringAlias(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js": "const { api: { buildUser: makeUser } } = require('./build')\n" +
			"export const user = makeUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if symbolBundleItemsContainFile(callers, "src/app.js") || symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, did not want nested CommonJS destructuring alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if symbolBundleItemsContainFile(imports, "src/app.js") || symbolBundleItemsContainSnippet(imports, "api:") {
		t.Fatalf("imports = %+v, did not want nested CommonJS destructuring evidence", imports)
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsSeparateDefaultExportAlias(t *testing.T) {
	tests := []struct {
		name   string
		export string
	}{
		{name: "export specifier default alias", export: "export { buildUser as default }"},
		{name: "export default identifier", export: "export default buildUser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/build.js": "function buildUser(id) { return id }\n" + tt.export + "\n",
				"src/app.js":   "import makeUser from './build.js'\nexport const user = makeUser('1')\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

			assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
			if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
				t.Fatalf("callers = %+v, want default import alias call for %s", callers, tt.export)
			}
			imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
			if !symbolBundleItemsContainSnippet(imports, "import makeUser") {
				t.Fatalf("imports = %+v, want default import evidence for %s", imports, tt.export)
			}
		})
	}
}
