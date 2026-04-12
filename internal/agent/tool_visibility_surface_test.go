package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/investigation"
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
	if policy.investigationSurface != investigation.SurfaceEditExactControl {
		t.Fatalf("default edit mode should use edit exact-control surface, got %q", policy.investigationSurface)
	}
	if policy.investigationSurface.AllowsLowLevelOverrides() {
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
	if policy.investigationSurface != investigation.SurfaceLegacyOverrides {
		t.Fatalf("legacy edit mode should use legacy override surface, got %q", policy.investigationSurface)
	}
	if !policy.investigationSurface.AllowsLowLevelOverrides() {
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
			policy := newToolVisibilityPolicy(EditToolModeApplyPatch, tt.phase, toolVisibilityOptions{allowSubAgents: true})
			if policy.investigationSurface.ReadFileRole() != investigation.ToolRoleEditExactControl {
				t.Fatalf("%s mode should keep read_file as exact-control override, got %q", tt.phase, policy.investigationSurface.ReadFileRole())
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
	policy := newToolVisibilityPolicy(EditToolModeLegacy, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})
	if policy.investigationSurface.SearchCodeRole() != investigation.ToolRoleLowLevelOverride || policy.investigationSurface.ReadFileRole() != investigation.ToolRoleLowLevelOverride {
		t.Fatalf("legacy mode should keep search_code/read_file as low-level overrides, got search=%q read=%q", policy.investigationSurface.SearchCodeRole(), policy.investigationSurface.ReadFileRole())
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
