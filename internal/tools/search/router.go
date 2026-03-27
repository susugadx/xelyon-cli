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

	symbolAllowed := opts.FilePattern == "" && lang != "" && isSymbolResolvableLanguage(lang)

	switch mode {
	case SearchModeRegex:
		trace.InitialLane = searchLaneRegex
		return trace
	case SearchModeLiteral:
		trace.InitialLane = searchLaneLiteral
		return trace
	case SearchModeSymbol:
		trace.FallbackLane = searchLaneLiteral
		trace.InitialLane = searchLaneSymbol
		trace.SymbolQuery = analysis.TrimmedPattern
		trace.SymbolCandidates = []string{analysis.TrimmedPattern}
		return trace
	default:
		trace.FallbackLane = analysis.defaultTextLane()
		if symbolAllowed && (analysis.LooksLikeBareIdentifier || analysis.LooksLikeDottedSymbol) {
			trace.InitialLane = searchLaneSymbol
			trace.SymbolQuery = analysis.TrimmedPattern
			trace.SymbolCandidates = []string{analysis.TrimmedPattern}
			return trace
		}
		if lang == "go" && opts.FilePattern == "" {
			if shouldTryGoSymbolRescue(pattern, analysis) {
				if candidates := extractGoSymbolRescueCandidates(pattern); len(candidates) > 0 {
					trace.InitialLane = searchLaneSymbol
					trace.SymbolQuery = candidates[0]
					trace.SymbolCandidates = candidates
					trace.SymbolRescue = true
					return trace
				}
			}
			if analysis.LooksLikeBareIdentifier || analysis.LooksLikeDottedSymbol {
				trace.InitialLane = searchLaneSymbol
				trace.SymbolQuery = analysis.TrimmedPattern
				trace.SymbolCandidates = []string{analysis.TrimmedPattern}
				return trace
			}
		}
		trace.InitialLane = analysis.defaultTextLane()
		return trace
	}
}

func (t searchRouteTrace) cacheSignature() string {
	return fmt.Sprintf("requested=%s|lang=%s|initial=%s|fallback=%s|symbol=%s|rescue=%t",
		t.RequestedMode, t.Language, t.InitialLane, t.FallbackLane, t.SymbolQuery, t.SymbolRescue)
}

func (t searchRouteTrace) textIsRegex() bool {
	return t.FinalLane == searchLaneRegex
}
