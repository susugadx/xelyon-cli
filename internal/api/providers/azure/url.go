package azure

import (
	"net/url"
	"strings"
)

func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(raw, "/")
	}

	path := strings.TrimRight(parsed.Path, "/")
	switch path {
	case "", "/":
		parsed.Path = azureOpenAIBasePath
	case "/openai":
		parsed.Path = azureOpenAIBasePath
	default:
		parsed.Path = path
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func joinEndpoint(baseURL, endpoint string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpoint = strings.TrimLeft(strings.TrimSpace(endpoint), "/")
	if baseURL == "" {
		return "/" + endpoint
	}
	if endpoint == "" {
		return baseURL
	}
	return baseURL + "/" + endpoint
}
