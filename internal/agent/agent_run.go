package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunOnceWithConfig は指定設定で単一クエリを1ターンだけ実行して終了する。
func RunOnceWithConfig(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove
	out := runtime.effectiveUI().Output()

	configureRuntimeAuditLoggerFromEnv(runtime, out, !quiet)
	agent := NewAgentWithRuntime(model, provider, false, runtime)
	agent.setAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示（quiet 時はスキップ）
	if !quiet {
		printHeaderToWriter(runtime.effectiveUI().Output(), model, provider)
		printModeInfoToWriter(runtime.effectiveUI().Output(), autoApprove, false)
	}

	// プロジェクト instruction 読み込み（xelyon.yaml + guidance）
	initializeProjectInstructions(agent, projectInstructionApplyOptions{
		showStatus:       !quiet,
		injectProjectMap: true,
	})

	// 明示的に1ターンのみ実行（ChatOnce は stdin を読まず、REPL に入らない）
	return agent.ChatOnce(query)
}

// RunOnceWithImageWithConfig は指定設定で画像付きの単一クエリを実行する。
func RunOnceWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove
	out := runtime.effectiveUI().Output()

	configureRuntimeAuditLoggerFromEnv(runtime, out, !quiet)
	agent := NewAgentWithRuntime(model, provider, false, runtime)
	agent.setAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示（quiet 時はスキップ）
	if !quiet {
		printHeaderToWriter(runtime.effectiveUI().Output(), model, provider)
		printModeInfoToWriter(runtime.effectiveUI().Output(), autoApprove, false)
	}

	// プロバイダーが画像対応かチェック
	if !provider.SupportsImages() {
		return fmt.Errorf("provider %q does not support image input", provider.Name())
	}

	// 画像読み込み
	image, err := api.LoadImage(imagePath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	if !quiet {
		green.Fprintf(out, "🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))
	}

	// プロジェクト instruction 読み込み（xelyon.yaml + guidance）
	initializeProjectInstructions(agent, projectInstructionApplyOptions{
		showStatus:       !quiet,
		injectProjectMap: true,
	})

	if !quiet {
		_, _ = fmt.Fprintln(agent.output())
	}

	// デフォルトメッセージ
	if query == "" {
		query = "Please analyze this image."
	}

	if !quiet {
		green.Fprintf(out, "🖼️  Sending image: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))
	}
	return agent.ChatOnceWithImage(query, image)
}
