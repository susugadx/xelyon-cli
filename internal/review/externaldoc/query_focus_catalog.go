package externaldoc

import "strings"

const (
	searchQueryFocusPreviousResponseID  = "previous_response_id"
	searchQueryFocusResponseFormat      = "response_format"
	searchQueryFocusWebSearch           = "web_search"
	searchQueryFocusResponsesAPI        = "responses API"
	searchQueryFocusFunctionCalling     = "function calling"
	searchQueryFocusToolChoice          = "tool_choice"
	searchQueryFocusToolCalls           = "tool calls"
	searchQueryFocusServiceTier         = "service_tier"
	searchQueryFocusAnthropicVersion    = "anthropic_version"
	searchQueryFocusCacheControl        = "cache_control"
	searchQueryFocusEventStream         = "text/event-stream"
	searchQueryFocusJSONSchema          = "JSON schema"
	searchQueryFocusRedirectURIField    = "redirect_uri"
	searchQueryFocusRedirectURIText     = "redirect URI"
	searchQueryFocusAccessToken         = "access token"
	searchQueryFocusAuthorizationCode   = "authorization code"
	searchQueryFocusOAuth20             = "OAuth 2.0"
	searchQueryFocusFilepathEvalSymlink = "filepath.EvalSymlinks"
	searchQueryFocusEvalSymlink         = "EvalSymlinks"
	searchQueryFocusFilepathClean       = "filepath.Clean"
	searchQueryFocusPathTraversal       = "path traversal"
	searchQueryFocusDirectoryTraversal  = "directory traversal"
	searchQueryFocusSymlink             = "symlink"
	searchQueryFocusTokenEndpoint       = "token endpoint"
)

type searchQueryFocusTokenCatalogEntry struct {
	token     string
	preferred bool
}

var searchQueryFocusTokenCatalog = []searchQueryFocusTokenCatalogEntry{
	{token: searchQueryFocusPreviousResponseID, preferred: true},
	{token: searchQueryFocusResponseFormat, preferred: true},
	{token: searchQueryFocusWebSearch, preferred: true},
	{token: searchQueryFocusResponsesAPI, preferred: true},
	{token: searchQueryFocusFunctionCalling, preferred: true},
	{token: searchQueryFocusToolChoice, preferred: true},
	{token: searchQueryFocusToolCalls, preferred: true},
	{token: searchQueryFocusServiceTier, preferred: true},
	{token: searchQueryFocusAnthropicVersion, preferred: true},
	{token: searchQueryFocusCacheControl, preferred: true},
	{token: searchQueryFocusEventStream, preferred: true},
	{token: searchQueryFocusJSONSchema, preferred: true},
	{token: searchQueryFocusRedirectURIField, preferred: true},
	{token: searchQueryFocusRedirectURIText, preferred: true},
	{token: searchQueryFocusAccessToken, preferred: true},
	{token: searchQueryFocusAuthorizationCode, preferred: true},
	{token: searchQueryFocusOAuth20, preferred: true},
	{token: searchQueryFocusFilepathEvalSymlink, preferred: true},
	{token: searchQueryFocusEvalSymlink, preferred: true},
	{token: searchQueryFocusFilepathClean, preferred: true},
	{token: searchQueryFocusPathTraversal, preferred: true},
	{token: searchQueryFocusDirectoryTraversal, preferred: true},
	{token: searchQueryFocusSymlink, preferred: true},
	{token: searchQueryFocusTokenEndpoint},
}

var searchQueryAllowedFocusTokenKeys = buildSearchQueryFocusTokenKeySet(searchQueryFocusTokenCatalog)

func searchQueryFocusTokens(genericTokens []string, corpus string) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, entry := range searchQueryFocusTokenCatalog {
		if !entry.preferred {
			continue
		}
		token := entry.token
		if !strings.Contains(corpus, strings.ToLower(token)) {
			continue
		}
		key := strings.ToLower(token)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, token)
		if len(result) >= 3 {
			return result
		}
	}
	for _, token := range genericTokens {
		token = strings.TrimSpace(token)
		if !searchQueryGenericFocusTokenIsConcrete(token) {
			continue
		}
		key := strings.ToLower(token)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, token)
		if len(result) >= 3 {
			return result
		}
	}
	return result
}

func searchQueryPreferredFocusTokenIsAllowed(token string) bool {
	_, ok := searchQueryAllowedFocusTokenKeys[searchQueryFocusTokenKey(token)]
	return ok
}

func buildSearchQueryFocusTokenKeySet(entries []searchQueryFocusTokenCatalogEntry) map[string]struct{} {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		key := searchQueryFocusTokenKey(entry.token)
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	return seen
}

func searchQueryFocusTokenKey(token string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(token)), " "))
}

func searchQueryGenericFocusTokenIsConcrete(token string) bool {
	normalized, ok := normalizeFocusTerm(token)
	if !ok {
		return false
	}
	lower := strings.ToLower(normalized)
	switch lower {
	case "api", "apis", "config", "configuration", "provider", "providers", "model", "models", "request", "requests", "response", "responses", "streaming":
		return false
	}
	return strings.ContainsAny(normalized, "_-./:") || containsReviewExternalDocDigit(normalized) || containsReviewExternalDocCamelBoundary(normalized)
}
