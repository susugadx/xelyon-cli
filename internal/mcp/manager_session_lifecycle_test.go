package mcp

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

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

	if manager.out() != io.Discard {
		t.Fatalf("expected default output to be io.Discard")
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

func TestManagerConnectSkippedAndFailedServersDoNotDirtyState(t *testing.T) {
	manager := NewManager()
	manager.config = &Config{
		MCPServers: map[string]ServerConfig{
			"blocked-server": {
				Command: "bash",
				Args:    []string{"-lc", "echo blocked"},
			},
			"disabled-server": {
				Command:  "npx",
				Disabled: true,
			},
			"failing-server": {
				Command: "python3",
				Args:    []string{"-c", "import sys; sys.exit(1)"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := manager.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v, want nil for per-server failures", err)
	}

	if len(manager.sessions) != 0 {
		t.Fatalf("sessions = %d, want 0", len(manager.sessions))
	}
	if len(manager.tools) != 0 {
		t.Fatalf("tools = %d, want 0", len(manager.tools))
	}
	if len(manager.healthCheck) != 0 {
		t.Fatalf("healthCheck = %d, want 0", len(manager.healthCheck))
	}
	if status := manager.HealthStatus(); len(status) != 0 {
		t.Fatalf("HealthStatus() = %#v, want empty", status)
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

func TestManager_SetOutput_NilFallsBackToDiscard(t *testing.T) {
	manager := NewManager()
	manager.SetOutput(nil)
	if manager.out() != io.Discard {
		t.Fatalf("out() should fall back to io.Discard when output is nil")
	}
}

func TestManager_Reconnect_NoConfig(t *testing.T) {
	manager := NewManager()
	err := manager.Reconnect(context.Background(), "github")
	if err == nil || !strings.Contains(err.Error(), "no configuration loaded") {
		t.Fatalf("Reconnect() error = %v, want no configuration loaded", err)
	}
}

func TestManager_Reconnect_ServerNotFound(t *testing.T) {
	manager := NewManager()
	manager.config = &Config{MCPServers: map[string]ServerConfig{}}

	err := manager.Reconnect(context.Background(), "github")
	if err == nil || !strings.Contains(err.Error(), "not found in config") {
		t.Fatalf("Reconnect() error = %v, want server not found", err)
	}
}

func TestManager_Reconnect_DisabledServer(t *testing.T) {
	manager := NewManager()
	manager.config = &Config{
		MCPServers: map[string]ServerConfig{
			"github": {
				Command:  "npx",
				Disabled: true,
			},
		},
	}

	err := manager.Reconnect(context.Background(), "github")
	if err == nil || !strings.Contains(err.Error(), "is disabled") {
		t.Fatalf("Reconnect() error = %v, want disabled server error", err)
	}
}
