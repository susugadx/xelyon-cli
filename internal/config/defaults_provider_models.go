package config

func defaultProviderModels() map[string]ProviderModelConfig {
	return map[string]ProviderModelConfig{
		"deepseek": {
			DefaultModel:    "deepseek-chat",
			MaxOutputTokens: 16384,
		},
		"openai": {
			DefaultModel:    "gpt-5.4",
			MaxOutputTokens: 16384,
		},
		"gemini": {
			DefaultModel:    "gemini-3.1-pro-preview-customtools",
			MaxOutputTokens: 65536,
		},
		"claude": {
			DefaultModel:     "claude-sonnet-4-6",
			MaxOutputTokens:  64000,
			AnthropicVersion: "2023-06-01",
		},
		"ollama": {
			DefaultModel:    "qwen2.5-coder:7b",
			MaxOutputTokens: 4096,
		},
		"groq": {
			DefaultModel:    "meta-llama/llama-4-scout-17b-16e-instruct",
			MaxOutputTokens: 8192,
		},
		"openrouter": {
			DefaultModel:    "anthropic/claude-sonnet-4.6",
			MaxOutputTokens: 64000,
		},
		"bedrock": {
			DefaultModel:     "global.anthropic.claude-sonnet-4-6-v1",
			MaxOutputTokens:  64000,
			AnthropicVersion: "bedrock-2023-05-31",
		},
	}
}

func defaultProviderModelStore() providerModelStore {
	return providerModelStore{
		state: providerModelSectionStateInMemoryEffectiveOnly,
	}
}
