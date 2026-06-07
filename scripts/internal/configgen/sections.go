package configgen

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// SectionInfo describes a user-facing config section and its fields.
type SectionInfo struct {
	StructName string
	Title      string
	Icon       string
	Comments   []string
	Fields     map[string]string
	FieldTypes map[string]string
	SelectOpts map[string][]string
	Example    ExampleSectionPolicy
}

// CategoryInfo describes a config category shown in the UI and docs.
type CategoryInfo struct {
	DisplayName string
	Icon        string
}

// ExampleFilterMode は example 生成時に section 配下をどう扱うかの方針。
type ExampleFilterMode string

const (
	ExampleFilterModeFields  ExampleFilterMode = "fields"
	ExampleFilterModeKeepAll ExampleFilterMode = "keep_all"
)

// ExampleSectionPolicy は config.yaml.example 生成時の section 別ポリシー。
type ExampleSectionPolicy struct {
	FilterMode      ExampleFilterMode
	OmittedFields   map[string]bool
	Overrides       map[string]any
	CommentedFields map[string]CommentedExampleField
}

// CommentedExampleField は空 default などで YAML から省略される optional field を
// example 上ではコメント付きの記入例として表示する。
type CommentedExampleField struct {
	Value  string
	Before string
}

// Sections is the canonical section metadata shared by config generators.
var Sections = map[string]SectionInfo{
	"default_provider": {
		Title: "プロバイダー設定",
		Icon:  "🤖",
		Fields: map[string]string{
			"default_provider": "デフォルトで使用するLLMプロバイダー",
		},
		FieldTypes: map[string]string{
			"default_provider": "select",
		},
		SelectOpts: map[string][]string{
			"default_provider": llmcatalog.DisplayProviderKeys(),
		},
	},
	"default_model": {
		Icon: "🤖",
		Fields: map[string]string{
			"default_model": "デフォルトで使用するモデル",
		},
		FieldTypes: map[string]string{
			"default_model": "string",
		},
	},
	"provider_models": {
		Icon: "🤖",
		Fields: map[string]string{
			"provider_models": "プロバイダーごとのモデル設定",
		},
		FieldTypes: map[string]string{
			"provider_models": "structmap",
		},
		Example: ExampleSectionPolicy{
			FilterMode: ExampleFilterModeKeepAll,
		},
	},
	"review": {
		StructName: "ReviewConfig",
		Title:      "レビュー設定",
		Icon:       "🔎",
		Comments: []string{
			"/review 専用モデル設定",
			"未設定の場合は現在の provider/model を使用",
		},
		Fields: map[string]string{
			"provider":                        "/review 専用プロバイダー（空で現在の provider/model を使用）",
			"model":                           "/review 専用モデル（provider 設定時のみ有効。空で provider の既定モデル）",
			"web_search_evidence.enabled":     "/review の外部 Web 検索 evidence を有効化（初期検索 + Pass1 後の追加検索。raw result は discovery-only、fetch 済み external_doc snippet は citation-capable。デフォルト: false。XELYON_REVIEW_WEB_SEARCH=1 でも有効化）",
			"web_search_evidence.max_queries": "外部 Web 検索 evidence の最大クエリ数（初期検索 + Pass1 後追加検索の合計。デフォルト: 3）",
			"web_search_evidence.max_results_per_query": "外部 Web 検索 evidence の 1 クエリあたり最大結果数（デフォルト: 3）",
		},
		FieldTypes: map[string]string{
			"provider":                        "select",
			"model":                           "string",
			"web_search_evidence.enabled":     "bool",
			"web_search_evidence.max_queries": "int",
			"web_search_evidence.max_results_per_query": "int",
		},
		SelectOpts: map[string][]string{
			"provider": reviewProviderSelectOptions(),
		},
	},
	"gemini": {
		StructName: "GeminiConfig",
		Title:      "Gemini 設定",
		Icon:       "🤖",
		Comments: []string{
			"Gemini BYOK の同期 inference tier を制御",
			"standard は既定、flex は低コスト・高遅延、priority は高コスト・高優先度",
		},
		Fields: map[string]string{
			"service_tier": "Gemini service_tier（standard / flex / priority）",
		},
		FieldTypes: map[string]string{
			"service_tier": "select",
		},
		SelectOpts: map[string][]string{
			"service_tier": config.GeminiServiceTierValues(),
		},
	},
	"general": {
		StructName: "GeneralConfig",
		Title:      "一般設定",
		Icon:       "⚙️",
		Fields: map[string]string{
			"ui_language": "表示言語（auto, ja, en）",
		},
		FieldTypes: map[string]string{
			"ui_language": "select",
		},
		SelectOpts: map[string][]string{
			"ui_language": {"auto", "ja", "en"},
		},
	},
	"compression": {
		StructName: "CompressionConfig",
		Title:      "会話履歴圧縮設定",
		Icon:       "📦",
		Fields: map[string]string{
			"enabled":         "自動圧縮を有効化",
			"trigger_percent": "自動圧縮のトークン使用率閾値（%）",
			"keep_recent":     "圧縮時に保持する直近メッセージ数",
		},
		FieldTypes: map[string]string{
			"enabled":         "bool",
			"trigger_percent": "int",
			"keep_recent":     "int",
		},
	},
	"provider_history_reduction": {
		StructName: "ProviderHistoryReductionConfig",
		Title:      "Provider History Reduction",
		Icon:       "📉",
		Comments: []string{
			"古い tool result / command output / edit payload を provider-facing projection 上だけで軽量化",
			"raw history / session / audit / persisted JSONL は保持",
		},
		Fields: map[string]string{
			"mode":                                                  "provider-facing history reduction mode（off / dry_run / apply。env: XELYON_PROVIDER_HISTORY_REDUCTION）",
			"rehydrate_context":                                     "省略した古い evidence を必要時に request-local active context として戻す（env: XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT）",
			"raw_output_artifacts.mode":                             "data-bearing raw output artifact-backed compact mode（off / dry_run / apply）",
			"raw_output_artifacts.root":                             "raw output artifact store root path（env: XELYON_RAW_OUTPUT_ARTIFACT_ROOT）",
			"raw_output_artifacts.max_artifact_bytes":               "raw output artifact 1件あたりの最大保存 byte 数",
			"raw_output_artifacts.session_quota_bytes":              "session 単位の raw output artifact 保存上限 byte 数",
			"raw_output_artifacts.chunk_bytes":                      "raw output artifact 書き込み chunk byte 数",
			"raw_output_artifacts.active_context_budget_tokens":     "request-local raw output active context の既定 token budget",
			"raw_output_artifacts.active_context_budget_max_tokens": "request-local raw output active context の最大 token budget",
			"raw_output_artifacts.retention":                        "raw output artifact retention policy（現状 session のみ）",
		},
		FieldTypes: map[string]string{
			"mode":                                                  "select",
			"rehydrate_context":                                     "bool",
			"raw_output_artifacts.mode":                             "select",
			"raw_output_artifacts.root":                             "string",
			"raw_output_artifacts.max_artifact_bytes":               "int",
			"raw_output_artifacts.session_quota_bytes":              "int",
			"raw_output_artifacts.chunk_bytes":                      "int",
			"raw_output_artifacts.active_context_budget_tokens":     "int",
			"raw_output_artifacts.active_context_budget_max_tokens": "int",
			"raw_output_artifacts.retention":                        "select",
		},
		SelectOpts: map[string][]string{
			"mode":                           config.ProviderHistoryReductionModeValues(),
			"raw_output_artifacts.mode":      config.ProviderHistoryRawOutputArtifactsModeValues(),
			"raw_output_artifacts.retention": {string(config.ProviderHistoryRawOutputArtifactsRetentionSession)},
		},
		Example: ExampleSectionPolicy{
			CommentedFields: map[string]CommentedExampleField{
				"raw_output_artifacts.root": {
					Value:  "/absolute/path/to/rawoutputs",
					Before: "raw_output_artifacts.max_artifact_bytes",
				},
			},
		},
	},
	"execution": {
		StructName: "ExecutionConfig",
		Title:      "実行モード設定",
		Icon:       "🛡️",
		Comments: []string{
			"ツール実行の承認モードを制御",
			"balanced: read自動/write確認/verification bash安全自動",
			"trusted: workspace内通常編集も自動/高影響のみ確認",
			"full_auto: 原則自動（always_confirm指定は確認）",
		},
		Fields: map[string]string{
			"mode":                "実行モード（balanced / trusted / full_auto）",
			"always_confirm":      "どのモードでも確認するカテゴリ",
			"safe_shell_commands": "追加の安全シェルコマンド（verification / env 用）",
		},
		FieldTypes: map[string]string{
			"mode":                "select",
			"always_confirm":      "[]string",
			"safe_shell_commands": "[]string",
		},
		SelectOpts: map[string][]string{
			"mode": {"balanced", "trusted", "full_auto"},
		},
	},
	"paste": {
		StructName: "PasteConfig",
		Title:      "ペーストモード設定",
		Icon:       "📋",
		Fields: map[string]string{
			"bracketed_paste": "Bracketed Paste Mode を有効化（複数行ペースト対応）",
		},
		FieldTypes: map[string]string{
			"bracketed_paste": "bool",
		},
	},
	"project_map": {
		StructName: "ProjectMapConfig",
		Title:      "プロジェクト構造マップ設定",
		Icon:       "🗺️",
		Fields: map[string]string{
			"enabled":                "セッション開始時にプロジェクト構造マップを生成・注入",
			"context_ratio":          "ProjectMap のベース比率（0.01-0.20、デフォルト: 0.05。大規模 repo では 0.03-0.04 に自動補正）",
			"additional_ignore_dirs": "追加除外ディレクトリ（list_dir と共通）",
		},
		FieldTypes: map[string]string{
			"enabled":                "bool",
			"context_ratio":          "float",
			"additional_ignore_dirs": "[]string",
		},
	},
	"agent_instructions": {
		StructName: "AgentInstructionsConfig",
		Title:      "Agent Instructions 設定",
		Icon:       "📚",
		Comments: []string{
			"AGENTS.md / CLAUDE.md 互換ガイダンス読み込み設定",
			"xelyon.yaml の rules とは別レイヤーで扱われます",
		},
		Fields: map[string]string{
			"project.mode":               "project-local guidance の読み込みモード（off / fallback / always）",
			"project.files":              "project-local guidance ファイル候補",
			"project.include_gitignored": "gitignored / untracked guidance を許可",
			"global.enabled":             "global guidance 読み込みを有効化",
			"global.files":               "global guidance ファイル候補",
			"include_local_files":        "CLAUDE.local.md / AGENTS.local.md など local 系 guidance を許可",
			"expand_imports":             "@path import 行を展開して読み込む（相対パスは当該 guidance file 基準）",
			"max_file_bytes":             "1ファイルあたりの最大読み込みバイト数",
			"max_total_bytes":            "guidance 全体の最大読み込みバイト数",
		},
		FieldTypes: map[string]string{
			"project.mode":               "select",
			"project.files":              "[]string",
			"project.include_gitignored": "bool",
			"global.enabled":             "bool",
			"global.files":               "[]string",
			"include_local_files":        "bool",
			"expand_imports":             "bool",
			"max_file_bytes":             "int",
			"max_total_bytes":            "int",
		},
		SelectOpts: map[string][]string{
			"project.mode": {"off", "fallback", "always"},
		},
	},
	"lsp": {
		StructName: "LSPConfig",
		Title:      "LSP連携設定",
		Icon:       "🔧",
		Comments: []string{
			"23言語のデフォルト設定が内蔵済み。通常は変更不要です。",
			"詳細: docs/config.md の「LSP連携設定」セクション",
		},
		Fields: map[string]string{
			"enabled":             "LSP連携の有効/無効（有効時は検出言語の導入済みサーバーを起動準備）",
			"skip_install_prompt": "インストール提案をスキップ",
			"servers":             "LSPサーバー個別設定（コマンド・引数・有効無効）",
		},
		FieldTypes: map[string]string{
			"enabled":             "bool",
			"skip_install_prompt": "bool",
			"servers":             "structmap",
		},
		Example: ExampleSectionPolicy{
			OmittedFields: map[string]bool{
				"servers": true,
			},
			Overrides: map[string]any{
				"servers": nil,
			},
		},
	},
	"output": {
		StructName: "OutputConfig",
		Title:      "ツール出力表示設定",
		Icon:       "📤",
		Fields: map[string]string{
			"assistant_updates": "assistant prose の中間表示制御（verbose / phase / off、空でモード別デフォルト）",
			"max_lines":         "折りたたみ前の最大表示行数",
		},
		FieldTypes: map[string]string{
			"assistant_updates": "select",
			"max_lines":         "int",
		},
		SelectOpts: map[string][]string{
			"assistant_updates": {"", "verbose", "phase", "off"},
		},
	},
	"web_search": {
		StructName: "WebSearchConfig",
		Title:      "Web検索設定",
		Icon:       "🔍",
		Comments: []string{
			"ネイティブ Web 検索の実行プロバイダーとキャッシュ設定",
			"未設定の場合はメインプロバイダーの検索を使用",
			"メインが非対応の場合は kimi / moonshot / openai / gemini / claude / anthropic のいずれかを設定",
		},
		Fields: map[string]string{
			"provider":      "検索プロバイダー（kimi / moonshot / openai / gemini / claude / anthropic、未設定時はメインプロバイダーを使用）",
			"cache_enabled": "キャッシュを有効化（デフォルト: true）",
			"cache_ttl":     "キャッシュTTL秒数（デフォルト: 3600 = 1時間）",
			"cache_size":    "最大キャッシュ数（デフォルト: 50）",
		},
		FieldTypes: map[string]string{
			"provider":      "select",
			"cache_enabled": "bool",
			"cache_ttl":     "int",
			"cache_size":    "int",
		},
		SelectOpts: map[string][]string{
			"provider": llmcatalog.NativeWebSearchProviderKeys(true),
		},
		Example: ExampleSectionPolicy{
			Overrides: map[string]any{
				"provider": "gemini",
			},
		},
	},
	"sub_agent": {
		StructName: "SubAgentConfig",
		Title:      "サブエージェント設定",
		Icon:       "🚀",
		Comments: []string{
			"探索・調査タスクを低コストモデルへ委譲する設定",
			"spawn_agent / wait_agent の既定値と同時実行数を制御",
		},
		Fields: map[string]string{
			"enabled":        "サブエージェント機能を有効化",
			"default_model":  "既定モデル（空でメイン provider の最安モデルを自動選択）",
			"default_effort": "既定推論強度（off / low / medium / high）",
			"max_concurrent": "同時実行上限（デフォルト: 1）",
		},
		FieldTypes: map[string]string{
			"enabled":        "bool",
			"default_model":  "string",
			"default_effort": "select",
			"max_concurrent": "int",
		},
		SelectOpts: map[string][]string{
			"default_effort": {"off", "low", "medium", "high"},
		},
	},
	"mcp": {
		StructName: "MCPConfig",
		Title:      "MCP設定",
		Icon:       "🔌",
		Comments: []string{
			"MCP (Model Context Protocol) サーバー接続の設定",
			"個別サーバー設定は ~/.xelyon/mcp.json で管理",
		},
		Fields: map[string]string{
			"enabled":  "MCP接続を有効化（デフォルト: true）",
			"headless": "Headlessモードでも接続（デフォルト: false）",
		},
		FieldTypes: map[string]string{
			"enabled":  "bool",
			"headless": "bool",
		},
	},
	"final_checks": {
		StructName: "FinalChecksConfig",
		Title:      "Final Checks 設定",
		Icon:       "🧪",
		Comments: []string{
			"completed_with_changes 時に自動実行する final checks コマンド",
			"変更ファイルは XELYON_CHANGED_FILES 環境変数で参照可能",
		},
		Fields: map[string]string{
			"commands": "completed_with_changes 時に実行する final checks コマンド（例: go test ./...）",
			"timeout":  "final checks コマンドタイムアウト（秒）（デフォルト: 600）",
		},
		FieldTypes: map[string]string{
			"commands": "[]string",
			"timeout":  "int",
		},
	},
}

