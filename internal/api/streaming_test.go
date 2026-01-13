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

	_, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != context.Canceled {
		t.Errorf("ParseStreamingResponse() error = %v, want context.Canceled", err)
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

	if cfg.Streaming.IdleTimeoutSeconds != 30 {
		t.Errorf("Expected default IdleTimeoutSeconds=30, got %d", cfg.Streaming.IdleTimeoutSeconds)
	}
}
