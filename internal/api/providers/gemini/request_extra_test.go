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

func TestBuildGeminiRequests_ServiceTier(t *testing.T) {
	flexCfg := config.DefaultConfig()
	flexCfg.Gemini.ServiceTier = config.GeminiServiceTierFlex
	flexCtx := config.WithContext(context.Background(), flexCfg)

	text := buildGeminiTextRequest(flexCtx, "system", []api.Message{{Role: "user", Content: "hello"}}, "gemini-3.5-flash", "", flexCfg)
	if text.ServiceTier != config.GeminiServiceTierFlex {
		t.Fatalf("text ServiceTier = %q, want flex", text.ServiceTier)
	}

	image := buildGeminiMultimodalRequest(
		flexCtx,
		"system",
		nil,
		"describe",
		&api.ImageData{MediaType: "image/png", Base64: "AA=="},
		"gemini-3.5-flash",
		nil,
		false,
		flexCfg,
	)
	if image.ServiceTier != config.GeminiServiceTierFlex {
		t.Fatalf("image ServiceTier = %q, want flex", image.ServiceTier)
	}

	tool := buildGeminiFunctionCallingRequest(
		flexCtx,
		"system",
		[]api.Message{{Role: "user", Content: "hello"}},
		"gemini-3.5-flash",
		"",
		nil,
		nil,
		flexCfg,
	)
	if tool.ServiceTier != config.GeminiServiceTierFlex {
		t.Fatalf("tool ServiceTier = %q, want flex", tool.ServiceTier)
	}

	web := buildGeminiWebSearchRequest(flexCtx, "query", "gemini-3.5-flash", flexCfg)
	if web.ServiceTier != config.GeminiServiceTierFlex {
		t.Fatalf("web ServiceTier = %q, want flex", web.ServiceTier)
	}

	standardCfg := config.DefaultConfig()
	standardCtx := config.WithContext(context.Background(), standardCfg)
	standard := buildGeminiTextRequest(standardCtx, "system", []api.Message{{Role: "user", Content: "hello"}}, "gemini-3.5-flash", "", standardCfg)
	if standard.ServiceTier != "" {
		t.Fatalf("standard ServiceTier = %q, want omitted empty value", standard.ServiceTier)
	}

	invalidCfg := config.DefaultConfig()
	invalidCfg.Gemini.ServiceTier = "turbo"
	invalidCtx := config.WithContext(context.Background(), invalidCfg)
	invalid := buildGeminiTextRequest(invalidCtx, "system", []api.Message{{Role: "user", Content: "hello"}}, "gemini-3.5-flash", "", invalidCfg)
	if invalid.ServiceTier != "" {
		t.Fatalf("invalid ServiceTier = %q, want omitted standard fallback", invalid.ServiceTier)
	}
}

func TestBuildGeminiMultimodalHistoryRequest_KeepsImageAndToolSequence(t *testing.T) {
	ctx := newGeminiRequestContext(true, "high")
	cfg := config.FromContext(ctx)
	req := buildGeminiMultimodalHistoryRequest(
		ctx,
		"system",
		[]api.Message{
			api.NewUserImageMessage("inspect", &api.ImageData{MediaType: "image/png", Base64: "aW1hZ2U="}),
			{
				Role: "assistant",
				ToolCalls: []api.OpenAIToolCall{{
					ID:       "call_1",
					Type:     "function",
					Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"README.md"}`},
				}},
			},
			{Role: "tool", ToolCallID: "call_1", ToolName: "read_file", Content: "README contents"},
		},
		"gemini-3.5-flash",
		[]api.ToolDefinition{{Name: "read_file"}},
		true,
		cfg,
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	contents, ok := body["contents"].([]any)
	if !ok || len(contents) != 3 {
		t.Fatalf("contents = %#v, want image user + model functionCall + user functionResponse", body["contents"])
	}
	imageParts := contents[0].(map[string]any)["parts"].([]any)
	if len(imageParts) != 2 {
		t.Fatalf("image parts = %#v, want inline_data + text", imageParts)
	}
	inline, ok := imageParts[0].(map[string]any)["inline_data"].(map[string]any)
	if !ok || inline["data"] != "aW1hZ2U=" || inline["mime_type"] != "image/png" {
		t.Fatalf("inline_data = %#v, want image/png payload", imageParts[0])
	}
	if imageParts[1].(map[string]any)["text"] != "inspect" {
		t.Fatalf("image text part = %#v, want inspect", imageParts[1])
	}
	functionCall := contents[1].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionCall"].(map[string]any)
	if functionCall["name"] != "read_file" {
		t.Fatalf("functionCall = %#v, want read_file", functionCall)
	}
	functionResponse := contents[2].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)
	if functionResponse["name"] != "read_file" || functionResponse["response"].(map[string]any)["result"] != "README contents" {
		t.Fatalf("functionResponse = %#v, want read_file result", functionResponse)
	}
	if _, ok := body["tools"]; !ok {
		t.Fatalf("tools = nil, want tool definitions on multimodal history request")
	}
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

func TestChatWithFunctionCalling_RequestUsesToolNameForReducedHistoryPlaceholder(t *testing.T) {
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

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "read_file",
		Description: "read file",
		Parameters:  map[string]any{"type": "object"},
	}})
	history := []api.Message{
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
		{
			Role:       "tool",
			ToolCallID: "call_1",
			ToolName:   "read_file",
			Content:    "[omitted old read_file result; evidence: README.md:L1 source=read_file]",
		},
	}

	if _, err := p.chatWithFunctionCalling(newGeminiRequestContext(false, "medium"), "System prompt", history, ""); err != nil {
		t.Fatalf("chatWithFunctionCalling() error = %v", err)
	}

	contents, ok := captured["contents"].([]any)
	if !ok || len(contents) != 2 {
		t.Fatalf("contents = %#v, want 2 entries", captured["contents"])
	}
	toolTurn, ok := contents[1].(map[string]any)
	if !ok || toolTurn["role"] != "user" {
		t.Fatalf("tool turn = %#v, want role=user", contents[1])
	}
	toolParts, ok := toolTurn["parts"].([]any)
	if !ok || len(toolParts) != 1 {
		t.Fatalf("tool parts = %#v, want single functionResponse", toolTurn["parts"])
	}
	functionResponse, ok := toolParts[0].(map[string]any)["functionResponse"].(map[string]any)
	if !ok || functionResponse["name"] != "read_file" {
		t.Fatalf("functionResponse = %#v, want read_file from ToolName metadata", toolParts[0])
	}
	if functionResponse["response"].(map[string]any)["result"] != "[omitted old read_file result; evidence: README.md:L1 source=read_file]" {
		t.Fatalf("function response payload = %#v, want reduced placeholder content preserved", functionResponse["response"])
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
