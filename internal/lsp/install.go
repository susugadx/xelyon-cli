package lsp

import (
	"os/exec"
)

// InstallInfo はLSPサーバーのインストール情報
type InstallInfo struct {
	ServerKey   string   // サーバーキー（go, typescript等）
	PackageName string   // パッケージ名（gopls, vtsls等）
	Commands    []string // インストールコマンド（複数可: 1つ目を優先）
}

// InstallCommands はサーバーキー -> インストール情報のマッピング
var InstallCommands = map[string]InstallInfo{
	// ===== Existing (4 languages) =====
	"go": {
		ServerKey:   "go",
		PackageName: "gopls",
		Commands:    []string{"go install golang.org/x/tools/gopls@latest"},
	},
	"typescript": {
		ServerKey:   "typescript",
		PackageName: "vtsls",
		Commands:    []string{"npm i -g @vtsls/language-server typescript"},
	},
	"python": {
		ServerKey:   "python",
		PackageName: "pyright",
		Commands:    []string{"pip install pyright", "npm i -g pyright"},
	},
	"rust": {
		ServerKey:   "rust",
		PackageName: "rust-analyzer",
		Commands:    []string{"rustup component add rust-analyzer"},
	},

	// ===== Tier 1: Backend languages (11 languages) =====
	"java": {
		ServerKey:   "java",
		PackageName: "jdtls",
		Commands:    []string{"brew install jdtls"}, // macOS. Linux: manual download from https://download.eclipse.org/jdtls/
	},
	"c": {
		ServerKey:   "c",
		PackageName: "clangd",
		Commands:    []string{"brew install llvm", "apt install clangd"},
	},
	"cpp": {
		ServerKey:   "cpp",
		PackageName: "clangd",
		Commands:    []string{"brew install llvm", "apt install clangd"},
	},
	"ruby": {
		ServerKey:   "ruby",
		PackageName: "solargraph",
		Commands:    []string{"gem install solargraph"},
	},
	"kotlin": {
		ServerKey:   "kotlin",
		PackageName: "kotlin-language-server",
		Commands:    []string{"brew install kotlin-language-server"}, // macOS. Other: manual from https://github.com/fwcd/kotlin-language-server
	},
	"swift": {
		ServerKey:   "swift",
		PackageName: "sourcekit-lsp",
		Commands:    []string{}, // Included with Xcode or Swift toolchain
	},
	"csharp": {
		ServerKey:   "csharp",
		PackageName: "csharp-ls",
		Commands:    []string{"dotnet tool install --global csharp-ls"},
	},
	"scala": {
		ServerKey:   "scala",
		PackageName: "metals",
		Commands:    []string{"brew install coursier/formulas/coursier && cs install metals"},
	},
	"php": {
		ServerKey:   "php",
		PackageName: "intelephense",
		Commands:    []string{"npm i -g intelephense"},
	},
	"elixir": {
		ServerKey:   "elixir",
		PackageName: "elixir-ls",
		Commands:    []string{"brew install elixir-ls"}, // macOS. Other: mix escript.install github elixir-lsp/elixir-ls
	},
	"lua": {
		ServerKey:   "lua",
		PackageName: "lua-language-server",
		Commands:    []string{"brew install lua-language-server", "apt install lua-language-server"},
	},

	// ===== Tier 2: Frontend languages (4 languages) =====
	"css": {
		ServerKey:   "css",
		PackageName: "vscode-css-language-server",
		Commands:    []string{"npm i -g vscode-langservers-extracted"},
	},
	"html": {
		ServerKey:   "html",
		PackageName: "vscode-html-language-server",
		Commands:    []string{"npm i -g vscode-langservers-extracted"},
	},
	"vue": {
		ServerKey:   "vue",
		PackageName: "@vue/language-server",
		Commands:    []string{"npm i -g @vue/language-server"},
	},
	"svelte": {
		ServerKey:   "svelte",
		PackageName: "svelte-language-server",
		Commands:    []string{"npm i -g svelte-language-server"},
	},

	// ===== Tier 3: Config/Script languages (5 languages) =====
	"yaml": {
		ServerKey:   "yaml",
		PackageName: "yaml-language-server",
		Commands:    []string{"npm i -g yaml-language-server"},
	},
	"toml": {
		ServerKey:   "toml",
		PackageName: "taplo",
		Commands:    []string{"cargo install taplo-cli --locked"},
	},
	"sql": {
		ServerKey:   "sql",
		PackageName: "sqls",
		Commands:    []string{"go install github.com/lighttiger2505/sqls@latest"},
	},
	"bash": {
		ServerKey:   "bash",
		PackageName: "bash-language-server",
		Commands:    []string{"npm i -g bash-language-server"},
	},
	"markdown": {
		ServerKey:   "markdown",
		PackageName: "marksman",
		Commands:    []string{"brew install marksman"}, // macOS. Other: download from https://github.com/artempyanykh/marksman/releases
	},
}

// GetInstallInfo はサーバーキーからインストール情報を取得
func GetInstallInfo(serverKey string) (InstallInfo, bool) {
	info, ok := InstallCommands[serverKey]
	return info, ok
}

// GetAllInstallInfos は全てのインストール情報を返す
func GetAllInstallInfos() map[string]InstallInfo {
	return InstallCommands
}

// IsServerInstalled はLSPサーバーがインストールされているかチェック
func IsServerInstalled(serverKey string, configs map[string]ServerConfig) bool {
	config, ok := configs[serverKey]
	if !ok {
		return false
	}

	if config.Disabled {
		return true // disabled は「インストール済みだが無効」として扱う
	}

	// コマンドが存在するかチェック
	_, err := exec.LookPath(config.Command)
	return err == nil
}
