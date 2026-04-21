package config

import "strings"

func (c *Config) configuredDefaultModelAppliesToProvider(provider, model string) bool {
	if model == "" {
		return false
	}

	if c != nil {
		matchedAnyProvider := false
		for _, candidate := range providerConfigCandidates() {
			pm, ok := c.GetProviderModelConfig(candidate)
			if !ok || pm.DefaultModel != model {
				continue
			}
			matchedAnyProvider = true
			if SameProviderRuntimeIdentity(provider, candidate) {
				return true
			}
		}
		if matchedAnyProvider {
			return false
		}
	}

	if providerTreatsConfiguredDefaultModelAsOpaque(provider, model) {
		return true
	}

	inferredProvider := InferProviderFromModel(model)
	if inferredProvider == "" {
		return true
	}
	return SameProviderRuntimeIdentity(provider, inferredProvider)
}

// InferProviderFromModel はモデル名から provider を推定する。
// 推定できない場合は空文字を返す。
func InferProviderFromModel(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(normalized, "gpt-"),
		normalized == "codex-mini",
		normalized == "codex",
		isOpenAIReasoningModelName(normalized):
		return "openai"
	case strings.HasPrefix(normalized, "gemini"):
		return "gemini"
	case strings.HasPrefix(normalized, "claude"):
		return "claude"
	case strings.HasPrefix(normalized, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(normalized, "global.anthropic."):
		return "bedrock"
	case strings.Contains(normalized, "/"):
		return "openrouter"
	default:
		return ""
	}
}

func isOpenAIReasoningModelName(model string) bool {
	switch {
	case strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3"),
		strings.HasPrefix(model, "o4"):
		return true
	default:
		return false
	}
}

func providerTreatsConfiguredDefaultModelAsOpaque(provider, model string) bool {
	switch CanonicalProviderName(provider) {
	case "ollama":
		return true
	case "groq":
		return strings.Contains(strings.TrimSpace(model), "/")
	default:
		return false
	}
}
