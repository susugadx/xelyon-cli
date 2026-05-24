package search

import (
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/impactplan"
)

func TestSemanticEvidenceFromGoInspectResult(t *testing.T) {
	result := semanticEvidenceGoBuildInspectFixture()

	evidence, ok := semanticEvidenceFromGoInspectResult("Build", result)
	if !ok {
		t.Fatal("semanticEvidenceFromGoInspectResult() ok = false")
	}

	if evidence.Language != "go" || evidence.Symbol != "Build" {
		t.Fatalf("evidence identity = (%q, %q), want go Build", evidence.Language, evidence.Symbol)
	}
	if len(evidence.Definitions) != 1 {
		t.Fatalf("definitions len = %d, want 1", len(evidence.Definitions))
	}
	def := evidence.Definitions[0]
	if def.File != "pkg/build.go" || def.Line != 3 || def.RootPath != "/repo" || len(def.Body) != 3 {
		t.Fatalf("definition = %+v, want Go candidate details", def)
	}
	assertSemanticDefinitionAttributes(t, def, true, true, false)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindCaller, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindReference, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindTest, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindImplementation, 1)
	if evidence.Diagnostics == nil {
		t.Fatal("Diagnostics = nil, want Go diagnostics")
	}
	assertDiagnosticsResolvedBy(t, *evidence.Diagnostics, SymbolBundleResolvedByLSP)
	assertIntPtr(t, evidence.Diagnostics.RawRefCount, 4, "RawRefCount")
	assertIntPtr(t, evidence.Diagnostics.AcceptedRefCount, 3, "AcceptedRefCount")

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence(go evidence) ok = false")
	}
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 0, "definition", "pkg/build.go")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 1, "callers", "pkg/app.go")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 2, "tests", "pkg/build_test.go")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 3, "implementations", "pkg/impl.go")
}

func TestSemanticEvidenceFromGoInspectResultPreservesStableIdentityAndSectionTotals(t *testing.T) {
	stableKey := stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error")
	result := semanticEvidenceGoRunInspectFixture(stableKey)

	evidence, ok := semanticEvidenceFromGoInspectResult("Run", result)
	if !ok {
		t.Fatal("semanticEvidenceFromGoInspectResult() ok = false")
	}
	def := evidence.Definitions[0]
	if def.DisplayName != "(*Agent).Run" {
		t.Fatalf("definition DisplayName = %q, want (*Agent).Run", def.DisplayName)
	}
	if def.Canonical != stableKey {
		t.Fatalf("definition Canonical = %q, want stable Go key %q", def.Canonical, stableKey)
	}

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence(go evidence) ok = false")
	}
	if bundle.Identity.DisplayName != "(*Agent).Run" {
		t.Fatalf("bundle display name = %q, want (*Agent).Run", bundle.Identity.DisplayName)
	}
	if bundle.Identity.Canonical != stableKey {
		t.Fatalf("bundle canonical = %q, want stable Go key %q", bundle.Identity.Canonical, stableKey)
	}
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindCallers, 4, true)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindReferences, 3, true)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindTests, 2, true)
	output := formatSymbolBundle(bundle, nil, nil)
	assertOutputContains(t, output, "Callers: 1 shown (of 4)")
	assertOutputContains(t, output, "Omitted: callers +3, references +2, tests +1")

	moved := result
	movedSymbol := *result.Symbol
	movedSymbol.Line = 40
	movedSymbol.EndLine = 42
	moved.Symbol = &movedSymbol
	movedEvidence, ok := semanticEvidenceFromGoInspectResult("Run", moved)
	if !ok {
		t.Fatal("semanticEvidenceFromGoInspectResult(moved) ok = false")
	}
	movedBundle, ok := buildSymbolBundleFromSemanticEvidence(movedEvidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence(moved go evidence) ok = false")
	}
	if movedBundle.Identity.Canonical != bundle.Identity.Canonical {
		t.Fatalf("canonical changed across line move: %q vs %q", movedBundle.Identity.Canonical, bundle.Identity.Canonical)
	}
}

func TestSemanticEvidenceFromGoInspectResultBudgetsImplementations(t *testing.T) {
	total := goImplementationLimit + 2
	result := semanticEvidenceGoImplementationBudgetFixture(total)

	evidence, ok := semanticEvidenceFromGoInspectResult("Build", result)
	if !ok {
		t.Fatal("semanticEvidenceFromGoInspectResult() ok = false")
	}
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindImplementation, goImplementationLimit)

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence(go evidence) ok = false")
	}
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindImplementations, goImplementationLimit)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindImplementations, total, true)
	output := formatSymbolBundle(bundle, nil, nil)
	assertOutputContains(t, output, "Related Implementations: 4 shown (of 6)")
	assertOutputContains(t, output, "Omitted: implementations +2")
}

