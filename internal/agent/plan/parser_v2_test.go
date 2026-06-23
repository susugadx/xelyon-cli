package plan

import (
	"strings"
	"testing"
)

func TestParsePlan_XelyonPlanV2(t *testing.T) {
	jsonStr := `{
		"schema_version": "xelyon.plan.v2",
		"goal": "Implement prompt contract",
		"acceptance_criteria": ["composer tests pass"],
		"findings": [
			{"fact": "plan parser owns runtime normalization", "evidence": ["internal/agent/plan/parser_v2.go"]}
		],
		"constraints": ["keep legacy wrapper compatibility"],
		"steps": [
			{"id": "step-setup", "outcome": "Add contract package", "files": ["internal/plancontract/contract.go"], "reason": "source of truth", "verification": ["go test ./internal/agent/plan"]},
			{"id": "2", "outcome": "Parse v2", "files": ["internal/agent/plan/parser_v2.go"], "reason": "runtime normalization", "verification": []}
		],
		"open_questions": []
	}`

	parsed, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}
	if parsed.Summary != "Implement prompt contract" {
		t.Fatalf("Summary = %q", parsed.Summary)
	}
	if got := parsed.AcceptanceCriteria; len(got) != 1 || got[0] != "composer tests pass" {
		t.Fatalf("AcceptanceCriteria = %#v", got)
	}
	if got := parsed.Findings; len(got) != 1 || got[0] != "plan parser owns runtime normalization" {
		t.Fatalf("Findings = %#v", got)
	}
	if got := parsed.Evidence; len(got) != 1 || got[0] != "internal/agent/plan/parser_v2.go" {
		t.Fatalf("Evidence = %#v", got)
	}
	if len(parsed.Steps) != 2 {
		t.Fatalf("Steps = %#v", parsed.Steps)
	}
	if parsed.Steps[0].ID != 1 {
		t.Fatalf("first step ID = %d, want normalized sequence ID 1", parsed.Steps[0].ID)
	}
	if parsed.Steps[1].ID != 2 {
		t.Fatalf("second step ID = %d, want parsed numeric string ID 2", parsed.Steps[1].ID)
	}
	if parsed.Steps[0].Description != "Add contract package" || parsed.Steps[0].Purpose != "source of truth" {
		t.Fatalf("first step = %#v", parsed.Steps[0])
	}
}

func TestParsePlan_XelyonPlanV2RejectsUnknownField(t *testing.T) {
	jsonStr := `{
		"schema_version": "xelyon.plan.v2",
		"goal": "Implement prompt contract",
		"acceptance_criteria": [],
		"findings": [],
		"constraints": [],
		"steps": [{"id": "step-1", "outcome": "Do it", "files": [], "reason": "", "verification": []}],
		"open_questions": [],
		"extra": true
	}`
	if _, err := ParsePlan(jsonStr); err == nil {
		t.Fatal("ParsePlan() error = nil, want strict unknown field error")
	}
}

func TestParsePlan_XelyonPlanV2AllowsNoStepPlan(t *testing.T) {
	jsonStr := `{
		"schema_version": "xelyon.plan.v2",
		"goal": "Existing behavior already satisfies the request",
		"acceptance_criteria": ["no code changes required"],
		"findings": [],
		"constraints": [],
		"steps": [],
		"open_questions": []
	}`

	parsed, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v, want nil", err)
	}
	if parsed.Summary != "Existing behavior already satisfies the request" {
		t.Fatalf("Summary = %q", parsed.Summary)
	}
	if parsed.Steps == nil {
		t.Fatal("Steps = nil, want empty runtime slice")
	}
	if len(parsed.Steps) != 0 {
		t.Fatalf("len(Steps) = %d, want 0", len(parsed.Steps))
	}
}

func TestParsePlan_XelyonPlanV2SchemaTypoWithV2StepSignalReturnsError(t *testing.T) {
	jsonStr := `{
		"schema_version": "xelyon.plan.v2 ",
		"goal": "Fix parser recovery",
		"steps": [
			{"id": 1, "outcome": "Retry malformed v2", "reason": "Avoid legacy field loss"}
		]
	}`

	parsed, err := ParsePlan(jsonStr)
	if err == nil {
		t.Fatalf("ParsePlan() = %#v, want schema retry error", parsed)
	}
	if !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("ParsePlan() error = %v, want schema_version error", err)
	}
}

func TestParsePlan_LegacyShapedSchemaVersionWithoutV2SignalRemainsLegacy(t *testing.T) {
	jsonStr := `{
		"schema_version": "legacy.plan.v1",
		"summary": "Fix legacy compatibility",
		"steps": [
			{"id": 1, "description": "Keep legacy parser", "tools": ["go test"]}
		]
	}`

	parsed, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}
	if parsed.Summary != "Fix legacy compatibility" || len(parsed.Steps) != 1 || parsed.Steps[0].Description != "Keep legacy parser" {
		t.Fatalf("ParsePlan() = %#v, want legacy plan", parsed)
	}
}

func TestExtractPlanJSON_TopLevelXelyonPlanV2(t *testing.T) {
	response := "Plan:\n```json\n" + `{"schema_version":"xelyon.plan.v2","goal":"Fix","acceptance_criteria":[],"findings":[],"constraints":[],"steps":[{"id":"step-1","outcome":"Do it","files":[],"reason":"","verification":[]}],"open_questions":[]}` + "\n```"
	got := ExtractPlanJSON(response)
	if got == "" {
		t.Fatal("ExtractPlanJSON() returned empty for top-level xelyon.plan.v2")
	}
}

func TestExtractPlanJSON_TopLevelXelyonPlanV2WithoutCodeFence(t *testing.T) {
	response := `Plan:
{"schema_version":"xelyon.plan.v2","goal":"Fix","acceptance_criteria":[],"findings":[],"constraints":[],"steps":[{"id":"step-1","outcome":"Do it","files":[],"reason":"","verification":[]}],"open_questions":[]}
Done.`
	got := ExtractPlanJSON(response)
	if got == "" {
		t.Fatal("ExtractPlanJSON() returned empty for unfenced top-level xelyon.plan.v2")
	}
	mustParsePlanJSON(t, got)
}

func TestExtractPlanJSONForNormalModeRecovery_IgnoresTopLevelXelyonPlanV2(t *testing.T) {
	response := `{"schema_version":"xelyon.plan.v2","goal":"Fix","acceptance_criteria":[],"findings":[],"constraints":[],"steps":[{"id":"step-1","outcome":"Do it","files":[],"reason":"","verification":[]}],"open_questions":[]}`
	if got := ExtractPlanJSONForNormalModeRecovery(response); got != "" {
		t.Fatalf("ExtractPlanJSONForNormalModeRecovery() = %q, want empty for top-level v2", got)
	}
}
