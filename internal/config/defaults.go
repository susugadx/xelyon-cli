package config

// DefaultConfig はデフォルト設定
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider:     "deepseek",
		DefaultModel:        "deepseek-v4-flash",
		General:             defaultGeneralConfig(),
		ProviderModels:      defaultProviderModels(),
		Review:              defaultReviewConfig(),
		Compression:         defaultCompressionConfig(),
		LoopDetection:       defaultLoopDetectionConfig(),
		APIRetry:            defaultAPIRetryConfig(),
		Diff:                defaultDiffConfig(),
		Execution:           defaultExecutionConfig(),
		ToolConfirm:         defaultToolConfirmConfig(),
		CommandAliases:      map[string]string{},
		PromptCache:         defaultPromptCacheConfig(),
		Paste:               defaultPasteConfig(),
		Responses:           defaultResponsesConfig(),
		Streaming:           defaultStreamingConfig(),
		Bash:                defaultBashConfig(),
		ListDir:             defaultListDirConfig(),
		ProjectMap:          defaultProjectMapConfig(),
		AgentInstructions:   defaultAgentInstructionsConfig(),
		GitStage:            defaultGitStageConfig(),
		LSP:                 defaultLSPConfig(),
		OpenAI:              defaultOpenAIConfig(),
		Gemini:              defaultGeminiConfig(),
		Thinking:            defaultThinkingConfig(),
		Output:              defaultOutputConfig(),
		WebSearch:           defaultWebSearchConfig(),
		SubAgent:            defaultSubAgentConfig(),
		MCP:                 defaultMCPConfig(),
		FinalChecks:         defaultFinalChecksConfig(),
		providerModelsStore: defaultProviderModelStore(),
	}
}
