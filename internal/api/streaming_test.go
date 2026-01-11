package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

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
