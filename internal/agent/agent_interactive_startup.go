package agent

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
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

	// シグナルハンドリング（Ctrl+C 2回で終了、1回目はAI応答中断）
	setupSignalHandler(agent)

	// プロジェクト instruction 読み込み（xelyon.yaml + guidance）
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

// setupSignalHandler は interactive agent のシグナルハンドラーを設定する。
func setupSignalHandler(agent *Agent) func() {
	if agent == nil {
		return func() {}
	}

	sigChan := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var lastInterrupt time.Time
	var interruptMu sync.Mutex
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			signal.Stop(sigChan)
			close(done)
		})
	}
	agent.signalCleanup = cleanup
	go func() {
		for {
			select {
			case sig := <-sigChan:
				interruptMu.Lock()
				handleSignalInterrupt(agent, &lastInterrupt, sig)
				interruptMu.Unlock()
			case <-done:
				return
			}
		}
	}()
	return cleanup
}

// checkRipgrepAvailability は ripgrep の有無をチェックし、未インストール時に案内を表示する。
func checkRipgrepAvailability(agent *Agent) {
	if agent == nil || common.IsRipgrepAvailable() {
		return
	}

	out := agent.output()
	yellow.Fprintln(out, "⚠️  ripgrep (rg) not found — Project Map disabled, search_code using grep fallback")
	dim.Fprintln(out, "   Install for better performance:")
	dim.Fprintln(out, "     Ubuntu/Debian : sudo apt install ripgrep")
	dim.Fprintln(out, "     macOS         : brew install ripgrep")
	dim.Fprintln(out, "     Windows       : winget install BurntSushi.ripgrep")
	dim.Fprintln(out, "     Other         : https://github.com/BurntSushi/ripgrep#installation")
}
