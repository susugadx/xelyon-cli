package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunHeadless はHeadlessモードでクエリを実行（マルチターンツール実行対応）
func RunHeadless(query string, model string, provider api.Provider) *HeadlessResult {
	return RunHeadlessWithConfig(query, model, provider, config.DefaultConfig())
}

// RunHeadlessWithConfig は指定設定で Headless モードのクエリを実行する。
func RunHeadlessWithConfig(query string, model string, provider api.Provider, cfg *config.Config) *HeadlessResult {
	startTime := time.Now()

	// Agent初期化
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = true
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if logger, err := audit.NewDefaultLogger(os.Getenv("XELYON_AUDIT_LOG") == "1"); err == nil {
		runtime.AuditLogger = logger
	}
	agent := NewAgentWithRuntime(model, provider, true, runtime)
	defer agent.Cleanup()
	agent.setAutoApprove(true) // Headlessモードは自動承認（SafetyLow以外）

	// プロジェクト設定読み込み（xelyon.yaml）
	if pc := loadProjectConfig(); pc != nil {
		agent.SystemPrompt = injectProjectConfig(agent.SystemPrompt, pc)
		// headless では hooks 解決のみ（UI 表示不要）
		if resolved := config.ResolveHooks(agent.cfg(), pc); resolved != nil {
			cfg := agent.cfg()
			cfg.Hooks = *resolved
		}
	}

	// Headless Mode は Normal Mode 相当: planning 系ツールを除外
	agent.registry().SetExcludedTools(prompt.PlanningToolNames)

	// ツール呼び出し結果を記録
	var allToolCalls []ToolCallResult

	// 初期ユーザーメッセージをHistoryに追加
	agent.History = append(agent.History, api.Message{
		Role:    "user",
		Content: query,
	})

	// イテレーションループ（最大10回で無限ループ防止）
	const maxIterations = 10
	var finalResponse string
	execCtx := agent.toolExecutionContext(nil, nil, nil)

	for iteration := 0; iteration < maxIterations; iteration++ {
		// API呼び出し
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

		response, err := provider.ChatWithTools(agent.requestContext(ctx), agent.SystemPrompt, agent.History, model)
		cancel()

		if err != nil {
			duration := time.Since(startTime).Milliseconds()
			return NewErrorResult(provider.Name(), model, "api_error", err.Error(), duration)
		}

		// ツール呼び出し解析
		parsedCalls := agent.parseToolCalls(response)

		// ツール呼び出しがなければ最終レスポンスとして終了
		if len(parsedCalls) == 0 {
			finalResponse = response
			break
		}

		// ツール実行と結果収集
		var toolOutputs []string
		for _, tc := range parsedCalls {
			output, change := tools.ExecuteQuietWithContext(execCtx, tc)

			// 成功判定（"Error:"を含むかどうかで簡易判定）
			success := !strings.Contains(output, "Error:")

			allToolCalls = append(allToolCalls, ToolCallResult{
				Tool:    tc.Tool,
				Args:    tc.Args,
				Output:  output,
				Success: success,
			})

			toolOutputs = append(toolOutputs, fmt.Sprintf("[%s result]\n%s", tc.Tool, output))

			// ファイル変更履歴を記録
			if change != nil {
				agent.changeStack = append(agent.changeStack, *change)
			}
		}

		// アシスタントメッセージをHistoryに追加
		agent.History = append(agent.History, api.Message{
			Role:    "assistant",
			Content: response,
		})

		// ツール結果をユーザーメッセージとしてHistoryに追加
		toolResultsMsg := strings.Join(toolOutputs, "\n\n")
		agent.History = append(agent.History, api.Message{
			Role:    "user",
			Content: toolResultsMsg,
		})

		finalResponse = response // 最大イテレーション到達時のフォールバック
	}

	duration := time.Since(startTime).Milliseconds()
	return NewSuccessResult(provider.Name(), model, finalResponse, allToolCalls, duration)
}

