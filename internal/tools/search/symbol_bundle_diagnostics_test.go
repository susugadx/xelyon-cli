package search

import (
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestSearchExecutionMetadataDiagnosticsCopiesLSPBundleDiagnostics(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser } from './build'\nbuildUser('semantic')\n",
	})
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.ts", Line: 2, Character: 1}},
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	bundleDiagnostics := artifact.Metadata.Bundle.Diagnostics
	assertDiagnosticsResolvedBy(t, bundleDiagnostics, symbolBundleResolvedByLSP)
	assertBoolPtr(t, bundleDiagnostics.LSPAttempted, true, "LSPAttempted")
	assertBoolPtr(t, bundleDiagnostics.LSPAvailable, true, "LSPAvailable")
	assertBoolPtr(t, bundleDiagnostics.FallbackUsed, false, "FallbackUsed")
	assertIntPtr(t, bundleDiagnostics.RawRefCount, 1, "RawRefCount")
	assertIntPtr(t, bundleDiagnostics.AcceptedRefCount, 1, "AcceptedRefCount")
	assertIntPtr(t, bundleDiagnostics.DroppedRefCount, 0, "DroppedRefCount")
	if bundleDiagnostics.Confidence != symbolBundleConfidenceHigh {
		t.Fatalf("Confidence = %q, want %q", bundleDiagnostics.Confidence, symbolBundleConfidenceHigh)
	}
	if artifact.Metadata.Diagnostics == nil {
		t.Fatal("Metadata.Diagnostics = nil, want bundle diagnostics snapshot")
	}
	if artifact.Metadata.Diagnostics == &artifact.Metadata.Bundle.Diagnostics {
		t.Fatal("Metadata.Diagnostics shares bundle diagnostics pointer, want defensive copy")
	}
	assertDiagnosticsResolvedBy(t, *artifact.Metadata.Diagnostics, symbolBundleResolvedByLSP)
}

func TestSearchExecutionMetadataDiagnosticsPreservedOnStructuredImpactCacheHit(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "import { buildUser } from './build'\nbuildUser('1')\n",
	})
	cache := &testSearchCache{data: make(map[string]string)}
	opts := newTypeScriptImpactSearchOptions(dir, "buildUser")

	first := ExecuteSearchCodeArtifactWithConfig(nil, cache, opts)
	assertTypeScriptStructuredImpactArtifact(t, first, "buildUser", "function")
	assertDiagnosticsResolvedBy(t, first.Metadata.Bundle.Diagnostics, symbolBundleResolvedByAST)

	second := ExecuteSearchCodeArtifactWithConfig(nil, cache, opts)
	assertTypeScriptStructuredImpactArtifact(t, second, "buildUser", "function")
	assertDiagnosticsResolvedBy(t, second.Metadata.Bundle.Diagnostics, symbolBundleResolvedByAST)
	if second.Metadata.Diagnostics == nil {
		t.Fatal("cached Metadata.Diagnostics = nil, want diagnostics snapshot")
	}
	assertDiagnosticsResolvedBy(t, *second.Metadata.Diagnostics, symbolBundleResolvedByAST)
}

func TestCloneSymbolBundleDeepCopiesDiagnosticsPointers(t *testing.T) {
	original := &SymbolBundle{
		Diagnostics: SymbolBundleDiagnostics{
			LSPAttempted:     boolPtr(true),
			RawRefCount:      intPtr(3),
			AcceptedRefCount: intPtr(2),
		},
	}

	cloned := cloneSymbolBundle(original)
	if cloned == nil {
		t.Fatal("cloneSymbolBundle() = nil")
	}
	*original.Diagnostics.LSPAttempted = false
	*original.Diagnostics.RawRefCount = 99
	*original.Diagnostics.AcceptedRefCount = 98

	assertBoolPtr(t, cloned.Diagnostics.LSPAttempted, true, "cloned LSPAttempted")
	assertIntPtr(t, cloned.Diagnostics.RawRefCount, 3, "cloned RawRefCount")
	assertIntPtr(t, cloned.Diagnostics.AcceptedRefCount, 2, "cloned AcceptedRefCount")
}

func TestFinalizeSymbolBundleDiagnosticsRecomputesConfidenceAfterSectionBudgetHit(t *testing.T) {
	bundle := &SymbolBundle{
		Diagnostics: SymbolBundleDiagnostics{
			ResolvedBy:     symbolBundleResolvedByAST,
			BudgetLimitHit: boolPtr(false),
			Confidence:     symbolBundleConfidenceMedium,
		},
		Sections: []SymbolBundleSection{
			{Kind: "references", More: true},
		},
	}

	finalizeSymbolBundleDiagnostics(bundle)

	assertBoolPtr(t, bundle.Diagnostics.BudgetLimitHit, true, "BudgetLimitHit")
	if bundle.Diagnostics.Confidence != symbolBundleConfidenceLow {
		t.Fatalf("Confidence = %q, want %q after final budget hit", bundle.Diagnostics.Confidence, symbolBundleConfidenceLow)
	}
}

