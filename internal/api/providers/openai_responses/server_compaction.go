package openairesponses

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

const (
	// minServerCompactionCompactThreshold は Responses API へ送る compact_threshold の最小値。
	minServerCompactionCompactThreshold = 1000
	// provider 既定 max_output_tokens fallback を使う時に最低限残す入力 budget。
	minProviderFallbackInputBudgetTokens = 8192
)

// ServerCompactionDecision は request payload と local auto-compress 制御の判定結果。
type ServerCompactionDecision struct {
	ContextManagement              []ContextManagementSetting
	ShouldSkipLocalAutoCompression bool
}

// ResolveServerCompactionDecision は Responses API request 用の server compaction 判定を解決する。
func ResolveServerCompactionDecision(ctx context.Context, provider string, model ModelIdentity, previousResponseID string) ServerCompactionDecision {
	cfg := config.FromContext(ctx)
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if !cfg.ResponsesServerCompactionEnabled() || strings.TrimSpace(previousResponseID) == "" {
		return ServerCompactionDecision{}
	}

	compactThreshold, ok := resolveServerCompactionThreshold(ctx, cfg, provider, model)
	if !ok {
		return ServerCompactionDecision{
			ShouldSkipLocalAutoCompression: !cfg.ResponsesServerCompactionLocalFallbackEnabled(),
		}
	}

	return ServerCompactionDecision{
		ContextManagement:              BuildServerCompactionContextManagement(compactThreshold),
		ShouldSkipLocalAutoCompression: true,
	}
}

func resolveServerCompactionThreshold(ctx context.Context, cfg *config.Config, provider string, model ModelIdentity) (int, bool) {
	configured := cfg.ResponsesServerCompactionCompactThreshold()
	if configured > 0 {
		if configured < minServerCompactionCompactThreshold {
			return 0, false
		}
		return configured, true
	}
	return resolveAutoServerCompactionThreshold(ctx, cfg, provider, model)
}

func resolveAutoServerCompactionThreshold(ctx context.Context, cfg *config.Config, provider string, model ModelIdentity) (int, bool) {
	catalogModel := cfg.ModelCatalogName(provider, model.RequestName())
	contextLimit, ok := llmcatalog.KnownModelContextLimit(catalogModel)
	if !ok || contextLimit <= 0 {
		return 0, false
	}

	triggerPercent := cfg.Compression.TriggerPercent
	if triggerPercent == 0 {
		triggerPercent = 80
	}
	threshold := contextLimit * triggerPercent / 100
	if headroomThreshold, ok := resolveOutputHeadroomThreshold(ctx, cfg, provider, model, contextLimit); ok && headroomThreshold > 0 && headroomThreshold < threshold {
		threshold = headroomThreshold
	}

	if threshold < minServerCompactionCompactThreshold {
		threshold = minServerCompactionCompactThreshold
	}
	return threshold, true
}

func resolveOutputHeadroomThreshold(ctx context.Context, cfg *config.Config, provider string, model ModelIdentity, contextLimit int) (int, bool) {
	if contextLimit <= 0 {
		return 0, false
	}

	maxOutputTokens, ok := resolveRequestMaxOutputTokens(ctx, cfg, provider, model, contextLimit)
	if !ok || maxOutputTokens <= 0 {
		return 0, false
	}

	inputBudget := contextLimit - maxOutputTokens
	if inputBudget <= 0 {
		return 1, true
	}
	return inputBudget, true
}

func resolveRequestMaxOutputTokens(_ context.Context, cfg *config.Config, provider string, model ModelIdentity, contextLimit int) (int, bool) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if hasKnownModelMaxOutputTokens(cfg, provider, model) {
		return api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), provider, model.RequestName()), true
	}

	if !hasProviderMaxOutputTokens(cfg, provider, model) {
		return 0, false
	}

	maxOutputTokens := api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), provider, model.RequestName())
	if !providerFallbackLeavesUsableInputBudget(contextLimit, maxOutputTokens) {
		return 0, false
	}
	return maxOutputTokens, true
}

func hasKnownModelMaxOutputTokens(cfg *config.Config, provider string, model ModelIdentity) bool {
	requestModel := model.RequestName()
	if override, ok := cfg.ModelOverrideForProvider(provider, requestModel); ok && override.MaxOutputTokens > 0 {
		return true
	}
	_, ok := llmcatalog.KnownMaxOutputTokens(model.CatalogName())
	return ok
}

func hasProviderMaxOutputTokens(cfg *config.Config, provider string, model ModelIdentity) bool {
	lookupProvider := cfg.RuntimeProviderConfigKey(provider, model.RequestName())
	providerCfg, ok := cfg.GetProviderModelConfig(lookupProvider)
	return ok && providerCfg.MaxOutputTokens > 0
}

func providerFallbackLeavesUsableInputBudget(contextLimit, maxOutputTokens int) bool {
	if contextLimit <= 0 || maxOutputTokens <= 0 {
		return false
	}

	inputBudget := contextLimit - maxOutputTokens
	if inputBudget <= 0 {
		return false
	}

	minInputBudget := contextLimit / 10
	if minInputBudget < minProviderFallbackInputBudgetTokens {
		minInputBudget = minProviderFallbackInputBudgetTokens
	}
	return inputBudget >= minInputBudget
}

// BuildServerCompactionContextManagement は Responses API の context_management.compaction payload を構築する。
func BuildServerCompactionContextManagement(compactThreshold int) []ContextManagementSetting {
	if compactThreshold < minServerCompactionCompactThreshold {
		return nil
	}
	return []ContextManagementSetting{{
		Type:             "compaction",
		CompactThreshold: compactThreshold,
	}}
}

// RequestUsesServerCompaction は request payload に compaction 設定が載っているか返す。
func RequestUsesServerCompaction(req Request) bool {
	for _, setting := range req.ContextManagement {
		if setting.Type == "compaction" && setting.CompactThreshold > 0 {
			return true
		}
	}
	return false
}
