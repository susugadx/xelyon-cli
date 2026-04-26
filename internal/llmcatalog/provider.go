package llmcatalog

import "strings"

// ProviderModelDefaults は provider ごとの組み込みモデル設定を表す。
type ProviderModelDefaults struct {
	DefaultModel     string
	MaxOutputTokens  int
	AnthropicVersion string
	AnthropicBeta    []string
}

// ProviderDescriptor は LLM provider の静的メタ情報を表す。
type ProviderDescriptor struct {
	Key                  string
	Aliases              []string
	CredentialKind       string
	APIKeyEnv            string
	BaseURLEnv           string
	DefaultBaseURL       string
	StaticCredential     string
	SetupInstructions    []string
	DefaultSubAgentModel string
	SupportsImages       bool
	NativeWebSearch      bool
	PricingFamily        string
	CompressionModel     string
	CompressionThreshold int
	ModelDefaults        ProviderModelDefaults
}

var providerOrder = []string{
	"deepseek",
	"openai",
	"gemini",
	"claude",
	"ollama",
	"groq",
	"openrouter",
	"bedrock",
}

var displayProviderOrder = []string{
	"deepseek",
	"claude",
	"openai",
	"gemini",
	"groq",
	"ollama",
	"openrouter",
	"bedrock",
}

var nativeWebSearchProviderOrder = []string{
	"openai",
	"gemini",
	"claude",
}

var providerDescriptors = map[string]ProviderDescriptor{
	"deepseek": {
		Key:                  "deepseek",
		CredentialKind:       "api_key",
		APIKeyEnv:            "DEEPSEEK_API_KEY",
		SetupInstructions:    []string{"export DEEPSEEK_API_KEY=your-api-key"},
		DefaultSubAgentModel: "deepseek-chat",
		PricingFamily:        "deepseek",
		CompressionThreshold: 80000,
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "deepseek-chat",
			MaxOutputTokens: 16384,
		},
	},
	"openai": {
		Key:                  "openai",
		CredentialKind:       "api_key",
		APIKeyEnv:            "OPENAI_API_KEY",
		SetupInstructions:    []string{"export OPENAI_API_KEY=your-api-key"},
		DefaultSubAgentModel: "gpt-5.4-mini",
		SupportsImages:       true,
		NativeWebSearch:      true,
		PricingFamily:        "openai",
		CompressionModel:     "gpt-5.4-mini",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "gpt-5.4",
			MaxOutputTokens: 16384,
		},
	},
	"gemini": {
		Key:                  "gemini",
		CredentialKind:       "api_key",
		APIKeyEnv:            "GEMINI_API_KEY",
		SetupInstructions:    []string{"export GEMINI_API_KEY=your-api-key"},
		DefaultSubAgentModel: "gemini-3.1-flash-lite-preview",
		SupportsImages:       true,
		NativeWebSearch:      true,
		PricingFamily:        "gemini",
		CompressionModel:     "gemini-3.1-flash-lite-preview",
		CompressionThreshold: 180000,
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "gemini-3.1-pro-preview-customtools",
			MaxOutputTokens: 65536,
		},
	},
	"claude": {
		Key:                  "claude",
		Aliases:              []string{"anthropic"},
		CredentialKind:       "api_key",
		APIKeyEnv:            "ANTHROPIC_API_KEY",
		SetupInstructions:    []string{"export ANTHROPIC_API_KEY=your-api-key"},
		DefaultSubAgentModel: "claude-haiku-4-5-20251001",
		SupportsImages:       true,
		NativeWebSearch:      true,
		PricingFamily:        "claude",
		CompressionModel:     "claude-haiku-4-5",
		CompressionThreshold: 150000,
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:     "claude-sonnet-4-6",
			MaxOutputTokens:  64000,
			AnthropicVersion: "2023-06-01",
		},
	},
	"ollama": {
		Key:               "ollama",
		CredentialKind:    "base_url",
		BaseURLEnv:        "OLLAMA_BASE_URL",
		DefaultBaseURL:    "http://localhost:11434",
		SetupInstructions: []string{"ローカルの Ollama サーバーを起動してください"},
		PricingFamily:     "ollama",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "qwen2.5-coder:7b",
			MaxOutputTokens: 4096,
		},
	},
	"groq": {
		Key:                  "groq",
		CredentialKind:       "api_key",
		APIKeyEnv:            "GROQ_API_KEY",
		SetupInstructions:    []string{"export GROQ_API_KEY=your-api-key"},
		DefaultSubAgentModel: "llama-3.3-70b-versatile",
		PricingFamily:        "groq",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "meta-llama/llama-4-scout-17b-16e-instruct",
			MaxOutputTokens: 8192,
		},
	},
	"openrouter": {
		Key:                  "openrouter",
		CredentialKind:       "api_key",
		APIKeyEnv:            "OPENROUTER_API_KEY",
		SetupInstructions:    []string{"export OPENROUTER_API_KEY=your-api-key"},
		DefaultSubAgentModel: "openai/gpt-5.4-mini",
		SupportsImages:       true,
		PricingFamily:        "openrouter",
		CompressionThreshold: 120000,
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "anthropic/claude-sonnet-4.6",
			MaxOutputTokens: 64000,
		},
	},
	"bedrock": {
		Key:                  "bedrock",
		CredentialKind:       "static",
		StaticCredential:     "aws-credentials",
		SetupInstructions:    []string{"AWS認証チェーン（IAMロール、環境変数、~/.aws/credentials等）を設定"},
		SupportsImages:       true,
		PricingFamily:        "bedrock",
		CompressionModel:     "claude-haiku-4-5",
		CompressionThreshold: 150000,
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:     "global.anthropic.claude-sonnet-4-6-v1",
			MaxOutputTokens:  64000,
			AnthropicVersion: "bedrock-2023-05-31",
		},
	},
}

