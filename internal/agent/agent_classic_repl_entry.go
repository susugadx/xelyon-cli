package agent

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type interactiveREPLEnvironment struct {
	runtime   *AgentRuntime
	runtimeUI *uiruntime.Runtime
	mlReader  *uiruntime.MultilineReader
}

// prepareInteractiveREPLEnvironment は legacy classic REPL 起動に必要な runtime/UI/reader を初期化する。
func prepareInteractiveREPLEnvironment(cfg *config.Config, autoApprove bool) (*interactiveREPLEnvironment, func()) {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove

	runtimeUI := runtime.effectiveUI()
	mlReader := uiruntime.NewMultilineReaderWithRuntime(runtimeUI)
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

// RunLegacyInteractiveWithConfig は legacy classic REPL を実行する。
func RunLegacyInteractiveWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	env, cleanup := prepareInteractiveREPLEnvironment(cfg, autoApprove)
	defer cleanup()

	agent := initInteractiveAgentWithRuntime(env.runtime, model, provider, autoApprove, commandcatalog.CommandSurfaceClassic)
	defer agent.Cleanup() // グレースフルシャットダウン

	// ヘッダー表示
	printHeaderToWriter(env.runtimeUI.Output(), agent.Model, provider)
	printModeInfoToWriter(env.runtimeUI.Output(), autoApprove, false)

	// コンテキストサイズ表示（ツリー形式）
	printContextSize(agent)

	// REPLループ開始
	agent.setPromptReader(env.mlReader)
	runREPLLoop(agent, env.mlReader)
}

// RunInteractiveWithConfig は legacy classic REPL の互換入口。
//
// Deprecated: interactive primary surface には RunTUIWithConfig を使う。
func RunInteractiveWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	RunLegacyInteractiveWithConfig(model, provider, cfg, autoApprove)
}

// RunLegacyInteractiveWithImageWithConfig は legacy classic REPL で画像付き初回ターンを実行する。
func RunLegacyInteractiveWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
	env, cleanup := prepareInteractiveREPLEnvironment(cfg, autoApprove)
	defer cleanup()

	if api.IsProviderSetupRequired(provider) {
		agent := initInteractiveAgentWithRuntime(env.runtime, model, provider, autoApprove, commandcatalog.CommandSurfaceClassic)
		defer agent.Cleanup()

		printHeaderToWriter(env.runtimeUI.Output(), agent.Model, provider)
		printModeInfoToWriter(env.runtimeUI.Output(), autoApprove, false)
		printContextSize(agent)
		agent.setPromptReader(env.mlReader)
		runREPLLoop(agent, env.mlReader)
		return nil
	}

	if !provider.SupportsImages() {
		return fmt.Errorf("provider %q does not support image input", provider.Name())
	}

	image, err := api.LoadImage(imagePath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	agent := initInteractiveAgentWithRuntime(env.runtime, model, provider, autoApprove, commandcatalog.CommandSurfaceClassic)
	defer agent.Cleanup()

	printHeaderToWriter(env.runtimeUI.Output(), agent.Model, provider)
	printModeInfoToWriter(env.runtimeUI.Output(), autoApprove, false)
	printContextSize(agent)
	green.Fprintf(env.runtimeUI.Output(), "🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	if query == "" {
		query = DefaultImagePrompt
	}

	agent.setPromptReader(env.mlReader)
	_ = agent.chatWithImage(query, image)
	runREPLLoop(agent, env.mlReader)
	return nil
}

// RunInteractiveWithImageWithConfig は legacy classic REPL の互換入口。
//
// Deprecated: interactive primary surface には RunTUIWithImageWithConfig を使う。
func RunInteractiveWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
	return RunLegacyInteractiveWithImageWithConfig(query, model, provider, imagePath, cfg, autoApprove)
}

// RunLegacyInteractiveWithResumeWithConfig は legacy classic REPL で前回セッションを再開する。
func RunLegacyInteractiveWithResumeWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	env, cleanup := prepareInteractiveREPLEnvironment(cfg, autoApprove)
	defer cleanup()

	agent := initInteractiveAgentWithRuntime(env.runtime, model, provider, autoApprove, commandcatalog.CommandSurfaceClassic)
	session, err := agent.ResumeStartupLastSession(history.ResumeListOptions{})
	if err != nil {
		if errors.Is(err, history.ErrNoResumeSessions) {
			yellow.Fprintln(env.runtimeUI.Output(), "No previous session found, starting new session")
		} else if strings.Contains(err.Error(), "load session") {
			red.Fprintf(env.runtimeUI.Output(), "Failed to load session: %v\n", err)
		} else if strings.Contains(err.Error(), "history storage not available") {
			red.Fprintf(env.runtimeUI.Output(), "Failed to initialize storage: %v\n", err)
		} else {
			red.Fprintf(env.runtimeUI.Output(), "Failed to resume session: %v\n", err)
		}
		agent.restoreSessionConversation(nil)
		agent.Cleanup()
		cleanup()
		RunLegacyInteractiveWithConfig(model, provider, cfg, autoApprove)
		return
	}
	defer agent.Cleanup() // グレースフルシャットダウン

	printHeaderToWriter(env.runtimeUI.Output(), agent.CurrentModel, agent.CurrentProvider)
	printModeInfoToWriter(env.runtimeUI.Output(), autoApprove, false)
	green.Fprintf(env.runtimeUI.Output(), "📂 Resumed session %s (%d messages)\n", session.ID, len(session.ToAPIMessages()))

	// コンテキストサイズ表示（ツリー形式）
	printContextSize(agent)

	// REPLループ開始
	agent.setPromptReader(env.mlReader)
	runREPLLoop(agent, env.mlReader)
}

// RunInteractiveWithResumeWithConfig は legacy classic REPL の互換入口。
//
// Deprecated: interactive primary surface には RunTUIWithResumeWithConfig を使う。
func RunInteractiveWithResumeWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	RunLegacyInteractiveWithResumeWithConfig(model, provider, cfg, autoApprove)
}
