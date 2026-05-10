package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// SearchExecutionArtifact は高レベル orchestration 向けの検索実行結果を表す。
type SearchExecutionArtifact struct {
	Rendered string
	Metadata SearchExecutionMetadata
}

// SearchExecutionMetadata は orchestration が参照する検索実行 metadata を表す。
type SearchExecutionMetadata struct {
	Bundle              *SymbolBundle
	AffectedFiles       []string
	Observation         *tools.RuntimeObservation
	PatternObservations map[string]*tools.RuntimeObservation
	StructuredImpact    bool
	Ambiguous           bool
	MultiPattern        bool
}

// AnalyzeQuery は search_code と同じクエリ解析を返す。
func AnalyzeQuery(pattern string) SearchQueryAnalysis {
	return analyzeSearchQuery(pattern)
}

// HasMultiplePatterns はカンマ区切り multi-pattern かを返す。
func HasMultiplePatterns(pattern string) bool {
	return len(splitPatterns(pattern)) > 1
}

// ShouldPreferImpactIntent は通常 investigation で impact routing を優先すべきか返す。
func ShouldPreferImpactIntent(pattern string) bool {
	if HasMultiplePatterns(pattern) {
		return false
	}
	analysis := analyzeSearchQuery(pattern)
	if strings.TrimSpace(analysis.TrimmedPattern) == "" {
		return false
	}
	return analysis.LooksLikeBareIdentifier || analysis.LooksLikeDottedSymbol
}

// ExecuteSearchCodeArtifactWithConfig はテキスト出力に加えて orchestration 用 metadata を返す。
func ExecuteSearchCodeArtifactWithConfig(cfg *config.Config, cache tools.ToolCacheInterface, opts SearchOptions) SearchExecutionArtifact {
	opts, errResult := prepareSearchOptionsForRuntime(cfg, opts)
	if errResult != "" {
		return SearchExecutionArtifact{Rendered: errResult}
	}

	if shouldExecuteImpactSearch(opts) {
		return executeImpactSearchArtifact(cache, opts)
	}

	patterns := effectiveSearchPatterns(opts)
	if len(patterns) > 1 {
		result := executeSearchPatternsDetailed(cache, patterns, opts)
		return searchPatternsArtifact(result, true)
	}

	result := executeSinglePatternDetailed(cache, patterns[0], opts)
	return singlePatternArtifact(patterns[0], result)
}

func executeImpactSearchArtifact(cache tools.ToolCacheInterface, opts SearchOptions) SearchExecutionArtifact {
	if artifact, ok := tryStructuredGoImpactSearchArtifact(cache, opts); ok {
		return artifact
	}

	basePatterns := expandImpactPatterns(strings.TrimSpace(opts.Pattern), opts)
	if len(basePatterns) == 0 {
		return SearchExecutionArtifact{Rendered: "Error: pattern is required"}
	}

	baseResult := executeSearchPatternsDetailed(cache, basePatterns, opts)
	if impactOutputHasTestCoverage(baseResult.Output) || len(basePatterns) >= impactPatternExpansionCap {
		return searchPatternsArtifact(baseResult, len(basePatterns) > 1)
	}

	testProbe := impactTestProbePattern(opts.Pattern)
	if testProbe == "" {
		return searchPatternsArtifact(baseResult, len(basePatterns) > 1)
	}
	for _, existing := range basePatterns {
		if existing == testProbe {
			return searchPatternsArtifact(baseResult, len(basePatterns) > 1)
		}
	}

	finalPatterns := append(append([]string(nil), basePatterns...), testProbe)
	finalResult := executeSearchPatternsDetailed(cache, finalPatterns, opts)
	return searchPatternsArtifact(finalResult, len(finalPatterns) > 1)
}

func tryStructuredGoImpactSearchArtifact(cache tools.ToolCacheInterface, opts SearchOptions) (SearchExecutionArtifact, bool) {
	ctx, ok := newStructuredGoImpactSearchContext(opts)
	if !ok {
		return SearchExecutionArtifact{}, false
	}

	if cached, ok := loadStructuredGoImpactCachedResult(cache, ctx, opts); ok {
		return structuredImpactArtifact(cached.Output, cached.Bundle, cached.AffectedFiles, cached.Observation, cached.Bundle == nil), true
	}

	resolved, ok := resolveStructuredGoImpactWithContext(cache, ctx, opts)
	if !ok {
		return SearchExecutionArtifact{}, false
	}
	return structuredImpactArtifact(resolved.Rendered, resolved.Bundle, resolved.AffectedFiles, resolved.Observation, resolved.Ambiguous), true
}

func searchPatternsArtifact(result searchPatternsExecution, multiPattern bool) SearchExecutionArtifact {
	return SearchExecutionArtifact{
		Rendered: result.Output,
		Metadata: SearchExecutionMetadata{
			AffectedFiles:       result.AffectedFiles,
			Observation:         result.Observation,
			PatternObservations: result.PatternObservations,
			MultiPattern:        multiPattern,
		},
	}
}

func singlePatternArtifact(pattern string, result singlePatternExecution) SearchExecutionArtifact {
	return SearchExecutionArtifact{
		Rendered: result.Output,
		Metadata: SearchExecutionMetadata{
			Bundle:              result.Bundle,
			AffectedFiles:       append([]string(nil), result.AffectedFiles...),
			Observation:         tools.CloneRuntimeObservation(result.Observation),
			PatternObservations: singlePatternObservationMap(pattern, result.Observation),
			Ambiguous:           result.Route.SymbolResolved && result.Bundle == nil && result.Route.FinalLane == searchLaneSymbol,
		},
	}
}

func structuredImpactArtifact(rendered string, bundle *SymbolBundle, affectedFiles []string, observation *tools.RuntimeObservation, ambiguous bool) SearchExecutionArtifact {
	return SearchExecutionArtifact{
		Rendered: rendered,
		Metadata: SearchExecutionMetadata{
			Bundle:           bundle,
			AffectedFiles:    affectedFiles,
			Observation:      observation,
			StructuredImpact: true,
			Ambiguous:        ambiguous,
		},
	}
}
