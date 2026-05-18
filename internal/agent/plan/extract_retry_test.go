package plan

import "testing"

func TestExtractPlanJSON_V2MalformedWrapperReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"plan": invalid}
End of response.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"plan": invalid}` {
		t.Fatalf("ExtractPlanJSON() = %q, want malformed wrapper candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
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

func TestExtractPlanJSON_V2EmptyWrapperReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"plan":{"summary":""}}
End of response.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"plan":{"summary":""}}` {
		t.Fatalf("ExtractPlanJSON() = %q, want empty wrapper retry candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
}

func TestExtractPlanJSON_V2MalformedHandoffFieldsReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"plan":{"findings":{},"evidence":["README.md"],"constraints":["x"],"steps":[]}}
End of response.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"plan":{"findings":{},"evidence":["README.md"],"constraints":["x"],"steps":[]}}` {
		t.Fatalf("ExtractPlanJSON() = %q, want malformed handoff-field retry candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
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
	assertPlanJSONNeedsRetry(t, result)
}

func TestExtractPlanJSON_LegacyMalformedStepsReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"steps": invalid}
Done.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"steps": invalid}` {
		t.Fatalf("ExtractPlanJSON() = %q, want malformed legacy candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
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

func TestExtractPlanJSON_LegacyEmptyPlanReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"summary":"","steps":[]}
Done.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"summary":"","steps":[]}` {
		t.Fatalf("ExtractPlanJSON() = %q, want empty legacy retry candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
}

func TestExtractPlanJSON_LegacyMalformedHandoffFieldsReturnsRetryCandidate(t *testing.T) {
	response := `Here is the plan:
{"findings":{},"evidence":["README.md"],"constraints":["x"],"steps":[]}
Done.`

	result := mustExtractPlanJSON(t, response)
	if result != `{"findings":{},"evidence":["README.md"],"constraints":["x"],"steps":[]}` {
		t.Fatalf("ExtractPlanJSON() = %q, want malformed legacy handoff-field retry candidate", result)
	}
	assertPlanJSONNeedsRetry(t, result)
}
