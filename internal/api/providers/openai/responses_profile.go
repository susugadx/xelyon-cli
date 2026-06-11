package openai

import (
	"context"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type responsesPayloadMode string
type responsesPromptCacheRetentionPolicy string
type responsesStorePolicy string
type responsesPreviousResponseIDPolicy string
type responsesContextManagementPolicy string
type responsesStreamPolicy string
type responsesMaxOutputTokensPolicy string

const (
	responsesPayloadModeDefault     responsesPayloadMode = "default"
	responsesPayloadModeFullPayload responsesPayloadMode = "full_payload"

	responsesPromptCacheRetention24h  responsesPromptCacheRetentionPolicy = "24h"
	responsesPromptCacheRetentionOmit responsesPromptCacheRetentionPolicy = "omit"

	responsesStoreFromConfig responsesStorePolicy = "from_config"
	responsesStoreForceFalse responsesStorePolicy = "force_false"

	responsesPreviousResponseIDEnabled  responsesPreviousResponseIDPolicy = "enabled"
	responsesPreviousResponseIDDisabled responsesPreviousResponseIDPolicy = "disabled"

	responsesContextManagementEnabled  responsesContextManagementPolicy = "enabled"
	responsesContextManagementDisabled responsesContextManagementPolicy = "disabled"

	responsesStreamFromModel responsesStreamPolicy = "from_model"
	responsesStreamForceTrue responsesStreamPolicy = "force_true"

	responsesMaxOutputTokensCatalogOrConfig responsesMaxOutputTokensPolicy = "catalog_or_config"
	responsesMaxOutputTokensOmit            responsesMaxOutputTokensPolicy = "omit"
)

type responsesRuntimeProfile struct {
	ProviderKey string
	DisplayName string
	DebugName   string

	ResponsesURL   func() string
	PrepareRequest func(ctx context.Context, url string, payload []byte) (*http.Request, error)

	ModelCatalogProviderKey string
	ConfigProviderKey       string
	CostFamily              string

	SupportsCompletions bool
	SupportsResponses   bool

	PayloadMode                responsesPayloadMode
	PromptCacheRetentionPolicy responsesPromptCacheRetentionPolicy
	StorePolicy                responsesStorePolicy
	PreviousResponseIDPolicy   responsesPreviousResponseIDPolicy
	ContextManagementPolicy    responsesContextManagementPolicy
	StreamPolicy               responsesStreamPolicy
	MaxOutputTokensPolicy      responsesMaxOutputTokensPolicy
	IncludeInstructions        bool
}

func openAIResponsesRuntimeProfile() responsesRuntimeProfile {
	return responsesRuntimeProfile{
		ProviderKey:                "openai",
		DisplayName:                "OpenAI",
		DebugName:                  "OpenAI",
		ResponsesURL:               resolveResponsesAPIURL,
		ModelCatalogProviderKey:    "openai",
		ConfigProviderKey:          "openai",
		CostFamily:                 "openai",
		SupportsCompletions:        true,
		SupportsResponses:          true,
		PayloadMode:                responsesPayloadModeDefault,
		PromptCacheRetentionPolicy: responsesPromptCacheRetention24h,
		StorePolicy:                responsesStoreFromConfig,
		PreviousResponseIDPolicy:   responsesPreviousResponseIDEnabled,
		ContextManagementPolicy:    responsesContextManagementEnabled,
		StreamPolicy:               responsesStreamFromModel,
		MaxOutputTokensPolicy:      responsesMaxOutputTokensCatalogOrConfig,
		IncludeInstructions:        false,
	}
}

func (p responsesRuntimeProfile) modelIdentity(ctx context.Context, model string) openairesponses.ModelIdentity {
	cfg := config.FromContext(ctx)
	return openairesponses.NewModelIdentity(model, cfg.ModelCatalogName(p.ModelCatalogProviderKey, model))
}

func (p responsesRuntimeProfile) maxOutputTokens(ctx context.Context, model openairesponses.ModelIdentity) int {
	if p.MaxOutputTokensPolicy == responsesMaxOutputTokensOmit {
		return 0
	}
	return api.GetMaxOutputTokens(ctx, p.ConfigProviderKey, model.RequestName())
}

func (p responsesRuntimeProfile) stream(model openairesponses.ModelIdentity) bool {
	if p.StreamPolicy == responsesStreamForceTrue {
		return true
	}
	return shouldStreamResponses(model.CatalogName())
}

func (p responsesRuntimeProfile) store(ctx context.Context) bool {
	if p.StorePolicy == responsesStoreForceFalse {
		return false
	}
	return config.FromContext(ctx).ResponsesStoreEnabled()
}

func (p responsesRuntimeProfile) promptCacheRetention() string {
	switch p.PromptCacheRetentionPolicy {
	case responsesPromptCacheRetention24h:
		return "24h"
	default:
		return ""
	}
}

func (p responsesRuntimeProfile) previousResponseID(ctx context.Context, responseID string, activeContext []api.ActiveContextBlock) string {
	if p.PreviousResponseIDPolicy == responsesPreviousResponseIDDisabled {
		return ""
	}
	previousResponseID := previousResponseIDForRequest(ctx, responseID)
	return openairesponses.PreviousResponseIDForRequestContext(ctx, previousResponseID, activeContext)
}

func (p responsesRuntimeProfile) serverCompactionDecision(ctx context.Context, model openairesponses.ModelIdentity, previousResponseID string) openairesponses.ServerCompactionDecision {
	if p.ContextManagementPolicy == responsesContextManagementDisabled || previousResponseID == "" {
		return openairesponses.ServerCompactionDecision{}
	}
	return openairesponses.ResolveServerCompactionDecision(ctx, p.ProviderKey, model, previousResponseID)
}