type sectionCatalogEntry struct {
	Name     string
	Category string
}

var sectionCatalog = []sectionCatalogEntry{
	{Name: "default_provider", Category: "provider"},
	{Name: "default_model", Category: "provider"},
	{Name: "provider_models", Category: "provider"},
	{Name: "gemini", Category: "provider"},
	{Name: "review", Category: "review"},
	{Name: "general", Category: "general"},
	{Name: "execution", Category: "execution"},
	{Name: "compression", Category: "compression"},
	{Name: "provider_history_reduction", Category: "provider_history_reduction"},
	{Name: "paste", Category: "paste"},
	{Name: "project_map", Category: "project_map"},
	{Name: "agent_instructions", Category: "agent_instructions"},
	{Name: "lsp", Category: "lsp"},
	{Name: "output", Category: "output"},
	{Name: "web_search", Category: "web_search"},
	{Name: "sub_agent", Category: "sub_agent"},
	{Name: "mcp", Category: "mcp"},
	{Name: "final_checks", Category: "final_checks"},
}

// SectionOrder is the display order for user-facing sections.
var SectionOrder = buildSectionOrder(sectionCatalog)

// SectionToCategory maps sections to UI categories.
var SectionToCategory = buildSectionCategoryMap(sectionCatalog)

type categoryCatalogEntry struct {
	Name string
	Info CategoryInfo
}

