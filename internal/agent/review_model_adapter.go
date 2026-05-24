package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const reviewModelProviderCacheNamespace = "review_model"

// agentReviewModel は ReviewRunner の model 境界を Agent の provider runtime へ接続する。
type agentReviewModel struct {
	agent *Agent
}

func (m agentReviewModel) CompleteReview(ctx context.Context, req review.ReviewModelRequest) (review.ReviewModelResponse, error) {
	a := m.agent
	target, err := a.currentReviewModelTarget()
	if err != nil {
		return review.ReviewModelResponse{}, fmt.Errorf("review model %s: %w", req.Phase, err)
	}

	restoreResponseID := suspendReviewModelResponseContinuation(target.provider)
	defer restoreResponseID()

	content, err := target.provider.ChatWithTools(
		a.reviewModelRequestContext(ctx),
		"",
		reviewModelPromptHistory(req.Prompt),
		target.model,
	)
	if err != nil {
		return review.ReviewModelResponse{}, fmt.Errorf("review model %s: %w", req.Phase, err)
	}
	return review.ReviewModelResponse{Content: content}, nil
}

type reviewModelTarget struct {
	provider api.Provider
	model    string
}

func (a *Agent) currentReviewModelTarget() (reviewModelTarget, error) {
	if a == nil {
		return reviewModelTarget{}, fmt.Errorf("agent is nil")
	}
	if a.CurrentProvider == nil {
		return reviewModelTarget{}, fmt.Errorf("provider is nil")
	}
	return reviewModelTarget{
		provider: a.CurrentProvider,
		model:    a.CurrentModel,
	}, nil
}

func reviewModelPromptHistory(prompt string) []api.Message {
	return []api.Message{{Role: "user", Content: prompt}}
}

func (a *Agent) reviewModelRequestContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = api.WithoutActiveContextBlocks(ctx)
	ctx = tools.WithRegistry(ctx, a.registry())
	ctx = tools.WithConfig(ctx, a.cfg())
	ctx = ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	ctx = api.WithProviderCacheNamespace(ctx, reviewModelProviderCacheNamespace)
	ctx = api.WithToolUseDisabled(ctx)
	ctx = api.WithToolDefinitions(ctx, nil)
	return api.WithAdditionalToolDefinitionsDisabled(ctx)
}

func suspendReviewModelResponseContinuation(provider api.Provider) func() {
	if provider == nil {
		return func() {}
	}
	ridProvider, ok := provider.(ResponseIDCapable)
	if !ok {
		return func() {}
	}

	previousResponseID := ridProvider.GetResponseID()
	ridProvider.SetResponseID("")
	return func() {
		ridProvider.SetResponseID(previousResponseID)
	}
}
