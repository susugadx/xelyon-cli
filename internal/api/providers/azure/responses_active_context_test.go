package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

var azureTestActiveContextSnapshot = taskstate.RenderRehydratedEvidenceBlock(taskstate.RehydratedEvidenceBlock{Items: []taskstate.RehydratedEvidenceItem{{
	Path:       "README.md",
	StartLine:  1,
	EndLine:    2,
	Source:     "read_file",
	Reason:     taskstate.RehydratePlanReasonOmittedProviderHistory,
	ToolCallID: "call_read",
	Content:    "line one\nline two",
}}})

func TestBuildChatResponsesRequest_IncludesActiveContextFromContext(t *testing.T) {
	ctx := azureTestContextWithActiveContext(azureTestResponsesModelConfig("corp-gpt55-deployment", "gpt-5.5"))
	p := New("azure-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(
		ctx,
		"system",
		[]api.Message{{Role: "user", Content: "hello"}},
		"corp-gpt55-deployment",
	)

	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty when active context is present", req.PreviousResponseID)
	}
	if len(req.ContextManagement) != 0 {
		t.Fatalf("ContextManagement = %#v, want omitted without previous_response_id", req.ContextManagement)
	}
	input := azureTestResponsesInputItems(t, req)
	if len(input) != 3 {
		t.Fatalf("Input length = %d, want developer plus active context plus history", len(input))
	}
	assertAzureTestInputMessage(t, input[1], "developer", azureTestActiveContextSnapshot)
	assertAzureTestInputMessage(t, input[2], "user", "hello")
}

func TestBuildImageResponsesRequest_IncludesActiveContextFromContext(t *testing.T) {
	ctx := azureTestContextWithActiveContext(azureTestResponsesModelConfig("corp-gpt55-deployment", "gpt-5.5"))
	req := New("azure-key").buildImageResponsesRequest(
		ctx,
		"system",
		[]api.Message{{Role: "user", Content: "before"}},
		"what is this?",
		&api.ImageData{MediaType: "image/png", Base64: "abc123"},
		"corp-gpt55-deployment",
	)

	input := azureTestResponsesInputItems(t, req)
	if len(input) != 4 {
		t.Fatalf("Input length = %d, want developer plus active context plus history plus image", len(input))
	}
	assertAzureTestInputMessage(t, input[1], "developer", azureTestActiveContextSnapshot)
	assertAzureTestInputMessage(t, input[2], "user", "before")
	if input[3].Role != "user" {
		t.Fatalf("Input[3] = %#v, want image user message", input[3])
	}
}

func TestChatWithTools_ActiveContextDoesNotCacheResponseID(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requests = append(requests, body)

		if _, ok := body["previous_response_id"]; ok {
			t.Fatalf("request should omit cached previous_response_id when active context is present: %#v", body)
		}
		if _, ok := body["context_management"]; ok {
			t.Fatalf("request should omit context_management without previous_response_id: %#v", body["context_management"])
		}
		if !azureTestRequestInputContainsContent(body, azureTestActiveContextSnapshot) {
			t.Fatalf("request should include active context in full-history input: %#v", body["input"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_new","output_text":"Fresh","usage":{"input_tokens":4,"output_tokens":2}}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	p := New("azure-key")
	p.SetResponseID("resp_old")

	ctx := azureTestContextWithActiveContext(azureTestResponsesModelConfig("corp-gpt55-pro-deployment", "gpt-5.5-pro"))
	content, err := p.ChatWithTools(ctx, "system", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "Fresh" {
		t.Fatalf("content = %q, want Fresh", content)
	}
	if got := p.GetResponseID(); got != "" {
		t.Fatalf("GetResponseID() = %q, want empty after active-context request", got)
	}
	if len(requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(requests))
	}
}

func TestChatWithImageResponses_ActiveContextDoesNotCacheResponseID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if !azureTestRequestInputContainsContent(body, azureTestActiveContextSnapshot) {
			t.Fatalf("request should include active context in image input: %#v", body["input"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_image","output_text":"Fresh image","usage":{"input_tokens":4,"output_tokens":2}}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	p := New("azure-key")
	p.SetResponseID("resp_old")

	ctx := azureTestContextWithActiveContext(azureTestResponsesModelConfig("corp-gpt55-pro-deployment", "gpt-5.5-pro"))
	content, err := p.chatWithImageResponses(
		ctx,
		"system",
		[]api.Message{{Role: "user", Content: "before"}},
		"what is this?",
		&api.ImageData{MediaType: "image/png", Base64: "abc123"},
		"corp-gpt55-pro-deployment",
	)
	if err != nil {
		t.Fatalf("chatWithImageResponses() error = %v", err)
	}
	if content != "Fresh image" {
		t.Fatalf("content = %q, want Fresh image", content)
	}
	if got := p.GetResponseID(); got != "" {
		t.Fatalf("GetResponseID() = %q, want empty after active-context image request", got)
	}
}

func azureTestResponsesModelConfig(defaultModel, catalogModel string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: defaultModel,
		CatalogModel: catalogModel,
	})
	return cfg
}

func azureTestContextWithActiveContext(cfg *config.Config) context.Context {
	return api.WithActiveContextBlocks(azureTestContext(cfg), azureTestActiveContextBlocks())
}

func azureTestActiveContextBlocks() []api.ActiveContextBlock {
	return []api.ActiveContextBlock{{
		Name:    "provider_history_rehydrated_evidence",
		Content: azureTestActiveContextSnapshot,
	}}
}

func azureTestResponsesInputItems(t *testing.T, req responsesRequest) []api.InputItem {
	t.Helper()
	input, ok := req.Input.([]api.InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []api.InputItem", req.Input)
	}
	return input
}

func assertAzureTestInputMessage(t *testing.T, item api.InputItem, role, content string) {
	t.Helper()
	if item.Type != "message" || item.Role != role || item.Content != content {
		t.Fatalf("Input item = %#v, want %s message with content %q", item, role, content)
	}
}

func azureTestRequestInputContainsContent(raw map[string]any, content string) bool {
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
