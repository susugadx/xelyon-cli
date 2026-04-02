//go:build ignore

// config_sections.go は設定セクションの共通定義
//
// gen-config-example.go と gen-config-registry.go から参照される
package main

// SectionInfo はセクション情報
type SectionInfo struct {
	Title      string              // セクションタイトル
	Icon       string              // カテゴリアイコン（/config UI用）
	Comments   []string            // セクション説明コメント
	Fields     map[string]string   // フィールド名 -> 説明
	FieldTypes map[string]string   // フィールド名 -> 型 (bool/int/string/select/[]string/map/structmap)
	SelectOpts map[string][]string // フィールド名 -> 選択肢（select型用）
}

// Sections は全セクション定義
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
			"default_provider": {"deepseek", "claude", "openai", "gemini", "groq", "ollama", "openrouter", "bedrock"},
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
		Title: "一般設定",
		Icon:  "⚙️",
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
		Title: "会話履歴圧縮設定",
		Icon:  "📦",
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
	// loop_detection, api_retry, diff: 内部既定値として保持（user-facing config から削除済み）
	"execution": {
		Title: "実行モード設定",
		Icon:  "🛡️",
		Comments: []string{
			"ツール実行の承認モードを制御",
			"balanced: read自動/write確認/verification bash安全自動",
			"trusted: workspace内通常編集も自動/高影響のみ確認",
			"full_auto: 原則自動（always_confirm指定は確認）",
		},
		Fields: map[string]string{
			"mode":                "実行モード（balanced / trusted / full_auto）",
			"always_confirm":     "どのモードでも確認するカテゴリ",
			"safe_shell_commands": "追加の安全シェルコマンド（verification / env 用）",
		},
		FieldTypes: map[string]string{
			"mode":                "select",
			"always_confirm":     "[]string",
			"safe_shell_commands": "[]string",
		},
		SelectOpts: map[string][]string{
			"mode": {"balanced", "trusted", "full_auto"},
		},
	},
	// tool_confirm: execution に統合済み（YAML 互換は維持）
	// bash: execution.safe_shell_commands + 内部ポリシーに統合
	// command_aliases: user-facing config から削除済み（YAML 互換は維持）
	// prompt_cache: user-facing config から削除済み（YAML advanced 扱い）
	// streaming: user-facing config から削除済み（YAML advanced 扱い）
	"paste": {
		Title: "ペーストモード設定",
		Icon:  "📋",
		Fields: map[string]string{
			"bracketed_paste": "Bracketed Paste Mode を有効化（複数行ペースト対応）",
		},
		FieldTypes: map[string]string{
			"bracketed_paste": "bool",
		},
	},
	// bash: user-facing config から削除済み（execution + 内部ポリシーに統合）
	// YAML 互換は維持。allow_pipe は未使用のため非公開化。
	"project_map": {
		Title: "プロジェクト構造マップ設定",
		Icon:  "🗺️",
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

	// git_stage: batch_confirm は標準挙動としてハードコード（user-facing config から削除済み）
	// plan_mode: 内部ガードのみ保持（user-facing config から削除済み）
	"lsp": {
		Title: "LSP連携設定",
		Icon:  "🔧",
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
	// openai: user-facing config から削除済み（内部ルーティングで使用、YAML 互換は維持）
	// thinking: user-facing config から削除済み（/think コマンドが正規ルート、YAML 互換は維持）
	"output": {
		Title: "ツール出力表示設定",
		Icon:  "📤",
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
		Title: "Web検索設定",
		Icon:  "🔍",
		Comments: []string{
			"ネイティブ Web 検索の実行プロバイダーとキャッシュ設定",
			"未設定の場合はメインプロバイダーの検索を使用",
			"メインが非対応の場合は openai / gemini / claude のいずれかを設定",
		},
		Fields: map[string]string{
			"provider":      "検索プロバイダー（openai / gemini / claude、未設定時はメインプロバイダーを使用）",
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
			"provider": {"openai", "gemini", "claude"},
		},
	},
	"sub_agent": {
		Title: "サブエージェント設定",
		Icon:  "🚀",
		Comments: []string{
			"探索・調査タスクを低コストモデルへ委譲する設定",
			"spawn_agent / wait_agent の既定値と同時実行数を制御",
		},
		Fields: map[string]string{
			"enabled":        "サブエージェント機能を有効化",
			"default_model":  "既定モデル（空でメイン provider の最安モデルを自動選択）",
			"default_effort": "既定推論強度（off / low / medium / high）",
			"max_concurrent": "同時実行上限（デフォルト: 5）",
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
		Title: "MCP設定",
		Icon:  "🔌",
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
	"hooks": {
		Title: "フック設定",
		Icon:  "🏁",
		Comments: []string{
			"タスク完了時に自動実行するシェルコマンド（LSPチェック後）",
			"変更ファイルは XELYON_CHANGED_FILES 環境変数で参照可能",
		},
		Fields: map[string]string{
			"on_completion":   "完了時に実行するコマンド（例: go test ./...）",
			"on_step_complete": "ステップ完了時に実行するコマンド（Plan Mode用）",
			"timeout":         "コマンドタイムアウト（秒）（デフォルト: 60）",
			"max_retry":       "フック失敗時の最大リトライ回数（デフォルト: 3）",
		},
		FieldTypes: map[string]string{
			"on_completion":   "[]string",
			"on_step_complete": "[]string",
			"timeout":         "int",
			"max_retry":       "int",
		},
	},
}

// SectionOrder はセクションの表示順序
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
	"hooks",
}

// CategoryOrder はカテゴリの表示順序（UIでのグループ化用）
// default_provider と provider_models は同じ "provider" カテゴリにまとめる
var CategoryOrder = []string{
	"provider", // default_provider, provider_models
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
	"hooks",
}

// SectionToCategory はセクション名をカテゴリ名にマップ
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
	"hooks":            "hooks",
}

// CategoryInfo はカテゴリの表示情報
type CategoryInfo struct {
	DisplayName string
	Icon        string
	Sections    []string // このカテゴリに含まれるセクション
}

// Categories はカテゴリ情報
var Categories = map[string]CategoryInfo{
	"provider": {
		DisplayName: "Provider & Model",
		Icon:        "🤖",
		Sections:    []string{"default_provider", "default_model", "provider_models"},
	},
	"general": {
		DisplayName: "General",
		Icon:        "⚙️",
		Sections:    []string{"general"},
	},
	"compression": {
		DisplayName: "Compression",
		Icon:        "📦",
		Sections:    []string{"compression"},
	},
	"execution": {
		DisplayName: "Execution Mode",
		Icon:        "🛡️",
		Sections:    []string{"execution"},
	},
	"paste": {
		DisplayName: "Paste Mode",
		Icon:        "📋",
		Sections:    []string{"paste"},
	},
	"project_map": {
		DisplayName: "Project Map",
		Icon:        "🗺️",
		Sections:    []string{"project_map"},
	},
	"lsp": {
		DisplayName: "LSP Servers",
		Icon:        "🔧",
		Sections:    []string{"lsp"},
	},
	"output": {
		DisplayName: "Output",
		Icon:        "📤",
		Sections:    []string{"output"},
	},
	"web_search": {
		DisplayName: "Web Search",
		Icon:        "🔍",
		Sections:    []string{"web_search"},
	},
	"sub_agent": {
		DisplayName: "Sub-agent",
		Icon:        "🚀",
		Sections:    []string{"sub_agent"},
	},
	"mcp": {
		DisplayName: "MCP Servers",
		Icon:        "🔌",
		Sections:    []string{"mcp"},
	},
	"hooks": {
		DisplayName: "Hooks",
		Icon:        "🏁",
		Sections:    []string{"hooks"},
	},
}