var categoryCatalog = []categoryCatalogEntry{
	{Name: "provider", Info: CategoryInfo{DisplayName: "Provider & Model", Icon: "🤖"}},
	{Name: "review", Info: CategoryInfo{DisplayName: "Review", Icon: "🔎"}},
	{Name: "general", Info: CategoryInfo{DisplayName: "General", Icon: "⚙️"}},
	{Name: "execution", Info: CategoryInfo{DisplayName: "Execution Mode", Icon: "🛡️"}},
	{Name: "compression", Info: CategoryInfo{DisplayName: "Compression", Icon: "📦"}},
	{Name: "provider_history_reduction", Info: CategoryInfo{DisplayName: "Provider History Reduction", Icon: "📉"}},
	{Name: "paste", Info: CategoryInfo{DisplayName: "Paste Mode", Icon: "📋"}},
	{Name: "project_map", Info: CategoryInfo{DisplayName: "Project Map", Icon: "🗺️"}},
	{Name: "agent_instructions", Info: CategoryInfo{DisplayName: "Agent Instructions", Icon: "📚"}},
	{Name: "lsp", Info: CategoryInfo{DisplayName: "LSP Servers", Icon: "🔧"}},
	{Name: "output", Info: CategoryInfo{DisplayName: "Output", Icon: "📤"}},
	{Name: "web_search", Info: CategoryInfo{DisplayName: "Web Search", Icon: "🔍"}},
	{Name: "sub_agent", Info: CategoryInfo{DisplayName: "Sub-agent", Icon: "🚀"}},
	{Name: "mcp", Info: CategoryInfo{DisplayName: "MCP Servers", Icon: "🔌"}},
	{Name: "final_checks", Info: CategoryInfo{DisplayName: "Final Checks", Icon: "🧪"}},
}

