package externaldoc

import (
	"sort"
	"strings"
)

// SearchSubjectsForCorpus は repo/evidence corpus から external documentation 検索対象を抽出する。
func SearchSubjectsForCorpus(corpus string) []string {
	subjects := []struct {
		token   string
		subject string
	}{
		{token: "openai", subject: "OpenAI API"},
		{token: "responses", subject: "OpenAI Responses API"},
		{token: "anthropic", subject: "Anthropic API"},
		{token: "claude", subject: "Claude API"},
		{token: "gemini", subject: "Gemini API"},
		{token: "google", subject: "Google Gemini API"},
		{token: "kimi", subject: "Kimi API"},
		{token: "moonshot", subject: "Moonshot Kimi API"},
		{token: "bedrock", subject: "Amazon Bedrock API"},
		{token: "aws", subject: "AWS API"},
		{token: "azure", subject: "Azure OpenAI API"},
		{token: "groq", subject: "Groq API"},
		{token: "openrouter", subject: "OpenRouter API"},
		{token: "mcp", subject: "Model Context Protocol"},
		{token: "oauth", subject: "OAuth"},
		{token: "cloudflare", subject: "Cloudflare Workers"},
	}
	result := make([]string, 0)
	seen := make(map[string]struct{})
	for _, subject := range subjects {
		if !strings.Contains(corpus, subject.token) {
			continue
		}
		if _, exists := seen[subject.subject]; exists {
			continue
		}
		seen[subject.subject] = struct{}{}
		result = append(result, subject.subject)
	}
	sort.Strings(result)
	return result
}

// HasFetchedSnippet は fetched external_doc evidence に引用可能 snippet があるかを返す。
func HasFetchedSnippet(docs []Evidence) bool {
	for _, doc := range docs {
		if len(doc.Snippets) > 0 {
			return true
		}
	}
	return false
}
