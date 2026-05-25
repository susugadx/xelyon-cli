package search

import (
	"fmt"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func semanticEvidenceGoBuildInspectFixture() navigation.InspectResult {
	return navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:      "Build",
			Kind:      "function",
			File:      "pkg/build.go",
			Line:      3,
			EndLine:   5,
			Signature: "func Build() string",
			Exported:  true,
			RootPath:  "/repo",
		},
		Body: []string{
			"3: func Build() string {",
			"4:   return \"ok\"",
			"5: }",
		},
		Callers: []navigation.Reference{{
			File:         "pkg/app.go",
			ResolvedPath: "/repo/pkg/app.go",
			Line:         8,
			Scope:        "Run",
			Snippet:      "Build()",
		}},
		Refs: []navigation.Reference{{
			File:    "pkg/doc.go",
			Line:    2,
			Snippet: "Build",
		}},
		Tests: []navigation.TestRef{{
			File: "pkg/build_test.go",
			Line: 7,
			Name: "TestBuild",
		}},
		Implementations: []navigation.ImplementationRef{{
			File: "pkg/impl.go",
			Line: 11,
			Name: "Builder",
		}},
		ResolvedViaLSP: true,
		ReferenceDiagnostics: navigation.InspectReferenceDiagnostics{
			ResolvedBy:       SymbolBundleResolvedByLSP,
			LSPAttempted:     true,
			LSPAvailable:     true,
			RawRefCount:      4,
			AcceptedRefCount: 3,
			DroppedRefCount:  1,
		},
	}
}

func semanticEvidenceGoRunInspectFixture(stableKey string) navigation.InspectResult {
	return navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:         "Run",
			Kind:         "method",
			File:         "pkg/run.go",
			Line:         10,
			EndLine:      12,
			Receiver:     "*Agent",
			ReceiverNorm: "Agent",
			Signature:    "func (a *Agent) Run() error",
			Exported:     true,
			PackageDir:   "pkg",
			StableKey:    stableKey,
		},
		Body: []string{
			"10: func (a *Agent) Run() error {",
			"11:   return nil",
			"12: }",
		},
		Callers: []navigation.Reference{{
			File:    "pkg/main.go",
			Line:    20,
			Snippet: "agent.Run()",
		}},
		Refs: []navigation.Reference{{
			File:    "pkg/app.go",
			Line:    30,
			Snippet: "Run",
		}},
		Tests: []navigation.TestRef{{
			File: "pkg/run_test.go",
			Line: 14,
			Name: "TestRun",
		}},
		TotalCallers: 4,
		MoreCallers:  true,
		TotalRefs:    3,
		MoreRefs:     true,
		TotalTests:   2,
		MoreTests:    true,
	}
}

func semanticEvidenceGoImplementationBudgetFixture(total int) navigation.InspectResult {
	result := semanticEvidenceGoBuildInspectFixture()
	result.Symbol.Kind = "interface"
	result.Implementations = make([]navigation.ImplementationRef, 0, total)
	for i := 1; i <= total; i++ {
		result.Implementations = append(result.Implementations, navigation.ImplementationRef{
			File: fmt.Sprintf("pkg/impl_%02d.go", i),
			Line: 10 + i,
			Name: fmt.Sprintf("Implementation%d", i),
		})
	}
	return result
}

func semanticEvidenceJSButtonFixture() (genericSymbolDef, []genericSymbolRef) {
	def := genericSymbolDef{
		Name:      "Button",
		Kind:      "function",
		File:      "src/Button.tsx",
		Line:      1,
		Signature: "export function Button() { return <button /> }",
	}
	refs := []genericSymbolRef{
		{File: "src/App.tsx", Line: 4, Snippet: "<Button />", Class: codeast.ClassCall},
		{File: "src/index.ts", Line: 1, Snippet: "export { Button }", Class: jsast.ClassExport},
		{File: "src/types.ts", Line: 2, Snippet: "type View = Button", Class: jsast.ClassTypeRef},
		{File: "src/imports.ts", Line: 1, Snippet: "import { Button } from './Button'", Class: codeast.ClassImport},
		{File: "src/readme.ts", Line: 9, Snippet: "const ref = Button", Class: codeast.ClassRef},
		{File: "src/Button.test.tsx", Line: 5, Snippet: "render(<Button />)", Class: codeast.ClassCall, IsTest: true},
		{File: "src/comment.ts", Line: 1, Snippet: "// Button", Class: codeast.ClassComment},
	}
	return def, refs
}

func semanticEvidenceJSBudgetFixture() (genericSymbolDef, []genericSymbolRef) {
	def := genericSymbolDef{
		Name:      "Widget",
		Kind:      "function",
		File:      "src/Widget.tsx",
		Line:      1,
		Signature: "export function Widget() { return <div /> }",
	}
	refs := make([]genericSymbolRef, 0,
		(jsImportLimit+2)+
			(jsCallerLimit+2)+
			(jsTypeRefLimit+2)+
			(genericRefLimit+2)+
			(genericTestLimit+2),
	)
	for i := 1; i <= jsImportLimit+2; i++ {
		refs = append(refs, genericSymbolRef{
			File:    fmt.Sprintf("src/import_%02d.ts", i),
			Line:    i,
			Snippet: "import { Widget } from './Widget'",
			Class:   codeast.ClassImport,
		})
	}
	for i := 1; i <= jsCallerLimit+2; i++ {
		refs = append(refs, genericSymbolRef{
			File:    fmt.Sprintf("src/caller_%02d.tsx", i),
			Line:    i,
			Snippet: "<Widget />",
			Class:   codeast.ClassCall,
		})
	}
	for i := 1; i <= jsTypeRefLimit+2; i++ {
		refs = append(refs, genericSymbolRef{
			File:    fmt.Sprintf("src/type_%02d.ts", i),
			Line:    i,
			Snippet: "type View = Widget",
			Class:   jsast.ClassTypeRef,
		})
	}
	for i := 1; i <= genericRefLimit+2; i++ {
		refs = append(refs, genericSymbolRef{
			File:    fmt.Sprintf("src/ref_%02d.ts", i),
			Line:    i,
			Snippet: "const ref = Widget",
			Class:   codeast.ClassRef,
		})
	}
	for i := 1; i <= genericTestLimit+2; i++ {
		refs = append(refs, genericSymbolRef{
			File:    fmt.Sprintf("src/Widget_%02d.test.tsx", i),
			Line:    i,
			Snippet: "render(<Widget />)",
			Class:   codeast.ClassCall,
			IsTest:  true,
		})
	}
	return def, refs
}

func semanticEvidenceASTDiagnosticsFixture() SymbolBundleDiagnostics {
	return SymbolBundleDiagnostics{
		ResolvedBy:       SymbolBundleResolvedByAST,
		RawRefCount:      intPtr(6),
		AcceptedRefCount: intPtr(5),
	}
}
