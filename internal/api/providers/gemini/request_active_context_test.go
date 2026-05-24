package gemini

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const geminiTestActiveContextSnapshot = "<rehydrated_evidence>\nREADME.md:L1-L2\n</rehydrated_evidence>"

func newGeminiRequestContextWithActiveContext() context.Context {
	return api.WithActiveContextBlocks(newGeminiRequestContext(false, "medium"), []api.ActiveContextBlock{{
		Name:    "provider_history_rehydrated_evidence",
		Content: geminiTestActiveContextSnapshot,
	}})
}

func TestBuildGeminiTextRequest_InsertsActiveContextBeforeLatestUserAndKeepsCacheClean(t *testing.T) {
	ctx := newGeminiRequestContextWithActiveContext()
	req := buildGeminiTextRequest(ctx, "System prompt", []api.Message{
		{Role: "user", Content: "old question"},
		{Role: "assistant", Content: "old answer"},
		{Role: "user", Content: "latest question"},
	}, "gemini-2.5-flash", "cachedContents/main", config.DefaultConfig())

	if req.CachedContent != "cachedContents/main" {
		t.Fatalf("CachedContent = %q, want cached cache name", req.CachedContent)
	}
	if req.SystemInstruction != nil {
		t.Fatalf("SystemInstruction = %+v, want nil when cacheName is set", req.SystemInstruction)
	}
	if len(req.Contents) != 4 {
		t.Fatalf("len(Contents) = %d, want old user/model + active context + latest user", len(req.Contents))
	}
	if req.Contents[2].Role != "user" || req.Contents[2].Parts[0].Text != geminiTestActiveContextSnapshot {
		t.Fatalf("Contents[2] = %+v, want active context user content before latest request", req.Contents[2])
	}
	if req.Contents[3].Role != "user" || req.Contents[3].Parts[0].Text != "latest question" {
		t.Fatalf("Contents[3] = %+v, want latest user request after active context", req.Contents[3])
	}
}

func TestBuildGeminiFunctionCallingRequest_InsertsActiveContextWithoutBreakingToolContinuity(t *testing.T) {
	ctx := newGeminiRequestContextWithActiveContext()
	req := buildGeminiFunctionCallingRequest(
		ctx,
		"System prompt",
		[]api.Message{
			{Role: "user", Content: "inspect README"},
			{
				Role: "assistant",
				ToolCalls: []api.OpenAIToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"README.md"}`,
					},
				}},
			},
			{Role: "tool", ToolCallID: "call_1", ToolName: "read_file", Content: "README contents"},
		},
		"gemini-3.1-pro-preview-customtools",
		"",
		nil,
		nil,
		config.DefaultConfig(),
	)

	if len(req.Contents) != 4 {
		t.Fatalf("len(Contents) = %d, want active context + user + model functionCall + functionResponse", len(req.Contents))
	}
	active, ok := req.Contents[0].(GeminiContent)
	if !ok || active.Role != "user" || active.Parts[0].Text != geminiTestActiveContextSnapshot {
		t.Fatalf("Contents[0] = %#v, want active context user content", req.Contents[0])
	}
	modelTurn, ok := req.Contents[2].(GeminiGenericContent)
	if !ok || modelTurn.Role != "model" {
		t.Fatalf("Contents[2] = %#v, want model functionCall turn", req.Contents[2])
	}
	if _, ok := modelTurn.Parts[0].(GeminiFunctionCallPart); !ok {
		t.Fatalf("model parts = %#v, want functionCall after original user request", modelTurn.Parts)
	}
	toolTurn, ok := req.Contents[3].(GeminiGenericContent)
	if !ok || toolTurn.Role != "user" {
		t.Fatalf("Contents[3] = %#v, want functionResponse user turn", req.Contents[3])
	}
	if _, ok := toolTurn.Parts[0].(GeminiFunctionResponsePart); !ok {
		t.Fatalf("tool parts = %#v, want functionResponse after functionCall", toolTurn.Parts)
	}
}

func TestBuildGeminiMultimodalRequest_InsertsActiveContextBeforeCurrentImageMessage(t *testing.T) {
	ctx := newGeminiRequestContextWithActiveContext()
	req := buildGeminiMultimodalRequest(
		ctx,
		"System prompt",
		[]api.Message{{Role: "user", Content: "previous text"}},
		"describe image",
		&api.ImageData{MediaType: "image/png", Base64: "aW1hZ2U="},
		"gemini-2.5-flash",
		nil,
		false,
		config.DefaultConfig(),
	)

	if len(req.Contents) != 3 {
		t.Fatalf("len(Contents) = %d, want history + active context + current image request", len(req.Contents))
	}
	active, ok := req.Contents[1].(GeminiContent)
	if !ok || active.Role != "user" || active.Parts[0].Text != geminiTestActiveContextSnapshot {
		t.Fatalf("Contents[1] = %#v, want active context before multimodal user request", req.Contents[1])
	}
	imageTurn, ok := req.Contents[2].(GeminiMultimodalContent)
	if !ok || imageTurn.Role != "user" || len(imageTurn.Parts) != 2 || imageTurn.Parts[0].InlineData == nil || imageTurn.Parts[1].Text != "describe image" {
		t.Fatalf("Contents[2] = %#v, want current image request after active context", req.Contents[2])
	}
}
