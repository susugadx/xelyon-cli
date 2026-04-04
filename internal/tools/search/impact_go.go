package search

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	goImpactRiskLow    = "low"
	goImpactRiskMedium = "medium"
	goImpactRiskHigh   = "high"

	structuredGoImpactRouteTag = "impact-structured-go-v1"
)

type goImpactPlan struct {
	riskLevel           string
	budget              navigation.Budget
	implementationLimit int
}

var goImpactLowBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 3,
	RefLimit:    3,
	TestLimit:   2,
}

var goImpactMediumBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 5,
	RefLimit:    5,
	TestLimit:   3,
}

var goImpactHighBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 8,
	RefLimit:    8,
	TestLimit:   4,
}

func tryStructuredGoImpactSearch(cache tools.ToolCacheInterface, opts SearchOptions) (string, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	if !shouldAttemptStructuredGoImpactSearch(opts, pattern) {
		return "", false
	}

	route := planSearchRoute(pattern, opts)
	if route.InitialLane != searchLaneSymbol || route.Language != "go" {
		return "", false
	}

	cacheKey := buildSearchCacheKeyWithRoute(opts, route.cacheSignature()+"|"+structuredGoImpactRouteTag)
	if cache != nil {
		if cached, ok := cache.GetSearch(pattern, cacheKey); ok {
			return cached, true
		}
	}

	resolved := resolveStructuredGoImpactSymbol(pattern, opts)
	if resolved.Status != symbolResolveSingle || resolved.Bundle == nil {
		return "", false
	}

	route.SymbolAttempted = true
	route.SymbolResolved = true
	route.FinalLane = searchLaneSymbol
	resolved.Bundle = attachBundleRoute(resolved.Bundle, route)
	affectedFiles := collectSymbolBundleAffectedFiles(resolved.Bundle, opts)

	if cache != nil {
		cache.SetSearch(pattern, cacheKey, resolved.Output, affectedFiles)
		storeSinglePatternBundle(pattern, cacheKey, resolved.Bundle)
		storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
	}

	return resolved.Output, true
}

func shouldAttemptStructuredGoImpactSearch(opts SearchOptions, pattern string) bool {
	if !strings.EqualFold(strings.TrimSpace(opts.Intent), "impact") {
		return false
	}
	if len(splitPatterns(opts.Pattern)) != 1 {
		return false
	}
	if strings.TrimSpace(pattern) == "" {
		return false
	}
	return resolveLanguage(opts) == "go"
}

