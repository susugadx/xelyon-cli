package agent

import "testing"

func TestResolveEditToolMode(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		model      string
		wantLegacy bool
	}{
		{name: "openai uses apply_patch", provider: "openai", model: "gpt-5.4"},
		{name: "gemini uses apply_patch", provider: "gemini", model: "gemini-3.1-pro"},
		{name: "claude uses legacy", provider: "claude", model: "claude-sonnet-4-6", wantLegacy: true},
		{name: "anthropic alias uses legacy", provider: "anthropic", model: "claude-sonnet-4-6", wantLegacy: true},
		{name: "deepseek uses legacy", provider: "deepseek", model: "deepseek-chat", wantLegacy: true},
		{name: "openrouter anthropic uses legacy", provider: "openrouter", model: "anthropic/claude-sonnet-4.6", wantLegacy: true},
		{name: "openrouter openai uses apply_patch", provider: "openrouter", model: "openai/gpt-5.4"},
		{name: "openrouter google uses apply_patch", provider: "openrouter", model: "google/gemini-2.5-pro"},
		{name: "bedrock claude family uses legacy", provider: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1", wantLegacy: true},
		{name: "bedrock non claude defaults apply_patch", provider: "bedrock", model: "amazon.nova-pro-v1:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveEditToolMode(tt.provider, tt.model)
			if tt.wantLegacy && got != EditToolModeLegacy {
				t.Fatalf("ResolveEditToolMode(%q, %q) = %q, want %q", tt.provider, tt.model, got, EditToolModeLegacy)
			}
			if !tt.wantLegacy && got != EditToolModeApplyPatch {
				t.Fatalf("ResolveEditToolMode(%q, %q) = %q, want %q", tt.provider, tt.model, got, EditToolModeApplyPatch)
			}
		})
	}
}

func TestResolveEditToolMode_EnvOverride(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "legacy")

	if got := ResolveEditToolMode("openai", "gpt-5.4"); got != EditToolModeLegacy {
		t.Fatalf("ResolveEditToolMode() with env override = %q, want %q", got, EditToolModeLegacy)
	}
}

func TestPlanModeExcludedTools_DefaultEditTool(t *testing.T) {
	excluded := planModeExcludedTools(EditToolModeApplyPatch)

	if containsString(excluded, "ask_user_question") {
		t.Fatal("plan mode should not exclude planning tools")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if !containsString(excluded, name) {
			t.Fatalf("plan mode should exclude %s in default edit mode", name)
		}
	}
	if containsString(excluded, "apply_patch") {
		t.Fatal("plan mode should keep apply_patch visible in default edit mode")
	}
}

func TestPlanModeExcludedTools_LegacyEditTool(t *testing.T) {
	excluded := planModeExcludedTools(EditToolModeLegacy)
	if containsString(excluded, "ask_user_question") {
		t.Fatal("plan mode should not exclude planning tools")
	}
	if !containsString(excluded, "apply_patch") {
		t.Fatal("plan mode should exclude apply_patch in legacy edit mode")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if containsString(excluded, name) {
			t.Fatalf("plan mode should keep %s visible in legacy edit mode", name)
		}
	}
}

func TestNormalModeExcludedTools_IncludesListDir(t *testing.T) {
	excluded := normalModeExcludedTools(EditToolModeApplyPatch)
	if !containsString(excluded, "list_dir") {
		t.Fatal("normal mode should exclude list_dir")
	}
	if containsString(excluded, "inspect_symbol") {
		t.Fatal("inspect_symbol is no longer a public tool, should not appear in excluded list")
	}
}

func TestPlanModeExcludedTools_IncludesListDir(t *testing.T) {
	excluded := planModeExcludedTools(EditToolModeApplyPatch)
	if !containsString(excluded, "list_dir") {
		t.Fatal("plan mode should exclude list_dir")
	}
	if containsString(excluded, "inspect_symbol") {
		t.Fatal("inspect_symbol is no longer a public tool, should not appear in excluded list")
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
