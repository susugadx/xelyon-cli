package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// initInteractiveAgentWithRuntime は、事前に用意した runtime を使ってインタラクティブ用 Agent を初期化する。
func initInteractiveAgentWithRuntime(runtime *AgentRuntime, model string, provider api.Provider, autoApprove bool) *Agent {
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

	// シグナルハンドリング（Ctrl+C 2回で終了、1回目はAI応答中断）
	setupSignalHandler(agent)

	// プロジェクト設定読み込み（xelyon.yaml）
	if pc := loadProjectConfig(); pc != nil {
		applyProjectConfig(agent, pc)
	}
	injectProjectMap(agent, "")
	checkRipgrepAvailability(agent)

	return agent
}

// RunInteractiveWithConfig は指定設定でインタラクティブモードを実行する。
func RunInteractiveWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
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

	if runtimeCfg.Paste.BracketedPaste {
		mlReader.EnableBracketedPaste()
		defer mlReader.DisableBracketedPaste()
	}

	agent := initInteractiveAgentWithRuntime(runtime, model, provider, autoApprove)
	defer agent.Cleanup() // グレースフルシャットダウン

	// ヘッダー表示
	printHeaderToWriter(runtimeUI.Output(), agent.Model, provider)
	printModeInfoToWriter(runtimeUI.Output(), autoApprove, false)

	// コンテキストサイズ表示（ツリー形式）
	printContextSize(agent)

	// REPLループ開始
	agent.setPromptReader(mlReader)
	runREPLLoop(agent, mlReader)
}

// RunInteractiveWithImageWithConfig は指定設定で画像付きのインタラクティブモードを実行する。
func RunInteractiveWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove

	runtimeUI := runtime.effectiveUI()
	mlReader := ui.NewMultilineReaderWithRuntime(runtimeUI)
	runtimeUI.SetPromptReader(mlReader)
	runtimeCfg := runtime.effectiveConfig()

	if os.Getenv("XELYON_DEBUG_PASTE") == "1" {
		_, _ = fmt.Fprintf(runtimeUI.ErrorOutput(), "[DEBUG] cfg.Paste.BracketedPaste = %v\n", runtimeCfg.Paste.BracketedPaste)
	}

	if runtimeCfg.Paste.BracketedPaste {
		mlReader.EnableBracketedPaste()
		defer mlReader.DisableBracketedPaste()
	}

	if !provider.SupportsImages() {
		return fmt.Errorf("provider %q does not support image input", provider.Name())
	}

	image, err := api.LoadImage(imagePath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	agent := initInteractiveAgentWithRuntime(runtime, model, provider, autoApprove)
	defer agent.Cleanup()

	printHeaderToWriter(runtimeUI.Output(), agent.Model, provider)
	printModeInfoToWriter(runtimeUI.Output(), autoApprove, false)
	printContextSize(agent)
	green.Fprintf(runtimeUI.Output(), "🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	if query == "" {
		query = "Please analyze this image."
	}

	agent.setPromptReader(mlReader)
	agent.chatWithImage(query, image)
	runREPLLoop(agent, mlReader)
	return nil
}

