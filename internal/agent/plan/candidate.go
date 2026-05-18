package plan

import (
	"encoding/json"
)

type planJSONCandidateKind int

const (
	planJSONNotCandidate planJSONCandidateKind = iota
	planJSONValidCandidate
	planJSONMalformedCandidate
)

type planJSONCandidateScope int

const (
	planJSONCandidateAny planJSONCandidateScope = iota
	planJSONCandidateWrapper
	planJSONCandidateNormalModeWrapper
	planJSONCandidateLegacy
)

func isPlanJSONCandidate(jsonStr string) bool {
	return isPlanJSONCandidateForScope(jsonStr, planJSONCandidateAny)
}

func isPlanJSONCandidateForScope(jsonStr string, scope planJSONCandidateScope) bool {
	return classifyPlanJSONCandidateForScope(jsonStr, scope) != planJSONNotCandidate
}

func classifyPlanJSONCandidateForScope(jsonStr string, scope planJSONCandidateScope) planJSONCandidateKind {
	if isToolCallJSON(jsonStr) {
		return planJSONNotCandidate
	}

	switch scope {
	case planJSONCandidateWrapper:
		return classifyPlanWrapperJSONCandidate(jsonStr)
	case planJSONCandidateNormalModeWrapper:
		return classifyNormalModePlanWrapperJSONCandidate(jsonStr)
	case planJSONCandidateLegacy:
		return classifyLegacyPlanJSONCandidate(jsonStr)
	}

	if kind := classifyPlanWrapperJSONCandidate(jsonStr); kind != planJSONNotCandidate {
		return kind
	}
	return classifyLegacyPlanJSONCandidate(jsonStr)
}

func classifyLegacyPlanJSONCandidate(jsonStr string) planJSONCandidateKind {
	value, ok, valid := topLevelRawValue(jsonStr, legacyPlanStepsKey)
	if !ok {
		return planJSONNotCandidate
	}

	if valid {
		return classifyLegacyPlanObjectJSONCandidate(jsonStr, value)
	}

	valueStart, ok := topLevelValueStart(jsonStr, legacyPlanStepsKey)
	if !ok {
		return planJSONNotCandidate
	}
	switch jsonValueKindAt(jsonStr, valueStart) {
	case jsonValueArray, jsonValueInvalid:
		return planJSONMalformedCandidate
	default:
		return planJSONNotCandidate
	}
}

func classifyPlanObjectJSONCandidate(jsonStr string) planJSONCandidateKind {
	if isPlanObjectShape(jsonStr) {
		return planJSONValidCandidate
	}
	if isPlanObjectLike(jsonStr) {
		return planJSONMalformedCandidate
	}
	return planJSONNotCandidate
}

func classifyLegacyPlanObjectJSONCandidate(jsonStr string, steps json.RawMessage) planJSONCandidateKind {
	if !hasLegacyPlanEvidence(jsonStr, steps) {
		return planJSONNotCandidate
	}
	if isLegacyPlanObjectShape(jsonStr) {
		return planJSONValidCandidate
	}
	return planJSONMalformedCandidate
}

func isPlanObjectShape(jsonStr string) bool {
	var p Plan
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return false
	}
	return hasPlanContent(p)
}

func isLegacyPlanObjectShape(jsonStr string) bool {
	p, err := ParsePlan(jsonStr)
	if err != nil {
		return false
	}
	return hasPlanContent(*p)
}

func isPlanObjectLike(jsonStr string) bool {
	if hasPlanContentKey(jsonStr) {
		return true
	}

	steps, ok, valid := topLevelRawValue(jsonStr, legacyPlanStepsKey)
	if !ok {
		return false
	}
	if !valid {
		return true
	}
	return isPlanStepsValueLike(steps)
}

func hasLegacyPlanEvidence(jsonStr string, steps json.RawMessage) bool {
	return hasPlanContentKey(jsonStr) ||
		hasLegacyPlanTopLevelEvidence(jsonStr) ||
		hasPlanSpecificStepEvidence(steps)
}

func hasLegacyPlanTopLevelEvidence(jsonStr string) bool {
	return hasAnyTopLevelJSONKey(jsonStr, legacyPlanTopLevelEvidenceJSONKeys[:])
}

func hasPlanSpecificStepEvidence(value json.RawMessage) bool {
	switch jsonValueKindAt(string(value), 0) {
	case jsonValueObject:
		return hasPlanSpecificStepKey(value)
	case jsonValueArray:
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			return true
		}
		for _, item := range items {
			if hasPlanSpecificStepKey(item) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func hasPlanSpecificStepKey(value json.RawMessage) bool {
	if jsonValueKindAt(string(value), 0) != jsonValueObject {
		return false
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(value, &obj); err != nil {
		return true
	}
	for _, key := range planSpecificStepEvidenceJSONKeys {
		if _, ok := obj[key]; ok {
			return true
		}
	}
	return false
}

func isPlanStepsValueLike(value json.RawMessage) bool {
	switch jsonValueKindAt(string(value), 0) {
	case jsonValueObject:
		return isPlanStepObjectLike(value)
	case jsonValueArray:
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			return true
		}
		for _, item := range items {
			if isPlanStepObjectLike(item) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func isPlanStepObjectLike(value json.RawMessage) bool {
	if jsonValueKindAt(string(value), 0) != jsonValueObject {
		return false
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(value, &obj); err != nil {
		return true
	}
	for _, key := range planStepShapeJSONKeys {
		if _, ok := obj[key]; ok {
			return true
		}
	}
	return false
}

// isToolCallJSON はJSONがツール呼び出しかどうかを判定
func isToolCallJSON(jsonStr string) bool {
	_, ok, _ := topLevelRawValue(jsonStr, "tool")
	return ok
}
