package search

import (
	"errors"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestJSFamilySemanticEvidenceShadowBundle_ASTScenarios(t *testing.T) {
	tests := []jsFamilySemanticShadowASTScenario{
		{
			name:   "typescript function",
			symbol: "buildUser",
			files: map[string]string{
				"src/build.ts":      "export function buildUser(id: string) { return id }\n",
				"src/app.ts":        "import { buildUser } from './build'\nexport const user = buildUser('1')\n",
				"src/build.test.ts": "import { buildUser } from './build'\nit('builds', () => buildUser('test'))\n",
			},
			resolve: resolveTypeScriptSemanticShadowScenario,
		},
		{
			name:   "tsx component",
			symbol: "Button",
			files: map[string]string{
				"src/Button.tsx":      "export function Button() { return <button /> }\n",
				"src/App.tsx":         "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
				"src/Button.test.tsx": "import { Button } from './Button'\nit('renders', () => <Button />)\n",
			},
			resolve: resolveTSXSemanticShadowScenario,
		},
		{
			name:   "javascript function",
			symbol: "buildUser",
			files: map[string]string{
				"src/build.js":      "export function buildUser(id) { return id }\n",
				"src/app.js":        "import { buildUser } from './build.js'\nexport const user = buildUser('1')\n",
				"src/index.js":      "export { buildUser } from './build.js'\n",
				"src/build.test.js": "import { buildUser } from './build.js'\nit('builds', () => buildUser('test'))\n",
			},
			resolve:          resolveJavaScriptSemanticShadowScenario,
			wantShadowImport: "src/index.js",
		},
		{
			name:   "jsx component",
			symbol: "Button",
			files: map[string]string{
				"src/Button.jsx":      "export function Button() { return <button /> }\n",
				"src/App.jsx":         "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
				"src/Button.test.jsx": "import { Button } from './Button'\nit('renders', () => <Button />)\n",
			},
			resolve: resolveJSXSemanticShadowScenario,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, tt.files)

			result := tt.resolve(dir, tt.symbol)

			assertJSFamilySemanticShadowBundleEquivalent(t, result)
			assertDiagnosticsResolvedBy(t, result.Bundle.Diagnostics, symbolBundleResolvedByAST)
			assertDiagnosticsResolvedBy(t, result.SemanticShadowBundle.Diagnostics, symbolBundleResolvedByAST)
			assertJSFamilyShadowConfidence(t, result, symbolBundleConfidenceMedium)
			if tt.wantShadowImport != "" {
				assertJSFamilyShadowImportsContainFile(t, result, tt.wantShadowImport)
			}
		})
	}
}

func TestJSFamilySemanticEvidenceShadowBundle_PreservesLSPDiagnostics(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser } from './build'\nbuildUser('semantic')\n",
	})
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.ts", Line: 2, Character: 1}},
	}

	result := resolveStructuredTypeScriptImpactSymbol("buildUser", structuredImpactSameScope(opts))

	assertJSFamilySemanticShadowBundleEquivalent(t, result)
	assertDiagnosticsResolvedBy(t, result.Bundle.Diagnostics, symbolBundleResolvedByLSP)
	assertDiagnosticsResolvedBy(t, result.SemanticShadowBundle.Diagnostics, symbolBundleResolvedByLSP)
	assertJSFamilyShadowConfidence(t, result, symbolBundleConfidenceHigh)
}

func TestJSFamilySemanticEvidenceShadowBundle_PreservesMixedFallbackDiagnostics(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js":   "import { buildUser } from './build.js'\nbuildUser('fallback')\n",
	})
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{err: errors.New("lsp unavailable")}

	result := resolveStructuredJavaScriptImpactSymbol("buildUser", structuredImpactSameScope(opts))

	assertJSFamilySemanticShadowBundleEquivalent(t, result)
	assertDiagnosticsResolvedBy(t, result.Bundle.Diagnostics, symbolBundleResolvedByMixed)
	assertDiagnosticsResolvedBy(t, result.SemanticShadowBundle.Diagnostics, symbolBundleResolvedByMixed)
	if result.Bundle.Diagnostics.FallbackReason != symbolBundleFallbackReasonLSPError {
		t.Fatalf("bundle fallback reason = %q, want %q", result.Bundle.Diagnostics.FallbackReason, symbolBundleFallbackReasonLSPError)
	}
	if result.SemanticShadowBundle.Diagnostics.FallbackReason != symbolBundleFallbackReasonLSPError {
		t.Fatalf("shadow fallback reason = %q, want %q", result.SemanticShadowBundle.Diagnostics.FallbackReason, symbolBundleFallbackReasonLSPError)
	}
	assertJSFamilyShadowConfidence(t, result, symbolBundleConfidenceLow)
}

