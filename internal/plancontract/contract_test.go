package plancontract

import (
	"strings"
	"testing"
)

func TestDecodeStrictAcceptsPlanV2Document(t *testing.T) {
	doc, err := DecodeStrict([]byte(validPlanV2JSON()))
	if err != nil {
		t.Fatalf("DecodeStrict() error = %v", err)
	}
	if doc.SchemaVersion != SchemaVersion || doc.Goal != "Fix prompt contract" {
		t.Fatalf("DecodeStrict() = %+v", doc)
	}
}

func TestDecodeStrictRejectsUnknownField(t *testing.T) {
	raw := strings.Replace(validPlanV2JSON(), `"open_questions":[]`, `"open_questions":[],"extra":true`, 1)
	if _, err := DecodeStrict([]byte(raw)); err == nil {
		t.Fatal("DecodeStrict() error = nil, want unknown field rejection")
	}
}

func TestDecodeStrictRejectsMissingTopLevelField(t *testing.T) {
	raw := strings.Replace(validPlanV2JSON(), `"constraints":[],`, "", 1)
	if _, err := DecodeStrict([]byte(raw)); err == nil || !strings.Contains(err.Error(), "constraints") {
		t.Fatalf("DecodeStrict() error = %v, want missing constraints", err)
	}
}

func TestDecodeStrictRejectsMissingNestedStepField(t *testing.T) {
	raw := strings.Replace(validPlanV2JSON(), `,"verification":["go test ./internal/prompt"]`, "", 1)
	if _, err := DecodeStrict([]byte(raw)); err == nil || !strings.Contains(err.Error(), "steps[0]") || !strings.Contains(err.Error(), "verification") {
		t.Fatalf("DecodeStrict() error = %v, want missing step verification", err)
	}
}

func TestDecodeStrictAllowsNoStepPlanV2Document(t *testing.T) {
	doc, err := DecodeStrict([]byte(validNoStepPlanV2JSON()))
	if err != nil {
		t.Fatalf("DecodeStrict() error = %v, want nil", err)
	}
	if len(doc.Steps) != 0 {
		t.Fatalf("len(Steps) = %d, want 0", len(doc.Steps))
	}
}

func TestSchemaInstructionsAllowEmptyStepsForNoOpPlan(t *testing.T) {
	got := SchemaInstructions()
	for _, want := range []string{"steps: use an empty array when no implementation is needed", "normally 2-6 steps"} {
		if !strings.Contains(got, want) {
			t.Fatalf("SchemaInstructions() missing %q:\n%s", want, got)
		}
	}
}

func validPlanV2JSON() string {
	return `{
		"schema_version":"xelyon.plan.v2",
		"goal":"Fix prompt contract",
		"acceptance_criteria":["tests pass"],
		"findings":[{"fact":"composer owns sections","evidence":["internal/prompt/composer.go"]}],
		"constraints":[],
		"steps":[{"id":"step-1","outcome":"Add contract","files":["internal/plancontract/contract.go"],"reason":"","verification":["go test ./internal/prompt"]}],
		"open_questions":[]
	}`
}

func validNoStepPlanV2JSON() string {
	return `{
		"schema_version":"xelyon.plan.v2",
		"goal":"Existing behavior already satisfies the request",
		"acceptance_criteria":["no code changes required"],
		"findings":[],
		"constraints":[],
		"steps":[],
		"open_questions":[]
	}`
}
