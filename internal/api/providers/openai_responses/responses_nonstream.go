package openairesponses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunOptions は Responses API request 実行時の provider 差分を表す。
type RunOptions struct {
	URL                      string
	BuildRequest             func() Request
	PrepareRequest           func(ctx context.Context, url string, payload []byte) (*http.Request, error)
	ExecuteRequest           func(req *http.Request, stream bool) (*http.Response, error)
	HandleStreaming          func(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error)
	HandleNonStreaming       func(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error)
	HandleHTTPError          func(resp *http.Response, spinner *ui.Spinner, providerName string) error
	RequestObserver          func(Request)
	SetLocalAutoCompressSkip func(bool)
	HasPreviousResponseID    func() bool
	ClearPreviousResponseID  func()
	ProviderName             string
	DebugName                string
	Debug                    bool
	DebugWriter              io.Writer
	DebugPayloadRedactor     func([]byte) []byte
}

// NonStreamingOptions は Responses API の非ストリーミング解析時の provider 差分を表す。
type NonStreamingOptions struct {
	ProviderName        string
	UsageCallback       api.UsageCallback
	ReplayItemsCallback func([]api.InputItem)
}

type responsesNonStreamingResponse struct {
	ID         string                            `json:"id"`
	Status     string                            `json:"status"`
	Error      *Error                            `json:"error"`
	OutputText string                            `json:"output_text"`
	Output     []responsesNonStreamingOutputItem `json:"output"`
	Usage      *Usage                            `json:"usage"`
}

type responsesNonStreamingOutputItem struct {
	Type             string                             `json:"type"`
	ID               string                             `json:"id"`
	Status           string                             `json:"status"`
	CallID           string                             `json:"call_id"`
	Name             string                             `json:"name"`
	Arguments        string                             `json:"arguments"`
	Output           string                             `json:"output"`
	Content          []responsesNonStreamingContentPart `json:"content"`
	Summary          []map[string]any                   `json:"summary"`
	EncryptedContent string                             `json:"encrypted_content"`
}

type responsesNonStreamingContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// RunResponsesRequest は Responses API request を送信し、invalid previous_response_id を一度だけ回復する。
func RunResponsesRequest(ctx context.Context, options RunOptions) (string, string, error) {
	if options.BuildRequest == nil {
		return "", "", fmt.Errorf("responses request builder is nil")
	}
	if options.PrepareRequest == nil {
		return "", "", fmt.Errorf("responses request factory is nil")
	}
	if options.ExecuteRequest == nil {
		return "", "", fmt.Errorf("responses executor is nil")
	}
	if options.HandleStreaming == nil || options.HandleNonStreaming == nil {
		return "", "", fmt.Errorf("responses handlers are nil")
	}
	if options.HandleHTTPError == nil {
		options.HandleHTTPError = api.HandleHTTPError
	}

	for attempt := 0; attempt < 2; attempt++ {
		reqBody := options.BuildRequest()
		if options.RequestObserver != nil {
			options.RequestObserver(reqBody)
		}
		if options.SetLocalAutoCompressSkip != nil {
			options.SetLocalAutoCompressSkip(reqBody.SkipLocalAutoCompressionAfterResponse)
		}
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal request: %w", err)
		}
		writeResponsesDebugRequest(options, payload)

		req, err := options.PrepareRequest(ctx, options.URL, payload)
		if err != nil {
			return "", "", err
		}

		spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)
		resp, err := options.ExecuteRequest(req, reqBody.Stream)
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
			return "", "", options.HandleHTTPError(resp, spinner, options.ProviderName)
		}

		if reqBody.Stream {
			return options.HandleStreaming(ctx, resp, spinner)
		}
		return options.HandleNonStreaming(ctx, resp, spinner)
	}

	return "", "", fmt.Errorf("responses API retry exhausted")
}

func writeResponsesDebugRequest(options RunOptions, payload []byte) {
	if !options.Debug || options.DebugWriter == nil {
		return
	}
	debugName := options.DebugName
	if debugName == "" {
		debugName = "Responses"
	}
	if options.DebugPayloadRedactor != nil {
		payload = options.DebugPayloadRedactor(payload)
	}
	fmt.Fprintf(options.DebugWriter, "[DEBUG %s Responses] Request body:\n%s\n", debugName, string(payload))
}

