package agent

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHeadlessResult_ToJSON_IncludesDefaultExitPolicy(t *testing.T) {
	success := NewSuccessResult("openai", "gpt-5.4", "ok", nil, 10)
	jsonStr, err := success.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := parsed["failure_reason"]; ok {
		t.Fatalf("success JSON contains failure_reason: %v", parsed["failure_reason"])
	}
	if parsed["exit_policy"] != string(HeadlessExitPolicyLegacy) {
		t.Fatalf("exit_policy = %v, want %q", parsed["exit_policy"], HeadlessExitPolicyLegacy)
	}
	if parsed["recommended_exit_code"] != float64(0) {
		t.Fatalf("recommended_exit_code = %v, want 0", parsed["recommended_exit_code"])
	}
}

func TestHeadlessResult_FailureReasonForErrorTypes(t *testing.T) {
	tests := []struct {
		name    string
		errType string
		want    HeadlessFailureReason
	}{
		{name: "config", errType: HeadlessErrorTypeConfig, want: HeadlessFailureReasonConfigError},
		{name: "provider setup", errType: HeadlessErrorTypeProviderSetupRequired, want: HeadlessFailureReasonProviderSetupRequired},
		{name: "api", errType: HeadlessErrorTypeAPI, want: HeadlessFailureReasonAPIError},
		{name: "cancelled", errType: HeadlessErrorTypeCancelled, want: HeadlessFailureReasonCancelled},
		{name: "tool loop", errType: HeadlessErrorTypeToolLoopLimit, want: HeadlessFailureReasonToolLoopLimit},
		{name: "usage reason string", errType: string(HeadlessFailureReasonUsageError), want: HeadlessFailureReasonUsageError},
		{name: "tool reason string", errType: string(HeadlessFailureReasonToolError), want: HeadlessFailureReasonToolError},
		{name: "final check reason string", errType: string(HeadlessFailureReasonFinalCheckFailed), want: HeadlessFailureReasonFinalCheckFailed},
		{name: "read only reason string", errType: string(HeadlessFailureReasonReadOnlyViolation), want: HeadlessFailureReasonReadOnlyViolation},
		{name: "unsupported reason string", errType: string(HeadlessFailureReasonUnsupportedCapability), want: HeadlessFailureReasonUnsupportedCapability},
		{name: "unknown", errType: "other_error", want: HeadlessFailureReasonUnknownError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewErrorResult("openai", "gpt-5.4", tt.errType, "failed", 10)
			if result.FailureReason != tt.want {
				t.Fatalf("FailureReason = %q, want %q", result.FailureReason, tt.want)
			}
		})
	}
}

func TestRecommendedHeadlessExitCode(t *testing.T) {
	tests := []struct {
		name   string
		status string
		reason HeadlessFailureReason
		policy HeadlessExitPolicy
		want   int
	}{
		{name: "success legacy", status: HeadlessStatusSuccess, policy: HeadlessExitPolicyLegacy, want: 0},
		{name: "success ci", status: HeadlessStatusSuccess, policy: HeadlessExitPolicyCI, want: 0},
		{name: "legacy usage error", status: HeadlessStatusError, reason: HeadlessFailureReasonUsageError, policy: HeadlessExitPolicyLegacy, want: 1},
		{name: "ci usage error", status: HeadlessStatusError, reason: HeadlessFailureReasonUsageError, policy: HeadlessExitPolicyCI, want: 2},
		{name: "ci config error", status: HeadlessStatusError, reason: HeadlessFailureReasonConfigError, policy: HeadlessExitPolicyCI, want: 3},
		{name: "ci provider setup", status: HeadlessStatusError, reason: HeadlessFailureReasonProviderSetupRequired, policy: HeadlessExitPolicyCI, want: 3},
		{name: "ci tool error", status: HeadlessStatusError, reason: HeadlessFailureReasonToolError, policy: HeadlessExitPolicyCI, want: 4},
		{name: "ci final check", status: HeadlessStatusError, reason: HeadlessFailureReasonFinalCheckFailed, policy: HeadlessExitPolicyCI, want: 5},
		{name: "ci api error", status: HeadlessStatusError, reason: HeadlessFailureReasonAPIError, policy: HeadlessExitPolicyCI, want: 6},
		{name: "ci cancelled", status: HeadlessStatusError, reason: HeadlessFailureReasonCancelled, policy: HeadlessExitPolicyCI, want: 7},
		{name: "ci read only violation", status: HeadlessStatusError, reason: HeadlessFailureReasonReadOnlyViolation, policy: HeadlessExitPolicyCI, want: 8},
		{name: "ci unsupported capability", status: HeadlessStatusError, reason: HeadlessFailureReasonUnsupportedCapability, policy: HeadlessExitPolicyCI, want: 9},
		{name: "ci tool loop", status: HeadlessStatusError, reason: HeadlessFailureReasonToolLoopLimit, policy: HeadlessExitPolicyCI, want: 1},
		{name: "ci unknown", status: HeadlessStatusError, reason: HeadlessFailureReasonUnknownError, policy: HeadlessExitPolicyCI, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RecommendedHeadlessExitCode(tt.status, tt.reason, tt.policy)
			if got != tt.want {
				t.Fatalf("RecommendedHeadlessExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunHeadlessWithConfig_DefaultExitPolicyIsLegacy(t *testing.T) {
	provider := &headlessToolSetProbeProvider{name: "openai"}
	result := RunHeadlessWithConfig(context.Background(), "probe", "gpt-5.4", provider, newProjectMapDisabledConfig())
	if result.ExitPolicy != HeadlessExitPolicyLegacy {
		t.Fatalf("ExitPolicy = %q, want %q", result.ExitPolicy, HeadlessExitPolicyLegacy)
	}
	if result.RecommendedExitCode != 0 {
		t.Fatalf("RecommendedExitCode = %d, want 0", result.RecommendedExitCode)
	}
	if result.FailureReason != "" {
		t.Fatalf("FailureReason = %q, want empty success reason", result.FailureReason)
	}
}
