package gathercontext

import (
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

const maxPrefetchedReads = 3

type prefetchResult struct {
	output        string
	observation   *tools.RuntimeObservation
	discoveryNote string
}

type prefetchPolicy struct {
	shouldPrefetch bool
	limit          int
	diagnostics    *search.SymbolBundleDiagnostics
	reason         string
	limited        bool
}

func prefetchRecommendedEvidence(execCtx tools.ExecutionContext, artifact search.SearchExecutionArtifact) prefetchResult {
	policy := prefetchPolicyForArtifact(artifact.Metadata)
	if !policy.shouldPrefetch {
		if policy.reason != "" {
			return prefetchResult{discoveryNote: prefetchSkippedNote(policy.reason)}
		}
		return prefetchResult{}
	}
	items := boundedRecommendedReads(artifact.Metadata.Bundle.Impact.RecommendedReads, policy.limit)
	if len(items) == 0 {
		return prefetchResult{}
	}

	var sections []string
	var observations []*tools.RuntimeObservation
	reg := execCtx.EffectiveLocatorRegistry()
	for _, item := range items {
		target := registerPrefetchLocator(reg, item)
		if target == "" {
			continue
		}
		for _, section := range filetool.ExecuteReadTargetsWithDetailSections(execCtx, target, "compact") {
			if section.Failed || strings.TrimSpace(section.Output) == "" {
				continue
			}
			sections = append(sections, section.Output)
			observations = append(observations, section.Observation)
		}
	}
	if len(sections) == 0 {
		return prefetchResult{}
	}
	if policy.limited {
		sections = append([]string{prefetchLimitedNote(policy.reason)}, sections...)
	}
	return prefetchResult{
		output:      strings.Join(sections, "\n\n"),
		observation: tools.MergeRuntimeObservations(observations...),
	}
}

func prefetchPolicyForArtifact(metadata search.SearchExecutionMetadata) prefetchPolicy {
	limit, reason := prefetchLimitForDiagnostics(metadata.Diagnostics)
	policy := prefetchPolicy{
		limit:       limit,
		diagnostics: metadata.Diagnostics,
		limited:     limit < maxPrefetchedReads,
	}
	if !metadata.StructuredImpact {
		return policy
	}
	if metadata.Ambiguous {
		policy.reason = "ambiguous structured impact"
		return policy
	}
	if metadata.Bundle == nil || metadata.Bundle.Impact == nil || len(metadata.Bundle.Impact.RecommendedReads) == 0 {
		return policy
	}
	policy.shouldPrefetch = true
	policy.reason = reason
	return policy
}

func prefetchLimitForDiagnostics(diagnostics *search.SymbolBundleDiagnostics) (int, string) {
	limit := maxPrefetchedReads
	if diagnostics == nil {
		return limit, ""
	}

	var reasons []string
	if diagnosticFlagSet(diagnostics.Incomplete) {
		limit = min(limit, 1)
		reasons = append(reasons, "incomplete=true")
	}
	if diagnosticFlagSet(diagnostics.Truncated) {
		limit = min(limit, 1)
		reasons = append(reasons, "truncated=true")
	}
	if diagnosticFlagSet(diagnostics.BudgetLimitHit) {
		limit = min(limit, 1)
		reasons = append(reasons, "budget_limit_hit=true")
	}

	switch strings.TrimSpace(diagnostics.ResolvedBy) {
	case search.SymbolBundleResolvedByFallback, search.SymbolBundleResolvedByMixed:
		limit = min(limit, 1)
		reasons = append(reasons, "resolved_by="+strings.TrimSpace(diagnostics.ResolvedBy))
	}

	switch strings.TrimSpace(diagnostics.Confidence) {
	case search.SymbolBundleConfidenceLow:
		limit = min(limit, 1)
		reasons = append(reasons, "confidence="+search.SymbolBundleConfidenceLow)
	case search.SymbolBundleConfidenceMedium:
		limit = min(limit, 2)
		reasons = append(reasons, "confidence="+search.SymbolBundleConfidenceMedium)
	}

	if fallbackReason := strings.TrimSpace(diagnostics.FallbackReason); fallbackReason != "" && limit < maxPrefetchedReads {
		reasons = append(reasons, "fallback_reason="+fallbackReason)
	}

	return limit, strings.Join(reasons, ", ")
}

func diagnosticFlagSet(value *bool) bool {
	return value != nil && *value
}

func prefetchLimitedNote(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "Prefetch limited: diagnostics policy"
	}
	return "Prefetch limited: " + reason
}

func prefetchSkippedNote(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	return "Prefetch skipped: " + reason
}

func appendSearchDiscoveryNote(discovery, note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return discovery
	}
	discovery = strings.TrimSpace(discovery)
	if discovery == "" {
		return note
	}
	return discovery + "\n" + note
}

func boundedRecommendedReads(items []search.SymbolBundleItem, limit int) []search.SymbolBundleItem {
	if limit <= 0 || len(items) == 0 {
		return nil
	}
	result := make([]search.SymbolBundleItem, 0, min(limit, len(items)))
	seen := make(map[string]struct{}, limit)
	for _, item := range items {
		if item.File == "" || item.Line <= 0 {
			continue
		}
		key := item.File + "\x00" + item.ResolvedPath + "\x00" + item.Kind + "\x00" + item.Name + "\x00" + strconv.Itoa(item.Line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func registerPrefetchLocator(reg *locator.Registry, item search.SymbolBundleItem) string {
	if reg == nil {
		return ""
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = strings.TrimSpace(item.Kind)
	}
	return reg.Register(locator.Location{
		FilePath:     item.File,
		ResolvedPath: item.ResolvedPath,
		Line:         item.Line,
		EndLine:      item.EndLine,
		Name:         name,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
