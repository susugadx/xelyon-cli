package search

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactNameOnlyRefsStayNearDirectPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.js":        "export function buildUser(id) { return id }\n",
		"packages/app/src/app.js":          "buildUser('local fallback')\n",
		"packages/app/src/build.test.js":   "buildUser('local test')\n",
		"packages/other/src/app.js":        "buildUser('sibling false positive')\n",
		"packages/other/src/build.test.js": "buildUser('sibling test false positive')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:       "buildUser",
		Intent:        "impact",
		Path:          filepath.Join(dir, "packages", "app", "src", "build.js"),
		InvocationCWD: dir,
	})

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "packages/app/src/build.js" {
		t.Fatalf("definition file = %q, want scoped direct definition", got)
	}
	assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/app/src/app.js")
	assertJSFamilyImpactSectionContainsFile(t, artifact, "tests", "packages/app/src/build.test.js")
	assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "packages/other/src/app.js", "packages/other/src/build.test.js")
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactPreservesScopedFilePatternForWorkspaceRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.js":      "export function buildUser(id) { return id }\n",
		"packages/app/src/app.js":        "buildUser('in scope')\n",
		"packages/app/src/build.test.js": "buildUser('test')\n",
		"packages/other/test/out.js":     "buildUser('out of scoped glob')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "packages/app/src/**/*.js"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/app/src/app.js")
	assertJSFamilyImpactSectionContainsFile(t, artifact, "tests", "packages/app/src/build.test.js")
	assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "packages/other/test/out.js")
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactNameOnlyRefsStayNearDirectPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.ts":        "export function buildUser(id: string) { return id }\n",
		"packages/app/src/app.ts":          "buildUser('local fallback')\n",
		"packages/app/src/build.test.ts":   "buildUser('local test')\n",
		"packages/other/src/app.ts":        "buildUser('sibling false positive')\n",
		"packages/other/src/build.test.ts": "buildUser('sibling test false positive')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:       "buildUser",
		Intent:        "impact",
		Path:          filepath.Join(dir, "packages", "app", "src", "build.ts"),
		InvocationCWD: dir,
	})

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "packages/app/src/build.ts" {
		t.Fatalf("definition file = %q, want scoped direct definition", got)
	}
	assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/app/src/app.ts")
	assertJSFamilyImpactSectionContainsFile(t, artifact, "tests", "packages/app/src/build.test.ts")
	assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "packages/other/src/app.ts", "packages/other/src/build.test.ts")
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactPreservesScopedFilePatternForWorkspaceRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.ts":      "export function buildUser(id: string) { return id }\n",
		"packages/app/src/app.ts":        "buildUser('in scope')\n",
		"packages/app/src/build.test.ts": "buildUser('test')\n",
		"packages/other/test/out.ts":     "buildUser('out of scoped glob')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactFilePatternSearchOptions(dir, "buildUser", "packages/app/src/**/*.ts"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	assertJSFamilyImpactSectionContainsFile(t, artifact, "callers", "packages/app/src/app.ts")
	assertJSFamilyImpactSectionContainsFile(t, artifact, "tests", "packages/app/src/build.test.ts")
	assertJSFamilyImpactRenderedExcludesFiles(t, artifact, "packages/other/test/out.ts")
}

func assertJSFamilyImpactRenderedExcludesFiles(t *testing.T, artifact SearchExecutionArtifact, files ...string) {
	t.Helper()
	for _, file := range files {
		if strings.Contains(artifact.Rendered, file) {
			t.Fatalf("impact output should exclude %s, got:\n%s", file, artifact.Rendered)
		}
	}
}
