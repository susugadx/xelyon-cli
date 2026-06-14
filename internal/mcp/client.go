package mcp

import (
	"encoding/json"
	"io"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConfig はMCPサーバーの設定
type ServerConfig struct {
	Command               string            `json:"command"`
	Args                  []string          `json:"args,omitempty"`
	Env                   map[string]string `json:"env,omitempty"`
	Disabled              bool              `json:"disabled,omitempty"`              // サーバー全体を無効化
	Tools                 *ToolsFilter      `json:"tools,omitempty"`                 // ツール単位フィルタ
	StartupTimeoutSeconds int               `json:"startupTimeoutSeconds,omitempty"` // 起動・接続・tools/list timeout
	ToolTimeoutSeconds    int               `json:"toolTimeoutSeconds,omitempty"`    // tools/call timeout
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
	CallTimeout time.Duration
}

// Manager はMCPサーバーの接続を管理
type Manager struct {
	config      *Config
	sessions    map[string]*mcp.ClientSession
	tools       []MCPTool
	healthCheck map[string]time.Time // サーバーごとの最終正常接続時刻
	output      io.Writer
}

const (
	defaultMCPServerOperationTimeout = 120 * time.Second
	defaultMCPToolCallTimeout        = 600 * time.Second
	maxMCPServerOperationTimeout     = 3600 * time.Second
	maxMCPToolCallTimeout            = 3600 * time.Second
	toolCallMaxAttempts              = 2
)

type toolRegistrationSummary struct {
	registered int
	skipped    int
}

func (c ServerConfig) startupTimeoutDuration() time.Duration {
	return mcpTimeoutDuration(c.StartupTimeoutSeconds, defaultMCPServerOperationTimeout, maxMCPServerOperationTimeout)
}

func (c ServerConfig) toolTimeoutDuration() time.Duration {
	return mcpTimeoutDuration(c.ToolTimeoutSeconds, defaultMCPToolCallTimeout, maxMCPToolCallTimeout)
}

func mcpTimeoutDuration(seconds int, fallback, max time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	maxSeconds := int(max / time.Second)
	if seconds > maxSeconds {
		return max
	}
	return time.Duration(seconds) * time.Second
}