// RunOnce は単一クエリを1ターンだけ実行して終了する
func RunOnce(query string, model string, provider api.Provider, autoApprove bool, quiet bool) error {
	return RunOnceWithConfig(query, model, provider, config.DefaultConfig(), autoApprove, quiet)
}

// RunOnceWithConfig は指定設定で単一クエリを1ターンだけ実行して終了する。
func RunOnceWithConfig(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove
	out := runtime.effectiveUI().Output()

	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	logger, err := audit.NewDefaultLogger(auditEnabled)
	if err != nil {
		yellow.Fprintf(out, "Warning: Failed to initialize audit log: %v\n", err)
	}
	if auditEnabled && !quiet {
		green.Fprintln(out, "📝 Audit logging enabled")
	}
	if logger != nil {
		runtime.AuditLogger = logger
	}
	agent := NewAgentWithRuntime(model, provider, false, runtime)
	agent.setAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示（quiet 時はスキップ）
	if !quiet {
		printHeaderToWriter(runtime.effectiveUI().Output(), model, provider)
		printModeInfoToWriter(runtime.effectiveUI().Output(), autoApprove, false)
	}

	// プロジェクト設定読み込み（xelyon.yaml）
	if pc := loadProjectConfig(); pc != nil {
		applyProjectConfig(agent, pc)
	}

	// 明示的に1ターンのみ実行（ChatOnce は stdin を読まず、REPL に入らない）
	return agent.ChatOnce(query)
}

// RunOnceWithImage は画像付きの単一クエリを実行（CLIフラグ -i/--image 用）
func RunOnceWithImage(query string, model string, provider api.Provider, imagePath string, autoApprove bool) {
	RunOnceWithImageWithConfig(query, model, provider, imagePath, config.DefaultConfig(), autoApprove)
}

// RunOnceWithImageWithConfig は指定設定で画像付きの単一クエリを実行する。
func RunOnceWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = autoApprove
	out := runtime.effectiveUI().Output()

	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	logger, err := audit.NewDefaultLogger(auditEnabled)
	if err != nil {
		yellow.Fprintf(out, "Warning: Failed to initialize audit log: %v\n", err)
	}
	if logger != nil {
		runtime.AuditLogger = logger
	}
	agent := NewAgentWithRuntime(model, provider, false, runtime)
	agent.setAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示
	printHeaderToWriter(runtime.effectiveUI().Output(), model, provider)
	printModeInfoToWriter(runtime.effectiveUI().Output(), autoApprove, false)

	// プロバイダーが画像対応かチェック
	if !api.SupportsImages(provider.Name()) {
		red.Fprintf(out, "❌ Provider '%s' does not support image input\n", provider.Name())
		_, _ = fmt.Fprintln(agent.output(), "Supported providers for image input: gemini, claude, openai")
		return
	}

	// 画像読み込み
	image, err := api.LoadImage(imagePath)
	if err != nil {
		red.Fprintf(out, "❌ Failed to load image: %v\n", err)
		return
	}
	green.Fprintf(out, "🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	// プロジェクト設定読み込み（xelyon.yaml）
	if pc := loadProjectConfig(); pc != nil {
		applyProjectConfig(agent, pc)
	}

	_, _ = fmt.Fprintln(agent.output())

	// デフォルトメッセージ
	if query == "" {
		query = "Please analyze this image."
	}

	// 画像付きで会話
	agent.chatWithImage(query, image)

	// 対話ループに入る
	runtimeUI := runtime.effectiveUI()
	mlReader := ui.NewMultilineReaderWithRuntime(runtimeUI)
	runtimeUI.SetPromptReader(mlReader)
	mlReader.EnableBracketedPaste()
	defer mlReader.DisableBracketedPaste()
	agent.setPromptReader(mlReader)

	for {
		mlReader.FlushInput()
		agent.PrintStatusFooter()

		input, err := mlReader.ReadInput("\n> ")
		if err != nil {
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

		// 通常の会話
		agent.chat(input)
	}
}
