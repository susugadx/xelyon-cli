package llmcatalog

import "strings"

// KnownImageInputSupport は catalog metadata で確認済みの画像入力対応可否を返す。
func KnownImageInputSupport(provider, model string) (bool, bool) {
	model = normalizeModelName(model)
	provider = ResolveProviderRoute(provider, model, model).CapabilityFamily
	if provider == "" || model == "" {
		return false, false
	}

	switch provider {
	case "azure":
		return KnownImageInputSupport("openai", model)
	case "openrouter":
		return knownOpenRouterImageInputSupport(model)
	case "openai":
		return knownOpenAIImageInputSupport(model)
	case "claude":
		return knownClaudeImageInputSupport(model)
	case "gemini":
		return knownGeminiImageInputSupport(model)
	case "kimi":
		return knownKimiImageInputSupport(model)
	case "deepseek", "groq", "ollama":
		return false, true
	default:
		return false, false
	}
}

// KnownWebSearchSupport は catalog metadata で確認済みの native web search 対応可否を返す。
func KnownWebSearchSupport(provider, model string) (bool, bool) {
	model = normalizeModelName(model)
	provider = ResolveProviderRoute(provider, model, model).CapabilityFamily
	if provider == "" || model == "" {
		return false, false
	}

	switch provider {
	case "azure":
		return false, true
	case "openai":
		return knownOpenAIWebSearchSupport(model)
	case "gemini":
		return knownExactProviderModelSupport("gemini", model)
	case "claude":
		return knownExactProviderModelSupport("claude", model)
	case "kimi":
		return knownKimiWebSearchSupport(model)
	case "openrouter", "bedrock", "deepseek", "groq", "ollama":
		return false, true
	default:
		return false, false
	}
}

func knownOpenRouterImageInputSupport(model string) (bool, bool) {
	owner, routedModel, ok := splitRoutedModelID(model)
	if !ok {
		return false, false
	}

	switch owner {
	case "openai":
		return knownOpenAIImageInputSupport(routedModel)
	case "anthropic":
		return knownClaudeImageInputSupport(routedModel)
	case "google":
		return knownGeminiImageInputSupport(routedModel)
	case "moonshotai":
		return knownKimiImageInputSupport(routedModel)
	case "meta-llama":
		if strings.HasPrefix(routedModel, "llama-4-scout") {
			return true, true
		}
		return false, false
	case "deepseek":
		return false, true
	default:
		return false, false
	}
}

func knownOpenAIImageInputSupport(model string) (bool, bool) {
	switch {
	case strings.HasPrefix(model, "gpt-5"),
		strings.HasPrefix(model, "gpt-4.1"),
		strings.HasPrefix(model, "gpt-4o"):
		return true, true
	case strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3-mini"):
		return false, true
	default:
		return false, false
	}
}

func knownClaudeImageInputSupport(model string) (bool, bool) {
	if strings.HasPrefix(model, "claude-") {
		return true, true
	}
	return false, false
}

func knownGeminiImageInputSupport(model string) (bool, bool) {
	if strings.HasPrefix(model, "gemini-") && !strings.EqualFold(model, "gemini-pro") {
		return true, true
	}
	return false, false
}

func knownKimiImageInputSupport(model string) (bool, bool) {
	if strings.HasPrefix(model, "kimi-") {
		return true, true
	}
	return false, false
}

func knownOpenAIWebSearchSupport(model string) (bool, bool) {
	switch {
	case strings.HasPrefix(model, "gpt-5"),
		strings.HasPrefix(model, "gpt-4o"),
		strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3"),
		strings.HasPrefix(model, "o4"):
		return true, true
	default:
		return false, false
	}
}

func knownKimiWebSearchSupport(model string) (bool, bool) {
	if strings.HasPrefix(model, "kimi-") {
		return true, true
	}
	return false, false
}

func knownExactProviderModelSupport(provider, model string) (bool, bool) {
	if IsExactKnownModelNameForProvider(provider, model) {
		return true, true
	}
	return false, false
}

func splitRoutedModelID(model string) (string, string, bool) {
	owner, routedModel, ok := strings.Cut(model, "/")
	owner = strings.ToLower(strings.TrimSpace(owner))
	routedModel = normalizeModelName(routedModel)
	if !ok || owner == "" || routedModel == "" {
		return "", "", false
	}
	return owner, routedModel, true
}

// ModelCapabilitySupport は model capability のローカル判定結果を表す。
type ModelCapabilitySupport struct {
	Known       bool
	Supported   bool
	Reason      string
	Replacement string
}

// GeminiFunctionCallingSupport は Gemini model の function calling 対応状況を返す。
func GeminiFunctionCallingSupport(model string) ModelCapabilitySupport {
	model = CanonicalModelNameForProvider("gemini", model)
	if model == "" {
		return ModelCapabilitySupport{}
	}
	if support, ok := geminiFunctionCallingExact[model]; ok {
		return support
	}
	for _, family := range geminiFunctionCallingFamilies {
		if strings.HasPrefix(model, family.Prefix) {
			return family.Support
		}
	}
	return ModelCapabilitySupport{}
}

var geminiFunctionCallingExact = map[string]ModelCapabilitySupport{
	"gemini-3.5-flash": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-flash-lite": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-flash-lite-preview": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-pro": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-pro-preview": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-pro-preview-customtools": {
		Known:     true,
		Supported: true,
	},
	"gemini-3-pro-preview": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.5-pro": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.5-flash": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.5-flash-lite": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash-001": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash-exp": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash-lite": {
		Known:       true,
		Supported:   false,
		Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
		Replacement: "gemini-3.1-flash-lite",
	},
	"gemini-2.0-flash-lite-001": {
		Known:       true,
		Supported:   false,
		Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
		Replacement: "gemini-3.1-flash-lite",
	},
}

type geminiFunctionCallingFamily struct {
	Prefix  string
	Support ModelCapabilitySupport
}

var geminiFunctionCallingFamilies = []geminiFunctionCallingFamily{
	{
		Prefix: "gemini-2.0-flash-lite-",
		Support: ModelCapabilitySupport{
			Known:       true,
			Supported:   false,
			Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
			Replacement: "gemini-3.1-flash-lite",
		},
	},
	{Prefix: "gemini-3.5-flash-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-3.1-flash-lite-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-3.1-pro-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.5-pro-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.5-flash-lite-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.5-flash-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.0-flash-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
}
