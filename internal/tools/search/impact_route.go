package search

import "strings"

func structuredImpactSymbolRoute(pattern string, opts SearchOptions, language string, decision string) (searchRouteTrace, bool) {
	analysis := analyzeSearchQuery(pattern)
	route := planSearchRoute(pattern, opts)
	if route.InitialLane == searchLaneSymbol {
		if route.Language == "" {
			route.Language = language
		}
		return route, true
	}

	mode := SearchMode(opts.Mode)
	if mode == SearchModeLiteral || mode == SearchModeRegex {
		return searchRouteTrace{}, false
	}

	if analysis.LooksLikeBareIdentifier || analysis.LooksLikeDottedSymbol {
		return newStructuredImpactSymbolRoute(mode, strings.TrimSpace(language), decision, analysis, analysis.TrimmedPattern, []string{analysis.TrimmedPattern}), true
	}

	if strings.TrimSpace(language) == "go" && shouldTryGoSymbolRescue(pattern, analysis) {
		if candidates := extractGoSymbolRescueCandidates(pattern); len(candidates) > 0 {
			return newStructuredImpactGoRescueRoute(mode, analysis, candidates), true
		}
	}

	return searchRouteTrace{}, false
}

func newStructuredImpactSymbolRoute(mode SearchMode, language string, decision string, analysis SearchQueryAnalysis, query string, candidates []string) searchRouteTrace {
	return assignRouteSymbolQuery(searchRouteTrace{
		RequestedMode: mode,
		Language:      language,
		FallbackLane:  analysis.defaultTextLane(),
		Decision:      decision,
		Analysis:      analysis,
	}, query, candidates)
}

func newStructuredImpactGoRescueRoute(mode SearchMode, analysis SearchQueryAnalysis, candidates []string) searchRouteTrace {
	return assignRouteSymbolQuery(searchRouteTrace{
		RequestedMode: mode,
		Language:      "go",
		FallbackLane:  analysis.defaultTextLane(),
		Decision:      "go-rescue",
		Analysis:      analysis,
		SymbolRescue:  true,
	}, candidates[0], candidates)
}
