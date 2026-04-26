package search

import (
	"github.com/susugadx/xelyon-cli/internal/goimpact"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

const (
	goImpactRiskLow    = goimpact.RiskLow
	goImpactRiskMedium = goimpact.RiskMedium
	goImpactRiskHigh   = goimpact.RiskHigh
)

type goImpactPlan struct {
	riskLevel           string
	budget              navigation.Budget
	implementationLimit int
}

func goImpactPlanForRisk(risk string) goImpactPlan {
	plan := goimpact.PlanForRisk(risk)
	return goImpactPlan{
		riskLevel:           plan.RiskLevel,
		budget:              plan.Budget,
		implementationLimit: plan.ImplementationLimit,
	}
}

func goImpactPlanEqual(left, right goImpactPlan) bool {
	return goimpact.PlanEqual(toGoImpactPlan(left), toGoImpactPlan(right))
}

func goImpactPlanRank(plan goImpactPlan) int {
	return goimpact.PlanRank(toGoImpactPlan(plan))
}

func classifyGoImpactRisk(result navigation.InspectResult) string {
	return goimpact.ClassifyRisk(result)
}

func toGoImpactPlan(plan goImpactPlan) goimpact.Plan {
	return goimpact.Plan{
		RiskLevel:           plan.riskLevel,
		Budget:              plan.budget,
		ImplementationLimit: plan.implementationLimit,
	}
}
