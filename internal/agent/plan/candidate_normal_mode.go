package plan

import (
	"encoding/json"
	"strings"
)

func classifyNormalModePlanWrapperJSONCandidate(jsonStr string) planJSONCandidateKind {
	return classifyWrappedPlanJSONCandidate(jsonStr, classifyNormalModePlanObjectJSONCandidate)
}

func classifyNormalModePlanObjectJSONCandidate(jsonStr string) planJSONCandidateKind {
	if isNormalModePlanObjectShape(jsonStr) {
		return planJSONValidCandidate
	}
	if isNormalModePlanObjectLike(jsonStr) {
		return planJSONMalformedCandidate
	}
	return planJSONNotCandidate
}

func isNormalModePlanObjectShape(jsonStr string) bool {
	var p Plan
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return false
	}
	return hasNormalModeImplementationStepShape(p.Steps)
}

func hasNormalModeImplementationStepShape(steps []PlanStep) bool {
	for _, step := range steps {
		if normalModeStepHasImplementationSignal(step) {
			return true
		}
	}
	return false
}

func normalModeStepHasImplementationSignal(step PlanStep) bool {
	return strings.TrimSpace(step.Description) != "" ||
		strings.TrimSpace(step.Purpose) != "" ||
		hasNonEmptyPlanValues(step.Tools, step.Files, step.Verification)
}

func isNormalModePlanObjectLike(jsonStr string) bool {
	steps, ok, valid := topLevelRawValue(jsonStr, legacyPlanStepsKey)
	if !ok {
		return false
	}
	if !valid {
		return true
	}
	return isNormalModeImplementationStepValueLike(steps)
}

func isNormalModeImplementationStepValueLike(value json.RawMessage) bool {
	switch jsonValueKindAt(string(value), 0) {
	case jsonValueObject:
		return hasNormalModeImplementationStepKeys(value)
	case jsonValueArray:
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			return true
		}
		for _, item := range items {
			if hasNormalModeImplementationStepKeys(item) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func hasNormalModeImplementationStepKeys(value json.RawMessage) bool {
	if jsonValueKindAt(string(value), 0) != jsonValueObject {
		return false
	}

	var step PlanStep
	if err := json.Unmarshal(value, &step); err == nil {
		return normalModeStepHasImplementationSignal(step)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(value, &obj); err != nil {
		return true
	}
	for _, key := range normalModeImplementationStepJSONKeys {
		if _, ok := obj[key]; ok {
			return true
		}
	}
	return false
}
