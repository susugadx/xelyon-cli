package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// NewManager は新しいManagerを作成
func NewManager() *Manager {
	return &Manager{
		sessions:    make(map[string]*mcp.ClientSession),
		tools:       []MCPTool{},
		healthCheck: make(map[string]time.Time),
		output:      io.Discard,
	}
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

	serverNames := make([]string, 0, len(m.config.MCPServers))
	for name := range m.config.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		serverConfig := m.config.MCPServers[name]
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

		serverTools, summary, err := m.refreshServerTools(ctx, name, session, serverConfig)
		if err != nil {
			_ = session.Close()
			fmt.Fprintf(m.out(), "⚠️  Failed to list tools from '%s': %v\n", name, err)
			continue
		}

		previous := m.swapServerSession(name, session)
		m.replaceServerTools(name, serverTools)
		if previous != nil && previous != session {
			_ = previous.Close()
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
	connectCtx, cancel := mcpServerOperationContext(ctx, serverConfig.startupTimeoutDuration())
	defer cancel()
	return client.Connect(connectCtx, transport, nil)
}

func mcpServerOperationContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = defaultMCPServerOperationTimeout
	}
	return context.WithTimeout(ctx, timeout)
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

// Close は全ての接続を閉じる
func (m *Manager) Close() {
	for serverName := range m.sessions {
		m.removeServerSession(serverName)
	}
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

	client := m.newClient()
	session, err := m.openServerSession(ctx, client, serverConfig)
	if err != nil {
		return fmt.Errorf("reconnection failed: %w", err)
	}

	serverTools, summary, err := m.refreshServerTools(ctx, serverName, session, serverConfig)
	if err != nil {
		_ = session.Close()
		return fmt.Errorf("failed to list tools: %w", err)
	}

	previous := m.swapServerSession(serverName, session)
	m.replaceServerTools(serverName, serverTools)
	if previous != nil && previous != session {
		_ = previous.Close()
	}

	m.markServerHealthy(serverName)
	m.printServerConnectionStatus(serverName, summary, true)
	return nil
}
