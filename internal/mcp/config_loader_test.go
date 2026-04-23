package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMCPConfig_CreatesDefault(t *testing.T) {
	homeDir := t.TempDir()

	result, err := loadMCPConfig(homeDir)
	if err != nil {
		t.Fatalf("loadMCPConfig() error = %v", err)
	}
	if result == nil {
		t.Fatal("loadMCPConfig() returned nil result")
	}
	if !result.createdDefault {
		t.Fatal("createdDefault = false, want true")
	}
	if result.config == nil {
		t.Fatal("config = nil, want non-nil")
	}
	if _, ok := result.config.MCPServers["filesystem"]; !ok {
		t.Fatalf("default MCP server missing: %+v", result.config.MCPServers)
	}
	if _, err := os.Stat(result.configPath); err != nil {
		t.Fatalf("default config path not created: %v", err)
	}
}

func TestLoadMCPConfig_ReadsExisting(t *testing.T) {
	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	existing := Config{
		MCPServers: map[string]ServerConfig{
			"custom": {
				Command: "npx",
				Args:    []string{"@custom/server"},
			},
		},
	}
	data, err := json.Marshal(existing)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	configPath := filepath.Join(configDir, "mcp.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := loadMCPConfig(homeDir)
	if err != nil {
		t.Fatalf("loadMCPConfig() error = %v", err)
	}
	if result.createdDefault {
		t.Fatal("createdDefault = true, want false")
	}
	if result.config == nil {
		t.Fatal("config = nil, want non-nil")
	}
	if _, ok := result.config.MCPServers["custom"]; !ok {
		t.Fatalf("custom MCP server missing: %+v", result.config.MCPServers)
	}
	if result.configPath != configPath {
		t.Fatalf("configPath = %q, want %q", result.configPath, configPath)
	}
}

func TestLoadMCPConfig_InvalidJSON(t *testing.T) {
	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	configPath := filepath.Join(configDir, "mcp.json")
	if err := os.WriteFile(configPath, []byte(`{"mcpServers": {invalid`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loadMCPConfig(homeDir); err == nil {
		t.Fatal("loadMCPConfig() error = nil, want JSON parse error")
	}
}
