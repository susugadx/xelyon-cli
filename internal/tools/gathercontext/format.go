package gathercontext

import "strings"

func formatExecutionResult(result executionResult) string {
	if result.direct != nil {
		return formatWithRouteHint(result.routeHint, result.direct.body)
	}
	if result.search == nil {
		return formatWithRouteHint(result.routeHint, "")
	}
	if strings.TrimSpace(result.search.prefetchedEvidence) == "" {
		return formatWithRouteHint(result.routeHint, result.search.discovery)
	}
	return formatWithRouteHint(result.routeHint,
		joinSections(
			section("Search / Discovery", result.search.discovery),
			section("Prefetched Evidence", result.search.prefetchedEvidence),
		),
	)
}

func formatWithRouteHint(routeHint, body string) string {
	routeHint = strings.TrimSpace(routeHint)
	body = strings.TrimSpace(body)
	if routeHint == "" {
		return body
	}
	if body == "" {
		return "Route: " + routeHint
	}
	if strings.HasPrefix(body, "Error:") {
		return body + "\n\nRoute: " + routeHint
	}
	return "Route: " + routeHint + "\n\n" + body
}

func section(title, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return body
	}
	if body == "" {
		return ""
	}
	return title + "\n" + body
}

func joinSections(sections ...string) string {
	filtered := make([]string, 0, len(sections))
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		filtered = append(filtered, section)
	}
	return strings.Join(filtered, "\n\n")
}
