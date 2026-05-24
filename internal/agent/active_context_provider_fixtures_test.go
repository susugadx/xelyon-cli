package agent

type activeContextProviderFixture struct {
	providerName      string
	providerConfigKey string
	model             string
}

var (
	activeContextOpenAIResponses = activeContextProviderFixture{
		providerName:      "openai",
		providerConfigKey: "openai",
		model:             "gpt-5.4",
	}
	activeContextAzureResponses = activeContextProviderFixture{
		providerName:      "azure",
		providerConfigKey: "azure",
		model:             "corp-gpt55-deployment",
	}
	activeContextOpenAIChatCompletions = activeContextProviderFixture{
		providerName:      "openai",
		providerConfigKey: "openai",
		model:             "gpt-4-turbo",
	}
	activeContextDeepSeek = activeContextProviderFixture{
		providerName:      "deepseek",
		providerConfigKey: "deepseek",
		model:             "deepseek-chat",
	}
	activeContextGemini = activeContextProviderFixture{
		providerName:      "gemini",
		providerConfigKey: "gemini",
		model:             "gemini-2.5-pro",
	}
	activeContextClaude = activeContextProviderFixture{
		providerName:      "claude",
		providerConfigKey: "claude",
		model:             "claude-sonnet-4-6",
	}
	activeContextGroq = activeContextProviderFixture{
		providerName:      "groq",
		providerConfigKey: "groq",
		model:             "llama-3.3-70b-versatile",
	}
	activeContextKimi = activeContextProviderFixture{
		providerName:      "kimi",
		providerConfigKey: "kimi",
		model:             "kimi-k2.6",
	}
	activeContextOllama = activeContextProviderFixture{
		providerName:      "ollama",
		providerConfigKey: "ollama",
		model:             "qwen2.5-coder:14b",
	}
	activeContextOpenRouterOpenAI = activeContextProviderFixture{
		providerName:      "openrouter",
		providerConfigKey: "openrouter",
		model:             "openai/gpt-4o",
	}
	activeContextOpenRouterClaude = activeContextProviderFixture{
		providerName:      "openrouter",
		providerConfigKey: "openrouter",
		model:             "anthropic/claude-sonnet-4.6",
	}
	activeContextBedrockClaude = activeContextProviderFixture{
		providerName:      "bedrock",
		providerConfigKey: "bedrock",
		model:             "global.anthropic.claude-sonnet-4-6",
	}
	activeContextBedrockConverse = activeContextProviderFixture{
		providerName:      "bedrock",
		providerConfigKey: "bedrock",
		model:             "amazon.nova-pro-v1:0",
	}
	activeContextUnsupported = activeContextProviderFixture{
		providerName:      "unsupported",
		providerConfigKey: "unsupported",
		model:             "unsupported-model",
	}
)

func applyActiveContextProviderFixture(agent *Agent, fixture activeContextProviderFixture) {
	agent.CurrentModel = fixture.model
	agent.ProviderName = fixture.providerName
	agent.ProviderConfigKey = fixture.providerConfigKey
}
