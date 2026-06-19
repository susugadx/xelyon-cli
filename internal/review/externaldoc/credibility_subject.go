package externaldoc

import "strings"

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