func TestJSFamilySemanticEvidenceShadowBundle_PreservesRiskAndEvidenceAttributes(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser } from './build'\nbuildUser('semantic')\n",
	})

	result := resolveStructuredTypeScriptImpactSymbol("buildUser", structuredImpactSameScope(newTypeScriptImpactSearchOptions(dir, "buildUser")))

	assertJSFamilySemanticShadowBundleEquivalent(t, result)
	if result.SemanticShadowEvidence == nil {
		t.Fatal("SemanticShadowEvidence = nil, want shadow evidence")
	}
	evidence := result.SemanticShadowEvidence
	if evidence.RiskLevel != result.Bundle.Impact.RiskLevel {
		t.Fatalf("shadow evidence RiskLevel = %q, want production risk %q", evidence.RiskLevel, result.Bundle.Impact.RiskLevel)
	}
	if len(evidence.Definitions) != 1 {
		t.Fatalf("shadow evidence definitions = %+v, want one definition", evidence.Definitions)
	}
	assertSemanticDefinitionAttributes(t, evidence.Definitions[0], true, true, false)
}

func TestBuildSymbolBundleFromSemanticEvidenceDedupeRecommendedReadsByLocationOnly(t *testing.T) {
	evidence := SemanticEvidence{
		Language: "typescript",
		Query:    "buildUser",
		Symbol:   "buildUser",
		Definitions: []SemanticDefinition{{
			Name:      "buildUser",
			Kind:      "function",
			File:      "src/build.ts",
			Line:      1,
			Signature: "export function buildUser(id: string) { return id }",
		}},
		References: []SemanticReference{
			{Kind: SemanticReferenceKindCaller, File: "src/app.ts", Line: 8, Snippet: "buildUser('1')"},
			{Kind: SemanticReferenceKindReference, File: "src/app.ts", Line: 8, Snippet: "const ref = buildUser"},
		},
	}

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence() ok = false")
	}

	if got := recommendedReadLocationCount(bundle, "src/app.ts", 8); got != 1 {
		t.Fatalf("recommended read count for src/app.ts:8 = %d, want 1; reads = %+v", got, bundle.Impact.RecommendedReads)
	}
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindCallers, 1)
	assertSymbolBundleSectionItemCount(t, bundle, SemanticReferenceSectionKindReferences, 1)
}

func assertJSFamilySemanticShadowBundleEquivalent(t *testing.T, result symbolResolveResult) {
	t.Helper()
	if result.Status != symbolResolveSingle {
		t.Fatalf("status = %s, want %s; output:\n%s", result.Status, symbolResolveSingle, result.Output)
	}
	if result.Bundle == nil {
		t.Fatal("bundle = nil, want structured bundle")
	}
	if result.SemanticShadowBundle == nil {
		t.Fatalf("SemanticShadowBundle = nil, want shadow bundle for %+v", result.Bundle.Identity)
	}

	bundle := result.Bundle
	shadow := result.SemanticShadowBundle
	if bundle.Identity.Language != shadow.Identity.Language ||
		bundle.Identity.DisplayName != shadow.Identity.DisplayName ||
		bundle.Identity.Kind != shadow.Identity.Kind {
		t.Fatalf("shadow identity = %+v, want language/displayName/kind from %+v", shadow.Identity, bundle.Identity)
	}
	if bundle.Definition.File != shadow.Definition.File || bundle.Definition.Line != shadow.Definition.Line {
		t.Fatalf("shadow definition = %s:%d, want %s:%d", shadow.Definition.File, shadow.Definition.Line, bundle.Definition.File, bundle.Definition.Line)
	}
	if bundle.Diagnostics.ResolvedBy != shadow.Diagnostics.ResolvedBy || bundle.Diagnostics.Confidence != shadow.Diagnostics.Confidence {
		t.Fatalf("shadow diagnostics = resolved_by:%q confidence:%q, want %q/%q", shadow.Diagnostics.ResolvedBy, shadow.Diagnostics.Confidence, bundle.Diagnostics.ResolvedBy, bundle.Diagnostics.Confidence)
	}
	if bundle.Impact == nil || shadow.Impact == nil {
		t.Fatalf("impact presence mismatch: bundle=%+v shadow=%+v", bundle.Impact, shadow.Impact)
	}
	if bundle.Impact.RiskLevel != shadow.Impact.RiskLevel {
		t.Fatalf("shadow risk = %q, want %q", shadow.Impact.RiskLevel, bundle.Impact.RiskLevel)
	}

	assertJSFamilyShadowFirstRecommendedRead(t, bundle, shadow)
	for _, kind := range []string{
		SemanticReferenceSectionKindCallers,
		SemanticReferenceSectionKindImports,
		SemanticReferenceSectionKindReferences,
		SemanticReferenceSectionKindTypeRefs,
		SemanticReferenceSectionKindTests,
	} {
		assertJSFamilyShadowSectionSummary(t, bundle, shadow, kind)
	}
}

