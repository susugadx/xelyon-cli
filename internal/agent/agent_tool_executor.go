package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// argsToJSON は RawArgs を JSON 文字列に変換
func argsToJSON(args map[string]any) string {
	if len(args) == 0 {
		return "{}"
	}
	b, err := json.Marshal(args)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (a *Agent) toolExecutionContext(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) tools.ExecutionContext {
	runtimeUI := a.ui()
	if ctx == nil {
		ctx = a.currentRequestContext()
	}
	if stdin == nil {
		stdin = runtimeUI.Input()
	}
	if stdout == nil {
		stdout = runtimeUI.Output()
	}
	if stderr == nil {
		stderr = runtimeUI.ErrorOutput()
	}
	return tools.ExecutionContext{
		Context:      ctx,
		ProviderName: a.ProviderName,
		Model:        a.CurrentModel,
		Stdin:        stdin,
		Stdout:       stdout,
		Stderr:       stderr,
		PromptReader: runtimeUI.PromptReader(),
		Runtime:      runtimeUI,
		Registry:     a.registry(),
		ToolCache:    a.ToolCache,
		LSPClient:    a.GetLSPClient(),
		Config:       a.cfg(),
		AutoApprove:  a.autoApprove(),
		AuditLogger:  a.auditLogger(),
	}
}

func (a *Agent) currentRequestContext() context.Context {
	if a != nil && a.requestCtx != nil {
		return a.requestCtx
	}
	return context.Background()
}

func (a *Agent) executeToolWithSpinner(ctx context.Context, toolCall *tools.ToolCall) (string, *tools.FileChange) {
	// ネガティブキャッシュチェック（ブロックせずログ表示のみ）
	if a.ToolCache != nil {
		if result, hit := a.ToolCache.CheckNegativeCache(toolCall.Tool, toolCall.RawArgs); hit {
			yellow.Fprintf(a.output(), "⚠ Negative cache hit: %s previously returned: %s\n", toolCall.Tool, result)
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	spinner := a.ui().NewSpinner()
	spinner.Start(ui.SpinnerMessageForTool(toolCall.Tool))
	a.ui().SetSpinner(spinner)

	result, change := tools.ExecuteWithContext(a.toolExecutionContext(ctx, nil, nil, nil), toolCall)
	a.ui().StopSpinner()
	a.recordToolResultOptimizations(toolCall.Tool, result)

	// エラー/空結果をネガティブキャッシュに記録
	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(toolCall.Tool, toolCall.RawArgs, result)
	}

	return result, change
}

// addToolCallsToHistory はパラレル FC の全ツール呼び出しを1つの assistant メッセージにまとめて履歴に追加する。
// ThoughtSignature/ThoughtParts は最初の ToolCall からのみ取得（Gemini 3 仕様）。
func (a *Agent) addToolCallsToHistory(response string, toolCalls []*tools.ToolCall) {
	if len(toolCalls) == 0 {
		return
	}

	explanation, _ := extractExplanationAndTool(response)
	reasoningContent := a.getLastReasoningContent()

	openAIToolCalls := make([]api.OpenAIToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		openAIToolCalls[i] = api.OpenAIToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      tc.Tool,
				Arguments: argsToJSON(tc.RawArgs),
			},
		}
		// ThoughtSignature/ThoughtParts は最初の ToolCall のみ（Gemini 3 仕様）
		if i == 0 {
			openAIToolCalls[i].ThoughtSignature = tc.ThoughtSignature
			openAIToolCalls[i].ThoughtParts = tc.ThoughtParts
		}
	}

	a.History = append(a.History, api.Message{
		Role:             "assistant",
		Content:          explanation,
		ReasoningContent: reasoningContent,
		ToolCalls:        openAIToolCalls,
	})

	// セッションに保存（1回のみ）
	if a.session != nil {
		msg := a.History[len(a.History)-1]
		a.session.AddMessageFromAPI(msg, a.CurrentModel)
	}

	// 統計情報更新: AssistantMessages は1回、ToolExecution は各ツール
	if a.Stats != nil {
		a.Stats.AssistantMessages++
		for _, tc := range toolCalls {
			a.Stats.AddToolExecution(tc.Tool)
		}
	}
}

