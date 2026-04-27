package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) addToolCallsToHistory(response string, toolCalls []*tools.ToolCall) {
	if len(toolCalls) == 0 {
		return
	}

	explanation, _ := extractExplanationAndTool(response)
	reasoningContent := a.getLastReasoningContent()
	contentBlocks := a.getLastAnthropicContentBlocks()

	openAIToolCalls := make([]api.OpenAIToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		openAIToolCalls[i] = api.OpenAIToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      tc.Tool,
				Arguments: toolruntime.ArgsToJSON(tc.RawArgs),
			},
		}
		// ThoughtSignature/ThoughtParts は最初の ToolCall のみ（Gemini 3 仕様）
		if i == 0 {
			openAIToolCalls[i].ThoughtSignature = tc.ThoughtSignature
			openAIToolCalls[i].ThoughtParts = tc.ThoughtParts
		}
	}

	historyContent, historyReasoning := a.assistantToolHistoryContent(explanation, reasoningContent)

	msg := api.Message{
		Role:             "assistant",
		Content:          historyContent,
		ReasoningContent: historyReasoning,
		ToolCalls:        openAIToolCalls,
	}
	msg.SetAnthropicContentBlocks(contentBlocks)
	a.History = append(a.History, msg)

	// セッションに保存（1回のみ）
	if a.session != nil {
		msg := a.History[len(a.History)-1]
		a.appendSessionMessageFromAPI(msg, a.CurrentModel)
	}

	// 統計情報更新: AssistantMessages は1回カウント。
	// ToolExecution は Phase 2 で実際に実行された call のみカウントする
	// （same-turn duplicate や batch merge で省略された call は除外）。
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}
}

func (a *Agent) assistantToolHistoryContent(explanation, reasoningContent string) (string, string) {
	return explanation, reasoningContent
}

// executeToolOnly はツールを実行して結果を履歴に追加する（assistant メッセージは追加しない）。
// addToolCallsToHistory でバッチ化済みの場合に使用する。
func (a *Agent) executeToolOnly(toolCall *tools.ToolCall) string {
	// ツール実行
	result, change := a.executeToolWithSpinner(a.currentRequestContext(), toolCall)
	a.noteProjectMapMutation(toolCall, change)
	a.appendSessionToolExecution(toolCall, result)

	// str_replace エラー処理
	if a.handleStrReplaceErrors(toolCall, result) {
		return result
	}

	// comment 継続フロー処理
	if a.handleCommentFlow(toolCall, result) {
		return result
	}

	// 変更履歴を保存
	a.handleFileChange(change)

	// 結果を履歴に追加
	a.appendToolResultToHistory(toolCall, result)

	_, _ = fmt.Fprintln(a.output())
	return result
}

// addToolCallToHistory はツール呼び出しを会話履歴に追加する
func (a *Agent) addToolCallToHistory(response string, toolCall *tools.ToolCall) {
	// レスポンスから説明部分とツール呼び出しを分離
	explanation, _ := extractExplanationAndTool(response)

	// reasoning_content を取得（DeepSeek Reasoner 対応）
	reasoningContent := a.getLastReasoningContent()
	contentBlocks := a.getLastAnthropicContentBlocks()

	// Function Calling: tool_call_id がある場合は OpenAI 形式で履歴に追加
	isFunctionCalling := toolCall.ID != ""
	if isFunctionCalling {
		historyContent, historyReasoning := a.assistantToolHistoryContent(explanation, reasoningContent)

		// assistant メッセージに tool_calls を含める
		msg := api.Message{
			Role:             "assistant",
			Content:          historyContent, // 説明部分のみ（ツール呼び出しは ToolCalls に）
			ReasoningContent: historyReasoning,
			ToolCalls: []api.OpenAIToolCall{{
				ID:               toolCall.ID,
				Type:             "function",
				ThoughtSignature: toolCall.ThoughtSignature,
				ThoughtParts:     toolCall.ThoughtParts,
				Function: api.OpenAIToolCallFunction{
					Name:      toolCall.Tool,
					Arguments: toolruntime.ArgsToJSON(toolCall.RawArgs),
				},
			}},
		}
		msg.SetAnthropicContentBlocks(contentBlocks)
		a.History = append(a.History, msg)
	} else {
		// テキストベースのツール呼び出し（従来方式）
		a.History = append(a.History, api.Message{
			Role:             "assistant",
			Content:          response,
			ReasoningContent: reasoningContent,
		})
	}

	// FC パスの場合、セッションに assistant+ToolCalls を保存
	if isFunctionCalling && a.session != nil {
		msg := a.History[len(a.History)-1]
		a.appendSessionMessageFromAPI(msg, a.CurrentModel)
	}

	// 統計情報更新: Assistantメッセージ数とツール実行回数をカウント
	if a.Stats != nil {
		a.Stats.AssistantMessages++
		a.Stats.AddToolExecution(toolCall.Tool)
	}
}

// shouldAbortToolLoop は同じツール呼び出しの繰り返しを検知（後方互換性のため）
func (a *Agent) shouldAbortToolLoop(current, last *tools.ToolCall, count *int) bool {
	// 空のレスポンスで shouldAbortToolLoopWithResponse を呼び出す
	return a.shouldAbortToolLoopWithResponse("", current, last, count)
}

