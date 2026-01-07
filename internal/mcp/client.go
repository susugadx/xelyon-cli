package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	config   *Config
	sessions map[string]*mcp.ClientSession
	tools    []MCPTool
}

// NewManager は新しいManagerを作成
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*mcp.ClientSession),
		tools:    []MCPTool{},
	}
}

// LoadConfig は ~/.xelyon/mcp.json を読み込む
func (m *Manager) LoadConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".xelyon", "mcp.json")

	// ファイルが存在しない場合はスキップ
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
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
		Version: "0.12.0",
	}, nil)

	for name, serverConfig := range m.config.MCPServers {
		cmd := exec.Command(serverConfig.Command, serverConfig.Args...)

		// 環境変数を設定
		if len(serverConfig.Env) > 0 {
			cmd.Env = os.Environ()
			for k, v := range serverConfig.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}

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

// CallTool はMCPツールを呼び出す
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (string, error) {
	session, ok := m.sessions[serverName]
	if !ok {
		return "", fmt.Errorf("MCP server '%s' not connected", serverName)
	}

	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		return "", err
	}

	if result.IsError {
		return "", fmt.Errorf("tool returned error")
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
