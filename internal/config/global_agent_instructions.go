package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultGlobalAgentInstructionsFile = "AGENTS.md"

func ensureDefaultGlobalAgentInstructionsFile() error {
	path, err := getDefaultGlobalAgentInstructionsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		return file.Close()
	}
	if os.IsExist(err) {
		return nil
	}
	return fmt.Errorf("failed to create global agent instructions file: %w", err)
}

func getDefaultGlobalAgentInstructionsPath() (string, error) {
	dir, err := getConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultGlobalAgentInstructionsFile), nil
}
