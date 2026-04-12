package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newGeminiRequestContext(thinking bool, level string) context.Context {
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = level
	return config.WithContext(context.Background(), cfg)
}

func TestChatWithFunctionCalling_RequestTransformsHistoryAndThinkingConfig(t *testing.T) {
	var captured map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{
				{Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "done"}}}},
			},
		}
		jsonBytes, _ := json.Marshal(resp)
		_, _ = w.Write([]byte("data: " + string(jsonBytes) + "\n\n"))
	})
	t.Setenv("GEMINI_API_URL", server.URL)
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")
	t.Setenv("GEMINI_FC_MODE", "VALIDATED")

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "custom_lookup",
		Description: "custom lookup",
		Parameters:  map[string]any{"type": "object"},
	}})
	history := []api.Message{
		{Role: "user", Content: "Read main.go"},
		{
			Role:    "assistant",
			Content: "I will inspect that.",
			ToolCalls: []api.OpenAIToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "custom_lookup",
					Arguments: `{"path":"main.go"}`,
				},
				ThoughtSignature: "sig-1",
				ThoughtParts: []map[string]any{
					{"text": "checking", "thought": true, "thought_signature": "sig-1"},
				},
			}},
		},
		{
			Role:       "tool",
			ToolCallID: "call_1",
			Content:    "[Tool Result for custom_lookup] found it",
		},
	}

	_, err := p.chatWithFunctionCalling(newGeminiRequestContext(true, "high"), "System prompt", history, "")
	if err != nil {
		t.Fatalf("chatWithFunctionCalling() error = %v", err)
	}

	toolConfig, ok := captured["tool_config"].(map[string]any)
	if !ok {
		t.Fatalf("tool_config = %#v, want map", captured["tool_config"])
	}
	fcConfig, ok := toolConfig["function_calling_config"].(map[string]any)
	if !ok || fcConfig["mode"] != "VALIDATED" {
		t.Fatalf("function_calling_config = %#v, want mode VALIDATED", toolConfig["function_calling_config"])
	}

	systemInstruction, ok := captured["system_instruction"].(map[string]any)
	if !ok || systemInstruction["parts"] == nil {
		t.Fatalf("system_instruction = %#v, want parts", captured["system_instruction"])
	}

	generationConfig, ok := captured["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig = %#v, want map", captured["generationConfig"])
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok || thinkingConfig["thinkingLevel"] != "high" {
		t.Fatalf("thinkingConfig = %#v, want thinkingLevel=high", generationConfig["thinkingConfig"])
	}

	contents, ok := captured["contents"].([]any)
	if !ok || len(contents) != 3 {
		t.Fatalf("contents = %#v, want 3 entries", captured["contents"])
	}

	modelTurn, ok := contents[1].(map[string]any)
	if !ok || modelTurn["role"] != "model" {
		t.Fatalf("model turn = %#v, want role=model", contents[1])
	}
	modelParts, ok := modelTurn["parts"].([]any)
	if !ok || len(modelParts) < 3 {
		t.Fatalf("model parts = %#v, want text+thought+functionCall", modelTurn["parts"])
	}
	thoughtPart, ok := modelParts[1].(map[string]any)
	if !ok || thoughtPart["thought"] != true || thoughtPart["thoughtSignature"] != "sig-1" {
		t.Fatalf("thought part = %#v, want thought + signature", modelParts[1])
	}
	functionPart, ok := modelParts[2].(map[string]any)
	if !ok {
		t.Fatalf("function part = %#v, want map", modelParts[2])
	}
	functionCall, ok := functionPart["functionCall"].(map[string]any)
	if !ok || functionCall["name"] != "custom_lookup" {
		t.Fatalf("functionCall = %#v, want custom_lookup", functionPart["functionCall"])
	}

	toolTurn, ok := contents[2].(map[string]any)
	if !ok || toolTurn["role"] != "user" {
		t.Fatalf("tool turn = %#v, want role=user", contents[2])
	}
	toolParts, ok := toolTurn["parts"].([]any)
	if !ok || len(toolParts) != 1 {
		t.Fatalf("tool parts = %#v, want single functionResponse", toolTurn["parts"])
	}
	functionResponse, ok := toolParts[0].(map[string]any)
	if !ok {
		t.Fatalf("functionResponse part = %#v, want map", toolParts[0])
	}
	responseBody, ok := functionResponse["functionResponse"].(map[string]any)
	if !ok || responseBody["name"] != "custom_lookup" {
		t.Fatalf("functionResponse = %#v, want custom_lookup", functionResponse["functionResponse"])
	}
}

func TestChatWithTextMode_RequestUsesSystemInstructionAndThinkingBudget(t *testing.T) {
	var captured GeminiRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{
				{Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "text-mode"}}}},
			},
		}
		jsonBytes, _ := json.Marshal(resp)
		_, _ = w.Write([]byte("data: " + string(jsonBytes) + "\n\n"))
	})
	t.Setenv("GEMINI_API_URL", server.URL)
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")

	p := New("test-key")
	ctx := newGeminiRequestContext(true, "high")
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
	}

	got, err := p.chatWithTextMode(ctx, "System prompt", history, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("chatWithTextMode() error = %v", err)
	}
	if got != "text-mode" {
		t.Fatalf("chatWithTextMode() = %q, want %q", got, "text-mode")
	}
	if captured.SystemInstruction == nil || len(captured.SystemInstruction.Parts) != 1 || captured.SystemInstruction.Parts[0].Text != "System prompt" {
		t.Fatalf("SystemInstruction = %+v, want System prompt", captured.SystemInstruction)
	}
	if len(captured.Contents) != 2 || captured.Contents[0].Role != "user" || captured.Contents[1].Role != "model" {
		t.Fatalf("Contents = %+v, want user/model roles", captured.Contents)
	}
	if captured.GenerationConfig == nil || captured.GenerationConfig.ThinkingConfig == nil {
		t.Fatalf("GenerationConfig = %+v, want thinking config", captured.GenerationConfig)
	}
	if captured.GenerationConfig.ThinkingConfig.ThinkingBudget != api.LevelToBudgetTokens("high") {
		t.Fatalf("ThinkingBudget = %d, want %d", captured.GenerationConfig.ThinkingConfig.ThinkingBudget, api.LevelToBudgetTokens("high"))
	}
}

func TestWebSearch_EmptyCandidatesAndHTTPErrors(t *testing.T) {
	t.Run("empty candidates returns no results", func(t *testing.T) {
		server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"candidates":[]}`))
		})
		t.Setenv("GEMINI_API_URL", server.URL)

		got, err := New("test-key").webSearch(newGeminiRequestContext(false, "medium"), "coverage report", "gemini-3.1-pro-preview-customtools")
		if err != nil {
			t.Fatalf("webSearch() error = %v", err)
		}
		if got != "No results found." {
			t.Fatalf("webSearch() = %q, want %q", got, "No results found.")
		}
	})

	t.Run("http error is returned", func(t *testing.T) {
		server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("backend unavailable"))
		})
		t.Setenv("GEMINI_API_URL", server.URL)

		_, err := New("test-key").webSearch(newGeminiRequestContext(false, "medium"), "coverage report", "gemini-3.1-pro-preview-customtools")
		if err == nil {
			t.Fatal("webSearch() should return error for non-200 response")
		}
		if !strings.Contains(err.Error(), "backend unavailable") {
			t.Fatalf("webSearch() error = %q, want backend unavailable", err.Error())
		}
	})
}
