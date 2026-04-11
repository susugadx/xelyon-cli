package agent

import (
	"strings"
	"testing"
)

func TestToolVisibilityPolicy_Plan_DefaultEditTool(t *testing.T) {
	policy := newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true})
	excluded := policy.excluded()

	if toolNameInList(excluded, "ask_user_question") {
		t.Fatal("plan mode should not exclude planning tools")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if !toolNameInList(excluded, name) {
			t.Fatalf("plan mode should exclude %s in default edit mode", name)
		}
	}
	if toolNameInList(excluded, "apply_patch") {
		t.Fatal("plan mode should keep apply_patch visible in default edit mode")
	}
	if toolNameInList(excluded, "read_file") {
		t.Fatal("plan mode should keep read_file visible in default edit mode")
	}
	if policy.allowLowLevelInvestigation {
		t.Fatal("default edit mode should not expose low-level investigation overrides")
	}
}

func TestToolVisibilityPolicy_Plan_LegacyEditTool(t *testing.T) {
	policy := newToolVisibilityPolicy(EditToolModeLegacy, toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true})
	excluded := policy.excluded()
	if toolNameInList(excluded, "ask_user_question") {
		t.Fatal("plan mode should not exclude planning tools")
	}
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatal("plan mode should exclude apply_patch in legacy edit mode")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if toolNameInList(excluded, name) {
			t.Fatalf("plan mode should keep %s visible in legacy edit mode", name)
		}
	}
	for _, name := range []string{"search_code", "read_file"} {
		if toolNameInList(excluded, name) {
			t.Fatalf("plan mode should keep %s visible in legacy edit mode", name)
		}
	}
	if !toolNameInList(excluded, "list_dir") {
		t.Fatal("plan mode should keep list_dir hidden in legacy edit mode")
	}
	if !policy.allowLowLevelInvestigation {
		t.Fatal("legacy edit mode should expose low-level investigation overrides")
	}
}

func TestToolVisibilityPolicy_NormalAndPlan_GatherContextFirst(t *testing.T) {
	tests := []struct {
		name  string
		phase toolSurfacePhase
	}{
		{name: "normal", phase: toolSurfacePhaseNormal},
		{name: "plan", phase: toolSurfacePhasePlan},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := newToolVisibilityPolicy(EditToolModeApplyPatch, tt.phase, toolVisibilityOptions{allowSubAgents: true}).excluded()
			for _, name := range []string{"search_code", "list_dir"} {
				if !toolNameInList(excluded, name) {
					t.Fatalf("%s mode should exclude %s", tt.phase, name)
				}
			}
			if toolNameInList(excluded, "read_file") {
				t.Fatalf("%s mode should keep read_file visible in default edit mode", tt.phase)
			}
			if toolNameInList(excluded, "inspect_symbol") {
				t.Fatalf("%s mode should not mention removed inspect_symbol", tt.phase)
			}
			if toolNameInList(excluded, "gather_context") {
				t.Fatalf("%s mode should keep gather_context visible", tt.phase)
			}
		})
	}
}

func TestToolVisibilityPolicy_NormalLegacyKeepsLowLevelOverrides(t *testing.T) {
	excluded := newToolVisibilityPolicy(EditToolModeLegacy, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).excluded()
	for _, name := range []string{"search_code", "read_file"} {
		if toolNameInList(excluded, name) {
			t.Fatalf("normal mode should keep %s visible in legacy edit mode", name)
		}
	}
	if !toolNameInList(excluded, "list_dir") {
		t.Fatal("normal mode should keep list_dir hidden in legacy edit mode")
	}
}

func TestToolVisibilityPolicy_DisablesSubAgentToolsWhenRequested(t *testing.T) {
	excluded := newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: false}).excluded()
	for _, name := range []string{"spawn_agent", "wait_agent"} {
		if !toolNameInList(excluded, name) {
			t.Fatalf("sub-agent-disabled surface should exclude %s", name)
		}
	}
}

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
