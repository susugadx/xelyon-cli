package configgen

import "github.com/susugadx/xelyon-cli/internal/llmcatalog"

// SectionInfo describes a user-facing config section and its fields.
type SectionInfo struct {
	StructName string
	Title      string
	Icon       string
	Comments   []string
	Fields     map[string]string
	FieldTypes map[string]string
	SelectOpts map[string][]string
}

// CategoryInfo describes a config category shown in the UI and docs.
type CategoryInfo struct {
	DisplayName string
	Icon        string
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
	"lsp": {
		StructName: "LSPConfig",
		Title:      "LSP連携設定",
		Icon:       "🔧",
		Comments: []string{
			"23言語のデフォルト設定が内蔵済み。通常は変更不要です。",
			"詳細: docs/config.md の「LSP連携設定」セクション",
		},
		Fields: map[string]string{
			"enabled":             "LSP連携の有効/無効",
			"skip_install_prompt": "インストール提案をスキップ",
			"servers":             "LSPサーバー個別設定（コマンド・引数・有効無効）",
		},
		FieldTypes: map[string]string{
			"enabled":             "bool",
			"skip_install_prompt": "bool",
			"servers":             "structmap",
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
			"メインが非対応の場合は openai / gemini / claude / anthropic のいずれかを設定",
		},
		Fields: map[string]string{
			"provider":      "検索プロバイダー（openai / gemini / claude / anthropic、未設定時はメインプロバイダーを使用）",
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

// SectionOrder is the display order for user-facing sections.
var SectionOrder = []string{
	"default_provider",
	"default_model",
	"provider_models",
	"general",
	"execution",
	"compression",
	"paste",
	"project_map",
	"lsp",
	"output",
	"web_search",
	"sub_agent",
	"mcp",
	"final_checks",
}

// CategoryOrder is the UI grouping order.
var CategoryOrder = []string{
	"provider",
	"general",
	"execution",
	"compression",
	"paste",
	"project_map",
	"lsp",
	"output",
	"web_search",
	"sub_agent",
	"mcp",
	"final_checks",
}

// SectionToCategory maps sections to UI categories.
var SectionToCategory = map[string]string{
	"default_provider": "provider",
	"default_model":    "provider",
	"provider_models":  "provider",
	"general":          "general",
	"execution":        "execution",
	"compression":      "compression",
	"paste":            "paste",
	"project_map":      "project_map",
	"lsp":              "lsp",
	"output":           "output",
	"web_search":       "web_search",
	"sub_agent":        "sub_agent",
	"mcp":              "mcp",
	"final_checks":     "final_checks",
}

// Categories contains category display metadata.
var Categories = map[string]CategoryInfo{
	"provider": {
		DisplayName: "Provider & Model",
		Icon:        "🤖",
	},
	"general": {
		DisplayName: "General",
		Icon:        "⚙️",
	},
	"compression": {
		DisplayName: "Compression",
		Icon:        "📦",
	},
	"execution": {
		DisplayName: "Execution Mode",
		Icon:        "🛡️",
	},
	"paste": {
		DisplayName: "Paste Mode",
		Icon:        "📋",
	},
	"project_map": {
		DisplayName: "Project Map",
		Icon:        "🗺️",
	},
	"lsp": {
		DisplayName: "LSP Servers",
		Icon:        "🔧",
	},
	"output": {
		DisplayName: "Output",
		Icon:        "📤",
	},
	"web_search": {
		DisplayName: "Web Search",
		Icon:        "🔍",
	},
	"sub_agent": {
		DisplayName: "Sub-agent",
		Icon:        "🚀",
	},
	"mcp": {
		DisplayName: "MCP Servers",
		Icon:        "🔌",
	},
	"final_checks": {
		DisplayName: "Final Checks",
		Icon:        "🧪",
	},
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
