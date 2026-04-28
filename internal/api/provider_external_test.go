package api_test

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/all"
)

func TestNewProvider_MissingAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		envKey       string
	}{
		{
			name:         "DeepSeek without API key",
			providerName: "deepseek",
			envKey:       "DEEPSEEK_API_KEY",
		},
		{
			name:         "OpenAI without API key",
			providerName: "openai",
			envKey:       "OPENAI_API_KEY",
		},
		{
			name:         "Azure OpenAI without API key",
			providerName: "azure",
			envKey:       "AZURE_OPENAI_API_KEY",
		},
		{
			name:         "Gemini without API key",
			providerName: "gemini",
			envKey:       "GEMINI_API_KEY",
		},
		{
			name:         "Claude without API key",
			providerName: "claude",
			envKey:       "ANTHROPIC_API_KEY",
		},
		{
			name:         "Groq without API key",
			providerName: "groq",
			envKey:       "GROQ_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をクリア
			originalValue := os.Getenv(tt.envKey)
			os.Unsetenv(tt.envKey)
			originalAzureAuthToken := os.Getenv("AZURE_OPENAI_AUTH_TOKEN")
			if tt.providerName == "azure" {
				os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
			}
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.envKey, originalValue)
				}
				if tt.providerName == "azure" {
					if originalAzureAuthToken != "" {
						os.Setenv("AZURE_OPENAI_AUTH_TOKEN", originalAzureAuthToken)
					} else {
						os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
					}
				}
			}()

			_, err := api.NewProvider(tt.providerName)
			if err == nil {
				t.Errorf("NewProvider(%q) should return error when %s is not set", tt.providerName, tt.envKey)
			}
		})
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	_, err := api.NewProvider("unknown-provider")
	if err == nil {
		t.Error("NewProvider should return error for unknown provider")
	}

	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("Expected 'unknown provider' in error, got: %v", err)
	}
}

func TestNewProvider_AzureRequiresBaseURL(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_BASE_URL", "")

	_, err := api.NewProvider("azure")
	if err == nil {
		t.Fatal("NewProvider(\"azure\") error = nil, want missing base URL")
	}
	if !strings.Contains(err.Error(), "AZURE_OPENAI_BASE_URL not set") {
		t.Fatalf("NewProvider(\"azure\") error = %v, want AZURE_OPENAI_BASE_URL not set", err)
	}
}

func TestNewProvider_AzureWithEntraToken(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "test-entra-token")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")

	provider, err := api.NewProvider("azure")
	if err != nil {
		t.Fatalf("NewProvider(\"azure\") with Entra token failed: %v", err)
	}
	if provider.Name() != "Azure OpenAI" {
		t.Fatalf("NewProvider(\"azure\") Name() = %q, want Azure OpenAI", provider.Name())
	}
}

func TestNewProvider_OllamaWithDefaultURL(t *testing.T) {
	// OLLAMA_BASE_URLをクリア
	originalValue := os.Getenv("OLLAMA_BASE_URL")
	os.Unsetenv("OLLAMA_BASE_URL")
	defer func() {
		if originalValue != "" {
			os.Setenv("OLLAMA_BASE_URL", originalValue)
		}
	}()

	provider, err := api.NewProvider("ollama")
	if err != nil {
		t.Fatalf("NewProvider('ollama') failed: %v", err)
	}

	if provider.Name() != "Ollama" {
		t.Errorf("NewProvider('ollama') Name() = %v, want 'Ollama'", provider.Name())
	}
}

func TestNewProvider_OllamaWithCustomURL(t *testing.T) {
	originalValue := os.Getenv("OLLAMA_BASE_URL")
	os.Setenv("OLLAMA_BASE_URL", "http://custom-host:8080")
	defer func() {
		if originalValue != "" {
			os.Setenv("OLLAMA_BASE_URL", originalValue)
		} else {
			os.Unsetenv("OLLAMA_BASE_URL")
		}
	}()

	provider, err := api.NewProvider("ollama")
	if err != nil {
		t.Fatalf("NewProvider('ollama') with custom URL failed: %v", err)
	}

	if provider.Name() != "Ollama" {
		t.Errorf("NewProvider('ollama') Name() = %v, want 'Ollama'", provider.Name())
	}
}

