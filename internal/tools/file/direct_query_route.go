package file

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type gatherContextFallbackRouteMode string

const (
	gatherContextFallbackRouteModeNone gatherContextFallbackRouteMode = "none"
	gatherContextFallbackRouteModeAny  gatherContextFallbackRouteMode = "any"
	gatherContextFallbackRouteModeRead gatherContextFallbackRouteMode = "read"
)

// PlanGatherContextDirectRoute centralizes gather_context-specific direct query policy
// so high-level routing does not need to know explicit/implicit path heuristics.
// Route ownership lives here: classify scoped direct, generic explicit direct,
// implicit direct, or search fallback. Exact scoped resolution itself lives in
// direct_query_resolve.go.
func PlanGatherContextDirectRoute(execCtx tools.ExecutionContext, query string, policy GatherContextDirectRoutePolicy) GatherContextDirectRouteOutcome {
	input, ok := parseDirectQueryInput(query)
	if !ok {
		return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}
	}

	if inputHasOnlyExplicitPathSyntax(input) {
		return resolveExplicitGatherContextDirectRoute(execCtx, input, policy)
	}

	if hasScopedExactFilenameLookupScope(policy) {
		if outcome, handled := resolveScopedGatherContextDirectRoute(execCtx, input, policy); handled {
			return outcome
		}
	}

	return resolveFallbackGatherContextDirectRoute(execCtx, input, policy)
}

// ExecuteGatherContextDirectRoute executes the resolved direct route using file-package semantics.
func ExecuteGatherContextDirectRoute(execCtx tools.ExecutionContext, route GatherContextDirectRoute, detail string, depth int) string {
	output, _ := ExecuteGatherContextDirectRouteWithObservation(execCtx, route, detail, depth)
	return output
}

// ExecuteGatherContextDirectRouteWithObservation executes the resolved direct route and returns runtime observation facts.
func ExecuteGatherContextDirectRouteWithObservation(execCtx tools.ExecutionContext, route GatherContextDirectRoute, detail string, depth int) (string, *tools.RuntimeObservation) {
	switch route.Kind {
	case GatherContextDirectRouteRead:
		sections := ExecuteDirectReadTargetsWithDetailSections(execCtx, route.targets, detail)
		return RenderReadExecutionSections(sections), MergeReadExecutionSectionObservations(sections)
	case GatherContextDirectRouteDirectory:
		if len(route.targets) == 0 {
			return "Error: path is not a directory", nil
		}
		return ExecuteDirectListDirTarget(execCtx, route.targets[0], depth), nil
	default:
		return "", nil
	}
}

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

func strictScopedDirectErrorOutcome(input directQueryInput, scopedResolution scopedDirectResolutionOutcome) (GatherContextDirectRouteOutcome, bool) {
	if !inputHasStrictScopedDirectIntent(input) {
		return GatherContextDirectRouteOutcome{}, false
	}

	switch scopedResolution.Kind {
	case scopedDirectResolutionMissing:
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: scopedResolution.Error,
		}, true
	case scopedDirectResolutionAmbiguous:
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: "Error: direct path is ambiguous: " + joinDirectQueryRawEntries(input),
		}, true
	default:
		return GatherContextDirectRouteOutcome{}, false
	}
}

func joinDirectQueryRawEntries(input directQueryInput) string {
	entries := make([]string, 0, len(input.Entries))
	for _, entry := range input.Entries {
		entries = append(entries, entry.RawEntry)
	}
	return strings.Join(entries, ",")
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

func hasScopedExactFilenameLookupScope(policy GatherContextDirectRoutePolicy) bool {
	return policy.ScopedPath != "" || policy.FileFilter != ""
}

func inputHasOnlyScopedDirectCandidates(input directQueryInput) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		if !entryCanUseScopedDirectResolution(entry) {
			return false
		}
	}
	return true
}
