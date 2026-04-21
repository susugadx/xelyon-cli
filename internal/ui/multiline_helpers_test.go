package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestMultilineReaderHelperMethods(t *testing.T) {
	t.Run("GetBufioReader and Reader share same instance", func(t *testing.T) {
		r := NewMultilineReader(strings.NewReader("hello\n"))
		if r.GetBufioReader() == nil || r.Reader() == nil {
			t.Fatal("reader helpers should return non-nil bufio.Reader")
		}
		if r.GetBufioReader() != r.Reader() {
			t.Fatal("GetBufioReader() and Reader() should return the same reader")
		}
	})

	t.Run("ReadSimpleLine strips markers", func(t *testing.T) {
		r := NewMultilineReader(strings.NewReader("^[[200~hello^[[201~\n"))
		got, err := r.ReadSimpleLine()
		if err != nil {
			t.Fatalf("ReadSimpleLine() error = %v", err)
		}
		if got != "hello" {
			t.Fatalf("ReadSimpleLine() = %q, want %q", got, "hello")
		}
	})

	t.Run("readByteTimeoutFromChannel", func(t *testing.T) {
		r, ch, _ := newTestReaderWithChannel()
		ch <- 'x'
		if b, ok := r.readByteTimeoutFromChannel(10 * time.Millisecond); !ok || b != 'x' {
			t.Fatalf("readByteTimeoutFromChannel() = (%q, %v), want ('x', true)", b, ok)
		}
		if _, ok := r.readByteTimeoutFromChannel(5 * time.Millisecond); ok {
			t.Fatal("readByteTimeoutFromChannel() = ok=true, want timeout")
		}
	})

	t.Run("detectPasteMarker", func(t *testing.T) {
		r := NewMultilineReader(strings.NewReader(""))
		tests := []struct {
			seq  string
			want pasteMarkerKind
		}{
			{seq: pasteStart, want: pasteMarkerStart},
			{seq: "[200~", want: pasteMarkerStart},
			{seq: pasteEnd, want: pasteMarkerEnd},
			{seq: "[201~", want: pasteMarkerEnd},
			{seq: "[20x~", want: pasteMarkerNone},
		}
		for _, tt := range tests {
			if got := r.detectPasteMarker(tt.seq); got != tt.want {
				t.Fatalf("detectPasteMarker(%q) = %v, want %v", tt.seq, got, tt.want)
			}
		}
	})

	t.Run("readEscapeSequence", func(t *testing.T) {
		r := NewMultilineReader(strings.NewReader(""))

		t.Run("marker検出時はunhandledを返さない", func(t *testing.T) {
			tail := []byte{'2', '0', '0', '~'}
			idx := 0
			readNext := func(time.Duration) (byte, bool) {
				if idx >= len(tail) {
					return 0, false
				}
				b := tail[idx]
				idx++
				return b, true
			}
			unhandled, marker, err := r.readEscapeSequence('[', readNext, nil)
			if err != nil {
				t.Fatalf("readEscapeSequence() error = %v", err)
			}
			if marker != pasteMarkerStart {
				t.Fatalf("marker = %v, want %v", marker, pasteMarkerStart)
			}
			if len(unhandled) != 0 {
				t.Fatalf("unhandled = %q, want empty", string(unhandled))
			}
		})

		t.Run("marker非検出時は入力を保持", func(t *testing.T) {
			tail := []byte{'2', '0', 'x', '~'}
			idx := 0
			readNext := func(time.Duration) (byte, bool) {
				if idx >= len(tail) {
					return 0, false
				}
				b := tail[idx]
				idx++
				return b, true
			}
			unhandled, marker, err := r.readEscapeSequence('[', readNext, nil)
			if err != nil {
				t.Fatalf("readEscapeSequence() error = %v", err)
			}
			if marker != pasteMarkerNone {
				t.Fatalf("marker = %v, want %v", marker, pasteMarkerNone)
			}
			if got := string(unhandled); got != "[20x~" {
				t.Fatalf("unhandled = %q, want %q", got, "[20x~")
			}
		})
	})

	t.Run("FlushInput discards buffered bytes", func(t *testing.T) {
		r := NewMultilineReader(strings.NewReader("first\nsecond\n"))
		if _, err := r.GetBufioReader().Peek(5); err != nil {
			t.Fatalf("Peek() error = %v", err)
		}
		if r.GetBufioReader().Buffered() == 0 {
			t.Fatal("expected buffered data before FlushInput()")
		}
		r.FlushInput()
		if got := r.GetBufioReader().Buffered(); got != 0 {
			t.Fatalf("Buffered() after FlushInput = %d, want 0", got)
		}
	})

	t.Run("writeString uses runtime output", func(t *testing.T) {
		var out bytes.Buffer
		r := NewMultilineReaderWithOutput(strings.NewReader(""), &out)
		r.writeString("hello")
		if out.String() != "hello" {
			t.Fatalf("writeString() wrote %q, want %q", out.String(), "hello")
		}
	})
}
