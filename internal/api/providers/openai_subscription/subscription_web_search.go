package openaisubscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
)

const (
	subscriptionWebSearchInstructions = "Use web search to answer the query. Return a concise summary of the most relevant findings and include source URLs when available."
	subscriptionWebSearchToolType     = "web_search"
	subscriptionWebSearchToolChoice   = "required"
)

type subscriptionWebSearchRequest struct {
	Model          string                           `json:"model"`
	Input          []openairesponses.InputItem      `json:"input"`
	Instructions   string                           `json:"instructions,omitempty"`
	Stream         bool                             `json:"stream"`
	Store          bool                             `json:"store"`
	Reasoning      *openairesponses.ReasoningConfig `json:"reasoning,omitempty"`
	Tools          []subscriptionWebSearchTool      `json:"tools"`
	ToolChoice     string                           `json:"tool_choice"`
	PromptCacheKey string                           `json:"prompt_cache_key,omitempty"`
}

type subscriptionWebSearchTool struct {
	Type string `json:"type"`
}

func init() {
	websearch.RegisterWithContext(subscriptionProviderKey, WebSearchWithContext)
}

// WebSearchWithContext は OAuth subscription backend を使って OpenAI Subscription native web_search を実行します。
func WebSearchWithContext(ctx context.Context, query, model string) (string, error) {
	result, err := runSubscriptionWebSearchForDiagnostics(ctx, query, model)
	if err != nil {
		return "", err
	}
	return formatSubscriptionWebSearchResult(result), nil
}

func runSubscriptionWebSearchForDiagnostics(ctx context.Context, query, model string) (subscriptionWebSearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	model = api.ResolveProviderRequestModel(ctx, model, subscriptionProviderKey)
	if err := ValidateSubscriptionModel(model); err != nil {
		return subscriptionWebSearchResult{}, err
	}
	endpoint, err := validateSubscriptionResponsesEndpoint(DefaultSubscriptionAuthConfig().Endpoint)
	if err != nil {
		return subscriptionWebSearchResult{}, err
	}
	provider := NewSubscription()
	return provider.runSubscriptionWebSearch(ctx, endpoint, query, model)
}

func (p *SubscriptionProvider) runSubscriptionWebSearch(ctx context.Context, endpoint, query, model string) (subscriptionWebSearchResult, error) {
	if p == nil {
		return subscriptionWebSearchResult{}, fmt.Errorf("openai_subscription provider is nil")
	}
	requestBody := buildSubscriptionWebSearchRequest(ctx, query, model)
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return subscriptionWebSearchResult{}, fmt.Errorf("failed to marshal subscription web search request: %w", err)
	}
	req, err := p.prepareSubscriptionResponsesRequest(ctx, endpoint, payload)
	if err != nil {
		return subscriptionWebSearchResult{}, err
	}
	resp, err := p.executeSubscriptionResponsesRequest(req, true)
	if err != nil {
		return subscriptionWebSearchResult{}, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return subscriptionWebSearchResult{}, handleSubscriptionHTTPError(resp, nil, subscriptionDisplayName)
	}

	result, err := parseSubscriptionWebSearchStream(ctx, resp)
	if err != nil {
		return subscriptionWebSearchResult{}, err
	}
	if err := validateSubscriptionWebSearchRuntimeResult(result); err != nil {
		return subscriptionWebSearchResult{}, err
	}
	if result.Usage != nil {
		if callback := websearch.UsageCallbackFromContext(ctx); callback != nil {
			callback(*result.Usage)
		}
	}
	return result, nil
}

func buildSubscriptionWebSearchRequest(ctx context.Context, query, model string) subscriptionWebSearchRequest {
	modelIdentity := subscriptionModelIdentity(ctx, model)
	return subscriptionWebSearchRequest{
		Model:        modelIdentity.RequestName(),
		Input:        buildSubscriptionWebSearchInput(query),
		Instructions: subscriptionWebSearchInstructions,
		Stream:       true,
		Store:        false,
		Reasoning:    subscriptionResponsesReasoningConfig(ctx, modelIdentity),
		Tools: []subscriptionWebSearchTool{{
			Type: subscriptionWebSearchToolType,
		}},
		ToolChoice:     subscriptionWebSearchToolChoice,
		PromptCacheKey: openairesponses.BuildPromptCacheKey(modelIdentity.RequestName(), subscriptionWebSearchInstructions),
	}
}

func buildSubscriptionWebSearchInput(query string) []openairesponses.InputItem {
	return []openairesponses.InputItem{{
		Type:    "message",
		Role:    "user",
		Content: fmt.Sprintf("Search the web for the query below and return concise findings with source URLs.\n\nQuery: %s", query),
	}}
}
