//go:build ignore

package main

// CommandInfo はコマンド情報
type CommandInfo struct {
	Name          string       // "/config"
	Aliases       []string     // []string{"/quit", "/q"}
	Args          string       // "[id]", "<provider> [model]"
	Description   string       // "Interactive configuration menu"
	DescriptionJP string       // "対話式設定変更"
	SubCommands   []SubCommand // サブコマンド
}

// SubCommand はサブコマンド情報
type SubCommand struct {
	Name        string // "/config show"
	Description string // "Show all settings with diff from defaults"
}

// ToolInfo はツール情報
type ToolInfo struct {
	Name        string // "bash"
	Description string // "Execute shell commands"
}

// Commands はコマンド一覧
var Commands = []CommandInfo{
	{
		Name:          "/exit",
		Aliases:       []string{"/quit", "/q"},
		Description:   "Exit the CLI",
		DescriptionJP: "CLIを終了",
	},
	{
		Name:          "/clear",
		Description:   "Clear conversation history",
		DescriptionJP: "会話履歴をクリア",
	},
	{
		Name:          "/history",
		Description:   "Show conversation history",
		DescriptionJP: "会話履歴を表示",
	},
	{
		Name:          "/save",
		Description:   "Save current session",
		DescriptionJP: "セッションを保存",
	},
	{
		Name:          "/load",
		Args:          "[id]",
		Description:   "Load session (or last if no ID)",
		DescriptionJP: "セッションを読み込み",
	},
	{
		Name:          "/sessions",
		Description:   "List recent sessions",
		DescriptionJP: "最近のセッション一覧",
	},
	{
		Name:          "/status",
		Aliases:       []string{"/stats"},
		Description:   "Show current state, last request, and session statistics",
		DescriptionJP: "現在状態、直近リクエスト、セッション統計を表示",
	},
	{
		Name:          "/tokens",
		Description:   "Show token usage and context window status",
		DescriptionJP: "トークン使用量を表示",
	},
	{
		Name:          "/copy",
		Args:          "[code] [-n N]",
		Description:   "Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)",
		DescriptionJP: "AI出力をクリップボードにコピー",
	},
	{
		Name:          "/compress",
		Args:          "[N] [-c]",
		Description:   "Compress history (keep recent N, -c: use OpenAI Compact API)",
		DescriptionJP: "履歴を圧縮",
	},
	{
		Name:          "/use",
		Args:          "<provider> [model]",
		Description:   "Switch provider and optionally model (e.g., /use gemini gemini-2.0-flash-exp)",
		DescriptionJP: "プロバイダーを切り替え",
	},
	{
		Name:          "/providers",
		Description:   "List available providers and their API key status",
		DescriptionJP: "利用可能なプロバイダー一覧",
	},
	{
		Name:          "/config",
		Description:   "Interactive configuration menu",
		DescriptionJP: "対話式設定変更",
		SubCommands: []SubCommand{
			{Name: "/config show", Description: "Show all settings with diff from defaults"},
			{Name: "/config model <name>", Description: "Change default model"},
		},
	},
	{
		Name:          "/model",
		Args:          "[name]",
		Description:   "Show current model or switch model without restart",
		DescriptionJP: "モデルを表示/切り替え",
	},
	{
		Name:          "/init",
		Description:   "Create xelyon.yaml template (project config)",
		DescriptionJP: "xelyon.yamlテンプレートを作成",
	},
	{
		Name:          "/project",
		Description:   "Edit xelyon.yaml interactively (rules, hooks)",
		DescriptionJP: "xelyon.yamlを対話式で編集",
	},
	{
		Name:          "/paste",
		Aliases:       []string{"/p"},
		Description:   "Paste mode for long text (end with empty line x2, END, or Ctrl+D)",
		DescriptionJP: "ペーストモード",
	},
	{
		Name:          "/plan",
		Args:          "[on|off]",
		Description:   "Toggle Plan Mode (investigation -> plan -> approval -> execution)",
		DescriptionJP: "Plan Modeを切り替え",
	},
	{
		Name:          "/think",
		Args:          "[on|off|level]",
		Description:   "Toggle Extended Thinking mode (level: low/medium/high/xhigh)",
		DescriptionJP: "Extended Thinkingを切り替え",
	},
	{
		Name:          "/lsp",
		Args:          "[status]",
		Description:   "Show LSP server status (running/not started/disabled)",
		DescriptionJP: "LSPサーバー状態を表示",
	},
	{
		Name:          "/version",
		Description:   "Show version information",
		DescriptionJP: "バージョンを表示",
	},
	{
		Name:          "/help",
		Description:   "Show this help",
		DescriptionJP: "ヘルプを表示",
	},
}

// Tools はツール一覧
var Tools = []ToolInfo{
	{Name: "bash", Description: "Execute shell commands"},
	{Name: "read_file", Description: "Read file contents"},
	{Name: "write_file", Description: "Write/create files"},
	{Name: "str_replace", Description: "Replace text in file"},
	{Name: "list_dir", Description: "List directory contents"},
	{Name: "git_*", Description: "Git operations (status, diff, add, commit, push, log)"},
}

// Tips はTips一覧
var Tips = []string{
	"Just describe what you want in natural language",
	"AI will ask confirmation for dangerous operations",
	"Use Ctrl+C to cancel current operation",
	"Use git to revert file changes when needed",
}
