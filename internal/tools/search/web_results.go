package search

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	webSearchURLRE               = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `\]\)}]+`)
	webSearchResultIndexPrefixRE = regexp.MustCompile(`^\d+[.)]\s*`)
)

// ParseWebSearchResults は provider のテキスト回答から URL 付き検索結果を抽出する。
func ParseWebSearchResults(raw string) []WebSearchResult {
	lines := strings.Split(raw, "\n")
	results := make([]WebSearchResult, 0)
	seen := make(map[string]struct{})
	for i, line := range lines {
		urls := webSearchURLRE.FindAllString(line, -1)
		for _, rawURL := range urls {
			cleanURL := cleanWebSearchResultURL(rawURL)
			if cleanURL == "" {
				continue
			}
			if _, exists := seen[cleanURL]; exists {
				continue
			}
			seen[cleanURL] = struct{}{}
			results = append(results, WebSearchResult{
				Title:        webSearchResultTitle(lines, i, cleanURL),
				URL:          cleanURL,
				Snippet:      webSearchResultSnippet(lines, i),
				SourceDomain: webSearchResultDomain(cleanURL),
			})
		}
	}
	return results
}

func cleanWebSearchResultURL(rawURL string) string {
	candidate := strings.TrimSpace(rawURL)
	candidate = strings.TrimRight(candidate, ".,;:!?")
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return ""
	}
	return parsed.String()
}

func webSearchResultTitle(lines []string, urlLine int, resultURL string) string {
	for i := urlLine; i >= 0 && i >= urlLine-2; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.Contains(line, "URL:") {
			continue
		}
		line = strings.TrimSpace(webSearchResultIndexPrefixRE.ReplaceAllString(line, ""))
		if line != "" && !strings.Contains(line, resultURL) {
			return line
		}
	}
	if domain := webSearchResultDomain(resultURL); domain != "" {
		return domain
	}
	return resultURL
}

func webSearchResultSnippet(lines []string, urlLine int) string {
	start := max(0, urlLine-1)
	end := min(len(lines), urlLine+2)
	parts := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts = append(parts, line)
	}
	snippet := strings.Join(parts, " ")
	if len(snippet) > 500 {
		snippet = snippet[:500]
	}
	return snippet
}

func webSearchResultDomain(resultURL string) string {
	parsed, err := url.Parse(resultURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}
