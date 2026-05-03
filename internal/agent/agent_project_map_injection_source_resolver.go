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

func resolveProjectMapSourceRootPath(cwd string, bundle *config.ProjectInstructionBundle) string {
	if bundle == nil || strings.TrimSpace(bundle.RootPath) == "" {
		return cwd
	}
	return bundle.RootPath
}

func resolveProjectMapSourceRootPathWithFallback(cwd string, bundle *config.ProjectInstructionBundle, fallbackRootPath string) string {
	rootPath := resolveProjectMapSourceRootPath(cwd, bundle)
	if bundle == nil && strings.TrimSpace(fallbackRootPath) != "" {
		return fallbackRootPath
	}
	return rootPath
}
