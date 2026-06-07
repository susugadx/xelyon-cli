package config

import "strings"

type defaultApplyOptions struct {
	lspSectionExists bool
	lspServersExists bool
}

func resolveDefaultApplyOptions(opts []defaultApplyOptions) defaultApplyOptions {
	options := defaultApplyOptions{
		lspSectionExists: true,
		lspServersExists: true,
	}
	if len(opts) > 0 {
		return opts[0]
	}
	return options
}

func applyInternalSettingNormalization(cfg *Config, defaults *Config) {
	// --- 内部設定の正規化 ---
	if cfg.General.ToolLoopLimit < 0 {
		cfg.General.ToolLoopLimit = 0 // 負値は 0（無制限）にサイレント補正
	}
	// Execution: mode が空なら balanced をデフォルト適用
	if cfg.Execution.Mode == "" {
		cfg.Execution.Mode = defaults.Execution.Mode
	}
}

func applyProviderAndModelDefaults(cfg *Config) {
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "deepseek"
	}
	cfg.refreshEffectiveProviderModels()
}

func applyNestedSectionDefaults(cfg *Config, defaults *Config, options defaultApplyOptions) {
	for _, applier := range nestedDefaultAppliers {
		applier(cfg, defaults, options)
	}
}

type nestedDefaultApplier func(cfg *Config, defaults *Config, options defaultApplyOptions)

var nestedDefaultAppliers = []nestedDefaultApplier{
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyLoopDetectionDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyAPIRetryDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyCompressionDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) {
		applyProviderHistoryReductionDefaults(cfg, defaults)
	},
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyPasteDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyStreamingDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, options defaultApplyOptions) {
		applyLSPDefaults(cfg, defaults, options)
	},
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyThinkingDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyWebSearchDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) { applyFinalChecksDefaults(cfg, defaults) },
	func(cfg *Config, defaults *Config, _ defaultApplyOptions) {
		applyAgentInstructionDefaults(cfg, defaults)
	},
}

func applyLoopDetectionDefaults(cfg *Config, defaults *Config) {
	// ネストされた構造体のデフォルト値を適用（YAMLで省略された場合）
	if cfg.LoopDetection.Threshold == 0 {
		cfg.LoopDetection = defaults.LoopDetection
	}
}

func applyAPIRetryDefaults(cfg *Config, defaults *Config) {
	if cfg.APIRetry.Count == 0 {
		cfg.APIRetry = defaults.APIRetry
	}
	if cfg.APIRetry.Timeout == 0 {
		cfg.APIRetry.Timeout = defaults.APIRetry.Timeout
	}
}

func applyCompressionDefaults(cfg *Config, defaults *Config) {
	// Compression: ゼロ値のフィールドのみデフォルト適用（構造体全体の上書きはしない）
	if cfg.Compression.TriggerPercent == 0 {
		cfg.Compression.TriggerPercent = defaults.Compression.TriggerPercent
	}
	if cfg.Compression.KeepRecent == 0 {
		cfg.Compression.KeepRecent = defaults.Compression.KeepRecent
	}
}

func applyProviderHistoryReductionDefaults(cfg *Config, defaults *Config) {
	if cfg.ProviderHistoryReduction.Mode == "" {
		cfg.ProviderHistoryReduction.Mode = defaults.ProviderHistoryReduction.Mode
	}
	cfg.ProviderHistoryReduction.RawOutputArtifacts = mergeProviderHistoryRawOutputArtifactsConfig(
		defaults.ProviderHistoryReduction.RawOutputArtifacts,
		cfg.ProviderHistoryReduction.RawOutputArtifacts,
	)
}

func applyPasteDefaults(cfg *Config, defaults *Config) {
	// Paste: 他のフィールドがすべてデフォルト値の場合、BracketedPaste もデフォルト適用
	// （既存の設定ファイルに bracketed_paste がない場合に true にするため）
	if cfg.Paste.MaxLines == 0 && cfg.Paste.MaxBytes == 0 {
		// Paste セクションが未設定 → 全てデフォルト適用
		cfg.Paste = defaults.Paste
	} else {
		// 個別フィールドのデフォルト適用
		if cfg.Paste.MaxLines == 0 {
			cfg.Paste.MaxLines = defaults.Paste.MaxLines
		}
		if cfg.Paste.MaxBytes == 0 {
			cfg.Paste.MaxBytes = defaults.Paste.MaxBytes
		}
		// BracketedPaste: 明示的に false に設定されていない限り、デフォルト (true) を適用
		// 注: YAML で bracketed_paste: false を明示的に設定した場合のみ false になる
		// 既存の設定ファイル（フィールドがない）では true にする
	}
}

