package config

func defaultGeneralConfig() GeneralConfig {
	return GeneralConfig{
		UILanguage:    "auto", // デフォルト: 自動判定（フォールバック: ja）
		ToolLoopLimit: 0,      // 内部既定: 0 = unlimited tool loop
	}
}

func defaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Enabled:              true, // デフォルトON - コスト削減のため
		TriggerPercent:       80,   // 80%で自動圧縮
		KeepRecent:           20,   // 履歴を多めに保持
		ThresholdTokens:      0,    // 内部: 0 = 使用率ベース
		TokenThreshold:       0,    // 内部: 0 = カスタム絶対閾値を無効化
		Model:                "",   // 内部: 空 = プロバイダー別デフォルト圧縮モデル
		PreferCompactAPI:     true, // 内部: OpenAI Compact API 優先
		ClaudeCompaction:     true, // 内部: Claude Compaction 優先
		CompactionTrigger:    150000,
		ClearToolUses:        true, // 内部: Claude系の tool_use/tool_result clearing
		ClearToolUsesTrigger: 80000,
		ClearToolInputs:      false,
		ProviderThresholds:   defaultCompressionProviderThresholds(),
	}
}

func defaultCompressionProviderThresholds() map[string]int {
	return map[string]int{}
}

func defaultLoopDetectionConfig() LoopDetectionConfig {
	return LoopDetectionConfig{
		Threshold: 3,
	}
}

func defaultAPIRetryConfig() APIRetryConfig {
	return APIRetryConfig{
		Count:        3,
		InitialDelay: 1,
		MaxDelay:     30,
		Timeout:      3600, // xhigh thinking 対応（1時間）
	}
}

func defaultDiffConfig() DiffConfig {
	return DiffConfig{
		ContextLines: 10,
	}
}

func defaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		Mode:              string(ExecutionBalanced),
		AlwaysConfirm:     []string{},
		SafeShellCommands: []string{},
	}
}

func defaultToolConfirmConfig() ToolConfirmConfig {
	return ToolConfirmConfig{
		AutoApproveSafe:   true,
		AutoApproveMedium: false,
	}
}

func defaultPromptCacheConfig() PromptCacheConfig {
	return PromptCacheConfig{
		Enabled:  true,
		CacheTTL: "5m",
	}
}

func defaultResponsesConfig() ResponsesConfig {
	return ResponsesConfig{
		Store:             true,
		PersistResponseID: true,
		ServerCompaction: ResponsesServerCompactionConfig{
			Enabled:          true,
			CompactThreshold: 0,
			LocalFallback:    true,
		},
	}
}

func defaultPasteConfig() PasteConfig {
	return PasteConfig{
		BracketedPaste: true, // デフォルトON - 複数行ペースト対応
		MaxLines:       10000,
		MaxBytes:       1048576,
	}
}

func defaultStreamingConfig() StreamingConfig {
	return StreamingConfig{
		IdleTimeoutSeconds:     30,  // チャンク間隔タイムアウト（30秒）
		ThinkingTimeoutSeconds: 120, // thinking request のレスポンス開始 / SSE 進捗待ち上限
		ShowFileInfo:           true,
		ShowSearchProgress:     true,
		StreamBashOutput:       true,
	}
}

func defaultBashConfig() BashConfig {
	return BashConfig{
		SafetyLevel:     "permissive", // 確認出るので安全、利便性向上
		SafeCommands:    []string{},
		AllowRedirect:   true, // 利便性向上
		AllowInlineEdit: true, // 利便性向上
	}
}

func defaultListDirConfig() ListDirConfig {
	return ListDirConfig{
		AdditionalIgnoreDirs: []string{},
	}
}

func defaultProjectMapConfig() ProjectMapConfig {
	return ProjectMapConfig{
		Enabled:              true,
		ContextRatio:         ProjectMapContextRatioDefault,
		AdditionalIgnoreDirs: []string{},
	}
}

func defaultAgentInstructionsConfig() AgentInstructionsConfig {
	return AgentInstructionsConfig{
		Project: AgentInstructionsProjectConfig{
			Mode:              "fallback",
			Files:             defaultAgentInstructionProjectFiles(),
			IncludeGitignored: false,
		},
		Global: AgentInstructionsGlobalConfig{
			Enabled: false,
			Files:   defaultAgentInstructionGlobalFiles(),
		},
		IncludeLocalFiles: false,
		ExpandImports:     false,
		MaxFileBytes:      20000,
		MaxTotalBytes:     60000,
	}
}

func defaultAgentInstructionProjectFiles() []string {
	return []string{"AGENTS.md", "CLAUDE.md", ".claude/CLAUDE.md"}
}

func defaultAgentInstructionGlobalFiles() []string {
	return []string{"~/.xelyon/AGENTS.md", "~/.xelyon/CLAUDE.md"}
}

func defaultGitStageConfig() GitStageConfig {
	return GitStageConfig{
		BatchConfirm: true,
	}
}

func defaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		ResponsesAPIModels: []string{}, // 内部: プレフィックスマッチのフォールバック（YAML 直接編集で追加可能）
	}
}

func defaultGeminiConfig() GeminiConfig {
	return GeminiConfig{
		ServiceTier: GeminiServiceTierStandard,
	}
}

func defaultThinkingConfig() ThinkingConfig {
	return ThinkingConfig{
		Enabled: false,    // 内部 runtime 初期値（/thinking コマンドが正規ルート）
		Level:   "medium", // 内部 runtime 初期値（/thinking コマンドが正規ルート）
	}
}

func defaultOutputConfig() OutputConfig {
	return OutputConfig{
		MaxLines:         5,  // デフォルト5行で折りたたみ
		AssistantUpdates: "", // 空 = Normal Mode は phase / Plan Mode は verbose
	}
}

func defaultWebSearchConfig() WebSearchConfig {
	return WebSearchConfig{
		CacheEnabled: true,
		CacheTTL:     3600, // 1時間
		CacheSize:    50,
	}
}

func defaultSubAgentConfig() SubAgentConfig {
	return SubAgentConfig{
		Enabled:       true,
		DefaultModel:  "gpt-5.4-mini",
		DefaultEffort: "",
		MaxConcurrent: 1,
	}
}

func defaultMCPConfig() MCPConfig {
	return MCPConfig{
		Enabled:  true,  // デフォルトON - MCP接続有効
		Headless: false, // デフォルトOFF - Headlessモードでは接続しない
	}
}

func defaultFinalChecksConfig() FinalChecksConfig {
	return FinalChecksConfig{
		Commands: nil, // デフォルト: final checks なし
		Timeout:  600, // 10分タイムアウト
	}
}
