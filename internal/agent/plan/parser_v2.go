package plan

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/plancontract"
)

func parsePlanV2IfPresent(jsonStr string) (*Plan, bool, error) {
	value, ok, valid := topLevelRawValue(jsonStr, planSchemaVersionJSONKey)
	if !ok {
		return nil, false, nil
	}
	if !valid {
		return nil, true, fmt.Errorf("failed to parse %s: invalid JSON", plancontract.SchemaVersion)
	}
	var schemaVersion string
	if err := decodeRawJSONString(value, &schemaVersion); err != nil {
		if !hasPlanV2ShapeSignal(jsonStr) {
			return nil, false, nil
		}
		return nil, true, fmt.Errorf("failed to parse %s schema_version: %w", plancontract.SchemaVersion, err)
	}
	if schemaVersion != plancontract.SchemaVersion {
		if isMismatchedPlanV2Attempt(jsonStr, schemaVersion) {
			return nil, true, fmt.Errorf("schema_version must be %q: got %q", plancontract.SchemaVersion, schemaVersion)
		}
		return nil, false, nil
	}
	doc, err := plancontract.DecodeStrict([]byte(jsonStr))
	if err != nil {
		return nil, true, err
	}
	plan := runtimePlanFromV2Document(doc)
	markPlanStepsPending(&plan)
	return &plan, true, nil
}

func runtimePlanFromV2Document(doc plancontract.Document) Plan {
	plan := Plan{
		Summary:            strings.TrimSpace(doc.Goal),
		AcceptanceCriteria: compactPlanStrings(doc.AcceptanceCriteria),
		Constraints:        compactPlanStrings(doc.Constraints),
		OpenQuestions:      compactPlanStrings(doc.OpenQuestions),
		Steps:              make([]PlanStep, 0, len(doc.Steps)),
	}
	for _, finding := range doc.Findings {
		fact := strings.TrimSpace(finding.Fact)
		if fact != "" {
			plan.Findings = append(plan.Findings, fact)
		}
		plan.Evidence = append(plan.Evidence, compactPlanStrings(finding.Evidence)...)
	}
	plan.Evidence = compactPlanStrings(plan.Evidence)

	for idx, step := range doc.Steps {
		plan.Steps = append(plan.Steps, PlanStep{
			ID:           normalizePlanV2StepID(step.ID, idx),
			Description:  strings.TrimSpace(step.Outcome),
			Purpose:      strings.TrimSpace(step.Reason),
			Files:        compactPlanStrings(step.Files),
			Verification: compactPlanStrings(step.Verification),
		})
	}
	return plan
}

func normalizePlanV2StepID(id string, index int) int {
	id = strings.TrimSpace(id)
	if parsed, err := strconv.Atoi(id); err == nil && parsed > 0 {
		return parsed
	}
	return index + 1
}

func compactPlanStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