// executeToolOnly はツールを実行して結果を履歴に追加する（assistant メッセージは追加しない）。
// addToolCallsToHistory でバッチ化済みの場合に使用する。
func (a *Agent) executeToolOnly(toolCall *tools.ToolCall) string {
	// ツール実行
	result, change := a.executeToolWithSpinner(a.currentRequestContext(), toolCall)

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

	// 結果を履歴に追加（重複チェック → 入口圧縮）
	historyContent := a.compactToolResult(toolCall, result)
	if toolCall.ID != "" {
		// Function Calling: role="tool" で tool_call_id 付きで送信
		toolMsg := api.Message{
			Role:       "tool",
			Content:    historyContent,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Tool,
		}
		a.History = append(a.History, toolMsg)

		// セッションに tool result を保存
		if a.session != nil {
			a.session.AddMessageFromAPI(toolMsg, a.CurrentModel)
		}
	} else {
		// テキストベース: role="user" で送信（従来方式）
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, historyContent),
		})
	}

	_, _ = fmt.Fprintln(a.output())
	return result
}

// addToolCallToHistory はツール呼び出しを会話履歴に追加する
func (a *Agent) addToolCallToHistory(response string, toolCall *tools.ToolCall) {
	// レスポンスから説明部分とツール呼び出しを分離
	explanation, _ := extractExplanationAndTool(response)

	// reasoning_content を取得（DeepSeek Reasoner 対応）
	reasoningContent := a.getLastReasoningContent()

	// Function Calling: tool_call_id がある場合は OpenAI 形式で履歴に追加
	isFunctionCalling := toolCall.ID != ""
	if isFunctionCalling {
		// assistant メッセージに tool_calls を含める
		a.History = append(a.History, api.Message{
			Role:             "assistant",
			Content:          explanation, // 説明部分のみ（ツール呼び出しは ToolCalls に）
			ReasoningContent: reasoningContent,
			ToolCalls: []api.OpenAIToolCall{{
				ID:               toolCall.ID,
				Type:             "function",
				ThoughtSignature: toolCall.ThoughtSignature,
				ThoughtParts:     toolCall.ThoughtParts,
				Function: api.OpenAIToolCallFunction{
					Name:      toolCall.Tool,
					Arguments: argsToJSON(toolCall.RawArgs),
				},
			}},
		})
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
		a.session.AddMessageFromAPI(msg, a.CurrentModel)
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

	// 結果を履歴に追加（重複チェック → 入口圧縮）
	historyContent := a.compactToolResult(toolCall, result)
	if toolCall.ID != "" {
		// Function Calling: role="tool" で tool_call_id 付きで送信
		toolMsg := api.Message{
			Role:       "tool",
			Content:    historyContent,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Tool,
		}
		a.History = append(a.History, toolMsg)

		// セッションに tool result を保存
		if a.session != nil {
			a.session.AddMessageFromAPI(toolMsg, a.CurrentModel)
		}
	} else {
		// テキストベース: role="user" で送信（従来方式）
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, historyContent),
		})
	}

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
			_, _ = fmt.Fprintln(out, "   1. Use read_file to verify the exact content of the target file")
			_, _ = fmt.Fprintln(out, "   2. Use search_code to find the correct string pattern")
			_, _ = fmt.Fprintln(out, "   3. Ask the user for clarification on what to change")
			_, _ = fmt.Fprintln(out, "   4. Try delete_lines + insert_before/insert_after for line-based edits")
			_, _ = fmt.Fprintln(out)

			// AIに警告を送信
			if toolCall.ID != "" {
				// Function Calling 形式: role="tool" で tool_call_id 付き
				a.History = append(a.History, api.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
					ToolCallID: toolCall.ID,
					ToolName:   toolCall.Tool,
				})
			} else {
				// テキストベース: role="user" で送信
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
				})
			}
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
	if !strings.Contains(result, "[COMMENT]") {
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
	isFunctionCalling := toolCall.ID != ""
	if isFunctionCalling {
		a.History = append(a.History, api.Message{
			Role:       "tool",
			Content:    result,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Tool,
		})
	} else {
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
		})
	}

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
			yellow.Fprintf(a.output(), "Warning: Failed to persist change: %v\n", err)
		}
	}
}

