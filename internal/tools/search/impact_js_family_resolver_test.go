package search

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestResolveStructuredJSFamilyImpactSymbol_MultipleDefinitionsUsesSharedDefinitionFlow(t *testing.T) {
	definitionOpts := SearchOptions{Path: "definitions"}
	evidenceOpts := SearchOptions{Path: "evidence"}
	defs := []genericSymbolDef{
		{Name: "buildUser", Kind: "function", File: "src/build.ts", Line: 1},
		{Name: "buildUser", Kind: "function", File: "src/other.ts", Line: 4},
	}
	affectedFiles := []string{"/workspace/src/build.ts", "/workspace/src/other.ts"}

	collectCalled := false
	refCollectorCalled := false
	result := resolveStructuredJSFamilyImpactSymbol("buildUser", structuredImpactScope{
		Definition: definitionOpts,
		Evidence:   evidenceOpts,
	}, jsFamilyImpactResolverSpec{
		findDefinitions: func(symbol string, opts SearchOptions) jsFamilyImpactDefinitionSet {
			if symbol != "buildUser" {
				t.Fatalf("symbol = %q, want buildUser", symbol)
			}
			if opts.Path != definitionOpts.Path {
				t.Fatalf("definition opts path = %q, want %q", opts.Path, definitionOpts.Path)
			}
			return jsFamilyImpactDefinitionSet{defs: defs}
		},
		collectDefAffectedFiles: func(gotDefs []genericSymbolDef, opts SearchOptions) []string {
			collectCalled = true
			if !slices.Equal(gotDefs, defs) {
				t.Fatalf("defs = %+v, want %+v", gotDefs, defs)
			}
			if opts.Path != definitionOpts.Path {
				t.Fatalf("affected opts path = %q, want %q", opts.Path, definitionOpts.Path)
			}
			return affectedFiles
		},
		referenceOptions: func(genericSymbolDef, SearchOptions) jsFamilyReferenceOptions {
			refCollectorCalled = true
			return jsFamilyReferenceOptions{}
		},
		normalizeRefs: func(refs []genericSymbolRef) []genericSymbolRef {
			t.Fatalf("normalizeRefs called for multiple definitions: %+v", refs)
			return nil
		},
		buildBundle: func(string, genericSymbolDef, SearchOptions, []genericSymbolRef, []genericSymbolRef) *SymbolBundle {
			t.Fatal("buildBundle called for multiple definitions")
			return nil
		},
	})

	if result.Status != symbolResolveMultiple {
		t.Fatalf("status = %s, want %s", result.Status, symbolResolveMultiple)
	}
	if !collectCalled {
		t.Fatal("collectDefAffectedFiles was not called")
	}
	if refCollectorCalled {
		t.Fatal("referenceOptions called for multiple definitions")
	}
	if !slices.Equal(result.AffectedFiles, affectedFiles) {
		t.Fatalf("AffectedFiles = %+v, want %+v", result.AffectedFiles, affectedFiles)
	}
	if !strings.Contains(result.Output, `Multiple definitions found for "buildUser"`) {
		t.Fatalf("output = %q, want multiple definition output", result.Output)
	}
}

func TestResolveStructuredJSFamilyImpactSymbol_IncompleteSingleDefinitionDefersToFallback(t *testing.T) {
	def := genericSymbolDef{Name: "buildUser", Kind: "function", File: "src/build.ts", Line: 1}
	result := resolveStructuredJSFamilyImpactSymbol("buildUser", structuredImpactScope{
		Definition: SearchOptions{Path: "definitions"},
		Evidence:   SearchOptions{Path: "evidence"},
	}, jsFamilyImpactResolverSpec{
		findDefinitions: func(string, SearchOptions) jsFamilyImpactDefinitionSet {
			return jsFamilyImpactDefinitionSet{defs: []genericSymbolDef{def}, definitionIncomplete: true}
		},
		collectDefAffectedFiles: func([]genericSymbolDef, SearchOptions) []string {
			t.Fatal("collectDefAffectedFiles called for incomplete single definition")
			return nil
		},
		referenceOptions: func(genericSymbolDef, SearchOptions) jsFamilyReferenceOptions {
			t.Fatal("referenceOptions called for incomplete single definition")
			return jsFamilyReferenceOptions{}
		},
		normalizeRefs: func(refs []genericSymbolRef) []genericSymbolRef {
			t.Fatalf("normalizeRefs called for incomplete single definition: %+v", refs)
			return nil
		},
		buildBundle: func(string, genericSymbolDef, SearchOptions, []genericSymbolRef, []genericSymbolRef) *SymbolBundle {
			t.Fatal("buildBundle called for incomplete single definition")
			return nil
		},
	})

	if result.Status != symbolResolveNone {
		t.Fatalf("status = %s, want %s", result.Status, symbolResolveNone)
	}
}

