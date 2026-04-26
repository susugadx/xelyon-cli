package config

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func TestProviderCatalogUsesLLMCatalogProviderKeys(t *testing.T) {
	if got, want := ValidProviders, llmcatalog.ProviderKeys(true); !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidProviders = %v, want %v", got, want)
	}
	if got, want := GetDisplayProviders(), llmcatalog.DisplayProviderKeys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("GetDisplayProviders() = %v, want %v", got, want)
	}
}

func TestDefaultProviderModelsUseLLMCatalogDefaults(t *testing.T) {
	cfg := DefaultConfig()
	defaults := llmcatalog.DefaultProviderModelDescriptors()

	if len(cfg.ProviderModels) != len(defaults) {
		t.Fatalf("ProviderModels len = %d, want %d", len(cfg.ProviderModels), len(defaults))
	}
	for key, want := range defaults {
		got, ok := cfg.ProviderModels[key]
		if !ok {
			t.Fatalf("ProviderModels missing key %q", key)
		}
		if got.DefaultModel != want.DefaultModel ||
			got.MaxOutputTokens != want.MaxOutputTokens ||
			got.AnthropicVersion != want.AnthropicVersion ||
			!reflect.DeepEqual(got.AnthropicBeta, want.AnthropicBeta) {
			t.Fatalf("ProviderModels[%q] = %#v, want catalog default %#v", key, got, want)
		}
	}
}

func TestProviderCredentialEnvVarsAndAvailability(t *testing.T) {
	if got := ProviderCredentialEnvVars("openai"); !reflect.DeepEqual(got, []string{"OPENAI_API_KEY"}) {
		t.Fatalf("ProviderCredentialEnvVars(openai) = %v, want [OPENAI_API_KEY]", got)
	}
	if got := ProviderCredentialEnvVars("ollama"); len(got) != 0 {
		t.Fatalf("ProviderCredentialEnvVars(ollama) = %v, want empty", got)
	}

	t.Setenv("OPENAI_API_KEY", "")
	if ProviderHasAvailableCredential("openai") {
		t.Fatal("ProviderHasAvailableCredential(openai) = true, want false without key")
	}
	t.Setenv("OPENAI_API_KEY", "sk-test")
	if !ProviderHasAvailableCredential("openai") {
		t.Fatal("ProviderHasAvailableCredential(openai) = false, want true with key")
	}
	if !ProviderHasAvailableCredential("ollama") {
		t.Fatal("ProviderHasAvailableCredential(ollama) = false, want true with default base URL")
	}
}

func TestProviderSupportsResponsesAPI(t *testing.T) {
	if !ProviderSupportsResponsesAPI("openai") {
		t.Fatal("ProviderSupportsResponsesAPI(openai) = false, want true")
	}
	if ProviderSupportsResponsesAPI("groq") {
		t.Fatal("ProviderSupportsResponsesAPI(groq) = true, want false")
	}
}
