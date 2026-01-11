package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestValidateMCPCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "allowed npx command",
			command: "npx",
			wantErr: false,
		},
		{
			name:    "allowed node command",
			command: "node",
			wantErr: false,
		},
		{
			name:    "allowed python command",
			command: "python",
			wantErr: false,
		},
		{
			name:    "allowed python3 command",
			command: "python3",
			wantErr: false,
		},
		{
			name:    "allowed uvx command",
			command: "uvx",
			wantErr: false,
		},
		{
			name:    "allowed docker command",
			command: "docker",
			wantErr: false,
		},
		{
			name:    "blocked command (bash)",
			command: "bash",
			wantErr: true,
		},
		{
			name:    "blocked command (rm)",
			command: "rm",
			wantErr: true,
		},
		{
			name:    "path traversal with slashes",
			command: "/usr/bin/node",
			wantErr: true,
		},
		{
			name:    "path traversal with dots",
			command: "../node",
			wantErr: true,
		},
		{
			name:    "empty command",
			command: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPCommand(%q) error = %v, wantErr %v", tt.command, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeEnv(t *testing.T) {
	// 環境変数をモック
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, e := range originalEnv {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// テスト用の環境変数を設定
	os.Clearenv()
	os.Setenv("PATH", "/usr/bin")
	os.Setenv("HOME", "/home/user")
	os.Setenv("SECRET_KEY", "should-not-appear")
	os.Setenv("LANG", "en_US.UTF-8")

	customEnv := map[string]string{
		"PYTHONPATH":   "/custom/path",
		"API_KEY":      "should-be-filtered",
		"CUSTOM_TOKEN": "should-be-filtered",
		"MY_SECRET":    "should-be-filtered",
		"SAFE_VAR":     "should-be-filtered-too", // not in whitelist
		"NODE_OPTIONS": "--max-old-space-size=4096",
	}

	result := sanitizeEnv(customEnv)

	// 結果を検証
	resultMap := make(map[string]string)
	for _, e := range result {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			resultMap[parts[0]] = parts[1]
		}
	}

	// 安全な環境変数が含まれているべき
	if _, ok := resultMap["PATH"]; !ok {
		t.Error("Expected PATH in sanitized env")
	}

	if _, ok := resultMap["HOME"]; !ok {
		t.Error("Expected HOME in sanitized env")
	}

	if _, ok := resultMap["LANG"]; !ok {
		t.Error("Expected LANG in sanitized env")
	}

	// NODE_OPTIONS（カスタム）が含まれているべき
	if val, ok := resultMap["NODE_OPTIONS"]; !ok || val != "--max-old-space-size=4096" {
		t.Errorf("Expected NODE_OPTIONS='--max-old-space-size=4096', got %q", val)
	}

	// 危険な環境変数が除外されているべき
	if _, ok := resultMap["SECRET_KEY"]; ok {
		t.Error("SECRET_KEY should not be in sanitized env")
	}

	if _, ok := resultMap["API_KEY"]; ok {
		t.Error("API_KEY should not be in sanitized env")
	}

	if _, ok := resultMap["CUSTOM_TOKEN"]; ok {
		t.Error("CUSTOM_TOKEN should not be in sanitized env")
	}

	if _, ok := resultMap["MY_SECRET"]; ok {
		t.Error("MY_SECRET should not be in sanitized env")
	}

	// SAFE_VARは KEY/TOKEN/SECRET を含まないので含まれるべき
	if _, ok := resultMap["SAFE_VAR"]; !ok {
		t.Error("SAFE_VAR should be in sanitized env (doesn't contain KEY/TOKEN/SECRET)")
	}

	// PYTHONPATHも含まれるべき
	if _, ok := resultMap["PYTHONPATH"]; !ok {
		t.Error("PYTHONPATH should be in sanitized env")
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}

	if manager.tools == nil {
		t.Error("Expected tools slice to be initialized")
	}

	if manager.healthCheck == nil {
		t.Error("Expected healthCheck map to be initialized")
	}

	if len(manager.sessions) != 0 {
		t.Errorf("Expected empty sessions, got %d", len(manager.sessions))
	}

	if len(manager.tools) != 0 {
		t.Errorf("Expected empty tools, got %d", len(manager.tools))
	}
}

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

func TestManager_GetTools_Empty(t *testing.T) {
	manager := NewManager()
	tools := manager.GetTools()

	if len(tools) != 0 {
		t.Errorf("Expected empty tools, got %d", len(tools))
	}
}

func TestManager_Connect_NoConfig(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	// 設定なしで接続を試みる
	err := manager.Connect(ctx)

	// エラーにならないべき（単に何もしない）
	if err != nil {
		t.Errorf("Connect with no config should not error, got: %v", err)
	}
}

