package search

import (
	"github.com/susugadx/xelyon-cli/internal/goimpact"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func classifyGoImpactRisk(result navigation.InspectResult) string {
	return goimpact.ClassifyRisk(result)
}
