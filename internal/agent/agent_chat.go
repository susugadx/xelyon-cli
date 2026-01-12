package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	// Plan Mode が有効な場合、RunPlanMode を呼ぶ
	if a.PlanMode {
		ctx := context.Background()
		if err := a.RunPlanMode(ctx, input); err != nil {
			red.Printf("Plan execution failed: %v\n", err)
		}
		return
	}

	// 履歴に追加
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: input,
	})

	// セッションに保存
	if a.session != nil {
		a.session.AddMessage("user", input, a.CurrentModel)
	}

	// 統計情報更新: Userメッセージ数をカウント
	if a.Stats != nil {
		a.Stats.UserMessages++
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
	cfg := config.GetGlobalConfig()
	maxRetries := cfg.APIRetry.Count
	initialDelay := cfg.APIRetry.InitialDelay
	maxDelay := cfg.APIRetry.MaxDelay
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second

	var response string
	var err error

	for retry := 0; retry < maxRetries; retry++ {
		// API呼び出しタイムアウト設定（設定から取得、デフォルト5分）
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		response, err = a.CurrentProvider.ChatWithTools(ctx, a.SystemPrompt, a.History, a.CurrentModel)
		cancel() // リソースリーク防止

		if err == nil {
			return response, nil
		}

		if retry < maxRetries-1 {
			// 待機時間計算：initialDelay * (retry + 1) だが、maxDelay を超えない
			delay := initialDelay * (retry + 1)
			if delay > maxDelay {
				delay = maxDelay
			}
			yellow.Printf("⚠️  API error, retrying (%d/%d)...\n", retry+1, maxRetries-1)
			time.Sleep(time.Second * time.Duration(delay))
		}
	}

	red.Printf("エラー: %v\n", err)
	yellow.Println("API呼び出しに失敗しました。ネットワーク接続を確認してください。")
	return "", err
}

// extractExplanationAndTool はレスポンスから説明部分とツールJSONを分離
// NOTE: agent_plan.goでも使用されるため、パッケージレベルの関数として定義
func extractExplanationAndTool(response string) (explanation, toolJSON string) {
	// ツール呼び出しのJSON部分を探す
	toolStartIdx := -1
	patterns := []string{"{\"tool\"", "{ \"tool\"", "{\"tool\":", "{ \"tool\":"}
	for _, pattern := range patterns {
		idx := strings.Index(response, pattern)
		if idx != -1 && (toolStartIdx == -1 || idx < toolStartIdx) {
			toolStartIdx = idx
		}
	}

	if toolStartIdx == -1 {
		// ツール呼び出しなし
		return response, ""
	}

	// ツール呼び出しより前の部分が説明
	explanation = strings.TrimSpace(response[:toolStartIdx])

	// ツール呼び出しのJSON部分を抽出
	depth := 0
	inString := false
	escaped := false
	for i := toolStartIdx; i < len(response); i++ {
		ch := response[i]

		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				toolJSON = response[toolStartIdx : i+1]
				return explanation, toolJSON
			}
		}
	}

	// 閉じ括弧が見つからない場合は全体をツールJSONとみなす
	return explanation, response[toolStartIdx:]
}

// shouldAbortToolLoop は同じツール呼び出しの繰り返しを検知
func (a *Agent) shouldAbortToolLoop(current, last *tools.ToolCall, count *int) bool {
	cfg := config.GetGlobalConfig()
	threshold := cfg.LoopDetection.Threshold

	if isSameToolCall(current, last) {
		*count++
		if *count >= threshold {
			yellow.Printf("⚠️  Warning: Same tool call repeated %d times, stopping to prevent infinite loop\n", *count)
			yellow.Printf("   Tool: %s\n", current.Tool)

			// AI に警告メッセージを返す
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[SYSTEM WARNING] The same tool call was repeated %d times. Please try a different approach or ask the user for clarification.", threshold),
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
	// レスポンスから説明部分とツール呼び出しを分離
	explanation, _ := extractExplanationAndTool(response)

	// 説明部分を先に表示
	if explanation != "" {
		cyan.Println("\n💭 AI Explanation:")
		fmt.Println(explanation)
		fmt.Println()
	}

	// 結果を履歴に追加
	a.History = append(a.History, api.Message{
		Role:    "assistant",
		Content: response,
	})

	// 統計情報更新: Assistantメッセージ数とツール実行回数をカウント
	if a.Stats != nil {
		a.Stats.AssistantMessages++
		a.Stats.AddToolExecution(toolCall.Tool)
	}

	// Dry Run Mode: ツールを実行せず、結果だけをシミュレート
	if a.DryRunMode {
		result := "[Dry Run] Tool execution simulated"
		// 結果を履歴に追加
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
		})
		fmt.Println()
		return
	}

	// ツール実行
	result, change := tools.Execute(toolCall)

	// 変更履歴を保存
	if change != nil {
		a.changeStack = append(a.changeStack, *change)
		if len(a.changeStack) > config.MaxChangeStack {
			a.changeStack = a.changeStack[1:]
		}

		// 永続的変更履歴に保存
		if a.changeStorage != nil && a.session != nil {
			if err := a.changeStorage.AppendChange(a.session.ID, *change); err != nil {
				// エラーログは出すが実行は継続
				yellow.Printf("Warning: Failed to persist change: %v\n", err)
			}
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

	// 統計情報更新: Assistantメッセージ数をカウント
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}

	// 最後の出力を記録（最大10件）
	a.lastOutputs = append(a.lastOutputs, response)
	if len(a.lastOutputs) > 10 {
		a.lastOutputs = a.lastOutputs[1:]
	}

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
