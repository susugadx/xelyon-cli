package kimi

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestNewAndURLOverride(t *testing.T) {
	t.Setenv("KIMI_API_URL", "")
	if got := New("test-key").APIURL(); got != defaultKimiURL {
		t.Fatalf("APIURL() = %q, want %q", got, defaultKimiURL)
	}

	t.Setenv("KIMI_API_URL", "https://proxy.example/v1/chat/completions")
	if got := New("test-key").APIURL(); got != "https://proxy.example/v1/chat/completions" {
		t.Fatalf("APIURL() = %q, want custom URL", got)
	}
}

func TestProviderRegistrationAndMoonshotAlias(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "")
	if _, err := api.NewProvider("kimi"); err == nil || err.Error() != "MOONSHOT_API_KEY not set" {
		t.Fatalf("NewProvider(kimi) error = %v, want MOONSHOT_API_KEY not set", err)
	}

	t.Setenv("MOONSHOT_API_KEY", "test-key")
	for _, tt := range []struct {
		providerName  string
		wantConfigKey string
	}{
		{providerName: "kimi", wantConfigKey: "kimi"},
		{providerName: "moonshot", wantConfigKey: "moonshot"},
	} {
		providerName := tt.providerName
		t.Run(providerName, func(t *testing.T) {
			p, err := api.NewProvider(providerName)
			if err != nil {
				t.Fatalf("NewProvider(%q) error = %v", providerName, err)
			}
			if p.Name() != "Kimi" {
				t.Fatalf("Name() = %q, want Kimi", p.Name())
			}
			aware, ok := p.(interface{ ProviderConfigKey() string })
			if !ok {
				t.Fatalf("NewProvider(%q) does not expose ProviderConfigKey", providerName)
			}
			if got := aware.ProviderConfigKey(); got != tt.wantConfigKey {
				t.Fatalf("ProviderConfigKey() = %q, want %q", got, tt.wantConfigKey)
			}
		})
	}
}

func TestProviderCapabilities(t *testing.T) {
	p := New("test-key")
	if !p.SupportsImages() {
		t.Fatal("SupportsImages() = false, want true")
	}

	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	if p.IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = true, want false when KIMI_FUNCTION_CALLING=0")
	}
	t.Setenv("KIMI_FUNCTION_CALLING", "")
	if !p.IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = false, want true by default")
	}
}
