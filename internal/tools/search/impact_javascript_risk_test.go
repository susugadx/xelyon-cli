package search

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

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
