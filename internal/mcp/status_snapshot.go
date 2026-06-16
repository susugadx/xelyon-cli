package mcp

import (
	"sort"
	"time"

	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
)

// ServerStatusState は対話中 MCP status で表示する server 状態を表す。
type ServerStatusState string

const (
	// ServerStatusConnected は server session が現在 manager に登録されている状態。
	ServerStatusConnected ServerStatusState = "connected"
	// ServerStatusDisabled は config で server が無効化されている状態。
	ServerStatusDisabled ServerStatusState = "disabled"
	// ServerStatusBlocked は command allowlist により server 起動が拒否される状態。
	ServerStatusBlocked ServerStatusState = "blocked"
	// ServerStatusNotConnected は有効な server だが現在 session がない状態。
	ServerStatusNotConnected ServerStatusState = "not_connected"
)

// StatusSnapshot は現在の MCP runtime state の副作用なし snapshot。
type StatusSnapshot struct {
	ConfigLoaded            bool
	ServerCount             int
	ConnectedServerCount    int
	DisabledServerCount     int
	BlockedServerCount      int
	NotConnectedServerCount int
	RegisteredToolCount     int
	Servers                 []ServerStatusSnapshot
	Tools                   []ToolStatusSnapshot
}

// ServerStatusSnapshot は MCP server 単位の sanitized runtime state。
type ServerStatusSnapshot struct {
	Name                  string
	State                 ServerStatusState
	Approval              string
	ApprovalValid         bool
	StartupTimeoutSeconds int
	ToolTimeoutSeconds    int
	RegisteredToolCount   int
	LastHealthy           time.Time
	LastHealthySet        bool
}

// ToolStatusSnapshot は MCP tool 単位の sanitized runtime state。
type ToolStatusSnapshot struct {
	ServerName   string
	Name         string
	ExportedName string
	Approval     string
}

// StatusSnapshot は MCP manager が保持する現在状態を副作用なしで返す。
func (m *Manager) StatusSnapshot() StatusSnapshot {
	if m == nil {
		return StatusSnapshot{}
	}

	snapshot := StatusSnapshot{
		ConfigLoaded:        m.config != nil,
		RegisteredToolCount: len(m.tools),
		Tools:               m.statusToolSnapshots(),
	}
	if m.config == nil {
		return snapshot
	}

	registeredByServer := make(map[string]int, len(m.config.MCPServers))
	for _, tool := range m.tools {
		registeredByServer[tool.ServerName]++
	}

	serverNames := make([]string, 0, len(m.config.MCPServers))
	for name := range m.config.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		serverConfig := m.config.MCPServers[name]
		server := m.statusServerSnapshot(name, serverConfig, registeredByServer[name])
		snapshot.Servers = append(snapshot.Servers, server)
		snapshot.ServerCount++
		switch server.State {
		case ServerStatusConnected:
			snapshot.ConnectedServerCount++
		case ServerStatusDisabled:
			snapshot.DisabledServerCount++
		case ServerStatusBlocked:
			snapshot.BlockedServerCount++
		case ServerStatusNotConnected:
			snapshot.NotConnectedServerCount++
		}
	}

	return snapshot
}

func (m *Manager) statusServerSnapshot(name string, serverConfig ServerConfig, registeredTools int) ServerStatusSnapshot {
	approval, approvalValid := mcpapproval.Normalize(serverConfig.Approval)
	server := ServerStatusSnapshot{
		Name:                  name,
		State:                 ServerStatusNotConnected,
		Approval:              approval.String(),
		ApprovalValid:         approvalValid,
		StartupTimeoutSeconds: int(serverConfig.startupTimeoutDuration() / time.Second),
		ToolTimeoutSeconds:    int(serverConfig.toolTimeoutDuration() / time.Second),
		RegisteredToolCount:   registeredTools,
	}
	if lastHealthy, ok := m.healthCheck[name]; ok {
		server.LastHealthy = lastHealthy
		server.LastHealthySet = true
	}

	switch {
	case serverConfig.Disabled:
		server.State = ServerStatusDisabled
	case validateMCPCommand(serverConfig.Command) != nil:
		server.State = ServerStatusBlocked
	case m.hasServerSession(name):
		server.State = ServerStatusConnected
	}
	return server
}

func (m *Manager) statusToolSnapshots() []ToolStatusSnapshot {
	if len(m.tools) == 0 {
		return nil
	}
	tools := make([]ToolStatusSnapshot, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, ToolStatusSnapshot{
			ServerName:   tool.ServerName,
			Name:         tool.Name,
			ExportedName: mcpnames.ExportedToolName(tool.ServerName, tool.Name),
			Approval:     tool.ApprovalMode().String(),
		})
	}
	sort.SliceStable(tools, func(i, j int) bool {
		if tools[i].ServerName != tools[j].ServerName {
			return tools[i].ServerName < tools[j].ServerName
		}
		if tools[i].Name != tools[j].Name {
			return tools[i].Name < tools[j].Name
		}
		return tools[i].ExportedName < tools[j].ExportedName
	})
	return tools
}

func (m *Manager) hasServerSession(serverName string) bool {
	if m == nil || m.sessions == nil {
		return false
	}
	_, ok := m.sessions[serverName]
	return ok
}
