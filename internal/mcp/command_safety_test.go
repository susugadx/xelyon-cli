package mcp

import (
	"os"
	"strings"
	"testing"
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
		"API_KEY":      "my-api-key",
		"CUSTOM_TOKEN": "my-token",
		"MY_SECRET":    "my-secret",
		"SAFE_VAR":     "safe-value",
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

	// customEnvの値はすべて追加されるべき
	if val, ok := resultMap["API_KEY"]; !ok || val != "my-api-key" {
		t.Errorf("Expected API_KEY='my-api-key', got %q", val)
	}

	if val, ok := resultMap["CUSTOM_TOKEN"]; !ok || val != "my-token" {
		t.Errorf("Expected CUSTOM_TOKEN='my-token', got %q", val)
	}

	if val, ok := resultMap["MY_SECRET"]; !ok || val != "my-secret" {
		t.Errorf("Expected MY_SECRET='my-secret', got %q", val)
	}

	if val, ok := resultMap["SAFE_VAR"]; !ok || val != "safe-value" {
		t.Errorf("Expected SAFE_VAR='safe-value', got %q", val)
	}

	// PYTHONPATHも含まれるべき
	if val, ok := resultMap["PYTHONPATH"]; !ok || val != "/custom/path" {
		t.Errorf("Expected PYTHONPATH='/custom/path', got %q", val)
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

func TestSanitizeEnv_AllCustomEnvIncluded(t *testing.T) {
	// customEnvの値はすべて追加されるべき（フィルタリングなし）
	customEnv := map[string]string{
		"AWS_SECRET_ACCESS_KEY": "sensitive",
		"API_KEY":               "my-api-key",
		"GITHUB_TOKEN":          "my-token",
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

	// すべてのcustomEnv値が含まれるべき
	if val, ok := resultMap["AWS_SECRET_ACCESS_KEY"]; !ok || val != "sensitive" {
		t.Errorf("Expected AWS_SECRET_ACCESS_KEY='sensitive', got %q", val)
	}

	if val, ok := resultMap["API_KEY"]; !ok || val != "my-api-key" {
		t.Errorf("Expected API_KEY='my-api-key', got %q", val)
	}

	if val, ok := resultMap["GITHUB_TOKEN"]; !ok || val != "my-token" {
		t.Errorf("Expected GITHUB_TOKEN='my-token', got %q", val)
	}

	if val, ok := resultMap["SAFE_VALUE"]; !ok || val != "not-sensitive" {
		t.Errorf("Expected SAFE_VALUE='not-sensitive', got %q", val)
	}
}

func TestValidateMCPCommand_CaseInsensitive(t *testing.T) {
	tests := []struct {
		command string
		wantErr bool
	}{
		{"npx", false},
		{"NPX", true},  // 大文字は許可されない（厳密マッチ）
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
