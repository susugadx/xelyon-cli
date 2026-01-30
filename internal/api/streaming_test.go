package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type mockReadCloser struct {
	reader io.Reader
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *mockReadCloser) Close() error {
	return nil
}

func TestParseStreamingResponse_EmptyLines(t *testing.T) {
	body := "\n\n\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()
	ctx := context.Background()

	parser := func(line string) (string, bool, error) {
		return "", false, nil
	}

	result, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		t.Errorf("ParseStreamingResponse() should not error on empty lines, got: %v", err)
	}

	if result != "" {
		t.Errorf("ParseStreamingResponse() result = %q, want empty string", result)
	}
}

func TestParseStreamingResponse_ParserError(t *testing.T) {
	body := "line1\nline2\nline3\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()
	ctx := context.Background()

	// 常にエラーを返すパーサー
	parser := func(line string) (string, bool, error) {
		return "", false, errors.New("parse error")
	}

	// パースエラーは無視されて続行される
	result, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		t.Errorf("ParseStreamingResponse() should ignore parser errors, got: %v", err)
	}

	if result != "" {
		t.Errorf("ParseStreamingResponse() result = %q, want empty (all lines errored)", result)
	}
}

func TestParseStreamingResponse_DoneFlag(t *testing.T) {
	body := "chunk1\nchunk2\n[DONE]\nchunk3\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()
	ctx := context.Background()

	parser := func(line string) (string, bool, error) {
		if line == "[DONE]" {
			return "", true, nil // done=true でストリーム終了
		}
		return line + " ", false, nil
	}

	result, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
	}

	expected := "chunk1 chunk2 "
	if result != expected {
		t.Errorf("ParseStreamingResponse() result = %q, want %q", result, expected)
	}
}

func TestParseStreamingResponse_ContextCanceled(t *testing.T) {
	// 長いストリーム（読み取り中にキャンセルをシミュレート）
	body := "chunk1\nchunk2\nchunk3\nchunk4\nchunk5\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	parser := func(line string) (string, bool, error) {
		callCount++
		if callCount == 2 {
			// 2回目のパース時にキャンセル
			cancel()
		}
		return line + " ", false, nil
	}

	result, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	// 部分結果がある場合はエラーなしで部分結果を返す（Streaming UX Phase 3）
	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v, want nil (partial result)", err)
	}
	// 部分結果が含まれていることを確認
	if result == "" {
		t.Error("ParseStreamingResponse() expected partial result, got empty string")
	}
}

func TestParseStreamingResponse_NormalFlow(t *testing.T) {
	body := "Hello\nWorld\n!\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()
	ctx := context.Background()

	parser := func(line string) (string, bool, error) {
		return line, false, nil
	}

	result, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
	}

	expected := "HelloWorld!"
	if result != expected {
		t.Errorf("ParseStreamingResponse() result = %q, want %q", result, expected)
	}
}

// slowReader は遅延読み込みをシミュレート
type slowReader struct {
	lines    []string
	index    int
	delay    time.Duration
	lastRead time.Time
}

func (s *slowReader) Read(p []byte) (n int, err error) {
	if s.index >= len(s.lines) {
		return 0, io.EOF
	}

	// 指定された遅延を待機
	if !s.lastRead.IsZero() && s.delay > 0 {
		time.Sleep(s.delay)
	}
	s.lastRead = time.Now()

	line := s.lines[s.index] + "\n"
	s.index++
	copy(p, []byte(line))
	return len(line), nil
}

func (s *slowReader) Close() error {
	return nil
}

