package llmcatalog

import (
	"reflect"
	"testing"
)

func TestProviderDescriptorRequiredCredentialEnvVars(t *testing.T) {
	tests := []struct {
		name string
		desc ProviderDescriptor
		want []string
	}{
		{
			name: "explicit credential env vars win",
			desc: ProviderDescriptor{
				CredentialKind:    "api_key",
				APIKeyEnv:         "SINGLE_KEY",
				CredentialEnvVars: []string{"SERVICE_KEY", "SERVICE_ENDPOINT"},
			},
			want: []string{"SERVICE_KEY", "SERVICE_ENDPOINT"},
		},
		{
			name: "api key env is derived for existing providers",
			desc: ProviderDescriptor{
				CredentialKind: "api_key",
				APIKeyEnv:      "OPENAI_API_KEY",
			},
			want: []string{"OPENAI_API_KEY"},
		},
		{
			name: "base url providers do not require env when default exists",
			desc: ProviderDescriptor{
				CredentialKind: "base_url",
				BaseURLEnv:     "OLLAMA_BASE_URL",
				DefaultBaseURL: "http://localhost:11434",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.desc.RequiredCredentialEnvVars(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("RequiredCredentialEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderDescriptorFor_ClonesCredentialEnvVars(t *testing.T) {
	got := ProviderCredentialEnvVars("openai")
	if !reflect.DeepEqual(got, []string{"OPENAI_API_KEY"}) {
		t.Fatalf("ProviderCredentialEnvVars(openai) = %v, want [OPENAI_API_KEY]", got)
	}

	got[0] = "MUTATED"
	again := ProviderCredentialEnvVars("openai")
	if !reflect.DeepEqual(again, []string{"OPENAI_API_KEY"}) {
		t.Fatalf("ProviderCredentialEnvVars(openai) after mutation = %v, want [OPENAI_API_KEY]", again)
	}

	azure := ProviderCredentialEnvVars("azure")
	if !reflect.DeepEqual(azure, []string{"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_AUTH_TOKEN", "AZURE_OPENAI_AUTH_TOKEN_COMMAND", "AZURE_OPENAI_BASE_URL"}) {
		t.Fatalf("ProviderCredentialEnvVars(azure) = %v, want Azure API key, auth token, auth token command, and base URL", azure)
	}

	azureSets := ProviderCredentialEnvVarSets("azure")
	wantSets := [][]string{
		{"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_BASE_URL"},
		{"AZURE_OPENAI_AUTH_TOKEN", "AZURE_OPENAI_BASE_URL"},
		{"AZURE_OPENAI_AUTH_TOKEN_COMMAND", "AZURE_OPENAI_BASE_URL"},
	}
	if !reflect.DeepEqual(azureSets, wantSets) {
		t.Fatalf("ProviderCredentialEnvVarSets(azure) = %v, want %v", azureSets, wantSets)
	}
	azureSets[0][0] = "MUTATED"
	againSets := ProviderCredentialEnvVarSets("azure")
	if !reflect.DeepEqual(againSets, wantSets) {
		t.Fatalf("ProviderCredentialEnvVarSets(azure) after mutation = %v, want %v", againSets, wantSets)
	}
}

func TestCanonicalProviderKey_ResolvesDisplayName(t *testing.T) {
	if got := CanonicalProviderKey("Azure OpenAI"); got != "azure" {
		t.Fatalf("CanonicalProviderKey(%q) = %q, want azure", "Azure OpenAI", got)
	}
}

func TestProviderConfigKey_CanonicalizesDisplayNameButPreservesAlias(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{name: "azure display name", provider: "Azure OpenAI", want: "azure"},
		{name: "azure display name normalized", provider: " azure openai ", want: "azure"},
		{name: "anthropic alias is an owner key", provider: "anthropic", want: "anthropic"},
		{name: "moonshot alias is an owner key", provider: "moonshot", want: "moonshot"},
		{name: "canonical key", provider: "claude", want: "claude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProviderConfigKey(tt.provider); got != tt.want {
				t.Fatalf("ProviderConfigKey(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestProviderModelLookupKeys_CanonicalizesDisplayNameButPreservesAlias(t *testing.T) {
	azureKeys := ProviderModelLookupKeys("Azure OpenAI")
	if !reflect.DeepEqual(azureKeys, []string{"azure"}) {
		t.Fatalf("ProviderModelLookupKeys(Azure OpenAI) = %v, want [azure]", azureKeys)
	}

	anthropicKeys := ProviderModelLookupKeys("anthropic")
	if !reflect.DeepEqual(anthropicKeys, []string{"anthropic", "claude"}) {
		t.Fatalf("ProviderModelLookupKeys(anthropic) = %v, want [anthropic claude]", anthropicKeys)
	}

	moonshotKeys := ProviderModelLookupKeys("moonshot")
	if !reflect.DeepEqual(moonshotKeys, []string{"moonshot", "kimi"}) {
		t.Fatalf("ProviderModelLookupKeys(moonshot) = %v, want [moonshot kimi]", moonshotKeys)
	}
}

func TestProviderSupportsResponsesAPI(t *testing.T) {
	if !ProviderSupportsResponsesAPI("openai") {
		t.Fatal("ProviderSupportsResponsesAPI(openai) = false, want true")
	}
	if !ProviderSupportsResponsesAPI("azure") {
		t.Fatal("ProviderSupportsResponsesAPI(azure) = false, want true")
	}
	if ProviderSupportsResponsesAPI("openrouter") {
		t.Fatal("ProviderSupportsResponsesAPI(openrouter) = true, want false")
	}
	if ProviderSupportsResponsesAPI("kimi") {
		t.Fatal("ProviderSupportsResponsesAPI(kimi) = true, want false")
	}
}

func TestProviderDescriptorFor_Kimi(t *testing.T) {
	desc, ok := ProviderDescriptorFor("moonshot")
	if !ok {
		t.Fatal("ProviderDescriptorFor(moonshot) ok = false, want true")
	}
	if desc.Key != "kimi" {
		t.Fatalf("Key = %q, want kimi", desc.Key)
	}
	if desc.DisplayName != "Kimi" {
		t.Fatalf("DisplayName = %q, want Kimi", desc.DisplayName)
	}
	if desc.APIKeyEnv != "MOONSHOT_API_KEY" {
		t.Fatalf("APIKeyEnv = %q, want MOONSHOT_API_KEY", desc.APIKeyEnv)
	}
	if desc.DefaultSubAgentModel != "kimi-k2.5" {
		t.Fatalf("DefaultSubAgentModel = %q, want kimi-k2.5", desc.DefaultSubAgentModel)
	}
	if !desc.SupportsImages {
		t.Fatal("SupportsImages = false, want true")
	}
	if desc.PricingFamily != "kimi" {
		t.Fatalf("PricingFamily = %q, want kimi", desc.PricingFamily)
	}
	if desc.ModelDefaults.DefaultModel != "kimi-k2.6" || desc.ModelDefaults.MaxOutputTokens != 32768 {
		t.Fatalf("ModelDefaults = %#v, want kimi-k2.6 / 32768", desc.ModelDefaults)
	}
}
