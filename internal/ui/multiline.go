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
	bracketedPasteEnabled bool
	fd                    int // file descriptor for stdin (for raw mode)
	// Raw mode channels (initialized lazily, reused across calls)
	byteChan    chan byte
	errChan     chan error
	rawModeInit bool
}

// NewMultilineReader creates a new multiline reader
func NewMultilineReader(r io.Reader) *MultilineReader {
	fd := -1
	if f, ok := r.(*os.File); ok {
		fd = int(f.Fd())
	}
	return &MultilineReader{
		reader:                bufio.NewReaderSize(r, 1024*1024), // 1MB buffer
		bracketedPasteEnabled: false,
		fd:                    fd,
		byteChan:              nil,
		errChan:               nil,
		rawModeInit:           false,
	}
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
		for {
			_, err := os.Stdin.Read(b)
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
	// Debug: XELYON_DEBUG_PASTE=1 で詳細表示
	debug := os.Getenv("XELYON_DEBUG_PASTE") == "1"

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] EnableBracketedPaste: fd=%d, IsTerminal=%v\n", m.fd, m.fd >= 0 && term.IsTerminal(m.fd))
	}

	if m.fd >= 0 && term.IsTerminal(m.fd) {
		// Use WriteString for immediate, unbuffered output
		_, _ = os.Stdout.WriteString(bracketedPasteEnable)
		m.bracketedPasteEnabled = true
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Sent: \\x1b[?2004h (bracketed paste enable)\n")
		}
	} else if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Skipped: not a terminal\n")
	}
}

