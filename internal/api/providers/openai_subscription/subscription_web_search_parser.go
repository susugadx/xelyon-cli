package openaisubscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
)

var subscriptionWebSearchURLPattern = regexp.MustCompile(`https?://[^\s<>"'\]\)}]+`)

type subscriptionWebSearchResult struct {
	Summary            string
	Sources            []subscriptionWebSearchSource
	WebSearchCallCount int
	ResponseID         string
	Usage              *api.Usage
}

type subscriptionWebSearchSource struct {
	Title string
	URL   string
}

type subscriptionWebSearchSSEEvent struct {
	Type        string                            `json:"type"`
	Delta       string                            `json:"delta,omitempty"`
	ItemID      string                            `json:"item_id,omitempty"`
	OutputIndex *int                              `json:"output_index,omitempty"`
	CallID      string                            `json:"call_id,omitempty"`
	Response    *openairesponses.ResponseMetadata `json:"response,omitempty"`
	Item        map[string]any                    `json:"item,omitempty"`
	Usage       *openairesponses.Usage            `json:"usage,omitempty"`
	Error       *openairesponses.Error            `json:"error,omitempty"`
}

type subscriptionWebSearchParser struct {
	summaryDelta                strings.Builder
	messageTexts                []string
	sources                     []subscriptionWebSearchSource
	sourceURLs                  map[string]struct{}
	webSearchCallKeys           map[string]struct{}
	anonymousWebSearchCallSeen  bool
	anonymousWebSearchCallCount int
	responseID                  string
	usage                       *api.Usage
}

func parseSubscriptionWebSearchStream(ctx context.Context, resp *http.Response) (subscriptionWebSearchResult, error) {
	parser := newSubscriptionWebSearchParser()
	_, err := api.ParseStreamingResponse(ctx, resp, nil, parser.parseLine)
	if err != nil {
		return subscriptionWebSearchResult{}, err
	}
	return parser.result(), nil
}

func newSubscriptionWebSearchParser() *subscriptionWebSearchParser {
	return &subscriptionWebSearchParser{
		sourceURLs:        make(map[string]struct{}),
		webSearchCallKeys: make(map[string]struct{}),
	}
}

func (p *subscriptionWebSearchParser) parseLine(line string) (string, bool, error) {
	data, done, handled := openaicompatstream.ParseSSEDataLine(line)
	if !handled {
		return "", false, nil
	}
	if done {
		return "", true, nil
	}

	var event subscriptionWebSearchSSEEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return "", false, fmt.Errorf("failed to decode subscription web search stream event: %w", err)
	}
	if err := p.captureEvent(event); err != nil {
		return "", false, err
	}
	return "", event.Type == "response.completed" || event.Type == "response.done", nil
}

func (p *subscriptionWebSearchParser) captureEvent(event subscriptionWebSearchSSEEvent) error {
	if event.Response != nil {
		if strings.TrimSpace(event.Response.ID) != "" {
			p.responseID = strings.TrimSpace(event.Response.ID)
		}
		p.captureUsage(event.Response.Usage)
	}
	p.captureUsage(event.Usage)

	switch {
	case event.Type == "error":
		return subscriptionWebSearchEventError(subscriptionDisplayName+" web search stream error", event.Error)
	case event.Type == "response.failed":
		return subscriptionWebSearchEventError(subscriptionDisplayName+" web search request failed", event.Error)
	case event.Type == "response.output_text.delta":
		p.summaryDelta.WriteString(event.Delta)
	case strings.HasPrefix(event.Type, "response.web_search_call."):
		p.observeWebSearchCall(event.webSearchCallKey())
	case event.Type == "response.output_item.added" || event.Type == "response.output_item.done":
		p.captureOutputItem(event)
	}
	return nil
}

func subscriptionWebSearchEventError(fallback string, eventError *openairesponses.Error) error {
	message := fallback
	if eventError != nil {
		switch {
		case strings.TrimSpace(eventError.Message) != "":
			message = eventError.Message
		case strings.TrimSpace(eventError.Code) != "":
			message = fallback + ": " + eventError.Code
		}
	}
	return fmt.Errorf("%s", RedactSubscriptionSecrets(message))
}

func (event subscriptionWebSearchSSEEvent) webSearchCallKey() string {
	for _, value := range []string{event.ItemID, event.CallID} {
		if key := strings.TrimSpace(value); key != "" {
			return key
		}
	}
	if event.OutputIndex != nil {
		return fmt.Sprintf("output:%d", *event.OutputIndex)
	}
	return ""
}

func (p *subscriptionWebSearchParser) captureOutputItem(event subscriptionWebSearchSSEEvent) {
	if len(event.Item) == 0 {
		return
	}
	itemType := strings.TrimSpace(webSearchStringFromMap(event.Item, "type"))
	switch itemType {
	case "web_search_call":
		p.observeWebSearchCall(firstNonEmpty(webSearchStringFromMap(event.Item, "id"), event.webSearchCallKey()))
		p.addSourcesFromValue(event.Item["action"])
	case "message":
		p.captureMessageContent(event.Item["content"])
	}
}

