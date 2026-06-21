package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type goSymbolBundleBuildOptions struct {
	implementationLimit int
	impact              *SymbolBundleImpact
}

var searchCodeGoSymbolBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 3,
	RefLimit:    3,
	TestLimit:   2,
}

const goImplementationLimit = 4
const goSymbolBundleLSPSource = "gopls"

func buildGoSymbolBundle(query string, result navigation.InspectResult) *SymbolBundle {
	return buildGoSymbolBundleWithOptions(query, result, goSymbolBundleBuildOptions{
		implementationLimit: goImplementationLimit,
	})
}

func buildGoSymbolBundleWithOptions(query string, result navigation.InspectResult, opts goSymbolBundleBuildOptions) *SymbolBundle {
	if result.Symbol == nil {
		return nil
	}
	if opts.implementationLimit <= 0 {
		opts.implementationLimit = goImplementationLimit
	}

	bundle := newGoSymbolBundle(query, result, opts)

	if len(result.Body) > 0 {
		bundle.Definition.Signature = result.Body[0]
	}

	addNavigationSection(bundle, "callers", "Callers", result.Callers, result.TotalCallers, result.MoreCallers)
	addNavigationSection(bundle, "references", "References", result.Refs, result.TotalRefs, result.MoreRefs)
	addNavigationTestSection(bundle, result.Tests, result.TotalTests, result.MoreTests)
	appendGoImplementationSection(bundle, result.Implementations, opts.implementationLimit)
	finalizeSymbolBundleDiagnostics(bundle)

	return bundle
}

func newGoSymbolBundle(query string, result navigation.InspectResult, opts goSymbolBundleBuildOptions) *SymbolBundle {
	displayName := goSymbolBundleDisplayName(*result.Symbol)
	return &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "go",
			Query:       query,
			Canonical:   canonicalGoSymbolBundleKey(*result.Symbol),
			DisplayName: displayName,
			Kind:        result.Symbol.Kind,
			File:        result.Symbol.File,
			Line:        result.Symbol.Line,
			EndLine:     result.Symbol.EndLine,
		},
		Diagnostics: goSymbolBundleDiagnostics(result),
		Definition: SymbolBundleDefinition{
			File:    result.Symbol.File,
			Line:    result.Symbol.Line,
			EndLine: result.Symbol.EndLine,
			Body:    append([]string(nil), result.Body...),
		},
		Impact: cloneSymbolBundleImpact(opts.impact),
		Debug: SymbolBundleDebug{
			Source:       "go-inspect",
			FileRootPath: result.Symbol.RootPath,
		},
	}
}

func goSymbolBundleDiagnostics(result navigation.InspectResult) SymbolBundleDiagnostics {
	refDiagnostics := result.ReferenceDiagnostics
	hasRefDetails := inspectReferenceDiagnosticsHasDetails(refDiagnostics)
	resolvedBy := strings.TrimSpace(refDiagnostics.ResolvedBy)
	if resolvedBy == "" {
		if result.ResolvedViaLSP {
			resolvedBy = symbolBundleResolvedByLSP
		} else {
			resolvedBy = symbolBundleResolvedByUnknown
		}
	}

	diagnostics := SymbolBundleDiagnostics{
		ResolvedBy:         resolvedBy,
		ResolvedViaLSP:     resolvedBy == symbolBundleResolvedByLSP,
		UpstreamTruncated:  result.UpstreamTruncated,
		UpstreamIncomplete: result.UpstreamIncomplete,
	}
	if diagnostics.ResolvedViaLSP || refDiagnostics.LSPAttempted {
		diagnostics.LSPSource = goSymbolBundleLSPSource
	}
	if hasRefDetails {
		diagnostics.LSPAttempted = boolPtr(refDiagnostics.LSPAttempted)
		diagnostics.LSPAvailable = boolPtr(refDiagnostics.LSPAvailable)
		diagnostics.LSPTimedOut = boolPtr(refDiagnostics.LSPTimedOut)
		diagnostics.FallbackUsed = boolPtr(refDiagnostics.FallbackUsed)
		diagnostics.FallbackReason = refDiagnostics.FallbackReason
		diagnostics.Incomplete = boolPtr(result.UpstreamIncomplete)
		diagnostics.Truncated = boolPtr(result.UpstreamTruncated)
		diagnostics.BudgetLimitHit = boolPtr(inspectResultBudgetLimitHit(result))
		updateDiagnosticsRefCounts(&diagnostics, refDiagnostics.RawRefCount, refDiagnostics.AcceptedRefCount)
		if refDiagnostics.DroppedRefCount >= 0 {
			diagnostics.DroppedRefCount = intPtr(refDiagnostics.DroppedRefCount)
		}
	} else {
		if result.ResolvedViaLSP {
			diagnostics.LSPAttempted = boolPtr(true)
			diagnostics.LSPAvailable = boolPtr(true)
		}
		if result.UpstreamIncomplete {
			diagnostics.Incomplete = boolPtr(true)
		}
		if result.UpstreamTruncated {
			diagnostics.Truncated = boolPtr(true)
		}
		if inspectResultBudgetLimitHit(result) {
			diagnostics.BudgetLimitHit = boolPtr(true)
		}
	}
	diagnostics.Confidence = inferSymbolBundleConfidence(diagnostics)
	normalizeSymbolBundleDiagnostics(&diagnostics)
	return diagnostics
}

