package directquery

import "github.com/susugadx/xelyon-cli/internal/tools"

type gatherContextFallbackRouteMode string

const (
	gatherContextFallbackRouteModeNone gatherContextFallbackRouteMode = "none"
	gatherContextFallbackRouteModeAny  gatherContextFallbackRouteMode = "any"
	gatherContextFallbackRouteModeRead gatherContextFallbackRouteMode = "read"
)

func resolveFallbackRoute(execCtx tools.ExecutionContext, input directQueryInput, policy Policy) Outcome {
	switch resolveGatherContextFallbackRouteMode(input, policy.AllowImplicitBareFile) {
	case gatherContextFallbackRouteModeAny:
		if inputHasOnlyStrongDirectIntent(input) {
			return resolveRequiredCandidateGatherContextAnyRoute(execCtx, input, policy)
		}
		return resolveCandidateGatherContextAnyRoute(execCtx, input, policy)
	case gatherContextFallbackRouteModeRead:
		if inputHasOnlyStrongDirectIntent(input) {
			return resolveRequiredCandidateGatherContextReadRoute(execCtx, input)
		}
		return resolveCandidateGatherContextReadRoute(execCtx, input)
	default:
		return Outcome{Kind: OutcomeNone}
	}
}

func resolveGatherContextFallbackRouteMode(input directQueryInput, allowImplicitBareFile bool) gatherContextFallbackRouteMode {
	if inputHasOnlyPathCandidateSyntax(input) {
		return gatherContextFallbackRouteModeAny
	}
	if inputContainsPathCandidateSyntax(input) {
		if !inputHasOnlyCandidateDirectSyntax(input, allowImplicitBareFile) {
			return gatherContextFallbackRouteModeNone
		}
		return gatherContextFallbackRouteModeAny
	}
	if !allowImplicitBareFile {
		return gatherContextFallbackRouteModeNone
	}
	if !inputHasOnlyDirectReadCandidates(input, true) {
		return gatherContextFallbackRouteModeNone
	}
	return gatherContextFallbackRouteModeRead
}

func resolveCandidateGatherContextReadRoute(execCtx tools.ExecutionContext, input directQueryInput) Outcome {
	targets, ok := resolveExistingDirectReadTargets(execCtx, input)
	if ok {
		return Outcome{
			Kind: OutcomeResolved,
			Route: Route{
				Kind:    RouteRead,
				targets: targets,
			},
		}
	}
	if len(input.Entries) > 1 {
		return resolveRequiredCandidateGatherContextReadRoute(execCtx, input)
	}
	return Outcome{Kind: OutcomeNone}
}

func resolveCandidateGatherContextAnyRoute(execCtx tools.ExecutionContext, input directQueryInput, policy Policy) Outcome {
	resolution, errResult := resolveDirectQueryInput(execCtx, input)
	if errResult != "" {
		return Outcome{Kind: OutcomeNone}
	}
	route, ok := routeFromDirectResolution(resolution)
	if !ok {
		return Outcome{Kind: OutcomeNone}
	}
	route = finalizeRoute(route, policy)
	return Outcome{
		Kind:  OutcomeResolved,
		Route: route,
	}
}

func resolveRequiredCandidateGatherContextReadRoute(execCtx tools.ExecutionContext, input directQueryInput) Outcome {
	targets, errResult := resolveDirectReadTargets(execCtx, input)
	if errResult != "" {
		return Outcome{
			Kind:  OutcomeError,
			Error: errResult,
		}
	}
	return Outcome{
		Kind: OutcomeResolved,
		Route: Route{
			Kind:    RouteRead,
			targets: targets,
		},
	}
}

func resolveRequiredCandidateGatherContextAnyRoute(execCtx tools.ExecutionContext, input directQueryInput, policy Policy) Outcome {
	resolution, errResult := resolveDirectQueryInput(execCtx, input)
	if errResult != "" {
		return Outcome{
			Kind:  OutcomeError,
			Error: errResult,
		}
	}
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
