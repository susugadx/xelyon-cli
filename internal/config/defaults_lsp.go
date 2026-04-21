package config

func defaultLSPConfig() LSPConfig {
	return LSPConfig{
		Enabled: true,
		Servers: defaultLSPServers(),
	}
}

func defaultLSPServers() map[string]LSPServerConfig {
	return map[string]LSPServerConfig{
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
	}
}
