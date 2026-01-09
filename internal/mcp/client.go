package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// ServerConfig はMCPサーバーの設定
type ServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Config は ~/.xelyon/mcp.json の構造
type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// MCPTool は外部MCPサーバーから取得したツール
type MCPTool struct {
	ServerName  string
	Name        string
	Description string
	InputSchema json.RawMessage
	Session     *mcp.ClientSession
}

// Manager はMCPサーバーの接続を管理
type Manager struct {
	config      *Config
	sessions    map[string]*mcp.ClientSession
	tools       []MCPTool
	healthCheck map[string]time.Time // サーバーごとの最終正常接続時刻
}

// 安全なMCPコマンドのホワイトリスト
var allowedMCPCommands = map[string]bool{
	"npx":     true,
	"node":    true,
	"python":  true,
	"python3": true,
	"uvx":     true,
	"docker":  true,
}

// NewManager は新しいManagerを作成
func NewManager() *Manager {
	return &Manager{
		sessions:    make(map[string]*mcp.ClientSession),
		tools:       []MCPTool{},
		healthCheck: make(map[string]time.Time),
	}
}

// validateMCPCommand はMCPコマンドの安全性を検証
func validateMCPCommand(command string) error {
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// ホワイトリストチェック
	if !allowedMCPCommands[command] {
		return fmt.Errorf("command '%s' is not in the allowed list. Allowed: npx, node, python, python3, uvx, docker", command)
	}

	// パストラバーサルチェック
	if strings.Contains(command, "..") || strings.Contains(command, "/") {
		return fmt.Errorf("command contains path traversal characters")
	}

	return nil
}

// sanitizeEnv は環境変数を安全なもののみに制限
func sanitizeEnv(customEnv map[string]string) []string {
	// 安全な環境変数のホワイトリスト
	safeEnvKeys := map[string]bool{
		"PATH":         true,
		"HOME":         true,
		"USER":         true,
		"LANG":         true,
		"LC_ALL":       true,
		"NODE_OPTIONS": true,
		"PYTHONPATH":   true,
	}

	env := []string{}

	// システム環境変数から安全なもののみコピー
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && safeEnvKeys[parts[0]] {
			env = append(env, e)
		}
	}

	// カスタム環境変数を追加（APIキー系は除外）
	for k, v := range customEnv {
		// APIキー系の環境変数は除外
		if strings.Contains(strings.ToUpper(k), "KEY") ||
			strings.Contains(strings.ToUpper(k), "TOKEN") ||
			strings.Contains(strings.ToUpper(k), "SECRET") {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// LoadConfig は ~/.xelyon/mcp.json を読み込む
func (m *Manager) LoadConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".xelyon", "mcp.json")

	// .xelyonディレクトリが存在しない場合は作成
	xelyonDir := filepath.Join(homeDir, ".xelyon")
	if _, err := os.Stat(xelyonDir); os.IsNotExist(err) {
		if err := os.MkdirAll(xelyonDir, 0755); err != nil {
			return fmt.Errorf("failed to create .xelyon directory: %w", err)
		}
	}

	// ファイルが存在しない場合はデフォルト設定を作成
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := Config{
			MCPServers: map[string]ServerConfig{
				// 例: filesystemサーバー
				"filesystem": {
					Command: "npx",
					Args:    []string{"@modelcontextprotocol/server-filesystem", "/path/to/directory"},
					Env:     map[string]string{"NODE_OPTIONS": "--no-warnings"},
				},
			},
		}

		data, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal default config: %w", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}

		fmt.Printf("📄 Created default MCP config at %s\n", configPath)
		fmt.Println("ℹ️  Please edit the config file to add your MCP servers")
		m.config = &defaultConfig
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	m.config = &config
	return nil
}

