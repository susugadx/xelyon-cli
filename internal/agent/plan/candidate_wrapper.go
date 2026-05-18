package plan

func classifyPlanWrapperJSONCandidate(jsonStr string) planJSONCandidateKind {
	return classifyWrappedPlanJSONCandidate(jsonStr, classifyPlanObjectJSONCandidate)
}

func classifyWrappedPlanJSONCandidate(jsonStr string, classifyObject func(string) planJSONCandidateKind) planJSONCandidateKind {
	value, ok, valid := topLevelRawValue(jsonStr, planWrapperJSONKey)
	if !ok {
		return planJSONNotCandidate
	}

	if valid {
		valueStr := string(value)
		if jsonValueKindAt(valueStr, 0) != jsonValueObject {
			return planJSONNotCandidate
		}
		return classifyObject(valueStr)
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
