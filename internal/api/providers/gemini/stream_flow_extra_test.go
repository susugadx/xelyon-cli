package gemini

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestHandleSSEResponse_ContextCancelReturnsPartialText(t *testing.T) {
	p := New("test-key")
	baseCtx, _, _ := newGeminiResponseContext()
	ctx, cancel := context.WithCancel(baseCtx)

	pr, pw := io.Pipe()
	go func() {
		_, _ = fmt.Fprint(pw, geminiSSEPayload(t, GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{
				Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "partial"}}},
			}},
		}))
		time.Sleep(50 * time.Millisecond)
		cancel()
		_ = pw.Close()
	}()

	got, err := p.handleSSEResponse(ctx, &http.Response{Body: pr}, nil, "", "")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if got != "partial" {
		t.Fatalf("handleSSEResponse() = %q, want %q", got, "partial")
	}
}

func TestHandleSSEResponse_SuppressesSplitToolJSONAcrossChunks(t *testing.T) {
	p := New("test-key")
	ctx, _, _ := newGeminiResponseContext()

	body := geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: `{"tool":"read`}}},
		}},
	}) + geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: `_file","args":{"path":"/tmp/demo.txt"}}`}}},
		}},
	})

	got, err := p.handleSSEResponse(ctx, geminiSSEResponse(body), nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	want := `{"tool":"read_file","args":{"path":"/tmp/demo.txt"}}`
	if got != want {
		t.Fatalf("handleSSEResponse() = %q, want %q", got, want)
	}
}

func TestHandleSSEResponse_DedupesSignatureFunctionCallsAndCarriesThoughtParts(t *testing.T) {
	t.Setenv("XELYON_DEBUG_GEMINI", "1")

	p := New("test-key")
	ctx, _, errOut := newGeminiResponseContext()

	body := geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{
				{Thought: true, Text: "planning", ThoughtSignature: "sig-thought"},
				{
					Text:             "hidden signature text",
					ThoughtSignature: "sig-1",
					FunctionCall: &api.GeminiFunctionCall{
						Name: "read_file",
						Args: map[string]any{"path": "/tmp/demo.txt"},
					},
				},
			}},
		}},
	}) + geminiSSEPayload(t, GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{
				ThoughtSignature: "sig-2",
				FunctionCall: &api.GeminiFunctionCall{
					Name: "read_file",
					Args: map[string]any{"path": "/tmp/demo.txt"},
				},
			}}},
		}},
	})

	got, err := p.handleSSEResponse(ctx, geminiSSEResponse(body), nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if count := strings.Count(got, `"tool":"read_file"`); count != 1 {
		t.Fatalf("result = %q, want single deduped function call", got)
	}
	if !strings.Contains(got, `"thought_signature":"sig-1"`) || !strings.Contains(got, `"thought_parts"`) {
		t.Fatalf("result = %q, want thought metadata in tool JSON", got)
	}
	if strings.HasPrefix(got, "hidden signature text") {
		t.Fatalf("result = %q, want signature text to stay out of plain output", got)
	}
	if !strings.Contains(errOut.String(), "Collected signature part") {
		t.Fatalf("errOut = %q, want signature debug log", errOut.String())
	}
}

func TestHandleSSEResponse_ScanErrorAfterPartialChunk(t *testing.T) {
	p := New("test-key")
	ctx, _, _ := newGeminiResponseContext()
	spinner := uiruntime.NewSpinnerWithWriter(io.Discard)
	spinner.Start("Waiting for Gemini...")

	readErr := errors.New("boom")
	body := io.NopCloser(io.MultiReader(
		strings.NewReader(geminiSSEPayload(t, GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{
				Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "partial"}}},
			}},
		})),
		iotest.ErrReader(readErr),
	))

	_, err := p.handleSSEResponse(ctx, &http.Response{Body: body}, spinner, "", "")
	if err == nil {
		t.Fatal("handleSSEResponse() should return scan error")
	}
	if !strings.Contains(err.Error(), "SSE scan error") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("handleSSEResponse() error = %q, want scan error with cause", err.Error())
	}
	if spinner.IsActive() {
		t.Fatal("handleSSEResponse() should stop spinner on scan error")
	}
}