// RunInteractiveWithResumeWithConfig は指定設定で前回セッションを再開してインタラクティブモードを実行する。
func RunInteractiveWithResumeWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove
	runtimeUI := runtime.effectiveUI()

	// Bracketed Paste Mode を最初に有効化（Windows Terminal の警告回避のため）
	mlReader := ui.NewMultilineReaderWithRuntime(runtimeUI)
	runtimeUI.SetPromptReader(mlReader)
	runtimeCfg := runtime.effectiveConfig()

	if os.Getenv("XELYON_DEBUG_PASTE") == "1" {
		_, _ = fmt.Fprintf(runtimeUI.ErrorOutput(), "[DEBUG] cfg.Paste.BracketedPaste = %v\n", runtimeCfg.Paste.BracketedPaste)
	}

	if runtimeCfg.Paste.BracketedPaste {
		mlReader.EnableBracketedPaste()
		defer mlReader.DisableBracketedPaste()
	}

	storage, err := history.NewStorage()
	if err != nil {
		red.Fprintf(runtimeUI.Output(), "Failed to initialize storage: %v\n", err)
		RunInteractiveWithConfig(model, provider, cfg, autoApprove)
		return
	}

	sessionID, err := storage.GetLastSession()
	if err != nil {
		yellow.Fprintln(runtimeUI.Output(), "No previous session found, starting new session")
		RunInteractiveWithConfig(model, provider, cfg, autoApprove)
		return
	}

	session, err := storage.Load(sessionID)
	if err != nil {
		red.Fprintf(runtimeUI.Output(), "Failed to load session: %v\n", err)
		RunInteractiveWithConfig(model, provider, cfg, autoApprove)
		return
	}

	// ロード済みセッションでAgent作成
	agent := initInteractiveAgentWithRuntime(runtime, model, provider, autoApprove)
	agent.restoreSessionConversation(session)
	defer agent.Cleanup() // グレースフルシャットダウン

	printHeaderToWriter(runtimeUI.Output(), model, provider)
	printModeInfoToWriter(runtimeUI.Output(), autoApprove, false)
	green.Fprintf(runtimeUI.Output(), "📂 Resumed session %s (%d messages)\n", sessionID, len(session.ToAPIMessages()))

	// コンテキストサイズ表示（ツリー形式）
	printContextSize(agent)

	// REPLループ開始
	agent.setPromptReader(mlReader)
	runREPLLoop(agent, mlReader)
}

