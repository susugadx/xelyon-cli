package search

import (
	"strings"
	"testing"
)

func TestFindJSFamilyReferencesWithAST_CollectsCallerBehindCommentBudget(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": jsFamilyReferenceBudgetSource("export function buildUser(id: string) { return id }\n"),
	})

	refs := findJSFamilyReferencesWithAST("buildUser", SearchOptions{
		Path:               dir,
		FileType:           "ts",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	})
	classified := classifyJSFamilySymbolRefsFromAST(refs)

	if !genericRefsContainSnippet(classified.callers, "buildUser('real caller')") {
		t.Fatalf("callers = %+v, want caller behind comment budget; refs=%+v", classified.callers, refs)
	}
	if genericRefsContainSnippet(classified.others, "comment noise") {
		t.Fatalf("others = %+v, did not want comment refs to consume usable reference budget", classified.others)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JSFamilyStructuredImpactFallbackRefsCollectCallerBehindCommentBudget(t *testing.T) {
	tests := []struct {
		name           string
		file           string
		fileType       string
		definition     string
		assertArtifact func(*testing.T, SearchExecutionArtifact)
	}{
		{
			name:       "typescript",
			file:       "src/build.ts",
			fileType:   "ts",
			definition: "export function buildUser(id: string) { return id }\n",
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
		{
			name:       "javascript",
			file:       "src/build.js",
			fileType:   "js",
			definition: "export function buildUser(id) { return id }\n",
			assertArtifact: func(t *testing.T, artifact SearchExecutionArtifact) {
				t.Helper()
				assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				tt.file: jsFamilyReferenceBudgetSource(tt.definition),
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
				Pattern:            "buildUser",
				Intent:             "impact",
				Path:               dir,
				FileType:           tt.fileType,
				InvocationCWD:      dir,
				ProjectMapRootPath: dir,
			})

			tt.assertArtifact(t, artifact)
			callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
			if !symbolBundleItemsContainSnippet(callers, "buildUser('real caller')") {
				t.Fatalf("callers = %+v, want caller behind comment budget; output:\n%s", callers, artifact.Rendered)
			}
			if symbolBundleItemsContainSnippet(callers, "comment noise") {
				t.Fatalf("callers = %+v, did not want comment noise", callers)
			}
		})
	}
}

func jsFamilyReferenceBudgetSource(definition string) string {
	var source strings.Builder
	source.WriteString(definition)
	for i := 0; i < maxGenericRefs; i++ {
		source.WriteString("// buildUser comment noise\n")
	}
	source.WriteString("buildUser('real caller')\n")
	return source.String()
}

func genericRefsContainSnippet(refs []genericSymbolRef, snippet string) bool {
	for _, ref := range refs {
		if strings.Contains(ref.Snippet, snippet) {
			return true
		}
	}
	return false
}
