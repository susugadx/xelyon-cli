package goimpact

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

const (
	// RiskLow は低リスクの Go symbol impact を表す。
	RiskLow = "low"
	// RiskMedium は中リスクの Go symbol impact を表す。
	RiskMedium = "medium"
	// RiskHigh は高リスクの Go symbol impact を表す。
	RiskHigh = "high"
)

// Plan は Go impact 解析で使う探索 budget と出力上限を表す。
type Plan struct {
	RiskLevel           string
	Budget              navigation.Budget
	ImplementationLimit int
}

// LowBudget は低リスク symbol の inspect budget。
var LowBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 3,
	RefLimit:    3,
	TestLimit:   2,
}

// MediumBudget は中リスク symbol の inspect budget。
var MediumBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 5,
	RefLimit:    5,
	TestLimit:   3,
}

// HighBudget は高リスク symbol の inspect budget。
var HighBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 8,
	RefLimit:    8,
	TestLimit:   4,
}

// PlanForRisk は risk level に応じた Go impact plan を返す。
func PlanForRisk(risk string) Plan {
	switch strings.TrimSpace(risk) {
	case RiskHigh:
		return Plan{RiskLevel: RiskHigh, Budget: HighBudget, ImplementationLimit: 8}
	case RiskMedium:
		return Plan{RiskLevel: RiskMedium, Budget: MediumBudget, ImplementationLimit: 4}
	default:
		return Plan{RiskLevel: RiskLow, Budget: LowBudget, ImplementationLimit: 2}
	}
}

// PlanEqual は 2 つの Go impact plan が同じか返す。
func PlanEqual(left, right Plan) bool {
	return left.RiskLevel == right.RiskLevel &&
		left.ImplementationLimit == right.ImplementationLimit &&
		left.Budget == right.Budget
}

// PlanRank は Go impact plan の risk 順位を返す。
func PlanRank(plan Plan) int {
	switch plan.RiskLevel {
	case RiskHigh:
		return 3
	case RiskMedium:
		return 2
	default:
		return 1
	}
}

// ClassifyRisk は inspect 結果から Go impact risk を判定する。
func ClassifyRisk(result navigation.InspectResult) string {
	if result.Symbol == nil {
		return RiskLow
	}

	if result.Symbol.Exported || result.Symbol.Kind == "interface" || len(result.Implementations) > 0 {
		return RiskHigh
	}

	fileCount, dirCount := ReferenceSpread(result)
	if dirCount > 1 {
		return RiskHigh
	}
	if fileCount > 1 || IsSharedPackageSymbol(*result.Symbol) {
		return RiskMedium
	}
	if NeedsWidening(result) {
		return RiskMedium
	}

	return RiskLow
}

// NeedsWidening は upstream の truncation や未取得件数により探索拡張が必要か返す。
func NeedsWidening(result navigation.InspectResult) bool {
	if result.MoreCallers || result.MoreRefs || result.UpstreamTruncated || result.UpstreamIncomplete {
		return true
	}
	return result.TotalCallers > len(result.Callers) || result.TotalRefs > len(result.Refs)
}

// ReferenceSpread は caller/ref がまたがる file 数と directory 数を返す。
func ReferenceSpread(result navigation.InspectResult) (int, int) {
	fileSeen := make(map[string]struct{})
	dirSeen := make(map[string]struct{})
	add := func(file string) {
		file = strings.TrimSpace(file)
		if file == "" {
			return
		}
		file = filepath.ToSlash(filepath.Clean(file))
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

// IsSharedPackageSymbol は cmd 以外の共有 package にある symbol か返す。
func IsSharedPackageSymbol(symbol navigation.SymbolCandidate) bool {
	packageDir := filepath.ToSlash(strings.TrimSpace(symbol.PackageDir))
	if packageDir == "" || packageDir == "." {
		return false
	}
	return packageDir != "cmd" && !strings.HasPrefix(packageDir, "cmd/")
}
