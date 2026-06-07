package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func resolveProjectMapSourceCWD(agent *Agent) (string, bool) {
	if agent == nil {
		return "", false
	}
	cwd := strings.TrimSpace(agent.invocationCWD())
	if cwd == "" {
		return "", false
	}
	return cwd, true
}

func resolveProjectMapSourceRootPath(_ string, bundle *config.ProjectInstructionBundle) string {
	if bundle == nil {
		return ""
	}
	if rootPath, ok := bundle.ProjectRootPath(); ok {
		return rootPath
	}
	return ""
}

func resolveProjectMapSourceRootPathWithFallback(cwd string, bundle *config.ProjectInstructionBundle, fallbackRootPath string) string {
	rootPath := resolveProjectMapSourceRootPath(cwd, bundle)
	if strings.TrimSpace(rootPath) != "" {
		return rootPath
	}
	if bundle == nil && strings.TrimSpace(fallbackRootPath) != "" {
		return fallbackRootPath
	}
	return ""
}