func applyStreamingDefaults(cfg *Config, defaults *Config) {
	if cfg.Streaming.IdleTimeoutSeconds == 0 {
		cfg.Streaming.IdleTimeoutSeconds = defaults.Streaming.IdleTimeoutSeconds
	}
}

func applyLSPDefaults(cfg *Config, defaults *Config, options defaultApplyOptions) {
	// LSP設定のデフォルト適用
	// lsp.servers が YAML に明示されている場合は nil/empty/non-empty をそのまま保持し、
	// sibling field の値を巻き戻さない。
	if !options.lspSectionExists {
		cfg.LSP = defaults.LSP
	} else if !options.lspServersExists && cfg.LSP.Servers == nil {
		cfg.LSP.Servers = defaults.LSP.Servers
	}
}

func applyThinkingDefaults(cfg *Config, defaults *Config) {
	// Note: Diff.ContextLines は0が有効値なので、デフォルト適用は行わない
	// Thinking: 内部 runtime 初期値（/thinking コマンドが正規ルート）
	if cfg.Thinking.Level == "" {
		cfg.Thinking.Level = defaults.Thinking.Level
	}
}

func applyWebSearchDefaults(cfg *Config, defaults *Config) {
	// WebSearch: 全てゼロ値の場合のみデフォルト適用
	if !cfg.WebSearch.CacheEnabled && cfg.WebSearch.CacheTTL == 0 && cfg.WebSearch.CacheSize == 0 {
		cfg.WebSearch = defaults.WebSearch
	}
}

func applyFinalChecksDefaults(cfg *Config, defaults *Config) {
	// FinalChecks: Timeout が 0 の場合はデフォルト適用
	if cfg.FinalChecks.Timeout == 0 {
		cfg.FinalChecks.Timeout = defaults.FinalChecks.Timeout
	}
}

func applyAgentInstructionDefaults(cfg *Config, defaults *Config) {
	// AgentInstructions: mode とサイズ上限のデフォルト適用
	if strings.TrimSpace(cfg.AgentInstructions.Project.Mode) == "" {
		cfg.AgentInstructions.Project.Mode = defaults.AgentInstructions.Project.Mode
	} else {
		cfg.AgentInstructions.Project.Mode = normalizeAgentInstructionProjectMode(cfg.AgentInstructions.Project.Mode)
	}
	if len(cfg.AgentInstructions.Project.Files) == 0 {
		cfg.AgentInstructions.Project.Files = append([]string(nil), defaults.AgentInstructions.Project.Files...)
	}
	if len(cfg.AgentInstructions.Global.Files) == 0 {
		cfg.AgentInstructions.Global.Files = append([]string(nil), defaults.AgentInstructions.Global.Files...)
	}
	if cfg.AgentInstructions.MaxFileBytes <= 0 {
		cfg.AgentInstructions.MaxFileBytes = defaults.AgentInstructions.MaxFileBytes
	}
	if cfg.AgentInstructions.MaxTotalBytes <= 0 {
		cfg.AgentInstructions.MaxTotalBytes = defaults.AgentInstructions.MaxTotalBytes
	}
}

func applyOutputAssistantUpdateDefaults(cfg *Config, defaults *Config) {
	if cfg.Output.AssistantUpdates != "" {
		switch strings.ToLower(cfg.Output.AssistantUpdates) {
		case "verbose", "phase", "off":
			cfg.Output.AssistantUpdates = strings.ToLower(cfg.Output.AssistantUpdates)
		default:
			cfg.Output.AssistantUpdates = defaults.Output.AssistantUpdates
		}
	}
}

// applyDefaults はデフォルト値と内部正規化を適用する。
// 内部設定の補正（tool_loop_limit 負値→0 など）もこのフェーズで行う。
func applyDefaults(cfg *Config, opts ...defaultApplyOptions) {
	defaults := DefaultConfig()
	options := resolveDefaultApplyOptions(opts)

	applyInternalSettingNormalization(cfg, defaults)
	applyProviderAndModelDefaults(cfg)
	applyNestedSectionDefaults(cfg, defaults, options)
	applyOutputAssistantUpdateDefaults(cfg, defaults)
}
