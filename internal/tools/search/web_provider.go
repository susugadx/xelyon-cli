package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

type webSearchModelResolution struct {
	Model        string
	AdjustedFrom string
}

func resolveSearchProvider(cfg *config.Config, mainProvider, mainProviderConfigKey string) string {
	if cfg != nil {
		provider := normalizeProviderName(cfg.WebSearch.Provider)
		if resolved := resolveNativeSearchProviderKey(provider); resolved != "" {
			return resolved
		}
	}

	provider := normalizeProviderName(mainProviderConfigKey)
	if resolved := resolveNativeSearchProviderKey(provider); resolved != "" {
		return resolved
	}

	provider = normalizeProviderName(mainProvider)
	if resolved := resolveNativeSearchProviderKey(provider); resolved != "" {
		return resolved
	}

	return ""
}

func resolveSearchModel(cfg *config.Config, searchProvider, mainProvider, mainModel string) webSearchModelResolution {
	model := ""
	if config.SameProviderRuntimeIdentity(searchProvider, mainProvider) {
		model = strings.TrimSpace(mainModel)
		if model == "" && cfg != nil {
			model = cfg.GetSelectedModelForProvider(searchProvider)
		}
	} else if cfg != nil {
		model = cfg.GetSelectedModelForProvider(searchProvider)
	}
	return applyNativeWebSearchModelPolicy(cfg, searchProvider, model)
}

func applyNativeWebSearchModelPolicy(cfg *config.Config, searchProvider, model string) webSearchModelResolution {
	resolved := webSearchModelResolution{Model: model}
	if !config.SameProviderRuntimeIdentity(searchProvider, "kimi") {
		return resolved
	}
	catalogModel := ""
	if cfg != nil {
		catalogModel = cfg.ModelCatalogName(searchProvider, model)
	}
	requestModel, adjusted := llmcatalog.KimiBuiltinWebSearchRequestModel(model, catalogModel)
	if !adjusted {
		return resolved
	}
	resolved.Model = requestModel
	resolved.AdjustedFrom = strings.TrimSpace(model)
	if resolved.AdjustedFrom == "" {
		resolved.AdjustedFrom = strings.TrimSpace(catalogModel)
	}
	return resolved
}

func webSearchOwnerLabel(provider string, model webSearchModelResolution) string {
	label := provider
	if strings.TrimSpace(model.Model) != "" {
		label += "/" + strings.TrimSpace(model.Model)
	}
	if strings.TrimSpace(model.AdjustedFrom) != "" {
		label += ", adjusted from " + strings.TrimSpace(model.AdjustedFrom) + " for Kimi $web_search"
	}
	return label
}

func resolveNativeSearchProviderKey(provider string) string {
	if provider == "" {
		return ""
	}
	entry, ok := llmcatalog.ProviderDescriptorFor(provider)
	if !ok || !entry.NativeWebSearch {
		return ""
	}
	if entry.Key == "openai_subscription" {
		return entry.Key
	}
	return provider
}

func webSearchProviderError() string {
	return fmt.Sprintf(`Web search requires a provider with native search support.
Set web_search.provider in config.yaml to one of: %s

Example:
  web_search:
    provider: gemini

Gemini API key is free at https://aistudio.google.com/apikey`, strings.Join(llmcatalog.NativeWebSearchProviderKeys(true), ", "))
}

func normalizeProviderName(providerName string) string {
	return config.NormalizeProviderName(providerName)
}
