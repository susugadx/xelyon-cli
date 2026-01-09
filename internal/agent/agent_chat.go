package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// isSameToolCall は2つのToolCallが同じかを判定
func isSameToolCall(tc1, tc2 *tools.ToolCall) bool {
	if tc1 == nil || tc2 == nil {
		return false
	}
	if tc1.Tool != tc2.Tool {
		return false
	}

	// args を JSON 化して比較
	args1, _ := json.Marshal(tc1.Args)
	args2, _ := json.Marshal(tc2.Args)
	return string(args1) == string(args2)
}

// chat はAIと対話する
func (a *Agent) chat(input string) {
	// 履歴に追加
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: input,
	})

	// セッションに保存
	if a.session != nil {
		a.session.AddMessage("user", input, a.CurrentModel)
	}

	// AIに送信（ツール実行ループ）
	maxIterations := config.MaxToolIterations
	var lastToolCall *tools.ToolCall
	var sameCallCount int
	var loopCount int
	var normalExit bool

	for i := 0; i < maxIterations; i++ {
		loopCount = i + 1

		// API呼び出し（リトライあり）
		response, err := a.callAPIWithRetry()
		if err != nil {
			return
		}

		// ツール呼び出しチェック
		toolCall := tools.ParseToolCall(response)
		if toolCall != nil {
			// ループ検知
			if a.shouldAbortToolLoop(toolCall, lastToolCall, &sameCallCount) {
				continue
			}
			lastToolCall = toolCall

			// ツール実行
			a.executeToolCall(response, toolCall)
			continue
		}

		// 通常の回答
		a.handleNormalResponse(response)
		normalExit = true
		break
	}

	// 最大イテレーション警告
	if !normalExit && loopCount >= maxIterations {
		yellow.Printf("⚠️  Warning: Maximum iterations (%d) reached\n", maxIterations)
		yellow.Println("The task may be too complex or the AI is stuck in a loop.")
		yellow.Println("Consider breaking down the task into smaller steps.")
	}
}

// callAPIWithRetry はAPI呼び出しをリトライ付きで実行
func (a *Agent) callAPIWithRetry() (string, error) {
	maxRetries := config.MaxAPIRetries
	var response string
	var err error

	for retry := 0; retry < maxRetries; retry++ {
		// API呼び出しタイムアウト設定（3分）
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		response, err = a.CurrentProvider.ChatWithTools(ctx, a.SystemPrompt, a.History, a.CurrentModel)
		cancel() // リソースリーク防止
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

// shouldAbortToolLoop は同じツール呼び出しの繰り返しを検知
func (a *Agent) shouldAbortToolLoop(current, last *tools.ToolCall, count *int) bool {
	if isSameToolCall(current, last) {
		*count++
		if *count >= config.MaxSameToolCallCount {
			yellow.Printf("⚠️  Warning: Same tool call repeated %d times, stopping to prevent infinite loop\n", *count)
			yellow.Printf("   Tool: %s\n", current.Tool)

			// AI に警告メッセージを返す
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[SYSTEM WARNING] The same tool call was repeated %d times. Please try a different approach or ask the user for clarification.", config.MaxSameToolCallCount),
			})
			return true
		}
	} else {
		*count = 1
	}
	return false
}

// executeToolCall はツールを実行して結果を履歴に追加
func (a *Agent) executeToolCall(response string, toolCall *tools.ToolCall) {
	// 結果を履歴に追加
	a.History = append(a.History, api.Message{
		Role:    "assistant",
		Content: response,
	})

	// ツール実行
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

// handleNormalResponse は通常の回答（ツール呼び出しなし）を処理
func (a *Agent) handleNormalResponse(response string) {
	a.History = append(a.History, api.Message{
		Role:    "assistant",
		Content: response,
	})

	// セッションに保存
	if a.session != nil {
		a.session.AddMessage("assistant", response, a.CurrentModel)
		if a.storage != nil {
			if err := a.storage.Save(a.session); err != nil {
				// セッション保存失敗を警告（データ損失の可能性を通知）
				yellow.Printf("⚠️  Warning: Failed to save session: %v\n", err)
			}
		}
	}
}
