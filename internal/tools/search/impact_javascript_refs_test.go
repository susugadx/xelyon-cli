package search

import (
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactRefsAndCommentStringFiltering(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\nmodule.exports = buildUser\n",
		"src/app.js": strings.Join([]string{
			"import { buildUser } from './build.js'",
			"const { buildUser: requiredBuildUser } = require('./build')",
			"buildUser('1')",
			"buildUser?.('2')",
			"const label = `${buildUser('3')}`",
			"const re = /[//]/; buildUser('regex')",
			"const regexOnly = /buildUser/",
			"const rawTemplate = `buildUser()`",
			`const text = "buildUser()"`,
			"// buildUser()",
			"const { buildUser: externalBuildUser } = require('@external/build')",
			"export { buildUser } from './build.js'",
			"export { other as buildUser }",
			"exports.buildUser = buildUser",
			"module.exports.buildUser = buildUser",
			"",
		}, "\n"),
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	for _, want := range []string{"buildUser('1')", "buildUser?.('2')", "`${buildUser('3')}`", "buildUser('regex')"} {
		if !symbolBundleItemsContainSnippet(callers, want) {
			t.Fatalf("callers = %+v, want snippet %q", callers, want)
		}
	}
	for _, notCaller := range []string{"/buildUser/", "`buildUser()`", `"buildUser()"`, "// buildUser()"} {
		if symbolBundleItemsContainSnippet(callers, notCaller) {
			t.Fatalf("callers = %+v, did not want comment/string caller %q", callers, notCaller)
		}
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	for _, want := range []string{"import { buildUser }", "require('./build')", "module.exports = buildUser", "exports.buildUser = buildUser"} {
		if !symbolBundleItemsContainSnippet(imports, want) {
			t.Fatalf("imports = %+v, want snippet %q", imports, want)
		}
	}
	if symbolBundleItemsContainSnippet(imports, "export { other as buildUser }") {
		t.Fatalf("imports = %+v, did not want alias-only export", imports)
	}
	if symbolBundleItemsContainSnippet(imports, "@external/build") {
		t.Fatalf("imports = %+v, did not want external require evidence", imports)
	}
	refs := symbolBundleSectionItems(artifact.Metadata.Bundle, "references")
	if symbolBundleItemsContainSnippet(refs, "export { other as buildUser }") {
		t.Fatalf("refs = %+v, did not want alias-only export", refs)
	}
	if symbolBundleItemsContainSnippet(refs, "/buildUser/") {
		t.Fatalf("refs = %+v, did not want regex literal reference", refs)
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactCommonJSBracketExportEvidence(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": strings.Join([]string{
			"function buildUser(id) { return id }",
			`exports["buildUser"] = buildUser`,
			"",
		}, "\n"),
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, `exports["buildUser"] = buildUser`) {
		t.Fatalf("imports = %+v, want bracket CommonJS export evidence", imports)
	}
	assertRecommendedReadAt(t, artifact.Metadata.Bundle.Impact.RecommendedReads, 1, "imports", "src/build.js")
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactIgnoresCommentedCommonJSDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/commented.js": "/*\n" +
			"exports.buildUser = function() { return 'comment' }\n" +
			"*/\n",
		"src/app.js": "import { buildUser } from './build.js'\nbuildUser('real')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if strings.Contains(artifact.Rendered, "Multiple definitions") {
		t.Fatalf("commented CommonJS export should not make impact ambiguous:\n%s", artifact.Rendered)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "buildUser('real')") {
		t.Fatalf("callers = %+v, want real caller", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if symbolBundleItemsContainFile(imports, "src/commented.js") {
		t.Fatalf("imports = %+v, did not want commented CommonJS definition evidence", imports)
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactIgnoresTemplatedCommonJSDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/template.js": "const fixture = `\n" +
			"exports.buildUser = function() { return 'template' }\n" +
			"`\n",
		"src/app.js": "import { buildUser } from './build.js'\nbuildUser('real')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if strings.Contains(artifact.Rendered, "Multiple definitions") {
		t.Fatalf("templated CommonJS export should not make impact ambiguous:\n%s", artifact.Rendered)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "buildUser('real')") {
		t.Fatalf("callers = %+v, want real caller", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if symbolBundleItemsContainFile(imports, "src/template.js") {
		t.Fatalf("imports = %+v, did not want templated CommonJS definition evidence", imports)
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactRelatedTestsOrder(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js":                "export function buildUser(id) { return id }\n",
		"src/app.js":                  "import { buildUser } from './build.js'\nbuildUser('1')\n",
		"src/build.test.js":           "import { buildUser } from './build.js'\nbuildUser('test')\n",
		"src/build.spec.js":           "describe('build', () => {})\n",
		"src/__tests__/build.test.js": "describe('nested build', () => {})\n",
		"tests/build.test.js":         "describe('workspace build', () => {})\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/build.js")
	assertRecommendedReadAt(t, reads, 1, "callers", "src/app.js")
	assertRecommendedReadAt(t, reads, 2, "tests", "src/build.test.js")
	assertRecommendedReadAt(t, reads, 3, "imports", "src/app.js")
	assertRecommendedReadAt(t, reads, 4, "tests", "src/build.spec.js")

	tests := symbolBundleSectionItems(artifact.Metadata.Bundle, "tests")
	for _, want := range []string{"src/build.test.js", "src/build.spec.js", "src/__tests__/build.test.js", "tests/build.test.js"} {
		if !symbolBundleItemsContainFile(tests, want) {
			t.Fatalf("related test section = %+v, want %s", tests, want)
		}
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactRecommendsReferenceOnlyReads(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/copy.js":  "const copy = buildUser\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	references := symbolBundleSectionItems(artifact.Metadata.Bundle, "references")
	if !symbolBundleItemsContainFile(references, "src/copy.js") {
		t.Fatalf("references = %+v, want reference-only file", references)
	}
	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/build.js")
	assertRecommendedReadAt(t, reads, 1, "references", "src/copy.js")
}
