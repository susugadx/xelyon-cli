package agent

import "testing"

func TestResolveEditToolMode(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "")

	tests := []struct {
		name     string
		provider string
		model    string
		want     string
	}{
		{name: "apply_patch provider", provider: "openai", model: "gpt-5.4", want: EditToolModeApplyPatch},
		{name: "legacy provider", provider: "kimi", model: "kimi-k2.6", want: EditToolModeLegacy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveEditToolMode(tt.provider, tt.model); got != tt.want {
				t.Fatalf("ResolveEditToolMode(%q, %q) = %q, want %q", tt.provider, tt.model, got, tt.want)
			}
		})
	}
}