func shouldRetryResponsesWithoutPreviousResponseID(resp *http.Response, options RunOptions, attempt int) bool {
	if resp.StatusCode != http.StatusBadRequest || attempt > 0 {
		return false
	}
	if options.HasPreviousResponseID == nil || options.ClearPreviousResponseID == nil {
		return false
	}
	return options.HasPreviousResponseID()
}

// HandleNonStreaming は Responses API の非ストリーミング response を処理する。
func HandleNonStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner, options NonStreamingOptions) (string, string, error) {
	var result responsesNonStreamingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		api.StopSpinner(spinner)
		return "", "", err
	}
	api.StopSpinner(spinner)

	providerName := strings.TrimSpace(options.ProviderName)
	if providerName == "" {
		providerName = "OpenAI"
	}
	if err := result.validateSuccess(providerName); err != nil {
		return "", "", err
	}

	emitResponsesNonStreamingUsage(result.Usage, options.UsageCallback)
	text := result.extractText()
	if options.ReplayItemsCallback != nil {
		options.ReplayItemsCallback(result.openAIResponsesReplayItems(text))
	}
	content := composeResponsesNonStreamingContent(text, result.extractToolJSON())
	printResponsesNonStreamingText(ctx, text)
	return content, result.ID, nil
}

func (r responsesNonStreamingResponse) validateSuccess(providerName string) error {
	if r.Error != nil {
		return formatResponsesNonStreamingError(providerName, r.Status, r.Error)
	}
	if r.Status == "" || r.Status == "completed" {
		return nil
	}
	if r.Status == "failed" {
		return fmt.Errorf("%s Responses API request failed", providerName)
	}
	return fmt.Errorf("%s Responses API response status: %s", providerName, r.Status)
}

func formatResponsesNonStreamingError(providerName, status string, responsesError *Error) error {
	if responsesError.Message != "" {
		return errors.New(responsesError.Message)
	}
	if responsesError.Code != "" {
		return fmt.Errorf("%s API error: %s", providerName, responsesError.Code)
	}
	if responsesError.Type != "" {
		return fmt.Errorf("%s API error: %s", providerName, responsesError.Type)
	}
	if status == "failed" {
		return fmt.Errorf("%s Responses API request failed", providerName)
	}
	return fmt.Errorf("%s API error", providerName)
}

func emitResponsesNonStreamingUsage(usage *Usage, callback api.UsageCallback) {
	if callback == nil {
		return
	}
	apiUsage := responsesUsageToAPIUsage(usage)
	if apiUsage == nil {
		return
	}
	callback(*apiUsage)
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

func (r responsesNonStreamingResponse) openAIResponsesReplayItems(text string) []api.InputItem {
	items := make([]api.InputItem, 0, len(r.Output)+1)
	for _, output := range r.Output {
		switch output.Type {
		case "message":
			messageText := output.extractText()
			if messageText == "" {
				continue
			}
			items = append(items, api.InputItem{
				Type:    "message",
				Role:    "assistant",
				ID:      output.ID,
				Status:  output.Status,
				Content: messageText,
			})
		case "function_call":
			items = append(items, api.InputItem{
				Type:      "function_call",
				ID:        output.ID,
				Status:    output.Status,
				CallID:    output.CallID,
				Name:      output.Name,
				Arguments: output.Arguments,
			})
		case "reasoning":
			items = append(items, api.InputItem{
				Type:             "reasoning",
				ID:               output.ID,
				Status:           output.Status,
				Summary:          output.Summary,
				EncryptedContent: output.EncryptedContent,
			})
		}
	}
	if len(items) == 0 && text != "" {
		items = append(items, api.InputItem{
			Type:    "message",
			Role:    "assistant",
			Content: text,
		})
	}
	return items
}

func (i responsesNonStreamingOutputItem) extractText() string {
	var text strings.Builder
	for _, part := range i.Content {
		if part.Type == "output_text" {
			text.WriteString(part.Text)
		}
	}
	return text.String()
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
