package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newGeminiOrchestrationContext(idleSeconds, thinkingSeconds int) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	cfg := config.DefaultConfig()
	cfg.Streaming.IdleTimeoutSeconds = idleSeconds
	cfg.Streaming.ThinkingTimeoutSeconds = thinkingSeconds

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(nil, out, errOut))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return config.WithContext(ctx, cfg), out, errOut
}

func TestGeminiProviderFactory_IsRegisteredAndBuildsProvider(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	if !api.IsRegisteredProvider("gemini") {
		t.Fatal("gemini provider should be registered")
	}

	provider, err := api.NewProvider("gemini")
	if err != nil {
		t.Fatalf("api.NewProvider(gemini) error = %v", err)
	}
	if provider.Name() != "Gemini" {
		t.Fatalf("provider.Name() = %q, want %q", provider.Name(), "Gemini")
	}
}

func TestGetThinkingConfigForModel_Gemini3ProWithoutThinkingUsesLowLevel(t *testing.T) {
	ctx := newGeminiRequestContext(false, "high")
	cfg := config.FromContext(ctx)

	got := getThinkingConfigForModel(ctx, "gemini-3.1-pro-preview", cfg)
	if got == nil || got.ThinkingConfig == nil {
		t.Fatalf("getThinkingConfigForModel() = %+v, want thinking config", got)
	}
	if got.ThinkingConfig.ThinkingLevel != "low" {
		t.Fatalf("ThinkingLevel = %q, want %q", got.ThinkingConfig.ThinkingLevel, "low")
	}
	if got.Temperature == nil || *got.Temperature != 1.0 {
		t.Fatalf("Temperature = %+v, want 1.0", got.Temperature)
	}
}

func TestGetThinkingSpinnerMessage_Gemini25ImageWithThinking(t *testing.T) {
	ctx := newGeminiRequestContext(true, "medium")
	if got := getThinkingSpinnerMessage(ctx, "gemini-2.5-flash", true); got != "Deep thinking (image)" {
		t.Fatalf("getThinkingSpinnerMessage() = %q, want %q", got, "Deep thinking (image)")
	}
}

func TestChatWithTools_FCErrorRetryThenSuccess_Debug(t *testing.T) {
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")
	t.Setenv("GEMINI_FUNCTION_CALLING", "1")
	t.Setenv("XELYON_DEBUG_GEMINI", "1")

	p := New("test-key")
	var attempts atomic.Int32
	p.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch attempts.Add(1) {
			case 1:
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			case 2:
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(geminiSSEPayload(t, GeminiFunctionResponse{
						Candidates: []GeminiFunctionCandidate{{
							Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "retry ok"}}},
						}},
					}))),
					Header: make(http.Header),
				}, nil
			default:
				t.Fatalf("unexpected request count: %d", attempts.Load())
				return nil, nil
			}
		}),
	}

	ctx, _, errOut := newGeminiOrchestrationContext(1, 1)
	got, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "retry ok" {
		t.Fatalf("ChatWithTools() = %q, want %q", got, "retry ok")
	}
	if attempts.Load() != 2 {
		t.Fatalf("request count = %d, want 2", attempts.Load())
	}

	logs := errOut.String()
	if !strings.Contains(logs, "[DEBUG Gemini] Mode: Function Calling") {
		t.Fatalf("errOut = %q, want function calling debug log", logs)
	}
	if !strings.Contains(logs, "FC error, retrying FC mode") {
		t.Fatalf("errOut = %q, want FC retry warning", logs)
	}
	if !strings.Contains(logs, "[DEBUG Gemini] FC error detail:") {
		t.Fatalf("errOut = %q, want FC error detail", logs)
	}
}

func TestChatWithTools_FCErrorExceedsMaxRetries(t *testing.T) {
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")
	t.Setenv("GEMINI_FUNCTION_CALLING", "1")

	p := New("test-key")
	p.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx, _, _ := newGeminiOrchestrationContext(1, 1)
	ctx = context.WithValue(ctx, fcErrorRetryKey, maxFCErrorRetries)

	_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err == nil {
		t.Fatal("ChatWithTools() should return error when FC retry budget is exhausted")
	}
	if !strings.Contains(err.Error(), "FC mode failed after") {
		t.Fatalf("ChatWithTools() error = %q, want FC mode failed message", err.Error())
	}
}