// ToolExecCallback は executeToolCallsWithParallel の呼び出し元が各結果を処理するコールバック。
// 元のインデックス順で呼ばれる（skip / loopAbort されたツールには呼ばれない）。
type ToolExecCallback func(idx int, tc *tools.ToolCall, result string, change *tools.FileChange)

type toolExecResult struct {
	result string
	change *tools.FileChange
}

// executeToolForParallel は並列実行用のツール実行関数。
// goroutine から安全に呼び出せるよう、以下を省略している:
//   - spinner / SetGlobalSpinner: goroutine 内で global spinner を操作すると競合する
//   - stdout/stderr: io.Discard を使って補助出力を抑制する
//
// stdout 出力の抑制:
//
//	tools.ExecuteQuietWithContext を使用し、wrapper 層の出力（ヘッダー・引数・折りたたみ結果）を抑制する。
//	Tool.Run() 内部の補助的な出力は execCtx.Stdout=io.Discard に流し、process stdout へは出さない。
//
// cancel の実効性（best effort）:
//
//	ctx.Err() を実行前にチェックして早期リターンする。
//	Tool.Run() まで request context を伝播するが、各ツールがそれを使うかは個別実装次第。
//	現状は bash など context-aware なツールが実行中キャンセルを拾える。
func (a *Agent) executeToolForParallel(ctx context.Context, tc *tools.ToolCall) (string, *tools.FileChange) {
	if ctx.Err() != nil {
		return "Error: context cancelled", nil
	}

	// ネガティブキャッシュチェック（ToolCache は thread-safe）
	if a.ToolCache != nil {
		if _, hit := a.ToolCache.CheckNegativeCache(tc.Tool, tc.RawArgs); hit {
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	// ExecuteQuietWithContext: ヘッダー・引数・折りたたみ出力と補助 stdout を抑制（parallel path 用）
	result, change := tools.ExecuteQuietWithContext(a.toolExecutionContext(ctx, strings.NewReader(""), io.Discard, io.Discard), tc)
	a.recordToolResultOptimizations(tc.Tool, result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(tc.Tool, tc.RawArgs, result)
	}

	return result, change
}

// executeToolCallsWithParallel は parallel-safe なツールを並列実行し、sequential なツールを順次実行する。
// 通常モードと Plan Mode の両方で使用する共通 executor。
//
// NOTE: Investigation Phase（plan_investigation.go）は本 executor を使用しない。
// Investigation Phase は SafetyHigh 制約のある独自ループ（executeToolOnly）を持ち、
// 並列実行の対象外である。
//
// 処理フロー:
//
//	Phase 0: 実行前フィルタリング（元の tool call 順で走査）
//	  - loopDetectFn: ループ検知。true を返すとそのツール以降すべてを中止。
//	    履歴には変更を加えない（executor が Phase 2 でメッセージを追加する）。
//	  - skipFn: スキップ判定（例: deprecated planning tools）。
//	    (skip=true, msg) を返すとそのツールを実行せずに msg を結果として扱う。
//
//	  評価順序: loopDetectFn → skipFn（この順序は重要）
//	    - loopDetectFn を先に評価する理由: 呼び出し元の loopDetectFn は内部で
//	      lastToolCall / sameCallCount を更新する。skip 対象ツールにもループ検知を
//	      適用することで、skip 対象ツールも lastToolCall の連続性に参加する。
//	      これは旧 sequential 実装（shouldAbortToolLoopWithResponse が全ツールに
//	      適用されていた）と一致する。
//	    - skipFn はループ検知の後に評価されるため、skip 対象ツールがループの
//	      一部として数えられた後にスキップされる。
//
//	Phase 1: 実行（parallel-safe は goroutine、sequential は順次）
//	Phase 2: 結果配送（元の tool call 順で callback を呼ぶ）
//
// loop detection 履歴メッセージ:
//
//	旧実装（shouldAbortToolLoopWithResponse）と意味的に同等のメッセージを生成する:
//	  - FC trigger tool: role="tool", "[SYSTEM] Tool loop detected: ..."
//	  - text-based trigger: role="user", "[SYSTEM WARNING] ..."
//	  - FC 後続 tool: role="tool", "[SYSTEM] Skipped due to tool loop detection."
//	  - text-based 後続 tool: 何も追加しない（intentional spec, by design）
//	    → text-based パスでは tool_call_id がないため role="tool" メッセージを
//	      追加できない。旧 sequential 実装でも text-based 後続 tool にはダミーを
//	      追加していなかったため、互換性のためにこの挙動を維持している。
//
// loopDetected: ループが検知された場合に true を返す。
func (a *Agent) executeToolCallsWithParallel(
	ctx context.Context,
	allToolCalls []*tools.ToolCall,
	loopDetectFn func(tc *tools.ToolCall) (abort bool),
	skipFn func(tc *tools.ToolCall) (skip bool, msg string),
	callback ToolExecCallback,
) (loopDetected bool) {
	n := len(allToolCalls)
	if n == 0 {
		return false
	}

	// ── Phase 0: 実行前フィルタリング ──
	// 元の tool call 順で走査し、各ツールの状態を確定する。
	// ループ検知はスキップ対象ツールにも適用する（元の sequential 動作と一致）。
	const (
		statusExecute   = 0 // 実行対象
		statusSkip      = 1 // skipFn によりスキップ
		statusLoopAbort = 2 // ループ検知で中止
	)
	type entry struct {
		status  int
		skipMsg string // statusSkip 時のメッセージ
	}
	entries := make([]entry, n)
	aborted := false
	loopTriggerIdx := -1

	for i, tc := range allToolCalls {
		if aborted {
			entries[i] = entry{status: statusLoopAbort}
			continue
		}
		// loopDetectFn を skipFn より先に評価する。
		// 理由: loopDetectFn 内で lastToolCall / sameCallCount が更新されるため、
		// skip 対象ツールも連続性の判定に参加する必要がある。
		if loopDetectFn != nil && loopDetectFn(tc) {
			entries[i] = entry{status: statusLoopAbort}
			loopDetected = true
			aborted = true
			loopTriggerIdx = i
			continue
		}
		// skipFn はループ検知の後に評価する。
		if skipFn != nil {
			if skip, msg := skipFn(tc); skip {
				entries[i] = entry{status: statusSkip, skipMsg: msg}
				continue
			}
		}
		entries[i] = entry{status: statusExecute}
	}

	// ── Phase 1: 実行 ──
	// 実行対象のツールを parallel-safe / sequential に分類して実行する。
	results := make([]toolExecResult, n)

	var parallelEntries, seqEntries []int
	for i, e := range entries {
		if e.status != statusExecute {
			continue
		}
		if tools.IsParallelSafe(allToolCalls[i]) {
			parallelEntries = append(parallelEntries, i)
		} else {
			seqEntries = append(seqEntries, i)
		}
	}

	// Phase 1a: parallel-safe 群を並列実行
	//
	// 各 goroutine は個別の ExecutionContext を明示的に組み立てて
	// ExecuteQuietWithContext に渡す。process-global な実行コンテキストには依存しない。
	// parallel path では io.Discard を使い、補助 stdout を構造的に抑制する。
	//
	// cancel の実効性（best effort）:
	//   各 goroutine 起動前に ctx.Err() をチェックし、executeToolForParallel 内でも
	//   再チェックする。ただし Tool.Run() 自体は ctx を受け取らないため、
	//   実行開始後のキャンセルは効かない（次の goroutine 起動時にスキップされる）。
	if len(parallelEntries) > 0 {
		startedAt := time.Now()
		parallelSpinner := a.ui().NewSpinner()
		parallelSpinner.Start(parallelGroupSpinnerMessage(allToolCalls, parallelEntries))
		a.ui().SetSpinner(parallelSpinner)

		sem := make(chan struct{}, tools.MaxParallelTools)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, idx := range parallelEntries {
			if ctx.Err() != nil {
				mu.Lock()
				results[idx] = toolExecResult{result: "Error: context cancelled"}
				mu.Unlock()
				continue
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(i int) {
				defer wg.Done()
				defer func() { <-sem }()

				r, c := a.executeToolForParallel(ctx, allToolCalls[i])
				mu.Lock()
				results[i] = toolExecResult{result: r, change: c}
				mu.Unlock()
			}(idx)
		}
		wg.Wait()
		a.ui().StopSpinner()
		printParallelToolGroup(a.output(), a.cfg(), allToolCalls, parallelEntries, results, time.Since(startedAt))
	}

	// Phase 1b: sequential 群を順次実行（spinner あり）
	for _, idx := range seqEntries {
		if ctx.Err() != nil {
			results[idx] = toolExecResult{result: "Error: context cancelled"}
			continue
		}
		r, c := a.executeToolWithSpinner(ctx, allToolCalls[idx])
		results[idx] = toolExecResult{result: r, change: c}
	}

	// ── Phase 2: 結果配送（元の tool call 順） ──
	// 旧 shouldAbortToolLoopWithResponse と意味的に同等のメッセージを生成する。
	cfg := a.cfg()
	threshold := cfg.LoopDetection.Threshold

	for i := 0; i < n; i++ {
		tc := allToolCalls[i]
		e := entries[i]

		switch e.status {
		case statusSkip:
			// スキップされたツール: skipMsg を tool result として履歴に追加
			if tc.ID != "" {
				// FC: role="tool" + tool_call_id
				toolMsg := api.Message{
					Role:       "tool",
					Content:    e.skipMsg,
					ToolCallID: tc.ID,
					ToolName:   tc.Tool,
				}
				a.History = append(a.History, toolMsg)
				if a.session != nil {
					a.session.AddMessageFromAPI(toolMsg, a.CurrentModel)
				}
			} else {
				// text-based: role="user"
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: fmt.Sprintf("[Tool Result for %s]\n%s", tc.Tool, e.skipMsg),
				})
			}

		case statusLoopAbort:
			if i == loopTriggerIdx {
				// ループを引き起こしたツール: 検知メッセージを追加
				// （旧 shouldAbortToolLoopWithResponse と同じフォーマット）
				if tc.ID != "" {
					a.History = append(a.History, api.Message{
						Role:       "tool",
						Content:    fmt.Sprintf("[SYSTEM] Tool loop detected: %s was called %d times. Stopping to prevent infinite loop.", tc.Tool, threshold),
						ToolCallID: tc.ID,
						ToolName:   tc.Tool,
					})
				} else {
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: fmt.Sprintf("[SYSTEM WARNING] The same tool call was repeated %d times. Please try a different approach or ask the user for clarification.", threshold),
					})
				}
			} else {
				// ループ後の残りのツール: FC の場合のみダミー結果を追加。
				// text-based（ID=""）の場合は何も追加しない。
				// これは intentional spec であり、旧 sequential 実装と一致する挙動を
				// by design で維持している。text-based パスでは tool_call_id がないため
				// role="tool" メッセージを追加できず、role="user" のダミーを
				// 追加しても LLM の文脈を汚すだけであるため省略している。
				if tc.ID != "" {
					a.History = append(a.History, api.Message{
						Role:       "tool",
						Content:    "[SYSTEM] Skipped due to tool loop detection.",
						ToolCallID: tc.ID,
						ToolName:   tc.Tool,
					})
				}
			}

		case statusExecute:
			r := results[i]
			if callback != nil {
				callback(i, tc, r.result, r.change)
			}
		}
	}

	return loopDetected
}

