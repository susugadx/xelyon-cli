package file

import "github.com/susugadx/xelyon-cli/internal/tools"

type gatherContextFallbackRouteMode string

const (
	gatherContextFallbackRouteModeNone gatherContextFallbackRouteMode = "none"
	gatherContextFallbackRouteModeAny  gatherContextFallbackRouteMode = "any"
	gatherContextFallbackRouteModeRead gatherContextFallbackRouteMode = "read"
)

func resolveFallbackGatherContextDirectRoute(execCtx tools.ExecutionContext, input directQueryInput, policy GatherContextDirectRoutePolicy) GatherContextDirectRouteOutcome {
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
		return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}
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

func resolveCandidateGatherContextReadRoute(execCtx tools.ExecutionContext, input directQueryInput) GatherContextDirectRouteOutcome {
	targets, ok := resolveExistingDirectReadTargets(execCtx, input)
	if ok {
		return GatherContextDirectRouteOutcome{
			Kind: GatherContextDirectRouteOutcomeResolved,
			Route: GatherContextDirectRoute{
				Kind:    GatherContextDirectRouteRead,
				targets: targets,
			},
		}
	}
	if len(input.Entries) > 1 {
		return resolveRequiredCandidateGatherContextReadRoute(execCtx, input)
	}
	return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}
}

func resolveCandidateGatherContextAnyRoute(execCtx tools.ExecutionContext, input directQueryInput, policy GatherContextDirectRoutePolicy) GatherContextDirectRouteOutcome {
	resolution, errResult := resolveDirectQueryInput(execCtx, input)
	if errResult != "" {
		return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}
	}
	route, ok := routeFromDirectResolution(resolution)
	if !ok {
		return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}
	}
	route = finalizeGatherContextDirectRoute(route, policy)
	return GatherContextDirectRouteOutcome{
		Kind:  GatherContextDirectRouteOutcomeResolved,
		Route: route,
	}
}

func resolveRequiredCandidateGatherContextReadRoute(execCtx tools.ExecutionContext, input directQueryInput) GatherContextDirectRouteOutcome {
	targets, errResult := resolveDirectReadTargets(execCtx, input)
	if errResult != "" {
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: errResult,
		}
	}
	return GatherContextDirectRouteOutcome{
		Kind: GatherContextDirectRouteOutcomeResolved,
		Route: GatherContextDirectRoute{
			Kind:    GatherContextDirectRouteRead,
			targets: targets,
		},
	}
}

func resolveRequiredCandidateGatherContextAnyRoute(execCtx tools.ExecutionContext, input directQueryInput, policy GatherContextDirectRoutePolicy) GatherContextDirectRouteOutcome {
	resolution, errResult := resolveDirectQueryInput(execCtx, input)
	if errResult != "" {
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: errResult,
		}
	}
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
