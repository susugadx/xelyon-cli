package search

import "fmt"

type searchLane string

const (
	searchLaneSymbol  searchLane = "symbol"
	searchLaneLiteral searchLane = "literal"
	searchLaneRegex   searchLane = "regex"
)

// searchRouteTrace は router の判断と実行結果を保持する。
type searchRouteTrace struct {
	RequestedMode    SearchMode
	Language         string
	InitialLane      searchLane
	FinalLane        searchLane
	FallbackLane     searchLane
	SymbolQuery      string
	SymbolCandidates []string
	Decision         string
	SymbolRescue     bool
	SymbolAttempted  bool
	SymbolResolved   bool
	FallbackUsed     bool
	Analysis         SearchQueryAnalysis
}

func planSearchRoute(pattern string, opts SearchOptions) searchRouteTrace {
	analysis := analyzeSearchQuery(pattern)
	mode := SearchMode(opts.Mode)
	lang := resolveLanguage(opts)
	trace := searchRouteTrace{
		RequestedMode: mode,
		Language:      lang,
		Analysis:      analysis,
	}

	if explicit, ok := planExplicitSearchRoute(trace, analysis); ok {
		return explicit
	}
	return planAutoSearchRoute(trace, pattern, opts, analysis)
}

func planExplicitSearchRoute(trace searchRouteTrace, analysis SearchQueryAnalysis) (searchRouteTrace, bool) {
	switch trace.RequestedMode {
	case SearchModeRegex:
		trace.InitialLane = searchLaneRegex
		trace.Decision = "explicit-regex"
		return trace, true
	case SearchModeLiteral:
		trace.InitialLane = searchLaneLiteral
		trace.Decision = "explicit-literal"
		return trace, true
	case SearchModeSymbol:
		trace.FallbackLane = searchLaneLiteral
		trace = assignRouteSymbolQuery(trace, analysis.TrimmedPattern, []string{analysis.TrimmedPattern})
		trace.Decision = "explicit-symbol"
		return trace, true
	default:
		return searchRouteTrace{}, false
	}
}

func planAutoSearchRoute(trace searchRouteTrace, pattern string, opts SearchOptions, analysis SearchQueryAnalysis) searchRouteTrace {
	trace.FallbackLane = analysis.defaultTextLane()
	if symbolSearchAllowed(opts, trace.Language) && (analysis.LooksLikeBareIdentifier || analysis.LooksLikeDottedSymbol) {
		trace = assignRouteSymbolQuery(trace, analysis.TrimmedPattern, []string{analysis.TrimmedPattern})
		trace.Decision = "auto-symbol"
		return trace
	}
	if goSpecific, ok := planGoSpecificAutoSearchRoute(trace, pattern, opts, analysis); ok {
		return goSpecific
	}
	trace.InitialLane = analysis.defaultTextLane()
	trace.Decision = "text"
	return trace
}

func planGoSpecificAutoSearchRoute(trace searchRouteTrace, pattern string, opts SearchOptions, analysis SearchQueryAnalysis) (searchRouteTrace, bool) {
	if trace.Language != "go" || opts.FilePattern != "" {
		return searchRouteTrace{}, false
	}
	if shouldTryGoSymbolRescue(pattern, analysis) {
		if candidates := extractGoSymbolRescueCandidates(pattern); len(candidates) > 0 {
			trace = assignRouteSymbolQuery(trace, candidates[0], candidates)
			trace.SymbolRescue = true
			trace.Decision = "go-rescue"
			return trace, true
		}
	}
	if analysis.LooksLikeBareIdentifier || analysis.LooksLikeDottedSymbol {
		trace = assignRouteSymbolQuery(trace, analysis.TrimmedPattern, []string{analysis.TrimmedPattern})
		trace.Decision = "go-symbol"
		return trace, true
	}
	return searchRouteTrace{}, false
}

func assignRouteSymbolQuery(trace searchRouteTrace, query string, candidates []string) searchRouteTrace {
	trace.InitialLane = searchLaneSymbol
	trace.SymbolQuery = query
	trace.SymbolCandidates = candidates
	return trace
}

func symbolSearchAllowed(opts SearchOptions, language string) bool {
	return opts.FilePattern == "" && language != "" && isSymbolResolvableLanguage(language)
}

func (t searchRouteTrace) cacheSignature() string {
	return fmt.Sprintf("requested=%s|lang=%s|initial=%s|fallback=%s|symbol=%s|rescue=%t",
		t.RequestedMode, t.Language, t.InitialLane, t.FallbackLane, t.SymbolQuery, t.SymbolRescue)
}

func (t searchRouteTrace) textIsRegex() bool {
	return t.FinalLane == searchLaneRegex
}
