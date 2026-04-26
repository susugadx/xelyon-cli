package openairesponses

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// JSONRequestFactory は marshal 済み JSON payload から HTTP request を作る関数を表す。
type JSONRequestFactory func(ctx context.Context, url string, payload []byte) (*http.Request, error)

// StreamHandler は Responses API の streaming response を処理する関数を表す。
type StreamHandler func(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (content, responseID string, err error)

// StreamingRunOptions は Responses API streaming 実行の provider 差分を表す。
type StreamingRunOptions struct {
	URL                     string
	BuildRequest            func() Request
	NewRequest              JSONRequestFactory
	ExecuteRequest          func(*http.Request) (*http.Response, error)
	StreamHandler           StreamHandler
	ProviderName            string
	DebugName               string
	Debug                   bool
	DebugWriter             io.Writer
	HasPreviousResponseID   func() bool
	ClearPreviousResponseID func()
}

// RunStreaming は Responses API streaming request を送信し、invalid previous_response_id を一度だけ回復する。
func RunStreaming(ctx context.Context, options StreamingRunOptions) (string, string, error) {
	if options.BuildRequest == nil {
		return "", "", fmt.Errorf("responses request builder is nil")
	}
	if options.NewRequest == nil {
		return "", "", fmt.Errorf("responses request factory is nil")
	}
	if options.ExecuteRequest == nil {
		return "", "", fmt.Errorf("responses executor is nil")
	}
	if options.StreamHandler == nil {
		return "", "", fmt.Errorf("responses stream handler is nil")
	}

	providerName := options.ProviderName
	if providerName == "" {
		providerName = "Responses API"
	}

	for attempt := 0; attempt < 2; attempt++ {
		reqBody := options.BuildRequest()
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal request: %w", err)
		}
		writeDebugRequest(options, payload)

		req, err := options.NewRequest(ctx, options.URL, payload)
		if err != nil {
			return "", "", err
		}

		spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)
		resp, err := options.ExecuteRequest(req)
		if err != nil {
			spinner.Stop()
			return "", "", fmt.Errorf("API request failed: %w", err)
		}

		if shouldRetryWithoutPreviousResponseID(resp, options, attempt) {
			spinner.Stop()
			resp.Body.Close()
			options.ClearPreviousResponseID()
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", "", api.HandleHTTPError(resp, spinner, providerName)
		}

		return options.StreamHandler(ctx, resp, spinner)
	}

	return "", "", fmt.Errorf("responses API retry exhausted")
}

func writeDebugRequest(options StreamingRunOptions, payload []byte) {
	if !options.Debug || options.DebugWriter == nil {
		return
	}
	debugName := options.DebugName
	if debugName == "" {
		debugName = "Responses"
	}
	fmt.Fprintf(options.DebugWriter, "[DEBUG %s Responses] Request body:\n%s\n", debugName, string(payload))
}

func shouldRetryWithoutPreviousResponseID(resp *http.Response, options StreamingRunOptions, attempt int) bool {
	if resp.StatusCode != http.StatusBadRequest || attempt > 0 {
		return false
	}
	if options.HasPreviousResponseID == nil || options.ClearPreviousResponseID == nil {
		return false
	}
	return options.HasPreviousResponseID()
}
