package search

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

const structuredGoImpactRouteTag = "impact-structured-go-v1"

func shouldAttemptStructuredGoImpactSearch(opts SearchOptions, pattern string) bool {
	if !shouldAttemptSinglePatternImpactSearch(opts, pattern) {
		return false
	}
	return resolveLanguage(opts) == "go"
}

func resolveStructuredGoImpactSymbol(symbol string, opts SearchOptions) symbolResolveResult {
	inspected := inspectStructuredGoImpactSymbol(symbol, opts)
	switch inspected.status {
	case navigation.SymbolAutoMultiple:
		return resolveStructuredGoImpactMultipleSymbol(symbol, inspected.result, inspected.output, opts, inspected.plan.budget)
	case navigation.SymbolAutoSingle:
		return buildStructuredGoImpactSingleSymbolResult(symbol, inspected.result, opts, inspected.plan)
	default:
		return symbolResolveResult{Status: navigationStatusToSymbolResolveStatus(inspected.status)}
	}
}

type structuredGoImpactInspection struct {
	result navigation.InspectResult
	output string
	status navigation.SymbolAutoStatus
	plan   goImpactPlan
}

func inspectStructuredGoImpactSymbol(symbol string, opts SearchOptions) structuredGoImpactInspection {
	plan := goImpactPlanForRisk(goImpactRiskLow)
	inspected := structuredGoImpactInspection{plan: plan}

	for i := 0; i < 3; i++ {
		result, output, status := navigation.ResolveInspectSymbolAuto(symbol, opts.Path, navigation.InspectSymbolAutoOptions{
			Budget:             plan.budget,
			Registry:           nil,
			LSPClient:          opts.LSPClient,
			ProjectMap:         opts.ProjectMap,
			ProjectMapRootPath: opts.ProjectMapRootPath,
			ProjectMapStateKey: opts.ProjectMapStateKey,
			InvocationCWD:      opts.InvocationCWD,
		})
		inspected.result = result
		inspected.output = output
		inspected.status = status
		inspected.plan = plan

		if status != navigation.SymbolAutoSingle {
			return inspected
		}

		nextPlan := nextStructuredGoImpactPlan(plan, result)
		if goImpactPlanEqual(plan, nextPlan) {
			inspected.plan = nextPlan
			return inspected
		}
		plan = nextPlan
	}

	inspected.plan = plan
	return inspected
}

func nextStructuredGoImpactPlan(current goImpactPlan, result navigation.InspectResult) goImpactPlan {
	nextPlan := goImpactPlanForRisk(classifyGoImpactRisk(result))
	if goImpactPlanRank(nextPlan) < goImpactPlanRank(current) {
		return current
	}
	return nextPlan
}

