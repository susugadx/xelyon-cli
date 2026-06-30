package agent

import "time"

func (r *headlessRunner) successResult() *HeadlessResult {
	duration := time.Since(r.startedAt).Milliseconds()
	result := r.attachSummary(attachHeadlessStats(r.agent, NewSuccessResult(r.provider.Name(), r.model, r.finalReply, r.toolCalls, duration)))
	if r.cancelledErr != nil {
		return promoteHeadlessCancelledResult(result, r.cancelledErr)
	}
	if r.finalCheckFailed {
		return promoteHeadlessFinalCheckFailedResult(result)
	}
	if r.options.FailOnToolError && r.readOnlyViolation {
		return promoteHeadlessReadOnlyViolationResult(result)
	}
	if r.options.FailOnToolError && hasFailedHeadlessToolCall(r.toolCalls) {
		return promoteHeadlessToolErrorResult(result)
	}
	return result
}

func (r *headlessRunner) errorResult(errType, errMsg string) *HeadlessResult {
	duration := time.Since(r.startedAt).Milliseconds()
	return r.attachSummary(attachHeadlessStats(r.agent, NewErrorResult(r.provider.Name(), r.model, errType, errMsg, duration)))
}

func (r *headlessRunner) loopLimitResult(limit int) *HeadlessResult {
	duration := time.Since(r.startedAt).Milliseconds()
	result := NewToolLoopLimitResult(r.provider.Name(), r.model, limit, r.toolCalls, duration)
	if r.options.FailOnToolError && r.readOnlyViolation {
		result = promoteHeadlessReadOnlyViolationResult(result)
	}
	return r.attachSummary(attachHeadlessStats(r.agent, result))
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

func hasFailedHeadlessToolCall(toolCalls []ToolCallResult) bool {
	for _, call := range toolCalls {
		if !call.Success {
			return true
		}
	}
	return false
}
