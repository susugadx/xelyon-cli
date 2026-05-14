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
	if a == nil {
		return review.ReviewModelResponse{}, fmt.Errorf("review model %s: agent is nil", req.Phase)
	}
	if a.CurrentProvider == nil {
		return review.ReviewModelResponse{}, fmt.Errorf("review model %s: provider is nil", req.Phase)
	}

	restoreResponseID := a.suspendResponseContinuationForReviewModelCall()
	defer restoreResponseID()

	content, err := a.CurrentProvider.ChatWithTools(
		a.reviewModelRequestContext(ctx),
		"",
		[]api.Message{{Role: "user", Content: req.Prompt}},
		a.CurrentModel,
	)
	if err != nil {
		return review.ReviewModelResponse{}, fmt.Errorf("review model %s: %w", req.Phase, err)
	}
	return review.ReviewModelResponse{Content: content}, nil
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

func (a *Agent) suspendResponseContinuationForReviewModelCall() func() {
	if a == nil {
		return func() {}
	}
	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok {
		return func() {}
	}

	previousResponseID := ridProvider.GetResponseID()
	ridProvider.SetResponseID("")
	return func() {
		ridProvider.SetResponseID(previousResponseID)
	}
}
