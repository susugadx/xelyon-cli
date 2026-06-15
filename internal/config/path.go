package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// getConfigPath は設定ファイルのパスを返す
func getConfigPath() (string, error) {
	dir, err := getConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

func getConfigDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, configDir), nil
}