func printParallelToolGroup(out io.Writer, cfg *config.Config, allToolCalls []*tools.ToolCall, indices []int, results []toolExecResult, elapsed time.Duration) {
	if len(indices) == 0 {
		return
	}

	if len(indices) == 1 {
		idx := indices[0]
		_, _ = fmt.Fprintln(out, ui.FormatToolLine(ui.ToolDisplayInfo{
			ToolName: allToolCalls[idx].Tool,
			Args:     allToolCalls[idx].Args,
			Result:   results[idx].result,
			Error:    strings.HasPrefix(strings.TrimSpace(results[idx].result), "Error:"),
		}))
		printParallelCollapsedOutput(out, cfg, allToolCalls[idx].Tool, results[idx].result)
		return
	}

	ui.PrintParallelGroupStartToWriter(out, len(indices))
	for _, idx := range indices {
		ui.PrintParallelGroupLineToWriter(out, ui.FormatToolLine(ui.ToolDisplayInfo{
			ToolName: allToolCalls[idx].Tool,
			Args:     allToolCalls[idx].Args,
			Result:   results[idx].result,
			Error:    strings.HasPrefix(strings.TrimSpace(results[idx].result), "Error:"),
		}))
		printParallelCollapsedOutputWithPrefix(out, cfg, allToolCalls[idx].Tool, results[idx].result, "│    ")
	}
	ui.PrintParallelGroupEndToWriter(out, formatParallelGroupSummary(allToolCalls, indices, elapsed))
}

