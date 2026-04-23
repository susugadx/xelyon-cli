package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

const (
	toolCallMaxAttempts = 2
)

type toolRegistrationSummary struct {
	registered int
	skipped    int
}

// NewManager は新しいManagerを作成
func NewManager() *Manager {
	return &Manager{
		sessions:    make(map[string]*mcp.ClientSession),
		tools:       []MCPTool{},
		healthCheck: make(map[string]time.Time),
		output:      io.Discard,
	}
}

// SetOutput は Manager の出力先を設定する。
func (m *Manager) SetOutput(w io.Writer) {
	m.output = w
}

// out は Manager の出力先を返す。未設定時は出力を抑制する。
func (m *Manager) out() io.Writer {
	if m.output != nil {
		return m.output
	}
	return io.Discard
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

	loadResult, err := loadMCPConfig(homeDir)
	if err != nil {
		return err
	}

	if loadResult.createdDefault {
		fmt.Fprintf(m.out(), "📄 Created default MCP config at %s\n", loadResult.configPath)
		fmt.Fprintln(m.out(), "ℹ️  Please edit the config file to add your MCP servers")
	}

	m.config = loadResult.config
	return nil
}

// Connect は全てのMCPサーバーに接続
func (m *Manager) Connect(ctx context.Context) error {
	if m.config == nil || len(m.config.MCPServers) == 0 {
		return nil
	}

	client := m.newClient()

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

		session, err := m.openServerSession(ctx, client, serverConfig)
		if err != nil {
			// 接続失敗は警告のみ、続行
			fmt.Fprintf(m.out(), "⚠️  MCP server '%s' connection failed: %v\n", name, err)
			continue
		}

		previous := m.swapServerSession(name, session)
		if previous != nil && previous != session {
			_ = previous.Close()
		}

		summary, err := m.refreshServerTools(ctx, name, session, serverConfig.Tools, true)
		if err != nil {
			fmt.Fprintf(m.out(), "⚠️  Failed to list tools from '%s': %v\n", name, err)
			continue
		}

		m.markServerHealthy(name)
		m.printServerConnectionStatus(name, summary, false)
	}

	return nil
}

func (m *Manager) newClient() *mcp.Client {
	return mcp.NewClient(&mcp.Implementation{
		Name:    "xelyon-cli",
		Version: version.Version,
	}, nil)
}

func (m *Manager) openServerSession(ctx context.Context, client *mcp.Client, serverConfig ServerConfig) (*mcp.ClientSession, error) {
	cmd := exec.Command(serverConfig.Command, serverConfig.Args...)
	cmd.Env = sanitizeEnv(serverConfig.Env)

	transport := &mcp.CommandTransport{Command: cmd}
	return client.Connect(ctx, transport, nil)
}

func (m *Manager) swapServerSession(serverName string, session *mcp.ClientSession) *mcp.ClientSession {
	previous := m.sessions[serverName]
	m.sessions[serverName] = session
	return previous
}

func (m *Manager) removeServerSession(serverName string) {
	session, ok := m.sessions[serverName]
	if !ok {
		return
	}

	_ = session.Close()
	delete(m.sessions, serverName)
}

func (m *Manager) markServerHealthy(serverName string) {
	m.healthCheck[serverName] = time.Now()
}

func (m *Manager) refreshServerTools(
	ctx context.Context,
	serverName string,
	session *mcp.ClientSession,
	filter *ToolsFilter,
	replace bool,
) (toolRegistrationSummary, error) {
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return toolRegistrationSummary{}, err
	}

	summary := m.storeServerTools(serverName, session, toolsResult.Tools, filter, replace)
	return summary, nil
}

