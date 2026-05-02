package agent

import (
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/audit"
)

// configureRuntimeAuditLoggerFromEnv は環境変数設定に応じて runtime の監査ロガーを初期化する。
func configureRuntimeAuditLoggerFromEnv(runtime *AgentRuntime, out io.Writer, announce bool) {
	if runtime == nil {
		return
	}

	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	logger, err := audit.NewDefaultLogger(auditEnabled)
	if err != nil && out != nil {
		yellow.Fprintf(out, "Warning: Failed to initialize audit log: %v\n", err)
	}
	if auditEnabled && announce && out != nil {
		green.Fprintln(out, "📝 Audit logging enabled")
	}
	if logger != nil {
		runtime.AuditLogger = logger
	}
}
