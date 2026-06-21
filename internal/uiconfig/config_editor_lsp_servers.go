package uiconfig

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/configedit"
)

type lspServersSnapshot struct {
	servers []string
}

func buildLSPServersSnapshot(cfg *config.Config) lspServersSnapshot {
	servers := make([]string, 0, len(cfg.LSP.Servers))
	for server := range cfg.LSP.Servers {
		servers = append(servers, server)
	}
	sort.Strings(servers)
	return lspServersSnapshot{servers: servers}
}

func (e *StructMapEditor) runLSPServers(cfg *config.Config, promptIO PromptIO) (bool, error) {
	out := promptIO.Out

	for {
		snapshot := buildLSPServersSnapshot(cfg)
		e.renderLSPServersMenu(cfg, out, snapshot)

		input := readConfigEditorChoice(&promptIO)
		done, saved := e.handleLSPServersChoice(cfg, &promptIO, out, snapshot, input)
		if done {
			return saved, nil
		}
	}
}

func (e *StructMapEditor) renderLSPServersMenu(cfg *config.Config, out io.Writer, snapshot lspServersSnapshot) {
	_, _ = fmt.Fprintf(out, "\n%s── lsp.servers ──────────────────────────%s\n\n", colorCyan, colorReset)
	_, _ = fmt.Fprintln(out, "  Configured LSP servers:")

	if len(snapshot.servers) == 0 {
		_, _ = fmt.Fprintf(out, "    %s(using defaults)%s\n", colorDim, colorReset)
	} else {
		for i, server := range snapshot.servers {
			serverConfig := cfg.LSP.Servers[server]
			status := ""
			if serverConfig.Disabled {
				status = " (disabled)"
			}
			_, _ = fmt.Fprintf(out, "    %d. %s: %s%s\n", i+1, server, truncateString(serverConfig.Command, 20), status)
		}
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  [a] Add server")
	_, _ = fmt.Fprintln(out, "  [1-9] Edit server")
	_, _ = fmt.Fprintln(out, "  [d] Delete server")
	_, _ = fmt.Fprintln(out, "  [s] Save and back")
	_, _ = fmt.Fprintln(out, "  [c] Cancel")
	_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)
}

func (e *StructMapEditor) handleLSPServersChoice(cfg *config.Config, promptIO *PromptIO, out io.Writer, snapshot lspServersSnapshot, input string) (done bool, saved bool) {
	switch input {
	case "a", "add":
		e.addLSPServer(cfg, promptIO, out)
	case "d", "delete":
		e.deleteLSPServer(cfg, promptIO, out, snapshot)
	case "s", "save":
		return true, true
	case "c", "cancel":
		return true, false
	default:
		e.editLSPServer(cfg, promptIO, out, snapshot, input)
	}
	return false, false
}

func (e *StructMapEditor) addLSPServer(cfg *config.Config, promptIO *PromptIO, out io.Writer) {
	_, _ = fmt.Fprint(out, "Enter language (e.g., python, rust): ")
	lang := configedit.NormalizeLSPServerLanguage(readLineWithIO(promptIO))
	if lang == "" {
		return
	}

	_, _ = fmt.Fprint(out, "Enter command (e.g., pyright-langserver): ")
	cmd := readLineWithIO(promptIO)
	cfg.LSP.Servers = configedit.WithAddedLSPServer(cfg.LSP.Servers, lang, cmd)
	_, _ = fmt.Fprintf(out, "%s✓ Added: %s%s\n", colorGreen, lang, colorReset)
}

func (e *StructMapEditor) deleteLSPServer(cfg *config.Config, promptIO *PromptIO, out io.Writer, snapshot lspServersSnapshot) {
	if len(snapshot.servers) == 0 {
		return
	}

	_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(snapshot.servers))
	numStr := readLineWithIO(promptIO)
	lang, ok := configedit.SelectLSPServerByInput(numStr, snapshot.servers)
	if !ok {
		return
	}

	delete(cfg.LSP.Servers, lang)
	_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, lang, colorReset)
}

func (e *StructMapEditor) editLSPServer(cfg *config.Config, promptIO *PromptIO, out io.Writer, snapshot lspServersSnapshot, input string) {
	lang, ok := configedit.SelectLSPServerByInput(input, snapshot.servers)
	if !ok {
		return
	}

	sConfig := cfg.LSP.Servers[lang]

	_, _ = fmt.Fprintf(out, "\nEditing %s:\n", lang)
	_, _ = fmt.Fprintf(out, "  [1] command: %s\n", sConfig.Command)
	_, _ = fmt.Fprintf(out, "  [2] disabled: %v\n", sConfig.Disabled)
	_, _ = fmt.Fprintln(out, "  [b] Back")
	_, _ = fmt.Fprint(out, "\nChoice: ")

	subInput := strings.TrimSpace(readLineWithIO(promptIO))
	switch subInput {
	case "1":
		e.updateLSPServerCommand(cfg, promptIO, out, lang, sConfig)
	case "2":
		e.toggleLSPServerDisabled(cfg, out, lang, sConfig)
	}
}

func (e *StructMapEditor) updateLSPServerCommand(cfg *config.Config, promptIO *PromptIO, out io.Writer, lang string, sConfig config.LSPServerConfig) {
	_, _ = fmt.Fprint(out, "Enter new command: ")
	updated, ok := configedit.UpdateLSPServerCommandValue(sConfig, readLineWithIO(promptIO))
	if !ok {
		return
	}

	cfg.LSP.Servers[lang] = updated
	_, _ = fmt.Fprintf(out, "%s✓ Updated command%s\n", colorGreen, colorReset)
}

func (e *StructMapEditor) toggleLSPServerDisabled(cfg *config.Config, out io.Writer, lang string, sConfig config.LSPServerConfig) {
	updated := configedit.ToggleLSPServerDisabledValue(sConfig)
	cfg.LSP.Servers[lang] = updated
	_, _ = fmt.Fprintf(out, "%s✓ Disabled = %v%s\n", colorGreen, updated.Disabled, colorReset)
}