// DisableBracketedPaste disables bracketed paste mode
// This sends the escape sequence to the terminal to disable the mode
func (m *MultilineReader) DisableBracketedPaste() {
	if m.bracketedPasteEnabled {
		_, _ = os.Stdout.WriteString(bracketedPasteDisable)
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
	fmt.Print(prompt)

	// If bracketed paste mode is enabled and we're in a terminal, use raw mode
	if m.bracketedPasteEnabled && m.fd >= 0 && term.IsTerminal(m.fd) {
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
	debug := os.Getenv("XELYON_DEBUG_PASTE") == "1"

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Entering raw mode...\n")
	}

	oldState, err := term.MakeRaw(m.fd)
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] MakeRaw FAILED: %v\n", err)
		}
		return m.readLine()
	}
	defer func() { _ = term.Restore(m.fd, oldState) }()

	if debug {
		_, _ = os.Stderr.WriteString("[DEBUG] Raw mode OK\r\n")
	}

	var buf bytes.Buffer
	var pasteContent bytes.Buffer
	inPaste := false

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
				_ = term.Restore(m.fd, oldState)
				fmt.Print("^C\r\n")
				return "", ErrInterrupted
			}

			// ESC or '[' - check for paste marker
			// Some terminals send \x1b[200~, others send [200~ without ESC
			if b == 0x1b || b == '[' {
				// Try to read paste marker: [200~ or [201~ (or with ESC prefix)
				escBuf := []byte{b}
				maxRead := 5
				if b == 0x1b {
					maxRead = 6 // ESC + [200~ = 6 bytes total
				}
				markerDetected := false
				for i := 0; i < maxRead; i++ {
					nb, ok := readByteTimeout(10 * time.Millisecond)
					if !ok {
						break
					}
					// Check for Ctrl+C even inside escape sequence detection
					if nb == 0x03 {
						_ = term.Restore(m.fd, oldState)
						fmt.Print("^C\r\n")
						return "", ErrInterrupted
					}
					escBuf = append(escBuf, nb)

					escStr := string(escBuf)
					// Check both with and without ESC prefix
					if escStr == pasteStart || escStr == "[200~" { // \x1b[200~ or [200~
						// Only start paste mode if not already in paste mode
						if !inPaste {
							inPaste = true
							pasteContent.Reset()
							if debug {
								_, _ = os.Stderr.WriteString("[DEBUG] Paste START\r\n")
							}
						}
						// If already in paste mode, ignore duplicate start marker
						escBuf = nil
						markerDetected = true
						break
					}
					if escStr == pasteEnd || escStr == "[201~" { // \x1b[201~ or [201~
						inPaste = false
						content := pasteContent.String()
						if debug {
							fmt.Fprintf(os.Stderr, "[DEBUG] Paste END, %d bytes\r\n", len(content))
						}
						// Normalize line endings: \r\n -> \n, standalone \r -> \n
						content = strings.ReplaceAll(content, "\r\n", "\n")
						content = strings.ReplaceAll(content, "\r", "\n")
						// Remove trailing newlines
						content = strings.TrimRight(content, "\n")
						// Add pasted content to buffer (don't return yet - wait for Enter)
						buf.WriteString(content)
						// Echo pasted content to terminal so user can see it
						// In raw mode, need \r\n for proper line breaks
						displayContent := strings.ReplaceAll(content, "\n", "\r\n")
						fmt.Print(displayContent)
						pasteContent.Reset()
						escBuf = nil
						markerDetected = true
						break
					}
				}
				// Not a paste marker - add to buffer
				if !markerDetected && len(escBuf) > 0 {
					if inPaste {
						pasteContent.Write(escBuf)
					} else {
						buf.Write(escBuf)
					}
				}
				continue
			}

			// In paste mode - collect everything
			if inPaste {
				pasteContent.WriteByte(b)
				continue
			}

			// Ctrl+D
			if b == 0x04 {
				if buf.Len() == 0 {
					return "", io.EOF
				}
				fmt.Print("\r\n")
				return buf.String(), nil
			}

			// Enter
			if b == '\r' || b == '\n' {
				fmt.Print("\r\n")
				content := buf.String()
				content = stripAllBracketedPasteMarkers(content)
				if content == "```" {
					_ = term.Restore(m.fd, oldState)
					return m.readMultilineWithMarker()
				}
				return content, nil
			}

			// Backspace
			if b == 0x7f || b == 0x08 {
				if buf.Len() > 0 {
					data := buf.Bytes()
					r, size := utf8.DecodeLastRune(data)
					if size > 0 {
						buf.Reset()
						buf.Write(data[:len(data)-size])
						w := runewidth.RuneWidth(r)
						for i := 0; i < w; i++ {
							fmt.Print("\b \b")
						}
					}
				}
				continue
			}

			// Regular character
			buf.WriteByte(b)

			// UTF-8対応エコー
			if b < 0x80 {
				// ASCII: 印字可能文字のみエコー
				if b >= 0x20 && b != 0x7f {
					fmt.Print(string(b))
				}
			} else if b >= 0xC0 {
				// UTF-8マルチバイトの先頭: 何もしない（継続バイトを待つ）
			} else {
				// 継続バイト (0x80-0xBF): 文字が完成したかチェック
				data := buf.Bytes()
				n := len(data)
				if n >= 2 && utf8.Valid(data[n-2:]) {
					r, _ := utf8.DecodeLastRune(data)
					if r != utf8.RuneError {
						fmt.Print(string(r))
					}
				} else if n >= 3 && utf8.Valid(data[n-3:]) {
					r, _ := utf8.DecodeLastRune(data)
					if r != utf8.RuneError {
						fmt.Print(string(r))
					}
				} else if n >= 4 && utf8.Valid(data[n-4:]) {
					r, _ := utf8.DecodeLastRune(data)
					if r != utf8.RuneError {
						fmt.Print(string(r))
					}
				}
			}

		case err := <-m.errChan:
			if err == io.EOF {
				return buf.String(), nil
			}
			return "", err
		}
	}
}

