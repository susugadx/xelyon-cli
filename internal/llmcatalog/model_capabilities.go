package llmcatalog

import "strings"

// KnownImageInputSupport は catalog metadata で確認済みの画像入力対応可否を返す。
func KnownImageInputSupport(provider, model string) (bool, bool) {
	provider = CanonicalProviderKey(provider)
	model = normalizeModelName(model)
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
	provider = CanonicalProviderKey(provider)
	model = normalizeModelName(model)
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
	for _, known := range KnownModelNamesForProvider(provider) {
		if strings.EqualFold(model, known) {
			return true, true
		}
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
