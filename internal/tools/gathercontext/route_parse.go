package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/locator"
)

func classifyLocatorQuery(query string, reg *locator.Registry) locator.QueryPriority {
	return locator.ClassifyQueryPriority(query, reg)
}
