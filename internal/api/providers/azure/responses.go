package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type responsesRequestRunOptions struct {
	URL                     string
	BuildRequest            func() openairesponses.Request
	HasPreviousResponseID   func() bool
	ClearPreviousResponseID func()
	DebugName               string
	Debug                   bool
	DebugWriter             io.Writer
}

func (p *Provider) chatWithResponses(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	storeResponses := config.FromContext(ctx).ResponsesStoreEnabled()
	content, responseID, err := p.runResponsesRequest(ctx, responsesRequestRunOptions{
		URL:          p.responsesURL(),
		BuildRequest: func() openairesponses.Request { return p.buildChatResponsesRequest(ctx, systemPrompt, history, model) },
		DebugName:    "Azure",
		Debug:        os.Getenv("XELYON_DEBUG_AZURE") == "1",
		DebugWriter:  api.ErrorWriterFromContext(ctx),
		HasPreviousResponseID: func() bool {
			return storeResponses && p.HasCachedResponseID()
		},
		ClearPreviousResponseID: func() {
			p.ClearResponseID()
		},
	})
	if !storeResponses {
		p.ClearResponseID()
		return content, err
	}
	if err == nil && responseID != "" {
		p.SetResponseID(responseID)
	}
	return content, err
}

func (p *Provider) chatWithImageResponses(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	storeResponses := config.FromContext(ctx).ResponsesStoreEnabled()
	content, responseID, err := p.runResponsesRequest(ctx, responsesRequestRunOptions{
		URL: p.responsesURL(),
		BuildRequest: func() openairesponses.Request {
			return p.buildImageResponsesRequest(ctx, systemPrompt, history, userMessage, image, model)
		},
		DebugName:   "Azure",
		Debug:       os.Getenv("XELYON_DEBUG_AZURE") == "1",
		DebugWriter: api.ErrorWriterFromContext(ctx),
	})
	if !storeResponses {
		p.ClearResponseID()
		return content, err
	}
	if err == nil && responseID != "" {
		p.SetResponseID(responseID)
	}
	return content, err
}

func (p *Provider) runResponsesRequest(ctx context.Context, options responsesRequestRunOptions) (string, string, error) {
	if options.BuildRequest == nil {
		return "", "", fmt.Errorf("responses request builder is nil")
	}

	previousResponseRetryUsed := false
	authRetryUsed := false
	for attempt := 0; attempt < 3; attempt++ {
		reqBody := options.BuildRequest()
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal request: %w", err)
		}
		writeResponsesDebugRequest(options, payload)

		req, err := p.newAuthJSONRequest(ctx, options.URL, payload)
		if err != nil {
			return "", "", err
		}

		spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)
		resp, err := p.executeResponsesRequest(req, reqBody.Stream)
		if err != nil {
			spinner.Stop()
			return "", "", fmt.Errorf("API request failed: %w", err)
		}

		if shouldRetryResponsesWithoutPreviousResponseID(resp, options, previousResponseRetryUsed) {
			spinner.Stop()
			resp.Body.Close()
			options.ClearPreviousResponseID()
			previousResponseRetryUsed = true
			continue
		}

		if p.shouldRetryResponsesAfterAuthRefresh(ctx, resp, authRetryUsed) {
			spinner.Stop()
			resp.Body.Close()
			if err := p.refreshAuthToken(ctx); err != nil {
				return "", "", fmt.Errorf("Azure OpenAI auth token refresh failed: %w", err)
			}
			authRetryUsed = true
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", "", handleAzureResponsesHTTPError(resp, spinner, azureHTTPErrorContext{
				Deployment:  reqBody.Model,
				ToolPayload: len(reqBody.Tools) > 0 || reqBody.ToolChoice != nil,
			})
		}

		if reqBody.Stream {
			return p.handleResponsesStreaming(ctx, resp, spinner)
		}
		return openai.HandleResponsesNonStreaming(ctx, resp, spinner, openai.ResponsesNonStreamingOptions{
			ProviderName:  p.Name(),
			UsageCallback: p.usageCallback,
		})
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

func shouldRetryResponsesWithoutPreviousResponseID(resp *http.Response, options responsesRequestRunOptions, retryUsed bool) bool {
	if resp.StatusCode != http.StatusBadRequest || retryUsed {
		return false
	}
	if options.HasPreviousResponseID == nil || options.ClearPreviousResponseID == nil {
		return false
	}
	return options.HasPreviousResponseID()
}

func (p *Provider) shouldRetryResponsesAfterAuthRefresh(ctx context.Context, resp *http.Response, retryUsed bool) bool {
	if resp.StatusCode != http.StatusUnauthorized || retryUsed {
		return false
	}
	if !p.canRefreshAuthToken() {
		return false
	}
	return ctx.Err() == nil
}

func (p *Provider) newAuthJSONRequest(ctx context.Context, url string, payload []byte) (*http.Request, error) {
	apiKey := strings.TrimSpace(p.APIKey)
	if apiKey != "" {
		return openaicompat.NewHeaderJSONBytesRequest(ctx, url, "api-key", apiKey, payload)
	}
	authToken := strings.TrimSpace(p.authToken)
	if authToken == "" {
		var err error
		authToken, err = p.currentAuthToken(ctx)
		if err != nil {
			return nil, err
		}
	}
	return openaicompat.NewBearerJSONBytesRequest(ctx, url, authToken, payload)
}

func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	debugEnabled := os.Getenv("XELYON_DEBUG_AZURE") == "1"
	return openai.HandleResponsesStreaming(ctx, resp, spinner, openai.ResponsesStreamingOptions{
		ProviderName:  p.Name(),
		DebugName:     "Azure",
		Debug:         debugEnabled,
		DebugOverride: &debugEnabled,
		DebugWriter:   api.ErrorWriterFromContext(ctx),
		UsageCallback: p.usageCallback,
	})
}
