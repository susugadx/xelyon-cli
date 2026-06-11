package openaisubscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const diagnosticRouteSubscriptionCompact = "subscription_compact"
const openAIPlatformCompactURL = "https://api.openai.com/v1/responses/compact"

// SupportsCompact は subscription Compact API が設定されているか返します。
func (p *SubscriptionProvider) SupportsCompact() bool {
	return strings.TrimSpace(DefaultSubscriptionAuthConfig().CompactEndpoint) != ""
}

// CompactHistory は ChatGPT/Codex OAuth subscription Compact API で会話履歴を圧縮します。
func (p *SubscriptionProvider) CompactHistory(ctx context.Context, input []api.InputItem, model, instructions string) (*api.CompactResponse, error) {
	if p == nil {
		p = NewSubscription()
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = SubscriptionDefaultUtilityModel()
	}
	if err := ValidateSubscriptionModel(model); err != nil {
		return nil, err
	}
	return p.runSubscriptionCompactRequest(ctx, DefaultSubscriptionAuthConfig().CompactEndpoint, CompactRequest{
		Model:        model,
		Input:        api.CloneInputItems(input),
		Instructions: instructions,
	})
}

func runSubscriptionCompactProbe(ctx context.Context, provider *SubscriptionProvider, endpoint, model string) (*api.CompactResponse, error) {
	if endpoint == "" {
		endpoint = DefaultSubscriptionAuthConfig().CompactEndpoint
	}
	if provider == nil {
		provider = NewSubscription()
	}
	return provider.runSubscriptionCompactRequest(ctx, endpoint, CompactRequest{
		Model:        model,
		Input:        subscriptionCompactProbeInput(),
		Instructions: "Compact the diagnostic input without adding new facts.",
	})
}

func (p *SubscriptionProvider) runSubscriptionCompactRequest(ctx context.Context, endpoint string, compactRequest CompactRequest) (*api.CompactResponse, error) {
	endpoint, err := validateSubscriptionCompactEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p = NewSubscription()
	}
	payload, err := json.Marshal(compactRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription compact request: %w", err)
	}
	req, err := p.prepareSubscriptionResponsesRequest(ctx, endpoint, payload)
	if err != nil {
		return nil, err
	}
	resp, err := p.executeSubscriptionLongRunningRequest(req)
	if err != nil {
		return nil, fmt.Errorf("subscription compact request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, handleSubscriptionHTTPError(resp, nil, subscriptionDisplayName)
	}
	var compactResp api.CompactResponse
	if err := json.NewDecoder(resp.Body).Decode(&compactResp); err != nil {
		return nil, fmt.Errorf("failed to decode subscription compact response: %w", err)
	}
	if len(compactResp.Output) == 0 {
		return nil, fmt.Errorf("subscription compact response did not include output items")
	}
	return &compactResp, nil
}

func subscriptionCompactProbeInput() []api.InputItem {
	return []api.InputItem{
		{Type: "message", Role: "developer", Content: "You are compacting a XELYON diagnostic conversation."},
		{Type: "message", Role: "user", Content: "The subscription compact smoke should preserve this diagnostic fact."},
	}
}

func validateSubscriptionCompactEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("subscription Compact API endpoint is not configured")
	}
	if subscriptionCompactEndpointForbidden(endpoint) {
		return "", fmt.Errorf("subscription Compact API must not use OpenAI Platform Compact API endpoint: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("subscription Compact API endpoint is invalid: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("subscription Compact API endpoint must use http or https: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("subscription Compact API endpoint must include a host: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	return endpoint, nil
}

func subscriptionCompactEndpointForbidden(endpoint string) bool {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.EqualFold(endpoint, openAIPlatformCompactURL) {
		return true
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	return strings.EqualFold(parsed.Hostname(), "api.openai.com") && (path == "/v1/responses/compact" || strings.HasPrefix(path, "/v1/responses/compact/"))
}
