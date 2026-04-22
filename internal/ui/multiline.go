package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ErrInterrupted is returned when user presses Ctrl+C
var ErrInterrupted = errors.New("interrupted")
var errReadLineTimeout = errors.New("read line timeout")

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

// ReadSimpleLineWithTimeout は1行入力を読み取り、タイムアウト時に errReadLineTimeout を返す。
// 行ごとに goroutine を生成せず、対話入力のアイドルタイムアウトを扱う用途を想定する。
func (m *MultilineReader) ReadSimpleLineWithTimeout(timeout time.Duration) (string, error) {
	// 先行の ReadSimpleLine() などで m.reader に既に取り込まれたデータを優先消費する。
	line, prefix, hasBufferedLine, err := m.consumeBufferedLineForTimedRead()
	if err != nil {
		return "", err
	}
	if hasBufferedLine {
		return line, nil
	}

	// raw mode goroutine未起動時は bufio.Reader 経路で処理する。
	// 共有 reader に対してこの関数が独自 goroutine を常駐させないようにする。
	if !m.rawModeInit {
		return m.readSimpleLineFromReaderWithTimeout(timeout, prefix)
	}
	if m.byteChan == nil || m.errChan == nil {
		m.initRawModeChannels()
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	timeoutCh := timer.C

	// Enter raw mode to suppress terminal echo (paste markers would be
	// visible in cooked mode because the terminal echoes before we can strip).
	if m.isTerminalInput() {
		oldState, err := m.rawModeOps().makeRaw(m.fd)
		if err == nil {
			defer func() { _ = m.rawModeOps().restore(m.fd, oldState) }()
		}
	}
	return m.readLineFromChannelWithTimeoutAndPrefix(timeoutCh, prefix)
}

func (m *MultilineReader) consumeBufferedLineForTimedRead() (line string, prefix []byte, hasBufferedLine bool, err error) {
	buffered := m.reader.Buffered()
	if buffered == 0 {
		return "", nil, false, nil
	}

	peeked, err := m.reader.Peek(buffered)
	if err != nil {
		return "", nil, false, err
	}

	if consumeLen, found := findBufferedLineConsumeLen(peeked); found {
		lineBytes := append([]byte(nil), peeked[:consumeLen]...)
		if _, err := m.reader.Discard(consumeLen); err != nil {
			return "", nil, false, err
		}
		trimmed := strings.TrimRight(string(lineBytes), "\n\r")
		return StripBracketedPaste(trimmed), nil, true, nil
	}

	prefix = append([]byte(nil), peeked...)
	if _, err := m.reader.Discard(buffered); err != nil {
		return "", nil, false, err
	}
	return "", prefix, false, nil
}

func findBufferedLineConsumeLen(buf []byte) (consumeLen int, found bool) {
	for i, b := range buf {
		switch b {
		case '\n':
			return i + 1, true
		case '\r':
			if i+1 < len(buf) && buf[i+1] == '\n' {
				return i + 2, true
			}
			return i + 1, true
		}
	}
	return 0, false
}

func (m *MultilineReader) readSimpleLineFromReaderWithTimeout(timeout time.Duration, prefix []byte) (string, error) {
	if f, ok := m.input.(*os.File); ok && timeout > 0 {
		if err := f.SetReadDeadline(time.Now().Add(timeout)); err == nil {
			defer func() { _ = f.SetReadDeadline(time.Time{}) }()
			line, err := m.readSimpleLineFromReader(prefix)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					return "", errReadLineTimeout
				}
				return "", err
			}
			return line, nil
		}
	}
	return m.readSimpleLineFromReader(prefix)
}

func (m *MultilineReader) readSimpleLineFromReader(prefix []byte) (string, error) {
	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if len(prefix) > 0 {
		line = string(prefix) + line
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
	return m.readLineFromChannelWithTimeout(nil)
}

func (m *MultilineReader) readLineFromChannelWithTimeout(timeout <-chan time.Time) (string, error) {
	return m.readLineFromChannelWithTimeoutAndPrefix(timeout, nil)
}

func (m *MultilineReader) readLineFromChannelWithTimeoutAndPrefix(timeout <-chan time.Time, prefix []byte) (string, error) {
	var line lineAssembler
	if len(prefix) > 0 {
		line.appendBytes(prefix)
	}
	for {
		select {
		case <-timeout:
			return "", errReadLineTimeout
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
