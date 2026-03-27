package search

import "strings"

// SearchMode は search_code の公開モード。
type SearchMode string

const (
	SearchModeAuto    SearchMode = "auto"
	SearchModeSymbol  SearchMode = "symbol"
	SearchModeLiteral SearchMode = "literal"
	SearchModeRegex   SearchMode = "regex"
)

func normalizeSearchMode(mode string, legacyIsRegex bool, legacyIsRegexSet bool) (SearchMode, bool) {
	if trimmed := strings.TrimSpace(strings.ToLower(mode)); trimmed != "" {
		switch SearchMode(trimmed) {
		case SearchModeAuto, SearchModeSymbol, SearchModeLiteral, SearchModeRegex:
			return SearchMode(trimmed), true
		default:
			return "", false
		}
	}

	if legacyIsRegexSet {
		if legacyIsRegex {
			return SearchModeRegex, true
		}
		return SearchModeLiteral, true
	}
	if legacyIsRegex {
		return SearchModeRegex, true
	}

	return SearchModeAuto, true
}

func normalizeSearchOptions(opts SearchOptions) (SearchOptions, bool) {
	mode, ok := normalizeSearchMode(opts.Mode, opts.IsRegex, opts.LegacyIsRegexSet)
	if !ok {
		return opts, false
	}

	opts.Mode = string(mode)
	switch mode {
	case SearchModeRegex:
		opts.IsRegex = true
	case SearchModeLiteral, SearchModeSymbol, SearchModeAuto:
		opts.IsRegex = false
	}

	return opts, true
}
