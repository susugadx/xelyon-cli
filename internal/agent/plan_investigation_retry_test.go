package agent

import "testing"

func TestPlanInvestigationRunner_HandleNoToolResponse_MalformedPlanWrapperRequestsRetry(t *testing.T) {
	agent, runner := newPlanInvestigationNoToolTest(t)

	p, action, err := runner.handleNoToolResponse(`{"plan": invalid}`)
	if err != nil {
		t.Fatalf("handleNoToolResponse() error = %v", err)
	}
	if p != nil {
		t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
	}
	if action != investigationLoopContinue {
		t.Fatalf("handleNoToolResponse() action = %v, want continue", action)
	}
	assertPlanJSONRetryPromptAppended(t, agent)
}

func TestPlanInvestigationRunner_HandleNoToolResponse_MalformedLegacyStepsRequestsRetry(t *testing.T) {
	agent, runner := newPlanInvestigationNoToolTest(t)

	p, action, err := runner.handleNoToolResponse(`{"steps": invalid}`)
	if err != nil {
		t.Fatalf("handleNoToolResponse() error = %v", err)
	}
	if p != nil {
		t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
	}
	if action != investigationLoopContinue {
		t.Fatalf("handleNoToolResponse() action = %v, want continue", action)
	}
	assertPlanJSONRetryPromptAppended(t, agent)
}

func TestPlanInvestigationRunner_HandleNoToolResponse_SchemaInvalidPlanJSONRequestsRetry(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "wrapper steps object",
			response: `{"plan":{"summary":"Fix","steps":{"id":1,"description":"Do it"}}}`,
		},
		{
			name:     "legacy steps object",
			response: `{"summary":"Fix","steps":{"id":1,"description":"Do it"}}`,
		},
		{
			name:     "empty wrapper summary",
			response: `{"plan":{"summary":""}}`,
		},
		{
			name:     "malformed handoff-only fields",
			response: `{"plan":{"findings":{},"evidence":["README.md"],"constraints":["x"],"steps":[]}}`,
		},
		{
			name:     "empty legacy summary",
			response: `{"summary":"","steps":[]}`,
		},
		{
			name:     "legacy malformed handoff-only fields",
			response: `{"findings":{},"evidence":["README.md"],"constraints":["x"],"steps":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, runner := newPlanInvestigationNoToolTest(t)

			p, action, err := runner.handleNoToolResponse(tt.response)
			if err != nil {
				t.Fatalf("handleNoToolResponse() error = %v", err)
			}
			if p != nil {
				t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
			}
			if action != investigationLoopContinue {
				t.Fatalf("handleNoToolResponse() action = %v, want continue", action)
			}
			assertPlanJSONRetryPromptAppended(t, agent)
		})
	}
}
