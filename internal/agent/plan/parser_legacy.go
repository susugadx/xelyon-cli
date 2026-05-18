package plan

import (
	"encoding/json"
	"fmt"
)

func parseLegacyPlan(jsonStr string) (*Plan, error) {
	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}
	if !hasPlanContent(plan) && (isPlanObjectLike(jsonStr) || hasLegacyPlanTopLevelEvidence(jsonStr)) {
		return nil, fmt.Errorf("failed to parse plan: no plan content")
	}
	markPlanStepsPending(&plan)
	return &plan, nil
}

func markPlanStepsPending(plan *Plan) {
	for i := range plan.Steps {
		plan.Steps[i].Status = "pending"
	}
}
