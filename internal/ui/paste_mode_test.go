package ui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestPasteModeProcessLineAndFinalize(t *testing.T) {
	p := NewPasteMode(config.PasteConfig{MaxLines: 2, MaxBytes: 100, TimeoutSeconds: 1})

	t.Run("processLine", func(t *testing.T) {
		tests := []struct {
			name      string
			line      string
			lines     []string
			empty     int
			bytes     int
			maxLines  int
			maxBytes  int
			want      pasteAction
			wantLines []string
		}{
			{
				name:      "通常行は継続",
				line:      "first",
				maxLines:  10,
				maxBytes:  100,
				want:      pasteActionContinue,
				wantLines: []string{"first"},
			},
			{
				name:      "空行2回で終了",
				line:      "",
				lines:     []string{"first", ""},
				empty:     1,
				maxLines:  10,
				maxBytes:  100,
				want:      pasteActionDone,
				wantLines: []string{"first", ""},
			},
			{
				name:      "ENDで終了",
				line:      "END",
				maxLines:  10,
				maxBytes:  100,
				want:      pasteActionDone,
				wantLines: nil,
			},
			{
				name:      "cancelでキャンセル",
				line:      "/cancel",
				maxLines:  10,
				maxBytes:  100,
				want:      pasteActionCancel,
				wantLines: nil,
			},
			{
				name:      "最大行で終了",
				line:      "second",
				lines:     []string{"first"},
				maxLines:  2,
				maxBytes:  100,
				want:      pasteActionDone,
				wantLines: []string{"first", "second"},
			},
			{
				name:      "最大バイトで終了",
				line:      "abcde",
				maxLines:  10,
				maxBytes:  6, // len("abcde")+1
				want:      pasteActionDone,
				wantLines: []string{"abcde"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				lines := append([]string(nil), tt.lines...)
				emptyCount := tt.empty
				totalBytes := tt.bytes

				got := p.processLine(tt.line, &lines, &emptyCount, &totalBytes, tt.maxLines, tt.maxBytes)
				if got != tt.want {
					t.Fatalf("processLine() = %v, want %v", got, tt.want)
				}
				if strings.Join(lines, "\n") != strings.Join(tt.wantLines, "\n") {
					t.Fatalf("lines = %v, want %v", lines, tt.wantLines)
				}
			})
		}
	})

	t.Run("finalize", func(t *testing.T) {
		if got := p.finalize([]string{"line1", "", ""}); got != "line1" {
			t.Fatalf("finalize() = %q, want %q", got, "line1")
		}
		if got := p.finalize(nil); got != "" {
			t.Fatalf("finalize(nil) = %q, want empty string", got)
		}
	})
}

func TestPasteModeCaptureVariants(t *testing.T) {
	t.Run("Capture wrapper ends on max lines", func(t *testing.T) {
		p := NewPasteMode(config.PasteConfig{MaxLines: 2, MaxBytes: 100, TimeoutSeconds: 1})
		var out bytes.Buffer
		content, cancelled, err := p.Capture(strings.NewReader("a\nb\nc\n"), &out)
		if err != nil {
			t.Fatalf("Capture() error = %v", err)
		}
		if cancelled {
			t.Fatal("Capture() cancelled = true, want false")
		}
		if content != "a\nb" {
			t.Fatalf("Capture() = %q, want %q", content, "a\nb")
		}
		if !strings.Contains(out.String(), "Paste Mode") {
			t.Fatalf("expected banner output, got %q", out.String())
		}
	})

	t.Run("CaptureWithReader cancels", func(t *testing.T) {
		p := NewPasteMode(config.PasteConfig{MaxLines: 10, MaxBytes: 100, TimeoutSeconds: 1})
		var out bytes.Buffer
		content, cancelled, err := p.CaptureWithReader(bufio.NewReader(strings.NewReader("/cancel\n")), &out)
		if err != nil {
			t.Fatalf("CaptureWithReader() error = %v", err)
		}
		if !cancelled {
			t.Fatal("CaptureWithReader() cancelled = false, want true")
		}
		if content != "" {
			t.Fatalf("CaptureWithReader() content = %q, want empty", content)
		}
	})

	t.Run("CaptureWithMultilineReader handles end marker", func(t *testing.T) {
		p := NewPasteMode(config.PasteConfig{MaxLines: 10, MaxBytes: 100, TimeoutSeconds: 1})
		var out bytes.Buffer
		reader := NewMultilineReaderWithOutput(strings.NewReader("line1\n/end\n"), &out)
		content, cancelled, err := p.CaptureWithMultilineReader(reader, &out)
		if err != nil {
			t.Fatalf("CaptureWithMultilineReader() error = %v", err)
		}
		if cancelled {
			t.Fatal("CaptureWithMultilineReader() cancelled = true, want false")
		}
		if content != "line1" {
			t.Fatalf("CaptureWithMultilineReader() = %q, want %q", content, "line1")
		}
	})
}
