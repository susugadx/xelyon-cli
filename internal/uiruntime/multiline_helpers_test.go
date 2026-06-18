package uiruntime

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMultilineReader_ReaderHelpersShareSameInstance(t *testing.T) {
	r := NewMultilineReader(strings.NewReader("hello\n"))
	if r.GetBufioReader() == nil || r.Reader() == nil {
		t.Fatal("reader helpers should return non-nil bufio.Reader")
	}
	if r.GetBufioReader() != r.Reader() {
		t.Fatal("GetBufioReader() and Reader() should return the same reader")
	}
}

func TestReadSimpleLine_StripsBracketedPasteMarkers(t *testing.T) {
	r := NewMultilineReader(strings.NewReader("^[[200~hello^[[201~\n"))
	got, err := r.ReadSimpleLine()
	if err != nil {
		t.Fatalf("ReadSimpleLine() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("ReadSimpleLine() = %q, want %q", got, "hello")
	}
}

func TestReadSimpleLineWithTimeout_ConsumesBufferedLine(t *testing.T) {
	r := NewMultilineReader(strings.NewReader("a\nb\n"))
	first, err := r.ReadSimpleLine()
	if err != nil {
		t.Fatalf("ReadSimpleLine() error = %v", err)
	}
	if first != "a" {
		t.Fatalf("ReadSimpleLine() = %q, want %q", first, "a")
	}

	second, err := r.ReadSimpleLineWithTimeout(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("ReadSimpleLineWithTimeout() error = %v", err)
	}
	if second != "b" {
		t.Fatalf("ReadSimpleLineWithTimeout() = %q, want %q", second, "b")
	}
}

func TestReadSimpleLineWithTimeout_DoesNotInitRawModeWhenUnused(t *testing.T) {
	r := NewMultilineReader(strings.NewReader("line\n"))
	if r.rawModeInit {
		t.Fatal("rawModeInit should be false before ReadSimpleLineWithTimeout()")
	}

	got, err := r.ReadSimpleLineWithTimeout(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("ReadSimpleLineWithTimeout() error = %v", err)
	}
	if got != "line" {
		t.Fatalf("ReadSimpleLineWithTimeout() = %q, want %q", got, "line")
	}
	if r.rawModeInit {
		t.Fatal("ReadSimpleLineWithTimeout() should not initialize raw mode channels")
	}
}

func TestReadSimpleLineWithTimeout_ChannelPath(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()
	go feedBytes(ch, []byte("hello"))
	got, err := r.ReadSimpleLineWithTimeout(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("ReadSimpleLineWithTimeout() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("ReadSimpleLineWithTimeout() = %q, want %q", got, "hello")
	}
}

func TestReadSimpleLineWithTimeout_Timeout(t *testing.T) {
	r, _, _ := newTestReaderWithChannel()
	_, err := r.ReadSimpleLineWithTimeout(5 * time.Millisecond)
	if !errors.Is(err, errReadLineTimeout) {
		t.Fatalf("ReadSimpleLineWithTimeout() error = %v, want errReadLineTimeout", err)
	}
}

func TestReadByteTimeoutFromChannel(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()
	ch <- 'x'
	if b, ok := r.readByteTimeoutFromChannel(10 * time.Millisecond); !ok || b != 'x' {
		t.Fatalf("readByteTimeoutFromChannel() = (%q, %v), want ('x', true)", b, ok)
	}
	if _, ok := r.readByteTimeoutFromChannel(5 * time.Millisecond); ok {
		t.Fatal("readByteTimeoutFromChannel() = ok=true, want timeout")
	}
}

func TestDetectPasteMarker(t *testing.T) {
	r := NewMultilineReader(strings.NewReader(""))
	tests := []struct {
		name string
		seq  string
		want pasteMarkerKind
	}{
		{name: "start marker with esc", seq: pasteStart, want: pasteMarkerStart},
		{name: "start marker without esc", seq: "[200~", want: pasteMarkerStart},
		{name: "end marker with esc", seq: pasteEnd, want: pasteMarkerEnd},
		{name: "end marker without esc", seq: "[201~", want: pasteMarkerEnd},
		{name: "non marker", seq: "[20x~", want: pasteMarkerNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.detectPasteMarker(tt.seq); got != tt.want {
				t.Fatalf("detectPasteMarker(%q) = %v, want %v", tt.seq, got, tt.want)
			}
		})
	}
}

func TestReadEscapeSequence(t *testing.T) {
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
}

func TestFlushInput_DiscardsBufferedBytes(t *testing.T) {
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
}

func TestWriteString_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	r := NewMultilineReaderWithOutput(strings.NewReader(""), &out)
	r.writeString("hello")
	if out.String() != "hello" {
		t.Fatalf("writeString() wrote %q, want %q", out.String(), "hello")
	}
}

func TestReadLineFromReaderBuffer_TrimsLineBreakAndMarkers(t *testing.T) {
	r := NewMultilineReader(strings.NewReader("^[[200~hello^[[201~\r\n"))
	got, err := r.readLineFromReaderBuffer()
	if err != nil {
		t.Fatalf("readLineFromReaderBuffer() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("readLineFromReaderBuffer() = %q, want %q", got, "hello")
	}
}

func TestReadMarkerModeLines_StopsAtMarker(t *testing.T) {
	r := NewMultilineReader(strings.NewReader("line 1\nline 2\n```\n"))
	lines, err := r.readMarkerModeLines(false)
	if err != nil {
		t.Fatalf("readMarkerModeLines() error = %v", err)
	}
	if len(lines) != 2 || lines[0] != "line 1" || lines[1] != "line 2" {
		t.Fatalf("readMarkerModeLines() = %#v, want [line 1 line 2]", lines)
	}
}

func TestReadMarkerModeLines_EOFWithCapturedLines(t *testing.T) {
	r := NewMultilineReader(strings.NewReader("line 1\n"))
	lines, err := r.readMarkerModeLines(false)
	if err != nil {
		t.Fatalf("readMarkerModeLines() error = %v", err)
	}
	if len(lines) != 1 || lines[0] != "line 1" {
		t.Fatalf("readMarkerModeLines() = %#v, want [line 1]", lines)
	}
}
