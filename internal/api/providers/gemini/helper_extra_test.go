package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCreateCachedContent_BuildsRequest(t *testing.T) {
	provider := New("test-api-key")

	var (
		gotRequest map[string]any
		gotHeader  http.Header
	)
	provider.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(req.Body) error = %v", err)
			}
			if err := json.Unmarshal(body, &gotRequest); err != nil {
				t.Fatalf("json.Unmarshal(request) error = %v", err)
			}
			gotHeader = req.Header.Clone()
			respBody := `{"name":"cachedContents/1","model":"models/gemini-1.5-pro-001","createTime":"now","updateTime":"now","expireTime":"later"}`
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(respBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	tools := []api.GeminiToolConfig{{
		FunctionDeclarations: []api.GeminiFunctionDeclaration{{
			Name:        "read_file",
			Description: "Read a file",
		}},
	}}

	resp, err := provider.CreateCachedContent(context.Background(), "gemini-1.5-pro-001", "system prompt", []api.Message{
		{Role: "user", Content: "hello"},
		{
			Role:    "assistant",
			Content: "planning",
			ToolCalls: []api.OpenAIToolCall{{
				ID: "call-1",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"main.go"}`,
				},
				ThoughtSignature: "sig-main",
				ThoughtParts: []map[string]any{{
					"text":              "inspect file",
					"thought":           true,
					"thought_signature": "sig-thought",
				}},
			}},
		},
		{
			Role:       "tool",
			ToolCallID: "call-1",
			ToolName:   "read_file",
			Content:    "[omitted old read_file result; evidence: main.go:L1 source=read_file]",
		},
		{Role: "assistant", Content: "done"},
	}, "300s", tools, &GeminiToolConfigWrapper{
		FunctionCallingConfig: GeminiFunctionCallingConfig{Mode: "ANY"},
	})
	if err != nil {
		t.Fatalf("CreateCachedContent() error = %v", err)
	}
	if resp.Name != "cachedContents/1" {
		t.Fatalf("response.Name = %q, want %q", resp.Name, "cachedContents/1")
	}

	if got := gotHeader.Get("x-goog-api-key"); got != "test-api-key" {
		t.Fatalf("x-goog-api-key = %q, want %q", got, "test-api-key")
	}
	if got := gotHeader.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := gotRequest["model"]; got != "models/gemini-1.5-pro-001" {
		t.Fatalf("request model = %#v, want models/ prefix", got)
	}

	systemInstruction, ok := gotRequest["systemInstruction"].(map[string]any)
	if !ok {
		t.Fatalf("systemInstruction missing or wrong type: %#v", gotRequest["systemInstruction"])
	}
	sysParts := systemInstruction["parts"].([]any)
	if len(sysParts) != 1 || sysParts[0].(map[string]any)["text"] != "system prompt" {
		t.Fatalf("unexpected systemInstruction parts: %#v", sysParts)
	}

	contents := gotRequest["contents"].([]any)
	if len(contents) != 4 {
		t.Fatalf("len(contents) = %d, want 4", len(contents))
	}

	first := contents[0].(map[string]any)
	if first["role"] != "user" {
		t.Fatalf("first role = %#v, want user", first["role"])
	}
	second := contents[1].(map[string]any)
	if second["role"] != "model" {
		t.Fatalf("assistant tool call content role = %#v, want model", second["role"])
	}
	secondParts := second["parts"].([]any)
	if len(secondParts) != 3 {
		t.Fatalf("len(second parts) = %d, want 3", len(secondParts))
	}
	if secondParts[0].(map[string]any)["text"] != "planning" {
		t.Fatalf("first assistant part = %#v, want text part", secondParts[0])
	}
	if secondParts[1].(map[string]any)["thought"] != true {
		t.Fatalf("thought part missing thought=true: %#v", secondParts[1])
	}
	functionCall := secondParts[2].(map[string]any)["functionCall"].(map[string]any)
	if functionCall["name"] != "read_file" {
		t.Fatalf("functionCall.name = %#v, want read_file", functionCall["name"])
	}
	if functionCall["args"].(map[string]any)["path"] != "main.go" {
		t.Fatalf("functionCall.args.path = %#v, want main.go", functionCall["args"])
	}

	third := contents[2].(map[string]any)
	functionResponse := third["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)
	if functionResponse["name"] != "read_file" {
		t.Fatalf("functionResponse.name = %#v, want read_file", functionResponse["name"])
	}
	if functionResponse["response"].(map[string]any)["result"] != "[omitted old read_file result; evidence: main.go:L1 source=read_file]" {
		t.Fatalf("unexpected function response payload: %#v", functionResponse["response"])
	}

	toolConfig := gotRequest["tool_config"].(map[string]any)
	if toolConfig["function_calling_config"].(map[string]any)["mode"] != "ANY" {
		t.Fatalf("tool config mode = %#v, want ANY", toolConfig)
	}
	if len(gotRequest["tools"].([]any)) != 1 {
		t.Fatalf("expected one tool definition, got %#v", gotRequest["tools"])
	}
}

func TestCreateCachedContent_APIError(t *testing.T) {
	provider := New("test-api-key")
	provider.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("bad request")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := provider.CreateCachedContent(context.Background(), "gemini-1.5-pro-001", "", []api.Message{{Role: "user", Content: "hello"}}, "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "API error (status 400)") {
		t.Fatalf("CreateCachedContent() error = %v, want API error", err)
	}
}

func TestGeminiHelperFunctions(t *testing.T) {
	t.Run("isCacheExpiredError", func(t *testing.T) {
		if !isCacheExpiredError(http.StatusNotFound, []byte(`{"error":"cachedContent NOT_FOUND"}`)) {
			t.Fatal("isCacheExpiredError() = false, want true for NOT_FOUND cache error")
		}
		if !isCacheExpiredError(http.StatusBadRequest, []byte("cachedContent not found")) {
			t.Fatal("isCacheExpiredError() = false, want true for 400 cache error")
		}
		if isCacheExpiredError(http.StatusInternalServerError, []byte("cachedContent not found")) {
			t.Fatal("isCacheExpiredError() = true, want false for unrelated status")
		}
	})

	t.Run("getCacheTTL", func(t *testing.T) {
		t.Setenv("GEMINI_CACHE_TTL", "900")
		if got := getCacheTTL(); got != 900 {
			t.Fatalf("getCacheTTL() = %d, want 900", got)
		}

		t.Setenv("GEMINI_CACHE_TTL", "invalid")
		if got := getCacheTTL(); got != defaultCacheTTL {
			t.Fatalf("getCacheTTL() with invalid env = %d, want %d", got, defaultCacheTTL)
		}
	})

	t.Run("levelToThinkingLevel", func(t *testing.T) {
		tests := []struct {
			level string
			model string
			want  string
		}{
			{level: "low", model: "gemini-3-pro", want: "low"},
			{level: "medium", model: "gemini-3-flash", want: "medium"},
			{level: "medium", model: "gemini-3.1-pro", want: "medium"},
			{level: "medium", model: "gemini-3-pro", want: "low"},
			{level: "xhigh", model: "gemini-3-pro", want: "high"},
			{level: "unknown", model: "gemini-3-pro", want: "low"},
		}
		for _, tt := range tests {
			if got := levelToThinkingLevel(tt.level, tt.model); got != tt.want {
				t.Fatalf("levelToThinkingLevel(%q, %q) = %q, want %q", tt.level, tt.model, got, tt.want)
			}
		}
	})
}