func (m *Manager) storeServerTools(
	serverName string,
	session *mcp.ClientSession,
	toolDefs []*mcp.Tool,
	filter *ToolsFilter,
	replace bool,
) toolRegistrationSummary {
	if replace {
		m.removeServerTools(serverName)
	}

	summary := toolRegistrationSummary{}
	for _, tool := range toolDefs {
		if tool == nil {
			continue
		}
		if !shouldIncludeTool(tool.Name, filter) {
			summary.skipped++
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
		summary.registered++
	}

	return summary
}

func (m *Manager) removeServerTools(serverName string) {
	filteredTools := m.tools[:0]
	for _, tool := range m.tools {
		if tool.ServerName != serverName {
			filteredTools = append(filteredTools, tool)
		}
	}
	m.tools = filteredTools
}

func (m *Manager) printServerConnectionStatus(serverName string, summary toolRegistrationSummary, reconnect bool) {
	status := "connected"
	if reconnect {
		status = "reconnected"
	}

	if summary.skipped > 0 {
		fmt.Fprintf(m.out(), "🔌 MCP server '%s' %s (%d tools, %d filtered out)\n", serverName, status, summary.registered, summary.skipped)
		return
	}
	fmt.Fprintf(m.out(), "🔌 MCP server '%s' %s (%d tools)\n", serverName, status, summary.registered)
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

	result, err := m.callToolWithRetry(ctx, session, toolName, params)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("%s", toolResultErrorMessage(result))
	}

	return toolResultText(result), nil
}

func (m *Manager) callToolWithRetry(
	ctx context.Context,
	session *mcp.ClientSession,
	toolName string,
	params *mcp.CallToolParams,
) (*mcp.CallToolResult, error) {
	var result *mcp.CallToolResult
	var callErr error
	attempted := 0

	for attempt := 1; attempt <= toolCallMaxAttempts; attempt++ {
		attempted = attempt
		result, callErr = session.CallTool(ctx, params)
		if callErr == nil && result != nil && !result.IsError {
			return result, nil
		}
		if !shouldRetryToolCall(ctx, attempt, callErr, result) {
			break
		}

		fmt.Fprintf(
			m.out(),
			"⚠️  MCP tool '%s' call attempt %d failed: %s (retrying...)\n",
			toolName,
			attempt,
			toolCallFailureReason(callErr, result),
		)
		time.Sleep(toolRetryDelay(attempt))
	}

	if callErr != nil {
		return nil, fmt.Errorf("failed to call tool after %d attempts: %w", attempted, callErr)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to call tool after %d attempts: empty response", attempted)
	}
	return result, nil
}

func shouldRetryToolCall(ctx context.Context, attempt int, callErr error, result *mcp.CallToolResult) bool {
	if attempt >= toolCallMaxAttempts {
		return false
	}
	if ctx.Err() != nil {
		return false
	}
	if callErr != nil {
		if errors.Is(callErr, context.Canceled) || errors.Is(callErr, context.DeadlineExceeded) {
			return false
		}
		return true
	}
	return result != nil && result.IsError
}

func toolRetryDelay(attempt int) time.Duration {
	shift := min(attempt-1, 30)
	return time.Duration(1<<uint(shift)) * time.Second
}

func toolCallFailureReason(callErr error, result *mcp.CallToolResult) string {
	if callErr != nil {
		return callErr.Error()
	}
	if result != nil && result.IsError {
		return toolResultErrorMessage(result)
	}
	return "tool returned error"
}

func toolResultErrorMessage(result *mcp.CallToolResult) string {
	errMsg := "tool returned error"
	if result == nil {
		return errMsg
	}
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			errMsg = textContent.Text
			break
		}
	}
	return errMsg
}

func toolResultText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	var output strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			output.WriteString(textContent.Text)
			output.WriteByte('\n')
		}
	}
	return output.String()
}

// Close は全ての接続を閉じる
func (m *Manager) Close() {
	for serverName := range m.sessions {
		m.removeServerSession(serverName)
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

	if err := validateMCPCommand(serverConfig.Command); err != nil {
		return fmt.Errorf("server '%s' blocked: %w", serverName, err)
	}

	m.removeServerSession(serverName)
	m.removeServerTools(serverName)

	client := m.newClient()
	session, err := m.openServerSession(ctx, client, serverConfig)
	if err != nil {
		return fmt.Errorf("reconnection failed: %w", err)
	}

	previous := m.swapServerSession(serverName, session)
	if previous != nil && previous != session {
		_ = previous.Close()
	}

	summary, err := m.refreshServerTools(ctx, serverName, session, serverConfig.Tools, true)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	m.markServerHealthy(serverName)
	m.printServerConnectionStatus(serverName, summary, true)
	return nil
}
