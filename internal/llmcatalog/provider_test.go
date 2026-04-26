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
}

func TestProviderSupportsResponsesAPI(t *testing.T) {
	if !ProviderSupportsResponsesAPI("openai") {
		t.Fatal("ProviderSupportsResponsesAPI(openai) = false, want true")
	}
	if ProviderSupportsResponsesAPI("openrouter") {
		t.Fatal("ProviderSupportsResponsesAPI(openrouter) = true, want false")
	}
}
