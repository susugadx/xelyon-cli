package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setTestHome(t *testing.T) string {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	return homeDir
}

func writeConfigYAMLForTest(t *testing.T, yamlData string) string {
	t.Helper()

	homeDir := setTestHome(t)
	configDir := filepath.Join(homeDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0o600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	return configPath
}
