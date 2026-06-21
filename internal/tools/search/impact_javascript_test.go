package search

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactShapes(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		definition string
		wantKind   string
	}{
		{
			name:       "function",
			symbol:     "buildUser",
			definition: "function buildUser(id) { return id }\n",
			wantKind:   "function",
		},
		{
			name:       "async function",
			symbol:     "buildUser",
			definition: "async function buildUser(id) { return id }\n",
			wantKind:   "function",
		},
		{
			name:       "export function",
			symbol:     "buildUser",
			definition: "export function buildUser(id) { return id }\n",
			wantKind:   "function",
		},
		{
			name:       "export default function",
			symbol:     "buildOrg",
			definition: "export default function buildOrg() { return 'org' }\n",
			wantKind:   "function",
		},
		{
			name:       "class",
			symbol:     "UserBuilder",
			definition: "class UserBuilder {}\n",
			wantKind:   "class",
		},
		{
			name:       "export default class",
			symbol:     "UserBuilder",
			definition: "export default class UserBuilder {}\n",
			wantKind:   "class",
		},
		{
			name:       "const arrow",
			symbol:     "buildUser",
			definition: "const buildUser = (id) => id\n",
			wantKind:   "function",
		},
		{
			name:       "single-param arrow",
			symbol:     "buildUser",
			definition: "const buildUser = id => id\n",
			wantKind:   "function",
		},
		{
			name:       "async arrow",
			symbol:     "buildUser",
			definition: "const buildUser = async (id) => id\n",
			wantKind:   "function",
		},
		{
			name:       "function expression",
			symbol:     "buildUser",
			definition: "const buildUser = function buildUser(id) { return id }\n",
			wantKind:   "function",
		},
		{
			name:       "module exports inline function",
			symbol:     "buildUser",
			definition: "module.exports = function buildUser(id) { return id }\n",
			wantKind:   "function",
		},
		{
			name:       "named exports inline function",
			symbol:     "buildUser",
			definition: "exports.buildUser = function(id) { return id }\n",
			wantKind:   "function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/build.js": tt.definition,
				"src/app.js":   "buildUser?.('1')\nnew UserBuilder()\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, tt.symbol))

			assertJavaScriptStructuredImpactArtifact(t, artifact, tt.symbol, tt.wantKind)
			if got := artifact.Metadata.Bundle.Impact.RecommendedReads[0].Kind; got != "definition" {
				t.Fatalf("first recommended read kind = %q, want definition", got)
			}
			if got := artifact.Metadata.Bundle.Impact.RecommendedReads[0].File; got != "src/build.js" {
				t.Fatalf("definition recommended file = %q, want src/build.js", got)
			}
		})
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactIncludesFallbackExpansion(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js":      "export function buildUser(id) { return id }\n",
		"src/build_impl.js": "export function buildUserImpl(id) { return buildUser(id) }\n",
		"src/build.test.js": "function TestBuildUser() { expect(1).toBe(1) }\n",
	})

	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if !artifact.Metadata.MultiPattern {
		t.Fatal("MultiPattern = false, want true when JavaScript structured impact keeps impact expansion")
	}
	for _, want := range []string{"buildUserImpl", "TestBuildUser", "src/build_impl.js", "src/build.test.js"} {
		if !strings.Contains(artifact.Rendered, want) {
			t.Fatalf("expected structured JavaScript impact expansion output to contain %q, got:\n%s", want, artifact.Rendered)
		}
	}
	for _, want := range []string{filepath.Join(dir, "src/build_impl.js"), filepath.Join(dir, "src/build.test.js")} {
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
func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactClassCaller(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export class UserBuilder {}\n",
		"src/app.js":   "const builder = new UserBuilder()\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "UserBuilder"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "UserBuilder", "class")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "new UserBuilder()") {
		t.Fatalf("callers = %+v, want new UserBuilder()", callers)
	}
}
