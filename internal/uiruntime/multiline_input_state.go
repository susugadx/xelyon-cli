package uiruntime

import (
	"bytes"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type pasteMarkerKind int

const (
	pasteMarkerNone pasteMarkerKind = iota
	pasteMarkerStart
	pasteMarkerEnd
)

const escapeSequenceReadTimeout = 10 * time.Millisecond

type lineAssembler struct {
	data []byte
}

func (l *lineAssembler) appendByte(b byte) {
	l.data = append(l.data, b)
}

func (l *lineAssembler) appendBytes(bytes []byte) {
	l.data = append(l.data, bytes...)
}

func (l *lineAssembler) appendString(s string) {
	l.data = append(l.data, s...)
}

func (l *lineAssembler) len() int {
	return len(l.data)
}

func (l *lineAssembler) bytes() []byte {
	return l.data
}

func (l *lineAssembler) string() string {
	return string(l.data)
}

func (l *lineAssembler) deleteLastRune() (rune, bool) {
	if len(l.data) == 0 {
		return 0, false
	}
	r, size := utf8.DecodeLastRune(l.data)
	if size <= 0 {
		return 0, false
	}
	l.data = l.data[:len(l.data)-size]
	return r, true
}

type pasteAssembler struct {
	active bool
	buf    bytes.Buffer
}

func (p *pasteAssembler) start() {
	if p.active {
		return
	}
	p.active = true
	p.buf.Reset()
}

func (p *pasteAssembler) appendByte(b byte) {
	p.buf.WriteByte(b)
}

func (p *pasteAssembler) appendBytes(bytes []byte) {
	p.buf.Write(bytes)
}

func (p *pasteAssembler) finish() string {
	p.active = false
	content := p.buf.String()
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.TrimRight(content, "\n")
	p.buf.Reset()
	return content
}

type escapeByteReader func(timeout time.Duration) (byte, bool)
type escapeByteValidator func(b byte) error

func (m *MultilineReader) isEscapeLeadByte(b byte) bool {
	return b == 0x1b || b == '['
}

func (m *MultilineReader) maxEscapeSequenceRead(first byte) int {
	if first == 0x1b {
		return 6 // ESC + [200~ = 6 bytes total
	}
	return 5
}

func (m *MultilineReader) detectPasteMarker(sequence string) pasteMarkerKind {
	switch sequence {
	case pasteStart, "[200~":
		return pasteMarkerStart
	case pasteEnd, "[201~":
		return pasteMarkerEnd
	default:
		return pasteMarkerNone
	}
}

func (m *MultilineReader) readEscapeSequence(first byte, readNext escapeByteReader, validate escapeByteValidator) (unhandled []byte, marker pasteMarkerKind, err error) {
	escBuf := []byte{first}
	for i := 0; i < m.maxEscapeSequenceRead(first); i++ {
		nb, ok := readNext(escapeSequenceReadTimeout)
		if !ok {
			break
		}
		if validate != nil {
			if err := validate(nb); err != nil {
				return nil, pasteMarkerNone, err
			}
		}
		escBuf = append(escBuf, nb)
		if marker := m.detectPasteMarker(string(escBuf)); marker != pasteMarkerNone {
			return nil, marker, nil
		}
	}
	return escBuf, pasteMarkerNone, nil
}

func (m *MultilineReader) interruptRawInput(oldState *term.State) error {
	_ = m.rawModeOps().restore(m.fd, oldState)
	m.print("^C\r\n")
	return ErrInterrupted
}

func (m *MultilineReader) eraseRune(r rune) {
	w := runewidth.RuneWidth(r)
	for i := 0; i < w; i++ {
		m.print("\b \b")
	}
}

func (m *MultilineReader) handleBackspace(line *lineAssembler) {
	if r, ok := line.deleteLastRune(); ok {
		m.eraseRune(r)
	}
}

func (m *MultilineReader) echoByte(lineBytes []byte, b byte) {
	if b < 0x80 {
		// ASCII: 印字可能文字のみエコー
		if b >= 0x20 && b != 0x7f {
			m.print(string(b))
		}
		return
	}
	if b >= 0xC0 {
		// UTF-8マルチバイトの先頭: 何もしない（継続バイトを待つ）
		return
	}
	// 継続バイト (0x80-0xBF): 文字が完成したかチェック
	n := len(lineBytes)
	if n >= 2 && utf8.Valid(lineBytes[n-2:]) {
		r, _ := utf8.DecodeLastRune(lineBytes)
		if r != utf8.RuneError {
			m.print(string(r))
		}
		return
	}
	if n >= 3 && utf8.Valid(lineBytes[n-3:]) {
		r, _ := utf8.DecodeLastRune(lineBytes)
		if r != utf8.RuneError {
			m.print(string(r))
		}
		return
	}
	if n >= 4 && utf8.Valid(lineBytes[n-4:]) {
		r, _ := utf8.DecodeLastRune(lineBytes)
		if r != utf8.RuneError {
			m.print(string(r))
		}
	}
}

func (m *MultilineReader) echoPrintableASCII(bytes []byte) {
	for _, b := range bytes {
		if b >= 0x20 && b < 0x80 && b != 0x7f {
			m.print(string(b))
		}
	}
}

func (m *MultilineReader) echoPastedContent(content string) {
	// In raw mode, need \r\n for proper line breaks
	displayContent := strings.ReplaceAll(content, "\n", "\r\n")
	m.print(displayContent)
}
