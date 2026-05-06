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
	DisplayName          string
	Aliases              []string
	CredentialKind       string
	APIKeyEnv            string
	BaseURLEnv           string
	DefaultBaseURL       string
	CredentialEnvVars    []string
	CredentialEnvVarSets [][]string
	StaticCredential     string
	SetupInstructions    []string
	DefaultSubAgentModel string
	SupportsImages       bool
	NativeWebSearch      bool
	SupportsResponsesAPI bool
	PricingFamily        string
	CompressionModel     string
	ModelDefaults        ProviderModelDefaults
}

var providerOrder = []string{
	"deepseek",
	"kimi",
	"openai",
	"azure",
	"gemini",
	"claude",
	"ollama",
	"groq",
	"openrouter",
	"bedrock",
}

var displayProviderOrder = []string{
	"deepseek",
	"kimi",
	"claude",
	"openai",
	"azure",
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
		DefaultSubAgentModel: "deepseek-v4-flash",
		PricingFamily:        "deepseek",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "deepseek-v4-flash",
			MaxOutputTokens: 16384,
		},
	},
	"kimi": {
		Key:                  "kimi",
		DisplayName:          "Kimi",
		Aliases:              []string{"moonshot"},
		CredentialKind:       "api_key",
		APIKeyEnv:            "MOONSHOT_API_KEY",
		SetupInstructions:    []string{"export MOONSHOT_API_KEY=your-api-key"},
		DefaultSubAgentModel: "kimi-k2.5",
		SupportsImages:       true,
		PricingFamily:        "kimi",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "kimi-k2.6",
			MaxOutputTokens: 32768,
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
		SupportsResponsesAPI: true,
		PricingFamily:        "openai",
		CompressionModel:     "gpt-5.4-mini",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "gpt-5.4",
			MaxOutputTokens: 16384,
		},
	},
	"azure": {
		Key:                  "azure",
		DisplayName:          "Azure OpenAI",
		CredentialKind:       "api_key",
		APIKeyEnv:            "AZURE_OPENAI_API_KEY",
		BaseURLEnv:           "AZURE_OPENAI_BASE_URL",
		CredentialEnvVars:    []string{"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_AUTH_TOKEN", "AZURE_OPENAI_AUTH_TOKEN_COMMAND", "AZURE_OPENAI_BASE_URL"},
		CredentialEnvVarSets: [][]string{{"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_BASE_URL"}, {"AZURE_OPENAI_AUTH_TOKEN", "AZURE_OPENAI_BASE_URL"}, {"AZURE_OPENAI_AUTH_TOKEN_COMMAND", "AZURE_OPENAI_BASE_URL"}},
		SetupInstructions: []string{
			"export AZURE_OPENAI_BASE_URL=https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1",
			"export AZURE_OPENAI_API_KEY=your-api-key",
			"# or: export AZURE_OPENAI_AUTH_TOKEN=$(az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv)",
			"# or: export AZURE_OPENAI_AUTH_TOKEN_COMMAND='az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv'",
		},
		SupportsImages:       true,
		SupportsResponsesAPI: true,
		PricingFamily:        "openai",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "azure-gpt-5.4",
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
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:    "anthropic/claude-sonnet-4.6",
			MaxOutputTokens: 64000,
		},
	},
	"bedrock": {
		Key:               "bedrock",
		CredentialKind:    "static",
		StaticCredential:  "aws-credentials",
		SetupInstructions: []string{"AWS認証チェーン（IAMロール、環境変数、~/.aws/credentials等）を設定"},
		SupportsImages:    true,
		PricingFamily:     "bedrock",
		CompressionModel:  "claude-haiku-4-5",
		ModelDefaults: ProviderModelDefaults{
			DefaultModel:     "global.anthropic.claude-sonnet-4-6",
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
		if normalized != "" && normalized == NormalizeProviderKey(providerDescriptors[key].DisplayName) {
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

// ProviderConfigKey は provider_models/session owner に使う config key を返す。
// 表示名は永続化キーではないため canonical key に寄せ、明示 alias はそのまま保持する。
func ProviderConfigKey(name string) string {
	normalized := NormalizeProviderKey(name)
	if normalized == "" {
		return ""
	}
	for _, key := range providerOrder {
		if normalized == NormalizeProviderKey(providerDescriptors[key].DisplayName) {
			return key
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

// RequiredCredentialEnvVars は provider の認証案内に表示する環境変数名を返す。
func (d ProviderDescriptor) RequiredCredentialEnvVars() []string {
	if len(d.CredentialEnvVars) > 0 {
		return cloneStrings(d.CredentialEnvVars)
	}
	if d.CredentialKind == "api_key" || d.CredentialKind == "" {
		if d.APIKeyEnv != "" {
			return []string{d.APIKeyEnv}
		}
	}
	return nil
}

// ProviderCredentialEnvVars は provider の認証案内に表示する環境変数名を返す。
func ProviderCredentialEnvVars(name string) []string {
	entry, ok := ProviderDescriptorFor(name)
	if !ok {
		return nil
	}
	return entry.RequiredCredentialEnvVars()
}

// CredentialEnvVarSets は provider 利用に必要な環境変数セットを返す。
// いずれか 1 セットが満たされれば provider は利用可能。
func (d ProviderDescriptor) CredentialEnvVarSetsForAvailability() [][]string {
	if len(d.CredentialEnvVarSets) > 0 {
		return cloneStringSlices(d.CredentialEnvVarSets)
	}
	if vars := d.RequiredCredentialEnvVars(); len(vars) > 0 {
		return [][]string{vars}
	}
	return nil
}

// ProviderCredentialEnvVarSets は provider 利用に必要な環境変数セットを返す。
func ProviderCredentialEnvVarSets(name string) [][]string {
	entry, ok := ProviderDescriptorFor(name)
	if !ok {
		return nil
	}
	return entry.CredentialEnvVarSetsForAvailability()
}

// ProviderSupportsResponsesAPI は provider が OpenAI Responses API 形の実行経路を持つか返す。
func ProviderSupportsResponsesAPI(name string) bool {
	entry, ok := ProviderDescriptorFor(name)
	return ok && entry.SupportsResponsesAPI
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
	normalized = ProviderConfigKey(normalized)

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
	entry.CredentialEnvVars = cloneStrings(entry.CredentialEnvVars)
	entry.CredentialEnvVarSets = cloneStringSlices(entry.CredentialEnvVarSets)
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

func cloneStringSlices(values [][]string) [][]string {
	if len(values) == 0 {
		return nil
	}
	out := make([][]string, len(values))
	for i, value := range values {
		out[i] = cloneStrings(value)
	}
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
