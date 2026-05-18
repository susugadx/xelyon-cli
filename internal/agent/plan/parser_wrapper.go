package plan

import (
	"encoding/json"
	"fmt"
)

func parseWrappedPlanIfPresent(jsonStr string) (*Plan, bool, error) {
	value, ok, valid := topLevelRawValue(jsonStr, planWrapperJSONKey)
	if !ok {
		return nil, false, nil
	}
	if !valid {
		if planValueLooksLikeMalformedWrapper(jsonStr) {
			return nil, true, fmt.Errorf("failed to parse plan wrapper: invalid JSON")
		}
		return nil, false, nil
	}

	valueStr := string(value)
	if jsonValueKindAt(valueStr, 0) != jsonValueObject {
		return nil, false, nil
	}
	if classifyPlanObjectJSONCandidate(valueStr) == planJSONNotCandidate {
		return nil, false, nil
	}

	wrappedPlan, err := parseWrappedPlanObject(value)
	if err != nil {
		return nil, true, err
	}
	return wrappedPlan, true, nil
}

func planValueLooksLikeMalformedWrapper(jsonStr string) bool {
	valueStart, ok := topLevelValueStart(jsonStr, planWrapperJSONKey)
	if !ok {
		return false
	}
	switch jsonValueKindAt(jsonStr, valueStart) {
	case jsonValueObject, jsonValueInvalid:
		return true
	default:
		return false
	}
}

func parseWrappedPlanObject(value json.RawMessage) (*Plan, error) {
	var wrappedPlan Plan
	if err := json.Unmarshal(value, &wrappedPlan); err != nil {
		return nil, fmt.Errorf("failed to parse plan wrapper: %w", err)
	}
	if !hasPlanContent(wrappedPlan) {
		return nil, fmt.Errorf("failed to parse plan wrapper: no plan content")
	}
	markPlanStepsPending(&wrappedPlan)
	return &wrappedPlan, nil
}
