package search

import "testing"

func TestBuildSymbolBundleFromSemanticEvidenceBuildsBundle(t *testing.T) {
	diagnostics := &SymbolBundleDiagnostics{
		ResolvedBy:       SymbolBundleResolvedByAST,
		RawRefCount:      intPtr(4),
		AcceptedRefCount: intPtr(3),
	}
	evidence := SemanticEvidence{
		Language: "typescript",
		Query:    "buildUser",
		Symbol:   "buildUser",
		Definitions: []SemanticDefinition{{
			Name:      "buildUser",
			Kind:      "function",
			File:      "src/build.ts",
			Line:      2,
			EndLine:   4,
			Signature: "export function buildUser(id: string) { return id }",
			Body: []string{
				"2: export function buildUser(id: string) {",
				"3:   return id",
				"4: }",
			},
		}},
		References: []SemanticReference{
			{Kind: SemanticReferenceKindReference, File: "src/readme.ts", Line: 1, Snippet: "buildUser"},
			{Kind: SemanticReferenceKindTest, File: "src/build.test.ts", Line: 5, Snippet: "buildUser('test')"},
			{Kind: SemanticReferenceKindCaller, File: "src/app.ts", Line: 8, Snippet: "buildUser('app')", Scope: "main"},
		},
		Diagnostics: diagnostics,
	}

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence() ok = false")
	}

	if bundle.Identity.Language != "typescript" || bundle.Identity.DisplayName != "buildUser" {
		t.Fatalf("identity = %+v, want typescript buildUser", bundle.Identity)
	}
	if bundle.Definition.File != "src/build.ts" || bundle.Definition.Line != 2 || bundle.Definition.EndLine != 4 {
		t.Fatalf("definition = %+v, want src/build.ts:2-4", bundle.Definition)
	}
	if callers := symbolBundleSectionByKind(bundle, SemanticReferenceSectionKindCallers); callers == nil || len(callers.Items) != 1 || callers.Items[0].Scope != "main" {
		t.Fatalf("callers section = %+v, want one caller with scope", callers)
	}
	if tests := symbolBundleSectionByKind(bundle, SemanticReferenceSectionKindTests); tests == nil || len(tests.Items) != 1 || !tests.Items[0].IsTest {
		t.Fatalf("tests section = %+v, want one test item", tests)
	}
	if refs := symbolBundleSectionByKind(bundle, SemanticReferenceSectionKindReferences); refs == nil || len(refs.Items) != 1 {
		t.Fatalf("references section = %+v, want one reference item", refs)
	}
	assertDiagnosticsResolvedBy(t, bundle.Diagnostics, SymbolBundleResolvedByAST)
	assertIntPtr(t, bundle.Diagnostics.RawRefCount, 4, "RawRefCount")
	assertIntPtr(t, bundle.Diagnostics.AcceptedRefCount, 3, "AcceptedRefCount")
	if bundle.Diagnostics.Confidence != SymbolBundleConfidenceMedium {
		t.Fatalf("Confidence = %q, want %q", bundle.Diagnostics.Confidence, SymbolBundleConfidenceMedium)
	}

	reads := bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/build.ts")
	assertRecommendedReadAt(t, reads, 1, "callers", "src/app.ts")
	assertRecommendedReadAt(t, reads, 2, "tests", "src/build.test.ts")
	assertRecommendedReadAt(t, reads, 3, "references", "src/readme.ts")
	if bundle.Impact.RiskLevel != "" {
		t.Fatalf("RiskLevel = %q, want empty", bundle.Impact.RiskLevel)
	}
}