// CategoryOrder is the UI grouping order.
var CategoryOrder = buildCategoryOrder(categoryCatalog)

// Categories contains category display metadata.
var Categories = buildCategoryMap(categoryCatalog)

func buildSectionOrder(catalog []sectionCatalogEntry) []string {
	order := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		order = append(order, entry.Name)
	}
	return order
}

func buildSectionCategoryMap(catalog []sectionCatalogEntry) map[string]string {
	mapping := make(map[string]string, len(catalog))
	for _, entry := range catalog {
		mapping[entry.Name] = entry.Category
	}
	return mapping
}

func buildCategoryOrder(catalog []categoryCatalogEntry) []string {
	order := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		order = append(order, entry.Name)
	}
	return order
}

func buildCategoryMap(catalog []categoryCatalogEntry) map[string]CategoryInfo {
	categories := make(map[string]CategoryInfo, len(catalog))
	for _, entry := range catalog {
		categories[entry.Name] = entry.Info
	}
	return categories
}

func reviewProviderSelectOptions() []string {
	return append([]string{""}, llmcatalog.DisplayProviderKeys()...)
}

// OrderedSectionsForCategory returns the ordered sections that belong to a category.
func OrderedSectionsForCategory(category string) []string {
	var sections []string
	for _, sectionName := range SectionOrder {
		if SectionToCategory[sectionName] == category {
			sections = append(sections, sectionName)
		}
	}
	return sections
}
