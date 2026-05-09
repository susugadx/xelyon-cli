package gathercontext

type searchRouteRewritePlan struct {
	apply               bool
	query               string
	path                string
	useInlinePath       bool
	naturalSearchIntent bool
}

func normalizeSearchRouteFields(req request) request {
	plan := planSearchRouteRewrite(req)
	if !plan.apply {
		return req
	}
	req.searchQuery = plan.query
	if plan.naturalSearchIntent {
		req.naturalSearchIntent = true
	}
	if plan.useInlinePath && req.path == "" {
		req.searchPath = plan.path
	}
	return req
}

func planSearchRouteRewrite(req request) searchRouteRewritePlan {
	queryShape := classifyRequestQueryShape(req)
	if queryShape.rewriteProtected || !queryShape.singleEntry {
		return searchRouteRewritePlan{}
	}

	// Search rewrites are intentionally single-entry only. Comma-separated input is
	// either a direct batch or an explicit multi-pattern search string.
	if scope, ok := parseTrailingInlineSearchScope(req.query); ok {
		return searchRouteRewritePlan{
			apply:               true,
			query:               normalizeSearchPatternList(stripLeadingSearchIntent(scope.query)),
			path:                scope.path,
			useInlinePath:       true,
			naturalSearchIntent: true,
		}
	}

	if queryShape.explicitDirectoryMarker {
		return searchRouteRewritePlan{}
	}

	return searchRouteRewritePlan{
		apply: true,
		query: normalizeSearchPatternList(stripLeadingSearchIntent(req.query)),
	}
}
