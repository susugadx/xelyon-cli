package search

import (
	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

const structuredGoImpactRouteTag = "impact-structured-go-v1"

func resolveStructuredGoImpactSymbol(symbol string, scope structuredImpactScope) symbolResolveResult {
	inspected := inspectStructuredGoImpactSymbol(symbol, scope)
	switch inspected.status {
	case navigation.SymbolAutoMultiple:
		return resolveStructuredGoImpactMultipleSymbol(symbol, inspected.result, inspected.output, scope.Definition, inspected.plan.Budget)
	case navigation.SymbolAutoSingle:
		return buildStructuredGoImpactSingleSymbolResult(symbol, inspected.result, scope.Definition, scope.Evidence, inspected.plan)
	default:
		return symbolResolveResult{Status: navigationStatusToSymbolResolveStatus(inspected.status)}
	}
}

type structuredGoImpactInspection struct {
	result navigation.InspectResult
	output string
	status navigation.SymbolAutoStatus
	plan   impactplan.Plan
}

func inspectStructuredGoImpactSymbol(symbol string, scope structuredImpactScope) structuredGoImpactInspection {
	collectionPlan := impactplan.PlanForRisk(impactplan.RiskHigh)
	result, output, status := navigation.ResolveInspectSymbolAuto(symbol, scope.Definition.Path, navigation.InspectSymbolAutoOptions{
		Budget:                      collectionPlan.Budget,
		Registry:                    nil,
		LSPClient:                   scope.Definition.LSPClient,
		ProjectMap:                  scope.Definition.ProjectMap,
		ProjectMapRootPath:          scope.Definition.ProjectMapRootPath,
		ProjectMapStateKey:          scope.Definition.ProjectMapStateKey,
		InvocationCWD:               scope.Definition.InvocationCWD,
		ReferenceFilter:             structuredGoImpactReferenceFilter(scope.Evidence),
		FallbackReferenceSearchPath: structuredGoImpactFallbackReferenceSearchPath(scope.Evidence),
	})
	result = filterStructuredGoImpactEvidence(result, scope.Evidence)
	inspected := structuredGoImpactInspection{
		result: result,
		output: output,
		status: status,
		plan:   collectionPlan,
	}

	if status != navigation.SymbolAutoSingle {
		return inspected
	}

	inspected.plan = selectStructuredGoImpactPlan(result)
	inspected.result = navigation.ApplyInspectBudget(result, inspected.plan.Budget)
	return inspected
}

func selectStructuredGoImpactPlan(collected navigation.InspectResult) impactplan.Plan {
	plan := impactplan.PlanForRisk(impactplan.RiskLow)
	for i := 0; i < 3; i++ {
		budgeted := navigation.ApplyInspectBudget(collected, plan.Budget)
		nextPlan := nextStructuredGoImpactPlan(plan, budgeted)
		if impactplan.PlanEqual(plan, nextPlan) {
			return nextPlan
		}
		plan = nextPlan
	}
	return plan
}

func nextStructuredGoImpactPlan(current impactplan.Plan, result navigation.InspectResult) impactplan.Plan {
	nextPlan := impactplan.PlanForRisk(classifyGoImpactRisk(result))
	if impactplan.PlanRank(nextPlan) < impactplan.PlanRank(current) {
		return current
	}
	return nextPlan
}

func buildStructuredGoImpactSingleSymbolResult(symbol string, result navigation.InspectResult, formatOpts SearchOptions, evidenceOpts SearchOptions, plan impactplan.Plan) symbolResolveResult {
	var probeDependencies []string
	impactOpts := evidenceOpts
	if result.Symbol != nil {
		impactOpts = structuredImpactNameOnlyEvidenceOptions(result.Symbol.File, evidenceOpts)
	}
	result, probeDependencies = supplementGoImpactTestsFromProbe(symbol, result, impactOpts, plan.Budget.TestLimit)
	result = filterStructuredGoImpactEvidence(result, evidenceOpts)

	bundle, ok := buildStructuredGoImpactSemanticBundle(symbol, result, plan)
	if !ok {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	bundle.Debug.DependencyFiles = dedupePaths(append(bundle.Debug.DependencyFiles, probeDependencies...))

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, formatOpts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
}

func buildStructuredGoImpactSemanticBundle(symbol string, result navigation.InspectResult, plan impactplan.Plan) (*SymbolBundle, bool) {
	impact := buildGoImpactMetadata(result, plan.RiskLevel)
	var recommendedReads []SymbolBundleItem
	if impact != nil {
		recommendedReads = impact.RecommendedReads
	}

	evidence, ok := semanticEvidenceFromGoInspectResultWithOptions(symbol, result, goSemanticEvidenceOptions{
		riskLevel:           plan.RiskLevel,
		recommendedReads:    recommendedReads,
		implementationLimit: plan.ImplementationLimit,
	})
	if !ok {
		return nil, false
	}

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok || bundle == nil {
		return nil, false
	}
	bundle.Impact = cloneSymbolBundleImpact(impact)
	return bundle, true
}

func resolveStructuredGoImpactMultipleSymbol(symbol string, result navigation.InspectResult, output string, opts SearchOptions, budget navigation.Budget) symbolResolveResult {
	affectedFiles := collectNavigationCandidatesAffectedFiles(result.Candidates, opts)
	observation := observationForNavigationCandidates(result.Candidates, opts)
	if opts.LocatorRegistry != nil {
		_, output, _ = navigation.ResolveInspectSymbolAuto(symbol, opts.Path, navigation.InspectSymbolAutoOptions{
			Budget:             budget,
			Registry:           opts.LocatorRegistry,
			LSPClient:          opts.LSPClient,
			ProjectMap:         opts.ProjectMap,
			ProjectMapRootPath: opts.ProjectMapRootPath,
			ProjectMapStateKey: opts.ProjectMapStateKey,
			InvocationCWD:      opts.InvocationCWD,
		})
	}
	return symbolResolveResult{Output: output, Status: symbolResolveMultiple, AffectedFiles: affectedFiles, Observation: observation}
}

func navigationStatusToSymbolResolveStatus(status navigation.SymbolAutoStatus) symbolResolveStatus {
	switch status {
	case navigation.SymbolAutoMultiple:
		return symbolResolveMultiple
	case navigation.SymbolAutoSingle:
		return symbolResolveSingle
	default:
		return symbolResolveNone
	}
}
