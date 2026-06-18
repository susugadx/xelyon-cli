package directquery

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func resolveExplicitRoute(execCtx tools.ExecutionContext, input directQueryInput, policy Policy) Outcome {
	resolution, errResult := resolveDirectQueryInput(execCtx, input)
	if errResult != "" {
		return Outcome{
			Kind:  OutcomeError,
			Error: errResult,
		}
	}
	return gatherContextResolvedDirectRouteOutcome(resolution, policy)
}

func resolveScopedRoute(execCtx tools.ExecutionContext, input directQueryInput, policy Policy) (Outcome, bool) {
	scopedResolution := resolveScopedGatherContextDirectResolution(execCtx, input, policy)
	switch scopedResolution.Kind {
	case scopedDirectResolutionResolved:
		return gatherContextResolvedDirectRouteOutcome(scopedResolution.Resolution, policy), true
	case scopedDirectResolutionFiltered:
		return Outcome{Kind: OutcomeNone}, true
	}

	if !inputHasOnlyScopedDirectCandidates(input) {
		return Outcome{}, false
	}

	if outcome, ok := strictScopedDirectErrorOutcome(input, scopedResolution); ok {
		return outcome, true
	}
	return Outcome{Kind: OutcomeNone}, true
}

func finalizeRoute(route Route, policy Policy) Route {
	if route.Kind != RouteDirectory {
		return route
	}

	fileFilter := strings.TrimSpace(policy.FileFilter)
	if fileFilter == "" {
		return route
	}

	targets := append([]directQueryTarget(nil), route.targets...)
	for i := range targets {
		if targets[i].Kind == directQueryTargetDirectory {
			targets[i].FileFilter = fileFilter
		}
	}
	route.targets = targets
	return route
}

func routeFromDirectResolution(resolution directQueryResolution) (Route, bool) {
	switch resolution.Kind {
	case directQueryResolutionDirectory:
		return Route{
			Kind:    RouteDirectory,
			targets: resolution.Targets,
		}, true
	case directQueryResolutionFiles:
		return Route{
			Kind:    RouteRead,
			targets: resolution.Targets,
		}, true
	default:
		return Route{}, false
	}
}

func gatherContextResolvedDirectRouteOutcome(resolution directQueryResolution, policy Policy) Outcome {
	route, ok := routeFromDirectResolution(resolution)
	if !ok {
		return Outcome{
			Kind:  OutcomeError,
			Error: "Error: direct query could not be routed",
		}
	}
	route = finalizeRoute(route, policy)
	return Outcome{
		Kind:  OutcomeResolved,
		Route: route,
	}
}