func TestBuildSymbolBundleFromSemanticEvidenceDefensivelyCopiesEvidence(t *testing.T) {
	rawRefCount := 2
	diagnostics := &SymbolBundleDiagnostics{
		ResolvedBy:  SymbolBundleResolvedByLSP,
		RawRefCount: &rawRefCount,
	}
	body := []string{"10: func Build() {}"}
	refs := []SemanticReference{{
		Kind:    SemanticReferenceKindCaller,
		File:    "pkg/app.go",
		Line:    12,
		Snippet: "Build()",
	}}
	sections := []SemanticReferenceSection{{Kind: SemanticReferenceSectionKindCallers, Total: 5, More: true}}
	evidence := SemanticEvidence{
		Language: "go",
		Symbol:   "Build",
		Definitions: []SemanticDefinition{{
			Name:      "Build",
			Kind:      "function",
			File:      "pkg/build.go",
			Line:      10,
			Signature: "func Build()",
			Body:      body,
		}},
		References:        refs,
		ReferenceSections: sections,
		Diagnostics:       diagnostics,
	}

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence() ok = false")
	}

	body[0] = "10: func Mutated() {}"
	refs[0].Snippet = "Mutated()"
	sections[0].Total = 1
	sections[0].More = false
	diagnostics.ResolvedBy = SymbolBundleResolvedByFallback
	rawRefCount = 99

	if got := bundle.Definition.Body[0]; got != "10: func Build() {}" {
		t.Fatalf("bundle definition body mutated to %q", got)
	}
	callers := symbolBundleSectionByKind(bundle, SemanticReferenceSectionKindCallers)
	if callers == nil || callers.Items[0].Snippet != "Build()" {
		t.Fatalf("caller section after evidence mutation = %+v, want original snippet", callers)
	}
	if callers.Total != 5 || !callers.More {
		t.Fatalf("caller section total/more after evidence mutation = %d/%v, want 5/true", callers.Total, callers.More)
	}
	assertDiagnosticsResolvedBy(t, bundle.Diagnostics, SymbolBundleResolvedByLSP)
	assertIntPtr(t, bundle.Diagnostics.RawRefCount, 2, "bundle RawRefCount")
	if bundle.Diagnostics.RawRefCount == diagnostics.RawRefCount {
		t.Fatal("bundle diagnostics RawRefCount shares pointer with evidence diagnostics")
	}
}

func TestCloneBundleDiagnosticsForMetadataSnapshotsSemanticEvidenceBundle(t *testing.T) {
	bundle, ok := buildSymbolBundleFromSemanticEvidence(SemanticEvidence{
		Language: "go",
		Symbol:   "Build",
		Definitions: []SemanticDefinition{{
			Name: "Build",
			File: "pkg/build.go",
			Line: 10,
		}},
		Diagnostics: &SymbolBundleDiagnostics{
			ResolvedBy:       SymbolBundleResolvedByAST,
			RawRefCount:      intPtr(5),
			AcceptedRefCount: intPtr(4),
		},
	})
	if !ok {
		t.Fatal("buildSymbolBundleFromSemanticEvidence() ok = false")
	}

	metadata := cloneBundleDiagnosticsForMetadata(bundle)
	if metadata == nil {
		t.Fatal("cloneBundleDiagnosticsForMetadata() = nil")
	}
	bundle.Diagnostics.ResolvedBy = SymbolBundleResolvedByFallback
	*bundle.Diagnostics.RawRefCount = 99

	assertDiagnosticsResolvedBy(t, *metadata, SymbolBundleResolvedByAST)
	assertIntPtr(t, metadata.RawRefCount, 5, "metadata RawRefCount")
}

func TestBuildSymbolBundleFromSemanticEvidenceRejectsInvalidDefinition(t *testing.T) {
	tests := []struct {
		name     string
		evidence SemanticEvidence
	}{
		{
			name:     "missing definition",
			evidence: SemanticEvidence{Language: "go", Symbol: "Build"},
		},
		{
			name: "missing file",
			evidence: SemanticEvidence{
				Language:    "go",
				Symbol:      "Build",
				Definitions: []SemanticDefinition{{Name: "Build", Line: 1}},
			},
		},
		{
			name: "missing line",
			evidence: SemanticEvidence{
				Language:    "go",
				Symbol:      "Build",
				Definitions: []SemanticDefinition{{Name: "Build", File: "pkg/build.go"}},
			},
		},
		{
			name: "missing symbol identity",
			evidence: SemanticEvidence{
				Language:    "go",
				Query:       "Build",
				Definitions: []SemanticDefinition{{File: "pkg/build.go", Line: 1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if bundle, ok := buildSymbolBundleFromSemanticEvidence(tt.evidence); ok || bundle != nil {
				t.Fatalf("buildSymbolBundleFromSemanticEvidence() = (%+v, %v), want nil false", bundle, ok)
			}
		})
	}
}
