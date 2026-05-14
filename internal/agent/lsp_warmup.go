package agent

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

const lspWarmupTimeout = 15 * time.Second

func warmupLSPClient(client *lsp.Client, rootDir string, configs map[string]lsp.ServerConfig, errOut io.Writer) {
	for _, serverKey := range lspWarmupTargets(rootDir, configs) {
		warmCtx, warmCancel := context.WithTimeout(context.Background(), lspWarmupTimeout)
		if _, err := client.GetServer(warmCtx, serverKey); err != nil {
			fmt.Fprintf(lspWarmupErrorOutput(errOut), "LSP warm-up: failed to start %s (%v)\n", serverKey, err)
		}
		warmCancel()
	}
}

func lspWarmupTargets(rootDir string, configs map[string]lsp.ServerConfig) []string {
	languages, err := lspDetectProjectLanguages(rootDir)
	if err != nil || len(languages) == 0 {
		return nil
	}

	targets := make([]string, 0, len(languages))
	for _, language := range languages {
		serverConfig, ok := configs[language.ServerKey]
		if !ok || serverConfig.Disabled || serverConfig.Command == "" {
			continue
		}
		if _, err := lspLookPath(serverConfig.Command); err != nil {
			continue
		}
		targets = append(targets, language.ServerKey)
	}
	return targets
}

func lspWarmupErrorOutput(errOut io.Writer) io.Writer {
	if errOut != nil {
		return errOut
	}
	return io.Discard
}