// runREPLLoop は共通のREPLループを実行（RunInteractive/RunInteractiveWithResumeで共用）
func runREPLLoop(agent *Agent, mlReader *ui.MultilineReader) {
	var lastInterrupt time.Time

	for {
		// AI出力後に溜まった入力をクリア（出力中のEnter押下を無視）
		mlReader.FlushInput()

		// Status / 状態表示（常にプロンプト直前に表示）
		agent.PrintStatusFooter()

		// 緑色のプロンプト
		greenPrompt := green.Sprint(">")
		input, err := mlReader.ReadInput("\n" + greenPrompt + " ")
		if err != nil {
			// Handle Ctrl+C (ErrInterrupted)
			if err == ui.ErrInterrupted {
				now := time.Now()
				if now.Sub(lastInterrupt) < 3*time.Second {
					// 2回目（3秒以内）: アプリ終了
					_, _ = fmt.Fprintln(agent.output(), "\n👋 Gracefully shutting down...")
					agent.Cleanup()
					os.Exit(0)
				}
				lastInterrupt = now
				_, _ = fmt.Fprintln(agent.output(), "⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")
				continue
			}
			// Other errors (like EOF): exit loop
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 特殊コマンド
		if handleSpecialCommand(input, agent) {
			continue
		}

		// 画像入力チェック: image:/path/to/file.png 形式を検出
		if strings.Contains(input, "image:") {
			textPart, image := parseImageInputWithWriter(agent.output(), input)
			if image != nil {
				agent.chatWithImage(textPart, image)
				continue
			}
		}

		// AIに送信
		agent.chat(input)
	}
}

// setupSignalHandler はシグナルハンドラーを設定する
func setupSignalHandler(agent *Agent) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var lastInterrupt time.Time
	var interruptMu sync.Mutex
	go func() {
		for sig := range sigChan {
			interruptMu.Lock()
			now := time.Now()

			if now.Sub(lastInterrupt) < 3*time.Second {
				// 2回目（3秒以内）: アプリ終了
				interruptMu.Unlock()
				if agent.exitHook != nil {
					agent.exitHook()
				}
				_, _ = fmt.Fprintln(agent.output(), "\n\n👋 Gracefully shutting down...")
				agent.Cleanup()
				os.Exit(0)
			}

			lastInterrupt = now
			interruptMu.Unlock()

			// 1回目: 中断メッセージ
			_, _ = fmt.Fprintln(agent.output(), "\n\n⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")

			// 現在のAPI呼び出しをキャンセル
			agent.cancelActiveRequest(fmt.Sprintf("signal: %s", sig))
		}
	}()
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

// printContextSize はコンテキストサイズをツリー形式で表示
func printContextSize(agent *Agent) {
	out := agent.output()
	dim.Fprint(out, buildContextSizeBlock(agent))
}

func buildContextSizeBlock(agent *Agent) string {
	systemPromptTokens := agent.EstimateSystemPromptTokens()
	basePromptTokens := systemPromptTokens
	toolsTokens := 0
	projectMapTokens := estimateProjectMapTokens(agent.SystemPrompt)

	if agent.CurrentProvider != nil && agent.CurrentProvider.IsFunctionCallingEnabled() {
		toolsTokens = agent.estimateToolDefinitionTokens()
	}

	builtinCount, mcpCount := agent.countToolsByType()
	projectTokens := estimateProjectConfigTokens(loadProjectConfig())
	basePromptTokens -= projectMapTokens + projectTokens
	if basePromptTokens < 0 {
		basePromptTokens = 0
	}

	total := systemPromptTokens + toolsTokens

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "📋 Context size: ~%s tok\n", FormatTokens(total))

	lines := []string{
		fmt.Sprintf("Base prompt: ~%s", FormatTokens(basePromptTokens)),
	}
	if mcpCount > 0 {
		lines = append(lines, fmt.Sprintf("Tools (%d+%d MCP): ~%s",
			builtinCount, mcpCount, FormatTokens(toolsTokens)))
	} else {
		lines = append(lines, fmt.Sprintf("Tools (%d): ~%s",
			builtinCount, FormatTokens(toolsTokens)))
	}
	if projectMapTokens > 0 && agent.projectMapFileCount > 0 {
		if agent.projectMapSymbolCount > 0 {
			lines = append(lines, fmt.Sprintf("Project map (%d symbols, %d files): ~%s",
				agent.projectMapSymbolCount, agent.projectMapFileCount, FormatTokens(projectMapTokens)))
		} else {
			lines = append(lines, fmt.Sprintf("Project map (%d files): ~%s",
				agent.projectMapFileCount, FormatTokens(projectMapTokens)))
		}
	}
	if projectTokens > 0 {
		lines = append(lines, fmt.Sprintf("xelyon.yaml: ~%s", FormatTokens(projectTokens)))
	}

	for i, line := range lines {
		connector := "├──"
		if i == len(lines)-1 {
			connector = "└──"
		}
		fmt.Fprintf(&buf, "   %s %s\n", connector, line)
	}

	return buf.String()
}

func estimateProjectMapTokens(systemPrompt string) int {
	section := extractProjectMapSection(systemPrompt)
	if section == "" {
		return 0
	}
	return token.EstimateTokenCount(section)
}

func extractProjectMapSection(systemPrompt string) string {
	const marker = "## Project Map\n"

	idx := strings.LastIndex(systemPrompt, marker)
	if idx < 0 {
		return ""
	}

	section := systemPrompt[idx:]
	nextSection := strings.Index(section[len(marker):], "\n## ")
	if nextSection >= 0 {
		section = section[:len(marker)+nextSection]
	}

	return strings.TrimRight(section, "\n")
}

func stripProjectMapSection(systemPrompt string) string {
	section := extractProjectMapSection(systemPrompt)
	if section == "" {
		return systemPrompt
	}

	idx := strings.LastIndex(systemPrompt, section)
	if idx < 0 {
		return systemPrompt
	}

	stripped := strings.TrimRight(systemPrompt[:idx], "\n")
	if idx+len(section) < len(systemPrompt) {
		stripped += systemPrompt[idx+len(section):]
	}
	return strings.TrimRight(stripped, "\n")
}
