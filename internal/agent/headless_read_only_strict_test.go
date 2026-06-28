package agent

import (
	"context"
	"testing"
)

func TestHeadlessReadOnlyStrictPromotesReadOnlyViolation(t *testing.T) {
	provider := &headlessToolErrorUsageProvider{
		responses: []string{
			headlessToolCallJSON(t, "write_file", map[string]string{"path": "target.txt", "content": "changed\n"}),
			"final response after denied write",
		},
	}
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	result := RunHeadlessWithConfigOptions(context.Background(), "attempt write", "gpt-5.4-nano", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeReadOnlyViolation {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeReadOnlyViolation)
	}
	if result.FailureReason != HeadlessFailureReasonReadOnlyViolation {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonReadOnlyViolation)
	}
	if result.Response != "final response after denied write" {
		t.Fatalf("Response = %q, want preserved final response", result.Response)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Success {
		t.Fatalf("ToolCalls = %+v, want one failed denied tool call", result.ToolCalls)
	}
	if result.Tokens == nil {
		t.Fatal("Tokens = nil, want preserved token usage")
	}
	if result.Tokens.Input != 100 || result.Tokens.Cached != 20 || result.Tokens.Output != 30 || result.Tokens.Thinking != 5 || result.Tokens.Total != 135 {
		t.Fatalf("Tokens = %+v, want input=100 cached=20 output=30 thinking=5 total=135", result.Tokens)
	}
	if result.Cost <= 0 {
		t.Fatalf("Cost = %f, want preserved positive cost", result.Cost)
	}
}
