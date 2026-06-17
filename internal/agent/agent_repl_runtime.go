package agent

import (
	"fmt"
	"os"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// initInteractiveAgentWithRuntime は、事前に用意した runtime を使ってインタラクティブ用 Agent を初期化する。
func initInteractiveAgentWithRuntime(runtime *AgentRuntime, model string, provider api.Provider, autoApprove bool, commandSurface commandcatalog.CommandSurface) *Agent {
	// normalizeAgentRuntime は NewAgentWithRuntime 内部で呼ばれるためここでは不要。
	// ただし AutoApprove は normalize 前に設定する必要がある。
	runtime.AutoApprove = autoApprove

	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	runtimeUI := runtime.effectiveUI()
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	logger, err := audit.NewDefaultLogger(auditEnabled)
	if err != nil {
		yellow.Fprintf(runtimeUI.Output(), "Warning: Failed to initialize audit log: %v\n", err)
	} else {
		runtime.AuditLogger = logger
	}
	if auditEnabled {
		green.Fprintln(runtimeUI.Output(), "📝 Audit logging enabled")
	}

	agent := NewAgentWithRuntime(model, provider, false, runtime)
	agent.setAutoApprove(autoApprove)
	printProviderSetupRequiredNotice(agent)

	setupSignalHandler(agent)
	if err := initializeProjectInstructions(agent, projectInstructionApplyOptions{
		showStatus:       true,
		injectProjectMap: true,
	}); err != nil {
		red.Fprintf(runtimeUI.Output(), "Failed to load project instructions: %v\n", err)
	}
	checkRipgrepAvailability(agent)
	checkLSPInstallPrompt(agent, commandSurface)

	return agent
}

type interactiveREPLEnvironment struct {
	runtime   *AgentRuntime
	runtimeUI *ui.Runtime
	mlReader  *ui.MultilineReader
}

// prepareInteractiveREPLEnvironment は REPL 起動に必要な runtime/UI/reader を初期化する。
func prepareInteractiveREPLEnvironment(cfg *config.Config, autoApprove bool) (*interactiveREPLEnvironment, func()) {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove

	runtimeUI := runtime.effectiveUI()
	mlReader := ui.NewMultilineReaderWithRuntime(runtimeUI)
	runtimeUI.SetPromptReader(mlReader)
	runtimeCfg := runtime.effectiveConfig()

	// Bracketed Paste Mode を最初に有効化（Windows Terminal の警告回避のため）
	// 他の出力より前に送信する必要がある
	if os.Getenv("XELYON_DEBUG_PASTE") == "1" {
		_, _ = fmt.Fprintf(runtimeUI.ErrorOutput(), "[DEBUG] cfg.Paste.BracketedPaste = %v\n", runtimeCfg.Paste.BracketedPaste)
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if runtimeCfg.Paste.BracketedPaste {
				mlReader.DisableBracketedPaste()
			}
		})
	}

	if runtimeCfg.Paste.BracketedPaste {
		mlReader.EnableBracketedPaste()
	}

	return &interactiveREPLEnvironment{
		runtime:   runtime,
		runtimeUI: runtimeUI,
		mlReader:  mlReader,
	}, cleanup
}
