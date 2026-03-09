package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const webSearchBetaHeader = "web-search-2025-03-05"

type webSearchRequest struct {
	Model     string             `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
	Tools     []webSearchTool    `json:"tools"`
}

type webSearchTool struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	MaxUses int    `json:"max_uses,omitempty"`
}

type webSearchResponse struct {
	Content []webSearchContent `json:"content"`
}

type webSearchContent struct {
	Type      string              `json:"type"`
	Text      string              `json:"text,omitempty"`
	Citations []webSearchCitation `json:"citations,omitempty"`
}

type webSearchCitation struct {
	Type      string `json:"type,omitempty"`
	CitedText string `json:"cited_text,omitempty"`
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
}

type webSearchSource struct {
	Title string
	URL   string
}

func init() {
	websearch.RegisterWithContext("claude", WebSearchWithContext)
}

// WebSearch executes Anthropic's native web search tool.
func WebSearch(query, model string) (string, error) {
	return WebSearchWithContext(config.WithGlobalFallback(context.Background()), query, model)
}

// WebSearchWithContext は request context を使って Claude ネイティブ web_search を実行する。
func WebSearchWithContext(ctx context.Context, query, model string) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	model = api.GetDefaultModelWithContext(ctx, model, "claude", "claude-sonnet-4-6")
	provider := New(apiKey)
	return provider.webSearch(ctx, query, model)
}

func (p *Provider) webSearch(ctx context.Context, query, model string) (string, error) {
	maxTokens := 2048
	if providerMax := api.GetMaxOutputTokens(ctx, "claude", model); providerMax > 0 && providerMax < maxTokens {
		maxTokens = providerMax
	}

	reqBody := webSearchRequest{
		Model: model,
		Messages: []AnthropicMessage{{
			Role: "user",
			Content: []AnthropicContentBlock{{
				Type: "text",
				Text: buildWebSearchPrompt(query),
			}},
		}},
		MaxTokens: maxTokens,
		Stream:    false,
		Tools: []webSearchTool{{
			Type:    "web_search_20250305",
			Name:    "web_search",
			MaxUses: 3,
		}},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", getAnthropicVersion(ctx))
	req.Header.Set("anthropic-beta", strings.Join(getAnthropicBetaHeaders(ctx), ","))

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, nil, "Claude")
	}

	var parsed webSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	summary, sources := parseWebSearchResponse(parsed)
	return formatWebSearchResult(summary, sources), nil
}

func buildWebSearchPrompt(query string) string {
	return fmt.Sprintf("Use web search to answer the query below. Return a concise summary of the most relevant findings.\n\nQuery: %s", query)
}

func getAnthropicVersion(ctx context.Context) string {
	cfg := config.FromContext(ctx)
	if pm, ok := cfg.ProviderModels["claude"]; ok && pm.AnthropicVersion != "" {
		return pm.AnthropicVersion
	}
	return "2023-06-01"
}

func getAnthropicBetaHeaders(ctx context.Context) []string {
	cfg := config.FromContext(ctx)
	seen := map[string]bool{webSearchBetaHeader: true}
	headers := []string{webSearchBetaHeader}
	if pm, ok := cfg.ProviderModels["claude"]; ok {
		for _, header := range pm.AnthropicBeta {
			header = strings.TrimSpace(header)
			if header == "" || seen[header] {
				continue
			}
			seen[header] = true
			headers = append(headers, header)
		}
	}
	return headers
}

func parseWebSearchResponse(resp webSearchResponse) (string, []webSearchSource) {
	var summaryParts []string
	var sources []webSearchSource

	for _, block := range resp.Content {
		if text := strings.TrimSpace(block.Text); text != "" {
			summaryParts = append(summaryParts, text)
		}
		for _, citation := range block.Citations {
			if citation.URL == "" {
				continue
			}
			sources = append(sources, webSearchSource{
				Title: citation.Title,
				URL:   citation.URL,
			})
		}
	}

	return strings.Join(summaryParts, "\n\n"), dedupeWebSearchSources(sources)
}

func formatWebSearchResult(summary string, sources []webSearchSource) string {
	summary = strings.TrimSpace(summary)
	if summary == "" && len(sources) == 0 {
		return "No results found."
	}

	var b strings.Builder
	if summary != "" {
		b.WriteString("Summary:\n")
		b.WriteString(summary)
	}

	if len(sources) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("Sources:\n\n")
		for i, source := range sources {
			title := strings.TrimSpace(source.Title)
			if title == "" {
				title = source.URL
			}
			fmt.Fprintf(&b, "%d. %s\n", i+1, title)
			fmt.Fprintf(&b, "   URL: %s\n", source.URL)
			if i < len(sources)-1 {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func dedupeWebSearchSources(sources []webSearchSource) []webSearchSource {
	seen := make(map[string]bool, len(sources))
	result := make([]webSearchSource, 0, len(sources))
	for _, source := range sources {
		url := strings.TrimSpace(source.URL)
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		result = append(result, source)
	}
	return result
}
