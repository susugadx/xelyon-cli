package config

import "testing"

func TestActiveProviderConfigKey_PreservesExplicitAlias(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{name: "anthropic stays anthropic", provider: "anthropic", want: "anthropic"},
		{name: "claude stays claude", provider: "claude", want: "claude"},
		{name: "normalizes case and whitespace", provider: "  Claude ", want: "claude"},
		{name: "empty stays empty", provider: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActiveProviderConfigKey(tt.provider); got != tt.want {
				t.Fatalf("ActiveProviderConfigKey(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestFallbackProviderConfigKey_UsesDefaultProviderOnlyWhenProviderIsEmpty(t *testing.T) {
	tests := []struct {
		name            string
		provider        string
		defaultProvider string
		want            string
	}{
		{
			name:            "explicit anthropic alias wins over default provider spelling",
			provider:        "anthropic",
			defaultProvider: "claude",
			want:            "anthropic",
		},
		{
			name:            "explicit claude alias wins over anthropic default spelling",
			provider:        "claude",
			defaultProvider: "anthropic",
			want:            "claude",
		},
		{
			name:            "empty provider falls back to default provider",
			provider:        "",
			defaultProvider: "openai",
			want:            "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FallbackProviderConfigKey(tt.provider, tt.defaultProvider); got != tt.want {
				t.Fatalf("FallbackProviderConfigKey(%q, %q) = %q, want %q", tt.provider, tt.defaultProvider, got, tt.want)
			}
			if got := PreferredProviderConfigKey(tt.provider, tt.defaultProvider); got != tt.want {
				t.Fatalf("PreferredProviderConfigKey(%q, %q) = %q, want %q", tt.provider, tt.defaultProvider, got, tt.want)
			}
		})
	}
}

func TestDefaultModelSyncProviderKey_PrefersSessionOwnerUnlessDefaultProviderChangesRuntime(t *testing.T) {
	tests := []struct {
		name                            string
		currentSessionProviderConfigKey string
		currentDefaultProvider          string
		initialDefaultProvider          string
		want                            string
	}{
		{
			name:                            "edited default provider wins when runtime identity changes",
			currentSessionProviderConfigKey: "anthropic",
			currentDefaultProvider:          "openai",
			initialDefaultProvider:          "claude",
			want:                            "openai",
		},
		{
			name:                            "unchanged default provider keeps current anthropic session owner",
			currentSessionProviderConfigKey: "anthropic",
			currentDefaultProvider:          "claude",
			initialDefaultProvider:          "claude",
			want:                            "anthropic",
		},
		{
			name:                            "alias-only default provider change does not override session owner",
			currentSessionProviderConfigKey: "claude",
			currentDefaultProvider:          "anthropic",
			initialDefaultProvider:          "claude",
			want:                            "claude",
		},
		{
			name:                            "falls back to current default provider when session owner is unavailable",
			currentSessionProviderConfigKey: "",
			currentDefaultProvider:          "gemini",
			initialDefaultProvider:          "gemini",
			want:                            "gemini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultModelSyncProviderKey(tt.currentSessionProviderConfigKey, tt.currentDefaultProvider, tt.initialDefaultProvider); got != tt.want {
				t.Fatalf("DefaultModelSyncProviderKey(%q, %q, %q) = %q, want %q", tt.currentSessionProviderConfigKey, tt.currentDefaultProvider, tt.initialDefaultProvider, got, tt.want)
			}
		})
	}
}