func TestResolveStructuredJSFamilyImpactSymbol_IncompleteMultipleDefinitionsDefersToFallback(t *testing.T) {
	defs := []genericSymbolDef{
		{Name: "buildUser", Kind: "function", File: "src/build.ts", Line: 1},
		{Name: "buildUser", Kind: "function", File: "src/other.ts", Line: 2},
	}
	result := resolveStructuredJSFamilyImpactSymbol("buildUser", structuredImpactScope{
		Definition: SearchOptions{Path: "definitions"},
		Evidence:   SearchOptions{Path: "evidence"},
	}, jsFamilyImpactResolverSpec{
		findDefinitions: func(string, SearchOptions) jsFamilyImpactDefinitionSet {
			return jsFamilyImpactDefinitionSet{defs: defs, definitionIncomplete: true}
		},
		collectDefAffectedFiles: func([]genericSymbolDef, SearchOptions) []string {
			t.Fatal("collectDefAffectedFiles called for incomplete multiple definitions")
			return nil
		},
		referenceOptions: func(genericSymbolDef, SearchOptions) jsFamilyReferenceOptions {
			t.Fatal("referenceOptions called for incomplete multiple definitions")
			return jsFamilyReferenceOptions{}
		},
		normalizeRefs: func(refs []genericSymbolRef) []genericSymbolRef {
			t.Fatalf("normalizeRefs called for incomplete multiple definitions: %+v", refs)
			return nil
		},
		buildBundle: func(string, genericSymbolDef, SearchOptions, []genericSymbolRef, []genericSymbolRef) *SymbolBundle {
			t.Fatal("buildBundle called for incomplete multiple definitions")
			return nil
		},
	})

	if result.Status != symbolResolveNone {
		t.Fatalf("status = %s, want %s", result.Status, symbolResolveNone)
	}
}

