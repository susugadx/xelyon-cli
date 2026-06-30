package agent

import (
	"context"
	"strings"
	"time"
)

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
	var runtimeDirectives []string
	if strings.TrimSpace(r.agent.cfg().SubAgentPrompt) == "" {
		runtimeDirectives = r.agent.pendingRuntimeDirectives()
		effectivePrompt = r.agent.normalModeSystemPromptForRequestWithDirectives(reqCtx, r.query, iteration == 0, runtimeDirectives)
	}
	requestCtx := r.agent.prepareResponseContextForPrompt(r.agent.requestContext(reqCtx), effectivePrompt)
	if iteration == 0 && r.options.Image != nil {
		requestCtx, history := r.agent.providerFacingHistoryExcludingLatestMessageForRequest(requestCtx)
		response, err := r.provider.ChatWithImage(requestCtx, effectivePrompt, history, r.query, r.options.Image, r.model)
		if err != nil {
			return "", err
		}
		r.agent.recordResponseContextForPrompt(effectivePrompt)
		r.agent.markRuntimeDirectivesDelivered(runtimeDirectives)
		return response, nil
	}

	requestCtx, history := r.agent.providerFacingHistoryForRequest(requestCtx)
	response, err := r.provider.ChatWithTools(requestCtx, effectivePrompt, history, r.model)
	if err != nil {
		return "", err
	}
	r.agent.recordResponseContextForPrompt(effectivePrompt)
	r.agent.markRuntimeDirectivesDelivered(runtimeDirectives)
	return response, nil
}
