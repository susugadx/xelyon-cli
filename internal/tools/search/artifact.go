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
	Bundle           *SymbolBundle
	AffectedFiles    []string
	StructuredImpact bool
	Ambiguous        bool
	MultiPattern     bool
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
		return SearchExecutionArtifact{
			Rendered: executeSearchPatterns(cache, patterns, opts),
			Metadata: SearchExecutionMetadata{
				MultiPattern: true,
			},
		}
	}

	result := executeSinglePatternDetailed(cache, patterns[0], opts)
	return SearchExecutionArtifact{
		Rendered: result.Output,
		Metadata: SearchExecutionMetadata{
			Bundle:        result.Bundle,
			AffectedFiles: append([]string(nil), result.AffectedFiles...),
			Ambiguous:     result.Route.SymbolResolved && result.Bundle == nil && result.Route.FinalLane == searchLaneSymbol,
		},
	}
}

func executeImpactSearchArtifact(cache tools.ToolCacheInterface, opts SearchOptions) SearchExecutionArtifact {
	if result, ok := tryStructuredImpactSearchResultForIntent(cache, opts); ok {
		return newStructuredImpactSearchArtifact(result)
	}

	basePatterns := expandImpactPatterns(strings.TrimSpace(opts.Pattern), opts)
	if len(basePatterns) == 0 {
		return SearchExecutionArtifact{Rendered: "Error: pattern is required"}
	}

	baseOutput := executeSearchPatterns(cache, basePatterns, opts)
	if impactOutputHasTestCoverage(baseOutput) || len(basePatterns) >= impactPatternExpansionCap {
		return SearchExecutionArtifact{
			Rendered: baseOutput,
			Metadata: SearchExecutionMetadata{
				MultiPattern: len(basePatterns) > 1,
			},
		}
	}

	testProbe := impactTestProbePattern(opts.Pattern)
	if testProbe == "" {
		return SearchExecutionArtifact{
			Rendered: baseOutput,
			Metadata: SearchExecutionMetadata{
				MultiPattern: len(basePatterns) > 1,
			},
		}
	}
	for _, existing := range basePatterns {
		if existing == testProbe {
			return SearchExecutionArtifact{
				Rendered: baseOutput,
				Metadata: SearchExecutionMetadata{
					MultiPattern: len(basePatterns) > 1,
				},
			}
		}
	}

	finalPatterns := append(append([]string(nil), basePatterns...), testProbe)
	return SearchExecutionArtifact{
		Rendered: executeSearchPatterns(cache, finalPatterns, opts),
		Metadata: SearchExecutionMetadata{
			MultiPattern: len(finalPatterns) > 1,
		},
	}
}
