package ui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// ErrInterrupted is returned when user presses Ctrl+C
var ErrInterrupted = errors.New("interrupted")

// Bracketed Paste Mode escape sequences
const (
	bracketedPasteEnable  = "\x1b[?2004h" // Enable bracketed paste mode
	bracketedPasteDisable = "\x1b[?2004l" // Disable bracketed paste mode
	pasteStart            = "\x1b[200~"   // Paste start marker
	pasteEnd              = "\x1b[201~"   // Paste end marker
)

// MultilineReader handles multiline input with bracketed paste mode and ``` markers
type MultilineReader struct {
	reader                *bufio.Reader
	input                 io.Reader
	out                   io.Writer
	err                   io.Writer
	rawMode               rawModeController
	bracketedPasteEnabled bool
	fd                    int // file descriptor for stdin (for raw mode)
	// Raw mode channels (initialized lazily, reused across calls)
	byteChan    chan byte
	errChan     chan error
	rawModeInit bool
}

type rawModeController interface {
	isTerminal(fd int) bool
	makeRaw(fd int) (*term.State, error)
	restore(fd int, state *term.State) error
}

type systemRawModeController struct{}

func (systemRawModeController) isTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

func (systemRawModeController) makeRaw(fd int) (*term.State, error) {
	return term.MakeRaw(fd)
}

func (systemRawModeController) restore(fd int, state *term.State) error {
	return term.Restore(fd, state)
}

// NewMultilineReader creates a new multiline reader
func NewMultilineReader(r io.Reader) *MultilineReader {
	return NewMultilineReaderWithOutput(r, DefaultRuntime().Output())
}

// NewMultilineReaderWithOutput creates a new multiline reader with an explicit output writer.
func NewMultilineReaderWithOutput(r io.Reader, out io.Writer) *MultilineReader {
	return newMultilineReader(r, out, DefaultRuntime().ErrorOutput())
}

// NewMultilineReaderWithRuntime は runtime の入出力に紐づく multiline reader を返す。
func NewMultilineReaderWithRuntime(runtime *Runtime) *MultilineReader {
	rt := runtimeOrDefault(runtime)
	return newMultilineReader(rt.Input(), rt.Output(), rt.ErrorOutput())
}

func newMultilineReader(r io.Reader, out, errOut io.Writer) *MultilineReader {
	if r == nil {
		r = DefaultRuntime().Input()
	}
	fd := -1
	if f, ok := r.(*os.File); ok {
		fd = int(f.Fd())
	}
	if out == nil {
		out = DefaultRuntime().Output()
	}
	if errOut == nil {
		errOut = DefaultRuntime().ErrorOutput()
	}
	return &MultilineReader{
		reader:                bufio.NewReaderSize(r, 1024*1024), // 1MB buffer
		input:                 r,
		out:                   out,
		err:                   errOut,
		rawMode:               systemRawModeController{},
		bracketedPasteEnabled: false,
		fd:                    fd,
		byteChan:              nil,
		errChan:               nil,
		rawModeInit:           false,
	}
}

func (m *MultilineReader) rawModeOps() rawModeController {
	if m != nil && m.rawMode != nil {
		return m.rawMode
	}
	return systemRawModeController{}
}

func (m *MultilineReader) isTerminalInput() bool {
	return m != nil && m.fd >= 0 && m.rawModeOps().isTerminal(m.fd)
}

// initRawModeChannels initializes the raw mode channels and goroutine (once)
func (m *MultilineReader) initRawModeChannels() {
	if m.rawModeInit {
		return
	}
	m.byteChan = make(chan byte, 4096)
	m.errChan = make(chan error, 1)
	m.rawModeInit = true

	go func() {
		b := make([]byte, 1)
		input := m.input
		if input == nil {
			input = DefaultRuntime().Input()
		}
		for {
			_, err := input.Read(b)
			if err != nil {
				m.errChan <- err
				return
			}
			m.byteChan <- b[0]
		}
	}()
}

