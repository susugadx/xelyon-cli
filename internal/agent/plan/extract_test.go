package plan

import "testing"

func TestExtractPlanJSON_V2Pattern(t *testing.T) {
	response := `Here is the plan:
{"plan": {"summary": "test", "steps": [{"id": 1, "description": "Step 1"}]}}
End of response.`

	result := mustExtractPlanJSON(t, response)
	mustParsePlanJSON(t, result)
}

func TestExtractPlanJSON_DoesNotTreatToolStringValueAsToolCall(t *testing.T) {
	response := `Here is the plan:
{"plan": {"summary": "tool", "steps": [{"id": 1, "description": "Use the tool label"}]}}
End of response.`

	result := mustExtractPlanJSON(t, response)
	mustParsePlanJSON(t, result)
}

func TestExtractPlanJSON_V2MalformedWrapperReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"plan": invalid}
End of response.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"plan": invalid}` {
		t.Fatalf("ExtractPlanJSON() = %q, want malformed wrapper candidate", result)
	}
	if _, err := ParsePlan(result); err == nil {
		t.Fatal("ParsePlan() should fail for malformed wrapper candidate")
	}
}

func TestExtractPlanJSON_V2SchemaInvalidWrapperReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"plan":{"summary":"Fix","steps":{"id":1,"description":"Do it"}}}
End of response.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"plan":{"summary":"Fix","steps":{"id":1,"description":"Do it"}}}` {
		t.Fatalf("ExtractPlanJSON() = %q, want schema-invalid wrapper candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
}

func TestExtractPlanJSON_WrapperRequiresPlanShape(t *testing.T) {
	response := `Here is ordinary JSON:
{"title":"monthly","plan":"free"}
Done.`

	if got := ExtractPlanJSON(response); got != "" {
		t.Fatalf("ExtractPlanJSON() = %q, want empty for unrelated plan field", got)
	}
}

func TestExtractPlanJSON_WrapperObjectRequiresPlanShape(t *testing.T) {
	response := `Here is ordinary JSON:
{"plan":{"tier":"free"}}
Done.`

	if got := ExtractPlanJSON(response); got != "" {
		t.Fatalf("ExtractPlanJSON() = %q, want empty for unrelated plan object", got)
	}
}

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

func TestExtractPlanJSON_V2PrettyPrintedWithoutCodeFence(t *testing.T) {
	response := `Here is the plan:
{
  "plan": {
    "summary": "test",
    "steps": [
      {
        "id": 1,
        "description": "Step 1",
        "files": ["internal/agent/plan/parser.go"]
      }
    ]
  }
}
End of response.`

	parsed := mustParsePlanJSON(t, mustExtractPlanJSON(t, response))
	assertStringSliceEqual(t, "parsed step files", parsed.Steps[0].Files, []string{"internal/agent/plan/parser.go"})
}

func TestExtractPlanJSON_V2PrettyPrintedMalformedWrapperReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{
  "plan": invalid
}
End of response.`

	result := mustExtractPlanJSON(t, response)
	if result != "{\n  \"plan\": invalid\n}" {
		t.Fatalf("ExtractPlanJSON() = %q, want pretty malformed wrapper candidate", result)
	}
	if _, err := ParsePlan(result); err == nil {
		t.Fatal("ParsePlan() should fail for malformed wrapper candidate")
	}
}

func TestExtractPlanJSON_V2PatternAfterUnmatchedBrace(t *testing.T) {
	response := `Some prose with an unmatched brace {
Here is the plan:
{
  "plan": {
    "summary": "test",
    "steps": [{"id": 1, "description": "Step 1"}]
  }
}`

	mustExtractPlanJSON(t, response)
}

func TestExtractPlanJSON_CodeBlock(t *testing.T) {
	response := "Here is the plan:\n```json\n{\"summary\": \"test\", \"steps\": [{\"id\": 1, \"description\": \"Step 1\"}]}\n```"

	mustExtractPlanJSON(t, response)
}

func TestExtractPlanJSON_StepsPattern(t *testing.T) {
	response := `I'll create a plan:
{"steps": [{"id": 1, "description": "First step", "tools": ["read_file"]}]}
Done.`

	mustExtractPlanJSON(t, response)
}

func TestExtractPlanJSON_LegacyPrettyPrintedWithoutCodeFence(t *testing.T) {
	response := `I'll create a plan:
{
  "summary": "test",
  "steps": [
    {"id": 1, "description": "First step"}
  ]
}
Done.`

	mustExtractPlanJSON(t, response)
}

func TestExtractPlanJSON_LegacyStepsRequiresPlanShape(t *testing.T) {
	response := `Here is unrelated JSON:
{"title":"recipe","steps":["mix","bake"]}
Done.`

	if got := ExtractPlanJSON(response); got != "" {
		t.Fatalf("ExtractPlanJSON() = %q, want empty for unrelated steps JSON", got)
	}
}

func TestExtractPlanJSON_LegacyObjectStepsRequiresPlanSpecificEvidence(t *testing.T) {
	response := `Here is unrelated JSON:
{"title":"recipe","steps":[{"id":1,"description":"mix"}]}
Done.`

	if got := ExtractPlanJSON(response); got != "" {
		t.Fatalf("ExtractPlanJSON() = %q, want empty for object steps without legacy plan evidence", got)
	}
}

func TestExtractPlanJSON_LegacyStepsPurposeAndVerificationArePlanEvidence(t *testing.T) {
	response := `Here is the plan:
{"steps":[{"id":1,"description":"Update review","purpose":"Clarify approval","verification":["go test ./internal/ui"]}]}
Done.`

	parsed := mustParsePlanJSON(t, mustExtractPlanJSON(t, response))
	if parsed.Steps[0].Purpose != "Clarify approval" {
		t.Fatalf("parsed purpose = %q, want plan purpose", parsed.Steps[0].Purpose)
	}
	assertStringSliceEqual(t, "parsed verification", parsed.Steps[0].Verification, []string{"go test ./internal/ui"})
}

func TestExtractPlanJSON_FencedLegacyRetrySchemaReturnsCandidate(t *testing.T) {
	response := "Here is the plan:\n```json\n" + `{
  "title": "Fix parser",
  "goal": "Preserve legacy plan compatibility",
  "assumptions": ["Parser still accepts legacy steps"],
  "steps": [
    {
      "id": 1,
      "description": "Update legacy evidence",
      "expected_output": "Legacy fenced plan is extracted"
    }
  ]
}` + "\n```\nDone."

	parsed := mustParsePlanJSON(t, mustExtractPlanJSON(t, response))
	if parsed.Title != "Fix parser" {
		t.Fatalf("parsed.Title = %q, want legacy title", parsed.Title)
	}
	if len(parsed.Steps) != 1 || parsed.Steps[0].Description != "Update legacy evidence" {
		t.Fatalf("parsed steps = %#v, want legacy step", parsed.Steps)
	}
}

func TestExtractPlanJSON_LegacyMalformedStepsReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"steps": invalid}
Done.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"steps": invalid}` {
		t.Fatalf("ExtractPlanJSON() = %q, want malformed legacy candidate", result)
	}
	if _, err := ParsePlan(result); err == nil {
		t.Fatal("ParsePlan() should fail for malformed legacy candidate")
	}
}

func TestExtractPlanJSON_LegacySchemaInvalidStepsReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"summary":"Fix","steps":{"id":1,"description":"Do it"}}
Done.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"summary":"Fix","steps":{"id":1,"description":"Do it"}}` {
		t.Fatalf("ExtractPlanJSON() = %q, want schema-invalid legacy candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
}

func TestExtractPlanJSON_CodeBlockStepsRequiresPlanShape(t *testing.T) {
	response := "Here is unrelated JSON:\n```json\n" + `{"title":"recipe","steps":["mix","bake"]}` + "\n```"

	if got := ExtractPlanJSON(response); got != "" {
		t.Fatalf("ExtractPlanJSON() = %q, want empty for unrelated code block steps JSON", got)
	}
}

func TestExtractPlanJSON_NoJSON(t *testing.T) {
	response := "This is just plain text without any JSON."

	result := ExtractPlanJSON(response)

	if result != "" {
		t.Errorf("Expected empty string for text without JSON, got %q", result)
	}
}

func TestExtractPlanJSON_ToolCallExcluded(t *testing.T) {
	response := `I'll use a tool:
` + "```json\n" + `{"tool": "read_file", "path": "main.go"}` + "\n```"

	result := ExtractPlanJSON(response)

	if result != "" {
		t.Errorf("Tool call JSON should be excluded, got %q", result)
	}
}

func TestExtractPlanJSON_ToolCallCandidatesExcludedAcrossScans(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name: "code block with legacy-shaped steps",
			response: "I'll use a tool:\n```json\n" +
				`{"tool":"read_file","steps":[{"id":1,"description":"Read parser","tools":["read_file"]}],"args":{"paths":["internal/agent/plan/parser.go"]}}` +
				"\n```",
		},
		{
			name:     "unfenced legacy-shaped steps",
			response: `{"tool":"read_file","steps":[{"id":1,"description":"Read parser","tools":["read_file"]}],"args":{"paths":["internal/agent/plan/parser.go"]}}`,
		},
		{
			name:     "unfenced wrapper-shaped plan",
			response: `{"tool":"read_file","plan":{"summary":"Read parser","steps":[{"id":1,"description":"Read parser","tools":["read_file"]}]},"args":{"paths":["internal/agent/plan/parser.go"]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractPlanJSON(tt.response); got != "" {
				t.Fatalf("ExtractPlanJSON() = %q, want empty for tool-call JSON", got)
			}
		})
	}
}
