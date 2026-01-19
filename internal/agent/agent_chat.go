package agent

import (
	"context"
	"encoding/json"
	"errors"
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

	// ツール実行
	result, change := tools.Execute(toolCall)

	// str_replace の「old_str not found」エラー連続検知（Issue #45）
	if toolCall.Tool == "str_replace" {
		if strings.Contains(result, "Error: old_str not found") ||
			strings.Contains(result, "Error: old_str appears") {
			a.strReplaceErrorCount++
			cfg := config.GetGlobalConfig()
			threshold := cfg.LoopDetection.Threshold
			if threshold < 2 {
				threshold = 2
			}
			if a.strReplaceErrorCount >= threshold {
				yellow.Printf("⚠️  str_replace failed %d times consecutively. Stopping to prevent loop.\n", a.strReplaceErrorCount)
				yellow.Println("💡 Suggested alternatives / 代替案:")
				fmt.Println("   1. Use read_file to verify the exact content of the target file")
				fmt.Println("   2. Use search_code to find the correct string pattern")
				fmt.Println("   3. Ask the user for clarification on what to change")
				fmt.Println("   4. Try delete_lines + insert_before/insert_after for line-based edits")
				fmt.Println()

				// AIに警告を送信
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
				})
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: `[SYSTEM WARNING] str_replace has failed multiple times. The old_str pattern was not found in the file.

Suggested next steps:
1. Use read_file to see the actual file contents
2. Use search_code to find the correct pattern
3. Ask the user for clarification
4. Try a different approach (delete_lines + insert_before/insert_after)

IMPORTANT: Do NOT retry str_replace with the same or similar old_str pattern. Take a different approach.`,
				})
				a.strReplaceErrorCount = 0 // リセット
				fmt.Println()
				return
			}
		} else if strings.Contains(result, "Successfully replaced") {
			// 成功したらカウンターをリセット
			a.strReplaceErrorCount = 0
		}
	} else {
		// 他のツールが呼ばれたらカウンターをリセット
		a.strReplaceErrorCount = 0
	}

	// comment 継続フロー: ツール側が [COMMENT] シグナルを返した場合、コメントを履歴に入れて再提案を依頼
	// - これにより SafetyLow の確認で "comment" を選んでも作業を中断せず継続できる
	if strings.Contains(result, "[COMMENT]") {
		// ループ防止（同一コメントの連続は抑止）
		cfg := config.GetGlobalConfig()
		maxFeedback := cfg.LoopDetection.Threshold
		if maxFeedback < 3 {
			maxFeedback = 3
		}

		// フィードバック回数はこのツール結果メッセージ数で近似（簡易）
		feedbackCount := 0
		for _, msg := range a.History {
			if msg.Role == "user" && strings.Contains(msg.Content, "[COMMENT]") {
				feedbackCount++
			}
		}
		if feedbackCount >= maxFeedback {
			yellow.Printf("⚠️  Feedback loop detected (%d), stopping comment retry\n", feedbackCount)
		} else {
			// 結果を履歴に先に入れてから、AIへ再提案を要求
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
			})

			// AIに「コメントを反映して別案を提示」するよう促す
			a.History = append(a.History, api.Message{
				Role: "user",
				Content: fmt.Sprintf(`[USER FEEDBACK]
The previous tool execution was NOT performed because the user selected comment in the confirmation UI.

%s

IMPORTANT:
- Revise your approach/tool call based on this feedback.
- Do NOT repeat the exact same tool call.
- If you need to ask the user a question, ask it explicitly instead of calling tools.`, result),
			})

			// 次ループで callAPIWithRetry() が走るようにする
			fmt.Println()
			return
		}
	}

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

		// コード健全性チェック（on_changeフック）
		a.checkCodeHealthOnChange(change.FilePath)
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
