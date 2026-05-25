package search

import (
	"errors"
	"path/filepath"
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestJSFamilySemanticEvidenceProductionBundle_ASTScenarios(t *testing.T) {
	tests := []jsFamilySemanticProductionASTScenario{
		{
			name:        "typescript function",
			symbol:      "buildUser",
			language:    "typescript",
			debugSource: "typescript-impact-structured",
			files: map[string]string{
				"src/build.ts":      "export function buildUser(id: string) { return id }\n",
				"src/app.ts":        "import { buildUser } from './build'\nexport const user = buildUser('1')\n",
				"src/build.test.ts": "import { buildUser } from './build'\nit('builds', () => buildUser('test'))\n",
			},
			resolve: resolveTypeScriptSemanticProductionScenario,
		},
		{
			name:        "tsx component",
			symbol:      "Button",
			language:    "typescript",
			debugSource: "typescript-impact-structured",
			files: map[string]string{
				"src/Button.tsx":      "export function Button() { return <button /> }\n",
				"src/App.tsx":         "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
				"src/Button.test.tsx": "import { Button } from './Button'\nit('renders', () => <Button />)\n",
			},
			resolve: resolveTSXSemanticProductionScenario,
		},
		{
			name:        "javascript function",
			symbol:      "buildUser",
			language:    "javascript",
			debugSource: "javascript-impact-structured",
			files: map[string]string{
				"src/build.js":      "export function buildUser(id) { return id }\n",
				"src/app.js":        "import { buildUser } from './build.js'\nexport const user = buildUser('1')\n",
				"src/index.js":      "export { buildUser } from './build.js'\n",
				"src/build.test.js": "import { buildUser } from './build.js'\nit('builds', () => buildUser('test'))\n",
			},
			resolve:    resolveJavaScriptSemanticProductionScenario,
			wantImport: "src/index.js",
		},
		{
			name:        "jsx component",
			symbol:      "Button",
			language:    "javascript",
			debugSource: "javascript-impact-structured",
			files: map[string]string{
				"src/Button.jsx":      "export function Button() { return <button /> }\n",
				"src/App.jsx":         "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
				"src/Button.test.jsx": "import { Button } from './Button'\nit('renders', () => <Button />)\n",
			},
			resolve: resolveJSXSemanticProductionScenario,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, tt.files)

			result := tt.resolve(dir, tt.symbol)

			assertJSFamilySemanticProductionBundle(t, result, tt.language, tt.debugSource)
			assertDiagnosticsResolvedBy(t, result.Bundle.Diagnostics, symbolBundleResolvedByAST)
			assertJSFamilyProductionConfidence(t, result, symbolBundleConfidenceMedium)
			for _, kind := range []string{
				SemanticReferenceSectionKindImports,
				SemanticReferenceSectionKindCallers,
				SemanticReferenceSectionKindTests,
			} {
				assertJSFamilyProductionSectionSummaryPresent(t, result.Bundle, kind)
			}
			if tt.wantImport != "" {
				assertJSFamilyProductionImportsContainFile(t, result, tt.wantImport)
			}
		})
	}
}

func TestJSFamilySemanticEvidenceProductionBundle_PreservesLSPDiagnostics(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser } from './build'\nbuildUser('semantic')\n",
	})
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.ts", Line: 2, Character: 1}},
	}

	result := resolveStructuredTypeScriptImpactSymbol("buildUser", structuredImpactSameScope(opts))

	assertJSFamilySemanticProductionBundle(t, result, "typescript", "typescript-impact-structured")
	assertDiagnosticsResolvedBy(t, result.Bundle.Diagnostics, symbolBundleResolvedByLSP)
	assertJSFamilyProductionConfidence(t, result, symbolBundleConfidenceHigh)
	assertOutputContains(t, result.Output, "Note: resolved via TypeScript/JavaScript LSP.")
}

func TestJSFamilySemanticEvidenceProductionBundle_PreservesMixedFallbackDiagnostics(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js":   "import { buildUser } from './build.js'\nbuildUser('fallback')\n",
	})
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{err: errors.New("lsp unavailable")}

	result := resolveStructuredJavaScriptImpactSymbol("buildUser", structuredImpactSameScope(opts))

	assertJSFamilySemanticProductionBundle(t, result, "javascript", "javascript-impact-structured")
	assertDiagnosticsResolvedBy(t, result.Bundle.Diagnostics, symbolBundleResolvedByMixed)
	if result.Bundle.Diagnostics.FallbackReason != symbolBundleFallbackReasonLSPError {
		t.Fatalf("bundle fallback reason = %q, want %q", result.Bundle.Diagnostics.FallbackReason, symbolBundleFallbackReasonLSPError)
	}
	assertJSFamilyProductionConfidence(t, result, symbolBundleConfidenceLow)
	assertOutputContains(t, result.Output, "Diagnostics: resolved_by=mixed, confidence=low, fallback_reason=lsp_error")
}

