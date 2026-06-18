package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type headlessRunner struct {
	agent      *Agent
	provider   api.Provider
	model      string
	query      string
	startedAt  time.Time
	toolCalls  []ToolCallResult
	finalReply string
	initErr    error
}

// RunHeadlessWithConfig は指定設定で Headless モードのクエリを実行する。
// ctx が Done になるとサブエージェント含め処理を中断する。
func RunHeadlessWithConfig(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *HeadlessResult {
	startedAt := time.Now()
	runner := newHeadlessRunner(query, model, provider, cfg)
	runner.startedAt = startedAt
	defer runner.agent.Cleanup()
	return runner.run(ctx)
}

func newHeadlessRunner(query, model string, provider api.Provider, cfg *config.Config) *headlessRunner {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.AutoApprove = true
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	configureRuntimeAuditLoggerFromEnv(runtime, io.Discard, false)

	agent := NewAgentWithRuntime(model, provider, true, runtime)
	agent.setAutoApprove(true) // Headlessモードは自動承認（SafetyLow以外）

	if cfg != nil && cfg.SubAgentPrompt != "" {
		agent.SystemPrompt = prompt.BuildProviderSystemPromptWithConfig(cfg.SubAgentPrompt, agent.ProviderName, model, agent.cfg())
	}

	// プロジェクト instruction 読み込み（xelyon.yaml + guidance）
	initErr := initializeProjectInstructions(agent, projectInstructionApplyOptions{
		showStatus:       false,
		injectProjectMap: true,
	})

	// Headless Mode は Normal Mode 相当: 親と同じツール除外
	allowSubAgents := cfg == nil || cfg.SubAgentPrompt == ""
	toolVisibility := resolveToolVisibilityPolicyWithConfig(agent.ProviderName, model, agent.cfg(), toolSurfacePhaseNormal, toolVisibilityOptions{
		allowSubAgents: allowSubAgents,
	})
	agent.registry().SetExcludedTools(agent.excludedToolsForVisibilityPolicy(toolVisibility))

	// 初期ユーザーメッセージをHistoryに追加
	agent.History = append(agent.History, api.Message{
		Role:    "user",
		Content: query,
	})

	return &headlessRunner{
		agent:    agent,
		provider: provider,
		model:    model,
		query:    query,
		initErr:  initErr,
	}
}

func (r *headlessRunner) run(ctx context.Context) *HeadlessResult {
	if r.initErr != nil {
		return r.errorResult("config_error", r.initErr.Error())
	}

	maxIterations := normalizeToolLoopLimit(r.agent.cfg().General.ToolLoopLimit)

	for iteration := 0; maxIterations == 0 || iteration < maxIterations; iteration++ {
		if ctx.Err() != nil {
			return r.errorResult(HeadlessErrorTypeCancelled, ctx.Err().Error())
		}

		response, err := r.requestAssistantResponse(ctx, iteration)
		if err != nil {
			return r.errorResult(HeadlessErrorTypeAPI, err.Error())
		}

		if done := r.handleAssistantResponse(ctx, response); done {
			return r.successResult()
		}
	}

	if maxIterations > 0 {
		return r.loopLimitResult(maxIterations)
	}
	return r.successResult()
}

func (r *headlessRunner) requestAssistantResponse(ctx context.Context, iteration int) (string, error) {
	timeout := time.Duration(r.agent.cfg().APIRetry.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 3600 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// ツールループ初回のみ Project Map を更新。ループ中の再生成はキャッシュを破壊する。
	if iteration == 0 {
		r.agent.refreshProjectPromptIfDirty(r.query)
	}

	effectivePrompt := r.agent.SystemPrompt
	if strings.TrimSpace(r.agent.cfg().SubAgentPrompt) == "" {
		effectivePrompt = r.agent.normalModeSystemPromptForRequest(reqCtx, r.query, iteration == 0)
	}
	requestCtx := r.agent.prepareResponseContextForPrompt(r.agent.requestContext(reqCtx), effectivePrompt)
	requestCtx, history := r.agent.providerFacingHistoryForRequest(requestCtx)
	response, err := r.provider.ChatWithTools(requestCtx, effectivePrompt, history, r.model)
	if err != nil {
		return "", err
	}
	r.agent.recordResponseContextForPrompt(effectivePrompt)
	return response, nil
}

func (r *headlessRunner) handleAssistantResponse(ctx context.Context, response string) bool {
	parsedCalls := r.agent.parseToolCalls(response)
	assignHeadlessRescueToolCallIDs(parsedCalls)

	// ツール呼び出しがなければ最終レスポンスとして終了
	if len(parsedCalls) == 0 {
		r.finalReply = response
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

	execResult := r.agent.executeQuietToolResult(ctx, tc, strings.NewReader(""), io.Discard, io.Discard, true)
	output, change := execResult.Result, execResult.Change
	r.agent.noteProjectMapMutation(tc, change)
	r.agent.recordSkillActivationFromToolResult(tc, output, execResult.Error)

	success := isHeadlessToolCallSuccess(execResult)
	if r.agent.Stats != nil {
		r.agent.Stats.AddToolExecution(tc.Tool)
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

	// ファイル変更履歴を記録
	if change != nil {
		r.agent.appendChange(*change)
	}
}

// isHeadlessToolCallSuccess は headless 実行におけるツール結果の成功判定を返す。
// tools 層の共通契約（Error prefix 判定と Error flag）に揃えることで、
// 先頭空白付きの "Error:" を失敗として扱いつつ、文中の "Error:" は許容する。
func isHeadlessToolCallSuccess(execResult tools.ExecutionResult) bool {
	return !execResult.Error && !tools.IsErrorResult(execResult.Result)
}

func (r *headlessRunner) successResult() *HeadlessResult {
	duration := time.Since(r.startedAt).Milliseconds()
	return attachHeadlessStats(r.agent, NewSuccessResult(r.provider.Name(), r.model, r.finalReply, r.toolCalls, duration))
}

func (r *headlessRunner) errorResult(errType, errMsg string) *HeadlessResult {
	duration := time.Since(r.startedAt).Milliseconds()
	return attachHeadlessStats(r.agent, NewErrorResult(r.provider.Name(), r.model, errType, errMsg, duration))
}

func (r *headlessRunner) loopLimitResult(limit int) *HeadlessResult {
	duration := time.Since(r.startedAt).Milliseconds()
	result := NewToolLoopLimitResult(r.provider.Name(), r.model, limit, r.toolCalls, duration)
	return attachHeadlessStats(r.agent, result)
}

func attachHeadlessStats(agent *Agent, result *HeadlessResult) *HeadlessResult {
	if agent == nil || result == nil || agent.Stats == nil {
		return result
	}

	agent.statsMu.Lock()
	defer agent.statsMu.Unlock()

	result.Tokens = &TokenUsage{
		Input:    agent.Stats.InputTokens,
		Cached:   agent.Stats.CachedInputTokens,
		Output:   agent.Stats.OutputTokens,
		Thinking: agent.Stats.ThinkingTokens,
		Total:    agent.Stats.TotalTokens(),
	}
	if agent.Stats.WebSearchCalls > 0 {
		result.WebSearch = &WebSearchUsage{
			Calls:        agent.Stats.WebSearchCalls,
			FeeEstimate:  agent.Stats.WebSearchCost,
			ResultTokens: agent.Stats.WebSearchResultTokens,
		}
	}
	estimate := agent.Stats.EstimatedCostEstimateForConfig(agent.cfg())
	result.Cost = estimate.Cost
	result.PricingUnavailable = estimate.PricingUnavailable
	return result
}

func assignHeadlessRescueToolCallIDs(toolCalls []*tools.ToolCall) {
	for i, tc := range toolCalls {
		if tc == nil || tc.ID != "" {
			continue
		}
		toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
	}
}

func appendHeadlessToolCallsToHistory(agent *Agent, response string, toolCalls []*tools.ToolCall) {
	if agent == nil || len(toolCalls) == 0 {
		return
	}

	explanation, _ := extractExplanationAndTool(response)
	reasoningContent := agent.getLastReasoningContent()
	contentBlocks := agent.getLastAnthropicContentBlocks()
	openAIToolCalls := buildOpenAIToolCallsForHistory(toolCalls)
	appendAssistantToolCallsHistoryMessage(agent, explanation, reasoningContent, contentBlocks, openAIToolCalls)
	if agent.Stats != nil {
		agent.Stats.AssistantMessages++
	}
}

func appendHeadlessToolResultToHistory(agent *Agent, toolCall *tools.ToolCall, result string) {
	if agent == nil || toolCall == nil {
		return
	}
	if !keepToolResultHistory(toolCall) {
		return
	}
	agent.History = append(agent.History, toolruntime.BuildToolResultMessage(toolCall, result, toolruntime.FormatTextToolResultContent(toolCall.Tool, result)))
}

// extractToolFilePath はツール呼び出しから表示用ターゲットを抽出する。
func extractToolFilePath(tc *tools.ToolCall) string {
	if tc == nil {
		return ""
	}

	switch tc.Tool {
	case "gather_context":
		if query := tc.Args["query"]; query != "" {
			return query
		}
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "read_file", "write_file", "str_replace", "delete_file", "list_dir", "lint", "format":
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "search_code":
		if pattern := tc.Args["pattern"]; pattern != "" {
			return fmt.Sprintf("%q", pattern)
		}
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "bash":
		if cmd := tc.Args["command"]; cmd != "" {
			return truncateEventOutput(cmd, 40)
		}
	}

	if rawPaths := tc.Args["paths"]; rawPaths != "" {
		var paths []string
		if err := json.Unmarshal([]byte(rawPaths), &paths); err == nil && len(paths) > 0 {
			if len(paths) == 1 {
				return paths[0]
			}
			return fmt.Sprintf("%s (+%d files)", paths[0], len(paths)-1)
		}
	}
	if path := tc.Args["path"]; path != "" {
		return path
	}
	if pattern := tc.Args["pattern"]; pattern != "" {
		return pattern
	}
	if symbol := tc.Args["symbol"]; symbol != "" {
		return symbol
	}
	return ""
}

// truncateEventOutput はイベント出力を制限する。
func truncateEventOutput(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
