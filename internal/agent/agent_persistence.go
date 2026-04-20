package agent

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) appendSessionMessage(role, content, model string) {
	if a == nil || a.session == nil {
		return
	}
	a.invalidateSavedResponseContextForCurrentRuntime()
	a.session.AddMessage(role, content, model)
	a.persistSession()
}

func (a *Agent) appendSessionMessageFromAPI(msg api.Message, model string) {
	if a == nil || a.session == nil {
		return
	}
	a.invalidateSavedResponseContextForCurrentRuntime()
	a.session.AddMessageFromAPI(msg, model)
	a.persistSession()
}

func (a *Agent) appendSessionToolExecution(toolCall *tools.ToolCall, result string) {
	if a == nil || a.session == nil || toolCall == nil {
		return
	}
	a.invalidateSavedResponseContextForCurrentRuntime()
	success := !strings.HasPrefix(strings.TrimSpace(result), "Error:")
	a.session.AddToolExecution(toolCall.Tool, toolCall.Args, result, success, a.CurrentModel)
	a.persistSession()
}

func (a *Agent) persistSession() {
	a.saveSessionWithWarning("⚠️  Warning: Failed to save session: %v\n")
}

func (a *Agent) saveSessionWithWarning(format string) {
	if a == nil || a.session == nil || a.storage == nil {
		return
	}
	a.syncSessionPersistenceState()
	if err := a.storage.Save(a.session); err != nil {
		yellow.Fprintf(a.output(), format, err)
	}
}

// cleanupHook はテスト用フック（非nil時にCleanupから呼ばれる）
var cleanupHook func()

// exitProcess はテスト用に差し替え可能なプロセス終了フック。
var exitProcess = os.Exit

// Cleanup はエージェントのリソースをクリーンアップ
func (a *Agent) Cleanup() {
	if cleanupHook != nil {
		cleanupHook()
	}
	if a.mcpManager != nil {
		a.mcpManager.Close()
	}
	if a.lspClient != nil {
		a.lspClient.Close()
	}
	if a.ToolCache != nil {
		if err := a.ToolCache.Save(); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to save tool cache: %v\n", err)
		}
	}
	a.saveSessionWithWarning("Warning: Failed to save session: %v\n")
}