func TestManager_HealthStatus(t *testing.T) {
	manager := NewManager()

	// 何もない状態
	status := manager.HealthStatus()
	if len(status) != 0 {
		t.Errorf("Expected empty health status, got %d entries", len(status))
	}
}

func TestManager_Close(t *testing.T) {
	manager := NewManager()

	// Close should not panic even with no sessions
	manager.Close()
}

func TestManager_CallTool_ServerNotConnected(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	_, err := manager.CallTool(ctx, "nonexistent-server", "some-tool", nil)

	if err == nil {
		t.Error("CallTool should return error for non-connected server")
	}

	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}
}

func TestManager_Close_WithoutSessions(t *testing.T) {
	manager := NewManager()

	// Should not panic
	manager.Close()
}

func TestManager_HealthStatus_WithMockedHealth(t *testing.T) {
	manager := NewManager()

	// 手動でヘルスチェック情報を設定
	manager.healthCheck["test-server"] = time.Now().Add(-5 * time.Minute)

	status := manager.HealthStatus()

	if len(status) != 1 {
		t.Errorf("Expected 1 health status entry, got %d", len(status))
	}

	if _, ok := status["test-server"]; !ok {
		t.Error("Expected test-server in health status")
	}

	// 接続していないので❌が含まれるはず
	if !strings.Contains(status["test-server"], "❌") {
		t.Errorf("Expected ❌ for disconnected server, got: %s", status["test-server"])
	}
}

func TestManager_Connect_InvalidCommand(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	manager := NewManager()

	// ブロックされたコマンドを含む設定
	manager.config = &Config{
		MCPServers: map[string]ServerConfig{
			"blocked-server": {
				Command: "rm", // ブロックされたコマンド
				Args:    []string{"-rf", "/"},
			},
		},
	}

	ctx := context.Background()
	err := manager.Connect(ctx)

	// エラーにならない（ブロックされた接続は警告のみ）
	if err != nil {
		t.Errorf("Connect should not error for blocked commands, got: %v", err)
	}

	// セッションが作成されていないことを確認
	if len(manager.sessions) != 0 {
		t.Errorf("Expected no sessions for blocked command, got %d", len(manager.sessions))
	}
}

func TestSanitizeEnv_EmptyInput(t *testing.T) {
	result := sanitizeEnv(map[string]string{})

	// 最低限の環境変数（PATH, HOME, LANG等）が含まれるべき
	resultMap := make(map[string]string)
	for _, e := range result {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			resultMap[parts[0]] = parts[1]
		}
	}

	// PATH は常に含まれるべき
	if _, ok := resultMap["PATH"]; !ok {
		t.Error("Expected PATH in sanitized env even with empty input")
	}
}

func TestSanitizeEnv_FilterSensitiveKeys(t *testing.T) {
	customEnv := map[string]string{
		"AWS_SECRET_ACCESS_KEY": "sensitive", // SECRET含む
		"API_KEY":               "sensitive", // KEY含む
		"GITHUB_TOKEN":          "sensitive", // TOKEN含む
		"SAFE_VALUE":            "not-sensitive",
	}

	result := sanitizeEnv(customEnv)

	resultMap := make(map[string]string)
	for _, e := range result {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			resultMap[parts[0]] = parts[1]
		}
	}

	// センシティブなキーは除外されるべき（KEY, TOKEN, SECRETを含むもの）
	if _, ok := resultMap["AWS_SECRET_ACCESS_KEY"]; ok {
		t.Error("AWS_SECRET_ACCESS_KEY should be filtered out")
	}

	if _, ok := resultMap["API_KEY"]; ok {
		t.Error("API_KEY should be filtered out")
	}

	if _, ok := resultMap["GITHUB_TOKEN"]; ok {
		t.Error("GITHUB_TOKEN should be filtered out")
	}

	// 安全な値は含まれるべき
	if _, ok := resultMap["SAFE_VALUE"]; !ok {
		t.Error("SAFE_VALUE should be included")
	}
}

func TestValidateMCPCommand_CaseInsensitive(t *testing.T) {
	tests := []struct {
		command string
		wantErr bool
	}{
		{"npx", false},
		{"NPX", true}, // 大文字は許可されない（厳密マッチ）
		{"Node", true}, // 大文字は許可されない
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			err := validateMCPCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPCommand(%q) error = %v, wantErr %v", tt.command, err, tt.wantErr)
			}
		})
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

func TestManager_GetTools_AfterConnect(t *testing.T) {
	manager := NewManager()

	// 手動でツールを追加（実際の接続はスキップ）
	manager.tools = []MCPTool{
		{
			ServerName:  "test-server",
			Name:        "test-tool",
			Description: "A test tool",
		},
	}

	tools := manager.GetTools()

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got %q", tools[0].Name)
	}
}
