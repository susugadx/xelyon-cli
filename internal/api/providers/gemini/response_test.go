package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ===== extractCodeBlockToolJSON unit tests =====

func TestExtractCodeBlockToolJSON_Basic(t *testing.T) {
	text := "Some text.\n\n```json\n{\"tool\":\"read_file\",\"args\":{\"path\":\"/file\"}}\n```\n"
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 1 {
		t.Fatalf("expected 1 tool JSON, got %d", len(toolJSONs))
	}
	if !strings.Contains(toolJSONs[0], "read_file") {
		t.Errorf("toolJSON should contain read_file, got %q", toolJSONs[0])
	}
	if strings.Contains(remaining, "```") {
		t.Errorf("remaining should not contain code block markers, got %q", remaining)
	}
	if !strings.Contains(remaining, "Some text.") {
		t.Errorf("remaining should contain surrounding text, got %q", remaining)
	}
}

func TestExtractCodeBlockToolJSON_NotToolJSON(t *testing.T) {
	text := "```go\nfunc main() {}\n```\n"
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 0 {
		t.Errorf("expected 0 tool JSONs for non-tool code block, got %d", len(toolJSONs))
	}
	if remaining != text {
		t.Errorf("remaining should be unchanged, got %q", remaining)
	}
}

func TestExtractCodeBlockToolJSON_NoCodeBlock(t *testing.T) {
	text := "Just plain text."
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 0 {
		t.Errorf("expected 0 tool JSONs, got %d", len(toolJSONs))
	}
	if remaining != text {
		t.Errorf("remaining should be unchanged, got %q", remaining)
	}
}

func TestExtractCodeBlockToolJSON_Multiple(t *testing.T) {
	text := "First\n```json\n{\"tool\":\"read_file\",\"args\":{}}\n```\nMiddle\n```json\n{\"tool\":\"bash\",\"args\":{}}\n```\nLast"
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 2 {
		t.Fatalf("expected 2 tool JSONs, got %d", len(toolJSONs))
	}
	if !strings.Contains(remaining, "First") || !strings.Contains(remaining, "Last") {
		t.Errorf("remaining should contain surrounding text, got %q", remaining)
	}
}

func TestIsToolJSONPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`{"tool":"read_file","args":{}}`, true},
		{`{ "tool": "bash", "args": {} }`, true},
		{`{"id":"call_1","tool":"read_file"}`, false},
		{`Just text`, false},
		{``, false},
	}
	for _, tt := range tests {
		got := isToolJSONPrefix(tt.input)
		if got != tt.want {
			t.Errorf("isToolJSONPrefix(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ===== updateToolJSONDepth unit tests =====

func TestUpdateToolJSONDepth_SimpleObject(t *testing.T) {
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"tool":"read_file"}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (balanced braces)", depth)
	}
	if inStr {
		t.Error("inStr should be false after balanced JSON")
	}
}

func TestUpdateToolJSONDepth_NestedObject(t *testing.T) {
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"tool":"bash","args":{"command":"ls"}}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (balanced nested braces)", depth)
	}
}

func TestUpdateToolJSONDepth_PartialFirstChunk(t *testing.T) {
	// チャンク1: `{"tool":"read` → depth=1 (開いたまま)
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"tool":"read`, &depth, &inStr)
	if depth != 1 {
		t.Errorf("depth = %d, want 1 (unclosed brace)", depth)
	}
	if !inStr {
		t.Error("inStr should be true (inside unclosed string)")
	}
}

