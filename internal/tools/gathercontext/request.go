package gathercontext

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

type request struct {
	query      string
	path       string
	fileFilter string
}

func normalizeRequest(req request) request {
	return request{
		query:      strings.TrimSpace(req.query),
		path:       strings.TrimSpace(req.path),
		fileFilter: strings.TrimSpace(req.fileFilter),
	}
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
	if !search.HasEffectivePatternList(req.query) {
		return request{}, "Error: query must include at least one non-empty term"
	}
	return req, ""
}
