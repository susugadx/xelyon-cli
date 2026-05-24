package search

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/jsast"
)

func TestBuildSymbolBundleFromSemanticEvidenceAcceptsDefinitionAttributesAndReflectsRisk(t *testing.T) {
	evidence := SemanticEvidence{
		Language:  "typescript",
		Query:     "buildUser",
		Symbol:    "buildUser",
		RiskLevel: "high",
		Definitions: []SemanticDefinition{{
			Name:           "buildUser",
			Kind:           "function",
			Exported:       true,
			Implementation: true,
			Declaration:    false,
			File:           "src/build.ts",
			Line:           1,
			Signature:      "export function buildUser(id: string) { return id }",
		}},
	}

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence() ok = false")
	}
	if bundle.Impact == nil {
		t.Fatal("Impact = nil, want semantic impact metadata")
	}
	if got := bundle.Impact.RiskLevel; got != "high" {
		t.Fatalf("RiskLevel = %q, want high", got)
	}
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 0, "definition", "src/build.ts")
}

func TestSemanticEvidenceFromGoInspectResultDefinitionAttributes(t *testing.T) {
	exported := semanticEvidenceGoBuildInspectFixture()
	exportedEvidence, ok := semanticEvidenceFromGoInspectResult("Build", exported)
	if !ok {
		t.Fatal("semanticEvidenceFromGoInspectResult(exported) ok = false")
	}
	assertSemanticDefinitionAttributes(t, exportedEvidence.Definitions[0], true, true, false)

	unexported := semanticEvidenceGoBuildInspectFixture()
	unexported.Symbol.Name = "build"
	unexported.Symbol.Signature = "func build() string"
	unexported.Symbol.Exported = false
	unexportedEvidence, ok := semanticEvidenceFromGoInspectResult("build", unexported)
	if !ok {
		t.Fatal("semanticEvidenceFromGoInspectResult(unexported) ok = false")
	}
	assertSemanticDefinitionAttributes(t, unexportedEvidence.Definitions[0], false, true, false)

	method := semanticEvidenceGoRunInspectFixture(stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error"))
	methodEvidence, ok := semanticEvidenceFromGoInspectResult("Run", method)
	if !ok {
		t.Fatal("semanticEvidenceFromGoInspectResult(method) ok = false")
	}
	methodDef := methodEvidence.Definitions[0]
	if methodDef.Name != "Run" || methodDef.DisplayName != "(*Agent).Run" {
		t.Fatalf("method identity = name:%q display:%q, want Run / (*Agent).Run", methodDef.Name, methodDef.DisplayName)
	}
	assertSemanticDefinitionAttributes(t, methodDef, true, true, false)
}

func TestSemanticEvidenceFromJSFamilyRefsDefinitionAttributes(t *testing.T) {
	tests := []struct {
		name               string
		language           string
		def                genericSymbolDef
		refs               []genericSymbolRef
		wantExported       bool
		wantImplementation bool
		wantDeclaration    bool
	}{
		{
			name:     "typescript export function implementation",
			language: "typescript",
			def: genericSymbolDef{
				Name:      "buildUser",
				Kind:      "function",
				File:      "src/build.ts",
				Line:      1,
				Signature: "export function buildUser(id: string) { return id }",
			},
			wantExported:       true,
			wantImplementation: true,
		},
		{
			name:     "tsx export const implementation",
			language: "typescript",
			def: genericSymbolDef{
				Name:      "Button",
				Kind:      "const",
				File:      "src/Button.tsx",
				Line:      1,
				Signature: "export const Button = () => <button />",
			},
			wantExported:       true,
			wantImplementation: true,
		},
		{
			name:     "typescript declaration",
			language: "typescript",
			def: genericSymbolDef{
				Name:      "BuildOptions",
				Kind:      "interface",
				File:      "src/types.d.ts",
				Line:      1,
				Signature: "export interface BuildOptions { id: string }",
			},
			wantExported:    true,
			wantDeclaration: true,
		},
		{
			name:     "javascript export default function implementation",
			language: "javascript",
			def: genericSymbolDef{
				Name:      "buildUser",
				Kind:      "function",
				File:      "src/build.js",
				Line:      1,
				Signature: "export default function buildUser(id) { return id }",
			},
			wantExported:       true,
			wantImplementation: true,
		},
		{
			name:     "jsx export const implementation",
			language: "javascript",
			def: genericSymbolDef{
				Name:      "Button",
				Kind:      "const",
				File:      "src/Button.jsx",
				Line:      1,
				Signature: "export const Button = () => <button />",
			},
			wantExported:       true,
			wantImplementation: true,
		},
		{
			name:     "commonjs export implementation",
			language: "javascript",
			def: genericSymbolDef{
				Name:      "buildUser",
				Kind:      "function",
				File:      "src/build.js",
				Line:      1,
				Signature: "module.exports = function buildUser(id) { return id }",
			},
			wantExported:       true,
			wantImplementation: true,
		},
		{
			name:     "local function",
			language: "typescript",
			def: genericSymbolDef{
				Name:      "buildLocal",
				Kind:      "function",
				File:      "src/build.ts",
				Line:      1,
				Signature: "function buildLocal() { return 1 }",
			},
			wantImplementation: true,
		},
		{
			name:     "export reference marks definition exported",
			language: "typescript",
			def: genericSymbolDef{
				Name:      "buildUser",
				Kind:      "function",
				File:      "src/build.ts",
				Line:      1,
				Signature: "function buildUser(id: string) { return id }",
			},
			refs: []genericSymbolRef{{
				File:    "src/index.ts",
				Line:    1,
				Snippet: "export { buildUser }",
				Class:   jsast.ClassExport,
			}},
			wantExported:       true,
			wantImplementation: true,
		},
	}

	diagnostics := semanticEvidenceASTDiagnosticsFixture()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence, ok := semanticEvidenceFromJSFamilyRefs(tt.language, tt.def.Name, tt.def, tt.refs, diagnostics)
			if !ok {
				t.Fatal("semanticEvidenceFromJSFamilyRefs() ok = false")
			}
			assertSemanticDefinitionAttributes(t, evidence.Definitions[0], tt.wantExported, tt.wantImplementation, tt.wantDeclaration)
		})
	}
}