func TestUpdateToolJSONDepth_PartialSecondChunk(t *testing.T) {
	// チャンク1 → チャンク2 で閉じる
	depth := 1
	inStr := true
	updateToolJSONDepth(`_files","args":{"path":"/main.go"}}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (closed by second chunk)", depth)
	}
	if inStr {
		t.Error("inStr should be false after balanced JSON")
	}
}

func TestUpdateToolJSONDepth_BracesInString(t *testing.T) {
	// 文字列リテラル内の {} は深度に影響しない
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"content":"value with { and } inside"}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (braces in string should be ignored)", depth)
	}
}

func TestUpdateToolJSONDepth_EscapedQuotes(t *testing.T) {
	// エスケープされた引用符は文字列の終端にならない
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"content":"say \"hello\" world"}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0", depth)
	}
	if inStr {
		t.Error("inStr should be false after balanced JSON with escaped quotes")
	}
}

func TestUpdateToolJSONDepth_ThoughtSignatureChunk(t *testing.T) {
	// thought_signature を含む巨大チャンク
	depth := 1
	inStr := true
	sig := strings.Repeat("A", 10000) // 巨大な署名
	chunk := fmt.Sprintf(`_files","args":{"path":"/file"},"thought_signature":"%s"}`, sig)
	updateToolJSONDepth(chunk, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (should close after signature)", depth)
	}
}

func TestUpdateToolJSONDepth_EmptyString(t *testing.T) {
	depth := 5
	inStr := true
	updateToolJSONDepth("", &depth, &inStr)
	if depth != 5 {
		t.Errorf("depth = %d, want 5 (unchanged for empty string)", depth)
	}
	if !inStr {
		t.Error("inStr should remain true for empty string")
	}
}

// ===== ThinkingTimeout config tests =====

func TestErrThinkingTimeout_Error(t *testing.T) {
	err := &ErrThinkingTimeout{Message: "test timeout message"}
	if err.Error() != "test timeout message" {
		t.Errorf("ErrThinkingTimeout.Error() = %q, want %q", err.Error(), "test timeout message")
	}
}

func TestErrThinkingTimeout_Is(t *testing.T) {
	// errors.As で ErrThinkingTimeout を識別できることを確認
	var target *ErrThinkingTimeout
	err := fmt.Errorf("wrapped: %w", &ErrThinkingTimeout{Message: "inner"})

	// errors パッケージのインポートなしでも、ErrThinkingTimeout 自体のキャスト確認
	if !isThinkingTimeoutError(err) {
		t.Error("isThinkingTimeoutError should return true for wrapped ErrThinkingTimeout")
	}
	_ = target

	// 通常のエラーは false
	normalErr := fmt.Errorf("some other error")
	if isThinkingTimeoutError(normalErr) {
		t.Error("isThinkingTimeoutError should return false for non-ErrThinkingTimeout")
	}
}

func TestThinkingTimeoutDefaults(t *testing.T) {
	// config のデフォルト値が正しいことを確認
	cfg := config.DefaultConfig()
	if cfg.Streaming.ThinkingTimeoutSeconds != 120 {
		t.Errorf("ThinkingTimeoutSeconds default = %d, want 120", cfg.Streaming.ThinkingTimeoutSeconds)
	}
	if cfg.Streaming.IdleTimeoutSeconds != 30 {
		t.Errorf("IdleTimeoutSeconds default = %d, want 30", cfg.Streaming.IdleTimeoutSeconds)
	}
}

type timedSSEChunk struct {
	delay   time.Duration
	payload string
}

func newGeminiSSETestContext(idleSeconds, thinkingSeconds int) context.Context {
	cfg := config.DefaultConfig()
	cfg.Streaming.IdleTimeoutSeconds = idleSeconds
	cfg.Streaming.ThinkingTimeoutSeconds = thinkingSeconds

	runtime := ui.NewRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	ctx := ui.WithRuntime(context.Background(), runtime)
	return config.WithContext(ctx, cfg)
}

func newTimedSSEResponse(t *testing.T, chunks []timedSSEChunk, finalDelay time.Duration) *http.Response {
	t.Helper()

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for _, chunk := range chunks {
			time.Sleep(chunk.delay)
			if _, err := fmt.Fprintf(pw, "data: %s\n\n", chunk.payload); err != nil {
				return
			}
		}
		if finalDelay > 0 {
			time.Sleep(finalDelay)
		}
	}()

	return &http.Response{
		StatusCode: 200,
		Body:       pr,
	}
}

func mustSSEPayload(t *testing.T, parts ...GeminiFunctionPart) string {
	t.Helper()

	chunk := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: parts,
				},
			},
		},
	}
	b, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("failed to marshal SSE payload: %v", err)
	}
	return string(b)
}

func TestHandleSSEResponse_ThoughtOnlyChunksDoNotTriggerTransportIdle(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(1, 3)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-1"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-2"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-3"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Text: "final answer"})},
	}, 0)
	defer resp.Body.Close()

	got, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if got != "final answer" {
		t.Fatalf("handleSSEResponse() = %q, want %q", got, "final answer")
	}
}

