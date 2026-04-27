package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type responsesRequestRunOptions struct {
	URL                     string
	BuildRequest            func() ResponsesRequest
	HasPreviousResponseID   func() bool
	ClearPreviousResponseID func()
	DebugName               string
	Debug                   bool
	DebugWriter             io.Writer
}

type responsesNonStreamingResponse struct {
	ID         string                            `json:"id"`
	Status     string                            `json:"status"`
	Error      *ResponsesError                   `json:"error"`
	OutputText string                            `json:"output_text"`
	Output     []responsesNonStreamingOutputItem `json:"output"`
	Usage      *ResponsesUsage                   `json:"usage"`
}

type responsesNonStreamingOutputItem struct {
	Type      string                             `json:"type"`
	CallID    string                             `json:"call_id"`
	Name      string                             `json:"name"`
	Arguments string                             `json:"arguments"`
	Content   []responsesNonStreamingContentPart `json:"content"`
}

type responsesNonStreamingContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (p *Provider) runResponsesRequest(ctx context.Context, options responsesRequestRunOptions) (string, string, error) {
	if options.BuildRequest == nil {
		return "", "", fmt.Errorf("responses request builder is nil")
	}

	for attempt := 0; attempt < 2; attempt++ {
		reqBody := options.BuildRequest()
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal request: %w", err)
		}
		writeResponsesDebugRequest(options, payload)

		req, err := openaicompat.NewBearerJSONBytesRequest(ctx, options.URL, p.APIKey, payload)
		if err != nil {
			return "", "", err
		}

		spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)
		resp, err := p.executeResponsesRequest(req, reqBody.Stream)
		if err != nil {
			spinner.Stop()
			return "", "", fmt.Errorf("API request failed: %w", err)
		}

		if shouldRetryResponsesWithoutPreviousResponseID(resp, options, attempt) {
			spinner.Stop()
			resp.Body.Close()
			options.ClearPreviousResponseID()
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", "", api.HandleHTTPError(resp, spinner, p.Name())
		}

		if reqBody.Stream {
			return p.handleResponsesStreaming(ctx, resp, spinner)
		}
		return p.handleResponsesNonStreaming(ctx, resp, spinner)
	}

	return "", "", fmt.Errorf("responses API retry exhausted")
}

func (p *Provider) executeResponsesRequest(req *http.Request, stream bool) (*http.Response, error) {
	if stream {
		return p.ExecuteRequest(req)
	}
	return p.executeLongRunningResponsesRequest(req)
}

func (p *Provider) executeLongRunningResponsesRequest(req *http.Request) (*http.Response, error) {
	return api.DoWithRetry(req.Context(), newLongRunningResponsesHTTPClient(p.HTTPClient), req)
}

func newLongRunningResponsesHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	client := *base

	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if httpTransport, ok := transport.(*http.Transport); ok {
		cloned := httpTransport.Clone()
		cloned.ResponseHeaderTimeout = 0
		client.Transport = cloned
		return &client
	}

	client.Transport = transport
	return &client
}

func writeResponsesDebugRequest(options responsesRequestRunOptions, payload []byte) {
	if !options.Debug || options.DebugWriter == nil {
		return
	}
	debugName := options.DebugName
	if debugName == "" {
		debugName = "Responses"
	}
	fmt.Fprintf(options.DebugWriter, "[DEBUG %s Responses] Request body:\n%s\n", debugName, string(payload))
}

func shouldRetryResponsesWithoutPreviousResponseID(resp *http.Response, options responsesRequestRunOptions, attempt int) bool {
	if resp.StatusCode != http.StatusBadRequest || attempt > 0 {
		return false
	}
	if options.HasPreviousResponseID == nil || options.ClearPreviousResponseID == nil {
		return false
	}
	return options.HasPreviousResponseID()
}

func (p *Provider) handleResponsesNonStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	var result responsesNonStreamingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		api.StopSpinner(spinner)
		return "", "", err
	}
	api.StopSpinner(spinner)

	if err := result.validateSuccess(); err != nil {
		return "", "", err
	}

	p.emitResponsesNonStreamingUsage(result.Usage)
	text := result.extractText()
	content := composeResponsesNonStreamingContent(text, result.extractToolJSON())
	printResponsesNonStreamingText(ctx, text)
	return content, result.ID, nil
}

func (r responsesNonStreamingResponse) validateSuccess() error {
	if r.Error != nil {
		return formatResponsesNonStreamingError(r.Status, r.Error)
	}
	if r.Status == "" || r.Status == "completed" {
		return nil
	}
	if r.Status == "failed" {
		return errors.New("OpenAI Responses API request failed")
	}
	return fmt.Errorf("OpenAI Responses API response status: %s", r.Status)
}

func formatResponsesNonStreamingError(status string, responsesError *ResponsesError) error {
	if responsesError.Message != "" {
		return errors.New(responsesError.Message)
	}
	if responsesError.Code != "" {
		return fmt.Errorf("OpenAI API error: %s", responsesError.Code)
	}
	if responsesError.Type != "" {
		return fmt.Errorf("OpenAI API error: %s", responsesError.Type)
	}
	if status == "failed" {
		return errors.New("OpenAI Responses API request failed")
	}
	return errors.New("OpenAI API error")
}

func (p *Provider) emitResponsesNonStreamingUsage(usage *ResponsesUsage) {
	if p.usageCallback == nil {
		return
	}
	apiUsage := responsesUsageToAPIUsage(usage)
	if apiUsage == nil {
		return
	}
	p.usageCallback(*apiUsage)
}

func (r responsesNonStreamingResponse) extractText() string {
	if r.OutputText != "" {
		return r.OutputText
	}

	var text strings.Builder
	for _, item := range r.Output {
		if item.Type != "message" {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "output_text" {
				text.WriteString(part.Text)
			}
		}
	}
	return text.String()
}

func (r responsesNonStreamingResponse) extractToolJSON() string {
	var toolCalls strings.Builder
	for _, item := range r.Output {
		if item.Type != "function_call" {
			continue
		}
		toolJSON, err := convertResponsesFunctionCallToToolJSON(item.CallID, item.Name, item.Arguments)
		if err != nil {
			continue
		}
		toolCalls.WriteString(toolJSON)
	}
	return toolCalls.String()
}

func composeResponsesNonStreamingContent(text, toolJSON string) string {
	if toolJSON == "" {
		return text
	}
	if text != "" {
		return text + toolJSON
	}
	return toolJSON
}

func printResponsesNonStreamingText(ctx context.Context, text string) {
	if text == "" || !api.ShouldStreamAssistantText(ctx) {
		return
	}
	api.PrintAIHeaderWithContext(ctx)
	_, _ = fmt.Fprintln(api.OutputWriterFromContext(ctx), text)
}
