package config

import "strings"

// DefaultConfig はデフォルト設定
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "deepseek",
		DefaultModel:    "deepseek-chat",
		General: GeneralConfig{
			UILanguage:    "auto", // デフォルト: 自動判定（フォールバック: ja）
			ToolLoopLimit: 0,      // 内部既定: 0 = unlimited tool loop
		},
		ProviderModels: map[string]ProviderModelConfig{
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
		},
		Compression: CompressionConfig{
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
			ProviderThresholds: map[string]int{
				"gemini":             180000,
				"claude":             150000,
				"bedrock":            150000,
				"deepseek":           80000, // 128K window に対して出力/推論 headroom を残す安全側の値
				"openai":             100000,
				"openai:gpt-5.4":     260000, // 272K pricing cliff 手前
				"openai:gpt-5.4-pro": 260000,
				"openrouter":         120000,
			},
		},
		LoopDetection: LoopDetectionConfig{
			Threshold: 3,
		},
		APIRetry: APIRetryConfig{
			Count:        3,
			InitialDelay: 1,
			MaxDelay:     30,
			Timeout:      3600, // xhigh thinking 対応（1時間）
		},
		Diff: DiffConfig{
			ContextLines: 10,
		},
		Execution: ExecutionConfig{
			Mode:              string(ExecutionBalanced),
			AlwaysConfirm:     []string{},
			SafeShellCommands: []string{},
		},
		ToolConfirm: ToolConfirmConfig{
			AutoApproveSafe:   true,
			AutoApproveMedium: false,
		},
		CommandAliases: map[string]string{
			"c": "config",
			"u": "use",
		},
		PromptCache: PromptCacheConfig{
			Enabled:  true,
			CacheTTL: "5m",
		},
		Paste: PasteConfig{
			BracketedPaste: true, // デフォルトON - 複数行ペースト対応
			MaxLines:       10000,
			MaxBytes:       1048576,
		},
		Streaming: StreamingConfig{
			IdleTimeoutSeconds:     30,  // チャンク間隔タイムアウト（30秒）
			ThinkingTimeoutSeconds: 120, // thinking専用: text/FC が来なければタイムアウト（最大リトライ2回=360秒）
			ShowFileInfo:           true,
			ShowSearchProgress:     true,
			StreamBashOutput:       true,
		},
		Bash: BashConfig{
			SafetyLevel:     "permissive", // 確認出るので安全、利便性向上
			SafeCommands:    []string{},
			AllowRedirect:   true, // 利便性向上
			AllowInlineEdit: true, // 利便性向上
		},
		ListDir: ListDirConfig{
			AdditionalIgnoreDirs: []string{},
		},
		ProjectMap: ProjectMapConfig{
			Enabled:              true,
			ContextRatio:         ProjectMapContextRatioDefault,
			AdditionalIgnoreDirs: []string{},
		},

		GitStage: GitStageConfig{
			BatchConfirm: true,
		},
		LSP: LSPConfig{
			Enabled: true,
			Servers: map[string]LSPServerConfig{
				// ===== Existing (4 languages) =====
				"go": {
					Command: "gopls",
					Args:    []string{},
				},
				"typescript": {
					Command: "vtsls",
					Args:    []string{"--stdio"},
				},
				"python": {
					Command: "pyright-langserver",
					Args:    []string{"--stdio"},
				},
				"rust": {
					Command: "rust-analyzer",
					Args:    []string{},
				},
				// ===== Tier 1: Backend languages (11 languages) =====
				"java": {
					Command: "jdtls",
					Args:    []string{},
				},
				"c": {
					Command: "clangd",
					Args:    []string{},
				},
				"cpp": {
					Command: "clangd",
					Args:    []string{},
				},
				"ruby": {
					Command: "solargraph",
					Args:    []string{"stdio"},
				},
				"kotlin": {
					Command: "kotlin-language-server",
					Args:    []string{},
				},
				"swift": {
					Command: "sourcekit-lsp",
					Args:    []string{},
				},
				"csharp": {
					Command: "csharp-ls",
					Args:    []string{},
				},
				"scala": {
					Command: "metals",
					Args:    []string{},
				},
				"php": {
					Command: "intelephense",
					Args:    []string{"--stdio"},
				},
				"elixir": {
					Command: "elixir-ls",
					Args:    []string{},
				},
				"lua": {
					Command: "lua-language-server",
					Args:    []string{},
				},
				// ===== Tier 2: Frontend languages (4 languages) =====
				"css": {
					Command: "vscode-css-language-server",
					Args:    []string{"--stdio"},
				},
				"html": {
					Command: "vscode-html-language-server",
					Args:    []string{"--stdio"},
				},
				"vue": {
					Command: "vue-language-server",
					Args:    []string{"--stdio"},
				},
				"svelte": {
					Command: "svelteserver",
					Args:    []string{"--stdio"},
				},
				// ===== Tier 3: Config/Script languages (5 languages) =====
				"yaml": {
					Command: "yaml-language-server",
					Args:    []string{"--stdio"},
				},
				"toml": {
					Command: "taplo",
					Args:    []string{"lsp", "stdio"},
				},
				"sql": {
					Command: "sqls",
					Args:    []string{},
				},
				"bash": {
					Command: "bash-language-server",
					Args:    []string{"start"},
				},
				"markdown": {
					Command: "marksman",
					Args:    []string{"server"},
				},
			},
		},
		OpenAI: OpenAIConfig{
			ResponsesAPIModels: []string{}, // 内部: プレフィックスマッチのフォールバック（YAML 直接編集で追加可能）
		},
		Thinking: ThinkingConfig{
			Enabled: false,    // 内部 runtime 初期値（/think コマンドが正規ルート）
			Level:   "medium", // 内部 runtime 初期値（/think コマンドが正規ルート）
		},
		Output: OutputConfig{
			MaxLines:         5,  // デフォルト5行で折りたたみ
			AssistantUpdates: "", // 空 = Normal Mode は phase / Plan Mode は verbose
		},
		WebSearch: WebSearchConfig{
			CacheEnabled: true,
			CacheTTL:     3600, // 1時間
			CacheSize:    50,
		},
		SubAgent: SubAgentConfig{
			Enabled:       true,
			DefaultModel:  "gpt-5.4-mini",
			DefaultEffort: "",
			MaxConcurrent: 1,
		},
		MCP: MCPConfig{
			Enabled:  true,  // デフォルトON - MCP接続有効
			Headless: false, // デフォルトOFF - Headlessモードでは接続しない
		},
		FinalChecks: FinalChecksConfig{
			Commands: nil, // デフォルト: final checks なし
			Timeout:  600, // 10分タイムアウト
		},
		providerModelsStore: providerModelStore{
			state: providerModelSectionStateInMemoryEffectiveOnly,
		},
	}
}

