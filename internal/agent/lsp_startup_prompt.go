package agent

import (
	"io"
	"os/exec"

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

	cwd, err := lspCommandGetwd()
	if err != nil {
		return
	}
	languages, err := lspCommandDetectProjectLanguages(cwd)
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
	writeLSPInstallPromptRemediation(out, missing, commandSurface)
	dim.Fprintln(out, "   Set lsp.skip_install_prompt: true to hide this startup notice.")
}

func writeLSPInstallPromptRemediation(out io.Writer, missing []lspInstallPromptItem, commandSurface commandcatalog.CommandSurface) {
	if commandSurface == commandcatalog.CommandSurfaceClassic {
		dim.Fprintln(out, "   Suggested commands:")
		for _, item := range missing {
			dim.Fprintf(out, "     /lsp install %s\n", item.serverKey)
		}
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
		if _, err := exec.LookPath(serverConfig.Command); err == nil {
			continue
		}

		info, ok := lspCommandGetInstallInfo(language.ServerKey)
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
