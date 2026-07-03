package claude

import (
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

const (
	webSearchBetaHeader          = "web-search-2025-03-05"
	webSearchVisibleMaxTokensCap = 2048
)

type webSearchRequest struct {
	Model        string             `json:"model"`
	Messages     []AnthropicMessage `json:"messages"`
	MaxTokens    int                `json:"max_tokens"`
	Stream       bool               `json:"stream"`
	Thinking     *ThinkingConfig    `json:"thinking,omitempty"`
	OutputConfig *OutputConfig      `json:"output_config,omitempty"`
	Tools        []webSearchTool    `json:"tools"`
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

type claudeWebSearchRequestBuild struct {
	Model   string
	Request webSearchRequest
}

func init() {
	websearch.RegisterWithContext("claude", func(ctx context.Context, query, model string) (string, error) {
		return webSearchWithContextForProvider(ctx, "claude", query, model)
	})
	websearch.RegisterWithContext("anthropic", func(ctx context.Context, query, model string) (string, error) {
		return webSearchWithContextForProvider(ctx, "anthropic", query, model)
	})
}

// WebSearchWithContext は request context を使って Claude ネイティブ web_search を実行する。
func WebSearchWithContext(ctx context.Context, query, model string) (string, error) {
	return webSearchWithContextForProvider(ctx, "claude", query, model)
}

func webSearchWithContextForProvider(ctx context.Context, providerKey, query, model string) (string, error) {
	apiKey := os.Getenv(anthropicAPIKeyEnv)
	if apiKey == "" {
		return "", fmt.Errorf("%s not set", anthropicAPIKeyEnv)
	}

	model = api.ResolveProviderRequestModel(ctx, model, providerKey)
	provider := newProvider(apiKey, providerKey)
	return provider.webSearch(ctx, query, model)
}

func (p *Provider) webSearch(ctx context.Context, query, model string) (string, error) {
	built := p.buildWebSearchRequest(ctx, query, model)
	req, err := p.newAnthropicRequest(ctx, built.Request, built.Model, nil, webSearchBetaHeader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

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

func (p *Provider) buildWebSearchRequest(ctx context.Context, query, model string) claudeWebSearchRequestBuild {
	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	catalogModel := cfg.ModelCatalogName(p.configLookupKey(), model)
	thinking, outputConfig := buildClaudeThinkingRequestPolicy(ctx, cfg, catalogModel)
	maxTokens := p.webSearchMaxTokens(ctx, model, thinking)

	return claudeWebSearchRequestBuild{
		Model: model,
		Request: webSearchRequest{
			Model: model,
			Messages: []AnthropicMessage{{
				Role: "user",
				Content: []AnthropicContentBlock{{
					Type: "text",
					Text: buildWebSearchPrompt(query),
				}},
			}},
			MaxTokens:    maxTokens,
			Stream:       false,
			Thinking:     thinking,
			OutputConfig: outputConfig,
			Tools: []webSearchTool{{
				Type:    "web_search_20250305",
				Name:    "web_search",
				MaxUses: 3,
			}},
		},
	}
}

func (p *Provider) webSearchMaxTokens(ctx context.Context, model string, thinking *ThinkingConfig) int {
	visibleMaxTokens := webSearchVisibleMaxTokensCap
	providerMaxTokens := p.maxOutputTokens(ctx, model)
	if thinking != nil && thinking.BudgetTokens > 0 {
		if providerMaxTokens > thinking.BudgetTokens {
			if providerVisibleMaxTokens := providerMaxTokens - thinking.BudgetTokens; providerVisibleMaxTokens < visibleMaxTokens {
				visibleMaxTokens = providerVisibleMaxTokens
			}
		}
		return thinking.BudgetTokens + visibleMaxTokens
	}
	if providerMaxTokens > 0 && providerMaxTokens < visibleMaxTokens {
		return providerMaxTokens
	}
	return visibleMaxTokens
}

func buildWebSearchPrompt(query string) string {
	return fmt.Sprintf("Use web search to answer the query below. Return a concise summary of the most relevant findings.\n\nQuery: %s", query)
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