// EnableBracketedPaste enables bracketed paste mode
// This sends the escape sequence to the terminal to enable the mode
// Windows Terminal skips multiline paste warning when this mode is active
func (m *MultilineReader) EnableBracketedPaste() {
	pasteDebugf(m.errorWriter(), "[DEBUG] EnableBracketedPaste: fd=%d, IsTerminal=%v\n", m.fd, m.isTerminalInput())

	if m.isTerminalInput() {
		// Use WriteString for immediate, unbuffered output
		m.writeString(bracketedPasteEnable)
		m.bracketedPasteEnabled = true
		pasteDebugf(m.errorWriter(), "[DEBUG] Sent: \\x1b[?2004h (bracketed paste enable)\n")
	} else {
		pasteDebugf(m.errorWriter(), "[DEBUG] Skipped: not a terminal\n")
	}
}

// DisableBracketedPaste disables bracketed paste mode
// This sends the escape sequence to the terminal to disable the mode
func (m *MultilineReader) DisableBracketedPaste() {
	if m.bracketedPasteEnabled {
		m.writeString(bracketedPasteDisable)
		m.bracketedPasteEnabled = false
	}
}

// StripBracketedPaste removes all bracketed paste markers from input.
// Handles both ESC sequence forms (\x1b[200~) and literal forms (^[[200~).
// Exported so other packages can use it for consistent bracket paste cleaning.
func StripBracketedPaste(input string) string {
	// ESC sequence forms
	input = strings.ReplaceAll(input, "\x1b[200~", "")
	input = strings.ReplaceAll(input, "\x1b[201~", "")
	// Literal ^[ forms (displayed as text in some terminals)
	input = strings.ReplaceAll(input, "^[[200~", "")
	input = strings.ReplaceAll(input, "^[[201~", "")
	return input
}

// stripAllBracketedPasteMarkers is an alias for backward compatibility within this package.
func stripAllBracketedPasteMarkers(input string) string {
	return StripBracketedPaste(input)
}

// ReadInput reads user input, supporting:
// 1. Bracketed paste mode (multiline paste detection)
// 2. ``` markers for explicit multiline mode
// 3. Single line input (default)
// All bracketed paste markers are automatically stripped from input
func (m *MultilineReader) ReadInput(prompt string) (string, error) {
	m.print(prompt)

	// If bracketed paste mode is enabled and we're in a terminal, use raw mode
	if m.bracketedPasteEnabled && m.isTerminalInput() {
		return m.readWithBracketedPaste()
	}

	// Fallback: standard line-by-line reading
	return m.readLine()
}

// readLine reads a single line (standard mode)
func (m *MultilineReader) readLine() (string, error) {
	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimRight(line, "\n\r")

	// Always strip bracketed paste markers (in case terminal sends them without raw mode)
	line = stripAllBracketedPasteMarkers(line)

	// Case 1: ``` marker detected - explicit multiline mode
	if line == "```" {
		return m.readMultilineWithMarker()
	}

	// Case 2: Single line input
	return line, nil
}

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