func inspectReferenceDiagnosticsHasDetails(diag navigation.InspectReferenceDiagnostics) bool {
	return diag.LSPAttempted ||
		diag.LSPAvailable ||
		diag.LSPTimedOut ||
		diag.FallbackUsed ||
		strings.TrimSpace(diag.FallbackReason) != "" ||
		diag.RawRefCount != 0 ||
		diag.AcceptedRefCount != 0 ||
		diag.DroppedRefCount != 0
}

func inspectResultBudgetLimitHit(result navigation.InspectResult) bool {
	return result.MoreCallers || result.MoreRefs || result.MoreTests
}

func goSymbolBundleDisplayName(symbol navigation.SymbolCandidate) string {
	if symbol.Kind != "method" || symbol.Receiver == "" {
		return symbol.Name
	}
	if strings.HasPrefix(symbol.Receiver, "*") {
		return fmt.Sprintf("(%s).%s", symbol.Receiver, symbol.Name)
	}
	return symbol.Receiver + "." + symbol.Name
}

func appendGoImplementationSection(bundle *SymbolBundle, impls []navigation.ImplementationRef, implementationLimit int) {
	if len(impls) == 0 || implementationLimit <= 0 {
		return
	}

	limit := min(implementationLimit, len(impls))
	items := make([]SymbolBundleItem, 0, limit)
	for _, impl := range impls[:limit] {
		items = append(items, SymbolBundleItem{
			Kind:         "implementations",
			File:         impl.File,
			ResolvedPath: impl.ResolvedPath,
			Line:         impl.Line,
			Snippet:      strings.TrimSpace(impl.Name),
			Name:         impl.Name,
		})
	}

	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  "implementations",
		Title: "Related Implementations",
		Items: items,
		Total: len(impls),
		More:  len(impls) > len(items),
	})
}

func addNavigationSection(bundle *SymbolBundle, kind, title string, refs []navigation.Reference, total int, more bool) {
	if len(refs) == 0 {
		return
	}
	items := make([]SymbolBundleItem, 0, len(refs))
	for _, ref := range refs {
		snippet := strings.TrimSpace(ref.Snippet)
		if snippet == "" && ref.Scope != "" {
			snippet = ref.Scope
		}
		items = append(items, SymbolBundleItem{
			Kind:         kind,
			File:         ref.File,
			ResolvedPath: ref.ResolvedPath,
			Line:         ref.Line,
			Snippet:      snippet,
			Scope:        ref.Scope,
			IsTest:       ref.IsTest,
		})
	}
	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  kind,
		Title: title,
		Items: items,
		Total: total,
		More:  more,
	})
}

func addNavigationTestSection(bundle *SymbolBundle, tests []navigation.TestRef, total int, more bool) {
	if len(tests) == 0 {
		return
	}
	items := make([]SymbolBundleItem, 0, len(tests))
	for _, test := range tests {
		items = append(items, SymbolBundleItem{
			Kind:         "tests",
			File:         test.File,
			ResolvedPath: test.ResolvedPath,
			Line:         test.Line,
			Name:         test.Name,
			IsTest:       true,
		})
	}
	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  "tests",
		Title: "Related Tests",
		Items: items,
		Total: total,
		More:  more,
	})
}