func TestChatWithTools_IdleTimeoutRetryThenSuccess(t *testing.T) {
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")
	t.Setenv("GEMINI_FUNCTION_CALLING", "1")

	p := New("test-key")
	var attempts atomic.Int32
	p.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch attempts.Add(1) {
			case 1:
				pr, pw := io.Pipe()
				go func() {
					_, _ = fmt.Fprint(pw, ": keepalive\n\n")
					time.Sleep(1200 * time.Millisecond)
					_ = pw.Close()
				}()
				return &http.Response{StatusCode: http.StatusOK, Body: pr, Header: make(http.Header)}, nil
			case 2:
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(geminiSSEPayload(t, GeminiFunctionResponse{
						Candidates: []GeminiFunctionCandidate{{
							Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "idle retry ok"}}},
						}},
					}))),
					Header: make(http.Header),
				}, nil
			default:
				t.Fatalf("unexpected request count: %d", attempts.Load())
				return nil, nil
			}
		}),
	}

	ctx, _, errOut := newGeminiOrchestrationContext(1, 3)
	got, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "idle retry ok" {
		t.Fatalf("ChatWithTools() = %q, want %q", got, "idle retry ok")
	}
	if attempts.Load() != 2 {
		t.Fatalf("request count = %d, want 2", attempts.Load())
	}
	if !strings.Contains(errOut.String(), "Transport idle timeout, retrying FC mode") {
		t.Fatalf("errOut = %q, want idle timeout retry warning", errOut.String())
	}
}

func TestChatWithTools_IdleTimeoutExceedsMaxRetries(t *testing.T) {
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")
	t.Setenv("GEMINI_FUNCTION_CALLING", "1")

	p := New("test-key")
	p.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			pr, pw := io.Pipe()
			go func() {
				_, _ = fmt.Fprint(pw, ": keepalive\n\n")
				time.Sleep(1200 * time.Millisecond)
				_ = pw.Close()
			}()
			return &http.Response{StatusCode: http.StatusOK, Body: pr, Header: make(http.Header)}, nil
		}),
	}

	ctx, _, _ := newGeminiOrchestrationContext(1, 3)
	ctx = context.WithValue(ctx, idleTimeoutRetryKey, maxIdleTimeoutRetries)

	_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err == nil {
		t.Fatal("ChatWithTools() should return error when idle-timeout retry budget is exhausted")
	}
	if !strings.Contains(err.Error(), "transport idle timeout: exceeded max retries") {
		t.Fatalf("ChatWithTools() error = %q, want exceeded max retries", err.Error())
	}
}

func TestChatWithTools_TextModeDebugLogging(t *testing.T) {
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")
	t.Setenv("GEMINI_FUNCTION_CALLING", "0")
	t.Setenv("XELYON_DEBUG_GEMINI", "1")

	p := New("test-key")
	p.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(geminiSSEPayload(t, GeminiFunctionResponse{
					Candidates: []GeminiFunctionCandidate{{
						Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "text mode ok"}}},
					}},
				}))),
				Header: make(http.Header),
			}, nil
		}),
	}

	ctx, _, errOut := newGeminiOrchestrationContext(1, 1)
	got, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "text mode ok" {
		t.Fatalf("ChatWithTools() = %q, want %q", got, "text mode ok")
	}
	if !strings.Contains(errOut.String(), "Mode: TextMode (GEMINI_FUNCTION_CALLING=0)") {
		t.Fatalf("errOut = %q, want text mode debug log", errOut.String())
	}
}

func TestChatWithImage_RequestIncludesHistoryAndToolConfigWhenFCEnabled(t *testing.T) {
	var captured map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, geminiSSEPayload(t, GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{
				Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "image ok"}}},
			}},
		}))
	})
	t.Setenv("GEMINI_API_URL", server.URL)
	t.Setenv("GEMINI_FUNCTION_CALLING", "1")

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "custom_lookup",
		Description: "custom lookup",
		Parameters:  map[string]any{"type": "object"},
	}})

	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}
	history := []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "previous answer"},
	}

	got, err := p.ChatWithImage(newGeminiRequestContext(true, "high"), "System prompt", history, "describe this", image, "gemini-3.1-pro-preview")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if got != "image ok" {
		t.Fatalf("ChatWithImage() = %q, want %q", got, "image ok")
	}

	if _, ok := captured["tools"].([]any); !ok {
		t.Fatalf("tools = %#v, want tool definitions", captured["tools"])
	}
	toolCfg, ok := captured["tool_config"].(map[string]any)
	if !ok {
		t.Fatalf("tool_config = %#v, want map", captured["tool_config"])
	}
	if toolCfg["function_calling_config"].(map[string]any)["mode"] != "AUTO" {
		t.Fatalf("tool_config = %#v, want AUTO mode", toolCfg)
	}

	contents, ok := captured["contents"].([]any)
	if !ok || len(contents) != 3 {
		t.Fatalf("contents = %#v, want 3 entries", captured["contents"])
	}
	if contents[1].(map[string]any)["role"] != "model" {
		t.Fatalf("assistant history role = %#v, want model", contents[1].(map[string]any)["role"])
	}

	last := contents[2].(map[string]any)
	parts, ok := last["parts"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("multimodal parts = %#v, want image + text", last["parts"])
	}
	inlineData, ok := parts[0].(map[string]any)["inline_data"].(map[string]any)
	if !ok || inlineData["mime_type"] != "image/png" || inlineData["data"] != "dGVzdA==" {
		t.Fatalf("inline_data = %#v, want image payload", parts[0])
	}
	if parts[1].(map[string]any)["text"] != "describe this" {
		t.Fatalf("user text part = %#v, want describe this", parts[1])
	}
}

