package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunHeadless はHeadlessモードでクエリを実行（マルチターンツール実行対応）
func RunHeadless(query string, model string, provider api.Provider) *HeadlessResult {
	startTime := time.Now()

	// Agent初期化
	agent := NewAgent(model, provider, true)
	agent.AutoApprove = true // Headlessモードは自動承認（SafetyLow以外）
	tools.SetAutoApprove(true)

	// プロジェクト設定読み込み（UI出力なし）
	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
	}

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

	for iteration := 0; iteration < maxIterations; iteration++ {
		// API呼び出し
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

		response, err := provider.ChatWithTools(ctx, agent.SystemPrompt, agent.History, model)
		cancel()

		if err != nil {
			duration := time.Since(startTime).Milliseconds()
			return NewErrorResult(provider.Name(), model, "api_error", err.Error(), duration)
		}

		// ツール呼び出し解析
		parsedCalls := tools.ParseToolCalls(response)

		// ツール呼び出しがなければ最終レスポンスとして終了
		if len(parsedCalls) == 0 {
			finalResponse = response
			break
		}

		// ツール実行と結果収集
		var toolOutputs []string
		for _, tc := range parsedCalls {
			output, change := tools.Execute(tc)

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

// RunOnceWithImage は画像付きの単一クエリを実行（CLIフラグ -i/--image 用）
func RunOnceWithImage(query string, model string, provider api.Provider, imagePath string, autoApprove bool) {
	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	if err := audit.Init(auditEnabled); err != nil {
		yellow.Printf("Warning: Failed to initialize audit log: %v\n", err)
	}

	agent := NewAgent(model, provider, false)
	agent.AutoApprove = autoApprove
	tools.SetAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示
	printHeader(model, provider)
	printModeInfo(autoApprove, false)

	// プロバイダーが画像対応かチェック
	if !api.SupportsImages(provider.Name()) {
		red.Printf("❌ Provider '%s' does not support image input\n", provider.Name())
		fmt.Println("Supported providers for image input: gemini, claude, openai")
		return
	}

	// 画像読み込み
	image, err := api.LoadImage(imagePath)
	if err != nil {
		red.Printf("❌ Failed to load image: %v\n", err)
		return
	}
	green.Printf("🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	// XELYON.md読み込み
	if config := loadProjectConfig(); config != "" {
		// ルール系セクションを Workflow Rules 内に強制挿入
		rulesBlock := prompt.BuildProjectRulesBlock(config)
		if rulesBlock != "" {
			agent.SystemPrompt = prompt.InjectProjectRules(agent.SystemPrompt, rulesBlock)
		}
		// ルール系を除いた残りを Project Context として末尾に追加
		stripped := prompt.StripRuleSections(config)
		if stripped != "" {
			agent.SystemPrompt += "\n\n## Project Context:\n" + stripped
		}
		green.Println("📋 XELYON.md loaded")
	}

	fmt.Println()

	// デフォルトメッセージ
	if query == "" {
		query = "Please analyze this image."
	}

	// 画像付きで会話
	agent.chatWithImage(query, image)

	// 対話ループに入る
	mlReader := ui.NewMultilineReader(os.Stdin)
	mlReader.EnableBracketedPaste()
	defer mlReader.DisableBracketedPaste()
	agent.mlReader = mlReader    // ペーストモードで共有するため
	ui.SetGlobalReader(mlReader) // セレクターで共有するため

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
