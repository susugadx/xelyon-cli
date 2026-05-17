package impactplan

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

const (
	// RiskLow は低リスクの structured impact を表す。
	RiskLow = "low"
	// RiskMedium は中リスクの structured impact を表す。
	RiskMedium = "medium"
	// RiskHigh は高リスクの structured impact を表す。
	RiskHigh = "high"
)

// Plan は structured impact analysis で使う探索 budget と出力上限を表す。
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

// PlanForRisk は risk level に応じた impact plan を返す。
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

// PlanEqual は 2 つの impact plan が同じか返す。
func PlanEqual(left, right Plan) bool {
	return left.RiskLevel == right.RiskLevel &&
		left.ImplementationLimit == right.ImplementationLimit &&
		left.Budget == right.Budget
}

// PlanRank は impact plan の risk 順位を返す。
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
