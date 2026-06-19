package externaldoc

import (
	"net/url"
	"strings"
)

func reviewExternalDocSourceDomain(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}
