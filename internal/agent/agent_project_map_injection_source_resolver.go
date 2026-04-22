package agent

import (
	"os"
	"path/filepath"
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

func resolveProjectMapSourceRootPath(cwd string, pc *config.ProjectConfig) string {
	if pc == nil || strings.TrimSpace(pc.FilePath) == "" {
		return cwd
	}
	return filepath.Dir(pc.FilePath)
}
