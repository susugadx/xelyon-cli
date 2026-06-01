package externaldoc

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

var reviewExternalDocCredibilityTokenRE = regexp.MustCompile(`[a-z0-9]+`)

func classifySourceCredibility(fetchReq FetchRequest, doc Evidence, content string) (SourceCredibility, string) {
	source := reviewExternalDocNormalizeCredibilityDomain(doc.SourceDomain)
	title := strings.ToLower(strings.Join(strings.Fields(fetchReq.SearchResultTitle), " "))
	body := strings.ToLower(strings.Join(strings.Fields(content), " "))
	subjectTokens := reviewExternalDocSubjectCredibilityTokens(fetchReq.QuerySubjectHint)
	combined := strings.TrimSpace(source + " " + title + " " + body)

	if reviewExternalDocHasThirdPartySignal(source, title, fetchReq.URL) {
		return SourceCredibilityThirdParty, "third_party: known third-party host, source URL, or source metadata signal is present"
	}
	if combined == "" {
		return SourceCredibilityUnknown, "unknown: source metadata and content are unavailable"
	}

	trustedDomains := reviewExternalDocTrustedDomainsForSubject(fetchReq.QuerySubjectHint)
	if len(trustedDomains) == 0 {
		return SourceCredibilityUnknown, "unknown: query subject has no trusted domain mapping for official documentation"
	}
	if !reviewExternalDocSourceMatchesTrustedDomains(source, trustedDomains) {
		return SourceCredibilityUnknown, "unknown: source domain does not match trusted domains for the query subject"
	}
	if !reviewExternalDocSubjectMatchesText(title, body, subjectTokens) {
		return SourceCredibilityUnknown, "unknown: trusted domain matched but title or content lacks a query subject text signal"
	}
	if !reviewExternalDocContentHasReferenceSignal(body) {
		return SourceCredibilityUnknown, "unknown: trusted domain and subject text matched but reference content signal is absent"
	}
	return SourceCredibilityOfficialCandidate, "official_candidate: trusted source domain, query subject text signal, and reference content signal are present"
}

func normalizeSourceCredibility(value SourceCredibility) SourceCredibility {
	switch value {
	case SourceCredibilityOfficialCandidate, SourceCredibilityThirdParty, SourceCredibilityUnknown:
		return value
	default:
		return SourceCredibilityUnknown
	}
}

func normalizeSourceCredibilityReason(value SourceCredibility, reason string) string {
	reason = strings.Join(strings.Fields(strings.TrimSpace(reason)), " ")
	if reason != "" {
		return reason
	}
	switch normalizeSourceCredibility(value) {
	case SourceCredibilityOfficialCandidate:
		return "official_candidate: source signals matched official documentation heuristics"
	case SourceCredibilityThirdParty:
		return "third_party: source signals matched unofficial or community heuristics"
	default:
		return "unknown: source credibility was not established"
	}
}

