package cmd

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestCreateProvider_ValidProviders(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		envKey       string
		envValue     string
		wantErr      bool
	}{
		{
			name:         "deepseek with API key",
			providerName: "deepseek",
			envKey:       "DEEPSEEK_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "kimi with API key",
			providerName: "kimi",
			envKey:       "MOONSHOT_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "moonshot alias with API key",
			providerName: "moonshot",
			envKey:       "MOONSHOT_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "openai with API key",
			providerName: "openai",
			envKey:       "OPENAI_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "gemini with API key",
			providerName: "gemini",
			envKey:       "GEMINI_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "claude with API key",
			providerName: "claude",
			envKey:       "ANTHROPIC_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "anthropic alias with API key",
			providerName: "anthropic",
			envKey:       "ANTHROPIC_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "groq with API key",
			providerName: "groq",
			envKey:       "GROQ_API_KEY",
			envValue:     "test-key",
			wantErr:      false,
		},
		{
			name:         "ollama without API key (special case)",
			providerName: "ollama",
			envKey:       "", // Ollama doesn't need API key
			envValue:     "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			provider, err := createProvider(tt.providerName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("createProvider(%q) expected error, got nil", tt.providerName)
				}
			} else {
				if err != nil {
					t.Errorf("createProvider(%q) unexpected error: %v", tt.providerName, err)
				}
				if provider == nil {
					t.Errorf("createProvider(%q) returned nil provider", tt.providerName)
				}
			}
		})
	}
}

func TestCreateProvider_MissingAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		wantErrMsg   string
	}{
		{
			name:         "deepseek missing API key",
			providerName: "deepseek",
			wantErrMsg:   "DEEPSEEK_API_KEY not set",
		},
		{
			name:         "kimi missing API key",
			providerName: "kimi",
			wantErrMsg:   "MOONSHOT_API_KEY not set",
		},
		{
			name:         "moonshot alias missing API key",
			providerName: "moonshot",
			wantErrMsg:   "MOONSHOT_API_KEY not set",
		},
		{
			name:         "openai missing API key",
			providerName: "openai",
			wantErrMsg:   "OPENAI_API_KEY not set",
		},
		{
			name:         "gemini missing API key",
			providerName: "gemini",
			wantErrMsg:   "GEMINI_API_KEY not set",
		},
		{
			name:         "claude missing API key",
			providerName: "claude",
			wantErrMsg:   "ANTHROPIC_API_KEY not set",
		},
		{
			name:         "groq missing API key",
			providerName: "groq",
			wantErrMsg:   "GROQ_API_KEY not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure API key is not set
			t.Setenv("DEEPSEEK_API_KEY", "")
			t.Setenv("OPENAI_API_KEY", "")
			t.Setenv("GEMINI_API_KEY", "")
			t.Setenv("ANTHROPIC_API_KEY", "")
			t.Setenv("GROQ_API_KEY", "")
			t.Setenv("MOONSHOT_API_KEY", "")

			_, err := createProvider(tt.providerName)

			if err == nil {
				t.Errorf("createProvider(%q) expected error, got nil", tt.providerName)
				return
			}

			if err.Error() != tt.wantErrMsg {
				t.Errorf("createProvider(%q) error = %q, want %q", tt.providerName, err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestCreateProvider_UnknownProvider(t *testing.T) {
	tests := []string{
		"unknown",
		"invalid",
		"gpt",
		"",
	}

	for _, providerName := range tests {
		t.Run(providerName, func(t *testing.T) {
			_, err := createProvider(providerName)

			if err == nil {
				t.Errorf("createProvider(%q) expected error for unknown provider, got nil", providerName)
			}
		})
	}
}

func TestCreateProvider_OpenAISubscriptionAliasDoesNotRequireOpenAIAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	provider, err := createProvider("chatgpt")
	if err != nil {
		t.Fatalf("createProvider(chatgpt) error = %v, want nil", err)
	}
	if provider.Name() != "OpenAI Subscription" {
		t.Fatalf("provider.Name() = %q, want OpenAI Subscription", provider.Name())
	}
}

func TestCreateProvider_CaseInsensitive(t *testing.T) {
	tests := []struct {
		providerName string
		envKey       string
	}{
		{"DEEPSEEK", "DEEPSEEK_API_KEY"},
		{"DeepSeek", "DEEPSEEK_API_KEY"},
		{"deepseek", "DEEPSEEK_API_KEY"},
		{"KIMI", "MOONSHOT_API_KEY"},
		{"Kimi", "MOONSHOT_API_KEY"},
		{"kimi", "MOONSHOT_API_KEY"},
		{"MOONSHOT", "MOONSHOT_API_KEY"},
		{"Moonshot", "MOONSHOT_API_KEY"},
		{"OPENAI", "OPENAI_API_KEY"},
		{"OpenAI", "OPENAI_API_KEY"},
		{"GEMINI", "GEMINI_API_KEY"},
		{"Gemini", "GEMINI_API_KEY"},
		{"CLAUDE", "ANTHROPIC_API_KEY"},
		{"Claude", "ANTHROPIC_API_KEY"},
		{"OLLAMA", ""},
		{"Ollama", ""},
		{"GROQ", "GROQ_API_KEY"},
		{"Groq", "GROQ_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.providerName, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, "test-key")
			}

			provider, err := createProvider(tt.providerName)

			if err != nil {
				t.Errorf("createProvider(%q) unexpected error: %v", tt.providerName, err)
			}
			if provider == nil {
				t.Errorf("createProvider(%q) returned nil provider", tt.providerName)
			}
		})
	}
}

func TestResolveProviderName_Priority(t *testing.T) {
	tests := []struct {
		name         string
		flagValue    string
		envValue     string
		configValue  string
		expectedName string
	}{
		{
			name:         "CLI flag takes precedence",
			flagValue:    "openai",
			envValue:     "gemini",
			configValue:  "claude",
			expectedName: "openai",
		},
		{
			name:         "env var when no flag",
			flagValue:    "",
			envValue:     "gemini",
			configValue:  "claude",
			expectedName: "gemini",
		},
		{
			name:         "config when no flag or env",
			flagValue:    "",
			envValue:     "",
			configValue:  "claude",
			expectedName: "claude",
		},
		{
			name:         "default when nothing set",
			flagValue:    "",
			envValue:     "",
			configValue:  "",
			expectedName: "deepseek",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			t.Setenv("XELYON_PROVIDER", tt.envValue)

			result := resolveProviderName(tt.flagValue, tt.configValue)

			if result != tt.expectedName {
				t.Errorf("resolveProviderName() = %q, want %q", result, tt.expectedName)
			}
		})
	}
}

