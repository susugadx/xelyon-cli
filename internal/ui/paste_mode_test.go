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

	var (
		lines      []string
		emptyCount int
		totalBytes int
	)

	if action := p.processLine("first", &lines, &emptyCount, &totalBytes, 2, 100); action != pasteActionContinue {
		t.Fatalf("processLine(first) = %v, want continue", action)
	}
	if action := p.processLine("", &lines, &emptyCount, &totalBytes, 2, 100); action != pasteActionContinue {
		t.Fatalf("processLine(empty) = %v, want continue", action)
	}
	if action := p.processLine("", &lines, &emptyCount, &totalBytes, 2, 100); action != pasteActionDone {
		t.Fatalf("second empty processLine() = %v, want done", action)
	}

	if got := p.finalize([]string{"line1", "", ""}); got != "line1" {
		t.Fatalf("finalize() = %q, want %q", got, "line1")
	}
	if got := p.finalize(nil); got != "" {
		t.Fatalf("finalize(nil) = %q, want empty string", got)
	}
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
