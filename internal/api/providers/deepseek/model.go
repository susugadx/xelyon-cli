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

func normalizeDeepSeekModel(model string) string {
	trimmed := strings.TrimSpace(model)
	switch strings.ToLower(trimmed) {
	case "", "deepseek-chat", "deepseek-reasoner", "deepseek-v4-flash":
		return "deepseek-v4-flash"
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
	if !model.supportsV4Thinking() {
		return nil, "", ""
	}

	thinkingType := "disabled"
	if api.IsThinkingEnabled(ctx) {
		thinkingType = "enabled"
		cfg := config.FromContext(ctx)
		reasoningEffort = deepSeekReasoningEffort(cfg.Thinking.Level)
		spinnerSuffix = "Reasoner"
	}

	extraFields = map[string]any{
		"thinking": map[string]any{"type": thinkingType},
	}
	return extraFields, reasoningEffort, spinnerSuffix
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