func resolveStructuredGoImpactSymbol(symbol string, opts SearchOptions) symbolResolveResult {
	plan := goImpactPlanForRisk(goImpactRiskLow)
	var (
		result navigation.InspectResult
		status navigation.SymbolAutoStatus
	)

	for i := 0; i < 3; i++ {
		result, _, status = navigation.ResolveInspectSymbolAuto(symbol, opts.Path, navigation.InspectSymbolAutoOptions{
			Budget:             plan.budget,
			Registry:           nil,
			LSPClient:          opts.LSPClient,
			ProjectMap:         opts.ProjectMap,
			ProjectMapRootPath: opts.ProjectMapRootPath,
			ProjectMapStateKey: opts.ProjectMapStateKey,
			InvocationCWD:      opts.InvocationCWD,
		})
		if status != navigation.SymbolAutoSingle {
			return symbolResolveResult{Status: navigationStatusToSymbolResolveStatus(status)}
		}

		nextPlan := goImpactPlanForRisk(classifyGoImpactRisk(result))
		if goImpactPlanEqual(plan, nextPlan) {
			plan = nextPlan
			break
		}
		plan = nextPlan
	}

	result = supplementGoImpactTestsFromProbe(symbol, result, opts, plan.budget.TestLimit)
	impact := buildGoImpactMetadata(result, plan.riskLevel)
	bundle := buildGoSymbolBundleWithOptions(symbol, result, goSymbolBundleBuildOptions{
		implementationLimit: plan.implementationLimit,
		impact:              impact,
	})
	if bundle == nil {
		return symbolResolveResult{Status: symbolResolveNone}
	}

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
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

func goImpactPlanForRisk(risk string) goImpactPlan {
	switch strings.TrimSpace(risk) {
	case goImpactRiskHigh:
		return goImpactPlan{riskLevel: goImpactRiskHigh, budget: goImpactHighBudget, implementationLimit: 8}
	case goImpactRiskMedium:
		return goImpactPlan{riskLevel: goImpactRiskMedium, budget: goImpactMediumBudget, implementationLimit: 4}
	default:
		return goImpactPlan{riskLevel: goImpactRiskLow, budget: goImpactLowBudget, implementationLimit: 2}
	}
}

func goImpactPlanEqual(left, right goImpactPlan) bool {
	return left.riskLevel == right.riskLevel &&
		left.implementationLimit == right.implementationLimit &&
		left.budget == right.budget
}

func classifyGoImpactRisk(result navigation.InspectResult) string {
	if result.Symbol == nil {
		return goImpactRiskLow
	}

	if result.Symbol.Exported || result.Symbol.Kind == "interface" || len(result.Implementations) > 0 {
		return goImpactRiskHigh
	}

	fileCount, dirCount := goImpactReferenceSpread(result)
	if dirCount > 1 {
		return goImpactRiskHigh
	}
	if fileCount > 1 || isSharedGoPackageSymbol(*result.Symbol) {
		return goImpactRiskMedium
	}

	return goImpactRiskLow
}

func goImpactReferenceSpread(result navigation.InspectResult) (int, int) {
	fileSeen := make(map[string]struct{})
	dirSeen := make(map[string]struct{})
	add := func(file string) {
		file = filepath.ToSlash(filepath.Clean(strings.TrimSpace(file)))
		if file == "" {
			return
		}
		fileSeen[file] = struct{}{}
		dirSeen[filepath.ToSlash(filepath.Dir(file))] = struct{}{}
	}

	for _, ref := range result.Callers {
		add(ref.File)
	}
	for _, ref := range result.Refs {
		add(ref.File)
	}
	return len(fileSeen), len(dirSeen)
}

func isSharedGoPackageSymbol(symbol navigation.SymbolCandidate) bool {
	packageDir := filepath.ToSlash(strings.TrimSpace(symbol.PackageDir))
	if packageDir == "" || packageDir == "." {
		return false
	}
	return packageDir != "cmd" && !strings.HasPrefix(packageDir, "cmd/")
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
			Kind:    "callers",
			File:    ref.File,
			Line:    ref.Line,
			Snippet: snippet,
			Scope:   ref.Scope,
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
			Kind:   "tests",
			File:   test.File,
			Line:   test.Line,
			Name:   test.Name,
			IsTest: true,
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
			Kind:    "implementations",
			File:    impl.File,
			Line:    impl.Line,
			Snippet: strings.TrimSpace(impl.Name),
			Name:    impl.Name,
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
			Kind:    "references",
			File:    ref.File,
			Line:    ref.Line,
			Snippet: snippet,
			Scope:   ref.Scope,
			IsTest:  ref.IsTest,
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

func supplementGoImpactTestsFromProbe(symbol string, result navigation.InspectResult, opts SearchOptions, limit int) navigation.InspectResult {
	if limit <= 0 || result.Symbol == nil || len(result.Tests) > 0 || result.TotalTests > 0 {
		return result
	}

	probe := impactTestProbePattern(symbol)
	if strings.TrimSpace(probe) == "" {
		return result
	}

	tests, total := findGoImpactTestsByNameProbe(probe, result.Symbol.RootPath, opts, limit)
	if len(tests) == 0 {
		return result
	}

	result.Tests = tests
	result.TotalTests = total
	result.MoreTests = total > len(tests)
	return result
}

func findGoImpactTestsByNameProbe(probe, rootPath string, opts SearchOptions, limit int) ([]navigation.TestRef, int) {
	probeOpts := opts
	probeOpts.Intent = ""
	probeOpts.Mode = string(SearchModeLiteral)
	probeOpts.IsRegex = false
	if strings.TrimSpace(probeOpts.FilePattern) == "" && strings.TrimSpace(probeOpts.FileType) == "" {
		probeOpts.FilePattern = "*_test.go"
	}

	output, useRipgrep, _, err := executeSearch(probe, probeOpts)
	if err != nil || strings.TrimSpace(output) == "" {
		return nil, 0
	}

	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output, 0)
	} else {
		results = parseGrepOutput(output, 0)
	}
	results = filterResultsByOptions(results, probeOpts)

	seen := make(map[string]struct{})
	tests := make([]navigation.TestRef, 0, min(limit, len(results)))
	total := 0
	for _, result := range results {
		file := normalizeImpactProbeFile(result.FilePath, rootPath, probeOpts)
		if file == "" {
			continue
		}
		for _, match := range result.Matches {
			if !match.IsMatch {
				continue
			}
			name := extractGoImpactTestName(match.Line, probe)
			if name == "" {
				continue
			}
			key := fmt.Sprintf("%s:%d:%s", file, match.LineNum, name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			total++
			if len(tests) < limit {
				tests = append(tests, navigation.TestRef{
					File: file,
					Name: name,
					Line: match.LineNum,
				})
			}
		}
	}
	return tests, total
}

func normalizeImpactProbeFile(file, rootPath string, opts SearchOptions) string {
	absPath := absoluteAffectedFilePath(file, opts, affectedFileSourceText)
	if absPath == "" {
		return ""
	}

	rootPath = strings.TrimSpace(rootPath)
	if rootPath != "" {
		if rel, err := filepath.Rel(rootPath, absPath); err == nil {
			rel = filepath.Clean(rel)
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return filepath.ToSlash(rel)
			}
		}
	}

	return filepath.ToSlash(filepath.Clean(file))
}

func extractGoImpactTestName(line, probe string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, probe) || !strings.Contains(trimmed, "func ") {
		return ""
	}

	idx := strings.Index(trimmed, "func ")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(trimmed[idx+5:])
	if rest == "" {
		return ""
	}
	end := strings.Index(rest, "(")
	if end <= 0 {
		return ""
	}
	name := strings.TrimSpace(rest[:end])
	if name == "" || !strings.HasPrefix(name, "Test") {
		return ""
	}
	return name
}
