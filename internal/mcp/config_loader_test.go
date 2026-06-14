package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	server, ok := result.config.MCPServers["filesystem"]
	if !ok {
		t.Fatalf("default MCP server missing: %+v", result.config.MCPServers)
	}
	if !server.Disabled {
		t.Fatalf("default filesystem server Disabled = false, want true")
	}
	if _, err := os.Stat(result.configPath); err != nil {
		t.Fatalf("default config path not created: %v", err)
	}
}

func TestManagerLoadConfig_CreatedDefaultDoesNotConnectActiveServer(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	manager := NewManager()
	var output bytes.Buffer
	manager.SetOutput(&output)

	if err := manager.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if err := manager.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if len(manager.sessions) != 0 {
		t.Fatalf("sessions = %d, want 0 for disabled default sample", len(manager.sessions))
	}
	if len(manager.tools) != 0 {
		t.Fatalf("tools = %d, want 0 for disabled default sample", len(manager.tools))
	}
	out := output.String()
	if !strings.Contains(out, "Created default MCP config") {
		t.Fatalf("output = %q, want default creation notice", out)
	}
	if !strings.Contains(out, "MCP server 'filesystem' is disabled, skipping") {
		t.Fatalf("output = %q, want disabled default skip notice", out)
	}
	if strings.Contains(out, "connection failed") {
		t.Fatalf("output = %q, default sample should not attempt connection", out)
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