// applyDefaults はデフォルト値と内部正規化を適用する。
// 内部設定の補正（tool_loop_limit 負値→0 など）もこのフェーズで行う。
func applyDefaults(cfg *Config, opts ...defaultApplyOptions) {
	defaults := DefaultConfig()
	options := defaultApplyOptions{
		lspSectionExists: true,
		lspServersExists: true,
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	// --- 内部設定の正規化 ---
	if cfg.General.ToolLoopLimit < 0 {
		cfg.General.ToolLoopLimit = 0 // 負値は 0（無制限）にサイレント補正
	}
	// Execution: mode が空なら balanced をデフォルト適用
	if cfg.Execution.Mode == "" {
		cfg.Execution.Mode = defaults.Execution.Mode
	}

	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "deepseek"
	}
	cfg.refreshEffectiveProviderModels()

	// ネストされた構造体のデフォルト値を適用（YAMLで省略された場合）
	if cfg.LoopDetection.Threshold == 0 {
		cfg.LoopDetection = defaults.LoopDetection
	}
	if cfg.APIRetry.Count == 0 {
		cfg.APIRetry = defaults.APIRetry
	}
	if cfg.APIRetry.Timeout == 0 {
		cfg.APIRetry.Timeout = defaults.APIRetry.Timeout
	}
	// Compression: ゼロ値のフィールドのみデフォルト適用（構造体全体の上書きはしない）
	if cfg.Compression.TriggerPercent == 0 {
		cfg.Compression.TriggerPercent = defaults.Compression.TriggerPercent
	}
	if cfg.Compression.KeepRecent == 0 {
		cfg.Compression.KeepRecent = defaults.Compression.KeepRecent
	}
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
	}
	if cfg.Streaming.IdleTimeoutSeconds == 0 {
		cfg.Streaming.IdleTimeoutSeconds = defaults.Streaming.IdleTimeoutSeconds
	}
	// LSP設定のデフォルト適用
	// lsp.servers が YAML に明示されている場合は nil/empty/non-empty をそのまま保持し、
	// sibling field の値を巻き戻さない。
	if !options.lspSectionExists {
		cfg.LSP = defaults.LSP
	} else if !options.lspServersExists && cfg.LSP.Servers == nil {
		cfg.LSP.Servers = defaults.LSP.Servers
	}
	// Note: Diff.ContextLines は0が有効値なので、デフォルト適用は行わない
	// Thinking: 内部 runtime 初期値（/think コマンドが正規ルート）
	if cfg.Thinking.Level == "" {
		cfg.Thinking.Level = defaults.Thinking.Level
	}
	// WebSearch: 全てゼロ値の場合のみデフォルト適用
	if !cfg.WebSearch.CacheEnabled && cfg.WebSearch.CacheTTL == 0 && cfg.WebSearch.CacheSize == 0 {
		cfg.WebSearch = defaults.WebSearch
	}
	// FinalChecks: Timeout が 0 の場合はデフォルト適用
	if cfg.FinalChecks.Timeout == 0 {
		cfg.FinalChecks.Timeout = defaults.FinalChecks.Timeout
	}
	if cfg.Output.AssistantUpdates != "" {
		switch strings.ToLower(cfg.Output.AssistantUpdates) {
		case "verbose", "phase", "off":
			cfg.Output.AssistantUpdates = strings.ToLower(cfg.Output.AssistantUpdates)
		default:
			cfg.Output.AssistantUpdates = defaults.Output.AssistantUpdates
		}
	}
}

type defaultApplyOptions struct {
	lspSectionExists bool
	lspServersExists bool
}
