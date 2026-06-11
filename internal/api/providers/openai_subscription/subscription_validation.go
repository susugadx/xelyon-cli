package openaisubscription

import (
	"fmt"
	"net/url"
	"strings"
)

const openAIPlatformResponsesURL = "https://api.openai.com/v1/responses"

func validateSubscriptionOriginatorForRequest(originator string) (string, error) {
	originator = strings.TrimSpace(originator)
	if originator == "" {
		originator = subscriptionDefaultOriginator
	}
	if originator != subscriptionDefaultOriginator {
		return "", fmt.Errorf("subscription originator must be %s: %s", subscriptionDefaultOriginator, originator)
	}
	return subscriptionDefaultOriginator, nil
}

func validateSubscriptionResponsesEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("subscription endpoint is not configured")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("subscription endpoint is invalid: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("subscription endpoint must use http or https: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("subscription endpoint must include a host: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	if subscriptionResponsesEndpointForbidden(parsed) {
		return "", fmt.Errorf("subscription endpoint must not use OpenAI Platform Responses API endpoint: %s", RedactSubscriptionEndpointForDisplay(endpoint))
	}
	return endpoint, nil
}

func subscriptionResponsesEndpointForbidden(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	return strings.EqualFold(parsed.Hostname(), "api.openai.com") && (path == "/v1/responses" || strings.HasPrefix(path, "/v1/responses/"))
}