// Connect は全てのMCPサーバーに接続
func (m *Manager) Connect(ctx context.Context) error {
	if m.config == nil || len(m.config.MCPServers) == 0 {
		return nil
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "xelyon-cli",
		Version: version.Version,
	}, nil)

	for name, serverConfig := range m.config.MCPServers {
		// コマンドの安全性を検証
		if err := validateMCPCommand(serverConfig.Command); err != nil {
			fmt.Printf("⚠️  MCP server '%s' blocked: %v\n", name, err)
			continue
		}

		cmd := exec.Command(serverConfig.Command, serverConfig.Args...)

		// 環境変数を安全なもののみに制限
		cmd.Env = sanitizeEnv(serverConfig.Env)

		transport := &mcp.CommandTransport{Command: cmd}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			// 接続失敗は警告のみ、続行
			fmt.Printf("⚠️  MCP server '%s' connection failed: %v\n", name, err)
			continue
		}

		m.sessions[name] = session

		// ツール一覧を取得
		toolsResult, err := session.ListTools(ctx, nil)
		if err != nil {
			fmt.Printf("⚠️  Failed to list tools from '%s': %v\n", name, err)
			continue
		}

		for _, tool := range toolsResult.Tools {
			schemaBytes, _ := json.Marshal(tool.InputSchema)
			m.tools = append(m.tools, MCPTool{
				ServerName:  name,
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: schemaBytes,
				Session:     session,
			})
		}

		fmt.Printf("🔌 MCP server '%s' connected (%d tools)\n", name, len(toolsResult.Tools))
	}

	return nil
}

// GetTools は利用可能なMCPツール一覧を返す
func (m *Manager) GetTools() []MCPTool {
	return m.tools
}

// CallTool はMCPツールを呼び出す（リトライ付き）
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (string, error) {
	session, ok := m.sessions[serverName]
	if !ok {
		return "", fmt.Errorf("MCP server '%s' not connected", serverName)
	}

	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}

	// 最大2回リトライ
	var result *mcp.CallToolResult
	var callErr error

	for attempt := 1; attempt <= 2; attempt++ {
		result, callErr = session.CallTool(ctx, params)
		if callErr == nil && !result.IsError {
			break // 成功
		}

		if attempt < 2 {
			errMsg := ""
			if callErr != nil {
				errMsg = callErr.Error()
			} else if result.IsError {
				errMsg = "tool returned error"
			}
			fmt.Printf("⚠️  MCP tool '%s' call attempt %d failed: %s (retrying...)\n", toolName, attempt, errMsg)
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}
	}

	if callErr != nil {
		return "", fmt.Errorf("failed to call tool after 2 attempts: %w", callErr)
	}

	if result.IsError {
		// エラーメッセージを抽出
		errMsg := "tool returned error"
		for _, content := range result.Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				errMsg = textContent.Text
				break
			}
		}
		return "", fmt.Errorf(errMsg)
	}

	// 結果をテキストに変換
	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			output += textContent.Text + "\n"
		}
	}

	return output, nil
}

// Close は全ての接続を閉じる
func (m *Manager) Close() {
	for _, session := range m.sessions {
		session.Close()
	}
}

// HealthStatus はサーバーのヘルスステータスを返す
func (m *Manager) HealthStatus() map[string]string {
	status := make(map[string]string)

	for serverName, lastHealthy := range m.healthCheck {
		connected := "❌"
		if _, ok := m.sessions[serverName]; ok {
			connected = "✅"
		}

		// 最終正常接続からの経過時間
		elapsed := time.Since(lastHealthy)
		status[serverName] = fmt.Sprintf("%s Last healthy: %v ago", connected, elapsed.Round(time.Second))
	}

	return status
}

// Reconnect は指定されたサーバーに再接続する
func (m *Manager) Reconnect(ctx context.Context, serverName string) error {
	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	serverConfig, ok := m.config.MCPServers[serverName]
	if !ok {
		return fmt.Errorf("server '%s' not found in config", serverName)
	}

	// 既存のセッションを閉じる
	if session, ok := m.sessions[serverName]; ok {
		defer session.Close()
		delete(m.sessions, serverName)
	}

	// ツールリストから削除
	var filteredTools []MCPTool
	for _, tool := range m.tools {
		if tool.ServerName != serverName {
			filteredTools = append(filteredTools, tool)
		}
	}
	m.tools = filteredTools

	// 再接続
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "xelyon-cli",
		Version: version.Version,
	}, nil)

	cmd := exec.Command(serverConfig.Command, serverConfig.Args...)
	if len(serverConfig.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range serverConfig.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("reconnection failed: %w", err)
	}

	m.sessions[serverName] = session

	// ツール一覧を取得
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	for _, tool := range toolsResult.Tools {
		schemaBytes, _ := json.Marshal(tool.InputSchema)
		m.tools = append(m.tools, MCPTool{
			ServerName:  serverName,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schemaBytes,
			Session:     session,
		})
	}

	m.healthCheck[serverName] = time.Now()
	fmt.Printf("🔌 MCP server '%s' reconnected (%d tools)\n", serverName, len(toolsResult.Tools))
	return nil
}
