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
