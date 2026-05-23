package search

import "testing"

func TestStructuredJavaScriptImpactDefinitionsFromCandidatesAppendsCommonJSInlineMatches(t *testing.T) {
	candidates := jsFamilyDefinitionCandidates{
		astDefs: []genericSymbolDef{{
			Name:      "buildUser",
			Kind:      "function",
			File:      "src/build.js",
			Line:      1,
			Signature: "function buildUser() { return 'ok' }",
		}},
		matches: []genericSymbolMatch{{
			File:    "src/exports.js",
			Line:    3,
			Content: `exports["buildUser"] = function() { return "ok" }`,
		}},
	}

	defs := structuredJavaScriptImpactDefinitionsFromCandidates("buildUser", SearchOptions{FileType: "js"}, candidates)

	if len(defs) != 2 {
		t.Fatalf("defs len = %d, want 2: %+v", len(defs), defs)
	}
	assertGenericDefsContain(t, defs, "src/build.js", "function")
	assertGenericDefsContain(t, defs, "src/exports.js", "function")
}

func TestStructuredJavaScriptImpactDefinitionsFromCandidatesIgnoresCommentedCommonJSInlineMatches(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/commented.js": "/*\nexports.buildUser = function() { return 'comment' }\n*/\n",
	})
	candidates := jsFamilyDefinitionCandidates{
		astDefs: []genericSymbolDef{{
			Name:      "buildUser",
			Kind:      "function",
			File:      "src/build.js",
			Line:      1,
			Signature: "export function buildUser() { return 'ok' }",
		}},
		matches: []genericSymbolMatch{{
			File:    "src/commented.js",
			Line:    2,
			Content: "exports.buildUser = function() { return 'comment' }",
		}},
	}

	defs := structuredJavaScriptImpactDefinitionsFromCandidates("buildUser", SearchOptions{
		Path:               dir,
		FileType:           "js",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}, candidates)

	if len(defs) != 1 {
		t.Fatalf("defs len = %d, want only real AST definition: %+v", len(defs), defs)
	}
	assertGenericDefsContain(t, defs, "src/build.js", "function")
}

func TestStructuredTypeScriptImpactDefinitionSetFromCandidatesUsesGenericFallbackMatches(t *testing.T) {
	candidates := jsFamilyDefinitionCandidates{
		matches: []genericSymbolMatch{{
			File:    "src/types.d.ts",
			Line:    1,
			Content: "export interface BuildOptions { id: string }",
		}},
	}

	definitionSet := structuredTypeScriptImpactDefinitionSetFromCandidates("BuildOptions", SearchOptions{FileType: "ts"}, candidates)

	if len(definitionSet.defs) != 1 {
		t.Fatalf("defs len = %d, want 1: %+v", len(definitionSet.defs), definitionSet.defs)
	}
	assertGenericDefsContain(t, definitionSet.defs, "src/types.d.ts", "interface")
}

func assertGenericDefsContain(t *testing.T, defs []genericSymbolDef, file string, kind string) {
	t.Helper()
	for _, def := range defs {
		if def.File == file && def.Kind == kind {
			return
		}
	}
	t.Fatalf("defs = %+v, want file %q kind %q", defs, file, kind)
}