func TestResolveStructuredJSFamilyImpactSymbol_SingleDefinitionUsesSharedReferenceFlow(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}

	dir := t.TempDir()
	writeJSFamilyResolverTestFile(t, dir, "src/build.ts", "export function buildUser(id: string) { return id }\n")
	writeJSFamilyResolverTestFile(t, dir, "src/app.ts", "import { buildUser } from './build'\nexport const user = buildUser('1')\n")

	definitionOpts := SearchOptions{
		Path:               filepath.Join(dir, "src", "build.ts"),
		FileType:           "ts",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}
	evidenceOpts := SearchOptions{
		Path:               dir,
		FileType:           "ts",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}
	def := genericSymbolDef{
		Name:      "buildUser",
		Kind:      "function",
		File:      "./src/build.ts",
		Line:      1,
		Signature: "export function buildUser(id: string) { return id }",
	}

	normalizeCalled := false
	filterCalled := false
	var builtRefs []genericSymbolRef
	result := resolveStructuredJSFamilyImpactSymbol("buildUser", structuredImpactScope{
		Definition: definitionOpts,
		Evidence:   evidenceOpts,
	}, jsFamilyImpactResolverSpec{
		findDefinitions: func(symbol string, opts SearchOptions) jsFamilyImpactDefinitionSet {
			if opts.Path != definitionOpts.Path {
				t.Fatalf("definition opts path = %q, want %q", opts.Path, definitionOpts.Path)
			}
			return jsFamilyImpactDefinitionSet{defs: []genericSymbolDef{def}}
		},
		collectDefAffectedFiles: func([]genericSymbolDef, SearchOptions) []string {
			t.Fatal("collectDefAffectedFiles called for single definition")
			return nil
		},
		referenceOptions: func(gotDef genericSymbolDef, opts SearchOptions) jsFamilyReferenceOptions {
			if gotDef != def {
				t.Fatalf("reference def = %+v, want %+v", gotDef, def)
			}
			if opts.Path != evidenceOpts.Path {
				t.Fatalf("evidence opts path = %q, want %q", opts.Path, evidenceOpts.Path)
			}
			return newJSFamilyStructuredImpactReferenceOptions(gotDef, opts, "ts")
		},
		normalizeRefs: func(refs []genericSymbolRef) []genericSymbolRef {
			normalizeCalled = true
			return refs
		},
		filterRefs: func(gotDef genericSymbolDef, defs jsFamilyImpactDefinitionSet, refs []genericSymbolRef) []genericSymbolRef {
			filterCalled = true
			if gotDef != def {
				t.Fatalf("filter def = %+v, want %+v", gotDef, def)
			}
			if !slices.Equal(defs.defs, []genericSymbolDef{def}) {
				t.Fatalf("filter defs = %+v, want %+v", defs.defs, []genericSymbolDef{def})
			}
			return refs
		},
		buildBundle: func(symbol string, gotDef genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef) *SymbolBundle {
			if opts.Path != evidenceOpts.Path {
				t.Fatalf("bundle opts path = %q, want %q", opts.Path, evidenceOpts.Path)
			}
			if !slices.Equal(refs, totalRefs) {
				t.Fatalf("total refs = %+v, want same refs for non-LSP fallback %+v", totalRefs, refs)
			}
			builtRefs = append([]genericSymbolRef(nil), refs...)
			return newJSFamilyResolverTestBundle(symbol, gotDef, refs)
		},
	})

	if result.Status != symbolResolveSingle {
		t.Fatalf("status = %s, want %s; output:\n%s", result.Status, symbolResolveSingle, result.Output)
	}
	if result.Bundle == nil {
		t.Fatal("bundle = nil, want structured bundle")
	}
	if !normalizeCalled {
		t.Fatal("normalizeRefs was not called")
	}
	if !filterCalled {
		t.Fatal("filterRefs was not called")
	}
	if !genericRefsContainCleanFile(builtRefs, "src/app.ts") {
		t.Fatalf("refs = %+v, want src/app.ts caller", builtRefs)
	}
	if genericRefsContainCleanFileLine(builtRefs, "src/build.ts", 1) {
		t.Fatalf("refs = %+v, did not want definition line after shared filter", builtRefs)
	}
	if !strings.Contains(result.Output, "Recommended reads:") {
		t.Fatalf("output = %q, want formatted bundle output", result.Output)
	}
}

func newJSFamilyResolverTestBundle(symbol string, def genericSymbolDef, refs []genericSymbolRef) *SymbolBundle {
	recommended := []SymbolBundleItem{{
		Kind:    "definition",
		File:    def.File,
		Line:    def.Line,
		Snippet: def.Signature,
	}}
	callers := make([]SymbolBundleItem, 0, len(refs))
	for _, ref := range refs {
		item := SymbolBundleItem{
			Kind:    "callers",
			File:    ref.File,
			Line:    ref.Line,
			Snippet: ref.Snippet,
		}
		callers = append(callers, item)
		if filepath.ToSlash(filepath.Clean(ref.File)) == "src/app.ts" {
			recommended = append(recommended, item)
		}
	}
	return &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "typescript",
			Query:       symbol,
			Canonical:   symbol,
			DisplayName: symbol,
			Kind:        def.Kind,
			File:        def.File,
			Line:        def.Line,
		},
		Definition: SymbolBundleDefinition{
			File:      def.File,
			Line:      def.Line,
			Signature: def.Signature,
		},
		Sections: []SymbolBundleSection{{
			Kind:  "callers",
			Title: "Callers",
			Items: callers,
			Total: len(callers),
		}},
		Impact: &SymbolBundleImpact{
			RiskLevel:        impactplan.RiskLow,
			RecommendedReads: recommended,
		},
	}
}

func genericRefsContainCleanFile(refs []genericSymbolRef, file string) bool {
	return slices.ContainsFunc(refs, func(ref genericSymbolRef) bool {
		return filepath.ToSlash(filepath.Clean(ref.File)) == file
	})
}

func genericRefsContainCleanFileLine(refs []genericSymbolRef, file string, line int) bool {
	return slices.ContainsFunc(refs, func(ref genericSymbolRef) bool {
		return filepath.ToSlash(filepath.Clean(ref.File)) == file && ref.Line == line
	})
}

func writeJSFamilyResolverTestFile(t *testing.T, root string, path string, content string) {
	t.Helper()
	absPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
