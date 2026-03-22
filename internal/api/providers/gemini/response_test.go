package gemini

import (
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
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
