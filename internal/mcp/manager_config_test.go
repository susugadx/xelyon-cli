package mcp

import (
	"encoding/json"
	"os"
	"testing"
)

func TestManager_LoadConfig_CreatesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	manager := NewManager()
	err := manager.LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if manager.config == nil {
		t.Fatal("Expected config to be loaded")
	}

	// デフォルト設定にfilesystemサーバーが含まれているべき
	if _, ok := manager.config.MCPServers["filesystem"]; !ok {
		t.Error("Expected default config to contain 'filesystem' server")
	}

	// ファイルが作成されているべき
	configPath := tmpDir + "/.xelyon/mcp.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Expected config file to be created at %s", configPath)
	}
}

func TestManager_LoadConfig_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// カスタム設定を作成
	configDir := tmpDir + "/.xelyon"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	customConfig := Config{
		MCPServers: map[string]ServerConfig{
			"custom-server": {
				Command: "npx",
				Args:    []string{"@custom/server"},
				Env:     map[string]string{"TEST": "value"},
			},
		},
	}

	data, _ := json.MarshalIndent(customConfig, "", "  ")
	if err := os.WriteFile(configDir+"/mcp.json", data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager := NewManager()
	err := manager.LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if manager.config == nil {
		t.Fatal("Expected config to be loaded")
	}

	// カスタムサーバーが読み込まれているべき
	if server, ok := manager.config.MCPServers["custom-server"]; !ok {
		t.Error("Expected custom-server to be loaded")
	} else {
		if server.Command != "npx" {
			t.Errorf("Expected command 'npx', got %q", server.Command)
		}
	}
}

func TestManager_LoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	configDir := tmpDir + "/.xelyon"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// 不正なJSONを書き込む
	invalidJSON := `{"mcpServers": {invalid json`
	if err := os.WriteFile(configDir+"/mcp.json", []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	manager := NewManager()
	err := manager.LoadConfig()

	// エラーが返るべき
	if err == nil {
		t.Error("LoadConfig should return error for invalid JSON")
	}
}

func TestServerConfig_Disabled(t *testing.T) {
	config := &Config{
		MCPServers: map[string]ServerConfig{
			"disabled_server": {
				Command:  "npx",
				Args:     []string{"some-server"},
				Disabled: true,
			},
		},
	}

	server := config.MCPServers["disabled_server"]
	if !server.Disabled {
		t.Error("Expected server to be disabled")
	}
}

func TestServerConfig_ToolsFilter_JSON(t *testing.T) {
	jsonStr := `{
		"mcpServers": {
			"github": {
				"command": "npx",
				"args": ["@modelcontextprotocol/server-github"],
				"disabled": false,
				"tools": {
					"include": ["create_issue", "list_issues"]
				}
			},
			"filesystem": {
				"command": "npx",
				"args": ["@modelcontextprotocol/server-filesystem"],
				"tools": {
					"exclude": ["delete_file"]
				}
			},
			"legacy": {
				"command": "npx",
				"args": ["some-server"]
			}
		}
	}`

	var config Config
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	gh := config.MCPServers["github"]
	if gh.Tools == nil || len(gh.Tools.Include) != 2 {
		t.Errorf("github tools include: got %+v", gh.Tools)
	}

	fs := config.MCPServers["filesystem"]
	if fs.Tools == nil || len(fs.Tools.Exclude) != 1 {
		t.Errorf("filesystem tools exclude: got %+v", fs.Tools)
	}

	legacy := config.MCPServers["legacy"]
	if legacy.Tools != nil {
		t.Errorf("legacy should have nil tools filter, got %+v", legacy.Tools)
	}
}
