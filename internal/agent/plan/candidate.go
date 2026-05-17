package plan

import (
	"encoding/json"
	"strings"
)

const (
	planWrapperJSONKey              = "plan"
	planSummaryJSONKey              = "summary"
	legacyPlanStepsKey              = "steps"
	legacyPlanGoalKey               = "goal"
	legacyPlanAssumptionsKey        = "assumptions"
	legacyPlanExpectedOutputStepKey = "expected_output"
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
	case planJSONCandidateLegacy:
		return classifyLegacyPlanJSONCandidate(jsonStr)
	}

	if kind := classifyPlanWrapperJSONCandidate(jsonStr); kind != planJSONNotCandidate {
		return kind
	}
	return classifyLegacyPlanJSONCandidate(jsonStr)
}

func classifyPlanWrapperJSONCandidate(jsonStr string) planJSONCandidateKind {
	value, ok, valid := topLevelRawValue(jsonStr, planWrapperJSONKey)
	if !ok {
		return planJSONNotCandidate
	}

	if valid {
		valueStr := string(value)
		if jsonValueKindAt(valueStr, 0) != jsonValueObject {
			return planJSONNotCandidate
		}
		return classifyPlanObjectJSONCandidate(valueStr)
	}

	valueStart, ok := topLevelValueStart(jsonStr, planWrapperJSONKey)
	if !ok {
		return planJSONNotCandidate
	}
	switch jsonValueKindAt(jsonStr, valueStart) {
	case jsonValueObject, jsonValueInvalid:
		return planJSONMalformedCandidate
	default:
		return planJSONNotCandidate
	}
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
	return isPlanShape(p)
}

func isLegacyPlanObjectShape(jsonStr string) bool {
	p, err := ParsePlan(jsonStr)
	if err != nil {
		return false
	}
	return isPlanShape(*p)
}

func isPlanObjectLike(jsonStr string) bool {
	if hasTopLevelJSONKey(jsonStr, planSummaryJSONKey) {
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

func hasTopLevelJSONKey(jsonStr string, key string) bool {
	_, ok, _ := topLevelRawValue(jsonStr, key)
	return ok
}

func hasLegacyPlanEvidence(jsonStr string, steps json.RawMessage) bool {
	return hasTopLevelJSONKey(jsonStr, planSummaryJSONKey) ||
		hasLegacyPlanTopLevelEvidence(jsonStr) ||
		hasPlanSpecificStepEvidence(steps)
}

func hasLegacyPlanTopLevelEvidence(jsonStr string) bool {
	for _, key := range []string{legacyPlanGoalKey, legacyPlanAssumptionsKey} {
		if hasTopLevelJSONKey(jsonStr, key) {
			return true
		}
	}
	return false
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
	for _, key := range []string{"tools", "depends_on", "files", legacyPlanExpectedOutputStepKey} {
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
	for _, key := range []string{"id", "description", "tools", "depends_on", "files"} {
		if _, ok := obj[key]; ok {
			return true
		}
	}
	return false
}

func isPlanShape(p Plan) bool {
	if strings.TrimSpace(p.Summary) != "" {
		return true
	}
	for _, step := range p.Steps {
		if isPlanStepShape(step) {
			return true
		}
	}
	return false
}

func isPlanStepShape(step PlanStep) bool {
	return step.ID != 0 ||
		strings.TrimSpace(step.Description) != "" ||
		len(step.Tools) > 0 ||
		len(step.DependsOn) > 0 ||
		len(step.Files) > 0
}

// isToolCallJSON はJSONがツール呼び出しかどうかを判定
func isToolCallJSON(jsonStr string) bool {
	_, ok, _ := topLevelRawValue(jsonStr, "tool")
	return ok
}
