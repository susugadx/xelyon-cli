package groq

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
)

func TestProvider_IsFunctionCallingEnabled(t *testing.T) {
	originalEnv := os.Getenv("GROQ_FUNCTION_CALLING")
	defer os.Setenv("GROQ_FUNCTION_CALLING", originalEnv)

	os.Setenv("GROQ_FUNCTION_CALLING", "0")
	if New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = true, want false when GROQ_FUNCTION_CALLING=0")
	}

	os.Unsetenv("GROQ_FUNCTION_CALLING")
	if !New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = false, want true by default")
	}
}

func TestProvider_SettersAffectRequestAndUsageCallback(t *testing.T) {
	originalEnv := os.Getenv("GROQ_FUNCTION_CALLING")
	defer os.Setenv("GROQ_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("GROQ_FUNCTION_CALLING")

	var requestBody openaicompat.ChatCompletionsRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		streamingHandler([]string{
			`{"choices":[{"delta":{"content":"done"}}]}`,
			`{"choices":[{"delta":{}}],"usage":{"prompt_tokens":12,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":3}}}`,
		})(w, r)
	})

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})
	p.SetToolChoice("read_file")

	result, err := p.ChatWithTools(context.Background(), "System", []api.Message{{Role: "user", Content: "hello"}}, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "done" {
		t.Fatalf("ChatWithTools() = %q, want %q", result, "done")
	}

	toolChoice, ok := requestBody.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("ToolChoice type = %T, want map[string]any", requestBody.ToolChoice)
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != "read_file" {
		t.Fatalf("ToolChoice function = %#v, want read_file", toolChoice["function"])
	}
	if gotUsage.InputTokens != 12 || gotUsage.OutputTokens != 4 || gotUsage.CachedInputTokens != 3 {
		t.Fatalf("usage callback = %#v, want input=12 output=4 cached=3", gotUsage)
	}

	p.ClearToolChoice()
	if p.ToolChoice() != nil {
		t.Fatal("ClearToolChoice() should clear toolChoice")
	}
}

func TestProvider_ChatWithTools_ToolUseDisabledOmitsToolFields(t *testing.T) {
	t.Setenv("GROQ_FUNCTION_CALLING", "1")

	var requestBody map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		streamingHandler([]string{`{"choices":[{"delta":{"content":"done"}}]}`})(w, r)
	})
	t.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	p.SetToolChoice("read_file")

	ctx := api.WithToolUseDisabled(context.Background())
	if _, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hello"}}, "llama3-70b-8192"); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if _, ok := requestBody["tools"]; ok {
		t.Fatalf("tools should be omitted when tool use is disabled: %#v", requestBody["tools"])
	}
	if _, ok := requestBody["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted when tool use is disabled: %#v", requestBody["tool_choice"])
	}
}
