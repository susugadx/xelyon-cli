package agent

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func resolveProjectMapSourceCWD(agent *Agent, overrideCWD string) (string, bool) {
	cwd := strings.TrimSpace(overrideCWD)
	if cwd != "" {
		return cwd, true
	}
	if agent == nil {
		return "", false
	}
	cwd = strings.TrimSpace(agent.invocationCWD())
	if cwd == "" {
		return "", false
	}
	return cwd, true
}

func resolveProjectMapSourceRootPath(cwd string, pc *config.ProjectConfig) string {
	if pc == nil || strings.TrimSpace(pc.FilePath) == "" {
		return cwd
	}
	return filepath.Dir(pc.FilePath)
}
