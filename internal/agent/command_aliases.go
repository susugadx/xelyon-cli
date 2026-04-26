package agent

import (
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func resolveCommandAliasWithConfig(cmd string, cfg *config.Config) string {
	return commandruntime.ResolveAlias(cmd, commandAliasesFromConfig(cfg))
}

func commandAliasesFromConfig(cfg *config.Config) map[string]string {
	if cfg == nil || cfg.CommandAliases == nil {
		return nil
	}
	return cfg.CommandAliases
}
