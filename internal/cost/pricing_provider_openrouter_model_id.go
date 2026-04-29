package cost

import (
	"strings"
)

type openRouterModelID struct {
	owner       string
	routedModel string
}

func parseOpenRouterModelID(model string) (openRouterModelID, bool) {
	raw := strings.ToLower(strings.TrimSpace(model))
	owner, routedModel, ok := strings.Cut(raw, "/")
	owner = strings.TrimSpace(owner)
	routedModel = strings.TrimSpace(routedModel)
	if !ok || owner == "" || routedModel == "" {
		return openRouterModelID{}, false
	}
	return openRouterModelID{
		owner:       owner,
		routedModel: routedModel,
	}, true
}