func TestJSFamilyDiagnosticsRecordsLSPFallbackAndBudgetHit(t *testing.T) {
	t.Run("lsp error fallback", func(t *testing.T) {
		dir := setupMultiLangDir(t, map[string]string{
			"src/build.js": "function buildUser(id) { return id }\n",
			"src/app.js":   "buildUser('fallback')\n",
		})
		opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
		opts.LSPClient = &mockJSFamilyLSPClient{err: errors.New("lsp unavailable")}

		artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

		assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
		diag := artifact.Metadata.Bundle.Diagnostics
		assertDiagnosticsResolvedBy(t, diag, symbolBundleResolvedByMixed)
		assertBoolPtr(t, diag.LSPAttempted, true, "LSPAttempted")
		assertBoolPtr(t, diag.FallbackUsed, true, "FallbackUsed")
		if diag.FallbackReason != symbolBundleFallbackReasonLSPError {
			t.Fatalf("FallbackReason = %q, want %q", diag.FallbackReason, symbolBundleFallbackReasonLSPError)
		}
		if diag.Confidence != symbolBundleConfidenceLow {
			t.Fatalf("Confidence = %q, want %q", diag.Confidence, symbolBundleConfidenceLow)
		}
	})

	t.Run("ast budget", func(t *testing.T) {
		var source strings.Builder
		source.WriteString("export function buildUser(id) { return id }\n")
		for i := 0; i < maxGenericRefs+5; i++ {
			source.WriteString("buildUser('caller')\n")
		}
		dir := setupMultiLangDir(t, map[string]string{
			"src/build.js": source.String(),
		})

		artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

		assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
		diag := artifact.Metadata.Bundle.Diagnostics
		assertDiagnosticsResolvedBy(t, diag, symbolBundleResolvedByAST)
		assertBoolPtr(t, diag.Truncated, true, "Truncated")
		assertBoolPtr(t, diag.BudgetLimitHit, true, "BudgetLimitHit")
		if diag.Confidence != symbolBundleConfidenceLow {
			t.Fatalf("Confidence = %q, want %q", diag.Confidence, symbolBundleConfidenceLow)
		}
	})

	t.Run("ast alias expansion budget", func(t *testing.T) {
		dir := setupMultiLangDir(t, map[string]string{
			"src/Button.tsx": "export function Button() { return <button /> }\n",
			"src/AApp.tsx":   jsFamilyAliasBudgetSource(maxGenericRefs + 5),
		})

		artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

		assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
		diag := artifact.Metadata.Bundle.Diagnostics
		assertDiagnosticsResolvedBy(t, diag, symbolBundleResolvedByAST)
		assertBoolPtr(t, diag.Truncated, true, "Truncated")
		assertBoolPtr(t, diag.BudgetLimitHit, true, "BudgetLimitHit")
		if diag.RawRefCount == nil || diag.AcceptedRefCount == nil {
			t.Fatalf("ref counts = raw %v accepted %v, want non-nil alias diagnostics counts", diag.RawRefCount, diag.AcceptedRefCount)
		}
		if *diag.RawRefCount < *diag.AcceptedRefCount {
			t.Fatalf("RawRefCount = %d, AcceptedRefCount = %d, want raw count to include alias expansion refs", *diag.RawRefCount, *diag.AcceptedRefCount)
		}
		if diag.Confidence != symbolBundleConfidenceLow {
			t.Fatalf("Confidence = %q, want %q", diag.Confidence, symbolBundleConfidenceLow)
		}
	})
}

func TestImpactFallbackArtifactDiagnostics(t *testing.T) {
	setupSearchTestMocks(t)

	dir := setupMultiLangDir(t, map[string]string{
		"docs/builder.md":      "Builder\n",
		"docs/builder_impl.md": "BuilderImpl\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:     "Builder",
		Intent:      "impact",
		Path:        dir,
		FilePattern: "*.md",
	})

	if artifact.Metadata.Diagnostics == nil {
		t.Fatal("Metadata.Diagnostics = nil, want fallback diagnostics")
	}
	diag := *artifact.Metadata.Diagnostics
	assertDiagnosticsResolvedBy(t, diag, symbolBundleResolvedByFallback)
	assertBoolPtr(t, diag.FallbackUsed, true, "FallbackUsed")
	if diag.FallbackReason != symbolBundleFallbackReasonStructuredUnavailable {
		t.Fatalf("FallbackReason = %q, want %q", diag.FallbackReason, symbolBundleFallbackReasonStructuredUnavailable)
	}
	if diag.Confidence != symbolBundleConfidenceLow {
		t.Fatalf("Confidence = %q, want %q", diag.Confidence, symbolBundleConfidenceLow)
	}
}

func assertDiagnosticsResolvedBy(t *testing.T, diag SymbolBundleDiagnostics, want string) {
	t.Helper()
	if diag.ResolvedBy != want {
		t.Fatalf("ResolvedBy = %q, want %q", diag.ResolvedBy, want)
	}
}

func assertBoolPtr(t *testing.T, got *bool, want bool, name string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", name, *got, want)
	}
}

func assertIntPtr(t *testing.T, got *int, want int, name string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", name, *got, want)
	}
}