func TestChatWithFunctionCalling_CacheExpiredInvalidatesAndRetries(t *testing.T) {
	t.Setenv("XELYON_DEBUG_GEMINI", "1")

	model := "gemini-3.1-pro-preview-customtools"
	systemPrompt := strings.Repeat("context block ", 5000)
	p := New("test-key")
	p.cacheMap = map[string]*cacheEntry{
		model: {
			name:       "cachedContents/stale",
			model:      model,
			expireTime: time.Now().Add(time.Hour),
		},
	}

	var streamRequests []map[string]any
	p.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(req.Body) error = %v", err)
			}

			switch {
			case req.Method == http.MethodPost && req.URL.Path == "/v1beta/cachedContents":
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"name":"cachedContents/fresh","model":"models/gemini-3.1-pro-preview-customtools","createTime":"now","updateTime":"now","expireTime":"later"}`)),
					Header:     make(http.Header),
				}, nil

			case req.Method == http.MethodPost && strings.Contains(req.URL.Path, ":streamGenerateContent"):
				var payload map[string]any
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("json.Unmarshal(stream request) error = %v", err)
				}
				streamRequests = append(streamRequests, payload)
				if len(streamRequests) == 1 {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(strings.NewReader(`cachedContent NOT_FOUND`)),
						Header:     make(http.Header),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(geminiSSEPayload(t, GeminiFunctionResponse{
						Candidates: []GeminiFunctionCandidate{{
							Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "after cache refresh"}}},
						}},
					}))),
					Header: make(http.Header),
				}, nil

			default:
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				return nil, nil
			}
		}),
	}

	ctx, _, errOut := newGeminiOrchestrationContext(1, 1)
	got, err := p.chatWithFunctionCalling(ctx, systemPrompt, []api.Message{{Role: "user", Content: "latest question"}}, model)
	if err != nil {
		t.Fatalf("chatWithFunctionCalling() error = %v", err)
	}
	if got != "after cache refresh" {
		t.Fatalf("chatWithFunctionCalling() = %q, want %q", got, "after cache refresh")
	}
	if len(streamRequests) != 2 {
		t.Fatalf("stream request count = %d, want 2", len(streamRequests))
	}
	if streamRequests[0]["cachedContent"] != "cachedContents/stale" {
		t.Fatalf("first cachedContent = %#v, want stale cache", streamRequests[0]["cachedContent"])
	}
	if streamRequests[1]["cachedContent"] != "cachedContents/fresh" {
		t.Fatalf("second cachedContent = %#v, want fresh cache", streamRequests[1]["cachedContent"])
	}
	if p.cacheMap[model] == nil || p.cacheMap[model].name != "cachedContents/fresh" {
		t.Fatalf("cacheMap[%q] = %+v, want fresh cache entry", model, p.cacheMap[model])
	}
	if !strings.Contains(errOut.String(), "Cache expired, invalidating and retrying") {
		t.Fatalf("errOut = %q, want cache expired debug log", errOut.String())
	}
}

func TestChatWithFunctionCalling_CacheRetryFailureIsReturned(t *testing.T) {
	model := "gemini-3.1-pro-preview-customtools"
	systemPrompt := strings.Repeat("context block ", 5000)
	p := New("test-key")
	p.cacheMap = map[string]*cacheEntry{
		model: {
			name:       "cachedContents/stale",
			model:      model,
			expireTime: time.Now().Add(time.Hour),
		},
	}
	p.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`cachedContent NOT_FOUND`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx, _, _ := newGeminiOrchestrationContext(1, 1)
	ctx = context.WithValue(ctx, cacheRetryKey, true)

	_, err := p.chatWithFunctionCalling(ctx, systemPrompt, []api.Message{{Role: "user", Content: "latest question"}}, model)
	if err == nil {
		t.Fatal("chatWithFunctionCalling() should return cache retry failure")
	}
	if !strings.Contains(err.Error(), "cache retry failed") {
		t.Fatalf("chatWithFunctionCalling() error = %q, want cache retry failed", err.Error())
	}
}