func buildStructuredGoImpactSingleSymbolResult(symbol string, result navigation.InspectResult, opts SearchOptions, plan goImpactPlan) symbolResolveResult {
	var probeDependencies []string
	result, probeDependencies = supplementGoImpactTestsFromProbe(symbol, result, opts, plan.budget.TestLimit)
	impact := buildGoImpactMetadata(result, plan.riskLevel)
	bundle := buildGoSymbolBundleWithOptions(symbol, result, goSymbolBundleBuildOptions{
		implementationLimit: plan.implementationLimit,
		impact:              impact,
	})
	if bundle == nil {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	bundle.Debug.DependencyFiles = dedupePaths(append(bundle.Debug.DependencyFiles, probeDependencies...))

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
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

func buildGoImpactMetadata(result navigation.InspectResult, risk string) *SymbolBundleImpact {
	if result.Symbol == nil {
		return nil
	}

	impact := &SymbolBundleImpact{
		RiskLevel:        risk,
		RecommendedReads: make([]SymbolBundleItem, 0, 8),
	}

	seen := make(map[string]struct{})
	add := func(item SymbolBundleItem) {
		if item.File == "" || item.Line <= 0 {
			return
		}
		key := fmt.Sprintf("%s:%d:%s", item.File, item.Line, item.Kind)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		impact.RecommendedReads = append(impact.RecommendedReads, item)
	}

	add(SymbolBundleItem{
		Kind:    "definition",
		File:    result.Symbol.File,
		Line:    result.Symbol.Line,
		EndLine: result.Symbol.EndLine,
		Snippet: impactDefinitionSnippet(result),
		Name:    result.Symbol.Name,
	})

	for _, item := range primaryCallerReadItems(result.Callers, impactReadLimit(risk, "callers")) {
		add(item)
	}
	for _, item := range primaryTestReadItems(result.Tests, impactReadLimit(risk, "tests")) {
		add(item)
	}
	for _, item := range primaryImplementationReadItems(result.Implementations, impactReadLimit(risk, "implementations")) {
		add(item)
	}
	for _, item := range crossPackageRefReadItems(result, impactReadLimit(risk, "references")) {
		add(item)
	}

	if len(impact.RecommendedReads) == 0 {
		return nil
	}
	return impact
}

func impactDefinitionSnippet(result navigation.InspectResult) string {
	if result.Symbol != nil && strings.TrimSpace(result.Symbol.Signature) != "" {
		return strings.TrimSpace(result.Symbol.Signature)
	}
	if len(result.Body) == 0 {
		return ""
	}
	line := strings.TrimSpace(result.Body[0])
	if idx := strings.Index(line, ":"); idx > 0 {
		allDigits := true
		for _, r := range line[:idx] {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return strings.TrimSpace(line[idx+1:])
		}
	}
	return line
}

func impactReadLimit(risk, kind string) int {
	switch kind {
	case "callers":
		switch risk {
		case goImpactRiskHigh:
			return 3
		case goImpactRiskMedium:
			return 2
		default:
			return 1
		}
	case "tests":
		if risk == goImpactRiskHigh {
			return 2
		}
		return 1
	case "implementations":
		switch risk {
		case goImpactRiskHigh:
			return 3
		case goImpactRiskMedium:
			return 2
		default:
			return 1
		}
	case "references":
		switch risk {
		case goImpactRiskHigh:
			return 2
		case goImpactRiskMedium:
			return 1
		default:
			return 0
		}
	default:
		return 0
	}
}

func primaryCallerReadItems(refs []navigation.Reference, limit int) []SymbolBundleItem {
	if limit <= 0 || len(refs) == 0 {
		return nil
	}
	items := make([]SymbolBundleItem, 0, min(limit, len(refs)))
	for _, ref := range refs {
		if len(items) >= limit {
			break
		}
		snippet := strings.TrimSpace(ref.Snippet)
		if snippet == "" && ref.Scope != "" {
			snippet = ref.Scope
		}
		items = append(items, SymbolBundleItem{
			Kind:         "callers",
			File:         ref.File,
			ResolvedPath: ref.ResolvedPath,
			Line:         ref.Line,
			Snippet:      snippet,
			Scope:        ref.Scope,
		})
	}
	return items
}

func primaryTestReadItems(tests []navigation.TestRef, limit int) []SymbolBundleItem {
	if limit <= 0 || len(tests) == 0 {
		return nil
	}
	items := make([]SymbolBundleItem, 0, min(limit, len(tests)))
	for _, test := range tests {
		if len(items) >= limit {
			break
		}
		items = append(items, SymbolBundleItem{
			Kind:         "tests",
			File:         test.File,
			ResolvedPath: test.ResolvedPath,
			Line:         test.Line,
			Name:         test.Name,
			IsTest:       true,
		})
	}
	return items
}

func primaryImplementationReadItems(impls []navigation.ImplementationRef, limit int) []SymbolBundleItem {
	if limit <= 0 || len(impls) == 0 {
		return nil
	}
	items := make([]SymbolBundleItem, 0, min(limit, len(impls)))
	for _, impl := range impls {
		if len(items) >= limit {
			break
		}
		items = append(items, SymbolBundleItem{
			Kind:         "implementations",
			File:         impl.File,
			ResolvedPath: impl.ResolvedPath,
			Line:         impl.Line,
			Snippet:      strings.TrimSpace(impl.Name),
			Name:         impl.Name,
		})
	}
	return items
}

func crossPackageRefReadItems(result navigation.InspectResult, limit int) []SymbolBundleItem {
	if result.Symbol == nil || limit <= 0 || len(result.Refs) == 0 {
		return nil
	}

	symbolDir := filepath.ToSlash(strings.TrimSpace(result.Symbol.PackageDir))
	items := make([]SymbolBundleItem, 0, min(limit, len(result.Refs)))
	seenFiles := make(map[string]struct{})
	for _, ref := range result.Refs {
		refDir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(ref.File)))
		if refDir == symbolDir {
			continue
		}
		if _, ok := seenFiles[ref.File]; ok {
			continue
		}
		seenFiles[ref.File] = struct{}{}
		snippet := strings.TrimSpace(ref.Snippet)
		if snippet == "" && ref.Scope != "" {
			snippet = ref.Scope
		}
		items = append(items, SymbolBundleItem{
			Kind:         "references",
			File:         ref.File,
			ResolvedPath: ref.ResolvedPath,
			Line:         ref.Line,
			Snippet:      snippet,
			Scope:        ref.Scope,
			IsTest:       ref.IsTest,
		})
		if len(items) >= limit {
			break
		}
	}
	return items
}

func cloneSymbolBundleImpact(impact *SymbolBundleImpact) *SymbolBundleImpact {
	if impact == nil {
		return nil
	}
	cloned := *impact
	if impact.RecommendedReads != nil {
		cloned.RecommendedReads = append([]SymbolBundleItem(nil), impact.RecommendedReads...)
	}
	return &cloned
}