func TestResolveProviderName_NormalizesAndCanonicalizes(t *testing.T) {
	tests := []struct {
		name         string
		flagValue    string
		envValue     string
		configValue  string
		expectedName string
	}{
		{
			name:         "flag is normalized and alias is preserved",
			flagValue:    " Anthropic ",
			envValue:     "openai",
			configValue:  "gemini",
			expectedName: "anthropic",
		},
		{
			name:         "env is normalized",
			flagValue:    "",
			envValue:     " OpenAI ",
			configValue:  "gemini",
			expectedName: "openai",
		},
		{
			name:         "config is normalized",
			flagValue:    "",
			envValue:     "",
			configValue:  " Gemini ",
			expectedName: "gemini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XELYON_PROVIDER", tt.envValue)

			result := resolveProviderName(tt.flagValue, tt.configValue)
			if result != tt.expectedName {
				t.Fatalf("resolveProviderName() = %q, want %q", result, tt.expectedName)
			}
		})
	}
}

func TestProviderModelLookupKeys_PreservesAliasBeforeCanonicalFallback(t *testing.T) {
	keys := config.ProviderModelLookupKeys(" Anthropic ")
	if len(keys) != 2 {
		t.Fatalf("len(ProviderModelLookupKeys()) = %d, want 2", len(keys))
	}
	if keys[0] != "anthropic" || keys[1] != "claude" {
		t.Fatalf("ProviderModelLookupKeys() = %v, want [anthropic claude]", keys)
	}
}

func TestProviderModelLookupKeys_MoonshotAliasBeforeCanonicalFallback(t *testing.T) {
	keys := config.ProviderModelLookupKeys(" Moonshot ")
	if len(keys) != 2 {
		t.Fatalf("len(ProviderModelLookupKeys()) = %d, want 2", len(keys))
	}
	if keys[0] != "moonshot" || keys[1] != "kimi" {
		t.Fatalf("ProviderModelLookupKeys() = %v, want [moonshot kimi]", keys)
	}
}

func TestProviderConfig(t *testing.T) {
	// 全プロバイダーが createProvider で生成可能かテスト
	tests := []struct {
		name   string
		envKey string
	}{
		{"deepseek", "DEEPSEEK_API_KEY"},
		{"kimi", "MOONSHOT_API_KEY"},
		{"moonshot", "MOONSHOT_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"claude", "ANTHROPIC_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"groq", "GROQ_API_KEY"},
		{"ollama", ""}, // Ollama は API キー不要
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, "test-key")
			}
			provider, err := createProvider(tt.name)
			if err != nil {
				t.Errorf("createProvider(%q) unexpected error: %v", tt.name, err)
			}
			if provider == nil {
				t.Errorf("createProvider(%q) returned nil provider", tt.name)
			}
		})
	}
}
