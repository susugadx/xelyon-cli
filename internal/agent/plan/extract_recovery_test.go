package plan

import "testing"

func TestExtractPlanJSONForNormalModeRecovery_UsesWrapperOnly(t *testing.T) {
	legacy := `Here is a legacy plan:
{"summary":"Fix","steps":[{"id":1,"description":"Do it","tools":["str_replace"]}]}
Done.`
	if got := ExtractPlanJSON(legacy); got == "" {
		t.Fatal("ExtractPlanJSON() returned empty for plan-mode legacy candidate")
	}
	if got := ExtractPlanJSONForNormalModeRecovery(legacy); got != "" {
		t.Fatalf("ExtractPlanJSONForNormalModeRecovery() = %q, want empty for legacy candidate", got)
	}

	wrapper := `Here is a wrapper plan:
{"plan":{"summary":"Fix","steps":[{"id":1,"description":"Do it"}]}}
Done.`
	if got := ExtractPlanJSONForNormalModeRecovery(wrapper); got == "" {
		t.Fatal("ExtractPlanJSONForNormalModeRecovery() returned empty for wrapper candidate")
	}
}

func TestExtractPlanJSONForNormalModeRecovery_IgnoresTopLevelV2Malformed(t *testing.T) {
	response := `{"schema_version":"xelyon.plan.v2","goal":"Fix","acceptance_criteria":[],"findings":[],"constraints":[],"open_questions":[]}`

	if got := ExtractPlanJSON(response); got == "" {
		t.Fatal("ExtractPlanJSON() should accept malformed top-level v2 for plan-mode retry")
	}
	if got := ExtractPlanJSONForNormalModeRecovery(response); got != "" {
		t.Fatalf("ExtractPlanJSONForNormalModeRecovery() = %q, want empty for top-level v2 malformed", got)
	}
}

func TestExtractPlanJSONForNormalModeRecovery_RequiresImplementationStep(t *testing.T) {
	response := `{"plan":{"summary":"Already done","findings":["Already done"],"evidence":["README.md"],"constraints":["Do not edit"],"steps":[]}}`

	if got := ExtractPlanJSON(response); got == "" {
		t.Fatal("ExtractPlanJSON() should still accept no-op plan mode output")
	}
	if got := ExtractPlanJSONForNormalModeRecovery(response); got != "" {
		t.Fatalf("ExtractPlanJSONForNormalModeRecovery() = %q, want empty", got)
	}
}

func TestExtractPlanJSONForNormalModeRecovery_IgnoresMalformedHandoffOnlyFields(t *testing.T) {
	response := `{"plan":{"findings":{},"evidence":["README.md"],"constraints":["Do not edit"],"steps":[]}}`

	if got := ExtractPlanJSON(response); got == "" {
		t.Fatal("ExtractPlanJSON() should accept malformed handoff-only plan mode output for retry")
	}
	if got := ExtractPlanJSONForNormalModeRecovery(response); got != "" {
		t.Fatalf("ExtractPlanJSONForNormalModeRecovery() = %q, want empty", got)
	}
}

func TestExtractPlanJSONForNormalModeRecovery_RequiresRealImplementationStep(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "id only",
			response: `{"plan":{"summary":"Fix","steps":[{"id":1}]}}`,
		},
		{
			name:     "depends only",
			response: `{"plan":{"summary":"Fix","steps":[{"id":2,"depends_on":[1]}]}}`,
		},
		{
			name:     "empty implementation fields",
			response: `{"plan":{"summary":"Fix","steps":[{"description":"","tools":[],"files":[],"verification":[]}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractPlanJSONForNormalModeRecovery(tt.response); got != "" {
				t.Fatalf("ExtractPlanJSONForNormalModeRecovery() = %q, want empty", got)
			}
		})
	}
}
