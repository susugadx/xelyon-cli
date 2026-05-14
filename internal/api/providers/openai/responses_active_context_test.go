package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const openAITestActiveContextSnapshot = "<current_task_state>\nstate\n</current_task_state>"

func TestBuildChatResponsesRequest_IncludesActiveContextFromContext(t *testing.T) {
	p := New("test-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(
		openAITestResponsesRequestContextWithActiveContext(),
		"system",
		[]api.Message{{Role: "user", Content: "hi"}},
		"gpt-5.4",
	)

	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty when active context is present", req.PreviousResponseID)
	}
	if len(req.ContextManagement) != 0 {
		t.Fatalf("ContextManagement = %#v, want omitted without previous_response_id", req.ContextManagement)
	}
	inputItems := openAITestResponsesInputItems(t, req)
	if len(inputItems) != 3 {
		t.Fatalf("Input length = %d, want developer plus active context plus history", len(inputItems))
	}
	assertOpenAITestInputMessage(t, inputItems[1], "developer", openAITestActiveContextSnapshot)
	assertOpenAITestInputMessage(t, inputItems[2], "user", "hi")
}

func TestBuildImageResponsesRequest_IncludesActiveContextFromContext(t *testing.T) {
	req := New("test-key").buildImageResponsesRequest(
		openAITestResponsesRequestContextWithActiveContext(),
		"system",
		[]api.Message{{Role: "user", Content: "before"}},
		"what is this?",
		&api.ImageData{MediaType: "image/png", Base64: "abc123"},
		"gpt-5.5-pro",
	)

	inputItems := openAITestResponsesInputItems(t, req)
	if len(inputItems) != 4 {
		t.Fatalf("Input length = %d, want developer plus active context plus history plus image", len(inputItems))
	}
	assertOpenAITestInputMessage(t, inputItems[1], "developer", openAITestActiveContextSnapshot)
	assertOpenAITestInputMessage(t, inputItems[2], "user", "before")
	if inputItems[3].Role != "user" {
		t.Fatalf("Input[3] = %#v, want image user message", inputItems[3])
	}
}

func TestChatWithResponses_ActiveContextDoesNotCacheResponseID(t *testing.T) {
	var requests []map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requests = append(requests, raw)

		if _, ok := raw["previous_response_id"]; ok {
			t.Fatalf("request should omit cached previous_response_id when active context is present: %#v", raw)
		}
		if _, ok := raw["context_management"]; ok {
			t.Fatalf("request should omit context_management without previous_response_id: %#v", raw["context_management"])
		}
		if !openAITestRequestInputContainsContent(raw, openAITestActiveContextSnapshot) {
			t.Fatalf("request should include active context in full-history input: %#v", raw["input"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_new","output_text":"Fresh"}`))
	})
	t.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	p.SetResponseID("resp_old")
	content, err := p.chatWithResponses(
		openAITestRuntimeContextWithActiveContext(t),
		"system",
		[]api.Message{{Role: "user", Content: "hi"}},
		"gpt-5.5-pro",
	)
	if err != nil {
		t.Fatalf("chatWithResponses() error = %v", err)
	}
	if content != "Fresh" {
		t.Fatalf("content = %q, want Fresh", content)
	}
	if p.GetResponseID() != "" {
		t.Fatalf("GetResponseID() = %q, want empty after active-context request", p.GetResponseID())
	}
	if len(requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(requests))
	}
}

func TestChatWithImageResponses_ActiveContextDoesNotCacheResponseID(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !openAITestRequestInputContainsContent(raw, openAITestActiveContextSnapshot) {
			t.Fatalf("request should include active context in image input: %#v", raw["input"])
		}
		if raw["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_image\"}}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"Fresh image\"}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"response.completed\"}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_image","output_text":"Fresh image"}`))
	})
	t.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	p.SetResponseID("resp_old")
	content, err := p.chatWithImageResponses(
		openAITestRuntimeContextWithActiveContext(t),
		"system",
		[]api.Message{{Role: "user", Content: "before"}},
		"what is this?",
		&api.ImageData{MediaType: "image/png", Base64: "abc123"},
		"gpt-5.4",
	)
	if err != nil {
		t.Fatalf("chatWithImageResponses() error = %v", err)
	}
	if content != "Fresh image" {
		t.Fatalf("content = %q, want Fresh image", content)
	}
	if p.GetResponseID() != "" {
		t.Fatalf("GetResponseID() = %q, want empty after active-context image request", p.GetResponseID())
	}
}

func openAITestResponsesRequestContextWithActiveContext() context.Context {
	ctx := config.WithContext(context.Background(), config.DefaultConfig())
	return api.WithActiveContextBlocks(ctx, openAITestActiveContextBlocks())
}

func openAITestRuntimeContextWithActiveContext(t *testing.T) context.Context {
	t.Helper()
	return api.WithActiveContextBlocks(newOpenAITestContext(t, false), openAITestActiveContextBlocks())
}

func openAITestActiveContextBlocks() []api.ActiveContextBlock {
	return []api.ActiveContextBlock{{
		Name:    "current_task_state",
		Content: openAITestActiveContextSnapshot,
	}}
}

func openAITestResponsesInputItems(t *testing.T, req ResponsesRequest) []openairesponses.InputItem {
	t.Helper()
	inputItems, ok := req.Input.([]openairesponses.InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []openairesponses.InputItem", req.Input)
	}
	return inputItems
}

func assertOpenAITestInputMessage(t *testing.T, item openairesponses.InputItem, role, content string) {
	t.Helper()
	if item.Type != "message" || item.Role != role || item.Content != content {
		t.Fatalf("Input item = %#v, want %s message with content %q", item, role, content)
	}
}

func openAITestRequestInputContainsContent(raw map[string]any, content string) bool {
	input, ok := raw["input"].([]any)
	if !ok {
		return false
	}
	for _, item := range input {
		message, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := message["content"].(string); ok && text == content {
			return true
		}
	}
	return false
}