func (p *subscriptionWebSearchParser) observeWebSearchCall(key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		if !p.anonymousWebSearchCallSeen {
			p.anonymousWebSearchCallSeen = true
			p.anonymousWebSearchCallCount = 1
		}
		return
	}
	p.webSearchCallKeys[key] = struct{}{}
}

func (p *subscriptionWebSearchParser) captureMessageContent(value any) {
	switch typed := value.(type) {
	case string:
		p.addMessageText(typed)
	case []any:
		for _, item := range typed {
			p.captureMessageContent(item)
		}
	case map[string]any:
		p.addMessageText(webSearchStringFromMap(typed, "text"))
		p.addSourcesFromValue(typed["annotations"])
	}
}

func (p *subscriptionWebSearchParser) addMessageText(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	p.messageTexts = append(p.messageTexts, text)
}

func (p *subscriptionWebSearchParser) addSourcesFromValue(value any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			p.addSourcesFromValue(item)
		}
	case map[string]any:
		p.addSource(firstNonEmpty(webSearchStringFromMap(typed, "title"), webSearchStringFromMap(typed, "name")), firstNonEmpty(webSearchStringFromMap(typed, "url"), webSearchStringFromMap(typed, "uri")))
		for key, child := range typed {
			if key == "title" || key == "name" || key == "url" || key == "uri" {
				continue
			}
			p.addSourcesFromValue(child)
		}
	}
}

func (p *subscriptionWebSearchParser) addSource(title, rawURL string) {
	cleanURL := normalizeSubscriptionWebSearchSourceURL(rawURL)
	if cleanURL == "" {
		return
	}
	if _, ok := p.sourceURLs[cleanURL]; ok {
		return
	}
	p.sourceURLs[cleanURL] = struct{}{}
	p.sources = append(p.sources, subscriptionWebSearchSource{
		Title: strings.TrimSpace(title),
		URL:   cleanURL,
	})
}

func (p *subscriptionWebSearchParser) captureUsage(usage *openairesponses.Usage) {
	apiUsage := openairesponses.UsageToAPIUsage(usage)
	if apiUsage == nil {
		return
	}
	p.usage = apiUsage
}

func (p *subscriptionWebSearchParser) result() subscriptionWebSearchResult {
	summary := strings.TrimSpace(p.summaryDelta.String())
	if summary == "" {
		summary = strings.TrimSpace(strings.Join(p.messageTexts, "\n\n"))
	}
	if len(p.sources) == 0 && summary != "" {
		p.addSourceURLsFromText(summary)
	}
	return subscriptionWebSearchResult{
		Summary:            summary,
		Sources:            append([]subscriptionWebSearchSource(nil), p.sources...),
		WebSearchCallCount: len(p.webSearchCallKeys) + p.anonymousWebSearchCallCount,
		ResponseID:         p.responseID,
		Usage:              p.usage,
	}
}

func (p *subscriptionWebSearchParser) addSourceURLsFromText(text string) {
	for _, match := range subscriptionWebSearchURLPattern.FindAllString(text, -1) {
		p.addSource("", match)
	}
}

func validateSubscriptionWebSearchRuntimeResult(result subscriptionWebSearchResult) error {
	if result.WebSearchCallCount > 0 || len(result.Sources) > 0 {
		return nil
	}
	return fmt.Errorf("subscription web search response did not include a web_search_call or source URL")
}

func validateSubscriptionWebSearchSmokeResult(result subscriptionWebSearchResult) error {
	if result.WebSearchCallCount == 0 {
		return fmt.Errorf("subscription web search smoke response did not include a web_search_call")
	}
	if strings.TrimSpace(result.Summary) == "" && len(result.Sources) == 0 {
		return fmt.Errorf("subscription web search smoke response did not include summary or sources")
	}
	return nil
}

func formatSubscriptionWebSearchResult(result subscriptionWebSearchResult) string {
	summary := strings.TrimSpace(result.Summary)
	if summary == "" && len(result.Sources) == 0 {
		return "No results found."
	}

	var b strings.Builder
	if summary != "" {
		b.WriteString("Summary:\n")
		b.WriteString(summary)
	}
	if len(result.Sources) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("Sources:\n\n")
		for i, source := range result.Sources {
			title := strings.TrimSpace(source.Title)
			if title == "" {
				title = source.URL
			}
			fmt.Fprintf(&b, "%d. %s\n", i+1, title)
			fmt.Fprintf(&b, "   URL: %s\n", source.URL)
			if i < len(result.Sources)-1 {
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

func normalizeSubscriptionWebSearchSourceURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	rawURL = strings.TrimRight(rawURL, ".,;:")
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return ""
	}
	return RedactSubscriptionSecrets(parsed.String())
}

func webSearchStringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
