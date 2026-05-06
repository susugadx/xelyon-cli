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
	if got := ProviderCredentialEnvVars("azure"); !reflect.DeepEqual(got, []string{"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_AUTH_TOKEN", "AZURE_OPENAI_AUTH_TOKEN_COMMAND", "AZURE_OPENAI_BASE_URL"}) {
		t.Fatalf("ProviderCredentialEnvVars(azure) = %v, want Azure API key, auth token, auth token command, and base URL", got)
	}
	if got := ProviderCredentialEnvVars("ollama"); len(got) != 0 {
		t.Fatalf("ProviderCredentialEnvVars(ollama) = %v, want empty", got)
	}
	if got := ProviderAPIKeyEnv("kimi"); got != "MOONSHOT_API_KEY" {
		t.Fatalf("ProviderAPIKeyEnv(kimi) = %q, want MOONSHOT_API_KEY", got)
	}
	if got := ProviderAPIKeyEnv("moonshot"); got != "MOONSHOT_API_KEY" {
		t.Fatalf("ProviderAPIKeyEnv(moonshot) = %q, want MOONSHOT_API_KEY", got)
	}

	t.Setenv("OPENAI_API_KEY", "")
	if ProviderHasAvailableCredential("openai") {
		t.Fatal("ProviderHasAvailableCredential(openai) = true, want false without key")
	}
	t.Setenv("OPENAI_API_KEY", "sk-test")
	if !ProviderHasAvailableCredential("openai") {
		t.Fatal("ProviderHasAvailableCredential(openai) = false, want true with key")
	}
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_BASE_URL", "")
	if ProviderHasAvailableCredential("azure") {
		t.Fatal("ProviderHasAvailableCredential(azure) = true, want false without base URL")
	}
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	if !ProviderHasAvailableCredential("azure") {
		t.Fatal("ProviderHasAvailableCredential(azure) = false, want true with key and base URL")
	}
	t.Setenv("AZURE_OPENAI_API_KEY", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "entra-token")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")
	if !ProviderHasAvailableCredential("azure") {
		t.Fatal("ProviderHasAvailableCredential(azure) = false, want true with Entra token and base URL")
	}
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "printf token")
	if !ProviderHasAvailableCredential("azure") {
		t.Fatal("ProviderHasAvailableCredential(azure) = false, want true with Entra token command and base URL")
	}
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")
	if ProviderHasAvailableCredential("azure") {
		t.Fatal("ProviderHasAvailableCredential(azure) = true, want false without API key, Entra token, or Entra token command")
	}
	if !ProviderHasAvailableCredential("ollama") {
		t.Fatal("ProviderHasAvailableCredential(ollama) = false, want true with default base URL")
	}
}

func TestProviderSupportsResponsesAPI(t *testing.T) {
	if !ProviderSupportsResponsesAPI("openai") {
		t.Fatal("ProviderSupportsResponsesAPI(openai) = false, want true")
	}
	if !ProviderSupportsResponsesAPI("azure") {
		t.Fatal("ProviderSupportsResponsesAPI(azure) = false, want true")
	}
	if ProviderSupportsResponsesAPI("groq") {
		t.Fatal("ProviderSupportsResponsesAPI(groq) = true, want false")
	}
	if ProviderSupportsResponsesAPI("kimi") {
		t.Fatal("ProviderSupportsResponsesAPI(kimi) = true, want false")
	}
}

func TestAzureProviderCatalogDoesNotHardcodeHelperDeployments(t *testing.T) {
	if got := ProviderDefaultSubAgentModel("azure"); got != "" {
		t.Fatalf("ProviderDefaultSubAgentModel(azure) = %q, want empty so configured deployment is used", got)
	}
}

func TestKimiProviderCatalog(t *testing.T) {
	if got := ProviderDefaultSubAgentModel("kimi"); got != "kimi-k2.5" {
		t.Fatalf("ProviderDefaultSubAgentModel(kimi) = %q, want kimi-k2.5", got)
	}
	if got := ProviderDefaultSubAgentModel("moonshot"); got != "kimi-k2.5" {
		t.Fatalf("ProviderDefaultSubAgentModel(moonshot) = %q, want kimi-k2.5", got)
	}
	if ProviderSupportsImages("kimi") {
		t.Fatal("ProviderSupportsImages(kimi) = true, want false")
	}
	cfg := DefaultConfig()
	pm, ok := cfg.ProviderModels["kimi"]
	if !ok {
		t.Fatal("DefaultConfig().ProviderModels missing kimi")
	}
	if pm.DefaultModel != "kimi-k2.6" || pm.MaxOutputTokens != 32768 {
		t.Fatalf("ProviderModels[kimi] = %#v, want kimi-k2.6 / 32768", pm)
	}
}