func reviewExternalDocSubjectCredibilityTokens(subject string) []string {
	lower := strings.ToLower(subject)
	tokens := reviewExternalDocCredibilityTokenRE.FindAllString(lower, -1)
	tokens = append(tokens, reviewExternalDocSubjectCredibilityAliases(lower)...)

	result := make([]string, 0, len(tokens))
	seen := make(map[string]struct{})
	for _, token := range tokens {
		if reviewExternalDocSubjectTokenIsGeneric(token) {
			continue
		}
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}

func reviewExternalDocSubjectCredibilityAliases(subject string) []string {
	var aliases []string
	if strings.Contains(subject, "gemini") {
		aliases = append(aliases, "google")
	}
	if strings.Contains(subject, "claude") {
		aliases = append(aliases, "anthropic")
	}
	if strings.Contains(subject, "bedrock") {
		aliases = append(aliases, "amazon", "aws")
	}
	if strings.Contains(subject, "aws") {
		aliases = append(aliases, "amazon")
	}
	if strings.Contains(subject, "kimi") {
		aliases = append(aliases, "moonshot")
	}
	if strings.Contains(subject, "model context protocol") {
		aliases = append(aliases, "mcp", "modelcontextprotocol")
	}
	return aliases
}

func reviewExternalDocTrustedDomainsForSubject(subject string) []string {
	lower := strings.ToLower(strings.TrimSpace(subject))
	if lower == "" {
		return nil
	}
	rules := []struct {
		signals []string
		domains []string
	}{
		{signals: []string{"azure", "microsoft"}, domains: []string{"microsoft.com"}},
		{signals: []string{"bedrock", "aws", "amazon"}, domains: []string{"aws.amazon.com"}},
		{signals: []string{"openrouter"}, domains: []string{"openrouter.ai"}},
		{signals: []string{"openai", "responses"}, domains: []string{"openai.com"}},
		{signals: []string{"anthropic", "claude"}, domains: []string{"anthropic.com"}},
		{signals: []string{"gemini", "google"}, domains: []string{"google.com", "google.dev"}},
		{signals: []string{"kimi", "moonshot"}, domains: []string{"moonshot.ai"}},
		{signals: []string{"groq"}, domains: []string{"groq.com"}},
		{signals: []string{"cloudflare"}, domains: []string{"cloudflare.com"}},
		{signals: []string{"model context protocol", "mcp", "modelcontextprotocol"}, domains: []string{"modelcontextprotocol.io"}},
	}
	for _, rule := range rules {
		for _, signal := range rule.signals {
			if reviewExternalDocSubjectContainsSignal(lower, signal) {
				return append([]string(nil), rule.domains...)
			}
		}
	}
	return nil
}

func reviewExternalDocSubjectContainsSignal(subject, signal string) bool {
	if strings.Contains(signal, " ") {
		return strings.Contains(subject, signal)
	}
	for _, token := range reviewExternalDocCredibilityTokenRE.FindAllString(subject, -1) {
		if token == signal {
			return true
		}
	}
	return false
}

func reviewExternalDocSubjectTokenIsGeneric(token string) bool {
	switch token {
	case "", "api", "apis", "doc", "docs", "documentation", "official", "reference", "developer", "developers", "platform":
		return true
	default:
		return len(token) < 3
	}
}

func reviewExternalDocSubjectMatchesText(title, body string, tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	textTokens := reviewExternalDocCredibilityTokenRE.FindAllString(title+" "+body, -1)
	textTokenSet := make(map[string]struct{}, len(textTokens))
	for _, token := range textTokens {
		textTokenSet[token] = struct{}{}
	}
	for _, token := range tokens {
		if _, ok := textTokenSet[token]; ok {
			return true
		}
	}
	return false
}

func reviewExternalDocContentHasReferenceSignal(body string) bool {
	for _, signal := range []string{
		"api reference",
		"authentication",
		"authorization",
		"documentation",
		"endpoint",
		"http",
		"parameter",
		"request",
		"response",
		"sdk",
	} {
		if strings.Contains(body, signal) {
			return true
		}
	}
	return false
}

func reviewExternalDocHasThirdPartySignal(sourceDomain, title, sourceURL string) bool {
	for _, thirdPartyDomain := range []string{
		"blogspot.com",
		"dev.to",
		"hashnode.com",
		"hashnode.dev",
		"medium.com",
		"reddit.com",
		"stackexchange.com",
		"stackoverflow.com",
		"substack.com",
	} {
		if reviewExternalDocSourceMatchesTrustedDomain(sourceDomain, thirdPartyDomain) {
			return true
		}
	}
	if reviewExternalDocHasGitHubThirdPartySignal(sourceDomain, title, sourceURL) {
		return true
	}
	metadata := strings.TrimSpace(sourceDomain + " " + title)
	for _, signal := range []string{
		"community guide",
		"community tutorial",
		"community blog",
		"community article",
		"not official",
		"personal blog",
		"unofficial",
	} {
		if strings.Contains(metadata, signal) {
			return true
		}
	}
	return reviewExternalDocMetadataHasThirdPartySourceType(metadata)
}

func reviewExternalDocHasGitHubThirdPartySignal(sourceDomain, title, sourceURL string) bool {
	if !reviewExternalDocSourceMatchesTrustedDomain(sourceDomain, "github.com") {
		return false
	}
	path := reviewExternalDocCredibilityURLPath(sourceURL)
	for _, signal := range []string{"/issues/", "/pull/", "/discussions/"} {
		if strings.Contains(path, signal) {
			return true
		}
	}
	metadata := " " + title + " "
	for _, signal := range []string{
		" github issue ",
		" github discussion ",
		" github pull request ",
		" issue #",
		" discussion #",
		" pull request #",
	} {
		if strings.Contains(metadata, signal) {
			return true
		}
	}
	return false
}

func reviewExternalDocCredibilityURLPath(sourceURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(sourceURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Path)
}

func reviewExternalDocMetadataHasThirdPartySourceType(metadata string) bool {
	for _, sourceType := range []string{"guide", "tutorial", "blog", "article", "post"} {
		if strings.Contains(metadata, "third party "+sourceType) || strings.Contains(metadata, "third-party "+sourceType) {
			return true
		}
	}
	return false
}

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
