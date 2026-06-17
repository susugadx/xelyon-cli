package file

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func resolveExplicitGatherContextDirectRoute(execCtx tools.ExecutionContext, input directQueryInput, policy GatherContextDirectRoutePolicy) GatherContextDirectRouteOutcome {
	resolution, errResult := resolveDirectQueryInput(execCtx, input)
	if errResult != "" {
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: errResult,
		}
	}
	return gatherContextResolvedDirectRouteOutcome(resolution, policy)
}

func resolveScopedGatherContextDirectRoute(execCtx tools.ExecutionContext, input directQueryInput, policy GatherContextDirectRoutePolicy) (GatherContextDirectRouteOutcome, bool) {
	scopedResolution := resolveScopedGatherContextDirectResolution(execCtx, input, policy)
	switch scopedResolution.Kind {
	case scopedDirectResolutionResolved:
		return gatherContextResolvedDirectRouteOutcome(scopedResolution.Resolution, policy), true
	case scopedDirectResolutionFiltered:
		return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}, true
	}

	if !inputHasOnlyScopedDirectCandidates(input) {
		return GatherContextDirectRouteOutcome{}, false
	}

	if outcome, ok := strictScopedDirectErrorOutcome(input, scopedResolution); ok {
		return outcome, true
	}
	return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}, true
}

func finalizeGatherContextDirectRoute(route GatherContextDirectRoute, policy GatherContextDirectRoutePolicy) GatherContextDirectRoute {
	if route.Kind != GatherContextDirectRouteDirectory {
		return route
	}

	fileFilter := strings.TrimSpace(policy.FileFilter)
	if fileFilter == "" {
		return route
	}

	targets := append([]DirectQueryTarget(nil), route.targets...)
	for i := range targets {
		if targets[i].Kind == DirectQueryTargetDirectory {
			targets[i].FileFilter = fileFilter
		}
	}
	route.targets = targets
	return route
}

func routeFromDirectResolution(resolution DirectQueryResolution) (GatherContextDirectRoute, bool) {
	switch resolution.Kind {
	case DirectQueryResolutionDirectory:
		return GatherContextDirectRoute{
			Kind:    GatherContextDirectRouteDirectory,
			targets: resolution.Targets,
		}, true
	case DirectQueryResolutionFiles:
		return GatherContextDirectRoute{
			Kind:    GatherContextDirectRouteRead,
			targets: resolution.Targets,
		}, true
	default:
		return GatherContextDirectRoute{}, false
	}
}

func gatherContextResolvedDirectRouteOutcome(resolution DirectQueryResolution, policy GatherContextDirectRoutePolicy) GatherContextDirectRouteOutcome {
	route, ok := routeFromDirectResolution(resolution)
	if !ok {
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: "Error: direct query could not be routed",
		}
	}
	route = finalizeGatherContextDirectRoute(route, policy)
	return GatherContextDirectRouteOutcome{
		Kind:  GatherContextDirectRouteOutcomeResolved,
		Route: route,
	}
}