// readWithBracketedPaste reads input using raw mode with paste marker detection
func (m *MultilineReader) readWithBracketedPaste() (string, error) {
	pasteDebugf(m.errorWriter(), "[DEBUG] Entering raw mode...\n")

	oldState, err := m.rawModeOps().makeRaw(m.fd)
	if err != nil {
		pasteDebugf(m.errorWriter(), "[DEBUG] MakeRaw FAILED: %v\n", err)
		return m.readLine()
	}
	defer func() { _ = m.rawModeOps().restore(m.fd, oldState) }()

	pasteDebugWriteString(m.errorWriter(), "[DEBUG] Raw mode OK\r\n")

	var line lineAssembler
	var paste pasteAssembler

	// Initialize raw mode channels (once, reused across calls)
	m.initRawModeChannels()

	// Helper to read next byte with timeout
	readByteTimeout := func(timeout time.Duration) (byte, bool) {
		select {
		case b := <-m.byteChan:
			return b, true
		case <-time.After(timeout):
			return 0, false
		}
	}

	for {
		select {
		case b := <-m.byteChan:
			// Ctrl+C - always handle first (even in paste mode)
			if b == 0x03 {
				return "", m.interruptRawInput(oldState)
			}

			// ESC or '[' - check for paste marker
			// Some terminals send \x1b[200~, others send [200~ without ESC
			if m.isEscapeLeadByte(b) {
				escBuf, marker, err := m.readEscapeSequence(b, readByteTimeout, func(next byte) error {
					if next == 0x03 {
						return m.interruptRawInput(oldState)
					}
					return nil
				})
				if err != nil {
					return "", err
				}
				switch marker {
				case pasteMarkerStart:
					// Only start paste mode if not already in paste mode.
					if !paste.active {
						paste.start()
						pasteDebugWriteString(m.errorWriter(), "[DEBUG] Paste START\r\n")
					}
				case pasteMarkerEnd:
					content := paste.finish()
					pasteDebugf(m.errorWriter(), "[DEBUG] Paste END, %d bytes\r\n", len(content))
					// Add pasted content to buffer (don't return yet - wait for Enter)
					line.appendString(content)
					// Echo pasted content to terminal so user can see it
					m.echoPastedContent(content)
				default:
					if len(escBuf) > 0 {
						if paste.active {
							paste.appendBytes(escBuf)
						} else {
							line.appendBytes(escBuf)
						}
					}
				}
				continue
			}

			// In paste mode - collect everything
			if paste.active {
				paste.appendByte(b)
				continue
			}

			// Ctrl+D
			if b == 0x04 {
				if line.len() == 0 {
					return "", io.EOF
				}
				m.print("\r\n")
				return line.string(), nil
			}

			// Enter
			if b == '\r' || b == '\n' {
				m.print("\r\n")
				content := line.string()
				content = stripAllBracketedPasteMarkers(content)
				if content == "```" {
					_ = m.rawModeOps().restore(m.fd, oldState)
					return m.readMultilineWithMarker()
				}
				return content, nil
			}

			// Backspace
			if b == 0x7f || b == 0x08 {
				m.handleBackspace(&line)
				continue
			}

			// Regular character
			line.appendByte(b)
			m.echoByte(line.bytes(), b)

		case err := <-m.errChan:
			// If bytes are still buffered, process them first.
			// This avoids dropping pending control bytes (e.g. Ctrl+C)
			// when EOF races with byte delivery.
			if len(m.byteChan) > 0 {
				continue
			}
			if err == io.EOF {
				return line.string(), nil
			}
			return "", err
		}
	}
}

// readMultilineWithMarker handles explicit multiline mode with ``` markers
func (m *MultilineReader) readMultilineWithMarker() (string, error) {
	m.println("📝 Multiline input mode (end with ``` on a new line)")

	// When raw mode goroutine is active, enter raw mode to suppress
	// terminal echo (prevents paste markers from being displayed)
	useChannel := m.rawModeInit && m.byteChan != nil
	var oldState *term.State
	if useChannel && m.isTerminalInput() {
		st, err := m.rawModeOps().makeRaw(m.fd)
		if err == nil {
			oldState = st
			defer func() { _ = m.rawModeOps().restore(m.fd, oldState) }()
		}
	}

	var lines []string
	lineNum := 1

	for {
		m.printf("%3d | ", lineNum)

		var line string
		var err error
		if useChannel {
			line, err = m.readLineFromChannel()
		} else {
			line, err = m.reader.ReadString('\n')
			if err == nil {
				line = strings.TrimRight(line, "\n\r")
				line = stripAllBracketedPasteMarkers(line)
			}
		}
		if err != nil {
			if err == io.EOF && len(lines) > 0 {
				break
			}
			return "", err
		}

		// Check for end marker
		if line == "```" {
			break
		}

		lines = append(lines, line)
		lineNum++
	}

	result := strings.Join(lines, "\n")
	m.printf("✅ Captured %d lines\n", len(lines))
	return result, nil
}

// IsMultilineMarker checks if the input is a multiline marker
func IsMultilineMarker(input string) bool {
	return input == "```"
}

// TrimBracketedPasteMarkers removes bracketed paste markers from input (both forms)
func TrimBracketedPasteMarkers(input string) string {
	return stripAllBracketedPasteMarkers(input)
}

// GetBufioReader returns the internal bufio.Reader for direct access
func (m *MultilineReader) GetBufioReader() *bufio.Reader {
	return m.reader
}

