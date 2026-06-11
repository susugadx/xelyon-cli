package openai

import (
	"context"
	"io"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type responsesRequestRunOptions struct {
	URL                      string
	BuildRequest             func() ResponsesRequest
	PrepareRequest           func(ctx context.Context, url string, payload []byte) (*http.Request, error)
	ExecuteRequest           func(req *http.Request, stream bool) (*http.Response, error)
	HandleStreaming          func(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error)
	HandleNonStreaming       func(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error)
	HandleHTTPError          func(resp *http.Response, spinner *ui.Spinner, providerName string) error
	RequestObserver          func(ResponsesRequest)
	SetLocalAutoCompressSkip func(bool)
	HasPreviousResponseID    func() bool
	ClearPreviousResponseID  func()
	ProviderName             string
	DebugName                string
	Debug                    bool
	DebugWriter              io.Writer
	DebugPayloadRedactor     func([]byte) []byte
}

func (p *Provider) runResponsesRequest(ctx context.Context, options responsesRequestRunOptions) (string, string, error) {
	if options.PrepareRequest == nil {
		options.PrepareRequest = func(ctx context.Context, url string, payload []byte) (*http.Request, error) {
			return openaicompat.NewBearerJSONBytesRequest(ctx, url, p.APIKey, payload)
		}
	}
	if options.ExecuteRequest == nil {
		options.ExecuteRequest = p.executeResponsesRequest
	}
	if options.HandleStreaming == nil {
		options.HandleStreaming = p.handleResponsesStreaming
	}
	if options.HandleNonStreaming == nil {
		options.HandleNonStreaming = p.handleResponsesNonStreaming
	}
	if options.RequestObserver == nil {
		options.RequestObserver = p.responsesRequestObserver
	}
	if options.SetLocalAutoCompressSkip == nil {
		options.SetLocalAutoCompressSkip = func(skip bool) {
			p.responsesLocalAutoCompressSkip = skip
		}
	}
	if options.ProviderName == "" {
		options.ProviderName = p.Name()
	}
	return runResponsesRequest(ctx, options)
}

func runResponsesRequest(ctx context.Context, options responsesRequestRunOptions) (string, string, error) {
	return openairesponses.RunResponsesRequest(ctx, openairesponses.RunOptions{
		URL:                      options.URL,
		BuildRequest:             options.BuildRequest,
		PrepareRequest:           options.PrepareRequest,
		ExecuteRequest:           options.ExecuteRequest,
		HandleStreaming:          options.HandleStreaming,
		HandleNonStreaming:       options.HandleNonStreaming,
		HandleHTTPError:          options.HandleHTTPError,
		RequestObserver:          options.RequestObserver,
		SetLocalAutoCompressSkip: options.SetLocalAutoCompressSkip,
		HasPreviousResponseID:    options.HasPreviousResponseID,
		ClearPreviousResponseID:  options.ClearPreviousResponseID,
		ProviderName:             options.ProviderName,
		DebugName:                options.DebugName,
		Debug:                    options.Debug,
		DebugWriter:              options.DebugWriter,
		DebugPayloadRedactor:     options.DebugPayloadRedactor,
	})
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

// NewLongRunningResponsesHTTPClient は非ストリーミング Responses 用に header timeout を外した HTTP client を返す。
func NewLongRunningResponsesHTTPClient(base *http.Client) *http.Client {
	return openairesponses.NewLongRunningHTTPClient(base)
}

func newLongRunningResponsesHTTPClient(base *http.Client) *http.Client {
	return NewLongRunningResponsesHTTPClient(base)
}

func (p *Provider) handleResponsesNonStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	return openairesponses.HandleNonStreaming(ctx, resp, spinner, openairesponses.NonStreamingOptions{
		ProviderName:        "OpenAI",
		UsageCallback:       p.usageCallback,
		ReplayItemsCallback: p.setLastOpenAIResponsesInputItems,
	})
}
