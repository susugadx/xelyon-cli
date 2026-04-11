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
