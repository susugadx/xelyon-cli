package agent

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func resolveProjectMapSourceCWD() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
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
