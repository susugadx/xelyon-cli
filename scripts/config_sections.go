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
			"default_provider": {"deepseek", "claude", "openai", "gemini", "groq", "ollama"},
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
	"compression": {
		Title: "会話履歴圧縮設定",
		Icon:  "📦",
		Fields: map[string]string{
			"auto_compress":      "トークン使用率が閾値を超えたら自動圧縮",
			"threshold_tokens":   "トークン数の閾値（0 = 使用しない）",
			"threshold_percent":  "トークン使用率の閾値（%）",
			"keep_recent":        "圧縮時に保持する直近メッセージ数",
			"prefer_compact_api": "プロバイダーのCompact APIを優先使用",
		},
		FieldTypes: map[string]string{
			"auto_compress":      "bool",
			"threshold_tokens":   "int",
			"threshold_percent":  "int",
			"keep_recent":        "int",
			"prefer_compact_api": "bool",
		},
	},
	"backup": {
		Title: "バックアップ設定",
		Icon:  "💾",
		Fields: map[string]string{
			"max_generations": "保持する世代数",
		},
		FieldTypes: map[string]string{
			"max_generations": "int",
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
			"context_lines": "差分表示時のコンテキスト行数",
		},
		FieldTypes: map[string]string{
			"context_lines": "int",
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
		Title: "プロンプトキャッシュ設定（Claude専用）",
		Icon:  "💨",
		Fields: map[string]string{
			"enabled":     "有効化",
			"max_entries": "最大エントリ数",
			"ttl_seconds": "キャッシュTTL（秒）",
		},
		FieldTypes: map[string]string{
			"enabled":     "bool",
			"max_entries": "int",
			"ttl_seconds": "int",
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
	"code_health": {
		Title: "コード健全性チェック設定",
		Icon:  "🏥",
		Fields: map[string]string{
			"enabled":            "有効化",
			"max_file_lines":     "ファイルの最大行数警告",
			"max_function_lines": "関数の最大行数警告",
			"auto_suggest":       "変更時に自動提案",
			"on_change":          "変更時のチェック項目",
		},
		FieldTypes: map[string]string{
			"enabled":            "bool",
			"max_file_lines":     "int",
			"max_function_lines": "int",
			"auto_suggest":       "bool",
			"on_change":          "[]string",
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
			"parallel":         "並列モード有効化",
			"max_workers":      "並列ワーカー数",
			"max_retry":        "最大リトライ回数",
			"step_timeout":     "ステップタイムアウト（秒）",
			"confirm_level":    "確認レベル: all / dangerous / none",
			"supervisor_model": "Supervisor用モデル（空=メインモデル）",
			"light_model":      "Worker用軽量モデル（空=メインモデル）",
			"heavy_model":      "エスカレーション用モデル（空=無効）",
		},
		FieldTypes: map[string]string{
			"parallel":         "bool",
			"max_workers":      "int",
			"max_retry":        "int",
			"step_timeout":     "int",
			"confirm_level":    "select",
			"supervisor_model": "string",
			"light_model":      "string",
			"heavy_model":      "string",
		},
		SelectOpts: map[string][]string{
			"confirm_level": {"all", "dangerous", "none"},
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
	"repomap": {
		Title: "RepoMap設定",
		Icon:  "🗺️",
		Fields: map[string]string{
			"max_tokens": "最大トークン数（0 = 自動計算）",
		},
		FieldTypes: map[string]string{
			"max_tokens": "int",
		},
	},
	"web_search": {
		Title: "Web検索設定",
		Icon:  "🔍",
		Comments: []string{
			"Serper API Web検索のキャッシュ設定",
		},
		Fields: map[string]string{
			"cache_enabled": "キャッシュを有効化（デフォルト: true）",
			"cache_ttl":     "キャッシュTTL秒数（デフォルト: 1800 = 30分）",
			"cache_size":    "最大キャッシュ数（デフォルト: 100）",
		},
		FieldTypes: map[string]string{
			"cache_enabled": "bool",
			"cache_ttl":     "int",
			"cache_size":    "int",
		},
	},
}

// SectionOrder はセクションの表示順序
var SectionOrder = []string{
	"default_provider",
	"default_model",
	"provider_models",
	"compression",
	"backup",
	"loop_detection",
	"api_retry",
	"diff",
	"tool_confirm",
	"command_aliases",
	"prompt_cache",
	"paste",
	"streaming",
	"bash",
	"code_health",
	"git_stage",
	"plan_mode",
	"lsp",
	"openai",
	"thinking",
	"output",
	"repomap",
	"web_search",
}

// CategoryOrder はカテゴリの表示順序（UIでのグループ化用）
// default_provider と provider_models は同じ "provider" カテゴリにまとめる
var CategoryOrder = []string{
	"provider", // default_provider, provider_models
	"compression",
	"backup",
	"loop_detection",
	"api_retry",
	"diff",
	"tool_confirm",
	"command_aliases",
	"prompt_cache",
	"paste",
	"streaming",
	"bash",
	"code_health",
	"git_stage",
	"plan_mode",
	"lsp",
	"openai",
	"thinking",
	"output",
	"repomap",
	"web_search",
}

// SectionToCategory はセクション名をカテゴリ名にマップ
var SectionToCategory = map[string]string{
	"default_provider": "provider",
	"default_model":    "provider",
	"provider_models":  "provider",
	"compression":      "compression",
	"backup":           "backup",
	"loop_detection":   "loop_detection",
	"api_retry":        "api_retry",
	"diff":             "diff",
	"tool_confirm":     "tool_confirm",
	"command_aliases":  "command_aliases",
	"prompt_cache":     "prompt_cache",
	"paste":            "paste",
	"streaming":        "streaming",
	"bash":             "bash",
	"code_health":      "code_health",
	"git_stage":        "git_stage",
	"plan_mode":        "plan_mode",
	"lsp":              "lsp",
	"openai":           "openai",
	"thinking":         "thinking",
	"output":           "output",
	"repomap":          "repomap",
	"web_search":       "web_search",
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
	"backup": {
		DisplayName: "Backup",
		Icon:        "💾",
		Sections:    []string{"backup"},
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
	"code_health": {
		DisplayName: "Code Health",
		Icon:        "🏥",
		Sections:    []string{"code_health"},
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
	"repomap": {
		DisplayName: "RepoMap",
		Icon:        "🗺️",
		Sections:    []string{"repomap"},
	},
	"web_search": {
		DisplayName: "Web Search",
		Icon:        "🔍",
		Sections:    []string{"web_search"},
	},
}
