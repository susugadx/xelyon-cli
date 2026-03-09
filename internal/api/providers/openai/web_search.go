package openai

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

type webSearchRequest struct {
	Model     string           `json:"model"`
	Input     string           `json:"input"`
	Tools     []map[string]any `json:"tools"`
	Include   []string         `json:"include,omitempty"`
	Store     bool             `json:"store"`
	Reasoning *ReasoningConfig `json:"reasoning,omitempty"`
}

type webSearchResponse struct {
	Output []webSearchOutputItem `json:"output"`
}

type webSearchOutputItem struct {
	Type    string                 `json:"type,omitempty"`
	Content []webSearchContentPart `json:"content,omitempty"`
	Action  *webSearchAction       `json:"action,omitempty"`
}

type webSearchContentPart struct {
	Type        string                `json:"type,omitempty"`
	Text        string                `json:"text,omitempty"`
	Annotations []webSearchAnnotation `json:"annotations,omitempty"`
}

type webSearchAnnotation struct {
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

type webSearchAction struct {
	Sources []webSearchSource `json:"sources,omitempty"`
}

type webSearchSource struct {
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

func init() {
	websearch.RegisterWithContext("openai", WebSearchWithContext)
}

// WebSearch executes OpenAI's native web_search tool via the Responses API.
func WebSearch(query, model string) (string, error) {
	return WebSearchWithContext(config.WithGlobalFallback(context.Background()), query, model)
}

// WebSearchWithContext は request context を使って OpenAI ネイティブ web_search を実行する。
func WebSearchWithContext(ctx context.Context, query, model string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	model = api.GetDefaultModelWithContext(ctx, model, "openai", "gpt-4o")
	cfg := config.FromContext(ctx)
	if !cfg.IsResponsesAPIModel(model) {
		return "", fmt.Errorf("model %q does not support Responses API web search", model)
	}

	provider := New(apiKey)
	return provider.webSearch(ctx, query, model)
}

func (p *Provider) webSearch(ctx context.Context, query, model string) (string, error) {
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	reqBody := webSearchRequest{
		Model:   model,
		Input:   buildWebSearchPrompt(query),
		Tools:   []map[string]any{{"type": "web_search"}},
		Include: []string{"web_search_call.action.sources"},
		Store:   false,
	}

	cfg := config.FromContext(ctx)
	if api.IsThinkingEnabled(ctx) {
		reqBody.Reasoning = &ReasoningConfig{Effort: LevelToReasoningEffort(cfg.Thinking.Level)}
	} else if isCodexModel(model) {
		reqBody.Reasoning = &ReasoningConfig{Effort: "low"}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	p.SetBearerAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, nil, "OpenAI Responses API")
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

func parseWebSearchResponse(resp webSearchResponse) (string, []webSearchSource) {
	var summaryParts []string
	var sources []webSearchSource

	for _, item := range resp.Output {
		if item.Action != nil {
			sources = append(sources, item.Action.Sources...)
		}
		for _, part := range item.Content {
			if text := strings.TrimSpace(part.Text); text != "" {
				summaryParts = append(summaryParts, text)
			}
			for _, annotation := range part.Annotations {
				if annotation.URL == "" {
					continue
				}
				sources = append(sources, webSearchSource(annotation))
			}
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
