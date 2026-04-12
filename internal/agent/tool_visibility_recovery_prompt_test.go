package agent

import (
	"strings"
	"testing"
)

func TestToolVisibilityPolicy_NormalModeRecoveryPrompt_DefaultSurface(t *testing.T) {
	policy := newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})

	for _, kind := range []normalModeRecoveryPromptKind{
		normalModeRecoveryPromptDirectExecution,
		normalModeRecoveryPromptStopPlanning,
		normalModeRecoveryPromptNoTextPlan,
	} {
		got := policy.normalModeRecoveryPrompt(kind)
		if !containsAll(got, "gather_context", "read_file", "apply_patch") {
			t.Fatalf("default recovery prompt should mention gather_context/read_file/apply_patch, got %q", got)
		}
		if containsAny(got, "str_replace") {
			t.Fatalf("default recovery prompt should avoid hidden legacy edit tools, got %q", got)
		}
	}
}

func TestToolVisibilityPolicy_NormalModeRecoveryPrompt_LegacySurface(t *testing.T) {
	policy := newToolVisibilityPolicy(EditToolModeLegacy, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})

	got := policy.normalModeRecoveryPrompt(normalModeRecoveryPromptDirectExecution)
	if !containsAll(got, "gather_context", "read_file", "str_replace") {
		t.Fatalf("legacy recovery prompt should mention visible low-level tools, got %q", got)
	}
	if containsAny(got, "apply_patch") {
		t.Fatalf("legacy recovery prompt should avoid apply_patch, got %q", got)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !containsString(s, sub) {
			return false
		}
	}
	return true
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if containsString(s, sub) {
			return true
		}
	}
	return false
}

func containsString(s, sub string) bool { return strings.Contains(s, sub) }