func TestHandleSSEResponse_ThoughtProgressResetsThinkingTimeout(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(3, 1)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-1"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-2"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-3"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Text: "final after thinking"})},
	}, 0)
	defer resp.Body.Close()

	got, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if got != "final after thinking" {
		t.Fatalf("handleSSEResponse() = %q, want %q", got, "final after thinking")
	}
}

func TestHandleSSEResponse_ThoughtOnlyChunksBecomeThinkingTimeout(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(2, 1)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-1"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking-2"})},
	}, 1500*time.Millisecond)
	defer resp.Body.Close()

	_, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err == nil {
		t.Fatal("handleSSEResponse() should return thinking timeout")
	}

	var thinkingErr *ErrThinkingTimeout
	if !errors.As(err, &thinkingErr) {
		t.Fatalf("error should be ErrThinkingTimeout, got %T: %v", err, err)
	}
	if !strings.Contains(thinkingErr.Error(), "no Gemini progress") {
		t.Fatalf("thinking timeout message should describe Gemini progress starvation, got %q", thinkingErr.Error())
	}
}

func TestHandleSSEResponse_ThinkingModelUsesThinkingTimeoutForInitialIdle(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(1, 3)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{delay: 1200 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Text: "delayed first chunk"})},
	}, 0)
	defer resp.Body.Close()

	got, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if got != "delayed first chunk" {
		t.Fatalf("handleSSEResponse() = %q, want %q", got, "delayed first chunk")
	}
}

func TestHandleSSEResponse_NoSSEDataTriggersTransportIdleTimeout(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(1, 3)
	resp := newTimedSSEResponse(t, nil, 1500*time.Millisecond)
	defer resp.Body.Close()

	_, err := p.handleSSEResponse(ctx, resp, nil, "", "")
	if err == nil {
		t.Fatal("handleSSEResponse() should return transport idle timeout")
	}

	var idleErr *ErrIdleTimeout
	if !errors.As(err, &idleErr) {
		t.Fatalf("error should be ErrIdleTimeout, got %T: %v", err, err)
	}
	if !strings.Contains(idleErr.Error(), "transport idle timeout") {
		t.Fatalf("idle timeout message should describe transport idle, got %q", idleErr.Error())
	}
}

func TestHandleSSEResponse_FunctionCallResetsThinkingTimeout(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(2, 1)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{
			FunctionCall: &api.GeminiFunctionCall{
				Name: "read_file",
				Args: map[string]any{"path": "/tmp/a.txt"},
			},
		})},
	}, 700*time.Millisecond)
	defer resp.Body.Close()

	got, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if !strings.Contains(got, "read_file") {
		t.Fatalf("handleSSEResponse() should include function call, got %q", got)
	}
}

func TestHandleSSEResponse_TextResetsThinkingTimeout(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(2, 1)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Thought: true, Text: "thinking"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{Text: "partial answer"})},
	}, 700*time.Millisecond)
	defer resp.Body.Close()

	got, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if got != "partial answer" {
		t.Fatalf("handleSSEResponse() = %q, want %q", got, "partial answer")
	}
}

func TestHandleSSEResponse_TextWithThoughtSignaturePreservesText(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(2, 1)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{payload: mustSSEPayload(t, GeminiFunctionPart{
			Text:             "final answer",
			ThoughtSignature: "sig-text",
		})},
	}, 0)
	defer resp.Body.Close()

	got, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if got != "final answer" {
		t.Fatalf("handleSSEResponse() = %q, want %q", got, "final answer")
	}
}

func TestHandleSSEResponse_ThoughtSignatureOnlyResetsTransportIdle(t *testing.T) {
	p := New("test-key")
	ctx := newGeminiSSETestContext(2, 1)
	resp := newTimedSSEResponse(t, []timedSSEChunk{
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{ThoughtSignature: "sig-1"})},
		{delay: 400 * time.Millisecond, payload: mustSSEPayload(t, GeminiFunctionPart{ThoughtSignature: "sig-2"})},
	}, 1500*time.Millisecond)
	defer resp.Body.Close()

	_, err := p.handleSSEResponse(ctx, resp, nil, "", "gemini-3.5-flash")
	if err == nil {
		t.Fatal("handleSSEResponse() should return thinking timeout")
	}

	var thinkingErr *ErrThinkingTimeout
	if !errors.As(err, &thinkingErr) {
		t.Fatalf("error should be ErrThinkingTimeout, got %T: %v", err, err)
	}
}
