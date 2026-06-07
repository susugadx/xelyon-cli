package agent

import (
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/lsp"
)

type lspInstallPromptItem struct {
	serverKey string
	command   string
}

// checkLSPInstallPrompt は検出済みプロジェクト言語に必要な未導入 LSP サーバーだけを案内する。
func checkLSPInstallPrompt(agent *Agent, commandSurface commandcatalog.CommandSurface) {
	if agent == nil {
		return
	}

	cfg := agent.cfg()
	if cfg == nil || !cfg.LSP.Enabled || cfg.LSP.SkipInstallPrompt {
		return
	}

	cwd := strings.TrimSpace(agent.invocationCWD())
	if cwd == "" {
		var err error
		cwd, err = lspStartupGetwd()
		if err != nil {
			return
		}
	}
	rootDir, ok := resolveLSPStartupProjectRoot(cfg, cwd)
	if !ok {
		return
	}
	languages, err := lspDetectProjectLanguages(rootDir)
	if err != nil || len(languages) == 0 {
		return
	}

	missing := missingDetectedLSPServers(cfg.LSP.Servers, languages)
	if len(missing) == 0 {
		return
	}

	out := agent.output()
	yellow.Fprintln(out, "⚠️  LSP servers missing for detected project languages")
	dim.Fprintln(out, "   Install them to improve symbol/reference navigation and diagnostics:")
	for _, item := range missing {
		dim.Fprintf(out, "     %s: %s\n", item.serverKey, item.command)
	}
	writeLSPInstallPromptRemediation(out, commandSurface)
	dim.Fprintln(out, "   Set lsp.skip_install_prompt: true to hide this startup notice.")
}

func writeLSPInstallPromptRemediation(out io.Writer, commandSurface commandcatalog.CommandSurface) {
	if commandSurface == commandcatalog.CommandSurfaceClassic {
		dim.Fprintln(out, "   Install the listed command(s) in your shell. LSP server settings are in lsp.servers.")
		return
	}

	dim.Fprintln(out, "   Install in your shell. LSP server settings are under Config > LSP Servers.")
}

func missingDetectedLSPServers(configs map[string]config.LSPServerConfig, languages []lsp.LanguageInfo) []lspInstallPromptItem {
	missing := make([]lspInstallPromptItem, 0, len(languages))
	for _, language := range languages {
		serverConfig, ok := configs[language.ServerKey]
		if !ok || serverConfig.Disabled || serverConfig.Command == "" {
			continue
		}
		if _, err := lspLookPath(serverConfig.Command); err == nil {
			continue
		}

		info, ok := lspGetInstallInfo(language.ServerKey)
		if !ok || len(info.Commands) == 0 {
			continue
		}

		missing = append(missing, lspInstallPromptItem{
			serverKey: language.ServerKey,
			command:   info.Commands[0],
		})
	}
	return missing
}