// NormalizeProviderKey は provider 名を lookup 用に正規化する。
func NormalizeProviderKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// CanonicalProviderKey は alias を実行時 provider key へ変換する。
func CanonicalProviderKey(name string) string {
	normalized := NormalizeProviderKey(name)
	for _, key := range providerOrder {
		if normalized == key {
			return key
		}
		for _, alias := range providerDescriptors[key].Aliases {
			if normalized == alias {
				return key
			}
		}
	}
	return normalized
}

// ProviderKeys は有効な provider key を返す。
func ProviderKeys(includeAliases bool) []string {
	keys := make([]string, 0, len(providerOrder))
	for _, key := range providerOrder {
		keys = append(keys, key)
		if includeAliases {
			keys = append(keys, providerDescriptors[key].Aliases...)
		}
	}
	return keys
}

// DisplayProviderKeys はユーザー表示用の provider key を返す。
func DisplayProviderKeys() []string {
	keys := make([]string, len(displayProviderOrder))
	copy(keys, displayProviderOrder)
	return keys
}

// ProviderDescriptorFor は provider key の静的メタ情報を返す。
func ProviderDescriptorFor(name string) (ProviderDescriptor, bool) {
	key := CanonicalProviderKey(name)
	entry, ok := providerDescriptors[key]
	if !ok {
		return ProviderDescriptor{}, false
	}
	return cloneProviderDescriptor(entry), true
}

// IsKnownProvider は provider key または alias が組み込み provider か返す。
func IsKnownProvider(name string) bool {
	_, ok := ProviderDescriptorFor(name)
	return ok
}

// ProviderModelLookupKeys は provider_models lookup の優先キーを返す。
func ProviderModelLookupKeys(provider string) []string {
	normalized := NormalizeProviderKey(provider)
	if normalized == "" {
		return nil
	}

	keys := []string{normalized}
	canonical := CanonicalProviderKey(normalized)
	if canonical == "" || canonical == normalized {
		if entry, ok := providerDescriptors[canonical]; ok {
			keys = append(keys, entry.Aliases...)
		}
		return keys
	}

	keys = append(keys, canonical)
	if entry, ok := providerDescriptors[canonical]; ok {
		for _, alias := range entry.Aliases {
			if alias != normalized {
				keys = append(keys, alias)
			}
		}
	}
	return dedupe(keys)
}

// NativeWebSearchProviderKeys は native web search 対応 provider の key を返す。
func NativeWebSearchProviderKeys(includeAliases bool) []string {
	var keys []string
	for _, key := range nativeWebSearchProviderOrder {
		entry := providerDescriptors[key]
		if !entry.NativeWebSearch {
			continue
		}
		keys = append(keys, key)
		if includeAliases {
			keys = append(keys, entry.Aliases...)
		}
	}
	return keys
}

// DefaultProviderModelDescriptors は provider ごとの組み込みモデル設定を返す。
func DefaultProviderModelDescriptors() map[string]ProviderModelDefaults {
	out := make(map[string]ProviderModelDefaults, len(providerDescriptors))
	for _, key := range providerOrder {
		defaults := providerDescriptors[key].ModelDefaults
		out[key] = cloneProviderModelDefaults(defaults)
	}
	return out
}

func cloneProviderDescriptor(entry ProviderDescriptor) ProviderDescriptor {
	entry.Aliases = cloneStrings(entry.Aliases)
	entry.SetupInstructions = cloneStrings(entry.SetupInstructions)
	entry.ModelDefaults = cloneProviderModelDefaults(entry.ModelDefaults)
	return entry
}

func cloneProviderModelDefaults(defaults ProviderModelDefaults) ProviderModelDefaults {
	defaults.AnthropicBeta = cloneStrings(defaults.AnthropicBeta)
	return defaults
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