func TestNewProvider_SuccessPaths(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		envKey       string
		envValue     string
		wantName     string
	}{
		{
			name:         "DeepSeek with API key",
			providerName: "deepseek",
			envKey:       "DEEPSEEK_API_KEY",
			envValue:     "test-deepseek-key",
			wantName:     "DeepSeek",
		},
		{
			name:         "OpenAI with API key",
			providerName: "openai",
			envKey:       "OPENAI_API_KEY",
			envValue:     "test-openai-key",
			wantName:     "OpenAI",
		},
		{
			name:         "Azure OpenAI with API key and base URL",
			providerName: "azure",
			envKey:       "AZURE_OPENAI_API_KEY",
			envValue:     "test-azure-key",
			wantName:     "Azure OpenAI",
		},
		{
			name:         "Azure OpenAI display name alias",
			providerName: "Azure OpenAI",
			envKey:       "AZURE_OPENAI_API_KEY",
			envValue:     "test-azure-key",
			wantName:     "Azure OpenAI",
		},
		{
			name:         "Azure key with spaces",
			providerName: " azure ",
			envKey:       "AZURE_OPENAI_API_KEY",
			envValue:     "test-azure-key",
			wantName:     "Azure OpenAI",
		},
		{
			name:         "Gemini with API key",
			providerName: "gemini",
			envKey:       "GEMINI_API_KEY",
			envValue:     "test-gemini-key",
			wantName:     "Gemini",
		},
		{
			name:         "Claude with API key",
			providerName: "claude",
			envKey:       "ANTHROPIC_API_KEY",
			envValue:     "test-claude-key",
			wantName:     "Claude",
		},
		{
			name:         "Anthropic alias",
			providerName: "anthropic",
			envKey:       "ANTHROPIC_API_KEY",
			envValue:     "test-anthropic-key",
			wantName:     "Claude",
		},
		{
			name:         "Groq with API key",
			providerName: "groq",
			envKey:       "GROQ_API_KEY",
			envValue:     "test-groq-key",
			wantName:     "Groq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalValue := os.Getenv(tt.envKey)
			os.Setenv(tt.envKey, tt.envValue)
			originalAzureBaseURL := os.Getenv("AZURE_OPENAI_BASE_URL")
			normalizedProvider := strings.ToLower(strings.TrimSpace(tt.providerName))
			if normalizedProvider == "azure" || normalizedProvider == "azure openai" {
				os.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
			}
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.envKey, originalValue)
				} else {
					os.Unsetenv(tt.envKey)
				}
				if normalizedProvider == "azure" || normalizedProvider == "azure openai" {
					if originalAzureBaseURL != "" {
						os.Setenv("AZURE_OPENAI_BASE_URL", originalAzureBaseURL)
					} else {
						os.Unsetenv("AZURE_OPENAI_BASE_URL")
					}
				}
			}()

			provider, err := api.NewProvider(tt.providerName)
			if err != nil {
				t.Fatalf("NewProvider(%q) failed: %v", tt.providerName, err)
			}

			if provider.Name() != tt.wantName {
				t.Errorf("NewProvider(%q) Name() = %v, want %v", tt.providerName, provider.Name(), tt.wantName)
			}
		})
	}
}

// --- IsRegisteredProvider / ListProviders テスト ---

func TestIsRegisteredProvider(t *testing.T) {
	// 全 LLM プロバイダーが登録されていること
	registered := []string{"deepseek", "claude", "anthropic", "openai", "azure", "gemini", "groq", "ollama", "openrouter", "bedrock"}
	for _, name := range registered {
		if !api.IsRegisteredProvider(name) {
			t.Errorf("IsRegisteredProvider(%q) = false, want true", name)
		}
	}

	// 大文字小文字を無視すること
	if !api.IsRegisteredProvider("Claude") {
		t.Error("IsRegisteredProvider should be case-insensitive")
	}
	if !api.IsRegisteredProvider("OPENAI") {
		t.Error("IsRegisteredProvider should be case-insensitive")
	}
	if !api.IsRegisteredProvider(" azure ") {
		t.Error("IsRegisteredProvider should trim surrounding spaces")
	}
	if !api.IsRegisteredProvider("Azure OpenAI") {
		t.Error("IsRegisteredProvider should resolve provider display-name aliases")
	}

	// 未登録の名前
	if api.IsRegisteredProvider("nonexistent") {
		t.Error("IsRegisteredProvider('nonexistent') = true, want false")
	}
	if api.IsRegisteredProvider("") {
		t.Error("IsRegisteredProvider('') = true, want false")
	}
}

func TestListProviders(t *testing.T) {
	providers := api.ListProviders()

	// 全 LLM プロバイダーが含まれること
	required := []string{"azure", "bedrock", "claude", "deepseek", "gemini", "groq", "ollama", "openai", "openrouter"}
	for _, name := range required {
		found := false
		for _, p := range providers {
			if p == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListProviders() missing %q, got %v", name, providers)
		}
	}

	// anthropic エイリアスが除外されていること
	for _, p := range providers {
		if p == "anthropic" {
			t.Error("ListProviders() should exclude 'anthropic' alias")
		}
	}

	// ソート済みであること
	if !sort.StringsAreSorted(providers) {
		t.Errorf("ListProviders() should be sorted, got %v", providers)
	}
}
