package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

const (
	goImpactRiskLow    = "low"
	goImpactRiskMedium = "medium"
	goImpactRiskHigh   = "high"
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

func goImpactPlanRank(plan goImpactPlan) int {
	switch plan.riskLevel {
	case goImpactRiskHigh:
		return 3
	case goImpactRiskMedium:
		return 2
	default:
		return 1
	}
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
	if goImpactNeedsWidening(result) {
		return goImpactRiskMedium
	}

	return goImpactRiskLow
}

func goImpactNeedsWidening(result navigation.InspectResult) bool {
	if result.MoreCallers || result.MoreRefs || result.UpstreamTruncated || result.UpstreamIncomplete {
		return true
	}
	return result.TotalCallers > len(result.Callers) || result.TotalRefs > len(result.Refs)
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