func TestSemanticEvidenceFromJSFamilyRefs(t *testing.T) {
	diagnostics := semanticEvidenceASTDiagnosticsFixture()
	def, refs := semanticEvidenceJSButtonFixture()

	evidence, ok := semanticEvidenceFromJSFamilyRefs("typescript", "Button", def, refs, diagnostics)
	if !ok {
		t.Fatal("semanticEvidenceFromJSFamilyRefs() ok = false")
	}

	if evidence.Language != "typescript" || evidence.Symbol != "Button" {
		t.Fatalf("evidence identity = (%q, %q), want typescript Button", evidence.Language, evidence.Symbol)
	}
	if len(evidence.Definitions) != 1 || evidence.Definitions[0].File != "src/Button.tsx" {
		t.Fatalf("definitions = %+v, want Button definition", evidence.Definitions)
	}
	assertSemanticDefinitionAttributes(t, evidence.Definitions[0], true, true, false)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindCaller, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindExport, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindTypeRef, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindImport, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindReference, 1)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindTest, 1)
	assertSemanticReferenceKindCount(t, evidence.References, string(codeast.ClassComment), 0)
	if evidence.Diagnostics == nil {
		t.Fatal("Diagnostics = nil, want JS diagnostics")
	}
	assertDiagnosticsResolvedBy(t, *evidence.Diagnostics, SymbolBundleResolvedByAST)

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence(js evidence) ok = false")
	}
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 0, "definition", "src/Button.tsx")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 1, "callers", "src/App.tsx")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 2, "tests", "src/Button.test.tsx")
	if imports := symbolBundleSectionByKind(bundle, SemanticReferenceSectionKindImports); imports == nil || len(imports.Items) != 2 {
		t.Fatalf("imports section = %+v, want import and export refs", imports)
	}
	if typeRefs := symbolBundleSectionByKind(bundle, SemanticReferenceSectionKindTypeRefs); typeRefs == nil || len(typeRefs.Items) != 1 {
		t.Fatalf("type_refs section = %+v, want one type ref", typeRefs)
	}
}

func TestSemanticEvidenceFromJSFamilyRefsPreservesReferenceBudgets(t *testing.T) {
	diagnostics := semanticEvidenceASTDiagnosticsFixture()
	def, refs := semanticEvidenceJSBudgetFixture()

	evidence, ok := semanticEvidenceFromJSFamilyRefs("typescript", "Widget", def, refs, diagnostics)
	if !ok {
		t.Fatal("semanticEvidenceFromJSFamilyRefs() ok = false")
	}
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindImport, jsImportLimit)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindCaller, jsCallerLimit)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindTypeRef, jsTypeRefLimit)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindReference, genericRefLimit)
	assertSemanticReferenceKindCount(t, evidence.References, SemanticReferenceKindTest, genericTestLimit)

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence(js evidence) ok = false")
	}
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindImports, jsImportLimit)
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindCallers, jsCallerLimit)
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindTypeRefs, jsTypeRefLimit)
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindReferences, genericRefLimit)
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindTests, genericTestLimit)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindImports, jsImportLimit+2, true)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindCallers, jsCallerLimit+2, true)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindTypeRefs, jsTypeRefLimit+2, true)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindReferences, genericRefLimit+2, true)
	assertSymbolBundleSectionSummary(t, bundle, SemanticReferenceSectionKindTests, genericTestLimit+2, true)

	readBudget := 1 + jsImportLimit + jsCallerLimit + jsTypeRefLimit + genericRefLimit + genericTestLimit
	if got := len(bundle.Impact.RecommendedReads); got > readBudget {
		t.Fatalf("RecommendedReads len = %d, want <= %d", got, readBudget)
	}
	output := formatSymbolBundle(bundle, nil, nil)
	assertOutputContains(t, output, "Imports: 5 shown (of 7)")
	assertOutputContains(t, output, "Callers: 10 shown (of 12)")
	assertOutputContains(t, output, "Type References: 5 shown (of 7)")
	assertOutputContains(t, output, "References: 15 shown (of 17)")
	assertOutputContains(t, output, "Related Tests: 5 shown (of 7)")
	assertOutputContains(t, output, "Omitted: callers +2, references +2, tests +2")
}

func TestBuildJSFamilySemanticEvidenceUsesSearchOptionsForNearbyTestsAndRisk(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":      "export function buildUser(id: string) { return id }\n",
		"src/build.test.ts": "it('builds nearby', () => buildUser('nearby'))\n",
	})
	def := genericSymbolDef{
		Name:      "buildUser",
		Kind:      "function",
		File:      "src/build.ts",
		Line:      1,
		Signature: "export function buildUser(id: string) { return id }",
	}
	refs := []genericSymbolRef{
		{File: "src/app1.ts", Line: 1, Snippet: "buildUser('1')", Class: codeast.ClassCall},
		{File: "src/app2.ts", Line: 1, Snippet: "buildUser('2')", Class: codeast.ClassCall},
		{File: "src/app3.ts", Line: 1, Snippet: "buildUser('3')", Class: codeast.ClassCall},
		{File: "src/app4.ts", Line: 1, Snippet: "buildUser('4')", Class: codeast.ClassCall},
	}
	diagnostics := semanticEvidenceASTDiagnosticsFixture()

	withOptions, ok := buildJSFamilySemanticEvidence("typescript", "buildUser", def, newTypeScriptImpactSearchOptions(dir, "buildUser"), refs, refs, diagnostics)
	if !ok {
		t.Fatal("buildJSFamilySemanticEvidence(with options) ok = false")
	}
	if withOptions.RiskLevel != impactplan.RiskMedium {
		t.Fatalf("RiskLevel with nearby test = %q, want %q", withOptions.RiskLevel, impactplan.RiskMedium)
	}
	assertSemanticReferenceKindCount(t, withOptions.References, SemanticReferenceKindTest, 1)
	if withOptions.Definitions[0].RootPath == "" || withOptions.Definitions[0].ResolvedPath == "" {
		t.Fatalf("definition paths = root:%q resolved:%q, want production paths", withOptions.Definitions[0].RootPath, withOptions.Definitions[0].ResolvedPath)
	}

	withoutOptions, ok := buildJSFamilySemanticEvidence("typescript", "buildUser", def, SearchOptions{}, refs, refs, diagnostics)
	if !ok {
		t.Fatal("buildJSFamilySemanticEvidence(without options) ok = false")
	}
	if withoutOptions.RiskLevel != impactplan.RiskHigh {
		t.Fatalf("RiskLevel without nearby test = %q, want %q", withoutOptions.RiskLevel, impactplan.RiskHigh)
	}
	assertSemanticReferenceKindCount(t, withoutOptions.References, SemanticReferenceKindTest, 0)
}
