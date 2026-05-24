package gathercontext

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func TestBuildSearchExecutionResult_KeepsDiagnosticsSummaryAndLimitsLowConfidencePrefetch(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "definition.go"): "package sample\n\nfunc Build() {}\n",
		filepath.Join(root, "caller.go"):     "package sample\n\nfunc Use() { Build() }\n",
	})

	result := buildSearchExecutionResult(
		newGatherContextExecCtx(root),
		searchPlan{query: "Build", preferImpact: true},
		search.SearchExecutionArtifact{
			Rendered: "structured discovery\nDiagnostics: resolved_by=mixed, confidence=low, fallback_reason=lsp_error",
			Metadata: search.SearchExecutionMetadata{
				StructuredImpact: true,
				Diagnostics: &search.SymbolBundleDiagnostics{
					ResolvedBy:     search.SymbolBundleResolvedByMixed,
					Confidence:     search.SymbolBundleConfidenceLow,
					FallbackReason: search.SymbolBundleFallbackReasonLSPError,
				},
				Bundle: &search.SymbolBundle{
					Impact: &search.SymbolBundleImpact{
						RecommendedReads: []search.SymbolBundleItem{
							{
								Kind:         "definition",
								File:         "definition.go",
								ResolvedPath: filepath.Join(root, "definition.go"),
								Line:         3,
								EndLine:      3,
								Name:         "Build",
							},
							{
								Kind:         "callers",
								File:         "caller.go",
								ResolvedPath: filepath.Join(root, "caller.go"),
								Line:         3,
								EndLine:      3,
								Name:         "Use",
							},
						},
					},
				},
			},
		},
	)
	output := formatExecutionResult(result)

	assertGatherContextContainsAll(t, output,
		"Search / Discovery",
		"Diagnostics: resolved_by=mixed, confidence=low, fallback_reason=lsp_error",
		"Prefetched Evidence",
		"Prefetch limited:",
		"resolved_by=mixed",
		"confidence=low",
		"fallback_reason=lsp_error",
		"definition.go",
	)
	assertGatherContextExcludesAll(t, output, "caller.go")
	assertGatherContextPrefetchedFileCount(t, output, 1)
}

func TestBuildSearchExecutionResult_AmbiguousKeepsDiagnosticsSummaryAndSkipsPrefetch(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "definition.go"): "package sample\n\nfunc Build() {}\n",
	})

	result := buildSearchExecutionResult(
		newGatherContextExecCtx(root),
		searchPlan{query: "Build", preferImpact: true},
		search.SearchExecutionArtifact{
			Rendered: "Multiple symbols matched \"Build\":\nDiagnostics: resolved_by=mixed, confidence=low",
			Metadata: search.SearchExecutionMetadata{
				StructuredImpact: true,
				Ambiguous:        true,
				Diagnostics: &search.SymbolBundleDiagnostics{
					ResolvedBy: search.SymbolBundleResolvedByMixed,
					Confidence: search.SymbolBundleConfidenceLow,
				},
				Bundle: &search.SymbolBundle{
					Impact: &search.SymbolBundleImpact{
						RecommendedReads: []search.SymbolBundleItem{{
							Kind:         "definition",
							File:         "definition.go",
							ResolvedPath: filepath.Join(root, "definition.go"),
							Line:         3,
							EndLine:      3,
							Name:         "Build",
						}},
					},
				},
			},
		},
	)
	output := formatExecutionResult(result)

	assertGatherContextContainsAll(t, output,
		`Multiple symbols matched "Build":`,
		"Diagnostics: resolved_by=mixed, confidence=low",
		"Prefetch skipped: ambiguous structured impact",
	)
	assertGatherContextExcludesAll(t, output, "Prefetched Evidence", "Prefetch limited:")
	if count := strings.Count(output, "Diagnostics:"); count != 1 {
		t.Fatalf("Diagnostics line count = %d, want 1:\n%s", count, output)
	}
}
