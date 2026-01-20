package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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

	// str_replace エラー処理
	if a.handleStrReplaceErrors(toolCall, result) {
		return
	}

	// comment 継続フロー処理
	if a.handleCommentFlow(toolCall, result) {
		return
	}

	// 変更履歴を保存
	a.handleFileChange(change)

	// 結果を履歴に追加
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
	})

	fmt.Println()
}

// handleStrReplaceErrors は str_replace の「old_str not found」エラー連続検知を処理（Issue #45）
// 処理済みの場合は true を返す
func (a *Agent) handleStrReplaceErrors(toolCall *tools.ToolCall, result string) bool {
	if toolCall.Tool != "str_replace" {
		// 他のツールが呼ばれたらカウンターをリセット
		a.strReplaceErrorCount = 0
		return false
	}

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
			return true
		}
	} else if strings.Contains(result, "Successfully replaced") {
		// 成功したらカウンターをリセット
		a.strReplaceErrorCount = 0
	}
	return false
}

// handleCommentFlow はcomment 継続フローを処理
// ツール側が [COMMENT] シグナルを返した場合、コメントを履歴に入れて再提案を依頼
// 処理済みの場合は true を返す
func (a *Agent) handleCommentFlow(toolCall *tools.ToolCall, result string) bool {
	if !strings.Contains(result, "[COMMENT]") {
		return false
	}

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
		return false
	}

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
	return true
}

// handleFileChange は変更履歴を保存
func (a *Agent) handleFileChange(change *tools.FileChange) {
	if change == nil {
		return
	}

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
