package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type webSearchRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	Tools            []webSearchTool         `json:"tools"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type webSearchTool struct {
	GoogleSearch          *googleSearchTool          `json:"google_search,omitempty"`
	GoogleSearchRetrieval *googleSearchRetrievalTool `json:"google_search_retrieval,omitempty"`
}

type googleSearchTool struct{}

type googleSearchRetrievalTool struct {
	DynamicRetrievalConfig *dynamicRetrievalConfig `json:"dynamic_retrieval_config,omitempty"`
}

type dynamicRetrievalConfig struct {
	Mode             string  `json:"mode,omitempty"`
	DynamicThreshold float32 `json:"dynamic_threshold,omitempty"`
}

type webSearchResponse struct {
	Candidates []webSearchCandidate `json:"candidates"`
}

type webSearchCandidate struct {
	Content           GeminiContent      `json:"content"`
	GroundingMetadata *groundingMetadata `json:"groundingMetadata,omitempty"`
}

type groundingMetadata struct {
	GroundingChunks []groundingChunk `json:"groundingChunks,omitempty"`
}

type groundingChunk struct {
	Web *groundingWeb `json:"web,omitempty"`
}

type groundingWeb struct {
	URI   string `json:"uri,omitempty"`
	Title string `json:"title,omitempty"`
}

type webSearchSource struct {
	Title string
	URL   string
}

func init() {
	websearch.Register("gemini", WebSearch)
}

// WebSearch executes Gemini's native Google Search grounding.
func WebSearch(query, model string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not set")
	}

	model = api.GetDefaultModel(model, "gemini", "gemini-3.1-pro-preview-customtools")
	provider := New(apiKey)
	return provider.webSearch(context.Background(), query, model)
}

func (p *Provider) webSearch(ctx context.Context, query, model string) (string, error) {
	cfg := config.GetGlobalConfig()
	reqBody := webSearchRequest{
		Contents: []GeminiContent{{
			Role:  "user",
			Parts: []GeminiPart{{Text: buildWebSearchPrompt(query)}},
		}},
		Tools:            []webSearchTool{buildWebSearchTool(model)},
		GenerationConfig: getThinkingConfigForModel(ctx, model, cfg),
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := getGeminiFunctionCallingURL(model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.doRequestWithRetry(ctx, req, jsonBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("gemini API error (status %d): unable to read response body - %v", resp.StatusCode, err)
		}
		if len(body) == 0 {
			return "", fmt.Errorf("gemini API error (status %d): empty response body", resp.StatusCode)
		}
		return "", api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	var parsed webSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	summary, sources := parseWebSearchResponse(parsed)
	return formatWebSearchResult(summary, sources), nil
}

func buildWebSearchPrompt(query string) string {
	return fmt.Sprintf("Search the web for the query below and return a concise summary of the most relevant findings.\n\nQuery: %s", query)
}

func buildWebSearchTool(model string) webSearchTool {
	if strings.HasPrefix(strings.ToLower(model), "gemini-1.5") {
		return webSearchTool{
			GoogleSearchRetrieval: &googleSearchRetrievalTool{
				DynamicRetrievalConfig: &dynamicRetrievalConfig{
					Mode:             "MODE_DYNAMIC",
					DynamicThreshold: 0.0,
				},
			},
		}
	}
	return webSearchTool{GoogleSearch: &googleSearchTool{}}
}

func parseWebSearchResponse(resp webSearchResponse) (string, []webSearchSource) {
	if len(resp.Candidates) == 0 {
		return "", nil
	}

	candidate := resp.Candidates[0]
	var summaryParts []string
	for _, part := range candidate.Content.Parts {
		if text := strings.TrimSpace(part.Text); text != "" {
			summaryParts = append(summaryParts, text)
		}
	}

	var sources []webSearchSource
	if candidate.GroundingMetadata != nil {
		for _, chunk := range candidate.GroundingMetadata.GroundingChunks {
			if chunk.Web == nil || chunk.Web.URI == "" {
				continue
			}
			sources = append(sources, webSearchSource{
				Title: chunk.Web.Title,
				URL:   chunk.Web.URI,
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
