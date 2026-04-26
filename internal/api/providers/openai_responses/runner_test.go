package openairesponses

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRunStreaming_RetriesOnceWithoutInvalidPreviousResponseID(t *testing.T) {
	previousResponseID := "resp_old"
	var payloads []string
	executeCalls := 0

	content, responseID, err := RunStreaming(context.Background(), StreamingRunOptions{
		URL: "https://example.test/v1/responses",
		BuildRequest: func() Request {
			return Request{
				Model:              "gpt-5.4",
				Input:              "hello",
				PreviousResponseID: previousResponseID,
				Stream:             true,
				Store:              true,
			}
		},
		NewRequest: func(ctx context.Context, url string, payload []byte) (*http.Request, error) {
			payloads = append(payloads, string(payload))
			return http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
		},
		ExecuteRequest: func(*http.Request) (*http.Response, error) {
			executeCalls++
			status := http.StatusOK
			if executeCalls == 1 {
				status = http.StatusBadRequest
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		},
		StreamHandler: func(_ context.Context, _ *http.Response, spinner *ui.Spinner) (string, string, error) {
			spinner.Stop()
			return "ok", "resp_new", nil
		},
		HasPreviousResponseID: func() bool {
			return previousResponseID != ""
		},
		ClearPreviousResponseID: func() {
			previousResponseID = ""
		},
	})
	if err != nil {
		t.Fatalf("RunStreaming() error = %v", err)
	}
	if content != "ok" || responseID != "resp_new" {
		t.Fatalf("RunStreaming() = (%q, %q), want (ok, resp_new)", content, responseID)
	}
	if executeCalls != 2 {
		t.Fatalf("executeCalls = %d, want 2", executeCalls)
	}
	if len(payloads) != 2 {
		t.Fatalf("len(payloads) = %d, want 2", len(payloads))
	}
	if !strings.Contains(payloads[0], `"previous_response_id":"resp_old"`) {
		t.Fatalf("first payload = %s, want previous_response_id", payloads[0])
	}
	if strings.Contains(payloads[1], "previous_response_id") {
		t.Fatalf("second payload = %s, want previous_response_id cleared", payloads[1])
	}
}