func assertJSFamilyShadowFirstRecommendedRead(t *testing.T, bundle *SymbolBundle, shadow *SymbolBundle) {
	t.Helper()
	if bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("bundle recommended reads empty, want definition read")
	}
	if shadow.Impact == nil || len(shadow.Impact.RecommendedReads) == 0 {
		t.Fatal("shadow recommended reads empty, want definition read")
	}
	got := shadow.Impact.RecommendedReads[0]
	want := bundle.Impact.RecommendedReads[0]
	if got.Kind != want.Kind || got.File != want.File || got.Line != want.Line {
		t.Fatalf("shadow first recommended read = %+v, want %+v", got, want)
	}
	if got.Kind != "definition" {
		t.Fatalf("shadow first recommended read kind = %q, want definition", got.Kind)
	}
}

func assertJSFamilyShadowSectionSummary(t *testing.T, bundle *SymbolBundle, shadow *SymbolBundle, kind string) {
	t.Helper()
	want := symbolBundleSectionByKind(bundle, kind)
	got := symbolBundleSectionByKind(shadow, kind)
	if want == nil || got == nil {
		if want != nil || got != nil {
			t.Fatalf("section %q presence mismatch: bundle=%+v shadow=%+v", kind, want, got)
		}
		return
	}
	if got.Total != want.Total || got.More != want.More {
		t.Fatalf("shadow section %q total/more = %d/%v, want %d/%v", kind, got.Total, got.More, want.Total, want.More)
	}
}

func assertJSFamilyShadowConfidence(t *testing.T, result symbolResolveResult, want string) {
	t.Helper()
	if result.Bundle.Diagnostics.Confidence != want {
		t.Fatalf("bundle confidence = %q, want %q", result.Bundle.Diagnostics.Confidence, want)
	}
	if result.SemanticShadowBundle.Diagnostics.Confidence != want {
		t.Fatalf("shadow confidence = %q, want %q", result.SemanticShadowBundle.Diagnostics.Confidence, want)
	}
}

func recommendedReadLocationCount(bundle *SymbolBundle, file string, line int) int {
	if bundle == nil || bundle.Impact == nil {
		return 0
	}
	count := 0
	for _, read := range bundle.Impact.RecommendedReads {
		if read.File == file && read.Line == line {
			count++
		}
	}
	return count
}

type jsFamilySemanticShadowASTScenario struct {
	name             string
	symbol           string
	files            map[string]string
	resolve          func(dir string, symbol string) symbolResolveResult
	wantShadowImport string
}

func resolveTypeScriptSemanticShadowScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredTypeScriptImpactSymbol(symbol, structuredImpactSameScope(newTypeScriptImpactSearchOptions(dir, symbol)))
}

func resolveTSXSemanticShadowScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredTypeScriptImpactSymbol(symbol, structuredImpactSameScope(newTSXImpactSearchOptions(dir, symbol)))
}

func resolveJavaScriptSemanticShadowScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredJavaScriptImpactSymbol(symbol, structuredImpactSameScope(newJavaScriptImpactSearchOptions(dir, symbol)))
}

func resolveJSXSemanticShadowScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredJavaScriptImpactSymbol(symbol, structuredImpactSameScope(newJSXImpactSearchOptions(dir, symbol)))
}

func assertJSFamilyShadowImportsContainFile(t *testing.T, result symbolResolveResult, file string) {
	t.Helper()
	imports := symbolBundleSectionByKind(result.SemanticShadowBundle, SemanticReferenceSectionKindImports)
	if imports == nil || !symbolBundleItemsContainFile(imports.Items, file) {
		t.Fatalf("shadow imports section = %+v, want %s", imports, file)
	}
}
