package agent

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
)

func (r *headlessRunner) handleAssistantResponse(ctx context.Context, response string) bool {
	parsedCalls := r.agent.parseToolCalls(response)
	if r.options.ReadOnly {
		parsedCalls = append(parsedCalls, headlessReadOnlyMCPXMLToolAttempts(response)...)
	}
	assignHeadlessRescueToolCallIDs(parsedCalls)

	// ツール呼び出しがなければ最終レスポンスとして終了
	if len(parsedCalls) == 0 {
		r.finalReply = response
		r.runFinalChecksIfNeeded(ctx)
		return true
	}

	appendHeadlessToolCallsToHistory(r.agent, response, parsedCalls)
	r.executeToolCalls(ctx, parsedCalls)

	// 最大イテレーション到達時のフォールバック
	r.finalReply = response
	return false
}

func (r *headlessRunner) executeToolCalls(ctx context.Context, calls []*tools.ToolCall) {
	for _, tc := range calls {
		r.executeToolCall(ctx, tc)
	}
}

func (r *headlessRunner) executeToolCall(ctx context.Context, tc *tools.ToolCall) {
	toolCount := len(r.toolCalls) + 1
	subagent.EmitEvent(ctx, subagent.SubAgentEvent{
		Tool:      tc.Tool,
		Phase:     "start",
		FilePath:  extractToolFilePath(tc),
		ToolIndex: toolCount,
	})

	execResult, denied := r.readOnlyDeniedToolResult(tc)
	if denied {
		r.readOnlyViolation = true
	} else {
		execResult = r.agent.executeQuietToolResult(ctx, tc, strings.NewReader(""), io.Discard, io.Discard, true)
		r.agent.recordSkillActivationFromToolResult(tc, execResult.Result, execResult.Error)
		r.agent.mutationTracker().recordExecutedToolResult(tc, execResult, true)
	}
	output := execResult.Result

	success := isHeadlessToolCallSuccessForTool(tc.Tool, execResult)
	if r.agent.Stats != nil {
		r.agent.Stats.AddToolExecution(tc.Tool)
	}
	if summary, ok := newHeadlessCommandSummary(tc, execResult); ok {
		r.commands = append(r.commands, summary)
	}

	r.toolCalls = append(r.toolCalls, ToolCallResult{
		Tool:    tc.Tool,
		Args:    tc.Args,
		Output:  output,
		Success: success,
	})

	event := subagent.SubAgentEvent{
		Tool:      tc.Tool,
		Phase:     "end",
		FilePath:  extractToolFilePath(tc),
		Success:   success,
		Output:    truncateEventOutput(output, 200),
		ToolIndex: toolCount,
	}
	if tc.Tool == "str_replace" {
		event.OldStr = tc.Args["old_str"]
		event.NewStr = tc.Args["new_str"]
	}
	subagent.EmitEvent(ctx, event)
	appendHeadlessToolResultToHistory(r.agent, tc, output)
}

// isHeadlessToolCallSuccess は headless 実行におけるツール結果の成功判定を返す。
// tools 層の共通契約（Error prefix 判定と Error flag）に揃えることで、
// 先頭空白付きの "Error:" を失敗として扱いつつ、文中の "Error:" は許容する。
func isHeadlessToolCallSuccess(execResult tools.ExecutionResult) bool {
	return !execResult.Error && !tools.IsErrorResult(execResult.Result)
}

func isHeadlessToolCallSuccessForTool(toolName string, execResult tools.ExecutionResult) bool {
	if !isHeadlessToolCallSuccess(execResult) {
		return false
	}
	if toolName == subagent.WaitAgentToolName && headlessWaitAgentResponseHasFailure(execResult.Result) {
		return false
	}
	return true
}

func headlessWaitAgentResponseHasFailure(output string) bool {
	var response subagent.WaitResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return false
	}
	if isHeadlessWaitAgentFailureStatus(response.Status) {
		return true
	}
	for _, result := range response.Results {
		if isHeadlessWaitAgentFailureStatus(result.Status) {
			return true
		}
		for _, entry := range result.ToolBreakdown {
			if entry.Failures > 0 {
				return true
			}
		}
	}
	return false
}

func isHeadlessWaitAgentFailureStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "completed":
		return false
	default:
		return true
	}
}
