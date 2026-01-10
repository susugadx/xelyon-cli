package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// executeToolCallInteractive はツール実行を対話的確認付きで実行
// コメントフィードバックによる修正ループに対応（最大3回）
func (a *Agent) executeToolCallInteractive(response string, toolCall *tools.ToolCall) {
	const maxRevisions = 3

	for attempt := 0; attempt <= maxRevisions; attempt++ {
		if attempt > 0 {
			yellow.Printf("\n🔄 Revision attempt %d/%d\n", attempt, maxRevisions)
		}

		// AI のレスポンスを履歴に追加
		a.History = append(a.History, api.Message{
			Role:    "assistant",
			Content: response,
		})

		// ツール確認プロンプトを表示
		confirmResult := a.showToolConfirmation(toolCall)

		switch confirmResult.Action {
		case "yes":
			// 実行
			a.executeAndRecordTool(toolCall)
			return

		case "no":
			// キャンセル
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: "[Tool execution cancelled by user]",
			})
			yellow.Println("⚠️  Tool execution cancelled")
			return

		case "comment":
			// コメント（修正指示）
			if attempt >= maxRevisions {
				yellow.Printf("⚠️  Maximum revision attempts (%d) reached. Cancelling.\n", maxRevisions)
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: "[Tool execution cancelled: maximum revision attempts reached]",
				})
				return
			}

			// AI に修正指示を送信
			green.Println("\n🤖 AI: Processing your feedback...")
			feedbackMsg := fmt.Sprintf(
				"[USER FEEDBACK]\nThe previous tool call was rejected. User comment:\n\n%s\n\nPlease revise your approach based on this feedback.",
				confirmResult.Comment,
			)

			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: feedbackMsg,
			})

			// AI から修正案を取得
			revisedResponse, err := a.callAPIWithRetry()
			if err != nil {
				red.Println("Failed to get revised response from AI")
				return
			}

			// 修正されたツール呼び出しをパース
			revisedToolCall := tools.ParseToolCall(revisedResponse)
			if revisedToolCall == nil {
				// ツール呼び出しなし → 通常の回答として処理
				a.handleNormalResponse(revisedResponse)
				return
			}

			// 次のループで修正版を確認
			response = revisedResponse
			toolCall = revisedToolCall
		}
	}
}

// showToolConfirmation はツール確認プロンプトを表示
func (a *Agent) showToolConfirmation(toolCall *tools.ToolCall) tools.ConfirmResult {
	cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🔧 Tool: %s\n", toolCall.Tool)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 引数を表示
	for key, value := range toolCall.Args {
		yellow.Printf("  %s: %s\n", key, truncateForDisplay(value, 100))
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("Execute? [y]es / [n]o / [c]omment")

	// 確認プロンプト（内部的に tools.confirmInteractive を使用）
	return tools.ConfirmInteractive("Execute this tool?")
}

// executeAndRecordTool はツールを実行して結果を記録
func (a *Agent) executeAndRecordTool(toolCall *tools.ToolCall) {
	result, change := tools.Execute(toolCall)

	// 変更履歴を保存
	if change != nil {
		a.changeStack = append(a.changeStack, *change)
		if len(a.changeStack) > config.MaxChangeStack {
			a.changeStack = a.changeStack[1:]
		}

		// Goファイル変更時の自動検証提案
		if verifyResult := ShouldVerify(change.FilePath); verifyResult.NeedsVerify {
			a.suggestVerification(change.FilePath, verifyResult)
		}
	}

	// 結果を履歴に追加
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
	})

	fmt.Println()
}

// truncateForDisplay は表示用に文字列を切り詰め
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// callAPIWithRetry はAPI呼び出しをリトライ付きで実行（agent_chat.goから移動）
func (a *Agent) callAPIWithRetryInternal() (string, error) {
	maxRetries := config.MaxAPIRetries
	var response string
	var err error

	for retry := 0; retry < maxRetries; retry++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		response, err = a.CurrentProvider.ChatWithTools(ctx, a.SystemPrompt, a.History, a.CurrentModel)
		cancel()
		if err == nil {
			return response, nil
		}

		if retry < maxRetries-1 {
			yellow.Printf("⚠️  API error, retrying (%d/%d)...\n", retry+1, maxRetries-1)
			time.Sleep(time.Second * time.Duration(retry+1))
		}
	}

	red.Printf("エラー: %v\n", err)
	yellow.Println("API呼び出しに失敗しました。ネットワーク接続を確認してください。")
	return "", err
}
