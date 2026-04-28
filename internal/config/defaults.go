package config

// DefaultConfig はデフォルト設定
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider:     "deepseek",
		DefaultModel:        "deepseek-v4-flash",
		General:             defaultGeneralConfig(),
		ProviderModels:      defaultProviderModels(),
		Compression:         defaultCompressionConfig(),
		LoopDetection:       defaultLoopDetectionConfig(),
		APIRetry:            defaultAPIRetryConfig(),
		Diff:                defaultDiffConfig(),
		Execution:           defaultExecutionConfig(),
		ToolConfirm:         defaultToolConfirmConfig(),
		CommandAliases:      defaultCommandAliases(),
		PromptCache:         defaultPromptCacheConfig(),
		Paste:               defaultPasteConfig(),
		Responses:           defaultResponsesConfig(),
		Streaming:           defaultStreamingConfig(),
		Bash:                defaultBashConfig(),
		ListDir:             defaultListDirConfig(),
		ProjectMap:          defaultProjectMapConfig(),
		GitStage:            defaultGitStageConfig(),
		LSP:                 defaultLSPConfig(),
		OpenAI:              defaultOpenAIConfig(),
		Thinking:            defaultThinkingConfig(),
		Output:              defaultOutputConfig(),
		WebSearch:           defaultWebSearchConfig(),
		SubAgent:            defaultSubAgentConfig(),
		MCP:                 defaultMCPConfig(),
		FinalChecks:         defaultFinalChecksConfig(),
		providerModelsStore: defaultProviderModelStore(),
	}
}
