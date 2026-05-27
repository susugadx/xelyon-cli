package deepseek

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultDeepSeekModel = "deepseek-v4-flash"

type deepSeekModelSelection struct {
	actualModel  string
	catalogModel string
}

type deepSeekThinkingPolicy struct {
	Supported       bool
	Enabled         bool
	Type            string
	ReasoningEffort string
	SpinnerSuffix   string
	ExtraFields     map[string]any
}

func normalizeDeepSeekModel(model string) string {
	trimmed := strings.TrimSpace(model)
	switch strings.ToLower(trimmed) {
	case "", "deepseek-chat", "deepseek-reasoner", defaultDeepSeekModel:
		return defaultDeepSeekModel
	case "deepseek-v4-pro":
		return "deepseek-v4-pro"
	default:
		return trimmed
	}
}

func resolveDeepSeekModelSelection(ctx context.Context, requestedModel string) deepSeekModelSelection {
	actualModel := normalizeDeepSeekModel(requestedModel)
	catalogModel := normalizeDeepSeekModel(config.FromContext(ctx).ModelCatalogName("deepseek", requestedModel))
	if strings.TrimSpace(catalogModel) == "" {
		catalogModel = actualModel
	}
	return deepSeekModelSelection{
		actualModel:  actualModel,
		catalogModel: catalogModel,
	}
}

func (s deepSeekModelSelection) supportsV4Thinking() bool {
	return isDeepSeekV4Model(s.catalogModel)
}

func isDeepSeekV4Model(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "deepseek-v4")
}

func deepSeekThinkingConfig(ctx context.Context, model deepSeekModelSelection) (extraFields map[string]any, reasoningEffort string, spinnerSuffix string) {
	policy := resolveDeepSeekThinkingPolicy(ctx, model)
	return policy.ExtraFields, policy.ReasoningEffort, policy.SpinnerSuffix
}

func resolveDeepSeekThinkingPolicy(ctx context.Context, model deepSeekModelSelection) deepSeekThinkingPolicy {
	if !model.supportsV4Thinking() {
		return deepSeekThinkingPolicy{}
	}

	policy := deepSeekThinkingPolicy{
		Supported: true,
		Type:      "disabled",
		ExtraFields: map[string]any{
			"thinking": map[string]any{"type": "disabled"},
		},
	}
	if api.IsThinkingEnabled(ctx) {
		cfg := config.FromContext(ctx)
		policy.Enabled = true
		policy.Type = "enabled"
		policy.ReasoningEffort = deepSeekReasoningEffort(cfg.Thinking.Level)
		policy.SpinnerSuffix = "Reasoner"
		policy.ExtraFields = map[string]any{
			"thinking": map[string]any{"type": "enabled"},
		}
	}
	return policy
}

func deepSeekReasoningEffort(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "xhigh", "max":
		return "max"
	case "low", "medium", "high":
		return "high"
	default:
		return "high"
	}
}