// shouldAbortToolLoopWithResponse は同じツール呼び出しの繰り返しを検知（response パラメータ付き）
func (a *Agent) shouldAbortToolLoopWithResponse(response string, current, last *tools.ToolCall, count *int) bool {
	cfg := a.cfg()
	threshold := cfg.LoopDetection.Threshold

	if isSameToolCall(current, last) {
		*count++
		if *count >= threshold {
			yellow.Fprintf(a.output(), "⚠️  Warning: Same tool call repeated %d times, stopping to prevent infinite loop\n", *count)
			yellow.Fprintf(a.output(), "   Tool: %s\n", current.Tool)

			// ツール呼び出しを履歴に追加（response がある場合のみ）
			if response != "" {
				a.addToolCallToHistory(response, current)
			}

			// ツール呼び出しに対する応答を追加（OpenAI API の要件を満たすため）
			// Function Calling 形式かテキストベースかで処理を分ける
			if current.ID != "" {
				// Function Calling 形式: role="tool" で tool_call_id 付き
				a.History = append(a.History, api.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("[SYSTEM] Tool loop detected: %s was called %d times. Stopping to prevent infinite loop.", current.Tool, threshold),
					ToolCallID: current.ID,
					ToolName:   current.Tool,
				})
			} else {
				// テキストベース: role="user" で送信
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: fmt.Sprintf("[SYSTEM WARNING] The same tool call was repeated %d times. Please try a different approach or ask the user for clarification.", threshold),
				})
			}
			return true
		}
	} else {
		*count = 1
	}
	return false
}

// executeToolCallWithResult はツールを実行して結果を履歴に追加し、結果を返す
func (a *Agent) executeToolCallWithResult(response string, toolCall *tools.ToolCall) string {
	result := a.executeToolCallInternal(response, toolCall)
	return result
}

// executeToolCall はツールを実行して結果を履歴に追加（後方互換性のため維持）
func (a *Agent) executeToolCall(response string, toolCall *tools.ToolCall) {
	a.executeToolCallInternal(response, toolCall)
}

// executeToolCallInternal はツールを実行して結果を履歴に追加（内部実装）
func (a *Agent) executeToolCallInternal(response string, toolCall *tools.ToolCall) string {
	// NOTE: 説明テキストはストリーミング中に既に表示済みのため、ここでは表示しない
	// （Issue #114: 二重表示の修正）

	// ツール呼び出しを履歴に追加
	a.addToolCallToHistory(response, toolCall)

	// ツール実行
	result, change := a.executeToolWithSpinner(a.currentRequestContext(), toolCall)
	a.appendSessionToolExecution(toolCall, result)

	// str_replace エラー処理
	if a.handleStrReplaceErrors(toolCall, result) {
		return result
	}

	// comment 継続フロー処理
	if a.handleCommentFlow(toolCall, result) {
		return result
	}

	// 変更履歴を保存
	a.handleFileChange(change)

	// 結果を履歴に追加
	a.appendToolResultToHistory(toolCall, result)

	_, _ = fmt.Fprintln(a.output())
	return result
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
		cfg := a.cfg()
		threshold := cfg.LoopDetection.Threshold
		if threshold < 2 {
			threshold = 2
		}
		if a.strReplaceErrorCount >= threshold {
			out := a.output()
			yellow.Fprintf(out, "⚠️  str_replace failed %d times consecutively. Stopping to prevent loop.\n", a.strReplaceErrorCount)
			yellow.Fprintln(out, "💡 Suggested alternatives / 代替案:")
			_, _ = fmt.Fprintln(out, "   1. Use gather_context to inspect the target file or symbol before retrying")
			_, _ = fmt.Fprintln(out, "   2. Use read_file/search_code only if you need exact low-level control")
			_, _ = fmt.Fprintln(out, "   3. Ask the user for clarification on what to change")
			_, _ = fmt.Fprintln(out, "   4. Try delete_lines + insert_before/insert_after for line-based edits")
			_, _ = fmt.Fprintln(out)

			// AIに警告を送信
			content := toolruntime.FormatTextToolResultContent(toolCall.Tool, result)
			a.History = append(a.History, toolruntime.BuildToolResultMessage(toolCall, content, content))
			a.History = append(a.History, api.Message{
				Role: "user",
				Content: `[SYSTEM WARNING] str_replace has failed multiple times. The old_str pattern was not found in the file.

Suggested next steps:
1. Use gather_context to inspect the target file or symbol
2. Use read_file/search_code only if you need exact low-level control
3. Ask the user for clarification
4. Try a different approach (delete_lines + insert_before/insert_after)

IMPORTANT: Do NOT retry str_replace with the same or similar old_str pattern. Take a different approach.`,
			})
			a.strReplaceErrorCount = 0 // リセット
			_, _ = fmt.Fprintln(a.output())
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
	if !strings.HasPrefix(result, "[COMMENT]") {
		return false
	}

	// ループ防止（同一コメントの連続は抑止）
	cfg := a.cfg()
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
		yellow.Fprintf(a.output(), "⚠️  Feedback loop detected (%d), stopping comment retry\n", feedbackCount)
		return false
	}

	// 結果を履歴に追加（Function Calling形式を考慮）
	a.History = append(a.History, toolruntime.BuildToolResultMessage(toolCall, result, toolruntime.FormatTextToolResultContent(toolCall.Tool, result)))

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
	_, _ = fmt.Fprintln(a.output())
	return true
}

// handleFileChange は変更履歴を保存
func (a *Agent) handleFileChange(change *tools.FileChange) {
	a.mutationTracker().RecordFileChange(change)
}

func (a *Agent) noteProjectMapMutation(tc *tools.ToolCall, change *tools.FileChange) {
	a.mutationTracker().NoteProjectMapMutation(tc, change)
}

// ToolExecCallback は executeToolCallsWithParallel の呼び出し元が各結果を処理するコールバック。
// 元のインデックス順で呼ばれる（skip / loopAbort されたツールには呼ばれない）。
