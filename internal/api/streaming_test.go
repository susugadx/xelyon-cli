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
	ctx := config.WithContext(context.Background(), config.DefaultConfig())

	body := "\n\n\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()

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
	ctx := config.WithContext(context.Background(), config.DefaultConfig())

	body := "line1\nline2\nline3\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()

	// 常にエラーを返すパーサー
	parser := func(line string) (string, bool, error) {
		return "", false, errors.New("parse error")
	}

	result, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	if err == nil {
		t.Fatal("ParseStreamingResponse() should return parser error")
	}
	if result != "" {
		t.Errorf("ParseStreamingResponse() result = %q, want empty", result)
	}
	if !strings.Contains(err.Error(), "parse error") {
		t.Errorf("ParseStreamingResponse() error = %v, want parse error", err)
	}
}

func TestParseStreamingResponse_DoneFlag(t *testing.T) {
	ctx := config.WithContext(context.Background(), config.DefaultConfig())

	body := "chunk1\nchunk2\n[DONE]\nchunk3\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()

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
	baseCtx := config.WithContext(context.Background(), config.DefaultConfig())
	ctx, cancel := context.WithCancel(baseCtx)

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
	ctx := config.WithContext(context.Background(), config.DefaultConfig())

	body := "Hello\nWorld\n!\n"
	resp := &http.Response{
		Body: &mockReadCloser{reader: strings.NewReader(body)},
	}

	spinner := ui.NewSpinner()

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
	ctx := config.WithContext(context.Background(), cfg)

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
	_, err := ParseStreamingResponse(ctx, resp, spinner, parser)
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
	ctx := config.WithContext(context.Background(), cfg)

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

	_, err := ParseStreamingResponse(ctx, resp, spinner, parser)

	// エラーなしで完了すること
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestStreamingConfigDefault はデフォルト設定をテスト
func TestStreamingConfigDefault(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Streaming.IdleTimeoutSeconds != 30 {
		t.Errorf("Expected default IdleTimeoutSeconds=30, got %d", cfg.Streaming.IdleTimeoutSeconds)
	}
}

// filterToolJSON tests

func testFilterToolJSON(chunks []string) string {
	inToolJSON := false
	jsonDepth := 0
	inString := false
	escaped := false

	var output strings.Builder
	for _, chunk := range chunks {
		result := filterToolJSON(chunk, &inToolJSON, &jsonDepth, &inString, &escaped)
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
		{
			name:     "nested JSON object",
			chunks:   []string{`JSON: {"tool": "edit", "args": {"files": [{"path": "main.go"}]}} End`},
			expected: "JSON:  End",
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

func TestFilterToolJSON_EscapedBackslash(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected string
	}{
		{
			name: "escaped backslash before closing quote",
			// JSON: {"tool": "write", "args": {"path": "C:\\"}}
			// The \\ is an escaped backslash, so the " after it closes the string.
			chunks:   []string{`{"tool": "write", "args": {"path": "C:\\"}}`},
			expected: "",
		},
		{
			name: "escaped backslash in path",
			// JSON with Windows-style path: C:\\Users\\foo
			chunks:   []string{`{"tool": "write", "args": {"path": "C:\\Users\\foo"}}`},
			expected: "",
		},
		{
			name: "escaped backslash followed by brace in string",
			// The \\ is an escaped backslash, then } is a literal char in the string
			chunks:   []string{`{"tool": "bash", "args": {"cmd": "echo \\}"}}`},
			expected: "",
		},
		{
			name: "multiple escaped backslashes",
			// Four backslashes = two literal backslashes in JSON
			chunks:   []string{`{"tool": "write", "args": {"path": "C:\\\\server\\\\share"}}`},
			expected: "",
		},
		{
			name: "escaped backslash then escaped quote",
			// \\\\" = escaped backslash + escaped quote
			chunks:   []string{`{"tool": "bash", "args": {"cmd": "echo \\\"done\\\""}}`},
			expected: "",
		},
		{
			name:     "escaped backslash across chunks",
			chunks:   []string{`{"tool": "write", "args": {"path": "C:\`, `\"}}`},
			expected: "",
		},
		{
			name:     "text after tool JSON with escaped backslash",
			chunks:   []string{`Hello {"tool": "write", "args": {"path": "C:\\"}} World`},
			expected: "Hello  World",
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

func TestFilterToolJSON_NamePattern(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected string
	}{
		{
			name:     "name pattern (DeepSeek OpenAI format)",
			chunks:   []string{`{"name": "str_replace", "arguments": {"file": "main.go"}}`},
			expected: "",
		},
		{
			name:     "name pattern with space",
			chunks:   []string{`{ "name": "write_file", "arguments": {}}`},
			expected: "",
		},
		{
			name:     "text before name pattern",
			chunks:   []string{`I'll edit the file. {"name": "str_replace", "arguments": {}}`},
			expected: "I'll edit the file. ",
		},
		{
			name:     "name pattern across chunks",
			chunks:   []string{`{"name": "bash",`, ` "arguments": {"command": "ls"}}`},
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

func TestMatchesPatternPrefix(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "no prefix match",
			content:  "Hello world",
			expected: 0,
		},
		{
			name:     "single brace at end",
			content:  "text {",
			expected: 1, // matches `{` from `{"tool"`
		},
		{
			name:     "partial tool pattern",
			content:  `text {"to`,
			expected: 4, // matches `{"to` from `{"tool"`
		},
		{
			name:     "partial id pattern",
			content:  `text {"i`,
			expected: 3, // matches `{"i` from `{"id"`
		},
		{
			name:     "full pattern (not prefix)",
			content:  `text {"tool"`,
			expected: 0, // full pattern = not a prefix, filterToolJSON handles it
		},
		{
			name:     "partial name pattern",
			content:  `text {"na`,
			expected: 4, // matches `{"na` from `{"name"`
		},
		{
			name:     "brace with space",
			content:  `text { `,
			expected: 2, // matches `{ ` from `{ "tool"`
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPatternPrefix(tt.content)
			if result != tt.expected {
				t.Errorf("matchesPatternPrefix(%q) = %d, want %d", tt.content, result, tt.expected)
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
