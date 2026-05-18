package plan

import "strings"

func hasPlanContent(p Plan) bool {
	if strings.TrimSpace(p.Summary) != "" {
		return true
	}
	if hasNonEmptyPlanValues(p.Findings, p.Evidence, p.Constraints) {
		return true
	}
	for _, step := range p.Steps {
		if hasPlanStepContent(step) {
			return true
		}
	}
	return false
}

func hasPlanStepContent(step PlanStep) bool {
	return step.ID != 0 ||
		strings.TrimSpace(step.Description) != "" ||
		strings.TrimSpace(step.Purpose) != "" ||
		len(step.Tools) > 0 ||
		len(step.DependsOn) > 0 ||
		len(step.Files) > 0 ||
		len(step.Verification) > 0
}

func hasNonEmptyPlanValues(groups ...[]string) bool {
	for _, group := range groups {
		for _, value := range group {
			if strings.TrimSpace(value) != "" {
				return true
			}
		}
	}
	return false
}

func hasPlanContentKey(jsonStr string) bool {
	return hasAnyTopLevelJSONKey(jsonStr, planContentJSONKeys[:])
}

func hasAnyTopLevelJSONKey(jsonStr string, keys []string) bool {
	for _, key := range keys {
		if hasTopLevelJSONKey(jsonStr, key) {
			return true
		}
	}
	return false
}