// ReadSimpleLine reads a line for simple prompts (like selector, comment input).
// When the raw mode goroutine is active, enters raw mode so that terminal echo
// is suppressed and paste marker detection in readLineFromChannel works correctly.
func (m *MultilineReader) ReadSimpleLine() (string, error) {
	// If raw mode goroutine is running, read from channel
	if m.rawModeInit && m.byteChan != nil {
		// Enter raw mode to suppress terminal echo (paste markers would be
		// visible in cooked mode because the terminal echoes before we can strip)
		if m.isTerminalInput() {
			oldState, err := m.rawModeOps().makeRaw(m.fd)
			if err == nil {
				defer func() { _ = m.rawModeOps().restore(m.fd, oldState) }()
				return m.readLineFromChannel()
			}
		}
		return m.readLineFromChannel()
	}

	// Otherwise use standard bufio reading
	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\n\r")
	return StripBracketedPaste(line), nil
}

// readByteTimeoutFromChannel reads a byte from the channel with timeout
func (m *MultilineReader) readByteTimeoutFromChannel(timeout time.Duration) (byte, bool) {
	select {
	case b := <-m.byteChan:
		return b, true
	case <-time.After(timeout):
		return 0, false
	}
}

// readLineFromChannel reads a line from the byte channel (when goroutine is active)
func (m *MultilineReader) readLineFromChannel() (string, error) {
	var line lineAssembler
	for {
		select {
		case b := <-m.byteChan:
			// Enter (raw mode では '\r' が来る)
			if b == '\n' || b == '\r' {
				m.print("\r\n") // 改行をエコー
				return StripBracketedPaste(line.string()), nil
			}

			// ESC or '[' - check for paste marker (suppress echo of marker bytes)
			if m.isEscapeLeadByte(b) {
				escBuf, marker, err := m.readEscapeSequence(b, m.readByteTimeoutFromChannel, nil)
				if err != nil {
					return "", err
				}
				if marker == pasteMarkerNone && len(escBuf) > 0 {
					line.appendBytes(escBuf)
					// Echo non-marker bytes
					m.echoPrintableASCII(escBuf)
				}
				continue
			}

			// Backspace / DEL
			if b == 0x7f || b == 0x08 {
				m.handleBackspace(&line)
				continue
			}

			line.appendByte(b)
			m.echoByte(line.bytes(), b)
		case err := <-m.errChan:
			// If bytes are still buffered, process them first.
			// This avoids dropping pending bytes when EOF races with byte delivery.
			if len(m.byteChan) > 0 {
				continue
			}
			if line.len() > 0 {
				return StripBracketedPaste(line.string()), nil
			}
			return "", err
		}
	}
}

// FlushInput discards any buffered input data
// This should be called after AI output completes to ignore keypresses during output
func (m *MultilineReader) FlushInput() {
	// Discard all buffered data
	if _, err := m.reader.Discard(m.reader.Buffered()); err != nil {
		// Best-effort: ignore flush errors to avoid breaking interactive UX
		return
	}
}

// Reader returns the underlying bufio.Reader for sharing with other input handlers
func (m *MultilineReader) Reader() *bufio.Reader {
	return m.reader
}

// IsBracketedPasteEnabled returns whether bracketed paste mode is enabled
func (m *MultilineReader) IsBracketedPasteEnabled() bool {
	return m.bracketedPasteEnabled
}

func (m *MultilineReader) outputWriter() io.Writer {
	if m.out == nil {
		return DefaultRuntime().Output()
	}
	return m.out
}

func (m *MultilineReader) errorWriter() io.Writer {
	if m.err == nil {
		return DefaultRuntime().ErrorOutput()
	}
	return m.err
}

func (m *MultilineReader) print(args ...interface{}) {
	_, _ = fmt.Fprint(m.outputWriter(), args...)
}

func (m *MultilineReader) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(m.outputWriter(), format, args...)
}

func (m *MultilineReader) println(args ...interface{}) {
	_, _ = fmt.Fprintln(m.outputWriter(), args...)
}

func (m *MultilineReader) writeString(s string) {
	_, _ = io.WriteString(m.outputWriter(), s)
}
