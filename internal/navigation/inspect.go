package navigation

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// InspectSymbol は指定シンボルの定義・caller・ref・テストをまとめて返す。
func InspectSymbol(symbol, pathHint, mode string) string {
	if symbol == "" {
		return "Error: symbol is required"
	}

	inspectMode := ModeSummary
	if mode == "full" {
		inspectMode = ModeFull
	}

	budget := SummaryBudget
	if inspectMode == ModeFull {
		budget = FullBudget
	}

	_, output, status := resolveInspectSymbol(symbol, pathHint, inspectResolveOptions{
		budget: budget,
	})
	if status == SymbolAutoNone {
		return fmt.Sprintf("No symbol found: %q", symbol)
	}
	return output
}

func buildInspectResultForSingleCandidate(query symbolQuery, cand SymbolCandidate, opts inspectResolveOptions) InspectResult {
	result := InspectResult{Symbol: &cand}
	result.Body = readDefinitionBody(cand, opts.budget.BodyLines)

	references := collectInspectReferences(query.BaseName, cand, opts.referenceOptions())
	allRefs := references.refs
	result.Implementations = references.implementations
	result.ReferenceDiagnostics = references.diagnostics
	result.ResolvedViaLSP = references.diagnostics.ResolvedBy == "lsp"
	result.UpstreamTruncated = references.upstreamTruncated
	result.UpstreamIncomplete = references.upstreamIncomplete
	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, opts.budget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, opts.budget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(query.BaseName, allRefs, opts.budget.TestLimit)

	if opts.normalizePaths {
		normalizeInspectResultPaths(&result, opts.runtime)
	}
	return result
}

type inspectReferenceCollection struct {
	refs               []Reference
	implementations    []ImplementationRef
	diagnostics        InspectReferenceDiagnostics
	upstreamTruncated  bool
	upstreamIncomplete bool
}

func collectInspectReferences(baseName string, cand SymbolCandidate, opts inspectReferenceOptions) inspectReferenceCollection {
	collection := inspectReferenceCollection{
		diagnostics: InspectReferenceDiagnostics{
			ResolvedBy:     "fallback",
			FallbackUsed:   true,
			FallbackReason: "lsp_unavailable",
		},
	}
	if opts.lspClient != nil {
		collection.diagnostics = InspectReferenceDiagnostics{
			ResolvedBy:   "unknown",
			LSPAttempted: true,
			FallbackUsed: false,
		}
		if cand.Kind == "interface" {
			if impls, implErr := findImplementationsViaLSP(opts.lspClient, cand, opts.runtime.InvocationCWD); implErr == nil {
				collection.implementations = impls
			}
		}
		lspRefs, rawCount, err := findReferencesViaLSP(opts.lspClient, cand, opts.runtime.InvocationCWD, opts.referenceFilter)
		collection.diagnostics.RawRefCount = rawCount
		collection.diagnostics.AcceptedRefCount = len(lspRefs)
		collection.diagnostics.DroppedRefCount = maxInt(rawCount-len(lspRefs), 0)
		if err == nil && len(lspRefs) > 0 {
			collection.refs = lspRefs
			collection.diagnostics.ResolvedBy = "lsp"
			collection.diagnostics.LSPAvailable = true
			return collection
		}
		if err == nil {
			collection.diagnostics.LSPAvailable = true
			collection.diagnostics.FallbackReason = "lsp_empty"
		} else if lspReferenceErrorTimedOut(err) {
			collection.diagnostics.LSPTimedOut = true
			collection.diagnostics.FallbackReason = "lsp_timeout"
		} else {
			collection.diagnostics.FallbackReason = "lsp_error"
		}
	}

	refs, truncated, incomplete := findReferencesWithFallbackRuntime(baseName, cand, opts.runtime, opts.referenceFilter, opts.fallbackSearchPath)
	collection.refs = refs
	collection.upstreamTruncated = truncated
	collection.upstreamIncomplete = incomplete
	collection.diagnostics.FallbackUsed = true
	if collection.diagnostics.LSPAttempted {
		collection.diagnostics.ResolvedBy = "mixed"
	} else {
		collection.diagnostics.ResolvedBy = "fallback"
	}
	if collection.diagnostics.FallbackReason == "" {
		collection.diagnostics.FallbackReason = "lsp_unavailable"
	}
	collection.diagnostics.RawRefCount = len(refs)
	collection.diagnostics.AcceptedRefCount = len(refs)
	collection.diagnostics.DroppedRefCount = 0
	return collection
}

func lspReferenceErrorTimedOut(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "deadline") || strings.Contains(message, "timeout") || strings.Contains(message, "timed out")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