func TestJSFamilySemanticEvidenceProductionBundle_KeepsTypeScriptSummaryTotalsAndRisk(t *testing.T) {
	def := genericSymbolDef{
		Name:      "buildUser",
		Kind:      "function",
		File:      "src/build.ts",
		Line:      1,
		Signature: "function buildUser(id: string) { return id }",
	}
	displayRefs := []genericSymbolRef{{
		File:    "src/app0.ts",
		Line:    1,
		Snippet: "buildUser('0')",
		Class:   codeast.ClassCall,
	}}
	totalRefs := make([]genericSymbolRef, 0, jsFamilyImpactHighNonTestReferenceThreshold)
	for i := 0; i < jsFamilyImpactHighNonTestReferenceThreshold; i++ {
		totalRefs = append(totalRefs, genericSymbolRef{
			File:  filepath.ToSlash(filepath.Join("src", "app"+string(rune('0'+i))+".ts")),
			Line:  1,
			Class: codeast.ClassCall,
		})
	}

	evidence, ok := buildJSFamilySemanticEvidence("typescript", "buildUser", def, SearchOptions{}, displayRefs, totalRefs, semanticEvidenceASTDiagnosticsFixture())
	if !ok {
		t.Fatal("buildJSFamilySemanticEvidence() ok = false")
	}
	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence() ok = false")
	}

	if bundle == nil || bundle.Impact == nil {
		t.Fatal("bundle impact = nil, want structured impact bundle")
	}
	if got := bundle.Impact.RiskLevel; got != impactplan.RiskHigh {
		t.Fatalf("risk = %q, want %q from total refs summary", got, impactplan.RiskHigh)
	}
	callers := symbolBundleSectionByKind(bundle, SemanticReferenceSectionKindCallers)
	if callers == nil {
		t.Fatal("callers section = nil, want budgeted evidence section")
	}
	if len(callers.Items) != 1 {
		t.Fatalf("callers items len = %d, want one display evidence item", len(callers.Items))
	}
	if callers.Total != jsFamilyImpactHighNonTestReferenceThreshold || !callers.More {
		t.Fatalf("callers total/more = %d/%v, want total refs summary %d with More", callers.Total, callers.More, jsFamilyImpactHighNonTestReferenceThreshold)
	}
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

func assertJSFamilySemanticProductionBundle(t *testing.T, result symbolResolveResult, language string, debugSource string) {
	t.Helper()
	if result.Status != symbolResolveSingle {
		t.Fatalf("status = %s, want %s; output:\n%s", result.Status, symbolResolveSingle, result.Output)
	}
	if result.Bundle == nil {
		t.Fatal("bundle = nil, want structured bundle")
	}
	bundle := result.Bundle
	if bundle.Identity.Language != language {
		t.Fatalf("bundle language = %q, want %q", bundle.Identity.Language, language)
	}
	if bundle.Identity.DisplayName == "" || bundle.Identity.Kind == "" {
		t.Fatalf("bundle identity missing display/kind: %+v", bundle.Identity)
	}
	if bundle.Definition.File == "" || bundle.Definition.Line <= 0 {
		t.Fatalf("bundle definition = %+v, want file and line", bundle.Definition)
	}
	if bundle.Debug.Source != debugSource {
		t.Fatalf("bundle debug source = %q, want %q", bundle.Debug.Source, debugSource)
	}
	if bundle.Debug.FileRootPath == "" {
		t.Fatal("bundle debug file root path is empty")
	}
	if bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		t.Fatalf("bundle impact = %+v, want recommended reads", bundle.Impact)
	}
	if got := bundle.Impact.RecommendedReads[0]; got.Kind != "definition" || got.File != bundle.Definition.File || got.Line != bundle.Definition.Line {
		t.Fatalf("first recommended read = %+v, want definition %s:%d", got, bundle.Definition.File, bundle.Definition.Line)
	}
}

func assertJSFamilyProductionSectionSummaryPresent(t *testing.T, bundle *SymbolBundle, kind string) {
	t.Helper()
	section := symbolBundleSectionByKind(bundle, kind)
	if section == nil {
		t.Fatalf("section %q missing from bundle sections %+v", kind, bundle.Sections)
	}
	if section.Total < len(section.Items) {
		t.Fatalf("section %q total = %d, want >= visible %d", kind, section.Total, len(section.Items))
	}
}

func assertJSFamilyProductionConfidence(t *testing.T, result symbolResolveResult, want string) {
	t.Helper()
	if result.Bundle.Diagnostics.Confidence != want {
		t.Fatalf("bundle confidence = %q, want %q", result.Bundle.Diagnostics.Confidence, want)
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

type jsFamilySemanticProductionASTScenario struct {
	name        string
	symbol      string
	language    string
	debugSource string
	files       map[string]string
	resolve     func(dir string, symbol string) symbolResolveResult
	wantImport  string
}

func resolveTypeScriptSemanticProductionScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredTypeScriptImpactSymbol(symbol, structuredImpactSameScope(newTypeScriptImpactSearchOptions(dir, symbol)))
}

func resolveTSXSemanticProductionScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredTypeScriptImpactSymbol(symbol, structuredImpactSameScope(newTSXImpactSearchOptions(dir, symbol)))
}

func resolveJavaScriptSemanticProductionScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredJavaScriptImpactSymbol(symbol, structuredImpactSameScope(newJavaScriptImpactSearchOptions(dir, symbol)))
}

func resolveJSXSemanticProductionScenario(dir string, symbol string) symbolResolveResult {
	return resolveStructuredJavaScriptImpactSymbol(symbol, structuredImpactSameScope(newJSXImpactSearchOptions(dir, symbol)))
}

func assertJSFamilyProductionImportsContainFile(t *testing.T, result symbolResolveResult, file string) {
	t.Helper()
	imports := symbolBundleSectionByKind(result.Bundle, SemanticReferenceSectionKindImports)
	if imports == nil || !symbolBundleItemsContainFile(imports.Items, file) {
		t.Fatalf("imports section = %+v, want %s", imports, file)
	}
}
