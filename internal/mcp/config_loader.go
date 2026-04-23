package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type mcpConfigLoadResult struct {
	config         *Config
	configPath     string
	createdDefault bool
}

func loadMCPConfig(homeDir string) (*mcpConfigLoadResult, error) {
	xelyonDir := filepath.Join(homeDir, ".xelyon")
	configPath := filepath.Join(xelyonDir, "mcp.json")

	if err := ensureMCPConfigDir(xelyonDir); err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg, err := createDefaultMCPConfig(configPath)
		if err != nil {
			return nil, err
		}
		return &mcpConfigLoadResult{
			config:         cfg,
			configPath:     configPath,
			createdDefault: true,
		}, nil
	}

	cfg, err := readMCPConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &mcpConfigLoadResult{
		config:     cfg,
		configPath: configPath,
	}, nil
}

func ensureMCPConfigDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create .xelyon directory: %w", err)
		}
	}
	return nil
}

func createDefaultMCPConfig(configPath string) (*Config, error) {
	defaultConfig := newDefaultMCPConfig()
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write default config: %w", err)
	}
	return &defaultConfig, nil
}

func readMCPConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func newDefaultMCPConfig() Config {
	return Config{
		MCPServers: map[string]ServerConfig{
			// 例: filesystemサーバー
			"filesystem": {
				Command: "npx",
				Args:    []string{"@modelcontextprotocol/server-filesystem", "/path/to/directory"},
				Env:     map[string]string{"NODE_OPTIONS": "--no-warnings"},
			},
		},
	}
}
