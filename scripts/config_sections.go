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
			"language":        "表示言語（ja, en）",
			"tool_loop_limit": "ツールループ最大回数（0で無制限）",
		},
		FieldTypes: map[string]string{
			"language":        "select",
			"tool_loop_limit": "int",
		},
		SelectOpts: map[string][]string{
			"language": {"ja", "en"},
		},
	},
	"compression": {
		Title: "会話履歴圧縮設定",
		Icon:  "📦",
		Fields: map[string]string{
			"auto_compress":           "トークン使用率が閾値を超えたら自動圧縮",
			"threshold_tokens":        "トークン数の閾値（0 = 使用しない）",
			"threshold_percent":       "トークン使用率の閾値（%）",
			"token_threshold":         "絶対トークン閾値（デフォルト100000、キャッシュ有無に関係なく発動）",
			"model":                   "圧縮用モデル（空 = プロバイダー別デフォルト、main = メインモデル）",
			"keep_recent":             "圧縮時に保持する直近メッセージ数",
			"prefer_compact_api":      "プロバイダーのCompact APIを優先使用",
			"claude_compaction":       "Claude系の compact_20260112 を有効化",
			"compaction_trigger":      "compact のトリガー閾値（デフォルト: 150000）",
			"clear_tool_uses":         "Claude系の server-side tool clearing を有効化",
			"clear_tool_uses_trigger": "clear_tool_uses のトリガー閾値（デフォルト: 80000）",
			"clear_tool_inputs":       "tool_use 側の入力もクリアする",
		},
		FieldTypes: map[string]string{
			"auto_compress":           "bool",
			"threshold_tokens":        "int",
			"threshold_percent":       "int",
			"token_threshold":         "int",
			"model":                   "string",
			"keep_recent":             "int",
			"prefer_compact_api":      "bool",
			"claude_compaction":       "bool",
			"compaction_trigger":      "int",
			"clear_tool_uses":         "bool",
			"clear_tool_uses_trigger": "int",
			"clear_tool_inputs":       "bool",
		},
	},
	"loop_detection": {
		Title: "ループ検知設定",
		Icon:  "🔄",
		Fields: map[string]string{
			"threshold": "同じツール呼び出しの繰り返し回数でループと判定",
		},
		FieldTypes: map[string]string{
			"threshold": "int",
		},
	},
	"api_retry": {
		Title: "APIリトライ設定",
		Icon:  "🌐",
		Fields: map[string]string{
			"count":         "リトライ回数",
			"initial_delay": "初回リトライ待機時間（秒）",
			"max_delay":     "最大待機時間（秒）",
			"timeout":       "タイムアウト（秒）",
		},
		FieldTypes: map[string]string{
			"count":         "int",
			"initial_delay": "int",
			"max_delay":     "int",
			"timeout":       "int",
		},
	},
	"diff": {
		Title: "差分表示設定",
		Icon:  "📝",
		Fields: map[string]string{
			"context_lines":   "差分表示時のコンテキスト行数",
			"max_total_lines": "差分表示の最大行数（0で無制限）",
		},
		FieldTypes: map[string]string{
			"context_lines":   "int",
			"max_total_lines": "int",
		},
	},
	"tool_confirm": {
		Title: "ツール確認設定",
		Icon:  "✅",
		Fields: map[string]string{
			"auto_approve_safe":   "安全なツール（read_file等）を自動承認",
			"auto_approve_medium": "中程度のツール（write_file等）を自動承認",
		},
		FieldTypes: map[string]string{
			"auto_approve_safe":   "bool",
			"auto_approve_medium": "bool",
		},
	},
	"command_aliases": {
		Title: "コマンドエイリアス設定",
		Icon:  "🔗",
		Comments: []string{
			"スラッシュコマンドの短縮名を定義",
			"例: c → /compress, u → /use, h → /history",
		},
		FieldTypes: map[string]string{
			"command_aliases": "map",
		},
	},
	"prompt_cache": {
		Title: "プロンプトキャッシュ設定（Anthropic API cache_control）",
		Icon:  "💨",
		Fields: map[string]string{
			"enabled": "有効化（cache_control ブレークポイントを設定）",
		},
		FieldTypes: map[string]string{
			"enabled": "bool",
		},
	},
	"paste": {
		Title: "ペーストモード設定",
		Icon:  "📋",
		Fields: map[string]string{
			"bracketed_paste": "Bracketed Paste Mode を有効化（複数行ペースト対応）",
			"max_lines":       "最大行数",
			"max_bytes":       "最大バイト数",
			"timeout_seconds": "タイムアウト（秒）",
		},
		FieldTypes: map[string]string{
			"bracketed_paste": "bool",
			"max_lines":       "int",
			"max_bytes":       "int",
			"timeout_seconds": "int",
		},
	},
	"streaming": {
		Title: "ストリーミング設定",
		Icon:  "📺",
		Fields: map[string]string{
			"idle_timeout_seconds": "アイドルタイムアウト（秒）",
			"show_file_info":       "ファイル読み込み時にサイズ・行数を表示",
			"show_search_progress": "検索時に進捗を表示",
			"stream_bash_output":   "bashコマンドの出力をリアルタイム表示",
		},
		FieldTypes: map[string]string{
			"idle_timeout_seconds": "int",
			"show_file_info":       "bool",
			"show_search_progress": "bool",
			"stream_bash_output":   "bool",
		},
	},
	"bash": {
		Title: "bashツール設定",
		Icon:  "💻",
		Fields: map[string]string{
			"safety_level":      "安全レベル: strict / moderate / permissive",
			"safe_commands":     "追加の安全コマンド（例: - \"npm run\"）",
			"allow_pipe":        "パイプを許可",
			"allow_redirect":    "リダイレクトを許可",
			"allow_inline_edit": "インライン編集を許可（sed -i 等）",
		},
		FieldTypes: map[string]string{
			"safety_level":      "select",
			"safe_commands":     "[]string",
			"allow_pipe":        "bool",
			"allow_redirect":    "bool",
			"allow_inline_edit": "bool",
		},
		SelectOpts: map[string][]string{
			"safety_level": {"strict", "moderate", "permissive"},
		},
	},
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

	"git_stage": {
		Title: "git_add設定",
		Icon:  "📂",
		Fields: map[string]string{
			"batch_confirm": "複数ファイルをまとめて確認",
		},
		FieldTypes: map[string]string{
			"batch_confirm": "bool",
		},
	},
	"plan_mode": {
		Title: "Plan Mode設定",
		Icon:  "📋",
		Fields: map[string]string{
			"max_retry":                 "最大リトライ回数",
			"step_timeout":              "ステップタイムアウト（秒）",
			"clear_context_on_approval": "Plan 承認後に調査フェーズの履歴をクリアして実装を開始",
		},
		FieldTypes: map[string]string{
			"max_retry":                 "int",
			"step_timeout":              "int",
			"clear_context_on_approval": "bool",
		},
	},
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
		},
		FieldTypes: map[string]string{
			"enabled":             "bool",
			"skip_install_prompt": "bool",
			"servers":             "structmap",
		},
	},
	"openai": {
		Title: "OpenAI設定",
		Icon:  "🌟",
		Fields: map[string]string{
			"responses_api_models": "Responses APIを使用するモデル",
		},
		FieldTypes: map[string]string{
			"responses_api_models": "[]string",
		},
	},
	"thinking": {
		Title: "Extended Thinking設定",
		Icon:  "🧠",
		Fields: map[string]string{
			"enabled": "有効化",
			"level":   "レベル: low / medium / high / xhigh",
		},
		FieldTypes: map[string]string{
			"enabled": "bool",
			"level":   "select",
		},
		SelectOpts: map[string][]string{
			"level": {"low", "medium", "high", "xhigh"},
		},
	},
	"output": {
		Title: "ツール出力表示設定",
		Icon:  "📤",
		Fields: map[string]string{
			"max_lines": "折りたたみ前の最大表示行数",
		},
		FieldTypes: map[string]string{
			"max_lines": "int",
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
	"utility_model": {
		Title: "Utility Model設定",
		Icon:  "🪶",
		Comments: []string{
			"main 推論や compression.model とは別の軽量補助モデル設定",
			"初期実装では web_search 結果圧縮のような限定タスクだけに使用",
		},
		Fields: map[string]string{
			"enabled":  "utility model を有効化",
			"provider": "補助タスク用プロバイダー（openai / gemini / claude など）",
			"model":    "補助タスク用モデル（空 = provider_models の default_model）",
			"tasks":    "許可する軽量タスク（例: web_search_compaction）",
		},
		FieldTypes: map[string]string{
			"enabled":  "bool",
			"provider": "select",
			"model":    "string",
			"tasks":    "[]string",
		},
		SelectOpts: map[string][]string{
			"provider": {"openai", "gemini", "claude", "deepseek", "openrouter", "groq", "ollama", "bedrock"},
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
			"on_completion": "完了時に実行するコマンド（例: go test ./...）",
			"timeout":       "コマンドタイムアウト（秒）（デフォルト: 60）",
			"max_retry":     "フック失敗時の最大リトライ回数（デフォルト: 3）",
		},
		FieldTypes: map[string]string{
			"on_completion": "[]string",
			"timeout":       "int",
			"max_retry":     "int",
		},
	},
}

// SectionOrder はセクションの表示順序
var SectionOrder = []string{
	"default_provider",
	"default_model",
	"provider_models",
	"compression",
	"loop_detection",
	"api_retry",
	"diff",
	"tool_confirm",
	"command_aliases",
	"prompt_cache",
	"paste",
	"streaming",
	"bash",
	"project_map",
	"git_stage",
	"plan_mode",
	"lsp",
	"openai",
	"thinking",
	"output",
	"web_search",
	"sub_agent",
	"utility_model",
	"mcp",
	"hooks",
}

// CategoryOrder はカテゴリの表示順序（UIでのグループ化用）
// default_provider と provider_models は同じ "provider" カテゴリにまとめる
var CategoryOrder = []string{
	"provider", // default_provider, provider_models
	"compression",
	"loop_detection",
	"api_retry",
	"diff",
	"tool_confirm",
	"command_aliases",
	"prompt_cache",
	"paste",
	"streaming",
	"bash",
	"project_map",
	"git_stage",
	"plan_mode",
	"lsp",
	"openai",
	"thinking",
	"output",
	"web_search",
	"sub_agent",
	"utility_model",
	"mcp",
	"hooks",
}

// SectionToCategory はセクション名をカテゴリ名にマップ
var SectionToCategory = map[string]string{
	"default_provider": "provider",
	"default_model":    "provider",
	"provider_models":  "provider",
	"compression":      "compression",
	"loop_detection":   "loop_detection",
	"api_retry":        "api_retry",
	"diff":             "diff",
	"tool_confirm":     "tool_confirm",
	"command_aliases":  "command_aliases",
	"prompt_cache":     "prompt_cache",
	"paste":            "paste",
	"streaming":        "streaming",
	"bash":             "bash",
	"project_map":      "project_map",
	"git_stage":        "git_stage",
	"plan_mode":        "plan_mode",
	"lsp":              "lsp",
	"openai":           "openai",
	"thinking":         "thinking",
	"output":           "output",
	"web_search":       "web_search",
	"sub_agent":        "sub_agent",
	"utility_model":    "utility_model",
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
	"compression": {
		DisplayName: "Compression",
		Icon:        "📦",
		Sections:    []string{"compression"},
	},
	"loop_detection": {
		DisplayName: "Loop Detection",
		Icon:        "🔄",
		Sections:    []string{"loop_detection"},
	},
	"api_retry": {
		DisplayName: "API Settings",
		Icon:        "🌐",
		Sections:    []string{"api_retry"},
	},
	"diff": {
		DisplayName: "Diff Display",
		Icon:        "📝",
		Sections:    []string{"diff"},
	},
	"tool_confirm": {
		DisplayName: "Tool Confirm",
		Icon:        "✅",
		Sections:    []string{"tool_confirm"},
	},
	"command_aliases": {
		DisplayName: "Command Aliases",
		Icon:        "🔗",
		Sections:    []string{"command_aliases"},
	},
	"prompt_cache": {
		DisplayName: "Prompt Cache",
		Icon:        "💨",
		Sections:    []string{"prompt_cache"},
	},
	"paste": {
		DisplayName: "Paste Mode",
		Icon:        "📋",
		Sections:    []string{"paste"},
	},
	"streaming": {
		DisplayName: "Streaming",
		Icon:        "📺",
		Sections:    []string{"streaming"},
	},
	"bash": {
		DisplayName: "Bash Safety",
		Icon:        "💻",
		Sections:    []string{"bash"},
	},
	"project_map": {
		DisplayName: "Project Map",
		Icon:        "🗺️",
		Sections:    []string{"project_map"},
	},
	"git_stage": {
		DisplayName: "Git Settings",
		Icon:        "📂",
		Sections:    []string{"git_stage"},
	},
	"plan_mode": {
		DisplayName: "Plan Mode",
		Icon:        "📋",
		Sections:    []string{"plan_mode"},
	},
	"lsp": {
		DisplayName: "LSP Servers",
		Icon:        "🔧",
		Sections:    []string{"lsp"},
	},
	"openai": {
		DisplayName: "OpenAI",
		Icon:        "🌟",
		Sections:    []string{"openai"},
	},
	"thinking": {
		DisplayName: "Thinking",
		Icon:        "🧠",
		Sections:    []string{"thinking"},
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
	"utility_model": {
		DisplayName: "Utility Model",
		Icon:        "🪶",
		Sections:    []string{"utility_model"},
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