func shouldShowParallelCollapsed(toolName, result string) bool {
	trimmed := strings.TrimSpace(result)
	if strings.HasPrefix(trimmed, "Error:") {
		return true
	}
	if toolName == "search_code" && strings.Contains(result, "\n") {
		return true
	}
	return false
}

func printParallelCollapsedOutput(out io.Writer, cfg *config.Config, toolName, result string) {
	if !shouldShowParallelCollapsed(toolName, result) {
		return
	}
	_, _ = fmt.Fprintln(out, ui.FormatToolOutput(result, ui.GetMaxVisibleLinesWithConfig(cfg)))
}

func printParallelCollapsedOutputWithPrefix(out io.Writer, cfg *config.Config, toolName, result, prefix string) {
	if !shouldShowParallelCollapsed(toolName, result) {
		return
	}

	collapsed := ui.FormatToolOutput(result, ui.GetMaxVisibleLinesWithConfig(cfg))
	for _, line := range strings.Split(strings.TrimRight(collapsed, "\n"), "\n") {
		if line == "" {
			continue
		}
		_, _ = fmt.Fprintf(out, "%s%s\n", prefix, line)
	}
}

func formatParallelGroupSummary(allToolCalls []*tools.ToolCall, indices []int, elapsed time.Duration) string {
	counts := make(map[string]int)
	var order []string

	addCount := func(label string) {
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}

	for _, idx := range indices {
		switch allToolCalls[idx].Tool {
		case "read_file", "read_files", "list_dir":
			addCount("reads")
		case "search_code":
			addCount("searches")
		case "web_search":
			addCount("web")
		case "bash":
			addCount("bash")
		default:
			addCount(allToolCalls[idx].Tool)
		}
	}

	var parts []string
	for _, label := range order {
		count := counts[label]
		switch label {
		case "reads", "searches":
			parts = append(parts, fmt.Sprintf("%d %s", count, label))
		case "web":
			parts = append(parts, fmt.Sprintf("%d web", count))
		case "bash":
			parts = append(parts, fmt.Sprintf("%d bash", count))
		default:
			if count == 1 {
				parts = append(parts, fmt.Sprintf("%d %s", count, label))
			} else {
				parts = append(parts, fmt.Sprintf("%d %ss", count, label))
			}
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Done (%s)", ui.FormatParallelElapsed(elapsed))
	}
	return fmt.Sprintf("Done: %s (%s)", strings.Join(parts, ", "), ui.FormatParallelElapsed(elapsed))
}

func parallelGroupSpinnerMessage(allToolCalls []*tools.ToolCall, indices []int) string {
	if len(indices) == 0 {
		return "Running parallel tools..."
	}

	counts := make(map[string]int)
	for _, idx := range indices {
		switch allToolCalls[idx].Tool {
		case "read_file", "read_files", "list_dir":
			counts["reads"]++
		case "search_code":
			counts["searches"]++
		case "inspect_symbol":
			counts["inspects"]++
		case "web_search":
			counts["web"]++
		default:
			counts["tools"]++
		}
	}

	switch {
	case counts["reads"] > 0 && len(counts) == 1:
		return "Reading files..."
	case counts["searches"] > 0 && len(counts) == 1:
		return "Searching code..."
	case counts["inspects"] > 0 && len(counts) == 1:
		return "Inspecting symbols..."
	default:
		return fmt.Sprintf("Running %d parallel tools...", len(indices))
	}
}
