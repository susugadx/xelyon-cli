package externaldoc

import (
	"net"
	"strings"
)

func reviewExternalDocSourceMatchesTrustedDomains(sourceDomain string, trustedDomains []string) bool {
	for _, trustedDomain := range trustedDomains {
		if reviewExternalDocSourceMatchesTrustedDomain(sourceDomain, trustedDomain) {
			return true
		}
	}
	return false
}

func reviewExternalDocSourceMatchesTrustedDomain(sourceDomain, trustedDomain string) bool {
	sourceDomain = reviewExternalDocNormalizeCredibilityDomain(sourceDomain)
	trustedDomain = reviewExternalDocNormalizeCredibilityDomain(trustedDomain)
	if sourceDomain == "" || trustedDomain == "" {
		return false
	}
	return sourceDomain == trustedDomain || strings.HasSuffix(sourceDomain, "."+trustedDomain)
}

func reviewExternalDocNormalizeCredibilityDomain(sourceDomain string) string {
	sourceDomain = strings.ToLower(strings.TrimSpace(sourceDomain))
	if strings.Contains(sourceDomain, "://") {
		sourceDomain = reviewExternalDocSourceDomain(sourceDomain)
	}
	if host, _, err := net.SplitHostPort(sourceDomain); err == nil {
		sourceDomain = host
	}
	sourceDomain = strings.Trim(sourceDomain, "[]")
	return strings.TrimSuffix(sourceDomain, ".")
}
