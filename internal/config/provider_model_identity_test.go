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

func TestProviderModelLookupKey_PrefersExactThenAlias(t *testing.T) {
	t.Run("exact key wins when both aliases exist", func(t *testing.T) {
		key, ok := providerModelLookupKey(map[string]ProviderModelConfig{
			"anthropic": {},
			"claude":    {},
		}, "anthropic")
		if !ok || key != "anthropic" {
			t.Fatalf("providerModelLookupKey(anthropic with both aliases) = (%q, %v), want (%q, true)", key, ok, "anthropic")
		}
	})

	t.Run("falls back to sibling alias when exact key is absent", func(t *testing.T) {
		key, ok := providerModelLookupKey(map[string]ProviderModelConfig{
			"claude": {},
		}, "anthropic")
		if !ok || key != "claude" {
			t.Fatalf("providerModelLookupKey(anthropic via claude fallback) = (%q, %v), want (%q, true)", key, ok, "claude")
		}
	})
}

func TestProviderModelWriteAndDeleteTargetKeys_RespectAliasOwnership(t *testing.T) {
	t.Run("write reuses existing sibling alias entry", func(t *testing.T) {
		key, ok := providerModelWriteTargetKey(map[string]ProviderModelConfig{
			"claude": {},
		}, "anthropic")
		if !ok || key != "claude" {
			t.Fatalf("providerModelWriteTargetKey(anthropic with existing claude) = (%q, %v), want (%q, true)", key, ok, "claude")
		}
	})

	t.Run("write creates exact requested alias when no entry exists", func(t *testing.T) {
		key, ok := providerModelWriteTargetKey(nil, "anthropic")
		if !ok || key != "anthropic" {
			t.Fatalf("providerModelWriteTargetKey(anthropic, nil) = (%q, %v), want (%q, true)", key, ok, "anthropic")
		}
	})

	t.Run("delete keeps exact requested key precedence", func(t *testing.T) {
		keys := providerModelDeleteTargetKeys(map[string]ProviderModelConfig{
			"claude":    {},
			"anthropic": {},
		}, "claude")
		if len(keys) != 1 || keys[0] != "claude" {
			t.Fatalf("providerModelDeleteTargetKeys(claude with both aliases) = %v, want [claude]", keys)
		}
	})

	t.Run("delete falls back to sibling alias when exact key is absent", func(t *testing.T) {
		keys := providerModelDeleteTargetKeys(map[string]ProviderModelConfig{
			"claude": {},
		}, "anthropic")
		if len(keys) != 1 || keys[0] != "claude" {
			t.Fatalf("providerModelDeleteTargetKeys(anthropic via claude fallback) = %v, want [claude]", keys)
		}
	})
}
