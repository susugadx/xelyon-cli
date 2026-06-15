package search

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

func newJavaScriptImpactSearchOptions(dir string, symbol string) SearchOptions {
	return SearchOptions{
		Pattern:       symbol,
		Intent:        "impact",
		Path:          dir,
		FileType:      "js",
		InvocationCWD: dir,
	}
}

func newJavaScriptImpactFilePatternSearchOptions(dir string, symbol string, pattern string) SearchOptions {
	opts := newJavaScriptImpactSearchOptions(dir, symbol)
	opts.FileType = ""
	opts.FilePattern = pattern
	return opts
}

func assertJavaScriptStructuredImpactArtifact(t *testing.T, artifact SearchExecutionArtifact, symbol string, kind string) {
	t.Helper()
	assertJSFamilyStructuredImpactArtifact(t, artifact, "javascript", symbol, kind)
}

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

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFiltersPatternsAndPath(t *testing.T) {
	tests := []struct {
		name       string
		defPath    string
		callerPath string
		opts       func(dir string) SearchOptions
	}{
		{
			name:       "js file filter",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactSearchOptions(dir, "buildUser")
			},
		},
		{
			name:       "basename js glob",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "*.js")
			},
		},
		{
			name:       "double star js glob",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "**/*.js")
			},
		},
		{
			name:       "src double star js glob",
			defPath:    "src/build.js",
			callerPath: "src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "src/**/*.js")
			},
		},
		{
			name:       "package source double star js glob",
			defPath:    "packages/app/src/build.js",
			callerPath: "packages/app/src/app.js",
			opts: func(dir string) SearchOptions {
				return newJavaScriptImpactFilePatternSearchOptions(dir, "buildUser", "packages/*/src/**/*.js")
			},
		},
		{
			name:       "direct js path",
			defPath:    "packages/app/src/build.js",
			callerPath: "packages/app/src/app.js",
			opts: func(dir string) SearchOptions {
				return SearchOptions{
					Pattern:       "buildUser",
					Intent:        "impact",
					Path:          filepath.Join(dir, "packages", "app", "src", "build.js"),
					InvocationCWD: dir,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				tt.defPath:                    "export function buildUser(id) { return id }\n",
				tt.callerPath:                 "buildUser('1')\n",
				"packages/app/src/build.ts":   "export function buildUser(id: string) { return id }\n",
				"packages/app/src/view.jsx":   "export function buildUser() { return <></> }\n",
				"packages/other/src/other.js": "export function otherUser() { return 'other' }\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, tt.opts(dir))

			assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			if got := artifact.Metadata.Bundle.Definition.File; got != tt.defPath {
				t.Fatalf("definition file = %q, want %s", got, tt.defPath)
			}
			if strings.Contains(artifact.Rendered, "build.ts") || strings.Contains(artifact.Rendered, "view.jsx") {
				t.Fatalf("JavaScript structured impact should stay on .js scope, got:\n%s", artifact.Rendered)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactUnsupportedFiltersFallback(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		filePath string
		source   string
	}{
		{name: "javascript filter", fileType: "javascript", filePath: "src/build.js", source: "export function buildUser() { return '1' }\n"},
		{name: "mjs filter", fileType: "mjs", filePath: "src/build.mjs", source: "export function buildUser() { return '1' }\n"},
		{name: "cjs filter", fileType: "cjs", filePath: "src/build.cjs", source: "function buildUser() { return '1' }\n"},
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
				t.Fatalf("StructuredImpact = true for unsupported JavaScript structured filter %q", tt.fileType)
			}
			if !strings.Contains(artifact.Rendered, tt.filePath) {
				t.Fatalf("expected fallback output to keep searching %s, got:\n%s", tt.filePath, artifact.Rendered)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactCacheKeepsRecommendedReadsAndScope(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	dir := setupMultiLangDir(t, map[string]string{
		"src/shared.ts":  "export function buildUser() { return 'ts' }\n",
		"src/shared.tsx": "export function buildUser() { return <button /> }\n",
		"src/shared.js":  "export function buildUser() { return 'js' }\n",
		"src/app.ts":     "buildUser()\n",
		"src/App.tsx":    "export function App() { return <buildUser /> }\n",
		"src/app.js":     "buildUser()\n",
	})
	cache := &testSearchCache{data: make(map[string]string)}

	tsFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newTypeScriptImpactSearchOptions(dir, "buildUser"))
	assertTypeScriptStructuredImpactArtifact(t, tsFirst, "buildUser", "function")
	if got := tsFirst.Metadata.Bundle.Definition.File; got != "src/shared.ts" {
		t.Fatalf("ts definition file = %q, want src/shared.ts", got)
	}

	tsxFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newTSXImpactSearchOptions(dir, "buildUser"))
	assertTypeScriptStructuredImpactArtifact(t, tsxFirst, "buildUser", "function")
	if got := tsxFirst.Metadata.Bundle.Definition.File; got != "src/shared.tsx" {
		t.Fatalf("tsx definition file = %q, want src/shared.tsx", got)
	}

	jsFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newJavaScriptImpactSearchOptions(dir, "buildUser"))
	assertJavaScriptStructuredImpactArtifact(t, jsFirst, "buildUser", "function")
	if got := jsFirst.Metadata.Bundle.Definition.File; got != "src/shared.js" {
		t.Fatalf("js definition file = %q, want src/shared.js", got)
	}

	jsSecond := ExecuteSearchCodeArtifactWithConfig(nil, cache, newJavaScriptImpactSearchOptions(dir, "buildUser"))
	assertJavaScriptStructuredImpactArtifact(t, jsSecond, "buildUser", "function")
	if got := jsSecond.Metadata.Bundle.Definition.File; got != "src/shared.js" {
		t.Fatalf("cached js definition file = %q, want src/shared.js", got)
	}
	if len(jsSecond.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("cached JavaScript structured impact lost recommended reads")
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactAmbiguous(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/a.js": "export function buildUser(id) { return id }\n",
		"src/b.js": "export function buildUser(id) { return id }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:  "buildUser",
		Intent:   "impact",
		Path:     dir,
		FileType: "js",
	})

	if !artifact.Metadata.StructuredImpact {
		t.Fatal("StructuredImpact = false, want true for JavaScript ambiguous result")
	}
	if !artifact.Metadata.Ambiguous {
		t.Fatal("Ambiguous = false, want true")
	}
	if artifact.Metadata.Bundle != nil {
		t.Fatalf("Bundle = %+v, want nil for ambiguous JavaScript impact", artifact.Metadata.Bundle)
	}
	if len(artifact.Metadata.AffectedFiles) != 2 {
		t.Fatalf("AffectedFiles = %v, want two JavaScript candidates", artifact.Metadata.AffectedFiles)
	}
	for _, want := range []string{filepath.Join(dir, "src/a.js"), filepath.Join(dir, "src/b.js")} {
		if !slices.Contains(artifact.Metadata.AffectedFiles, want) {
			t.Fatalf("AffectedFiles = %v, want %s", artifact.Metadata.AffectedFiles, want)
		}
	}
	if !strings.Contains(artifact.Rendered, `Multiple definitions found for "buildUser":`) {
		t.Fatalf("expected JavaScript ambiguous output, got:\n%s", artifact.Rendered)
	}
}

func TestClassifyJavaScriptImpactRisk(t *testing.T) {
	tests := []struct {
		name string
		def  genericSymbolDef
		refs javaScriptImpactRefs
		want string
	}{
		{
			name: "exported with direct tests is medium",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "export function buildUser() {}"},
			refs: javaScriptImpactRefs{
				callers:     genericSymbolRefsForTest("src/app", ".js", 8),
				directTests: []genericSymbolRef{{File: "src/build.test.js", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "commonjs exported many refs without tests is high",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}"},
			refs: javaScriptImpactRefs{
				imports: []genericSymbolRef{{File: "src/build.js", Line: 2, Snippet: "module.exports = buildUser", Class: jsast.ClassExport}},
				callers: genericSymbolRefsForTest("src/app", ".js", 4),
			},
			want: impactplan.RiskHigh,
		},
		{
			name: "esm named export with direct tests is medium",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}"},
			refs: javaScriptImpactRefs{
				imports:     []genericSymbolRef{{File: "src/index.js", Line: 2, Snippet: "export { buildUser }", Class: jsast.ClassExport}},
				directTests: []genericSymbolRef{{File: "src/build.test.js", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "ast exported definition with direct tests is medium",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}", Exported: true},
			refs: javaScriptImpactRefs{
				directTests: []genericSymbolRef{{File: "src/build.test.js", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "local few refs with nearby tests is low",
			def:  genericSymbolDef{Name: "buildUser", Kind: "function", Signature: "function buildUser() {}"},
			refs: javaScriptImpactRefs{
				callers:     genericSymbolRefsForTest("src/app", ".js", 1),
				nearbyTests: []genericSymbolRef{{File: "src/build.spec.js", Line: 1, IsTest: true}},
			},
			want: impactplan.RiskLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyJavaScriptImpactRisk(tt.def, tt.refs); got != tt.want {
				t.Fatalf("classifyJavaScriptImpactRisk() = %q, want %q", got, tt.want)
			}
		})
	}
}
