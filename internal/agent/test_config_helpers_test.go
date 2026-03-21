package agent

import "github.com/susugadx/xelyon-cli/internal/config"

func newProjectMapDisabledConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProjectMap.Enabled = false
	return cfg
}
