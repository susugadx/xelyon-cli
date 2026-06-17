package gathercontext

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filequery"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

type request struct {
	query                string
	path                 string
	fileFilter           string
	searchQuery          string
	searchPath           string
	searchRouteIntent    bool
	literalSearchPattern bool
}

func normalizeRequest(req request) request {
	rawQuery := strings.TrimSpace(req.query)
	normalizedQuery := normalizeQueryTextWithMetadata(rawQuery)
	path := strings.TrimSpace(req.path)

	normalized := request{
		query:                normalizedQuery.text,
		path:                 path,
		fileFilter:           strings.TrimSpace(req.fileFilter),
		searchQuery:          strings.TrimSpace(req.searchQuery),
		searchPath:           strings.TrimSpace(req.searchPath),
		searchRouteIntent:    req.searchRouteIntent || filequery.LooksLikeNaturalLanguageSearchIntent(rawQuery) || normalizedQuery.searchRouteIntent,
		literalSearchPattern: req.literalSearchPattern || normalizedQuery.literalPattern,
	}
	if normalized.searchQuery == "" {
		normalized.searchQuery = normalized.query
	}
	if normalized.searchPath == "" {
		normalized.searchPath = normalized.path
	}
	return normalizeSearchRouteFields(normalized)
}

func parseRequestArgs(args map[string]string) (request, string) {
	req := normalizeRequest(request{
		query:      strings.TrimSpace(args["query"]),
		path:       strings.TrimSpace(args["path"]),
		fileFilter: strings.TrimSpace(args["file_filter"]),
	})
	if req.query == "" {
		return request{}, "Error: query is required"
	}
	if !hasEffectiveSearchPattern(req) {
		return request{}, "Error: query must include at least one non-empty term"
	}
	return req, ""
}

func hasEffectiveSearchPattern(req request) bool {
	if req.literalSearchPattern {
		return strings.TrimSpace(req.searchQuery) != ""
	}
	return search.HasEffectivePatternList(req.searchQuery)
}
