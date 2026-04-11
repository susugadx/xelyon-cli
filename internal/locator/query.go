package locator

import "strings"

type QueryIntent int

const (
	QueryIntentNone QueryIntent = iota
	QueryIntentStrong
	QueryIntentAmbiguous
)

type QueryPriority struct {
	Intent             QueryIntent
	HasResolvedLocator bool
}

func (p QueryPriority) ShouldRouteLocator() bool {
	switch p.Intent {
	case QueryIntentStrong:
		return true
	case QueryIntentAmbiguous:
		return p.HasResolvedLocator
	default:
		return false
	}
}

// LooksLikeLocatorQuery reports whether raw follows the same locator-ID syntax
// accepted by Resolve / ResolveMulti, including bare IDs and lowercase forms.
func LooksLikeLocatorQuery(raw string) bool {
	_, ok := parseLocatorQueryParts(raw)
	return ok
}

// ClassifyQueryPriority owns gather_context locator follow-up priority.
// Bracketed forms are strong locator intent; bare locator-like forms only
// promote to locator follow-up when the current registry can resolve them.
func ClassifyQueryPriority(raw string, reg *Registry) QueryPriority {
	parts, ok := parseLocatorQueryParts(raw)
	if !ok {
		return QueryPriority{}
	}

	priority := QueryPriority{Intent: parseLocatorQueryIntent(raw)}
	if priority.Intent == QueryIntentAmbiguous && reg != nil {
		raw = strings.TrimSpace(raw)
		if len(parts) == 1 {
			_, priority.HasResolvedLocator = reg.Resolve(raw)
		} else {
			priority.HasResolvedLocator = parsedLocatorPartsResolveAll(reg, parts)
		}
	}
	return priority
}

func parsedLocatorPartsResolveAll(reg *Registry, parts []string) bool {
	if reg == nil || len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		if _, ok := reg.Resolve(part); !ok {
			return false
		}
	}
	return true
}

func parseLocatorQueryIntent(raw string) QueryIntent {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return QueryIntentNone
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		return QueryIntentStrong
	}
	return QueryIntentAmbiguous
}

func parseLocatorQueryParts(raw string) ([]string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}

	if strings.HasPrefix(raw, "[") || strings.HasSuffix(raw, "]") {
		if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
			return nil, false
		}
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
		if raw == "" {
			return nil, false
		}
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		if _, ok := parseID(part); !ok {
			return nil, false
		}
	}
	return parts, true
}
