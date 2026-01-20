package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// chat はAIと対話する
// PlanModeEnabled に応じて Plan Mode または通常モードで処理
func (a *Agent) chat(input string) {
	a.SetStatus(StateRunning, "Processing request", "処理中", "Wait for response", "応答を待ってください")

	// GitHub MCP ヒントを追加（GitHub関連リクエストの場合）
	input = a.AddGitHubHint(input)

	// セッションに保存
	if a.session != nil {
		a.session.AddMessage("user", input, a.CurrentModel)
	}

	// 統計情報更新: Userメッセージ数をカウント
	if a.Stats != nil {
		a.Stats.UserMessages++
	}

	// タイムアウト付きコンテキスト作成
	cfg := config.GetGlobalConfig()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	a.cancelFunc = cancel

	var err error
	if a.PlanModeEnabled {
		// Plan Mode: 調査 → 計画 → 承認 → 実行
		err = a.RunPlanMode(ctx, input)
	} else {
		// 通常モード: ツール実行ループ
		err = a.runNormalMode(ctx, input)
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			yellow.Println("\n⚠️  Response interrupted")
		} else {
			red.Printf("Error: %v\n", err)
		}
		a.SetStatus(StateAborted, "Request failed", "リクエスト失敗", "Try again", "再試行してください")
		return
	}

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
}

// runNormalMode は通常モードでの処理（Plan Mode OFF 時）
// ツールを個別に確認しながら実行するループ
func (a *Agent) runNormalMode(ctx context.Context, input string) error {
	// 通常モード用の指示を追加（Plan JSON を出さないように）
	normalModeInput := input + `

[SYSTEM INSTRUCTION]
You are in NORMAL MODE (not Plan Mode).
- Execute tools DIRECTLY without outputting {"plan": ...} JSON
- Do NOT ask for permission or output implementation plans
- Just use the appropriate tool calls to complete the task
- If you need to modify files, use str_replace or write_file directly`

	// 履歴に追加
	a.History = append(a.History, api.Message{Role: "user", Content: normalModeInput})

	maxIterations := config.MaxToolIterations
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	for i := 0; i < maxIterations; i++ {
		// API呼び出し
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			return fmt.Errorf("API call failed: %w", err)
		}

		// Plan JSON を検出した場合、ツール実行を促す
		if planJSON := ExtractPlanJSON(response); planJSON != "" {
			yellow.Println("⚠️  Plan JSON detected in normal mode, requesting direct tool execution...")
			a.History = append(a.History, api.Message{Role: "assistant", Content: response})
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: "[SYSTEM] You are in NORMAL MODE. Do NOT output plan JSON. Execute the tools DIRECTLY now.",
			})
			continue
		}

		// ツール呼び出しをパース
		toolCalls := tools.ParseToolCalls(response)

		// デバッグログ
		if os.Getenv("XELYON_DEBUG_TOOLS") == "1" {
			fmt.Printf("[DEBUG Tools] Response length: %d, ToolCalls found: %d\n", len(response), len(toolCalls))
			if len(response) < 500 {
				fmt.Printf("[DEBUG Tools] Response: %s\n", response)
			} else {
				fmt.Printf("[DEBUG Tools] Response (first 500): %s...\n", response[:500])
			}
			for i, tc := range toolCalls {
				fmt.Printf("[DEBUG Tools] ToolCall[%d]: tool=%s, args=%v\n", i, tc.Tool, tc.Args)
			}
		}

		// ツール呼び出しなし = 通常の回答
		if len(toolCalls) == 0 {
			a.handleNormalResponse(response)
			return nil
		}

		// ツールを実行
		for _, toolCall := range toolCalls {
			// ループ検知
			if a.shouldAbortToolLoop(toolCall, lastToolCall, &sameCallCount) {
				return fmt.Errorf("tool loop detected")
			}
			lastToolCall = toolCall

			// ツール実行（executeToolCall は確認UIを含む）
			a.executeToolCall(response, toolCall)
		}
	}

	yellow.Printf("⚠️  Tool loop limit reached (%d iterations)\n", maxIterations)
	return nil
}

// chatWithImage は画像付きメッセージでAIと対話する
func (a *Agent) chatWithImage(input string, image *api.ImageData) {
	// プロバイダーが画像対応かチェック
	if !a.CurrentProvider.SupportsImages() {
		yellow.Printf("Warning: %s does not support images. The image will be ignored.\n", a.CurrentProvider.Name())
		a.chat(input)
		return
	}

	// 画像情報をログ
	green.Printf("🖼️  Sending image: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	// 履歴に追加（テキストのみ - 画像はセッションに保存しない）
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: input,
	})

	// セッションに保存（テキストのみ）
	if a.session != nil {
		a.session.AddMessage("user", input, a.CurrentModel)
	}

	// 統計情報更新
	if a.Stats != nil {
		a.Stats.UserMessages++
	}

	// 画像付きAPI呼び出し（設定からタイムアウト取得）
	cfg := config.GetGlobalConfig()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := a.CurrentProvider.ChatWithImage(ctx, a.SystemPrompt, a.History[:len(a.History)-1], input, image, a.CurrentModel)
	if err != nil {
		red.Printf("エラー: %v\n", err)
		yellow.Println("API呼び出しに失敗しました。ネットワーク接続を確認してください。")
		return
	}

	// 通常の回答処理
	a.handleNormalResponse(response)
}
