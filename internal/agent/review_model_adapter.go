package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/setup"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

const reviewModelProviderCacheNamespace = "review_model"

var newReviewModelProvider = api.NewProvider

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
	cfg := a.cfg()
	reviewProvider := strings.TrimSpace(cfg.Review.Provider)
	reviewModel := strings.TrimSpace(cfg.Review.Model)
	if reviewProvider != "" {
		return a.configuredReviewModelTarget(cfg, reviewProvider, reviewModel)
	}
	if reviewModel != "" {
		return reviewModelTarget{}, fmt.Errorf("review.model requires review.provider")
	}
	if a.CurrentProvider == nil {
		return reviewModelTarget{}, fmt.Errorf("provider is nil")
	}
	return reviewModelTarget{
		provider: a.CurrentProvider,
		model:    a.CurrentModel,
	}, nil
}

func (a *Agent) configuredReviewModelTarget(cfg *config.Config, requestedProvider, requestedModel string) (reviewModelTarget, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	providerConfigKey := config.ActiveProviderConfigKey(requestedProvider)
	runtimeProviderName := config.CanonicalProviderName(requestedProvider)
	if _, ok := config.ProviderCatalogEntryFor(runtimeProviderName); !ok {
		return reviewModelTarget{}, fmt.Errorf("unknown review provider: %s", requestedProvider)
	}
	if providerConfigKey == "" {
		providerConfigKey = runtimeProviderName
	}
	if setup.ProviderSetupRequired(runtimeProviderName) {
		return reviewModelTarget{}, fmt.Errorf("%s", setup.ProviderSetupRequiredMessage(runtimeProviderName))
	}

	provider, err := newReviewModelProvider(providerConfigKey)
	if err != nil {
		return reviewModelTarget{}, fmt.Errorf("review provider initialization failed: %w", err)
	}
	api.ApplyRuntimeConfig(provider, cfg)
	api.ApplyUIRuntime(provider, a.ui())
	syncProviderConfigKeyToProvider(provider, providerConfigKey)
	if key := providerConfigKeyFromProvider(provider); key != "" {
		providerConfigKey = key
	}

	explicitModel := requestedModel != ""
	model := requestedModel
	if model == "" {
		model = strings.TrimSpace(cfg.GetSelectedModelForProvider(providerConfigKey))
	}
	if err := validateProviderModelSelection(cfg, runtimeProviderName, providerConfigKey, model, explicitModel); err != nil {
		return reviewModelTarget{}, err
	}
	setReviewModelUsageReporter(a, provider, providerConfigKey, model)

	return reviewModelTarget{
		provider: provider,
		model:    model,
	}, nil
}

func setReviewModelUsageReporter(agent *Agent, provider api.Provider, providerConfigKey, model string) {
	if agent == nil || agent.Stats == nil {
		return
	}
	reporter, ok := provider.(api.UsageReporter)
	if !ok {
		return
	}
	reporter.SetUsageCallback(func(u api.Usage) {
		agent.statsMu.Lock()
		defer agent.statsMu.Unlock()
		if agent.Stats == nil {
			return
		}
		agent.Stats.AddUsageForProviderConfig(agent.cfg(), providerConfigKey, model, u)
	})
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
	ctx = tools.WithConfig(ctx, a.reviewModelRequestConfig())
	ctx = uiruntime.WithRuntime(ctx, uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	ctx = api.WithProviderCacheNamespace(ctx, reviewModelProviderCacheNamespace)
	ctx = api.WithToolUseDisabled(ctx)
	ctx = api.WithToolDefinitions(ctx, nil)
	return api.WithAdditionalToolDefinitionsDisabled(ctx)
}

func (a *Agent) reviewModelRequestConfig() *config.Config {
	cfg := a.cfg()
	effectiveThinking := config.ResolveReviewThinkingConfig(cfg.Thinking, cfg.Review.Thinking)
	if effectiveThinking == cfg.Thinking {
		return cfg
	}
	reviewCfg := config.CloneConfig(cfg)
	reviewCfg.Thinking = effectiveThinking
	return reviewCfg
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
