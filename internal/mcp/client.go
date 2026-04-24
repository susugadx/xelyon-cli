package mcp

import (
	"encoding/json"
	"io"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

const (
	toolCallMaxAttempts = 2
)

type toolRegistrationSummary struct {
	registered int
	skipped    int
}
