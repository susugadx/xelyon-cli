package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func newGeminiResponseContext() (context.Context, *bytes.Buffer, *bytes.Buffer) {
	cfg := config.DefaultConfig()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := uiruntime.WithRuntime(context.Background(), uiruntime.NewRuntime(nil, out, errOut))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return config.WithContext(ctx, cfg), out, errOut
}

func geminiSSEPayload(t *testing.T, chunk GeminiFunctionResponse) string {
	t.Helper()
	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return "data: " + string(data) + "\n\n"
}

func geminiSSEResponse(body string) *http.Response {
	return &http.Response{Body: io.NopCloser(strings.NewReader(body))}
}

func TestHandleSSEResponse_RescuesCodeBlockToolJSONAndReportsUsage(t *testing.T) {
	p := New("test-key")
	var usage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		usage = u
	})

	ctx, _, errOut := newGeminiResponseContext()
	body := geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{
				Parts: []GeminiFunctionPart{{
					Text: "Before\n```json\n{\"tool\":\"read_file\",\"args\":{\"path\":\"/tmp/demo.txt\"}}\n```\nAfter",
				}},
			},
		}},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:        11,
			CandidatesTokenCount:    5,
			ThoughtsTokenCount:      2,
			CachedContentTokenCount: 3,
		},
	})

	got, err := p.handleSSEResponse(ctx, geminiSSEResponse(body), nil, "", "")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Fatalf("result = %q, want surrounding prose", got)
	}
	if !strings.Contains(got, `"tool":"read_file"`) {
		t.Fatalf("result = %q, want rescued tool JSON", got)
	}
	if !strings.Contains(errOut.String(), "FC rescue") {
		t.Fatalf("errOut = %q, want FC rescue warning", errOut.String())
	}
	if usage.InputTokens != 11 || usage.OutputTokens != 5 || usage.ThinkingTokens != 2 || usage.CachedInputTokens != 3 {
		t.Fatalf("usage = %+v, want prompt=11 output=5 thinking=2 cached=3", usage)
	}
}

func TestHandleSSEResponse_ReportsBillingServiceTierDowngrade(t *testing.T) {
	p := New("test-key")
	var usage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		usage = u
	})

	ctx, _, _ := newGeminiResponseContext()
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierPriority
	ctx = config.WithContext(ctx, cfg)

	body := geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{
				Parts: []GeminiFunctionPart{{Text: "done"}},
			},
		}},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     11,
			CandidatesTokenCount: 5,
		},
	})
	resp := geminiSSEResponse(body)
	resp.Header = make(http.Header)
	resp.Header.Set("x-gemini-service-tier", config.GeminiServiceTierStandard)

	if _, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash"); err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if usage.BillingServiceTier != config.GeminiServiceTierStandard {
		t.Fatalf("BillingServiceTier = %q, want standard downgrade", usage.BillingServiceTier)
	}
}

func TestHandleSSEResponse_ReportsBillingServiceTierFromUsageMetadata(t *testing.T) {
	p := New("test-key")
	var usage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		usage = u
	})

	ctx, _, _ := newGeminiResponseContext()
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierPriority
	ctx = config.WithContext(ctx, cfg)

	body := geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{
				Parts: []GeminiFunctionPart{{Text: "done"}},
			},
		}},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     11,
			CandidatesTokenCount: 5,
			ServiceTier:          config.GeminiServiceTierStandard,
		},
	})

	if _, err := p.handleSSEResponse(ctx, geminiSSEResponse(body), nil, "", "gemini-3.5-flash"); err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if usage.BillingServiceTier != config.GeminiServiceTierStandard {
		t.Fatalf("BillingServiceTier = %q, want usageMetadata standard downgrade", usage.BillingServiceTier)
	}
}

func TestHandleSSEResponse_NoContentReturnsError(t *testing.T) {
	p := New("test-key")
	ctx, _, _ := newGeminiResponseContext()

	body := geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates:    []GeminiFunctionCandidate{},
		UsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 1},
	})

	_, err := p.handleSSEResponse(ctx, geminiSSEResponse(body), nil, "", "")
	if err == nil {
		t.Fatal("handleSSEResponse() should return error when stream has no content")
	}
	if !strings.Contains(err.Error(), "no content in Gemini SSE response") {
		t.Fatalf("handleSSEResponse() error = %q, want no content message", err.Error())
	}
}
