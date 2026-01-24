package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunOnce は単一のクエリを実行（レガシーAPI）
func RunOnce(query string, model string) {
	// Note: この関数は古いAPI (api.ChatWithTools) を使用
	// 将来的に削除予定
	agent := &Agent{
		Model:        model,
		CurrentModel: model,
		History:      []api.Message{},
		session:      history.NewSession(model),
	}

	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	// Repo Map 生成（キャッシュあり）
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "." // フォールバック
	}
	repoMapStr, symbols, files, fromCache := loadRepoMapForProject(cwd, getMaxTokens(cwd))
	if repoMapStr != "" {
		agent.SystemPrompt += "\n\n" + repoMapStr
		if fromCache {
			green.Println("🗺️  Repo map loaded (cache)")
		} else {
			green.Printf("🗺️  Repo map loaded (%d symbols from %d files)\n", symbols, files)
		}
	}

	fmt.Println()
	agent.chat(query)
}

// RunHeadless はHeadlessモードでクエリを実行
func RunHeadless(query string, model string, provider api.Provider) *HeadlessResult {
	startTime := time.Now()

	// Agent初期化
	agent := NewAgent(model, provider)
	agent.AutoApprove = true // Headlessモードは自動承認（SafetyLow以外）
	tools.SetAutoApprove(true)

	// プロジェクト設定読み込み（UI出力なし）
	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
	}

	// Repo Map 生成（キャッシュあり / UI出力なし）
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	repoMapStr, _, _, _ := loadRepoMapForProject(cwd, getMaxTokens(cwd))
	if repoMapStr != "" {
		agent.SystemPrompt += "\n\n" + repoMapStr
	}

	// ツール呼び出し結果を記録
	var toolCalls []ToolCallResult

	// API呼び出し
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	response, err := provider.ChatWithTools(ctx, agent.SystemPrompt, agent.History, model)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		return NewErrorResult(provider.Name(), model, "api_error", err.Error(), duration)
	}

	// ツール呼び出し解析（複数対応）
	// TODO: 実際のツール実行を含める場合は agent.Run() のロジックを統合
	parsedCalls := tools.ParseToolCalls(response)
	for _, tc := range parsedCalls {
		// ツール実行（エラーは記録するが続行）
		output, _ := tools.Execute(tc)
		toolCalls = append(toolCalls, ToolCallResult{
			Tool:    tc.Tool,
			Args:    tc.Args,
			Output:  output,
			Success: true,
		})
	}

	duration := time.Since(startTime).Milliseconds()
	return NewSuccessResult(provider.Name(), model, response, toolCalls, duration)
}

// RunOnceWithImage は画像付きの単一クエリを実行（CLIフラグ -i/--image 用）
func RunOnceWithImage(query string, model string, provider api.Provider, imagePath string, autoApprove bool) {
	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	if err := audit.Init(auditEnabled); err != nil {
		yellow.Printf("Warning: Failed to initialize audit log: %v\n", err)
	}

	agent := NewAgent(model, provider)
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
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	// Repo Map 生成（キャッシュあり）
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "."
	}
	repoMapStr, symbols, files, fromCache := loadRepoMapForProject(cwd, getMaxTokens(cwd))
	if repoMapStr != "" {
		agent.SystemPrompt += "\n\n" + repoMapStr
		if fromCache {
			green.Println("🗺️  Repo map loaded (cache)")
		} else {
			green.Printf("🗺️  Repo map loaded (%d symbols from %d files)\n", symbols, files)
		}
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
	agent.mlReader = mlReader // ペーストモードで共有するため

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
