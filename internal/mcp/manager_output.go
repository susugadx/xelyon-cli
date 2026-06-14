package mcp

import (
	"fmt"
	"io"
	"time"
)

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

func (m *Manager) printServerConnectionStatus(serverName string, summary toolRegistrationSummary, reconnect bool) {
	status := "connected"
	if reconnect {
		status = "reconnected"
	}

	if summary.skipped > 0 {
		fmt.Fprintf(m.out(), "🔌 MCP server '%s' %s (%d tools, %d skipped)\n", serverName, status, summary.registered, summary.skipped)
		return
	}
	fmt.Fprintf(m.out(), "🔌 MCP server '%s' %s (%d tools)\n", serverName, status, summary.registered)
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
