package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

// RunLegacyInteractiveWithConfig は legacy classic REPL を実行する。
func RunLegacyInteractiveWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	env, cleanup := prepareInteractiveREPLEnvironment(cfg, autoApprove)
	defer cleanup()

	agent := initInteractiveAgentWithRuntime(env.runtime, model, provider, autoApprove, commandcatalog.CommandSurfaceClassic)
	defer agent.Cleanup() // グレースフルシャットダウン

	printHeaderToWriter(env.runtimeUI.Output(), agent.Model, provider)
	printModeInfoToWriter(env.runtimeUI.Output(), autoApprove, false)
	printContextSize(agent)

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
		query = "Please analyze this image."
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
	printContextSize(agent)

	agent.setPromptReader(env.mlReader)
	runREPLLoop(agent, env.mlReader)
}

// RunInteractiveWithResumeWithConfig は legacy classic REPL の互換入口。
//
// Deprecated: interactive primary surface には RunTUIWithResumeWithConfig を使う。
func RunInteractiveWithResumeWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	RunLegacyInteractiveWithResumeWithConfig(model, provider, cfg, autoApprove)
}
