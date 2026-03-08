package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Command  string            `json:"command"`
	Args     []string          `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Disabled bool              `json:"disabled,omitempty"` // サーバー全体を無効化
	Tools    *ToolsFilter      `json:"tools,omitempty"`    // ツール単位フィルタ
}

// ToolsFilter はツールのinclude/excludeフィルタ
type ToolsFilter struct {
	Include []string `json:"include,omitempty"` // ホワイトリスト（指定時はこれだけ有効）
	Exclude []string `json:"exclude,omitempty"` // ブラックリスト（include未指定時に使用）
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
	output      io.Writer
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

// SetOutput は Manager の出力先を設定する。
func (m *Manager) SetOutput(w io.Writer) {
	m.output = w
}

// out は Manager の出力先を返す。nil なら os.Stdout へ fallback する。
func (m *Manager) out() io.Writer {
	if m.output != nil {
		return m.output
	}
	return os.Stdout
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

// sanitizeEnv は環境変数を構築する
// システム環境変数から安全なものをコピーし、customEnvの値をすべて追加する
// customEnvの値は ${VAR} 形式で環境変数を参照できる
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

	// カスタム環境変数をすべて追加（${VAR} を展開）
	for k, v := range customEnv {
		expandedValue := os.ExpandEnv(v)
		env = append(env, fmt.Sprintf("%s=%s", k, expandedValue))
	}

	return env
}

// shouldIncludeTool はツールがフィルタを通過するか判定
// - include が設定されている場合: include に含まれるツールのみ通過（ホワイトリスト）
// - include 未設定で exclude が設定されている場合: exclude に含まれないツールが通過（ブラックリスト）
// - どちらも未設定: 全ツール通過
func shouldIncludeTool(toolName string, filter *ToolsFilter) bool {
	if filter == nil {
		return true
	}

	if len(filter.Include) > 0 {
		for _, name := range filter.Include {
			if name == toolName {
				return true
			}
		}
		return false
	}

	if len(filter.Exclude) > 0 {
		for _, name := range filter.Exclude {
			if name == toolName {
				return false
			}
		}
	}

	return true
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

		fmt.Fprintf(m.out(), "📄 Created default MCP config at %s\n", configPath)
		fmt.Fprintln(m.out(), "ℹ️  Please edit the config file to add your MCP servers")
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
		if serverConfig.Disabled {
			fmt.Fprintf(m.out(), "⏭️  MCP server '%s' is disabled, skipping\n", name)
			continue
		}

		// コマンドの安全性を検証
		if err := validateMCPCommand(serverConfig.Command); err != nil {
			fmt.Fprintf(m.out(), "⚠️  MCP server '%s' blocked: %v\n", name, err)
			continue
		}

		cmd := exec.Command(serverConfig.Command, serverConfig.Args...)

		// 環境変数を安全なもののみに制限
		cmd.Env = sanitizeEnv(serverConfig.Env)

		transport := &mcp.CommandTransport{Command: cmd}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			// 接続失敗は警告のみ、続行
			fmt.Fprintf(m.out(), "⚠️  MCP server '%s' connection failed: %v\n", name, err)
			continue
		}

		m.sessions[name] = session

		// ツール一覧を取得
		toolsResult, err := session.ListTools(ctx, nil)
		if err != nil {
			fmt.Fprintf(m.out(), "⚠️  Failed to list tools from '%s': %v\n", name, err)
			continue
		}

		registered := 0
		skipped := 0
		for _, tool := range toolsResult.Tools {
			if !shouldIncludeTool(tool.Name, serverConfig.Tools) {
				skipped++
				continue
			}
			schemaBytes, _ := json.Marshal(tool.InputSchema)
			m.tools = append(m.tools, MCPTool{
				ServerName:  name,
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: schemaBytes,
				Session:     session,
			})
			registered++
		}

		if skipped > 0 {
			fmt.Fprintf(m.out(), "🔌 MCP server '%s' connected (%d tools, %d filtered out)\n", name, registered, skipped)
		} else {
			fmt.Fprintf(m.out(), "🔌 MCP server '%s' connected (%d tools)\n", name, registered)
		}
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
			fmt.Fprintf(m.out(), "⚠️  MCP tool '%s' call attempt %d failed: %s (retrying...)\n", toolName, attempt, errMsg)
			shift := min(attempt-1, 30)
			time.Sleep(time.Duration(1<<uint(shift)) * time.Second)
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
		return "", fmt.Errorf("%s", errMsg)
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
	if serverConfig.Disabled {
		return fmt.Errorf("server '%s' is disabled", serverName)
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

	registered := 0
	skipped := 0
	for _, tool := range toolsResult.Tools {
		if !shouldIncludeTool(tool.Name, serverConfig.Tools) {
			skipped++
			continue
		}
		schemaBytes, _ := json.Marshal(tool.InputSchema)
		m.tools = append(m.tools, MCPTool{
			ServerName:  serverName,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schemaBytes,
			Session:     session,
		})
		registered++
	}

	m.healthCheck[serverName] = time.Now()
	if skipped > 0 {
		fmt.Fprintf(m.out(), "🔌 MCP server '%s' reconnected (%d tools, %d filtered out)\n", serverName, registered, skipped)
	} else {
		fmt.Fprintf(m.out(), "🔌 MCP server '%s' reconnected (%d tools)\n", serverName, registered)
	}
	return nil
}
