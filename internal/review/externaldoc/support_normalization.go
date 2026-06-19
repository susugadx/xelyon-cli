package externaldoc

import (
	"net"
	"net/url"
	"strings"
)

func externalSupportUniqueKeys(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func externalSupportNormalizedHash(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func externalSupportNormalizedURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.ToLower(rawURL)
	}

	normalized := *parsed
	normalized.Scheme = strings.ToLower(normalized.Scheme)
	normalized.Host = externalSupportNormalizedURLHost(normalized.Scheme, normalized.Host)
	normalized.User = nil
	normalized.Fragment = ""
	normalized.RawPath = ""
	if normalized.Path == "" {
		normalized.Path = "/"
	}
	if normalized.RawQuery != "" {
		normalized.RawQuery = normalized.Query().Encode()
	}
	return normalized.String()
}

func externalSupportNormalizedURLHost(scheme, host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if parsedHost, port, err := net.SplitHostPort(host); err == nil {
		parsedHost = strings.TrimSuffix(parsedHost, ".")
		if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
			return parsedHost
		}
		return net.JoinHostPort(parsedHost, port)
	}
	return strings.TrimSuffix(host, ".")
}

func externalSupportSnippetCitationCapable(doc Evidence, snippet SnippetEvidence) bool {
	return strings.TrimSpace(doc.Error) == "" &&
		strings.TrimSpace(doc.DocID) != "" &&
		strings.TrimSpace(doc.URL) != "" &&
		!doc.FetchedAt.IsZero() &&
		strings.TrimSpace(snippet.SnippetID) != "" &&
		strings.TrimSpace(snippet.Content) != "" &&
		strings.TrimSpace(snippet.ContentHash) != ""
}

func appendExternalSupportUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