// readMultilineWithMarker handles explicit multiline mode with ``` markers
func (m *MultilineReader) readMultilineWithMarker() (string, error) {
	fmt.Println("📝 Multiline input mode (end with ``` on a new line)")

	// When raw mode goroutine is active, enter raw mode to suppress
	// terminal echo (prevents paste markers from being displayed)
	useChannel := m.rawModeInit && m.byteChan != nil
	var oldState *term.State
	if useChannel && m.fd >= 0 && term.IsTerminal(m.fd) {
		st, err := term.MakeRaw(m.fd)
		if err == nil {
			oldState = st
			defer func() { _ = term.Restore(m.fd, oldState) }()
		}
	}

	var lines []string
	lineNum := 1

	for {
		fmt.Printf("%3d | ", lineNum)

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
	fmt.Printf("✅ Captured %d lines\n", len(lines))
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

// globalReader is the shared MultilineReader for the application
var globalReader *MultilineReader

// SetGlobalReader sets the global MultilineReader for shared access
func SetGlobalReader(r *MultilineReader) {
	globalReader = r
}

// GetGlobalReader returns the global MultilineReader
func GetGlobalReader() *MultilineReader {
	return globalReader
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
		if m.fd >= 0 && term.IsTerminal(m.fd) {
			oldState, err := term.MakeRaw(m.fd)
			if err == nil {
				defer func() { _ = term.Restore(m.fd, oldState) }()
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
	var buf []byte
	for {
		select {
		case b := <-m.byteChan:
			// Enter (raw mode では '\r' が来る)
			if b == '\n' || b == '\r' {
				fmt.Print("\r\n") // 改行をエコー
				return StripBracketedPaste(string(buf)), nil
			}

			// ESC or '[' - check for paste marker (suppress echo of marker bytes)
			if b == 0x1b || b == '[' {
				escBuf := []byte{b}
				maxRead := 5
				if b == 0x1b {
					maxRead = 6 // ESC + [200~ = 6 bytes total
				}
				markerDetected := false
				for i := 0; i < maxRead; i++ {
					nb, ok := m.readByteTimeoutFromChannel(10 * time.Millisecond)
					if !ok {
						break
					}
					escBuf = append(escBuf, nb)
					escStr := string(escBuf)
					if escStr == pasteStart || escStr == "[200~" ||
						escStr == pasteEnd || escStr == "[201~" {
						markerDetected = true
						break
					}
				}
				if !markerDetected && len(escBuf) > 0 {
					buf = append(buf, escBuf...)
					// Echo non-marker bytes
					for _, eb := range escBuf {
						if eb >= 0x20 && eb < 0x80 && eb != 0x7f {
							fmt.Print(string(eb))
						}
					}
				}
				continue
			}

			// Backspace / DEL
			if b == 0x7f || b == 0x08 {
				if len(buf) > 0 {
					r, size := utf8.DecodeLastRune(buf)
					if size > 0 {
						buf = buf[:len(buf)-size]
						w := runewidth.RuneWidth(r)
						for i := 0; i < w; i++ {
							fmt.Print("\b \b")
						}
					}
				}
				continue
			}

			buf = append(buf, b)

			// UTF-8対応エコー
			if b < 0x80 {
				if b >= 0x20 && b != 0x7f {
					fmt.Print(string(b))
				}
			} else if b >= 0xC0 {
				// UTF-8マルチバイトの先頭: 何もしない
			} else {
				// 継続バイト: 文字が完成したかチェック
				n := len(buf)
				if n >= 2 && utf8.Valid(buf[n-2:]) {
					r, _ := utf8.DecodeLastRune(buf)
					if r != utf8.RuneError {
						fmt.Print(string(r))
					}
				} else if n >= 3 && utf8.Valid(buf[n-3:]) {
					r, _ := utf8.DecodeLastRune(buf)
					if r != utf8.RuneError {
						fmt.Print(string(r))
					}
				} else if n >= 4 && utf8.Valid(buf[n-4:]) {
					r, _ := utf8.DecodeLastRune(buf)
					if r != utf8.RuneError {
						fmt.Print(string(r))
					}
				}
			}
		case err := <-m.errChan:
			if len(buf) > 0 {
				return StripBracketedPaste(string(buf)), nil
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