// TestStreamingIdleTimeout はアイドルタイムアウトをテスト
func TestStreamingIdleTimeout(t *testing.T) {
	// 短いアイドルタイムアウトを設定
	cfg := config.DefaultConfig()
	cfg.Streaming.IdleTimeoutSeconds = 1 // 1秒
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(config.DefaultConfig()) // テスト後にリセット

	// 2秒遅延で読み込むリーダー（タイムアウトする）
	slowR := &slowReader{
		lines: []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" World"}}]}`,
		},
		delay: 2 * time.Second, // 1秒タイムアウトより長い
	}

	resp := &http.Response{
		Body: &mockReadCloser{reader: slowR},
	}

	spinner := ui.NewSpinner()
	spinner.Start("Testing...")

	parser := func(line string) (string, bool, error) {
		if strings.HasPrefix(line, "data: ") {
			return "chunk", false, nil
		}
		return "", false, nil
	}

	start := time.Now()
	_, err := ParseStreamingResponse(context.Background(), resp, spinner, parser)
	elapsed := time.Since(start)

	// タイムアウトエラーが発生すること
	if err == nil {
		t.Error("Expected idle timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "idle timeout") {
		t.Errorf("Expected idle timeout error, got: %v", err)
	}

	// 1〜2秒程度で終了すること（遅延2秒より前）
	if elapsed > 3*time.Second {
		t.Errorf("Expected timeout within ~1-2s, took %v", elapsed)
	}
}

// TestStreamingNoTimeoutWithContinuousData は連続データでタイムアウトしないことをテスト
func TestStreamingNoTimeoutWithContinuousData(t *testing.T) {
	// 短いアイドルタイムアウトを設定
	cfg := config.DefaultConfig()
	cfg.Streaming.IdleTimeoutSeconds = 2 // 2秒
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(config.DefaultConfig()) // テスト後にリセット

	// 0.5秒遅延で読み込むリーダー（タイムアウトしない）
	slowR := &slowReader{
		lines: []string{
			`data: chunk1`,
			`data: chunk2`,
			`data: chunk3`,
			`data: chunk4`,
			`data: chunk5`,
			`[DONE]`,
		},
		delay: 500 * time.Millisecond, // 2秒タイムアウトより短い
	}

	resp := &http.Response{
		Body: &mockReadCloser{reader: slowR},
	}

	spinner := ui.NewSpinner()
	spinner.Start("Testing...")

	parser := func(line string) (string, bool, error) {
		if line == "[DONE]" {
			return "", true, nil
		}
		if strings.HasPrefix(line, "data: ") {
			return "x", false, nil
		}
		return "", false, nil
	}

	_, err := ParseStreamingResponse(context.Background(), resp, spinner, parser)

	// エラーなしで完了すること
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestStreamingConfigDefault はデフォルト設定をテスト
func TestStreamingConfigDefault(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Streaming.IdleTimeoutSeconds != 3600 {
		t.Errorf("Expected default IdleTimeoutSeconds=3600, got %d", cfg.Streaming.IdleTimeoutSeconds)
	}
}

// filterToolJSON tests

func testFilterToolJSON(chunks []string) string {
	inToolJSON := false
	jsonDepth := 0
	inString := false
	var prevChar rune

	var output strings.Builder
	for _, chunk := range chunks {
		result := filterToolJSON(chunk, &inToolJSON, &jsonDepth, &inString, &prevChar)
		output.WriteString(result)
	}

	return output.String()
}

func TestFilterToolJSON_SingleChunk(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected string
	}{
		{
			name:     "no tool JSON",
			chunks:   []string{"Hello World"},
			expected: "Hello World",
		},
		{
			name:     "tool JSON only",
			chunks:   []string{`{"tool": "read_file", "args": {}}`},
			expected: "",
		},
		{
			name:     "text before tool JSON",
			chunks:   []string{`Hello {"tool": "read_file", "args": {}}`},
			expected: "Hello ",
		},
		{
			name:     "text after tool JSON",
			chunks:   []string{`{"tool": "read_file"} World`},
			expected: " World",
		},
		{
			name:     "text around tool JSON",
			chunks:   []string{`Hello {"tool": "read_file"} World`},
			expected: "Hello  World",
		},
		{
			name:     "multiple tool JSONs",
			chunks:   []string{`First {"tool": "a"} Second {"tool": "b"} Third`},
			expected: "First  Second  Third",
		},
		{
			name:     "space in pattern",
			chunks:   []string{`Hello { "tool": "read_file"}`},
			expected: "Hello ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testFilterToolJSON(tt.chunks)
			if result != tt.expected {
				t.Errorf("filterToolJSON(%v) = %q, want %q", tt.chunks, result, tt.expected)
			}
		})
	}
}

func TestFilterToolJSON_ChunkBoundary(t *testing.T) {
	// 注: シンプル化したロジックでは、チャンク境界でパターンが分断された場合、
	// 最初のチャンクはそのまま表示される（パターン検出できないため）。
	// これは実用上問題にならない（APIレスポンスは通常完全なチャンクを送る）。
	tests := []struct {
		name     string
		chunks   []string
		expected string
	}{
		{
			name:     "complete pattern in second chunk",
			chunks:   []string{`Hello `, `{"tool": "read_file"}`},
			expected: "Hello ",
		},
		{
			name:     "tool JSON spans chunks (continuation)",
			chunks:   []string{`{"tool": "a",`, ` "args": {}}`},
			expected: "", // 最初のチャンクでパターン検出、以降は非表示
		},
		{
			name:     "text then tool JSON in separate chunks",
			chunks:   []string{`First message. `, `{"tool": "test"}`, ` More text.`},
			expected: "First message.  More text.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testFilterToolJSON(tt.chunks)
			if result != tt.expected {
				t.Errorf("filterToolJSON(%v) = %q, want %q", tt.chunks, result, tt.expected)
			}
		})
	}
}

func TestFilterToolJSON_NestedJSON(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected string
	}{
		{
			name:     "nested object",
			chunks:   []string{`{"tool": "a", "args": {"nested": "value"}}`},
			expected: "",
		},
		{
			name:     "deeply nested",
			chunks:   []string{`{"tool": "a", "args": {"level1": {"level2": {"level3": true}}}}`},
			expected: "",
		},
		{
			name:     "nested across chunks",
			chunks:   []string{`{"tool": "a", "args": {`, `"nested": {"deep": true}}}`},
			expected: "",
		},
		{
			name:     "braces in string value",
			chunks:   []string{`{"tool": "bash", "args": {"command": "echo }"}}`},
			expected: "",
		},
		{
			name:     "multiple braces in string",
			chunks:   []string{`{"tool": "bash", "args": {"command": "echo {{}}}"}}`},
			expected: "",
		},
		{
			name:     "escaped quote in string",
			chunks:   []string{`{"tool": "bash", "args": {"command": "echo \"}\"}}`},
			expected: "",
		},
		{
			name:     "braces in string across chunks",
			chunks:   []string{`{"tool": "bash", "args": {"command": "echo `, `}"}}`},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testFilterToolJSON(tt.chunks)
			if result != tt.expected {
				t.Errorf("filterToolJSON(%v) = %q, want %q", tt.chunks, result, tt.expected)
			}
		})
	}
}

func TestFilterToolJSON_NonToolJSON(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected string
	}{
		{
			name:     "other JSON key",
			chunks:   []string{`{"other": "value"}`},
			expected: `{"other": "value"}`,
		},
		{
			name:     "brace in text",
			chunks:   []string{`Hello {world}`},
			expected: `Hello {world}`,
		},
		{
			name:     "partial pattern not matching",
			chunks:   []string{`Hello {"to`, `pic": "value"}`},
			expected: `Hello {"topic": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testFilterToolJSON(tt.chunks)
			if result != tt.expected {
				t.Errorf("filterToolJSON(%v) = %q, want %q", tt.chunks, result, tt.expected)
			}
		})
	}
}
